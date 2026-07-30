package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"time"
)

// ContextoCliente es lo que recibe clienteDetalle.html. Antes esta misma
// estructura estaba repetida (con pequeñas variaciones) en 7 lugares
// distintos de este archivo; armarContextoCliente() la arma una sola vez.
type ContextoCliente struct {
	*Cliente
	Ventas      []Venta
	Cobros      []Cobro
	Config      Configuracion
	DiasEnDeuda int
	UltimoAviso string
	Error       string
}

func armarContextoCliente(db *sql.DB, cliente *Cliente, errMsg string) ContextoCliente {
	ventas := ObtenerVentasDeCliente(db, cliente.ID)
	cobros := ObtenerCobrosDeCliente(db, cliente.ID)
	config := ObtenerConfiguracion(db)

	dias := 0
	if fechaRef, err := ObtenerFechaReferencia(db, cliente.ID); err == nil {
		dias = DiasDesde(fechaRef)
	}

	return ContextoCliente{
		Cliente:     cliente,
		Ventas:      ventas,
		Cobros:      cobros,
		Config:      config,
		DiasEnDeuda: dias,
		UltimoAviso: ObtenerUltimoAviso(db, cliente.ID),
		Error:       errMsg,
	}
}

func main() {

	// --- 1. Logs: antes que nada, para que hasta los errores de arranque queden registrados ---
	archivoLog, err := ConfigurarLogs()
	if err != nil {
		// Si esto falla seguimos igual: log.Println sin ConfigurarLogs()
		// simplemente escribe a la consola, no es fatal.
		log.Println("No se pudo configurar el log a archivo, sigo solo por consola:", err)
	} else {
		defer archivoLog.Close()
	}

	// --- 2. Config: leemos config.json (puerto, empresa) de al lado del ejecutable ---
	config := CargarConfigApp()
	log.Println("Configuración cargada. Empresa:", config.Empresa, "| Puerto preferido:", config.Puerto)

	// --- 3. Ruta de la DB: siempre relativa al ejecutable, no a "donde se lo invocó" ---
	dirBase, err := dirEjecutable()
	if err != nil {
		log.Fatal("ERROR: No se pudo determinar la carpeta del ejecutable:", err)
	}
	rutaDB := filepath.Join(dirBase, "data", "cobrapp.db")

	db, err := ConectarDB(rutaDB)
	if err != nil {
		log.Fatal("ERROR: No se pudo conectar a la db:", err)
	}
	defer db.Close()
	log.Println("Se conectó correctamente a la db en:", rutaDB)

	// --- 4. Backups: uno ahora, y de ahí en más uno cada 24hs, en segundo plano ---
	go IniciarBackupsPeriodicos(rutaDB)

	mux := http.NewServeMux()

	// Archivos estáticos (css, imágenes) embebidos en el binario (ver
	// embebidos.go). fs.Sub "recorta" el prefijo "static/" para que las
	// URLs sigan siendo /static/css/style.css como antes.
	staticSinPrefijo, err := fs.Sub(staticFS, "static")
	if err != nil {
		log.Fatal("ERROR: No se pudo preparar los archivos estáticos embebidos:", err)
	}
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticSinPrefijo))))

	// --- Login / logout: NUNCA van envueltas en requiereLogin, si no nadie podría loguearse ---

	mux.HandleFunc("GET /login", func(w http.ResponseWriter, r *http.Request) {
		renderizar(w, "baseLogin.html", "login.html", struct{ Error string }{})
	})

	mux.HandleFunc("POST /login", func(w http.ResponseWriter, r *http.Request) {
		usuario := r.FormValue("usuario")
		password := r.FormValue("password")

		if !validarCredenciales(usuario, password) {
			renderizar(w, "baseLogin.html", "login.html", struct{ Error string }{Error: "Usuario o contraseña incorrectos."})
			return
		}

		if err := crearSesion(w); err != nil {
			http.Error(w, "No se pudo iniciar sesión", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
	})

	mux.HandleFunc("POST /logout", func(w http.ResponseWriter, r *http.Request) {
		cerrarSesion(w, r)
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	})

	// --- De acá para abajo, todo pasa primero por requiereLogin ---

	mux.HandleFunc("GET /{$}", requiereLogin(func(w http.ResponseWriter, r *http.Request) {
		clientes := ObtenerClientes(db)

		var totalACobrar, totalAFavor float64
		var clientesEnDeuda int

		for _, c := range clientes {
			if c.Saldo > 0 {
				totalACobrar += c.Saldo
				clientesEnDeuda++
			} else if c.Saldo < 0 {
				totalAFavor += -c.Saldo // lo pasamos a positivo para mostrarlo
			}
		}

		config := ObtenerConfiguracion(db)

		renderizar(w, "base.html", "menuPrincipal.html", struct {
			Clientes         []Cliente
			TotalACobrar     float64
			TotalAFavor      float64
			ClientesEnDeuda  int
			ClientesSinDeuda int
			TotalClientes    int
			Config           Configuracion
		}{
			Clientes:         clientes,
			TotalACobrar:     totalACobrar,
			TotalAFavor:      totalAFavor,
			ClientesEnDeuda:  clientesEnDeuda,
			ClientesSinDeuda: len(clientes) - clientesEnDeuda,
			TotalClientes:    len(clientes),
			Config:           config,
		})
	}))

	// Formulario de cliente nuevo (mostrar)
	mux.HandleFunc("GET /cliente_nuevo", requiereLogin(func(w http.ResponseWriter, r *http.Request) {
		renderizar(w, "base.html", "clienteNuevo.html", struct{ Error string }{})
	}))

	// Formulario de cliente nuevo (guardar)
	mux.HandleFunc("POST /cliente_nuevo", requiereLogin(func(w http.ResponseWriter, r *http.Request) {
		nombre := r.FormValue("nombre")
		apellido := r.FormValue("apellido")
		email := r.FormValue("email")
		telefono := r.FormValue("telefono")

		if nombre == "" || apellido == "" {
			renderizar(w, "base.html", "clienteNuevo.html", struct{ Error string }{Error: "Nombre y apellido son obligatorios."})
			return
		}

		if err := CrearCliente(db, nombre, apellido, email, telefono); err != nil {
			log.Println("Error creando cliente:", err)
			renderizar(w, "base.html", "clienteNuevo.html", struct{ Error string }{Error: "No se pudo guardar el cliente."})
			return
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
	}))

	// Detalle de un cliente puntual
	mux.HandleFunc("GET /clientes/{id}", requiereLogin(func(w http.ResponseWriter, r *http.Request) {
		idStr := r.PathValue("id")

		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "ID inválido", http.StatusBadRequest)
			return
		}

		cliente, err := ObtenerClientePorID(db, id)
		if err != nil {
			http.NotFound(w, r)
			return
		}

		renderizar(w, "base.html", "clienteDetalle.html", armarContextoCliente(db, cliente, ""))
	}))

	// Formulario de venta nueva (mostrar)
	mux.HandleFunc("GET /clientes/{id}/venta_nueva", requiereLogin(func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			http.Error(w, "ID inválido", http.StatusBadRequest)
			return
		}

		cliente, err := ObtenerClientePorID(db, id)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		renderizar(w, "base.html", "ventaNueva.html", struct {
			Cliente *Cliente
			Error   string
		}{Cliente: cliente})
	}))

	// Formulario de venta nueva (guardar)
	mux.HandleFunc("POST /clientes/{id}/venta_nueva", requiereLogin(func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			http.Error(w, "ID inválido", http.StatusBadRequest)
			return
		}

		totalStr := r.FormValue("total")
		descripcion := r.FormValue("descripcion")
		fechaForm := r.FormValue("fecha")

		total, err := strconv.ParseFloat(totalStr, 64)
		if err != nil || total <= 0 {
			cliente, _ := ObtenerClientePorID(db, id)
			renderizar(w, "base.html", "ventaNueva.html", struct {
				Cliente *Cliente
				Error   string
			}{Cliente: cliente, Error: "El total tiene que ser un número mayor a 0."})
			return
		}

		fecha, err := normalizarFecha(fechaForm)
		if err != nil {
			cliente, _ := ObtenerClientePorID(db, id)
			renderizar(w, "base.html", "ventaNueva.html", struct {
				Cliente *Cliente
				Error   string
			}{Cliente: cliente, Error: "La fecha ingresada no es válida."})
			return
		}

		if err := CrearVenta(db, id, total, descripcion, fecha); err != nil {
			log.Println("Error creando venta:", err)
			http.Error(w, "No se pudo guardar la venta", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/clientes/"+strconv.Itoa(id), http.StatusSeeOther)
	}))

	// Formulario de cobro nuevo (mostrar)
	mux.HandleFunc("GET /clientes/{id}/cobro_nuevo", requiereLogin(func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			http.Error(w, "ID inválido", http.StatusBadRequest)
			return
		}

		cliente, err := ObtenerClientePorID(db, id)
		if err != nil {
			http.NotFound(w, r)
			return
		}

		renderizar(w, "base.html", "cobroNuevo.html", struct {
			Cliente *Cliente
			Error   string
		}{Cliente: cliente})
	}))

	// Formulario de cobro nuevo (guardar)
	mux.HandleFunc("POST /clientes/{id}/cobro_nuevo", requiereLogin(func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			http.Error(w, "ID inválido", http.StatusBadRequest)
			return
		}

		montoStr := r.FormValue("monto")
		observacion := r.FormValue("observacion")
		fechaForm := r.FormValue("fecha")

		monto, err := strconv.ParseFloat(montoStr, 64)
		if err != nil || monto <= 0 {
			cliente, _ := ObtenerClientePorID(db, id)
			renderizar(w, "base.html", "cobroNuevo.html", struct {
				Cliente *Cliente
				Error   string
			}{Cliente: cliente, Error: "El monto tiene que ser un número mayor a 0."})
			return
		}

		fecha, err := normalizarFecha(fechaForm)
		if err != nil {
			cliente, _ := ObtenerClientePorID(db, id)
			renderizar(w, "base.html", "cobroNuevo.html", struct {
				Cliente *Cliente
				Error   string
			}{Cliente: cliente, Error: "La fecha ingresada no es válida."})
			return
		}

		if err := CrearCobro(db, id, monto, observacion, fecha); err != nil {
			log.Println("Error creando cobro:", err)
			http.Error(w, "No se pudo guardar el cobro", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/clientes/"+strconv.Itoa(id), http.StatusSeeOther)
	}))

	// Modificar clientes
	mux.HandleFunc("GET /clientes/{id}/editar", requiereLogin(func(w http.ResponseWriter, r *http.Request) {

		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			http.Error(w, "ID inválido", http.StatusBadRequest)
			return
		}

		cliente, err := ObtenerClientePorID(db, id)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		renderizar(w, "base.html", "modificarCliente.html", struct {
			Cliente *Cliente
			Error   string
		}{Cliente: cliente})
	}))

	mux.HandleFunc("POST /clientes/{id}/editar", requiereLogin(func(w http.ResponseWriter, r *http.Request) {

		id, err := strconv.Atoi(r.PathValue("id"))

		if err != nil {
			http.Error(w, "ID inválido", http.StatusBadRequest)
			return
		}
		nombre := r.FormValue("nombre")
		apellido := r.FormValue("apellido")
		email := r.FormValue("email")
		telefono := r.FormValue("telefono")

		if nombre == "" || apellido == "" {
			renderizar(w, "base.html", "clienteNuevo.html", struct{ Error string }{Error: "Nombre y apellido son obligatorios."})
			return
		}

		if err := ModificarCliente(db, id, nombre, apellido, email, telefono); err != nil {
			log.Println("Error modificando al cliente:", err)
			renderizar(w, "base.html", "clienteNuevo.html", struct{ Error string }{Error: "No se pudo modificar el cliente."})
			return
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
	}))

	// Registrar que se le mandó un aviso (WhatsApp o email) a un cliente
	mux.HandleFunc("POST /clientes/{id}/aviso", requiereLogin(func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			http.Error(w, "ID inválido", http.StatusBadRequest)
			return
		}

		tipo := r.FormValue("tipo") // "wp" o "email"

		if err := RegistrarAviso(db, id, tipo); err != nil {
			log.Println("Error registrando el aviso:", err)
			http.Error(w, "No se pudo registrar el aviso", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}))

	// Eliminar clientes
	mux.HandleFunc("POST /clientes/{id}/eliminar", requiereLogin(func(w http.ResponseWriter, r *http.Request) {

		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			http.Error(w, "ID inválido", http.StatusBadRequest)
			return
		}

		cliente, err := ObtenerClientePorID(db, id)
		if err != nil {
			http.NotFound(w, r)
			return
		}

		password := r.FormValue("password")

		if !validarCredenciales("admin", password) {
			renderizar(w, "base.html", "clienteDetalle.html", armarContextoCliente(db, cliente, "Contraseña incorrecta."))
			return
		}

		if err := eliminarCliente(db, id); err != nil {
			renderizar(w, "base.html", "clienteDetalle.html", armarContextoCliente(db, cliente, "No se pudo eliminar el cliente."))
			return
		}

		http.Redirect(
			w,
			r,
			"/?eliminado=cliente",
			http.StatusSeeOther,
		)
	}))

	// Eliminar cobros
	mux.HandleFunc("POST /cobro/{id}/eliminar", requiereLogin(func(w http.ResponseWriter, r *http.Request) {

		idCobro, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			http.Error(w, "ID inválido", http.StatusBadRequest)
			return
		}

		cobro, err := ObtenerCobroPorID(db, idCobro)
		if err != nil {
			http.NotFound(w, r)
			return
		}

		cliente, err := ObtenerClientePorID(db, cobro.ClienteID)

		if err != nil {
			http.NotFound(w, r)
			return
		}

		password := r.FormValue("password")

		if !validarCredenciales("admin", password) {
			renderizar(w, "base.html", "clienteDetalle.html", armarContextoCliente(db, cliente, "Contraseña incorrecta."))
			return
		}

		if err := eliminarCobro(db, idCobro); err != nil {
			renderizar(w, "base.html", "clienteDetalle.html", armarContextoCliente(db, cliente, "No se pudo eliminar el cobro."))
			return
		}

		http.Redirect(
			w,
			r,
			"/clientes/"+strconv.Itoa(cliente.ID)+"?eliminado=cobro",
			http.StatusSeeOther,
		)
	}))

	// Eliminar ventas
	mux.HandleFunc("POST /venta/{id}/eliminar", requiereLogin(func(w http.ResponseWriter, r *http.Request) {

		idVenta, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			http.Error(w, "ID inválido", http.StatusBadRequest)
			return
		}

		venta, err := ObtenerVentaPorID(db, idVenta)
		if err != nil {
			http.NotFound(w, r)
			return
		}

		cliente, err := ObtenerClientePorID(db, venta.ClienteID)
		if err != nil {
			http.NotFound(w, r)
			return
		}

		password := r.FormValue("password")

		if !validarCredenciales("admin", password) {
			renderizar(w, "base.html", "clienteDetalle.html", armarContextoCliente(db, cliente, "Contraseña incorrecta."))
			return
		}

		if err := eliminarVenta(db, idVenta); err != nil {
			renderizar(w, "base.html", "clienteDetalle.html", armarContextoCliente(db, cliente, "No se pudo eliminar la venta."))
			return
		}

		http.Redirect(
			w,
			r,
			"/clientes/"+strconv.Itoa(cliente.ID)+"?eliminado=venta",
			http.StatusSeeOther,
		)
	}))

	// Seccion de estadisticas
	mux.HandleFunc("GET /estadisticas", requiereLogin(func(w http.ResponseWriter, r *http.Request) {

		TotalClientesUltMes, err := MicroEstadistica(db, "mensual")
		if err != nil {
			http.NotFound(w, r)
			return
		}

		TotalClientesUltAño, err := MicroEstadistica(db, "anual")
		if err != nil {
			http.NotFound(w, r)
			return
		}

		CantClientesMensualesUltAño, err := MacroEstadisticaMensualClientes(db)
		if err != nil {
			http.NotFound(w, r)
			return
		}

		CantVentasMensualesUltAño, err := MacroEstadisticaMensualVentas(db)
		if err != nil {
			http.NotFound(w, r)
			return
		}

		CantCobrosMensualesUltAño, err := MacroEstadisticaMensualCobros(db)
		if err != nil {
			http.NotFound(w, r)
			return
		}

		TopDeudores := ObtenerTopDeudores(db, 10)

		meses := [13]string{
			"",
			"Enero",
			"Febrero",
			"Marzo",
			"Abril",
			"Mayo",
			"Junio",
			"Julio",
			"Agosto",
			"Septiembre",
			"Octubre",
			"Noviembre",
			"Diciembre",
		}

		type DatoGrafico struct {
			Mes      string `json:"mes"`
			Clientes int    `json:"clientes"`
			Ventas   int    `json:"ventas"`
			Cobros   int    `json:"cobros"`
		}

		datos := make([]DatoGrafico, 0, 12)

		for i := 1; i <= 12; i++ {
			datos = append(datos, DatoGrafico{
				Mes:      meses[i],
				Clientes: CantClientesMensualesUltAño[i],
				Ventas:   CantVentasMensualesUltAño[i],
				Cobros:   CantCobrosMensualesUltAño[i],
			})
		}

		jsonDatos, err := json.Marshal(datos)
		if err != nil {
			http.NotFound(w, r)
			return
		}

		renderizar(w, "base.html", "estadisticas.html", struct {
			TotalClientesUltMes int
			TotalClientesUltAño int
			DatosGraficoJSON    template.JS
			TopDeudores         []Cliente
		}{
			TotalClientesUltMes: TotalClientesUltMes,
			TotalClientesUltAño: TotalClientesUltAño,
			DatosGraficoJSON:    template.JS(jsonDatos),
			TopDeudores:         TopDeudores,
		})
	}))

	mux.HandleFunc("GET /contacto", requiereLogin(func(w http.ResponseWriter, r *http.Request) {
		renderizar(w, "base.html", "contacto.html", nil)
	}))

	//nuevo esto tambien juani
	// MOSTRAR la pantalla de configuración
	mux.HandleFunc("GET /configuracion", requiereLogin(func(w http.ResponseWriter, r *http.Request) {
		config := ObtenerConfiguracion(db)

		renderizar(w, "base.html", "configuracion.html", struct {
			Config Configuracion
			Error  string
			Exito  string
		}{Config: config})
	}))

	// GUARDAR los cambios cuando tocan el botón
	mux.HandleFunc("POST /configuracion", requiereLogin(func(w http.ResponseWriter, r *http.Request) {
		diasStr := r.FormValue("dias_alerta")
		mensaje := r.FormValue("mensaje_wp")
		minutos_inactividadStr := r.FormValue("minutos_inactividad")

		dias, err := strconv.Atoi(diasStr)
		minutos_inactividad, err := strconv.Atoi(minutos_inactividadStr)

		if err != nil || dias < 1 || minutos_inactividad < 0 {
			config := ObtenerConfiguracion(db)
			renderizar(w, "base.html", "configuracion.html", struct {
				Config Configuracion
				Error  string
				Exito  string
			}{Config: config, Error: "No se puede guardar esos valores, compruebe dias y minutos."})
			return
		}

		if err := GuardarConfiguracion(db, dias, mensaje, minutos_inactividad); err != nil {
			config := ObtenerConfiguracion(db)
			renderizar(w, "base.html", "configuracion.html", struct {
				Config Configuracion
				Error  string
				Exito  string
			}{Config: config, Error: "No se pudo guardar la configuración."})
			return
		}

		// Volvemos a pedir la configuración actualizada para mandarla a la vista
		configActualizada := ObtenerConfiguracion(db)
		renderizar(w, "base.html", "configuracion.html", struct {
			Config Configuracion
			Error  string
			Exito  string
		}{Config: configActualizada, Exito: "¡Configuración guardada con éxito!"})
	}))

	// Modificar cobros
	mux.HandleFunc("GET /cobro/{id}/modificar", requiereLogin(func(w http.ResponseWriter, r *http.Request) {

		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			http.Error(w, "ID inválido", http.StatusBadRequest)
			return
		}

		cobro, err := ObtenerCobroPorID(db, id)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		renderizar(w, "base.html", "modificarCobro.html", struct {
			Cobro *Cobro
			Error string
		}{Cobro: cobro})
	}))

	mux.HandleFunc("POST /cobro/{id}/modificar", requiereLogin(func(w http.ResponseWriter, r *http.Request) {

		cobroID, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			http.Error(w, "ID inválido", http.StatusBadRequest)
			return
		}

		montoStr := r.FormValue("monto")
		montoFloat, err := strconv.ParseFloat(montoStr, 64)
		if err != nil {
			cobro, _ := ObtenerCobroPorID(db, cobroID)
			renderizar(w, "base.html", "modificarCobro.html", struct {
				Cobro *Cobro
				Error string
			}{Cobro: cobro, Error: "Monto inválido."})
			return
		}

		observacion := r.FormValue("observacion")
		fechaForm := r.FormValue("fecha")

		fecha, err := normalizarFecha(fechaForm)
		if err != nil {
			cobro, _ := ObtenerCobroPorID(db, cobroID)
			renderizar(w, "base.html", "modificarCobro.html", struct {
				Cobro *Cobro
				Error string
			}{Cobro: cobro, Error: "La fecha ingresada no es válida."})
			return
		}

		if err := ModificarCobro(db, cobroID, montoFloat, observacion, fecha); err != nil {
			log.Println("Error modificando el cobro:", err)
			cobro, _ := ObtenerCobroPorID(db, cobroID)
			renderizar(w, "base.html", "modificarCobro.html", struct {
				Cobro *Cobro
				Error string
			}{Cobro: cobro, Error: "No se pudo modificar el cobro."})
			return
		}

		cobro, err := ObtenerCobroPorID(db, cobroID)
		if err != nil {
			http.NotFound(w, r)
			return
		}

		http.Redirect(w, r, "/clientes/"+strconv.Itoa(cobro.ClienteID), http.StatusSeeOther)
	}))

	// Modificar ventas
	mux.HandleFunc("GET /venta/{id}/modificar", requiereLogin(func(w http.ResponseWriter, r *http.Request) {

		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			http.Error(w, "ID inválido", http.StatusBadRequest)
			return
		}

		venta, err := ObtenerVentaPorID(db, id)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		renderizar(w, "base.html", "modificarVenta.html", struct {
			Venta *Venta
			Error string
		}{Venta: venta})
	}))

	mux.HandleFunc("POST /venta/{id}/modificar", requiereLogin(func(w http.ResponseWriter, r *http.Request) {

		ventaID, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			http.Error(w, "ID inválido", http.StatusBadRequest)
			return
		}

		totalStr := r.FormValue("total")
		totalFloat, err := strconv.ParseFloat(totalStr, 64)
		if err != nil {
			venta, _ := ObtenerVentaPorID(db, ventaID)
			renderizar(w, "base.html", "modificarVenta.html", struct {
				Venta *Venta
				Error string
			}{Venta: venta, Error: "Total inválido."})
			return
		}

		descripcion := r.FormValue("descripcion")
		fechaForm := r.FormValue("fecha")

		fecha, err := normalizarFecha(fechaForm)
		if err != nil {
			venta, _ := ObtenerVentaPorID(db, ventaID)
			renderizar(w, "base.html", "modificarVenta.html", struct {
				Venta *Venta
				Error string
			}{Venta: venta, Error: "La fecha ingresada no es válida."})
			return
		}

		if err := ModificarVenta(db, ventaID, totalFloat, descripcion, fecha); err != nil {
			log.Println("Error modificando la venta:", err)
			venta, _ := ObtenerVentaPorID(db, ventaID)
			renderizar(w, "base.html", "modificarVenta.html", struct {
				Venta *Venta
				Error string
			}{Venta: venta, Error: "No se pudo modificar la venta."})
			return
		}

		venta, err := ObtenerVentaPorID(db, ventaID)
		if err != nil {
			http.NotFound(w, r)
			return
		}

		http.Redirect(w, r, "/clientes/"+strconv.Itoa(venta.ClienteID), http.StatusSeeOther)
	}))

	// 1. Ruta para OBTENER las notificaciones (La que alimenta la bandeja)
	mux.HandleFunc(
		"GET /api/notificaciones",
		requiereLogin(func(w http.ResponseWriter, r *http.Request) {

			err := SincronizarNotificaciones(db)
			if err != nil {
				http.Error(w, "Error al sincronizar notificaciones", http.StatusInternalServerError)
				return
			}

			notificaciones, err := ObtenerNotificacionesValidas(db)
			if err != nil {
				http.Error(w, "Error al obtener notificaciones", http.StatusInternalServerError)
				return
			}

			type NotificacionConCliente struct {
				Cliente      Cliente      `json:"cliente"`
				Notificacion Notificacion `json:"notificacion"`
			}

			var notificacionesConClientes []NotificacionConCliente

			for _, notificacion := range notificaciones {
				cliente, err := ObtenerClientePorID(db, notificacion.Cliente_id)

				if err != nil {
					http.Error(w, "Error al obtener cliente", http.StatusInternalServerError)
					return
				} else {
					notificacionesConClientes = append(
						notificacionesConClientes,
						NotificacionConCliente{
							Notificacion: notificacion,
							Cliente:      *cliente,
						},
					)
				}
			}

			type RespuestaNotificaciones struct {
				Notificaciones []NotificacionConCliente `json:"notificaciones"`
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(
				RespuestaNotificaciones{
					Notificaciones: notificacionesConClientes,
				},
			)
		}),
	)

	// 2. Ruta nueva para MARCAR las notificaciones como vistas (La que se ejecuta al tocar el botón)
	mux.HandleFunc(
		"POST /api/notificaciones/marcar-vistas",
		requiereLogin(func(w http.ResponseWriter, r *http.Request) {

			// Llamamos a la función de la base de datos
			err := marcarNotificacionComoVista(db)

			if err != nil {
				http.Error(w, "Error al actualizar notificaciones", http.StatusInternalServerError)
				return
			}

			// Le respondemos al frontend que todo salió bien (Status 200 OK)
			w.WriteHeader(http.StatusOK)
		}),
	)

	//lleva las configuraciones a js
	mux.HandleFunc("GET /configuracion/temporizador", func(w http.ResponseWriter, r *http.Request) {
		config := ObtenerConfiguracion(db)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(config.Minutos_inactividad)
	})

	//exportacion a pdf
	mux.HandleFunc("GET /clientes/{id}/pdf", func(w http.ResponseWriter, r *http.Request) {

		idCliente, err := strconv.Atoi(r.PathValue("id"))

		if err != nil {
			http.Error(w, "ID inválido", 400)
			return
		}

		cliente, err := ObtenerClientePorID(db, idCliente)

		if err != nil {
			http.NotFound(w, r)
			return
		}

		ventas := ObtenerVentasDeCliente(db, idCliente)
		cobros := ObtenerCobrosDeCliente(db, idCliente)

		pdfBytes, err := generarPDF(cliente, ventas, cobros)

		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		w.Header().Set("Content-Type", "application/pdf")

		w.Header().Set(
			"Content-Disposition",
			`inline; filename="cliente.pdf"`,
		)

		w.Write(pdfBytes)

	})

	// --- Puerto: probamos el de config.json; si está ocupado, buscamos uno libre ---
	puerto, err := puertoLibre(config.Puerto)
	if err != nil {
		log.Fatal("ERROR: No se pudo encontrar un puerto libre:", err)
	}
	if puerto != config.Puerto {
		log.Printf("El puerto %d estaba ocupado, uso el %d en su lugar\n", config.Puerto, puerto)
	}

	url := fmt.Sprintf("http://localhost:%d", puerto)
	log.Println("Servidor iniciado en", url)

	// Abrimos el navegador un instante después de arrancar, en otra
	// goroutine: si lo hacemos antes de que el servidor esté escuchando,
	// el navegador puede llegar a mostrar "no se pudo conectar".
	go func() {
		time.Sleep(300 * time.Millisecond)
		if err := abrirNavegador(url); err != nil {
			log.Println("No se pudo abrir el navegador automáticamente:", err)
		}
	}()

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", puerto), mux))
}

func renderizar(w http.ResponseWriter, layout, pagina string, datos any) {

	// ParseFS es el equivalente de ParseFiles pero leyendo de un embed.FS
	// (memoria, dentro del binario) en vez de leer del disco. Las rutas
	// se escriben igual, por eso el resto del código no cambia.
	tmpl, err := template.ParseFS(
		plantillasFS,
		"templates/sideBar.html",
		"templates/bandejaDeEntrada.html",
		"templates/"+pagina,
		"templates/"+layout,
	)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = tmpl.ExecuteTemplate(w, "base", datos)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

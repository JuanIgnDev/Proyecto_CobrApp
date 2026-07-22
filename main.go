package main

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"strconv"
)

func main() {

	//Conectar la DB primero
	db, err := ConectarDB("./data/cobrapp.db")
	if err != nil {
		log.Fatal("ERROR: No se pudo conectar a la db:", err)
	}
	defer db.Close()
	log.Println("Se conectó correctamente a la db!")

	mux := http.NewServeMux()

	//carga archivos css
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

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

		renderizar(w, "base.html", "menuPrincipal.html", struct {
			Clientes         []Cliente
			TotalACobrar     float64
			TotalAFavor      float64
			ClientesEnDeuda  int
			ClientesSinDeuda int
			TotalClientes    int
		}{
			Clientes:         clientes,
			TotalACobrar:     totalACobrar,
			TotalAFavor:      totalAFavor,
			ClientesEnDeuda:  clientesEnDeuda,
			ClientesSinDeuda: len(clientes) - clientesEnDeuda,
			TotalClientes:    len(clientes),
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

		ventas := ObtenerVentasDeCliente(db, id)
		cobros := ObtenerCobrosDeCliente(db, id)

		renderizar(w, "base.html", "clienteDetalle.html", struct {
			*Cliente
			Ventas []Venta
			Cobros []Cobro
			Error  string
		}{Cliente: cliente, Ventas: ventas, Cobros: cobros, Error: ""})
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

		ventas := ObtenerVentasDeCliente(db, id)
		cobros := ObtenerCobrosDeCliente(db, id)

		password := r.FormValue("password")

		if !validarCredenciales("admin", password) {
			renderizar(w, "base.html", "clienteDetalle.html", struct {
				*Cliente
				Ventas []Venta
				Cobros []Cobro
				Error  string
			}{
				Cliente: cliente,
				Ventas:  ventas,
				Cobros:  cobros,
				Error:   "Contraseña incorrecta.",
			})
			return
		}

		if err := eliminarCliente(db, id); err != nil {
			renderizar(w, "base.html", "clienteDetalle.html", struct {
				*Cliente
				Ventas []Venta
				Cobros []Cobro
				Error  string
			}{
				Cliente: cliente,
				Ventas:  ventas,
				Cobros:  cobros,
				Error:   "No se pudo eliminar el cliente.",
			})
			return
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
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
		}{
			TotalClientesUltMes: TotalClientesUltMes,
			TotalClientesUltAño: TotalClientesUltAño,
			DatosGraficoJSON:    template.JS(jsonDatos),
		})
	}))

	mux.HandleFunc("GET /contacto", requiereLogin(func(w http.ResponseWriter, r *http.Request) {
		renderizar(w, "base.html", "contacto.html", nil)
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


mux.HandleFunc(
	"GET /api/notificaciones",
	requiereLogin(func(w http.ResponseWriter, r *http.Request) {

		err := SincronizarNotificaciones(db)

		if err != nil {
			http.Error(
				w,
				"Error al sincronizar notificaciones",
				http.StatusInternalServerError,
			)
			return
		}

		notificaciones, err := ObtenerNotificacionesValidas(db)

		if err != nil {
			http.Error(
				w,
				"Error al obtener notificaciones",
				http.StatusInternalServerError,
			)
			return
		}

		type NotificacionConCliente struct {
			Cliente      Cliente      `json:"cliente"`
			Notificacion Notificacion `json:"notificacion"`
		}

		var notificacionesConClientes []NotificacionConCliente

		for _, notificacion := range notificaciones {

			cliente, err := ObtenerClientePorID(
				db,
				notificacion.Cliente_id,
			)

			if err != nil {
				http.Error(
					w,
					"Error al obtener cliente",
					http.StatusInternalServerError,
				)
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

		w.Header().Set(
			"Content-Type",
			"application/json",
		)

		json.NewEncoder(w).Encode(
			RespuestaNotificaciones{
				Notificaciones: notificacionesConClientes,
			},
		)
	}),
)

	log.Println("Servidor iniciado en http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", mux))

}

func renderizar(w http.ResponseWriter, layout, pagina string, datos any) {
	tmpl, err := template.ParseFiles("templates/sideBar.html","templates/bandejaDeEntrada.html", "templates/"+pagina, "templates/"+layout)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := tmpl.ExecuteTemplate(w, "base", datos); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

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
		
		renderizar(w,"baseLogin.html", "login.html", struct{ Error string
		}{ })
	})

	mux.HandleFunc("POST /login", func(w http.ResponseWriter, r *http.Request) {
		usuario := r.FormValue("usuario")
		password := r.FormValue("password")

		if !validarCredenciales(usuario, password) {
			renderizar(w,"baseLogin.html", "login.html", struct{ Error string }{Error: "Usuario o contraseña incorrectos."})
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

		renderizar(w,"base.html", "menuPrincipal.html", struct {
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

	// 4. Formulario de cliente nuevo (mostrar)
	mux.HandleFunc("GET /cliente_nuevo", requiereLogin(func(w http.ResponseWriter, r *http.Request) {
		renderizar(w,"base.html", "clienteNuevo.html", struct{ Error string}{})
	}))

	// 5. Formulario de cliente nuevo (guardar)
	mux.HandleFunc("POST /cliente_nuevo", requiereLogin(func(w http.ResponseWriter, r *http.Request) {
		nombre := r.FormValue("nombre")
		apellido := r.FormValue("apellido")
		email  := r.FormValue("email")
		telefono := r.FormValue("telefono")

		if nombre == "" || apellido == "" {
			renderizar(w,"base.html", "clienteNuevo.html", struct{ Error string }{Error: "Nombre y apellido son obligatorios."})
			return
		}

		if err := CrearCliente(db, nombre, apellido, email, telefono); err != nil {
			log.Println("Error creando cliente:", err)
			renderizar(w,"base.html", "clienteNuevo.html", struct{ Error string
				 }{Error: "No se pudo guardar el cliente.", })
			return
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
	}))

	// 6. Detalle de un cliente puntual
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

		compras := ObtenerComprasDeCliente(db, id)
		pagos := ObtenerPagosDeCliente(db, id)

		renderizar(w,"base.html", "clienteDetalle.html", struct {
			*Cliente
			Compras []Compra
			Pagos   []Pago
			Error string
			
		}{Cliente: cliente, Compras: compras, Pagos: pagos, Error: "", })
	}))

	// 8. Formulario de venta nueva (mostrar)
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
		renderizar(w,"base.html", "ventaNueva.html", struct {
			Cliente *Cliente
			Error   string

		}{Cliente: cliente, })
	}))

	// 9. Formulario de venta nueva (guardar)
	mux.HandleFunc("POST /clientes/{id}/venta_nueva", requiereLogin(func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			http.Error(w, "ID inválido", http.StatusBadRequest)
			return
		}

		totalStr := r.FormValue("total")
		descripcion := r.FormValue("descripcion")

		total, err := strconv.ParseFloat(totalStr, 64)
		if err != nil || total <= 0 {
			cliente, _ := ObtenerClientePorID(db, id)
			renderizar(w,"base.html", "ventaNueva.html", struct {
				Cliente *Cliente
				Error   string
			}{Cliente: cliente, Error: "El total tiene que ser un número mayor a 0."})
			return
		}

		if err := CrearCompra(db, id, total, descripcion); err != nil {
			log.Println("Error creando compra:", err)
			http.Error(w, "No se pudo guardar la venta", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/clientes/"+strconv.Itoa(id), http.StatusSeeOther)
	}))

	// 10. Formulario de pago nuevo (mostrar)
	mux.HandleFunc("GET /clientes/{id}/pago_nuevo", requiereLogin(func(w http.ResponseWriter, r *http.Request) {
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

		renderizar(w,"base.html", "pagoNuevo.html", struct {
			Cliente *Cliente
			Error   string
		}{Cliente: cliente,})
	}))

	// 11. Formulario de pago nuevo (guardar)
	mux.HandleFunc("POST /clientes/{id}/pago_nuevo", requiereLogin(func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			http.Error(w, "ID inválido", http.StatusBadRequest)
			return
		}

		montoStr := r.FormValue("monto")
		observacion := r.FormValue("observacion")

		monto, err := strconv.ParseFloat(montoStr, 64)
		if err != nil || monto <= 0 {
			cliente, _ := ObtenerClientePorID(db, id)
			renderizar(w,"base.html", "pagoNuevo.html", struct {
				Cliente *Cliente
				Error   string
			}{Cliente: cliente, Error: "El monto tiene que ser un número mayor a 0."})
			return
		}

		if err := CrearPago(db, id, monto, observacion); err != nil {
			log.Println("Error creando pago:", err)
			http.Error(w, "No se pudo guardar el pago", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/clientes/"+strconv.Itoa(id), http.StatusSeeOther)
	}))
	
	//para modificar los clientes
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
		renderizar(w,"base.html", "modificarCliente.html", struct {
			Cliente *Cliente
			Error   string
		}{Cliente: cliente, })
	}))

	mux.HandleFunc("POST /clientes/{id}/editar", requiereLogin(func(w http.ResponseWriter, r *http.Request) {
		
		id, err := strconv.Atoi(r.PathValue("id"))

		if err != nil {
			http.Error(w, "ID inválido", http.StatusBadRequest)
			return
		}
		nombre := r.FormValue("nombre")
		apellido := r.FormValue("apellido")
		email  := r.FormValue("email")
		telefono := r.FormValue("telefono")

		if nombre == "" || apellido == "" {
			renderizar(w,"base.html", "clienteNuevo.html", struct{ Error string }{Error: "Nombre y apellido son obligatorios."})
			return
		}
		
		if err := ModificarCliente(db, id, nombre, apellido, email, telefono); err != nil {
			log.Println("Error modificando al cliente:", err)
			renderizar(w,"base.html", "clienteNuevo.html", struct{ Error string }{Error: "No se pudo modificar el cliente."})
			return
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
	}))



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

		compras := ObtenerComprasDeCliente(db, id)
		pagos := ObtenerPagosDeCliente(db, id)

		password := r.FormValue("password")

		if !validarCredenciales("admin", password) {
			renderizar(w,"base.html", "clienteDetalle.html", struct {
				*Cliente
				Compras []Compra
				Pagos   []Pago
				Error   string
			}{
				Cliente: cliente,
				Compras: compras,
				Pagos:   pagos,
				Error:   "Contraseña incorrecta.",
			})
			return
		}

		if err := eliminarCliente(db, id); err != nil {
			renderizar(w,"base.html", "clienteDetalle.html", struct {
				*Cliente
				Compras []Compra
				Pagos   []Pago
				Error   string
			}{
				Cliente: cliente,
				Compras: compras,
				Pagos:   pagos,
				Error:   "No se pudo eliminar el cliente.",
			})
			return
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
	}))

	//seccion de estadisticas
	mux.HandleFunc("GET /estadisticas", requiereLogin(func(w http.ResponseWriter, r *http.Request) {

		TotalClientesUltMes, err := MicroEstadistica(db, "mensual")
		TotalClientesUltAño, err := MicroEstadistica(db, "anual")
		CantClientesMensualesUltAño,err := MacroEstadisticaMensualClientes(db)
		CantVentasMensualesUltAño, err := MacroEstadisticaMensualVentas(db)
		CantCobrosMensualesUltAño, err := MacroEstadisticaMensualCobros(db)

		if err != nil {
			http.NotFound(w, r)
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
			Ventas int `json:"ventas"`
			Cobros int `json:"cobros"`
		}

		datos := make([]DatoGrafico, 0, 12)

		for i := 1; i <= 12; i++ {
			datos = append(datos, DatoGrafico{
				Mes:      meses[i],
				Clientes: CantClientesMensualesUltAño[i],
				Ventas: CantVentasMensualesUltAño[i],
				Cobros: CantCobrosMensualesUltAño[i],
			})
		}

		jsonDatos, err := json.Marshal(datos)
		if err != nil {
			http.NotFound(w, r)
		}

		renderizar(w,"base.html", "estadisticas.html", struct {
			TotalClientesUltMes int
			TotalClientesUltAño int
			DatosGraficoJSON template.JS
		}{
			TotalClientesUltMes: TotalClientesUltMes,
			TotalClientesUltAño: TotalClientesUltAño,
			DatosGraficoJSON: template.JS(jsonDatos),
		})
	}))

	mux.HandleFunc("GET /contacto", requiereLogin(func(w http.ResponseWriter, r *http.Request) {
			renderizar(w, "base.html", "contacto.html", nil)
	}))


	log.Println("Servidor iniciado en http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", mux))

}

func renderizar(w http.ResponseWriter,layout, pagina string, datos any) {
    tmpl, err := template.ParseFiles("templates/sideBar.html", "templates/"+pagina, "templates/"+layout)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    if err := tmpl.ExecuteTemplate(w, "base", datos); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }
}
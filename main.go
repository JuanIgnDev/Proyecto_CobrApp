package main

import (
	"html/template"
	"log"
	"net/http"
	"strconv"
)

func main() {
	// 1. Conectar la DB primero
	db, err := ConectarDB("./data/cobrapp.db")
	if err != nil {
		log.Fatal("ERROR: No se pudo conectar a la db:", err)
	}
	defer db.Close()
	log.Println("Se conectó correctamente a la db!")

	mux := http.NewServeMux()

	// 2. Servir archivos CSS
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// 3. Ruta principal: lista de clientes con su saldo
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		tmpl, err := template.ParseFiles("templates/menuPrincipal.html")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		clientes := ObtenerClientes(db)
		tmpl.Execute(w, struct{ Clientes []Cliente }{Clientes: clientes})
	})

	// 4. Formulario de cliente nuevo (mostrar)
	mux.HandleFunc("GET /cliente_nuevo", func(w http.ResponseWriter, r *http.Request) {
		tmpl, err := template.ParseFiles("templates/clienteNuevo.html")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		tmpl.Execute(w, nil)
	})

	// 5. Formulario de cliente nuevo (guardar)
	mux.HandleFunc("POST /cliente_nuevo", func(w http.ResponseWriter, r *http.Request) {
		nombre := r.FormValue("nombre")
		apellido := r.FormValue("apellido")
		email := r.FormValue("email")
		telefono := r.FormValue("telefono")

		if nombre == "" || apellido == "" {
			tmpl, _ := template.ParseFiles("templates/clienteNuevo.html")
			tmpl.Execute(w, struct{ Error string }{Error: "Nombre y apellido son obligatorios."})
			return
		}

		if err := CrearCliente(db, nombre, apellido, email, telefono); err != nil {
			log.Println("Error creando cliente:", err)
			tmpl, _ := template.ParseFiles("templates/clienteNuevo.html")
			tmpl.Execute(w, struct{ Error string }{Error: "No se pudo guardar el cliente."})
			return
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
	})

	// 6. Detalle de un cliente puntual
	mux.HandleFunc("GET /clientes/{id}", func(w http.ResponseWriter, r *http.Request) {
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

		tmpl, err := template.ParseFiles("templates/clienteDetalle.html")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		tmpl.Execute(w, struct {
			*Cliente
			Compras []Compra
			Pagos   []Pago
		}{Cliente: cliente, Compras: compras, Pagos: pagos})
	})

	// 8. Formulario de venta nueva (mostrar)
	mux.HandleFunc("GET /clientes/{id}/venta_nueva", func(w http.ResponseWriter, r *http.Request) {
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

		tmpl, err := template.ParseFiles("templates/ventaNueva.html")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		tmpl.Execute(w, struct {
			Cliente *Cliente
			Error   string
		}{Cliente: cliente})
	})

	// 9. Formulario de venta nueva (guardar)
	mux.HandleFunc("POST /clientes/{id}/venta_nueva", func(w http.ResponseWriter, r *http.Request) {
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
			tmpl, _ := template.ParseFiles("templates/ventaNueva.html")
			tmpl.Execute(w, struct {
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
	})

	// 10. Formulario de pago nuevo (mostrar)
	mux.HandleFunc("GET /clientes/{id}/pago_nuevo", func(w http.ResponseWriter, r *http.Request) {
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

		tmpl, err := template.ParseFiles("templates/pagoNuevo.html")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		tmpl.Execute(w, struct {
			Cliente *Cliente
			Error   string
		}{Cliente: cliente})
	})

	// 11. Formulario de pago nuevo (guardar)
	mux.HandleFunc("POST /clientes/{id}/pago_nuevo", func(w http.ResponseWriter, r *http.Request) {
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
			tmpl, _ := template.ParseFiles("templates/pagoNuevo.html")
			tmpl.Execute(w, struct {
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
	})

	// 12. Arranca el servidor
	log.Println("Servidor iniciado en http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
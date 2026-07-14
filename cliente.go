package main

import (
	"database/sql"
	"log"
)

type Cliente struct {
	ID       int
	Nombre   string
	Apellido string
	Email    string
	Telefono string
	Saldo    float64 // saldo_pendiente de vista_saldo_cliente
}

// ObtenerClientes trae todos los clientes junto con su saldo pendiente,
// usando la vista vista_saldo_cliente definida en schema.sql.
func ObtenerClientes(db *sql.DB) []Cliente {
	rows, err := db.Query(`
		SELECT cliente_id, nombre, apellido, email, telefono, saldo_pendiente
		FROM vista_saldo_cliente
		ORDER BY apellido
	`)
	if err != nil {
		log.Println("Error consultando clientes:", err)
		return nil
	}
	defer rows.Close()

	var clientes []Cliente
	for rows.Next() {
		var c Cliente
		if err := rows.Scan(&c.ID, &c.Nombre, &c.Apellido, &c.Email, &c.Telefono, &c.Saldo); err != nil {
			log.Println("Error leyendo fila de cliente:", err)
			continue
		}
		clientes = append(clientes, c)
	}
	return clientes
}

// ObtenerClientePorID trae un cliente puntual (para la vista de detalle).
func ObtenerClientePorID(db *sql.DB, id int) (*Cliente, error) {
	var c Cliente
	err := db.QueryRow(`
		SELECT cliente_id, nombre, apellido, email, telefono, saldo_pendiente
		FROM vista_saldo_cliente
		WHERE cliente_id = ?
	`, id).Scan(&c.ID, &c.Nombre, &c.Apellido, &c.Email, &c.Telefono, &c.Saldo)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// CrearCliente inserta un cliente nuevo. email y telefono pueden venir vacíos.
func CrearCliente(db *sql.DB, nombre, apellido, email, telefono string) error {
	_, err := db.Exec(`
		INSERT INTO cliente (nombre, apellido, email, telefono)
		VALUES (?, ?, ?, ?)
	`, nombre, apellido, email, telefono)
	return err
}
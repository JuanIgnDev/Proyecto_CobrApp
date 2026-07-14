package main

import (
	"database/sql"
	"log"
)

type Compra struct {
	ID          int
	ClienteID   int
	Total       float64
	Descripcion string
	Fecha       string
}

// ObtenerComprasDeCliente trae todas las compras de un cliente, más recientes primero.
func ObtenerComprasDeCliente(db *sql.DB, clienteID int) []Compra {
	rows, err := db.Query(`
		SELECT id, cliente_id, total, descripcion, fecha
		FROM compra
		WHERE cliente_id = ?
		ORDER BY fecha DESC
	`, clienteID)
	if err != nil {
		log.Println("Error consultando compras:", err)
		return nil
	}
	defer rows.Close()

	var compras []Compra
	for rows.Next() {
		var c Compra
		if err := rows.Scan(&c.ID, &c.ClienteID, &c.Total, &c.Descripcion, &c.Fecha); err != nil {
			log.Println("Error leyendo fila de compra:", err)
			continue
		}
		compras = append(compras, c)
	}
	return compras
}

// CrearCompra inserta una compra nueva asociada a un cliente.
func CrearCompra(db *sql.DB, clienteID int, total float64, descripcion string) error {
	_, err := db.Exec(`
		INSERT INTO compra (cliente_id, total, descripcion)
		VALUES (?, ?, ?)
	`, clienteID, total, descripcion)
	return err
}
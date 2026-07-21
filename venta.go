package main

import (
	"database/sql"
	"log"
	"strconv"
)

type Venta struct {
	ID          int
	ClienteID   int
	Total       float64
	Descripcion string
	Fecha       string
}

// ObtenerVentasDeCliente trae todas las ventas de un cliente, más recientes primero.
func ObtenerVentasDeCliente(db *sql.DB, clienteID int) []Venta {
	rows, err := db.Query(`
		SELECT id, cliente_id, total, descripcion, fecha
		FROM venta
		WHERE cliente_id = ?
		ORDER BY fecha DESC
	`, clienteID)
	if err != nil {
		log.Println("Error consultando ventas:", err)
		return nil
	}
	defer rows.Close()

	var ventas []Venta
	for rows.Next() {
		var v Venta
		if err := rows.Scan(&v.ID, &v.ClienteID, &v.Total, &v.Descripcion, &v.Fecha); err != nil {
			log.Println("Error leyendo fila de venta:", err)
			continue
		}
		ventas = append(ventas, v)
	}
	return ventas
}

// CrearVenta inserta una venta nueva asociada a un cliente.
// fecha ya viene normalizada (ver fecha.go) — nunca vacía en este punto.
func CrearVenta(db *sql.DB, clienteID int, total float64, descripcion, fecha string) error {
	_, err := db.Exec(`
		INSERT INTO venta (cliente_id, total, descripcion, fecha)
		VALUES (?, ?, ?, ?)
	`, clienteID, total, descripcion, fecha)
	return err
}

// ObtenerVentaPorID trae una venta puntual (para la pantalla de modificar).
func ObtenerVentaPorID(db *sql.DB, ventaID int) (*Venta, error) {
	var v Venta

	err := db.QueryRow(`
		SELECT id, cliente_id, total, descripcion, fecha
		FROM venta
		WHERE id = ?
	`, ventaID).Scan(&v.ID, &v.ClienteID, &v.Total, &v.Descripcion, &v.Fecha)

	if err != nil {
		return nil, err
	}

	return &v, nil
}

// ModificarVenta actualiza total, descripción y fecha de una venta existente.
func ModificarVenta(db *sql.DB, id int, total float64, descripcion, fecha string) error {
	_, err := db.Exec(`
		UPDATE venta
		SET total = ?,
			descripcion = ?,
			fecha = ?
		WHERE id = ?
	`, total, descripcion, fecha, id)

	return err
}

func MacroEstadisticaMensualVentas(db *sql.DB) ([13]int, error) {
	var meses [13]int

	rows, err := db.Query(`
		SELECT
			strftime('%m', fecha) AS mes,
			COUNT(*) AS cantidad
		FROM venta
		WHERE strftime('%Y', fecha) = strftime('%Y', 'now')
		GROUP BY mes
		ORDER BY mes;
	`)
	if err != nil {
		return meses, err
	}
	defer rows.Close()

	for rows.Next() {
		var mes string
		var cantidad int

		if err := rows.Scan(&mes, &cantidad); err != nil {
			return meses, err
		}
		nroMes, _ := strconv.Atoi(mes) // "01" -> 1
		meses[nroMes] = cantidad
	}

	return meses, rows.Err()
}

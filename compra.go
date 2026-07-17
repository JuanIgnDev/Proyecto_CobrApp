package main

import (
	"database/sql"
	"log"
	"strconv"
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


func MacroEstadisticaMensualVentas(db *sql.DB) ([13]int, error) {
	var meses [13]int

	rows, err := db.Query(`
		SELECT
			strftime('%m', fecha) AS mes,
			COUNT(*) AS cantidad
		FROM compra
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
		//ARRANCA DE 1 Y VA HASTA 12
		nroMes, _ := strconv.Atoi(mes) // "01" -> 1
		meses[nroMes] = cantidad
	}

	return meses, rows.Err()
}
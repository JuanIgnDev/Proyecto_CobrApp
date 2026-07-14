package main

import (
	"database/sql"
	"log"
)

type Pago struct {
	ID          int
	ClienteID   int
	Monto       float64
	Observacion string
	Fecha       string
}

// ObtenerPagosDeCliente trae todos los pagos de un cliente, más recientes primero.
func ObtenerPagosDeCliente(db *sql.DB, clienteID int) []Pago {
	rows, err := db.Query(`
		SELECT id, cliente_id, monto, observacion, fecha
		FROM pago
		WHERE cliente_id = ?
		ORDER BY fecha DESC
	`, clienteID)
	if err != nil {
		log.Println("Error consultando pagos:", err)
		return nil
	}
	defer rows.Close()

	var pagos []Pago
	for rows.Next() {
		var p Pago
		if err := rows.Scan(&p.ID, &p.ClienteID, &p.Monto, &p.Observacion, &p.Fecha); err != nil {
			log.Println("Error leyendo fila de pago:", err)
			continue
		}
		pagos = append(pagos, p)
	}
	return pagos
}

// CrearPago inserta un pago nuevo asociado a un cliente.
func CrearPago(db *sql.DB, clienteID int, monto float64, observacion string) error {
	_, err := db.Exec(`
		INSERT INTO pago (cliente_id, monto, observacion)
		VALUES (?, ?, ?)
	`, clienteID, monto, observacion)
	return err
}
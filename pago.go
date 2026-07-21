package main

import (
	"database/sql"
	"log"
	"strconv"
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


func MacroEstadisticaMensualCobros(db *sql.DB) ([13]int, error) {
	var meses [13]int

	rows, err := db.Query(`
		SELECT
			strftime('%m', fecha) AS mes,
			COUNT(*) AS cantidad
		FROM pago
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

func ObtenerPagoPorId(db *sql.DB, pagoId int) (*Pago, error){

	var pago Pago

	err := db.QueryRow(`
		SELECT id, cliente_id, monto, observacion, fecha
		FROM pago
		WHERE id = ?
	`, pagoId).Scan(&pago.ID, &pago.ClienteID, &pago.Monto, &pago.Observacion,&pago.Fecha)
	
	if err != nil {
		return nil, err
	}

	return &pago, nil
}

func ModificarPago(db *sql.DB, id int, monto float64, observacion string) error{

	_, err := db.Exec(`
		UPDATE pago
		SET monto = ?,
			observacion = ?
		WHERE id = ?
	`, monto, observacion, id)

	return err
}
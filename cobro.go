package main

import (
	"database/sql"
	"log"
	"strconv"
)

type Cobro struct {
	ID          int
	ClienteID   int
	Monto       float64
	Observacion string
	Fecha       string
}

// ObtenerCobrosDeCliente trae todos los cobros de un cliente, más recientes primero.
func ObtenerCobrosDeCliente(db *sql.DB, clienteID int) []Cobro {
	rows, err := db.Query(`
		SELECT id, cliente_id, monto, observacion, fecha
		FROM cobro
		WHERE cliente_id = ?
		ORDER BY fecha DESC
	`, clienteID)
	if err != nil {
		log.Println("Error consultando cobros:", err)
		return nil
	}
	defer rows.Close()

	var cobros []Cobro
	for rows.Next() {
		var c Cobro
		if err := rows.Scan(&c.ID, &c.ClienteID, &c.Monto, &c.Observacion, &c.Fecha); err != nil {
			log.Println("Error leyendo fila de cobro:", err)
			continue
		}
		cobros = append(cobros, c)
	}
	return cobros
}

// CrearCobro inserta un cobro nuevo asociado a un cliente.
// fecha ya viene normalizada (ver fecha.go) — nunca vacía en este punto.
func CrearCobro(db *sql.DB, clienteID int, monto float64, observacion, fecha string) error {
	_, err := db.Exec(`
		INSERT INTO cobro (cliente_id, monto, observacion, fecha)
		VALUES (?, ?, ?, ?)
	`, clienteID, monto, observacion, fecha)
	return err
}

// ObtenerCobroPorID trae un cobro puntual (para la pantalla de modificar).
func ObtenerCobroPorID(db *sql.DB, cobroID int) (*Cobro, error) {
	var c Cobro

	err := db.QueryRow(`
		SELECT id, cliente_id, monto, observacion, fecha
		FROM cobro
		WHERE id = ?
	`, cobroID).Scan(&c.ID, &c.ClienteID, &c.Monto, &c.Observacion, &c.Fecha)

	if err != nil {
		return nil, err
	}

	return &c, nil
}

// ModificarCobro actualiza monto, observación y fecha de un cobro existente.
func ModificarCobro(db *sql.DB, id int, monto float64, observacion, fecha string) error {
	_, err := db.Exec(`
		UPDATE cobro
		SET monto = ?,
			observacion = ?,
			fecha = ?
		WHERE id = ?
	`, monto, observacion, fecha, id)

	return err
}

func MacroEstadisticaMensualCobros(db *sql.DB) ([13]int, error) {
	var meses [13]int

	rows, err := db.Query(`
		SELECT
			strftime('%m', fecha) AS mes,
			COUNT(*) AS cantidad
		FROM cobro
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

func eliminarCobro(db *sql.DB, idCobro int) error {
	_, err := db.Exec(`
		DELETE FROM cobro
		WHERE id = ?
	`, idCobro)

	return err
}
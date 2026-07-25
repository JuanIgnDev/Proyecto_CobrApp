package main

import (
	"database/sql"
)

type Configuracion struct {
	DiasAlerta int
	MensajeWP  string
	Minutos_inactividad int
}

// ObtenerConfiguracion lee los ajustes de la base de datos
func ObtenerConfiguracion(db *sql.DB) Configuracion {
	var c Configuracion

	// Buscamos siempre la fila con id = 1
	err := db.QueryRow(`
		SELECT dias_alerta, mensaje_wp, minutos_inactividad
		FROM configuracion
		WHERE id = 1
	`).Scan(
		&c.DiasAlerta,
		&c.MensajeWP,
		&c.Minutos_inactividad,
	)

	if err != nil {
		// Si por algún motivo falla, devolvemos valores por defecto seguros
		return Configuracion{
			DiasAlerta:          45,
			MensajeWP:           "Hola {nombre}, te escribo para recordarte que tenés un saldo pendiente de ${saldo}.",
			Minutos_inactividad: 30,
		}
	}

	return c
}

// GuardarConfiguracion actualiza la fila 1 con los nuevos datos
func GuardarConfiguracion(db *sql.DB, dias int, mensaje string, minutos_inactividad int) error {
	_, err := db.Exec(`
		UPDATE configuracion 
		SET dias_alerta = ?, mensaje_wp = ?, minutos_inactividad = ?
		WHERE id = 1
	`, dias, mensaje, minutos_inactividad)
	return err
}
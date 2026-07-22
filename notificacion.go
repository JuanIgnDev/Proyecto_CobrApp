package main

import (
	"database/sql"
	"time"
)

type Notificacion struct {
	ID int
	Cliente_id int
	Tipo string
	Titulo string
	Fecha_referencia string
	Estado Estado
	Fecha_creacion string
}

type Estado string

const (
	Pendiente Estado = "pendiente"
	Vista Estado = "vista"
	Resuelta Estado = "resuelta"
	Eliminada Estado = "eliminada"
)

// 1. Sincronizar notificaciones
func SincronizarNotificaciones(db *sql.DB) error {

	clientes := ObtenerClientes(db)
	if clientes == nil {
		return nil
	}

	for _, cliente := range clientes {

		// ¿Tiene deuda?
		if cliente.Saldo <= 0 {
			continue
		}

		// Obtener fecha de referencia:
		// último cobro, o primera venta si nunca cobró
		fechaReferencia, err := ObtenerFechaReferencia(
			db,
			cliente.ID,
		)
		if err != nil {
			// Si no tiene ventas tampoco, no se puede obtener fecha
			if err == sql.ErrNoRows {
				continue
			}
			return err
		}

		// ¿Pasaron 45 días?
		pasaron, err := Pasaron45Dias(fechaReferencia)
		if err != nil {
			return err
		}

		if !pasaron {
			continue
		}

		// ¿Ya existe la notificación?
		existe, err := ExisteNotificacion(
			db,
			cliente.ID,
			fechaReferencia,
		)
		if err != nil {
			return err
		}

		// Crear solamente si no existe
		if !existe {
			err := CrearNotificacion(
				db,
				cliente.ID,
				fechaReferencia,
			)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
 
func marcarNotificacionComoVista(db *sql.DB) error{
	_, err := db.Exec(`
		UPDATE notificacion
		SET estado = 'vista'
		WHERE estado = 'pendiente'
	`)
	return err
}

func ObtenerFechaReferencia(db *sql.DB, clienteId int) (string, error) {

	var fechaRef string

	// Buscamos el último cobro
	err := db.QueryRow(`
		SELECT fecha
		FROM cobro
		WHERE cliente_id = ?
		ORDER BY fecha DESC
		LIMIT 1
	`, clienteId).Scan(&fechaRef)

	if err == nil {
		return fechaRef, nil
	}

	// Si el cliente nunca cobró, buscamos su primera venta
	if err == sql.ErrNoRows {

		err = db.QueryRow(`
			SELECT fecha
			FROM venta
			WHERE cliente_id = ?
			ORDER BY fecha ASC
			LIMIT 1
		`, clienteId).Scan(&fechaRef)

		if err != nil {
			return "", err
		}

		return fechaRef, nil
	}

	return "", err
}

func Pasaron45Dias(fechaRef string) (bool, error) {

	const formato = "2006-01-02 15:04:05"

	fecha, err := time.Parse(formato, fechaRef)
	if err != nil {
		return false, err
	}

	fechaLimite := fecha.AddDate(0, 0, 45)

	return time.Now().After(fechaLimite), nil
}

func ExisteNotificacion(
	db *sql.DB,
	clienteId int,
	fechaRef string,
) (bool, error) {

	var existe bool

	err := db.QueryRow(`
		SELECT EXISTS (
			SELECT 1
			FROM notificacion
			WHERE cliente_id = ?
			AND fecha_referencia = ?
		)
	`, clienteId, fechaRef).Scan(&existe)

	return existe, err
}

func CrearNotificacion(
	db *sql.DB,
	clienteId int,
	fechaRef string,
) error {

	_, err := db.Exec(`
		INSERT INTO notificacion (
			cliente_id,
			tipo,
			titulo,
			fecha_referencia
		)
		VALUES (?, ?, ?, ?)
	`,
		clienteId,
		"deuda_vencida",
		"El cliente tiene una deuda pendiente hace más de 45 días",
		fechaRef,
	)

	return err
}


func ObtenerNotificacionesValidas(db *sql.DB) ([]Notificacion, error) {

	notificaciones, err := ObtenerNotificacionesPendientes(db)
	if err != nil {
		return nil, err
	}

	var validas []Notificacion

	for _, notificacion := range notificaciones {

		fechaReferencia, err := ObtenerFechaReferencia(
			db,
			notificacion.Cliente_id,
		)

		if err != nil {
			return nil, err
		}

		if fechaReferencia == notificacion.Fecha_referencia {
			validas = append(validas, notificacion)
		}
	}

	return validas, nil
}

func ObtenerNotificacionesPendientes(
	db *sql.DB,
) ([]Notificacion, error) {

	rows, err := db.Query(`
		SELECT id, cliente_id, tipo, titulo,
		       fecha_referencia, estado, fecha_creacion
		FROM notificacion
		WHERE estado IN ('pendiente', 'vista')
		ORDER BY estado ASC, id DESC
	`)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notificaciones []Notificacion

	for rows.Next() {

		var n Notificacion

		err := rows.Scan(
			&n.ID,
			&n.Cliente_id,
			&n.Tipo,
			&n.Titulo,
			&n.Fecha_referencia,
			&n.Estado,
			&n.Fecha_creacion,
		)

		if err != nil {
			return nil, err
		}

		notificaciones = append(notificaciones, n)
	}

	return notificaciones, rows.Err()
}
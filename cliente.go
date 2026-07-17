package main

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
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
		var emailDB sql.NullString
		var telefonoDB sql.NullString

		if err := rows.Scan(&c.ID, &c.Nombre, &c.Apellido, &emailDB, &telefonoDB, &c.Saldo); err != nil {
			log.Println("Error leyendo fila de cliente:", err)
			continue
		}

		if telefonoDB.Valid {
			c.Telefono = telefonoDB.String
		}

		if emailDB.Valid {
			c.Email = emailDB.String
		}

		clientes = append(clientes, c)
	}
	return clientes
}

// ObtenerClientePorID trae un cliente puntual (para la vista de detalle).
func ObtenerClientePorID(db *sql.DB, id int) (*Cliente, error) {
	var c Cliente

	var emailDB sql.NullString
	var telefonoDB sql.NullString

	err := db.QueryRow(`
		SELECT cliente_id, nombre, apellido, email, telefono, saldo_pendiente
		FROM vista_saldo_cliente
		WHERE cliente_id = ?
	`, id).Scan(&c.ID, &c.Nombre, &c.Apellido, &emailDB, &telefonoDB, &c.Saldo)
	
	if err != nil {
		return nil, err
	}

	if emailDB.Valid {
		c.Email = emailDB.String
	}

	if telefonoDB.Valid {
		c.Telefono = telefonoDB.String
	}

	return &c, nil
}

// CrearCliente inserta un cliente nuevo. email y telefono pueden venir vacíos.
func CrearCliente(db *sql.DB, nombre, apellido, email, telefono string) error {
	
	var emailDB sql.NullString
	if email != "" {
		emailDB.String = email
		emailDB.Valid = true
	}

	var telefonoDB sql.NullString
	if telefono != "" {
		telefonoDB.String = telefono
		telefonoDB.Valid = true
	}
	
	_, err := db.Exec(`
		INSERT INTO cliente (nombre, apellido, email, telefono)
		VALUES (?, ?, ?, ?)
	`, nombre, apellido, emailDB, telefonoDB)
	return err
}

// ModificarCliente actualiza los datos de un cliente.
// email y telefono pueden venir vacíos.
func ModificarCliente(db *sql.DB, id int, nombre, apellido, email, telefono string) error {

	var emailDB sql.NullString
	if email != "" {
		emailDB.String = email
		emailDB.Valid = true
	}

	var telefonoDB sql.NullString
	if telefono != "" {
		telefonoDB.String = telefono
		telefonoDB.Valid = true
	}

	_, err := db.Exec(`
		UPDATE cliente
		SET nombre = ?,
			apellido = ?,
			email = ?,
			telefono = ?
		WHERE id = ?
	`, nombre, apellido, emailDB, telefonoDB, id)

	return err
}

func eliminarCliente(db *sql.DB, id int) error {
	_, err := db.Exec(`
		DELETE FROM cliente
		WHERE id = ?
	`, id)

	return err
}


//FUNCION PARA LAS ESTADISTICAS
func MicroEstadistica(db *sql.DB, periodo string) (int, error) {
	var formato string

	switch periodo {
	case "anual":
		formato = "%Y"
	case "mensual":
		formato = "%Y-%m"
	default:
		return 0, fmt.Errorf("periodo inválido")
	}

	var cantidad int

	err := db.QueryRow(`
		SELECT COUNT(*)
		FROM cliente
		WHERE strftime(?, creado_en) = strftime(?, 'now');
	`, formato, formato).Scan(&cantidad)

	return cantidad, err
}

func MacroEstadisticaMensualClientes(db *sql.DB) ([13]int, error) {
	var meses [13]int

	rows, err := db.Query(`
		SELECT
			strftime('%m', creado_en) AS mes,
			COUNT(*) AS cantidad
		FROM cliente
		WHERE strftime('%Y', creado_en) = strftime('%Y', 'now')
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
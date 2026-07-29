package main

import "database/sql"

// RegistrarAviso deja constancia de que se avisó a un cliente (por
// WhatsApp o email) sobre su saldo pendiente. Se llama desde el
// frontend justo después de abrir el link de wa.me o el mailto.
func RegistrarAviso(db *sql.DB, clienteID int, tipo string) error {
	_, err := db.Exec(`
		INSERT INTO aviso (cliente_id, tipo)
		VALUES (?, ?)
	`, clienteID, tipo)
	return err
}

// ObtenerUltimoAviso trae la fecha (YYYY-MM-DD) del último aviso enviado
// a un cliente. Si nunca se le mandó ninguno, devuelve "Sin avisos" en
// vez de un error, para que la ficha del cliente lo pueda mostrar directo.
func ObtenerUltimoAviso(db *sql.DB, clienteID int) string {
	var fecha string

	err := db.QueryRow(`
		SELECT fecha
		FROM aviso
		WHERE cliente_id = ?
		ORDER BY fecha DESC
		LIMIT 1
	`, clienteID).Scan(&fecha)

	if err != nil {
		return "Sin avisos"
	}

	if len(fecha) >= 10 {
		return fecha[:10]
	}
	return fecha
}
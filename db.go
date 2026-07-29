package main

import (
	"database/sql"
	"os"

	_ "modernc.org/sqlite"
)

func ConectarDB(ruta string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", ruta)
	if err != nil {
		return nil, err
	}

	if _, err := db.Exec(`DROP VIEW IF EXISTS vista_saldo_cliente;`); err != nil {
		return nil, err
	}

	esquema, err := os.ReadFile("./data/schema.sql")
	if err != nil {
		return nil, err
	}

	if _, err := db.Exec(string(esquema)); err != nil {
		return nil, err
	}

	return db, nil
}
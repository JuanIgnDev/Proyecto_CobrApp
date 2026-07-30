package main

import (
	"database/sql"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// ConectarDB abre (o crea, si no existe) el archivo sqlite en "ruta".
//
// Acá está la respuesta a "cómo le indico a sqlite3 dónde crearse":
// sql.Open NO valida que el archivo exista. Si "ruta" apunta a un archivo
// que no está, el propio driver de sqlite lo crea vacío en ese momento.
// Por eso alcanza con:
//  1. Asegurarnos de que la CARPETA que contiene ese archivo exista
//     (sqlite crea el archivo .db, pero no crea carpetas intermedias).
//  2. Pasarle la ruta que querramos: relativa, absoluta, en Windows con
//     backslashes o en Linux con slashes — filepath.Join ya se encarga
//     de usar el separador correcto según el sistema operativo.
func ConectarDB(ruta string) (*sql.DB, error) {
	carpeta := filepath.Dir(ruta)
	if err := os.MkdirAll(carpeta, 0755); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", ruta)
	if err != nil {
		return nil, err
	}

	if _, err := db.Exec(`DROP VIEW IF EXISTS vista_saldo_cliente;`); err != nil {
		return nil, err
	}

	// Antes leíamos data/schema.sql del disco con os.ReadFile. Ahora usamos
	// esquemaSQL, la variable embebida en el binario (ver embebidos.go),
	// así el schema viaja siempre pegado al ejecutable y no depende de que
	// exista la carpeta data/ con ese archivo en el destino de instalación.
	if _, err := db.Exec(esquemaSQL); err != nil {
		return nil, err
	}

	return db, nil
}

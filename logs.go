package main

import (
	"io"
	"log"
	"os"
	"path/filepath"
)

// ConfigurarLogs hace que todo lo que se escriba con log.Println / log.Fatal
// / etc. en el resto del programa vaya a DOS lugares a la vez: la consola
// (útil mientras desarrollás) y un archivo en logs/cobrapp.log (útil
// cuando el programa ya está instalado en la PC de un cliente y algo
// falla: le pedís el archivo de logs en vez de pedirle que te describa
// "una pantalla rara que apareció").
//
// Devolvemos el *os.File para que main() pueda cerrarlo prolijamente con
// defer antes de que el programa termine.
func ConfigurarLogs() (*os.File, error) {
	dir, err := dirEjecutable()
	if err != nil {
		return nil, err
	}

	logsDir := filepath.Join(dir, "logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return nil, err
	}

	rutaLog := filepath.Join(logsDir, "cobrapp.log")

	// O_APPEND: cada corrida agrega al final, no borra lo anterior.
	// O_CREATE: si es la primera vez, lo crea.
	archivo, err := os.OpenFile(rutaLog, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}

	// io.MultiWriter manda cada línea a ambos destinos con una sola
	// llamada a log.Println, no hace falta duplicar cada log.
	multi := io.MultiWriter(os.Stdout, archivo)
	log.SetOutput(multi)

	// Fecha, hora y el archivo:línea donde se generó el log. Esto último
	// (Lshortfile) es oro cuando un error viene de un log.Println perdido
	// en medio de un handler HTTP y no sabés cuál.
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	return archivo, nil
}

package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const maxBackups = 30

// carpetaBackups devuelve (y crea si hace falta) la carpeta de backups.
//
// A diferencia de la base de datos (que vive AL LADO del ejecutable, para
// que la app sea portable: copiás la carpeta entera y te la llevás), los
// backups van en os.UserConfigDir():
//   - Windows: C:\Users\<usuario>\AppData\Roaming
//   - Linux:   ~/.config
//
// Es LA carpeta pensada por el sistema operativo para que cada programa
// guarde sus datos de forma persistente y por-usuario. La ventaja frente
// a guardarlos junto al ejecutable: si alguien reinstala CobrApp pisando
// la carpeta del programa (o la borra), los backups sobreviven porque
// están en otro lado.
func carpetaBackups() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	ruta := filepath.Join(base, "CobrApp", "backups")
	if err := os.MkdirAll(ruta, 0755); err != nil {
		return "", err
	}

	return ruta, nil
}

// HacerBackup copia el archivo de la base de datos con un nombre con
// timestamp (backup_20260730_143000.db) y después limpia los backups
// viejos para no acumular para siempre.
//
// El formato YYYYMMDD_HHMMSS no es capricho: dos corridas en el mismo
// segundo son prácticamente imposibles en este uso, así que nunca se
// "pisan" (nunca dos backups terminan con el mismo nombre y se
// sobreescriben sin querer) — que es justo lo que pedían tus notas.
func HacerBackup(rutaDB string) error {
	carpeta, err := carpetaBackups()
	if err != nil {
		return fmt.Errorf("no se pudo preparar la carpeta de backups: %w", err)
	}

	nombre := fmt.Sprintf("backup_%s.db", time.Now().Format("20060102_150405"))
	destino := filepath.Join(carpeta, nombre)

	if err := copiarArchivo(rutaDB, destino); err != nil {
		return fmt.Errorf("no se pudo copiar la base de datos: %w", err)
	}

	log.Println("Backup creado:", destino)

	return limpiarBackupsViejos(carpeta)
}

func copiarArchivo(origen, destino string) error {
	in, err := os.Open(origen)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(destino)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

// limpiarBackupsViejos deja solo los últimos maxBackups backups, borrando
// el resto. Como el nombre del archivo empieza con la fecha en formato
// YYYYMMDD_HHMMSS, ordenar los nombres alfabéticamente es lo mismo que
// ordenarlos cronológicamente — no hace falta parsear fechas.
func limpiarBackupsViejos(carpeta string) error {
	entradas, err := os.ReadDir(carpeta)
	if err != nil {
		return err
	}

	var backups []os.DirEntry
	for _, e := range entradas {
		if !e.IsDir() && strings.HasPrefix(e.Name(), "backup_") && strings.HasSuffix(e.Name(), ".db") {
			backups = append(backups, e)
		}
	}

	sort.Slice(backups, func(i, j int) bool {
		return backups[i].Name() < backups[j].Name()
	})

	if len(backups) <= maxBackups {
		return nil // todavía no hay que borrar nada
	}

	aBorrar := backups[:len(backups)-maxBackups] // los más viejos, al principio de la lista ordenada
	for _, e := range aBorrar {
		ruta := filepath.Join(carpeta, e.Name())
		if err := os.Remove(ruta); err != nil {
			log.Println("No se pudo borrar backup viejo:", ruta, err)
			continue
		}
		log.Println("Backup viejo eliminado (rotación, quedan los últimos", maxBackups, "):", ruta)
	}

	return nil
}

// IniciarBackupsPeriodicos lanza un backup ya al arrancar (por si el
// programa se cierra mal ese mismo día, tenés algo reciente) y después
// uno cada 24hs mientras el programa esté corriendo. Se llama con
// "go IniciarBackupsPeriodicos(rutaDB)" para que no bloquee el arranque
// del servidor.
func IniciarBackupsPeriodicos(rutaDB string) {
	if err := HacerBackup(rutaDB); err != nil {
		log.Println("Error haciendo el backup inicial:", err)
	}

	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		if err := HacerBackup(rutaDB); err != nil {
			log.Println("Error haciendo backup periódico:", err)
		}
	}
}

package main

import (
	"os"
	"path/filepath"
)

// dirEjecutable devuelve la carpeta donde está el binario (el .exe en
// Windows, el ejecutable en Linux) — NO el directorio desde el que se
// lo invocó. Es la diferencia entre:
//
//	os.Executable()      -> C:\Programas\CobrApp\cobrapp.exe (fijo, confiable)
//	os.Getwd()            -> depende de desde dónde lo ejecutaste (poco confiable)
//
// La usamos para que "data/", "config.json" y "logs/" queden siempre
// pegados al ejecutable, sin importar desde qué carpeta lo abran
// (por ejemplo, con doble clic desde el Explorador de Windows).
//
// EvalSymlinks es por las dudas de que el ejecutable se abra a través
// de un symlink (común en Linux); sin esto, filepath.Dir podría devolver
// la carpeta del symlink y no la del binario real.
func dirEjecutable() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}

	real, err := filepath.EvalSymlinks(exe)
	if err != nil {
		real = exe // si falla, seguimos con lo que teníamos
	}

	return filepath.Dir(real), nil
}

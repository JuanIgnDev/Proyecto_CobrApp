package main

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
)

// ConfigApp son los valores que NO queremos "quemados" (hardcodeados) en
// el binario. "Quemar" un valor es escribirlo fijo en el código, tipo
// ListenAndServe(":8080"): si mañana el 8080 está ocupado, o el cliente
// quiere que el título de la app diga otra cosa, hay que TOCAR EL CÓDIGO
// y volver a compilar. Con config.json, el usuario final edita un archivo
// de texto y listo, no necesita ni saber que existe Go.
type ConfigApp struct {
	Puerto  int    `json:"puerto"`
	Empresa string `json:"empresa"`
}

func rutaConfigApp() (string, error) {
	dir, err := dirEjecutable()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// CargarConfigApp lee config.json de al lado del ejecutable. Si no existe
// (primera vez que se corre la app, o el usuario lo borró sin querer), lo
// crea con valores por defecto en lugar de romperse.
func CargarConfigApp() ConfigApp {
	defecto := ConfigApp{Puerto: 8080, Empresa: "CobrApp"}

	ruta, err := rutaConfigApp()
	if err != nil {
		log.Println("No se pudo resolver la ruta de config.json, uso valores por defecto:", err)
		return defecto
	}

	datos, err := os.ReadFile(ruta)
	if err != nil {
		log.Println("No existe config.json todavía, lo creo con valores por defecto en:", ruta)
		if err := guardarConfigApp(ruta, defecto); err != nil {
			log.Println("No se pudo crear config.json:", err)
		}
		return defecto
	}

	var cfg ConfigApp
	if err := json.Unmarshal(datos, &cfg); err != nil {
		log.Println("config.json tiene un formato inválido, uso valores por defecto:", err)
		return defecto
	}

	// Si el archivo existe pero le falta algún campo (o vino en 0/""),
	// completamos con el valor por defecto para no arrancar en un
	// estado raro (puerto 0, por ejemplo).
	if cfg.Puerto == 0 {
		cfg.Puerto = defecto.Puerto
	}
	if cfg.Empresa == "" {
		cfg.Empresa = defecto.Empresa
	}

	return cfg
}

func guardarConfigApp(ruta string, cfg ConfigApp) error {
	datos, err := json.MarshalIndent(cfg, "", "    ")
	if err != nil {
		return err
	}
	return os.WriteFile(ruta, datos, 0644)
}

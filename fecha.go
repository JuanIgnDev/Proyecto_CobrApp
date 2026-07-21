package main

import "time"

// normalizarFecha toma lo que vino del <input type="date"> (formato "2006-01-02")
// y lo devuelve en el mismo formato que usa la DB para guardar fechas
// ("2006-01-02 15:04:05"), para que el orden alfabético siga siendo
// el mismo que el orden cronológico, sin importar si la fecha la puso
// el usuario o si se generó sola.
//
// Si el usuario no eligió ninguna fecha (campo vacío), usa el momento
// actual — así, tanto al crear como al modificar, "no tocar el campo"
// significa "usar ahora".
func normalizarFecha(fechaForm string) (string, error) {
	if fechaForm == "" {
		return time.Now().Format("2006-01-02 15:04:05"), nil
	}

	fechaParseada, err := time.Parse("2006-01-02", fechaForm)
	if err != nil {
		return "", err
	}

	return fechaParseada.Format("2006-01-02 15:04:05"), nil
}

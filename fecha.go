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

// DiasDesde calcula cuántos días pasaron desde una fecha guardada en el
// mismo formato que usa la DB ("2006-01-02 15:04:05"). Se usa para
// "días en deuda" en la ficha del cliente. Si la fecha viene vacía o
// mal formada, devuelve 0 en vez de cortar el render de la página.
func DiasDesde(fechaRef string) int {
	const formato = "2006-01-02 15:04:05"

	fecha, err := time.Parse(formato, fechaRef)
	if err != nil {
		return 0
	}

	dias := int(time.Since(fecha).Hours() / 24)
	if dias < 0 {
		dias = 0
	}
	return dias
}
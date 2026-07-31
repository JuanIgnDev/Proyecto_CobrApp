package main

import (
	"fmt"
	"net"
	"os/exec"
	"runtime"
)

// puertoLibre intenta usar "preferido" (el que viene de config.json).
// Si está ocupado (por ejemplo porque ya hay una instancia de CobrApp
// corriendo, o algún otro programa lo está usando), le pedimos al
// sistema operativo que nos asigne uno libre.
//
// El truco es net.Listen("tcp", ":0"): el puerto ":0" es una convención
// de los sockets que significa "dame cualquier puerto disponible". El
// SO elige uno, lo reserva, y nosotros lo leemos de l.Addr(). Cerramos
// ese listener de prueba enseguida porque el que se va a usar de verdad
// lo abre http.ListenAndServe más adelante — esto solo es para "reservar
// y consultar" cuál está libre.
func puertoLibre(preferido int) (int, error) {
	direccion := fmt.Sprintf(":%d", preferido)

	if l, err := net.Listen("tcp", direccion); err == nil {
		l.Close()
		return preferido, nil
	}

	l, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, err
	}
	defer l.Close()

	puerto := l.Addr().(*net.TCPAddr).Port
	return puerto, nil
}

// abrirNavegador abre la URL en el navegador por defecto del usuario.
// No hay una forma portable en la librería estándar de Go para esto
// porque, a diferencia de crear un archivo o leer una carpeta, "abrir el
// navegador" es un concepto 100% del sistema operativo: cada uno tiene
// su propio comando para decirle al shell "abrí esto con el programa que
// corresponda".
func abrirNavegador(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		// rundll32 con url.dll es el mecanismo estándar de Windows para
		// pedirle al shell que abra una URL con el navegador por defecto.
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		// Linux (y la mayoría de los unix con entorno gráfico) usan
		// xdg-open, parte del estándar freedesktop.org.
		cmd = exec.Command("xdg-open", url)
	}

	// Start() (no Run()) porque no queremos esperar a que el navegador
	// se cierre para seguir ejecutando nuestro programa.
	return cmd.Start()
}

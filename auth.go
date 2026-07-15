package main

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"sync"
)

// ---- Credenciales del único usuario ----
// App local, un solo usuario, sin exposición a internet: alcanza con
// comparar directo. Si en algún momento vas a exponer esto en una red
// o querés más seguridad, ahí sí conviene pasar a un hash (bcrypt).
const (
	usuarioValido  = "admin"      // cambiá esto por el usuario que quieras
	passwordValida = "123" // cambiá esto por tu contraseña
)

// validarCredenciales compara usuario y contraseña.
// Usamos subtle.ConstantTimeCompare (de la librería estándar) para la
// contraseña, que evita filtrar por "cuánto tarda la comparación" cuántos
// caracteres acertaste — buena práctica gratis, sin dependencias externas.
func validarCredenciales(usuario, password string) bool {
	usuarioOK := usuario == usuarioValido
	passwordOK := subtle.ConstantTimeCompare([]byte(password), []byte(passwordValida)) == 1
	return usuarioOK && passwordOK
}

// ---- Sesiones en memoria ----
var (
	sesiones   = map[string]bool{}
	sesionesMu sync.Mutex
)

func generarToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// crearSesion genera un token, lo guarda en memoria, y lo manda como cookie.
func crearSesion(w http.ResponseWriter) error {
	token, err := generarToken()
	if err != nil {
		return err
	}

	sesionesMu.Lock()
	sesiones[token] = true
	sesionesMu.Unlock()

	http.SetCookie(w, &http.Cookie{
		Name:     "sesion",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		// Sin Expires ni MaxAge => cookie de sesión: se borra al cerrar el navegador
	})
	return nil
}

func sesionValida(r *http.Request) bool {
	cookie, err := r.Cookie("sesion")
	if err != nil {
		return false
	}
	sesionesMu.Lock()
	defer sesionesMu.Unlock()
	return sesiones[cookie.Value]
}

func cerrarSesion(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie("sesion"); err == nil {
		sesionesMu.Lock()
		delete(sesiones, cookie.Value)
		sesionesMu.Unlock()
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "sesion",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
}

func requiereLogin(siguiente http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !sesionValida(r) {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		siguiente(w, r)
	}
}
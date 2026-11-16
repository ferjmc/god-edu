// Package httpkit contiene el plumbing HTTP genérico que usan los
// controladores de todos los dominios — no le pertenece a ninguno, por eso
// vive acá y no dentro de internal/<dominio>, al mismo nivel que auth, db,
// email y storage.
package httpkit

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5/middleware"

	"github.com/ferjmc/god-edu/api/db"
)

func WriteJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func WriteError(w http.ResponseWriter, status int, message string) {
	WriteJSON(w, status, map[string]string{"error": message})
}

// RequireNonEmpty recorta espacios y devuelve 400 si el resultado queda
// vacío. label va en el mensaje ("el <label> es obligatorio") — patrón
// repetido en varios campos de texto obligatorios del panel admin.
func RequireNonEmpty(w http.ResponseWriter, value, label string) (string, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		WriteError(w, http.StatusBadRequest, fmt.Sprintf("el %s es obligatorio", label))
		return "", false
	}
	return value, true
}

// ClientIP devuelve la IP del cliente: la que dejó en el contexto
// middleware.ClientIPFromHeader a partir de CF-Connecting-IP (ver comentario
// en main.go). Sin ese header — desarrollo local, sin Cloudflare adelante —
// cae a r.RemoteAddr. Si por algo RemoteAddr no tiene el formato host:port
// esperado, devuelve el valor crudo tal cual en vez de fallar: mejor no
// romper el registro por esto.
func ClientIP(r *http.Request) string {
	if ip := middleware.GetClientIP(r.Context()); ip != "" {
		return ip
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// HandleNotFound escribe 404 si err es db.ErrNotFound y devuelve true (el
// caller debe hacer return). Si no, devuelve false y el caller sigue con
// su propio log + 500.
func HandleNotFound(w http.ResponseWriter, err error, notFoundMsg string) bool {
	if errors.Is(err, db.ErrNotFound) {
		WriteError(w, http.StatusNotFound, notFoundMsg)
		return true
	}
	return false
}

// Health responds with a simple JSON status payload, used for uptime checks
// and to confirm the API is reachable during local development.
func Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

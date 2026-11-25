// Package httpkit contiene el plumbing HTTP genérico que usan los
// controladores de todos los dominios — no le pertenece a ninguno, por eso
// vive acá y no dentro de internal/<dominio>, al mismo nivel que auth, db,
// email y storage.
package httpkit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

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

// Pinger es lo único que Health necesita de la base: *pgxpool.Pool lo
// cumple. Con una interfaz httpkit no importa pgx y se puede testear con un
// fake.
type Pinger interface {
	Ping(ctx context.Context) error
}

// Health responde {"status":"ok"} solo si la base contesta un ping; si no,
// 503 {"status":"unavailable"}. El detalle del error va al log, nunca a la
// respuesta (el endpoint es público). El timeout evita que una base colgada
// deje colgado el chequeo de uptime.
func Health(db Pinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := db.Ping(ctx); err != nil {
			log.Printf("health: falló el ping a la base: %v", err)
			WriteJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "unavailable"})
			return
		}
		WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}

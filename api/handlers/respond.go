package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/ferjmc/god-edu/api/db"
)

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// requireNonEmpty recorta espacios y devuelve 400 si el resultado queda
// vacío. label va en el mensaje ("el <label> es obligatorio") — patrón
// repetido en varios campos de texto obligatorios del panel admin.
func requireNonEmpty(w http.ResponseWriter, value, label string) (string, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("el %s es obligatorio", label))
		return "", false
	}
	return value, true
}

// handleNotFound escribe 404 si err es db.ErrNotFound y devuelve true (el
// caller debe hacer return). Si no, devuelve false y el caller sigue con
// su propio log + 500.
func handleNotFound(w http.ResponseWriter, err error, notFoundMsg string) bool {
	if errors.Is(err, db.ErrNotFound) {
		writeError(w, http.StatusNotFound, notFoundMsg)
		return true
	}
	return false
}

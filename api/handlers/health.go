package handlers

import (
	"encoding/json"
	"net/http"
)

// Health responds with a simple JSON status payload, used for uptime checks
// and to confirm the API is reachable during local development.
func Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

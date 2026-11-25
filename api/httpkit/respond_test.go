package httpkit

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5/middleware"

	"github.com/ferjmc/god-edu/api/db"
)

// TestClientIP pasa el request por el mismo middleware que usa main.go: lo
// que importa es que un header que controla el cliente (X-Forwarded-For)
// nunca decida la IP.
func TestClientIP(t *testing.T) {
	tests := []struct {
		name    string
		headers map[string]string
		want    string
	}{
		{
			name:    "detrás de Cloudflare usa CF-Connecting-IP",
			headers: map[string]string{"CF-Connecting-IP": "203.0.113.7"},
			want:    "203.0.113.7",
		},
		{
			name: "ignora un X-Forwarded-For falsificado",
			headers: map[string]string{
				"CF-Connecting-IP": "203.0.113.7",
				"X-Forwarded-For":  "198.51.100.99",
			},
			want: "203.0.113.7",
		},
		{
			name: "sin Cloudflare cae a RemoteAddr, no a headers del cliente",
			headers: map[string]string{
				"X-Forwarded-For": "198.51.100.99",
				"X-Real-IP":       "198.51.100.99",
				"True-Client-IP":  "198.51.100.99",
			},
			want: "192.0.2.1",
		},
		{
			name:    "CF-Connecting-IP inválido cae a RemoteAddr",
			headers: map[string]string{"CF-Connecting-IP": "no-es-una-ip"},
			want:    "192.0.2.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got string
			handler := middleware.ClientIPFromHeader("CF-Connecting-IP")(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					got = ClientIP(r)
				}),
			)

			// httptest.NewRequest deja RemoteAddr en 192.0.2.1:1234.
			req := httptest.NewRequest(http.MethodPost, "/auth/login", nil)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}
			handler.ServeHTTP(httptest.NewRecorder(), req)

			if got != tt.want {
				t.Fatalf("esperaba %q, recibí %q", tt.want, got)
			}
		})
	}
}

func TestRequireNonEmpty(t *testing.T) {
	w := httptest.NewRecorder()
	if _, ok := RequireNonEmpty(w, "   ", "título"); ok {
		t.Fatal("esperaba ok=false para un valor vacío")
	}
	if w.Code != 400 {
		t.Fatalf("esperaba 400, recibí %d", w.Code)
	}

	w2 := httptest.NewRecorder()
	value, ok := RequireNonEmpty(w2, "  hola  ", "título")
	if !ok || value != "hola" {
		t.Fatalf("esperaba ok=true y value recortado, recibí ok=%v value=%q", ok, value)
	}
}

type fakePinger struct{ err error }

func (f fakePinger) Ping(context.Context) error { return f.err }

func TestHealth(t *testing.T) {
	tests := []struct {
		name       string
		pingErr    error
		wantStatus int
		wantBody   string
	}{
		{name: "base activa", pingErr: nil, wantStatus: 200, wantBody: `{"status":"ok"}`},
		{name: "base caída", pingErr: errors.New("connection refused"), wantStatus: 503, wantBody: `{"status":"unavailable"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			Health(fakePinger{err: tt.pingErr})(w, httptest.NewRequest(http.MethodGet, "/health", nil))

			if w.Code != tt.wantStatus {
				t.Fatalf("esperaba %d, recibí %d", tt.wantStatus, w.Code)
			}
			if got := strings.TrimSpace(w.Body.String()); got != tt.wantBody {
				t.Fatalf("esperaba body %s, recibí %s", tt.wantBody, got)
			}
		})
	}
}

func TestHandleNotFound(t *testing.T) {
	w := httptest.NewRecorder()
	if !HandleNotFound(w, db.ErrNotFound, "no encontrado") {
		t.Fatal("esperaba true para db.ErrNotFound")
	}
	if w.Code != 404 {
		t.Fatalf("esperaba 404, recibí %d", w.Code)
	}

	w2 := httptest.NewRecorder()
	if HandleNotFound(w2, errors.New("otro error"), "no encontrado") {
		t.Fatal("esperaba false para un error que no es db.ErrNotFound")
	}
	if w2.Code != 200 {
		t.Fatalf("esperaba que no escriba nada (200 por default), recibí %d", w2.Code)
	}
}

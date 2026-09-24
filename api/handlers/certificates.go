package handlers

import (
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	certapp "github.com/ferjmc/god-edu/api/internal/certificate/application"
	certdomain "github.com/ferjmc/god-edu/api/internal/certificate/domain"
)

// CertificateHandler expone la verificación pública de certificados: sin
// sesión, a propósito — cualquiera con el link o el QR de un certificado
// tiene que poder confirmar si es válido y a quién fue emitido, sin tener
// cuenta en la plataforma (ver newRouter en main.go, fuera de todo grupo
// con auth.RequireAuth).
type CertificateHandler struct {
	Certificates *certapp.Service
}

type certificateResponse struct {
	RecipientName string    `json:"recipientName"`
	CourseTitle   string    `json:"courseTitle"`
	IssuedAt      time.Time `json:"issuedAt"`
}

// Verify busca un certificado por su código público (el que viaja en la
// URL y en el QR). 404 genérico si no existe — no distingue "código mal
// formado" de "código que nunca existió", para no darle a un tercero
// información sobre qué códigos son válidos por descarte.
func (h *CertificateHandler) Verify(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")

	cert, err := h.Certificates.VerifyByCode(r.Context(), code)
	if err != nil {
		if errors.Is(err, certdomain.ErrNotFound) {
			writeError(w, http.StatusNotFound, "certificado no encontrado")
			return
		}
		log.Printf("certificates: verificando código %q: %v", code, err)
		writeError(w, http.StatusInternalServerError, "no se pudo verificar el certificado")
		return
	}

	writeJSON(w, http.StatusOK, certificateResponse{
		RecipientName: cert.RecipientName,
		CourseTitle:   cert.CourseTitle,
		IssuedAt:      cert.IssuedAt,
	})
}

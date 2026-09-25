package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/ferjmc/god-edu/api/auth"
	"github.com/ferjmc/god-edu/api/db"
	authtokendomain "github.com/ferjmc/god-edu/api/internal/authtoken/domain"
)

type verifyEmailRequest struct {
	Token string `json:"token"`
}

// VerifyEmail consume un token de verificación y marca la cuenta como
// verificada. Es POST (no GET) a propósito: algunos escáneres de seguridad
// corporativos "clickean" en automático los links de un email, y si fuera
// un GET consumirían el token antes de que el usuario lo abra de verdad.
// La página del frontend lee el token de la URL y hace este POST por JS.
func (h *AuthHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	var req verifyEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Token == "" {
		writeError(w, http.StatusBadRequest, "token requerido")
		return
	}

	tok, err := h.Tokens.GetValidByHash(r.Context(), auth.HashToken(req.Token), authtokendomain.TokenPurposeEmailVerification)
	if err != nil {
		writeError(w, http.StatusBadRequest, "el link de verificación es inválido o venció")
		return
	}

	if err := h.Users.MarkEmailVerified(r.Context(), tok.UserID); err != nil {
		writeError(w, http.StatusInternalServerError, "no se pudo verificar la cuenta")
		return
	}
	_ = h.Tokens.MarkUsed(r.Context(), tok.ID)

	writeJSON(w, http.StatusOK, map[string]bool{"email_verified": true})
}

type resendVerificationRequest struct {
	Email string `json:"email"`
}

// ResendVerification reenvía el link de verificación. La respuesta es
// siempre el mismo mensaje genérico exista o no la cuenta, y esté o no ya
// verificada: no queremos que este endpoint sirva para averiguar qué
// emails están registrados.
func (h *AuthHandler) ResendVerification(w http.ResponseWriter, r *http.Request) {
	const genericMessage = "si el email existe y no fue verificado, te mandamos un nuevo link"

	var req resendVerificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "cuerpo JSON inválido")
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	user, err := h.Users.GetByEmail(r.Context(), req.Email)
	if err == nil && !user.EmailVerified {
		h.sendVerificationEmail(r.Context(), user)
	} else if err != nil && !errors.Is(err, db.ErrNotFound) {
		writeError(w, http.StatusInternalServerError, "no se pudo procesar el pedido")
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]string{"message": genericMessage})
}

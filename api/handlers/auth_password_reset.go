package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/ferjmc/god-edu/api/auth"
	"github.com/ferjmc/god-edu/api/db"
	"github.com/ferjmc/god-edu/api/email"
	"github.com/ferjmc/god-edu/api/models"
)

type forgotPasswordRequest struct {
	Email string `json:"email"`
}

// ForgotPassword genera un link de reset y lo manda por email.
// Igual que ResendVerification, la respuesta es siempre el mismo mensaje
// genérico para no revelar si el email está registrado.
func (h *AuthHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	const genericMessage = "si el email existe, te mandamos instrucciones para restablecer tu contraseña"

	var req forgotPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "cuerpo JSON inválido")
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	user, err := h.Users.GetByEmail(r.Context(), req.Email)
	switch {
	case err == nil:
		h.sendPasswordResetEmail(r.Context(), user)
	case errors.Is(err, db.ErrNotFound):
		// No existe: no hacemos nada, pero respondemos igual que si existiera.
	default:
		writeError(w, http.StatusInternalServerError, "no se pudo procesar el pedido")
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]string{"message": genericMessage})
}

// sendPasswordResetEmail genera un token de reset y manda el email.
// Best-effort, igual que sendVerificationEmail: no queremos que un
// problema con Resend tire un 500 en un endpoint que ya de por sí no debe
// revelar si el email existe.
func (h *AuthHandler) sendPasswordResetEmail(ctx context.Context, user models.User) {
	raw, hash, err := auth.GenerateToken()
	if err != nil {
		log.Printf("auth: generando token de reset para user %d: %v", user.ID, err)
		return
	}

	if _, err := h.Tokens.Create(ctx, user.ID, models.TokenPurposePasswordReset, hash, auth.PasswordResetTokenTTL); err != nil {
		log.Printf("auth: guardando token de reset para user %d: %v", user.ID, err)
		return
	}

	link := h.AppBaseURL + "/reset-password?token=" + raw
	subject, html := email.PasswordResetEmail(link)
	if err := h.Email.Send(ctx, user.Email, subject, html); err != nil {
		log.Printf("auth: enviando email de reset a %s: %v", user.Email, err)
	}
}

type resetPasswordRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

// ResetPassword consume un token de reset y actualiza el password.
func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req resetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Token == "" {
		writeError(w, http.StatusBadRequest, "token requerido")
		return
	}

	tok, err := h.Tokens.GetValidByHash(r.Context(), auth.HashToken(req.Token), models.TokenPurposePasswordReset)
	if err != nil {
		writeError(w, http.StatusBadRequest, "el link de restablecimiento es inválido o venció")
		return
	}

	hash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.Users.UpdatePasswordHash(r.Context(), tok.UserID, hash); err != nil {
		writeError(w, http.StatusInternalServerError, "no se pudo actualizar la contraseña")
		return
	}
	_ = h.Tokens.MarkUsed(r.Context(), tok.ID)

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

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

// EmailSender es lo mínimo que AuthHandler necesita para mandar correo.
// *email.Sender (Resend) la cumple; en tests o en un smoke test se puede
// pasar un fake sin tocar la API real.
type EmailSender interface {
	Send(ctx context.Context, to, subject, html string) error
}

// AuthHandler agrupa los endpoints de autenticación por email/password,
// verificación de cuenta y reset de password. Se instancia una sola vez en
// main.go con sus dependencias reales.
type AuthHandler struct {
	Users  *db.UserRepo
	Tokens *db.AuthTokenRepo
	JWT    *auth.JWTManager
	Email  EmailSender
	// AppBaseURL es el origen del frontend (ej. https://cursos.tuorg.org),
	// usado para armar los links de verificación y reset que se mandan
	// por email.
	AppBaseURL string
}

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// userResponse es la representación pública de un usuario: nunca incluye
// password_hash, aunque se agreguen campos a models.User más adelante.
type userResponse struct {
	ID            int64  `json:"id"`
	Email         string `json:"email"`
	Name          string `json:"name"`
	AuthProvider  string `json:"auth_provider"`
	EmailVerified bool   `json:"email_verified"`
	Role          string `json:"role"`
}

func toUserResponse(u models.User) userResponse {
	return userResponse{
		ID:            u.ID,
		Email:         u.Email,
		Name:          u.Name,
		AuthProvider:  string(u.AuthProvider),
		EmailVerified: u.EmailVerified,
		Role:          string(u.Role),
	}
}

// Register crea una cuenta nueva con email/password y abre sesión.
//
// La verificación de email (envío del correo con el link) se conecta en un
// paso aparte, junto con la integración de Resend; por ahora la cuenta
// queda creada con email_verified = false.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "cuerpo JSON inválido")
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	req.Name = strings.TrimSpace(req.Name)

	if req.Email == "" || !strings.Contains(req.Email, "@") {
		writeError(w, http.StatusBadRequest, "email inválido")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "el nombre es requerido")
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	created, err := h.Users.Create(r.Context(), models.User{
		Email:        req.Email,
		PasswordHash: &hash,
		Name:         req.Name,
		AuthProvider: models.AuthProviderEmail,
		Role:         models.RolePublicMember,
	})
	if err != nil {
		if errors.Is(err, db.ErrConflict) {
			writeError(w, http.StatusConflict, "ese email ya está registrado")
			return
		}
		writeError(w, http.StatusInternalServerError, "no se pudo crear la cuenta")
		return
	}

	token, err := h.JWT.Issue(created.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "no se pudo iniciar sesión")
		return
	}
	auth.SetSessionCookie(w, token)

	// Best-effort: si Resend está caído no queremos que falle el registro
	// completo. El usuario puede pedir que le reenvíen el link.
	h.sendVerificationEmail(r.Context(), created)

	writeJSON(w, http.StatusCreated, toUserResponse(created))
}

// sendVerificationEmail genera un token de verificación y manda el email.
// No devuelve error a propósito (ver comentario en Register); solo loguea.
func (h *AuthHandler) sendVerificationEmail(ctx context.Context, user models.User) {
	raw, hash, err := auth.GenerateToken()
	if err != nil {
		log.Printf("auth: generando token de verificación para user %d: %v", user.ID, err)
		return
	}

	if _, err := h.Tokens.Create(ctx, user.ID, models.TokenPurposeEmailVerification, hash, auth.EmailVerificationTokenTTL); err != nil {
		log.Printf("auth: guardando token de verificación para user %d: %v", user.ID, err)
		return
	}

	link := h.AppBaseURL + "/verify-email?token=" + raw
	subject, html := email.VerificationEmail(link)
	if err := h.Email.Send(ctx, user.Email, subject, html); err != nil {
		log.Printf("auth: enviando email de verificación a %s: %v", user.Email, err)
	}
}

// Login valida email/password y abre sesión.
//
// El error es siempre el mismo mensaje genérico, exista o no la cuenta y
// sea o no una cuenta de OAuth sin password: no queremos que la respuesta
// permita a alguien enumerar qué emails están registrados.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	const genericError = "email o contraseña inválidos"

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "cuerpo JSON inválido")
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	user, err := h.Users.GetByEmail(r.Context(), req.Email)
	if err != nil {
		writeError(w, http.StatusUnauthorized, genericError)
		return
	}

	if user.PasswordHash == nil || !auth.CheckPassword(*user.PasswordHash, req.Password) {
		writeError(w, http.StatusUnauthorized, genericError)
		return
	}

	token, err := h.JWT.Issue(user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "no se pudo iniciar sesión")
		return
	}
	auth.SetSessionCookie(w, token)

	writeJSON(w, http.StatusOK, toUserResponse(user))
}

// Logout borra la cookie de sesión. No requiere estar autenticado: si no
// hay cookie, no hace nada dañino.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	auth.ClearSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

// Me devuelve el usuario autenticado actual. Se monta detrás de
// auth.RequireAuth, que es quien deja el user_id en el contexto.
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "no autenticado")
		return
	}

	user, err := h.Users.GetByID(r.Context(), userID)
	if err != nil {
		// El JWT es válido pero el usuario ya no existe (cuenta borrada).
		writeError(w, http.StatusUnauthorized, "no autenticado")
		return
	}

	writeJSON(w, http.StatusOK, toUserResponse(user))
}

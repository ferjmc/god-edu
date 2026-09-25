package handlers

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"

	"github.com/ferjmc/god-edu/api/auth"
	"github.com/ferjmc/god-edu/api/db"
	userapp "github.com/ferjmc/god-edu/api/internal/user/application"
	userdomain "github.com/ferjmc/god-edu/api/internal/user/domain"
)

// OAuthHandler agrupa el flujo de login con Google/Facebook vía goth.
type OAuthHandler struct {
	Users *userapp.Service
	JWT   *auth.JWTManager
	// SuccessRedirectURL y FailureRedirectURL son páginas del frontend
	// (home y /ingresar, ver main.go) a las que se redirige después del
	// callback, ya sea con la cookie de sesión seteada o con el login
	// fallado.
	SuccessRedirectURL string
	FailureRedirectURL string
}

// BeginAuth arranca el flujo OAuth: redirige al usuario al provider
// (Google o Facebook, según el {provider} de la ruta).
func (h *OAuthHandler) BeginAuth(w http.ResponseWriter, r *http.Request) {
	provider := chi.URLParam(r, "provider")
	r = gothic.GetContextWithProvider(r, provider)
	gothic.BeginAuthHandler(w, r)
}

// Callback recibe la vuelta del provider, crea o recupera el usuario local
// y abre sesión con la misma cookie JWT que usa el login por password.
func (h *OAuthHandler) Callback(w http.ResponseWriter, r *http.Request) {
	provider := chi.URLParam(r, "provider")
	r = gothic.GetContextWithProvider(r, provider)

	gothUser, err := gothic.CompleteUserAuth(w, r)
	if err != nil {
		log.Printf("oauth: completando auth con %s: %v", provider, err)
		http.Redirect(w, r, h.FailureRedirectURL, http.StatusTemporaryRedirect)
		return
	}

	if gothUser.Email == "" {
		log.Printf("oauth: %s no devolvió email para el usuario", provider)
		http.Redirect(w, r, h.FailureRedirectURL, http.StatusTemporaryRedirect)
		return
	}

	user, err := h.findOrCreateOAuthUser(r.Context(), provider, gothUser)
	if err != nil {
		log.Printf("oauth: creando/recuperando usuario: %v", err)
		http.Redirect(w, r, h.FailureRedirectURL, http.StatusTemporaryRedirect)
		return
	}

	token, err := h.JWT.Issue(user.ID)
	if err != nil {
		log.Printf("oauth: emitiendo JWT: %v", err)
		http.Redirect(w, r, h.FailureRedirectURL, http.StatusTemporaryRedirect)
		return
	}
	auth.SetSessionCookie(w, token)

	http.Redirect(w, r, h.SuccessRedirectURL, http.StatusTemporaryRedirect)
}

// findOrCreateOAuthUser busca un usuario existente por email; si no existe,
// crea uno nuevo con email_verified = true (el provider ya lo verificó).
// Si existe pero todavía no estaba verificado, "reclama" la cuenta — ver
// el comentario de userpg.Repository.ClaimByOAuth.
func (h *OAuthHandler) findOrCreateOAuthUser(ctx context.Context, provider string, gu goth.User) (userdomain.User, error) {
	existing, err := h.Users.GetByEmail(ctx, gu.Email)
	if err == nil {
		if !existing.EmailVerified {
			if err := h.Users.ClaimByOAuth(ctx, existing.ID, userdomain.AuthProvider(provider)); err != nil {
				return userdomain.User{}, err
			}
			existing.EmailVerified = true
			existing.PasswordHash = nil
			existing.AuthProvider = userdomain.AuthProvider(provider)
		}
		return existing, nil
	}
	if !errors.Is(err, db.ErrNotFound) {
		return userdomain.User{}, err
	}

	name := gu.Name
	if name == "" {
		name = gu.NickName
	}
	if name == "" {
		name = gu.Email
	}

	return h.Users.Create(ctx, userdomain.User{
		Email:         gu.Email,
		Name:          name,
		AuthProvider:  userdomain.AuthProvider(provider),
		EmailVerified: true,
		Role:          userdomain.RolePublicMember,
	})
}

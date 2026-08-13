package auth

import (
	"log"
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/facebook"
	"github.com/markbates/goth/providers/google"
)

// OAuthConfig son las credenciales y URLs necesarias para armar los
// providers de OAuth de goth.
type OAuthConfig struct {
	// SessionSecret firma la cookie temporal que gothic usa para guardar
	// el "state" del flujo OAuth entre el begin y el callback. No tiene
	// nada que ver con la sesión de la app (esa es el JWT).
	SessionSecret string

	GoogleClientID     string
	GoogleClientSecret string

	FacebookClientID     string
	FacebookClientSecret string

	// APIBaseURL es la URL pública de este API (ej. https://api.tuorg.org
	// en prod, http://localhost:8080 en dev). Los providers redirigen acá
	// después del login.
	APIBaseURL string
}

// SetupOAuth registra los providers de Google y Facebook en goth (los que
// tengan credenciales configuradas) y configura el session store que
// gothic necesita para el intercambio de estado del flujo OAuth.
//
// Un provider sin credenciales simplemente no se registra: el servidor
// arranca igual, y ese provider puntual no funciona hasta que se le pongan
// las env vars correspondientes. Esto evita que un solo dev necesite tener
// las apps de Google Y Facebook creadas de entrada para poder levantar el
// resto de la API en desarrollo.
func SetupOAuth(cfg OAuthConfig) {
	store := sessions.NewCookieStore([]byte(cfg.SessionSecret))
	store.MaxAge(300) // el flujo OAuth dura segundos; no es la sesión de la app
	store.Options.Path = "/"
	store.Options.HttpOnly = true
	store.Options.Secure = true
	store.Options.SameSite = http.SameSiteLaxMode
	gothic.Store = store

	var providers []goth.Provider

	if cfg.GoogleClientID != "" && cfg.GoogleClientSecret != "" {
		providers = append(providers, google.New(cfg.GoogleClientID, cfg.GoogleClientSecret, cfg.APIBaseURL+"/auth/google/callback", "email", "profile"))
	} else {
		log.Println("auth: Google OAuth no configurado (faltan GOOGLE_CLIENT_ID/GOOGLE_CLIENT_SECRET), /auth/google no va a funcionar")
	}

	if cfg.FacebookClientID != "" && cfg.FacebookClientSecret != "" {
		providers = append(providers, facebook.New(cfg.FacebookClientID, cfg.FacebookClientSecret, cfg.APIBaseURL+"/auth/facebook/callback", "email"))
	} else {
		log.Println("auth: Facebook OAuth no configurado (faltan FACEBOOK_CLIENT_ID/FACEBOOK_CLIENT_SECRET), /auth/facebook no va a funcionar")
	}

	if len(providers) > 0 {
		goth.UseProviders(providers...)
	}
}

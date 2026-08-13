package main

import (
	"log"
	"os"
)

// config son todas las variables de entorno que necesita el API. Se leen
// una sola vez al arrancar (ver .env.example para la lista completa).
type config struct {
	Port        string
	DatabaseURL string
	JWTSecret   string
	// MigrationsPath es dónde busca las migraciones al arrancar (ver
	// db.RunMigrations). "migrations" a secas asume WORKDIR /app con la
	// carpeta migrations copiada ahí al lado del binario — así es como la
	// arma api/Dockerfile. Si no existe, RunMigrations lo loguea y sigue
	// sin migrar (cubre correr el binario suelto fuera de Docker).
	MigrationsPath string

	// AppBaseURL es el origen del frontend (Astro). Se usa para armar los
	// links de los emails y las redirecciones post-OAuth.
	AppBaseURL string
	// APIBaseURL es la URL pública de este API. goth arma los callbacks de
	// OAuth sobre esto.
	APIBaseURL string

	ResendAPIKey string
	EmailFrom    string

	OAuthSessionSecret   string
	GoogleClientID       string
	GoogleClientSecret   string
	FacebookClientID     string
	FacebookClientSecret string

	// Credenciales de Cloudflare R2 para subir PDFs desde el panel admin.
	// Opcionales: sin ellas, el server arranca igual (útil para desarrollar
	// sin cuenta de R2), pero el endpoint de subida de PDF no funciona —
	// ver main.go.
	R2AccountID       string
	R2AccessKeyID     string
	R2SecretAccessKey string
	R2Bucket          string
	R2PublicBaseURL   string
}

// loadConfig lee y valida las env vars. Corta el arranque (log.Fatalf) si
// falta algo sin lo cual el API no puede funcionar en absoluto; las
// integraciones opcionales (Resend, cada provider OAuth) se resuelven más
// abajo, en main(), con su propio fallback si faltan.
func loadConfig() config {
	return config{
		Port:           getEnv("PORT", "8080"),
		DatabaseURL:    requireEnv("DATABASE_URL"),
		JWTSecret:      requireEnv("JWT_SECRET"),
		MigrationsPath: getEnv("MIGRATIONS_PATH", "migrations"),

		AppBaseURL: requireEnv("APP_BASE_URL"),
		APIBaseURL: requireEnv("API_BASE_URL"),

		ResendAPIKey: os.Getenv("RESEND_API_KEY"),
		EmailFrom:    os.Getenv("EMAIL_FROM"),

		OAuthSessionSecret:   requireEnv("OAUTH_SESSION_SECRET"),
		GoogleClientID:       os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret:   os.Getenv("GOOGLE_CLIENT_SECRET"),
		FacebookClientID:     os.Getenv("FACEBOOK_CLIENT_ID"),
		FacebookClientSecret: os.Getenv("FACEBOOK_CLIENT_SECRET"),

		R2AccountID:       os.Getenv("R2_ACCOUNT_ID"),
		R2AccessKeyID:     os.Getenv("R2_ACCESS_KEY_ID"),
		R2SecretAccessKey: os.Getenv("R2_SECRET_ACCESS_KEY"),
		R2Bucket:          os.Getenv("R2_BUCKET"),
		R2PublicBaseURL:   os.Getenv("R2_PUBLIC_BASE_URL"),
	}
}

func requireEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("main: falta la variable de entorno %s (ver .env.example)", key)
	}
	return v
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

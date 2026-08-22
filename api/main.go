package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"

	"github.com/ferjmc/god-edu/api/auth"
	"github.com/ferjmc/god-edu/api/db"
	"github.com/ferjmc/god-edu/api/email"
	"github.com/ferjmc/god-edu/api/handlers"
	"github.com/ferjmc/god-edu/api/storage"
)

func main() {
	cfg := loadConfig()

	ctx := context.Background()
	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("main: conectando a la base: %v", err)
	}
	defer pool.Close()

	// Migrar antes de servir tráfico: así cada deploy (Coolify, Dokploy, o
	// un docker run suelto) queda con el schema al día sin un paso manual
	// aparte. Si falla, mejor no arrancar a que la API sirva contra un
	// schema que no es el que el código espera.
	if err := db.RunMigrations(cfg.DatabaseURL, cfg.MigrationsPath); err != nil {
		log.Fatalf("main: aplicando migraciones: %v", err)
	}

	users := db.NewUserRepo(pool)
	tokens := db.NewAuthTokenRepo(pool)
	courses := db.NewCourseRepo(pool)
	lessons := db.NewLessonRepo(pool)
	jwtManager := auth.NewJWTManager(cfg.JWTSecret)

	var sender handlers.EmailSender
	if cfg.ResendAPIKey != "" {
		sender = email.NewSender(cfg.ResendAPIKey, cfg.EmailFrom)
	} else {
		log.Println("main: RESEND_API_KEY no configurada, los emails solo se van a loguear")
		sender = email.NoopSender{}
	}

	var r2 *storage.R2
	if cfg.R2AccountID != "" && cfg.R2AccessKeyID != "" && cfg.R2SecretAccessKey != "" && cfg.R2Bucket != "" {
		r2, err = storage.NewR2(storage.R2Config{
			AccountID:       cfg.R2AccountID,
			AccessKeyID:     cfg.R2AccessKeyID,
			SecretAccessKey: cfg.R2SecretAccessKey,
			Bucket:          cfg.R2Bucket,
			PublicBaseURL:   cfg.R2PublicBaseURL,
		})
		if err != nil {
			log.Fatalf("main: configurando R2: %v", err)
		}
	} else {
		log.Println("main: credenciales de R2 no configuradas, la subida de PDFs no va a funcionar")
	}

	auth.SetupOAuth(auth.OAuthConfig{
		SessionSecret:        cfg.OAuthSessionSecret,
		GoogleClientID:       cfg.GoogleClientID,
		GoogleClientSecret:   cfg.GoogleClientSecret,
		FacebookClientID:     cfg.FacebookClientID,
		FacebookClientSecret: cfg.FacebookClientSecret,
		APIBaseURL:           cfg.APIBaseURL,
	})

	authHandler := &handlers.AuthHandler{
		Users:      users,
		Tokens:     tokens,
		JWT:        jwtManager,
		Email:      sender,
		AppBaseURL: cfg.AppBaseURL,
	}
	oauthHandler := &handlers.OAuthHandler{
		Users:              users,
		JWT:                jwtManager,
		SuccessRedirectURL: cfg.AppBaseURL + "/",
		FailureRedirectURL: cfg.AppBaseURL + "/ingresar?error=oauth",
	}
	courseHandler := &handlers.CourseHandler{Courses: courses, Users: users}
	lessonHandler := &handlers.LessonHandler{Courses: courses, Users: users, Lessons: lessons}
	adminHandler := &handlers.AdminHandler{Users: users, Courses: courses, Lessons: lessons, R2: r2}

	r := newRouter(cfg, authHandler, oauthHandler, courseHandler, lessonHandler, adminHandler, users, jwtManager)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("api escuchando en :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("main: %v", err)
		}
	}()

	waitForShutdown(srv)
}

// newRouter arma todas las rutas de la API. Las que requieren sesión
// (auth.RequireAuth) quedan agrupadas aparte para que quede a la vista
// cuáles son públicas y cuáles no.
func newRouter(cfg config, authHandler *handlers.AuthHandler, oauthHandler *handlers.OAuthHandler, courseHandler *handlers.CourseHandler, lessonHandler *handlers.LessonHandler, adminHandler *handlers.AdminHandler, users *db.UserRepo, jwtManager *auth.JWTManager) http.Handler {
	r := chi.NewRouter()
	// RealIP primero: sin esto, detrás del proxy de Coolify/Caddy todos los
	// requests llegan con el mismo r.RemoteAddr (el del proxy), y el rate
	// limiting de más abajo terminaría limitando a todos los usuarios como
	// si fueran uno solo. Lee X-Forwarded-For/X-Real-IP y confía en ellos
	// sin validar — asumible acá porque la API nunca queda expuesta
	// directamente a internet, solo detrás del proxy (si eso cambia
	// alguna vez, revisar esto).
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{cfg.AppBaseURL},
		AllowedMethods:   []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete},
		AllowedHeaders:   []string{"Content-Type"},
		AllowCredentials: true, // necesario para que el navegador mande la cookie de sesión
		MaxAge:           300,
	}))

	r.Get("/health", handlers.Health)

	r.Route("/courses", func(r chi.Router) {
		r.Get("/", courseHandler.List)

		r.Group(func(r chi.Router) {
			r.Use(auth.RequireAuth(jwtManager))
			r.Get("/{slug}", courseHandler.Detail)
			r.Get("/{slug}/lessons", lessonHandler.List)
			r.Get("/{slug}/lessons/{order}", lessonHandler.Detail)
			r.Post("/{slug}/lessons/{order}/complete", lessonHandler.MarkComplete)
		})
	})

	// /admin: carga de contenido (cursos, lecciones, video/markdown/PDF).
	// Detrás de sesión + rol ADMIN — ver handlers.RequireAdmin. No hay
	// bootstrap automático del primer admin: se promueve a mano con un
	// UPDATE contra la base (ver README de despliegue).
	r.Route("/admin/courses", func(r chi.Router) {
		r.Use(auth.RequireAuth(jwtManager))
		r.Use(handlers.RequireAdmin(users))

		r.Get("/", adminHandler.ListCourses)
		r.Get("/{slug}", adminHandler.CourseDetail)
		r.Post("/", adminHandler.CreateCourse)
		r.Put("/{slug}", adminHandler.UpdateCourseDetails)
		r.Patch("/{slug}", adminHandler.SetPublished)
		r.Delete("/{slug}", adminHandler.DeleteCourse)
		r.Post("/{slug}/lessons", adminHandler.CreateLesson)
		r.Put("/{slug}/lessons/{order}", adminHandler.UpdateLesson)
		r.Delete("/{slug}/lessons/{order}", adminHandler.DeleteLesson)
		r.Post("/{slug}/lessons/{order}/content", adminHandler.CreateContent)
		r.Post("/{slug}/lessons/{order}/content/pdf", adminHandler.UploadPDFContent)
		r.Put("/{slug}/lessons/{order}/content/{contentOrder}", adminHandler.UpdateContent)
		r.Delete("/{slug}/lessons/{order}/content/{contentOrder}", adminHandler.DeleteContent)
	})

	// /admin: consulta de usuarios.
	// Detrás de sesión + rol ADMIN — ver handlers.RequireAdmin. 
	r.Route("/admin/users", func(r chi.Router) {
		r.Use(auth.RequireAuth(jwtManager))
		r.Use(handlers.RequireAdmin(users))

		r.Get("/", adminHandler.ListUsers)
	})

	r.Route("/auth", func(r chi.Router) {
		// Límites por IP en los endpoints sensibles a fuerza bruta o abuso:
		// login (adivinar contraseñas), y los tres que disparan un email
		// (registro, reenvío de verificación, olvidé mi contraseña) — esos
		// últimos protegen la cuota gratis de Resend tanto como al usuario
		// destinatario de no recibir spam. verify-email y reset-password no
		// llevan límite propio: consumen un token de un solo uso de 256
		// bits, imposible de adivinar por fuerza bruta.
		r.With(rateLimit(10, time.Minute)).Post("/login", authHandler.Login)
		r.With(rateLimit(5, time.Hour)).Post("/register", authHandler.Register)
		r.With(rateLimit(5, time.Hour)).Post("/resend-verification", authHandler.ResendVerification)
		r.With(rateLimit(5, time.Hour)).Post("/forgot-password", authHandler.ForgotPassword)

		r.Post("/logout", authHandler.Logout)
		r.Post("/verify-email", authHandler.VerifyEmail)
		r.Post("/reset-password", authHandler.ResetPassword)

		r.Get("/{provider}", oauthHandler.BeginAuth)
		r.Get("/{provider}/callback", oauthHandler.Callback)

		r.Group(func(r chi.Router) {
			r.Use(auth.RequireAuth(jwtManager))
			r.Get("/me", authHandler.Me)
		})
	})

	return r
}

// rateLimit arma un limitador por IP, independiente para cada ruta que lo
// use — cada llamada a rateLimit crea su propio contador interno, así que
// gastar el límite de /auth/login no consume el de /auth/register. En
// memoria, sin Redis: para el tamaño de esta plataforma (~500 usuarios, un
// solo proceso de API) alcanza, y no suma otra pieza de infraestructura
// para mantener.
func rateLimit(requests int, window time.Duration) func(http.Handler) http.Handler {
	return httprate.Limit(
		requests,
		window,
		httprate.WithKeyFuncs(httprate.KeyByIP),
		httprate.WithLimitHandler(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(map[string]string{"error": "demasiados intentos, esperá un momento y volvé a intentar"})
		}),
	)
}

// waitForShutdown bloquea hasta recibir SIGINT/SIGTERM y apaga el server
// con gracia (deja terminar los requests en vuelo, no los corta de golpe).
// Importa en un deploy con Coolify/Dokploy: al hacer git-push, el
// contenedor viejo recibe SIGTERM antes de que lo maten.
func waitForShutdown(srv *http.Server) {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Println("main: apagando...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("main: apagado forzado: %v", err)
	}
}

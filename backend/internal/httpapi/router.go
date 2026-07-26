package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"theresa/backend/internal/auth"
	"theresa/backend/internal/config"
	"theresa/backend/internal/email"
	"theresa/backend/internal/gemini"
)

// noisyHealthCheckPaths are hit every few seconds by Render's own internal
// prober and external uptime monitors (both configured to keep the free-tier
// service from sleeping) - real traffic, but not worth a log line every time.
var noisyHealthCheckPaths = map[string]bool{
	"/healthz":     true,
	"/":            true,
	"/favicon.ico": true,
}

// quietLogger wraps chi's request logger so the constant health-check
// polling above doesn't drown out real application request logs.
func quietLogger(next http.Handler) http.Handler {
	logged := middleware.Logger(next)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if noisyHealthCheckPaths[r.URL.Path] {
			next.ServeHTTP(w, r)
			return
		}
		logged.ServeHTTP(w, r)
	})
}

// NewRouter assembles the chi router with the global middleware stack and
// registers all HTTP routes.
func NewRouter(db *mongo.Database, cfg config.Config, emailClient *email.Client, geminiClient *gemini.Client) http.Handler {
	r := chi.NewRouter()

	r.Use(quietLogger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{cfg.FrontendURL},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	r.Get("/healthz", HealthHandler(db))

	authHandler := NewAuthHandler(db, cfg, emailClient)
	r.Route("/api/auth", func(r chi.Router) {
		r.Post("/signup", authHandler.Signup)
		r.Post("/verify-email", authHandler.VerifyEmail)
		r.Post("/forgot-password", authHandler.ForgotPassword)
		r.Post("/reset-password", authHandler.ResetPassword)
		r.Post("/login", authHandler.Login)
		r.Post("/logout", authHandler.Logout)

		r.Group(func(r chi.Router) {
			r.Use(auth.RequireAuth(cfg.JWTCookieName, cfg.JWTSecret))
			r.Get("/me", authHandler.Me)
		})
	})

	docHandler := NewDocumentHandler(db, cfg, geminiClient)
	r.Route("/api/documents", func(r chi.Router) {
		r.Use(auth.RequireAuth(cfg.JWTCookieName, cfg.JWTSecret))
		r.Post("/", docHandler.Upload)
		r.Get("/", docHandler.List)
		r.Get("/{id}", docHandler.Get)
		r.Get("/{id}/file", docHandler.Download)
	})

	sessionHandler := NewSessionHandler(db, cfg, geminiClient)
	r.Route("/api/sessions", func(r chi.Router) {
		r.Use(auth.RequireAuth(cfg.JWTCookieName, cfg.JWTSecret))
		r.Post("/", sessionHandler.Create)
		r.Get("/", sessionHandler.List)
		r.Get("/{id}", sessionHandler.Get)
		r.Patch("/{id}", sessionHandler.UpdateMode)
		r.Delete("/{id}", sessionHandler.Delete)
		r.Post("/{id}/messages", sessionHandler.PostMessage)
	})

	liveHandler := NewLiveHandler(db, cfg, geminiClient, emailClient)
	r.Group(func(r chi.Router) {
		r.Use(auth.RequireAuth(cfg.JWTCookieName, cfg.JWTSecret))
		r.Get("/ws/session/{id}", liveHandler.HandleConnection)
	})

	paymentHandler := NewPaymentHandler(db, cfg)
	r.Route("/api/payments", func(r chi.Router) {
		// The webhook is Flutterwave calling us directly - it carries no
		// session cookie, and is authenticated by its own signature header
		// instead (checked inside Webhook), not the cookie-based middleware.
		r.Post("/webhook", paymentHandler.Webhook)

		r.Group(func(r chi.Router) {
			r.Use(auth.RequireAuth(cfg.JWTCookieName, cfg.JWTSecret))
			r.Post("/initiate", paymentHandler.Initiate)
			r.Get("/balance", paymentHandler.Balance)
		})
	})

	return r
}

package httpapi

import (
	"log"
	"net/http"
	"time"

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

// quietLogger logs every non-noisy request: method, path, remote address,
// status, response size, duration. Deliberately logs r.URL.Path rather than
// the full request URI (chi's stock middleware.Logger logs the latter) -
// some query params, like the voice WebSocket's short-lived ?ticket=, are
// bearer credentials that have no business sitting in a log file even for
// their short lifetime. No route today has a legitimate reason to want its
// query string logged, so this is a blanket policy, not a per-route carve-out.
func quietLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if noisyHealthCheckPaths[r.URL.Path] {
			next.ServeHTTP(w, r)
			return
		}
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		start := time.Now()
		next.ServeHTTP(ww, r)
		log.Printf("%q from %s - %d %dB in %s",
			r.Method+" "+r.URL.Path+" "+r.Proto, r.RemoteAddr, ww.Status(), ww.BytesWritten(), time.Since(start))
	})
}

// securityHeaders sets baseline defense-in-depth headers on every response.
// This is a JSON/WebSocket API, not an HTML-rendering surface, so there's no
// inline-script/style CSP tradeoff to worry about here the way there would
// be on the frontend - these are all safe, zero-functional-risk additions.
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		next.ServeHTTP(w, r)
	})
}

// requireCSRFHeader defends state-changing, cookie-authenticated endpoints
// against CSRF. The session cookie is SameSite=None (required so the split
// frontend/backend domains can share it at all), which means it's sent on
// cross-site requests too - a plain HTML form on an attacker's page can
// auto-submit a POST that carries it. That same form can't set a custom
// header, though, so requiring one here rejects a forged cross-site
// submission before any handler logic runs, without affecting our own
// frontend (which always sends this header - see lib/api.ts). Read-only
// methods are exempt since there's nothing to forge.
func requireCSRFHeader(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			next.ServeHTTP(w, r)
			return
		}
		if r.Header.Get("X-Requested-With") == "" {
			writeError(w, http.StatusForbidden, "forbidden")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// NewRouter assembles the chi router with the global middleware stack and
// registers all HTTP routes.
func NewRouter(db *mongo.Database, cfg config.Config, emailClient *email.Client, geminiClient *gemini.Client) http.Handler {
	r := chi.NewRouter()

	sessionCookieCfg := auth.SessionCookieConfig{
		Name:        cfg.JWTCookieName,
		TTL:         cfg.JWTTTL,
		Environment: cfg.Environment,
	}

	r.Use(quietLogger)
	r.Use(middleware.Recoverer)
	r.Use(securityHeaders)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{cfg.FrontendURL},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type", "Authorization", "X-Requested-With"},
		AllowCredentials: true,
	}))

	r.Get("/healthz", HealthHandler(db))

	authHandler := NewAuthHandler(db, cfg, emailClient)
	r.Route("/api/auth", func(r chi.Router) {
		r.Post("/signup", authHandler.Signup)
		r.Post("/verify-email", authHandler.VerifyEmail)
		r.Post("/resend-verification", authHandler.ResendVerification)
		r.Post("/forgot-password", authHandler.ForgotPassword)
		r.Post("/reset-password", authHandler.ResetPassword)
		r.Post("/login", authHandler.Login)
		r.Post("/logout", authHandler.Logout)

		r.Group(func(r chi.Router) {
			r.Use(auth.RequireAuth(db, cfg.JWTSecret, sessionCookieCfg))
			r.Get("/me", authHandler.Me)
		})
	})

	docHandler := NewDocumentHandler(db, cfg, geminiClient)
	r.Route("/api/documents", func(r chi.Router) {
		r.Use(auth.RequireAuth(db, cfg.JWTSecret, sessionCookieCfg))
		r.Use(requireCSRFHeader)
		r.Post("/", docHandler.Upload)
		r.Get("/", docHandler.List)
		r.Get("/{id}", docHandler.Get)
		r.Get("/{id}/file", docHandler.Download)
	})

	sessionHandler := NewSessionHandler(db, cfg, geminiClient)
	r.Route("/api/sessions", func(r chi.Router) {
		r.Use(auth.RequireAuth(db, cfg.JWTSecret, sessionCookieCfg))
		r.Use(requireCSRFHeader)
		r.Post("/", sessionHandler.Create)
		r.Get("/", sessionHandler.List)
		r.Get("/{id}", sessionHandler.Get)
		r.Patch("/{id}", sessionHandler.UpdateMode)
		r.Delete("/{id}", sessionHandler.Delete)
		r.Post("/{id}/messages", sessionHandler.PostMessage)
		r.Post("/{id}/ws-ticket", sessionHandler.IssueWSTicket)
	})

	liveHandler := NewLiveHandler(db, cfg, geminiClient, emailClient)
	r.Group(func(r chi.Router) {
		// This connection goes directly to the backend's own origin, unlike
		// every other route which the frontend now reaches through its own
		// proxy - see auth.RequireAuthCookieOrTicket for why the cookie alone
		// isn't reliable here.
		r.Use(auth.RequireAuthCookieOrTicket(db, cfg.JWTSecret, sessionCookieCfg))
		r.Get("/ws/session/{id}", liveHandler.HandleConnection)
	})

	paymentHandler := NewPaymentHandler(db, cfg)
	r.Route("/api/payments", func(r chi.Router) {
		// The webhook is Flutterwave calling us directly - it carries no
		// session cookie, and is authenticated by its own signature header
		// instead (checked inside Webhook), not the cookie-based middleware.
		r.Post("/webhook", paymentHandler.Webhook)

		r.Group(func(r chi.Router) {
			r.Use(auth.RequireAuth(db, cfg.JWTSecret, sessionCookieCfg))
			r.Use(requireCSRFHeader)
			r.Post("/initiate", paymentHandler.Initiate)
			r.Get("/balance", paymentHandler.Balance)
			r.Get("/transactions", paymentHandler.Transactions)
		})
	})

	return r
}

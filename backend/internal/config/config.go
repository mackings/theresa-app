package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Port        string
	MongoURI    string
	MongoDBName string
	Environment string

	JWTSecret     string
	JWTCookieName string
	JWTTTL        time.Duration

	ResendAPIKey    string
	ResendFromEmail string
	FrontendURL     string

	GeminiAPIKey       string
	GeminiTextModel    string
	GeminiLiveModel    string
	MaxUploadSizeBytes int64

	// FlutterwaveLiveMode picks which of the two credential sets below
	// (FLUTTERWAVE_LIVE_* or FLUTTERWAVE_TEST_*) populates the four fields
	// every other package actually reads - PaymentHandler and the payments
	// package are unaware this switch exists, they just read
	// cfg.FlutterwaveSecretKey etc. as always. Defaults false (test mode) so
	// a fresh/misconfigured environment can never accidentally move real
	// money - going live requires explicitly setting this to true.
	FlutterwaveLiveMode         bool
	FlutterwavePublicKey        string
	FlutterwaveSecretKey        string
	FlutterwaveEncryptionKey    string
	FlutterwaveWebhookSecretKey string
	USDToNGNRate                float64
}

func Load() Config {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, relying on process environment")
	}

	cfg := Config{
		Port:        getEnv("PORT", "8080"),
		MongoURI:    os.Getenv("MONGODB_URI"),
		MongoDBName: getEnv("MONGODB_DB_NAME", "theresa"),
		Environment: getEnv("ENVIRONMENT", "development"),

		JWTSecret:     os.Getenv("JWT_SECRET"),
		JWTCookieName: getEnv("JWT_COOKIE_NAME", "theresa_session"),
		// 60 days, refreshed on activity (see auth.SessionCookieConfig /
		// RequireAuth's sliding-expiration check) - an active user's session
		// effectively never expires, since every authenticated request past
		// the halfway point reissues a fresh 60-day cookie. Only a user who
		// never returns for a full 60 days ever needs to log in again.
		JWTTTL: 60 * 24 * time.Hour,

		ResendAPIKey:    os.Getenv("RESEND_API_KEY"),
		ResendFromEmail: os.Getenv("RESEND_FROM_EMAIL"),
		FrontendURL:     getEnv("FRONTEND_URL", "http://localhost:3000"),

		GeminiAPIKey:       os.Getenv("GEMINI_API_KEY"),
		GeminiTextModel:    getEnv("GEMINI_TEXT_MODEL", "gemini-2.5-flash"),
		GeminiLiveModel:    getEnv("GEMINI_LIVE_MODEL", "gemini-3.1-flash-live-preview"),
		MaxUploadSizeBytes: getEnvInt64("MAX_UPLOAD_SIZE_BYTES", 20*1024*1024),

		USDToNGNRate: getEnvFloat64("USD_TO_NGN_RATE", 1380.0),
	}

	// Not fail-fast, same as GeminiLiveModel above: the rest of the app
	// should still boot and serve text/voice teaching even if billing isn't
	// configured yet. Payment endpoints check for these at call-time instead
	// and fail clearly there if missing.
	cfg.FlutterwaveLiveMode = getEnvBool("FLUTTERWAVE_LIVE_MODE", false)
	if cfg.FlutterwaveLiveMode {
		cfg.FlutterwavePublicKey = os.Getenv("FLUTTERWAVE_LIVE_PUBLIC_KEY")
		cfg.FlutterwaveSecretKey = os.Getenv("FLUTTERWAVE_LIVE_SECRET_KEY")
		cfg.FlutterwaveEncryptionKey = os.Getenv("FLUTTERWAVE_LIVE_ENCRYPTION_KEY")
		cfg.FlutterwaveWebhookSecretKey = os.Getenv("FLUTTERWAVE_LIVE_WEBHOOK_SECRET_HASH")
	} else {
		cfg.FlutterwavePublicKey = os.Getenv("FLUTTERWAVE_TEST_PUBLIC_KEY")
		cfg.FlutterwaveSecretKey = os.Getenv("FLUTTERWAVE_TEST_SECRET_KEY")
		cfg.FlutterwaveEncryptionKey = os.Getenv("FLUTTERWAVE_TEST_ENCRYPTION_KEY")
		cfg.FlutterwaveWebhookSecretKey = os.Getenv("FLUTTERWAVE_TEST_WEBHOOK_SECRET_HASH")
	}
	log.Printf("flutterwave: live mode = %v", cfg.FlutterwaveLiveMode)

	var missing []string
	if cfg.MongoURI == "" {
		missing = append(missing, "MONGODB_URI")
	}
	if cfg.JWTSecret == "" {
		missing = append(missing, "JWT_SECRET")
	}
	if cfg.ResendAPIKey == "" {
		missing = append(missing, "RESEND_API_KEY")
	}
	if cfg.ResendFromEmail == "" {
		missing = append(missing, "RESEND_FROM_EMAIL")
	}
	if cfg.GeminiAPIKey == "" {
		missing = append(missing, "GEMINI_API_KEY")
	}
	if len(missing) > 0 {
		log.Fatal(fmt.Errorf("missing required env vars: %v", missing))
	}

	// JWT_SECRET being merely non-empty isn't enough - HS256 signatures are
	// only as strong as this key, so a short/guessable value (e.g. a
	// placeholder like "changeme" left in by mistake) would let anyone who
	// discovers it forge valid session tokens. 32 bytes matches this
	// project's own convention for generated secrets (see the webhook
	// secret hash, generated via `openssl rand -hex 24` = 48 hex chars).
	if len(cfg.JWTSecret) < 32 {
		log.Fatal("JWT_SECRET must be at least 32 characters - generate one with: openssl rand -hex 32")
	}

	return cfg
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt64(key string, fallback int64) int64 {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return fallback
	}
	return n
}

func getEnvBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}

func getEnvFloat64(key string, fallback float64) float64 {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return fallback
	}
	return n
}

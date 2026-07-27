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
		JWTTTL:        7 * 24 * time.Hour,

		ResendAPIKey:    os.Getenv("RESEND_API_KEY"),
		ResendFromEmail: os.Getenv("RESEND_FROM_EMAIL"),
		FrontendURL:     getEnv("FRONTEND_URL", "http://localhost:3000"),

		GeminiAPIKey:       os.Getenv("GEMINI_API_KEY"),
		GeminiTextModel:    getEnv("GEMINI_TEXT_MODEL", "gemini-2.5-flash"),
		GeminiLiveModel:    getEnv("GEMINI_LIVE_MODEL", "gemini-3.1-flash-live-preview"),
		MaxUploadSizeBytes: getEnvInt64("MAX_UPLOAD_SIZE_BYTES", 20*1024*1024),

		// Not fail-fast, same as GeminiLiveModel above: the rest of the app
		// should still boot and serve text/voice teaching even if billing
		// isn't configured yet. Payment endpoints check for these at
		// call-time instead and fail clearly there if missing.
		FlutterwavePublicKey:        os.Getenv("FLUTTERWAVE_PUBLIC_KEY"),
		FlutterwaveSecretKey:        os.Getenv("FLUTTERWAVE_SECRET_KEY"),
		FlutterwaveEncryptionKey:    os.Getenv("FLUTTERWAVE_ENCRYPTION_KEY"),
		FlutterwaveWebhookSecretKey: os.Getenv("FLUTTERWAVE_WEBHOOK_SECRET_HASH"),
		USDToNGNRate:                getEnvFloat64("USD_TO_NGN_RATE", 1380.0),
	}

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

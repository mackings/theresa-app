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

package main

import (
	"context"
	"log"
	"net/http"

	"theresa/backend/internal/config"
	"theresa/backend/internal/db"
	"theresa/backend/internal/email"
	"theresa/backend/internal/gemini"
	"theresa/backend/internal/httpapi"
)

func main() {
	cfg := config.Load()

	ctx := context.Background()
	client, database, err := db.Connect(ctx, cfg.MongoURI, cfg.MongoDBName)
	if err != nil {
		log.Fatalf("failed to connect to mongo: %v", err)
	}
	defer client.Disconnect(ctx)

	if err := db.EnsureIndexes(ctx, database); err != nil {
		log.Fatalf("failed to ensure indexes: %v", err)
	}

	emailClient := email.NewClient(cfg.ResendAPIKey, cfg.ResendFromEmail)

	geminiClient, err := gemini.NewClient(ctx, cfg.GeminiAPIKey, cfg.GeminiTextModel)
	if err != nil {
		log.Fatalf("failed to create gemini client: %v", err)
	}

	router := httpapi.NewRouter(database, cfg, emailClient, geminiClient)

	log.Printf("theresa backend listening on :%s (env=%s)", cfg.Port, cfg.Environment)
	if err := http.ListenAndServe(":"+cfg.Port, router); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

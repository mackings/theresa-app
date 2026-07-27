package httpapi

import (
	"encoding/json"
	"log"
	"net/http"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

type healthResponse struct {
	Status string `json:"status"`
	Mongo  string `json:"mongo"`
}

// HealthHandler pings MongoDB on every call so /healthz reflects live
// connectivity rather than just process liveness. This endpoint is
// unauthenticated (uptime monitors need to hit it without credentials), so
// the real error detail (which could include connection-string/DNS
// internals) is logged server-side only, never returned in the response.
func HealthHandler(db *mongo.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if err := db.Client().Ping(r.Context(), readpref.Primary()); err != nil {
			log.Printf("healthz: mongo ping failed: %v", err)
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(healthResponse{
				Status: "error",
				Mongo:  "unreachable",
			})
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(healthResponse{Status: "ok", Mongo: "ok"})
	}
}

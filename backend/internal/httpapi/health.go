package httpapi

import (
	"encoding/json"
	"net/http"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

type healthResponse struct {
	Status string `json:"status"`
	Mongo  string `json:"mongo"`
	Error  string `json:"error,omitempty"`
}

// HealthHandler pings MongoDB on every call so /healthz reflects live
// connectivity rather than just process liveness.
func HealthHandler(db *mongo.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if err := db.Client().Ping(r.Context(), readpref.Primary()); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(healthResponse{
				Status: "error",
				Mongo:  "unreachable",
				Error:  err.Error(),
			})
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(healthResponse{Status: "ok", Mongo: "ok"})
	}
}

package auth

import (
	"context"
	"encoding/json"
	"net/http"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type contextKey string

const userIDContextKey contextKey = "user_id"

// RequireAuth validates the session cookie's signature/expiry, then checks
// its embedded TokenVersion against the user's current value in the
// database - a mismatch means the token was issued before a logout or
// password reset revoked it, so it's rejected even though it's still
// cryptographically valid and unexpired. This is the one authenticated-path
// database round trip added for revocation support; it's a single indexed
// lookup by _id with a minimal projection, consistent with the per-request
// Mongo lookups already done elsewhere in this app (loadOwnedSession, Me).
func RequireAuth(db *mongo.Database, cookieName, jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(cookieName)
			if err != nil {
				unauthorized(w)
				return
			}

			claims, err := ParseToken(cookie.Value, jwtSecret)
			if err != nil {
				unauthorized(w)
				return
			}

			userID, err := bson.ObjectIDFromHex(claims.UserID)
			if err != nil {
				unauthorized(w)
				return
			}

			var current struct {
				TokenVersion int `bson:"token_version"`
			}
			opts := options.FindOne().SetProjection(bson.M{"token_version": 1})
			if err := db.Collection("users").FindOne(r.Context(), bson.M{"_id": userID}, opts).Decode(&current); err != nil {
				unauthorized(w)
				return
			}
			if current.TokenVersion != claims.TokenVersion {
				unauthorized(w)
				return
			}

			ctx := context.WithValue(r.Context(), userIDContextKey, claims.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func UserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(userIDContextKey).(string)
	return userID, ok
}

func unauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
}

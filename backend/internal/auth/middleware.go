package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

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
//
// It also slides the cookie's expiration forward: once less than half of its
// TTL remains, a fresh token/cookie is issued on the spot. A user who keeps
// visiting never hits the expiry at all; only someone who stops returning
// for a full TTL period actually gets logged out.
func RequireAuth(db *mongo.Database, jwtSecret string, cookieCfg SessionCookieConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(cookieCfg.Name)
			if err != nil {
				unauthorized(w)
				return
			}
			authenticate(db, jwtSecret, cookie.Value, w, r, next, &cookieCfg)
		})
	}
}

// RequireAuthCookieOrTicket is RequireAuth plus a fallback to a short-lived
// "?ticket=" query param. It exists only for the voice WebSocket route: that
// connection goes directly to the backend's own origin (Next.js rewrites
// can't proxy a WS upgrade), which is cross-site from the frontend's origin -
// Safari's ITP won't reliably carry the session cookie there even though the
// cookie itself now works fine for every proxied REST call. The ticket
// (minted by SessionHandler.IssueWSTicket over the already-proxied, cookie-
// authenticated REST path, 60s TTL) sidesteps that without needing a raw
// WS-upgrade proxy or a shared custom domain. The cookie is still checked
// first and still works for any client that does carry it (e.g. a browser
// with an existing same-site-friendly setup) - the ticket is additive, not a
// replacement.
func RequireAuthCookieOrTicket(db *mongo.Database, jwtSecret string, cookieCfg SessionCookieConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if ticket := r.URL.Query().Get("ticket"); ticket != "" {
				// A ws-ticket is a 60s-lived, single-purpose token, not the
				// real session - it never gets refreshed/slid forward.
				authenticate(db, jwtSecret, ticket, w, r, next, nil)
				return
			}
			cookie, err := r.Cookie(cookieCfg.Name)
			if err != nil {
				unauthorized(w)
				return
			}
			authenticate(db, jwtSecret, cookie.Value, w, r, next, &cookieCfg)
		})
	}
}

// authenticate validates a raw token string (from either a cookie or a
// ws-ticket - same Claims shape, same TokenVersion revocation check either
// way) and, on success, injects the user ID into the request context. When
// refreshCfg is non-nil and the token is past the halfway point of its
// lifetime, a fresh token is minted and written back as a new cookie before
// the request continues (sliding expiration).
func authenticate(db *mongo.Database, jwtSecret, tokenString string, w http.ResponseWriter, r *http.Request, next http.Handler, refreshCfg *SessionCookieConfig) {
	claims, err := ParseToken(tokenString, jwtSecret)
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

	if refreshCfg != nil && claims.ExpiresAt != nil {
		if time.Until(claims.ExpiresAt.Time) < refreshCfg.TTL/2 {
			if newToken, err := MintToken(claims.UserID, claims.TokenVersion, jwtSecret, refreshCfg.TTL); err == nil {
				SetSessionCookie(w, refreshCfg.Name, newToken, refreshCfg.Environment, refreshCfg.TTL)
			}
		}
	}

	ctx := context.WithValue(r.Context(), userIDContextKey, claims.UserID)
	next.ServeHTTP(w, r.WithContext(ctx))
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

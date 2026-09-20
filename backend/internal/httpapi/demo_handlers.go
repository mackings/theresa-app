package httpapi

import (
	"context"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"theresa/backend/internal/auth"
	"theresa/backend/internal/config"
	"theresa/backend/internal/models"
)

// demoFreeTrialSeconds is TEMPORARILY set to a full 30 minutes (matching the
// regular free trial) while Theresa is under review for Google for Startups
// Cloud credits - the reviewer needs to be able to actually hold a real
// conversation with Theresa via /try, not hit a wall after 90 seconds.
// Revert to a short preview window (e.g. 90) once approved. The existing
// FreeTrialSecondsRemaining/CreditBalanceKobo gate in
// LiveHandler.HandleConnection (a demo account has zero credit balance and
// no payment method) cuts the connection off automatically once this runs
// out, with the same graceful farewell every other out-of-credits account
// gets - no separate demo-specific billing logic needed anywhere.
const demoFreeTrialSeconds = 30 * 60

// DemoHandler is deliberately tiny: its only job is minting the ephemeral
// account and cookie. Every real capability (session creation, the voice
// WebSocket, board rendering, billing cutoff) is the exact same code path a
// real signed-up user goes through - see the IsDemo doc comment on
// models.User for what's different about this account.
type DemoHandler struct {
	db  *mongo.Database
	cfg config.Config

	// Keyed by client IP, not by anything account-based - a demo account has
	// no password to rate-limit login attempts against, so this is the only
	// lever bounding how many ephemeral accounts one visitor (or a script)
	// can mint.
	startLimiter *auth.RateLimiter
}

func NewDemoHandler(db *mongo.Database, cfg config.Config) *DemoHandler {
	return &DemoHandler{
		db:           db,
		cfg:          cfg,
		startLimiter: auth.NewRateLimiter(3, time.Hour),
	}
}

func (h *DemoHandler) users() *mongo.Collection {
	return h.db.Collection("users")
}

// Start creates a real but ephemeral, password-less account and logs the
// caller into it via the normal session cookie - no signup form, no email
// verification, and (unlike every other account) never reachable again once
// this browser's cookie is gone. Public - deliberately not behind
// RequireAuth, since the entire point is working without an account.
func (h *DemoHandler) Start(w http.ResponseWriter, r *http.Request) {
	if !h.startLimiter.Allow(clientIP(r)) {
		writeError(w, http.StatusTooManyRequests, "too many demo sessions started from this network, please try again later")
		return
	}

	now := time.Now()
	user := models.User{
		// Never a real inbox - only exists to satisfy the users.email unique
		// index, and nothing ever emails it (demo accounts skip every email
		// flow: no verification, no low-credits notice, no receipts).
		Email:                     "demo-" + bson.NewObjectID().Hex() + "@demo.asktheresa.internal",
		Name:                      "Guest",
		EmailVerified:             true,
		IsDemo:                    true,
		CreatedAt:                 now,
		FreeTrialSecondsRemaining: demoFreeTrialSeconds,
		FreeTrialResetAt:          now,
	}
	result, err := h.users().InsertOne(r.Context(), user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to start demo")
		return
	}
	user.ID = result.InsertedID.(bson.ObjectID)

	token, err := auth.MintToken(user.ID.Hex(), user.TokenVersion, h.cfg.JWTSecret, h.cfg.JWTTTL)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to start demo")
		return
	}
	auth.SetSessionCookie(w, h.cfg.JWTCookieName, token, h.cfg.Environment, h.cfg.JWTTTL)

	writeJSON(w, http.StatusOK, map[string]any{
		"free_trial_seconds_remaining": demoFreeTrialSeconds,
	})
}

// isDemoCaller reports whether the given user is an ephemeral demo account.
// Checked before any handler that would trigger real Gemini generation cost
// outside the voice/text chat path the tiny demo free-trial already bounds -
// document understanding, learning-plan generation, organization creation -
// since none of those are gated by the automatic free-trial/credit cutoff
// voice sessions get for free.
func isDemoCaller(ctx context.Context, db *mongo.Database, ownerID bson.ObjectID) bool {
	var u struct {
		IsDemo bool `bson:"is_demo"`
	}
	opts := options.FindOne().SetProjection(bson.M{"is_demo": 1})
	if err := db.Collection("users").FindOne(ctx, bson.M{"_id": ownerID}, opts).Decode(&u); err != nil {
		return false
	}
	return u.IsDemo
}

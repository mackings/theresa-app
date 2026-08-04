package billing

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"theresa/backend/internal/models"
)

// RefreshDailyFreeTrial grants a fresh FreeTrialSeconds allowance if the
// user's last reset wasn't today (UTC calendar day) - called wherever free
// trial time matters (starting a voice connection, checking the balance) so
// the daily allowance actually replenishes instead of only ever being
// granted once at signup. Mutates user in place and persists the change only
// when a reset actually happens; returns whether it did.
func RefreshDailyFreeTrial(ctx context.Context, db *mongo.Database, user *models.User) bool {
	now := time.Now().UTC()
	if sameUTCDay(user.FreeTrialResetAt, now) {
		return false
	}

	user.FreeTrialSecondsRemaining = FreeTrialSeconds
	user.FreeTrialResetAt = now

	db.Collection("users").UpdateOne(ctx, bson.M{"_id": user.ID}, bson.M{
		"$set": bson.M{
			"free_trial_seconds_remaining": user.FreeTrialSecondsRemaining,
			"free_trial_reset_at":          user.FreeTrialResetAt,
		},
	})
	return true
}

func sameUTCDay(a, b time.Time) bool {
	a, b = a.UTC(), b.UTC()
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

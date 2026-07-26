package billing

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"theresa/backend/internal/models"
)

// Thresholds are checked in used-percent terms (50% of the current credit
// cycle used, etc.), each firing at most once per cycle - Notified50/75/95
// on the user record reset to false on every new top-up.
var thresholds = []int{50, 75, 95}

// DeductResult reports what actually happened when trying to deduct kobo
// from a user's balance - a session's last billing tick might only be
// partially affordable, so the caller (the voice relay) needs the real
// deducted amount, not just whether the call "succeeded".
type DeductResult struct {
	DeductedKobo      int64
	RemainingKobo     int64
	OutOfCredits      bool // balance hit 0 and couldn't cover the full charge
	CrossedThresholds []int
}

// Deduct atomically subtracts up to chargeKobo from a user's balance,
// clamping at 0 rather than going negative, via a single aggregation-
// pipeline update so the clamp and the read of the prior balance happen in
// one atomic round trip - no separate read-then-write race window.
func Deduct(ctx context.Context, db *mongo.Database, userID bson.ObjectID, chargeKobo int64, sessionID bson.ObjectID, detail map[string]any) (DeductResult, error) {
	if chargeKobo <= 0 {
		return DeductResult{}, nil
	}

	var before models.User
	err := db.Collection("users").FindOneAndUpdate(
		ctx,
		bson.M{"_id": userID},
		mongo.Pipeline{
			bson.D{{Key: "$set", Value: bson.D{
				{Key: "credit_balance_kobo", Value: bson.D{{Key: "$max", Value: bson.A{
					0,
					bson.D{{Key: "$subtract", Value: bson.A{"$credit_balance_kobo", chargeKobo}}},
				}}}},
			}}},
		},
		options.FindOneAndUpdate().SetReturnDocument(options.Before),
	).Decode(&before)
	if err != nil {
		return DeductResult{}, err
	}

	newBalance := before.CreditBalanceKobo - chargeKobo
	if newBalance < 0 {
		newBalance = 0
	}
	deducted := before.CreditBalanceKobo - newBalance

	if deducted > 0 {
		txDetail := map[string]any{}
		for k, v := range detail {
			txDetail[k] = v
		}
		db.Collection("credit_transactions").InsertOne(ctx, models.CreditTransaction{
			UserID:     userID,
			SessionID:  &sessionID,
			Type:       "voice_usage",
			AmountKobo: -deducted,
			Detail:     txDetail,
			CreatedAt:  time.Now(),
		})
	}

	crossed := checkAndMarkThresholds(ctx, db, userID, before, newBalance)

	return DeductResult{
		DeductedKobo:      deducted,
		RemainingKobo:     newBalance,
		OutOfCredits:      newBalance == 0 && deducted < chargeKobo,
		CrossedThresholds: crossed,
	}, nil
}

// checkAndMarkThresholds compares used-percent before/after a deduction
// against the user's credit cycle baseline, marks any newly-crossed
// threshold as notified (so it never fires twice in the same cycle), and
// returns which ones were newly crossed by this specific deduction.
func checkAndMarkThresholds(ctx context.Context, db *mongo.Database, userID bson.ObjectID, before models.User, newBalanceKobo int64) []int {
	if before.CreditCycleStartKobo <= 0 {
		return nil
	}

	usedBefore := 1 - float64(before.CreditBalanceKobo)/float64(before.CreditCycleStartKobo)
	usedAfter := 1 - float64(newBalanceKobo)/float64(before.CreditCycleStartKobo)

	alreadyNotified := map[int]bool{
		50: before.Notified50Percent,
		75: before.Notified75Percent,
		95: before.Notified95Percent,
	}

	var newlyCrossed []int
	setFields := bson.M{}
	for _, t := range thresholds {
		frac := float64(t) / 100
		if !alreadyNotified[t] && usedAfter >= frac && usedBefore < frac {
			newlyCrossed = append(newlyCrossed, t)
			switch t {
			case 50:
				setFields["notified_50_percent"] = true
			case 75:
				setFields["notified_75_percent"] = true
			case 95:
				setFields["notified_95_percent"] = true
			}
		}
	}

	if len(setFields) > 0 {
		db.Collection("users").UpdateOne(ctx, bson.M{"_id": userID}, bson.M{"$set": setFields})
	}

	return newlyCrossed
}

// ErrAlreadyCredited means this flw_tx_ref has already been processed -
// Flutterwave retries a webhook until it gets a 200, so the same successful
// payment can genuinely arrive more than once. Not an error to the caller,
// just a signal to ack and stop.
var ErrAlreadyCredited = errors.New("transaction already credited")

// Credit adds kobo to a user's balance from a successful purchase and
// resets the credit-cycle baseline (and the three notification flags) to
// start fresh against the new total. Called only after a payment has been
// independently verified server-side - see the payments package - never
// directly from an unverified webhook body.
//
// The transaction record is inserted FIRST, before the balance is touched -
// flw_tx_ref has a unique index (see db.EnsureIndexes), so a duplicate
// insert attempt (a retried webhook for an already-processed payment) fails
// immediately with ErrAlreadyCredited and the balance is never
// double-credited, rather than relying on a separate read-check-write that
// would leave a race window.
func Credit(ctx context.Context, db *mongo.Database, userID bson.ObjectID, amountKobo int64, flwTxRef, flwTransactionID string) (newBalanceKobo int64, err error) {
	_, err = db.Collection("credit_transactions").InsertOne(ctx, models.CreditTransaction{
		UserID:           userID,
		Type:             "purchase",
		AmountKobo:       amountKobo,
		FlwTxRef:         flwTxRef,
		FlwTransactionID: flwTransactionID,
		Status:           "completed",
		CreatedAt:        time.Now(),
	})
	if mongo.IsDuplicateKeyError(err) {
		return 0, ErrAlreadyCredited
	}
	if err != nil {
		return 0, err
	}

	var after models.User
	err = db.Collection("users").FindOneAndUpdate(
		ctx,
		bson.M{"_id": userID},
		mongo.Pipeline{
			bson.D{{Key: "$set", Value: bson.D{
				{Key: "credit_balance_kobo", Value: bson.D{{Key: "$add", Value: bson.A{"$credit_balance_kobo", amountKobo}}}},
			}}},
		},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&after)
	if err != nil {
		return 0, err
	}

	db.Collection("users").UpdateOne(ctx, bson.M{"_id": userID}, bson.M{
		"$set": bson.M{
			"credit_cycle_start_kobo": after.CreditBalanceKobo,
			"notified_50_percent":     false,
			"notified_75_percent":     false,
			"notified_95_percent":     false,
		},
	})

	return after.CreditBalanceKobo, nil
}

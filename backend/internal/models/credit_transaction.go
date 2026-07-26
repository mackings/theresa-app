package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// CreditTransaction is an append-only audit log entry for every change to a
// user's credit balance - both deductions (voice usage) and additions
// (Flutterwave purchases). AmountKobo is signed: negative for a deduction,
// positive for a credit. Never mutated after creation; the user's current
// CreditBalanceKobo is the running total, this collection is the receipt
// trail behind it (and, for usage rows, the raw data needed to eventually
// replace assumed speaking-time splits with real measured ones).
type CreditTransaction struct {
	ID        bson.ObjectID  `bson:"_id,omitempty" json:"id"`
	UserID    bson.ObjectID  `bson:"user_id" json:"-"`
	SessionID *bson.ObjectID `bson:"session_id,omitempty" json:"session_id,omitempty"`

	// Type is "voice_usage" | "purchase" | "free_trial".
	Type       string `bson:"type" json:"type"`
	AmountKobo int64  `bson:"amount_kobo" json:"amount_kobo"`

	// FlwTxRef/FlwTransactionID are only set for Type "purchase" - FlwTxRef
	// is the reference we generate and send to Flutterwave when initiating
	// payment; it's what the unique index below relies on to make webhook
	// retries idempotent (Flutterwave retries a webhook until it gets a 200,
	// so the same successful payment can arrive more than once).
	FlwTxRef         string `bson:"flw_tx_ref,omitempty" json:"-"`
	FlwTransactionID string `bson:"flw_transaction_id,omitempty" json:"-"`
	// Status is only meaningful for Type "purchase": "pending" until the
	// webhook (plus server-side verification) confirms it, then "completed"
	// or "failed". Voice-usage rows are always already-settled facts, no
	// pending state.
	Status string `bson:"status,omitempty" json:"status,omitempty"`

	// Detail carries a breakdown for debugging/analytics - e.g. for
	// voice_usage: {"user_audio_seconds": 12.4, "theresa_audio_seconds":
	// 28.9, "text_in_tokens": 40, "text_out_tokens": 210}.
	Detail map[string]any `bson:"detail,omitempty" json:"detail,omitempty"`

	CreatedAt time.Time `bson:"created_at" json:"created_at"`
}

package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type User struct {
	ID                         bson.ObjectID `bson:"_id,omitempty"`
	Email                      string        `bson:"email"`
	Name                       string        `bson:"name"`
	PasswordHash               string        `bson:"password_hash"`
	EmailVerified              bool          `bson:"email_verified"`
	VerificationTokenHash      string        `bson:"verification_token_hash,omitempty"`
	VerificationTokenExpiresAt time.Time     `bson:"verification_token_expires_at,omitempty"`
	ResetTokenHash             string        `bson:"reset_token_hash,omitempty"`
	ResetTokenExpiresAt        time.Time     `bson:"reset_token_expires_at,omitempty"`
	CreatedAt                  time.Time     `bson:"created_at"`
	LastLoginAt                time.Time     `bson:"last_login_at"`

	// TokenVersion is embedded in every minted JWT and checked against the
	// user's current value on every authenticated request - bumping it
	// (logout, password reset) instantly invalidates every previously-issued
	// token, even ones that haven't expired yet. Missing/zero on existing
	// documents decodes as 0, matching a freshly-minted token's default, so
	// no backfill is needed for accounts that predate this field.
	TokenVersion int `bson:"token_version"`

	// CreditBalanceKobo is money, stored as an integer count of kobo (1/100
	// of a naira) rather than a float - avoids floating-point rounding error
	// accumulating across many small per-session deductions, which matters
	// far more for a running balance than for a one-off display value.
	CreditBalanceKobo int64 `bson:"credit_balance_kobo"`

	// FreeTrialSecondsRemaining starts at 600 (10 minutes) for a new account
	// and decrements as voice sessions run, before any credits are touched.
	// Cumulative across a single day's sessions, not per-session - a user can
	// spend it in several short calls instead of one 10-minute sitting. Reset
	// back to the full daily allowance once a day (see billing.RefreshDailyFreeTrial),
	// tracked by FreeTrialResetAt below.
	FreeTrialSecondsRemaining int `bson:"free_trial_seconds_remaining"`

	// FreeTrialResetAt is when FreeTrialSecondsRemaining was last reset to
	// the full daily allowance - compared against the current UTC calendar
	// day to decide whether today's allowance still needs granting.
	FreeTrialResetAt time.Time `bson:"free_trial_reset_at,omitempty"`

	// CreditCycleStartKobo is the balance immediately after the most recent
	// top-up (or 0 if never topped up) - the baseline the 50/75/95%-used
	// notifications are measured against. Reset on every successful
	// purchase, along with the three Notified flags below.
	CreditCycleStartKobo int64 `bson:"credit_cycle_start_kobo"`
	Notified50Percent    bool  `bson:"notified_50_percent"`
	Notified75Percent    bool  `bson:"notified_75_percent"`
	Notified95Percent    bool  `bson:"notified_95_percent"`
}

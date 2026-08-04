package billing

import (
	"testing"
	"time"
)

func TestSameUTCDay(t *testing.T) {
	day1 := time.Date(2026, 8, 4, 10, 0, 0, 0, time.UTC)
	day1Later := time.Date(2026, 8, 4, 23, 59, 0, 0, time.UTC)
	day2 := time.Date(2026, 8, 5, 0, 1, 0, 0, time.UTC)
	zero := time.Time{}

	if !sameUTCDay(day1, day1Later) {
		t.Error("same calendar day (different times) should be equal")
	}
	if sameUTCDay(day1, day2) {
		t.Error("different calendar days should not be equal")
	}
	if sameUTCDay(zero, day1) {
		t.Error("zero-value time (never reset) should never match a real day")
	}

	// A timestamp stored in a non-UTC offset must still be compared by its
	// underlying UTC calendar day, not its local wall-clock date.
	loc := time.FixedZone("UTC-5", -5*60*60)
	day1LateLocal := time.Date(2026, 8, 4, 23, 30, 0, 0, loc) // = 2026-08-05T04:30 UTC
	if sameUTCDay(day1, day1LateLocal) {
		t.Error("expected these to fall on different UTC calendar days")
	}
}

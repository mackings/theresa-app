package billing

import "testing"

func almostEqual(a, b, tolerance float64) bool {
	diff := a - b
	if diff < 0 {
		diff = -diff
	}
	return diff <= tolerance
}

func TestUsageCostUSD_AudioOnly(t *testing.T) {
	// 1 minute of Theresa speaking only, nothing else.
	got := UsageCostUSD(0, 60, 0, 0)
	want := 0.018 // Google's stated $0.018/min for audio output
	if !almostEqual(got, want, 1e-9) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestUsageCostUSD_UserAudioOnly(t *testing.T) {
	// 1 minute of the user speaking only.
	got := UsageCostUSD(60, 0, 0, 0)
	want := 0.005 // Google's stated $0.005/min for audio input
	if !almostEqual(got, want, 1e-9) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestUsageCostUSD_BlendedMinute(t *testing.T) {
	// 70% Theresa speaking, 30% user speaking, over 1 minute.
	got := UsageCostUSD(0.3*60, 0.7*60, 0, 0)
	want := 0.7*0.018 + 0.3*0.005
	if !almostEqual(got, want, 1e-9) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestUsageCostUSD_TextTokens(t *testing.T) {
	got := UsageCostUSD(0, 0, 1_000_000, 1_000_000)
	want := 0.75 + 4.50
	if !almostEqual(got, want, 1e-9) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestPriceUSD_PreservesMargin(t *testing.T) {
	cost := 1.00
	price := PriceUSD(cost)
	// cost should be exactly (1 - ProfitMargin) of price - i.e. profit is
	// exactly ProfitMargin of price, not of cost.
	impliedCostFraction := cost / price
	if !almostEqual(impliedCostFraction, 1-ProfitMargin, 1e-9) {
		t.Errorf("cost/price = %v, want %v", impliedCostFraction, 1-ProfitMargin)
	}
	profit := price - cost
	profitMargin := profit / price
	if !almostEqual(profitMargin, ProfitMargin, 1e-9) {
		t.Errorf("profit margin = %v, want %v", profitMargin, ProfitMargin)
	}
}

func TestKoboFromUSD_Rounding(t *testing.T) {
	cases := []struct {
		usd, rate float64
		wantKobo  int64
	}{
		{1.0, 1380, 138000},  // $1 at ₦1380/$1 = ₦1380.00 = 138000 kobo
		{0.0225, 1380, 3105}, // ₦31.05 = 3105 kobo
		{0.005, 1000, 500},   // ₦5.00 = 500 kobo
		{0.0001, 1380, 14},   // ₦0.138 rounds to 14 kobo
	}
	for _, c := range cases {
		got := KoboFromUSD(c.usd, c.rate)
		if got != c.wantKobo {
			t.Errorf("KoboFromUSD(%v, %v) = %v, want %v", c.usd, c.rate, got, c.wantKobo)
		}
	}
}

func TestChargeKobo_MatchesHandCalculation(t *testing.T) {
	// 1 minute, 70/30 Theresa/user split, no text overhead, at ₦1380/$1.
	// cost = 0.7*0.018 + 0.3*0.005 = 0.0141
	// price = 0.0141 / 0.8 = 0.017625
	// kobo = 0.017625 * 1380 * 100 = 2432.25 -> rounds to 2432
	got := ChargeKobo(0.3*60, 0.7*60, 0, 0, 1380)
	want := int64(2432)
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestChargeKobo_OneThousandNairaCoversExpectedMinutes(t *testing.T) {
	// Sanity check against the hand-computed business estimate: at a 70/30
	// split with no text overhead, ~1000 naira (100000 kobo) should cover
	// roughly 41 minutes (2466 seconds) - matches the ~41.1 min figure
	// derived by hand for the audio-only case.
	const targetKobo = int64(100000)
	seconds := 0
	for {
		charge := ChargeKobo(0.3*float64(seconds), 0.7*float64(seconds), 0, 0, 1380)
		if charge > targetKobo {
			break
		}
		seconds++
	}
	minutes := float64(seconds) / 60
	if minutes < 40 || minutes > 42 {
		t.Errorf("1000 naira covered %.1f minutes, expected roughly 41", minutes)
	}
}

func TestEstimateTextTokens(t *testing.T) {
	got := EstimateTextTokens(400)
	want := 100 // 400 chars / 4 chars-per-token
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

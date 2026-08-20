// Package billing turns real Gemini usage into what a user should be
// charged, in kobo (1/100 of a naira). All pricing constants are sourced
// directly from Gemini's official pricing page
// (ai.google.dev/gemini-api/docs/pricing) for gemini-3.1-flash-live-preview,
// Standard tier - the exact model Theresa's voice mode uses - not from any
// third-party pricing tracker. Google's page states audio tokenizes at 25
// tokens/second; the per-minute figures below are Google's own stated
// equivalents of that rate, not a derived estimate.
package billing

const (
	// Per-second USD cost, converted from Google's stated per-minute rates
	// ($0.005/min audio in, $0.018/min audio out).
	AudioInputUSDPerSecond  = 0.005 / 60.0 // user speaking -> Gemini audio input
	AudioOutputUSDPerSecond = 0.018 / 60.0 // Theresa speaking -> Gemini audio output

	// Per-token USD cost for the live session's text portion (system
	// instruction, show_working/draw_diagram tool-call payloads) - also
	// gemini-3.1-flash-live-preview Standard tier text rates.
	TextInputUSDPerToken  = 0.75 / 1_000_000
	TextOutputUSDPerToken = 4.50 / 1_000_000

	// ProfitMargin is the minimum fraction of what a user is charged that
	// must remain as profit after covering Gemini's raw cost - i.e. cost may
	// be at most (1 - ProfitMargin) of the price charged.
	ProfitMargin = 0.20

	// FreeTrialSeconds is 30 minutes of voice, free, before any credits are
	// touched - granted fresh once per calendar day (see
	// RefreshDailyFreeTrial), not a one-time lifetime allowance. Cumulative
	// across a day's sessions, not a per-session allowance.
	FreeTrialSeconds = 1800

	// charsPerTokenEstimate approximates tokens from character count, used
	// only for the small board-update tool-call payloads where an exact
	// token count isn't available from the API response. A standard rough
	// heuristic for English text - not a Gemini-published figure, since
	// Google's page prices per token, not per character.
	charsPerTokenEstimate = 4.0
)

// EstimateTextTokens approximates a token count from a character count for
// the small tool-call payloads inside a voice session. Not intended for
// anything billed at real scale, where an exact token count should be used
// instead.
func EstimateTextTokens(chars int) int {
	return int(float64(chars)/charsPerTokenEstimate + 0.5)
}

// UsageCostUSD is the raw Gemini cost (no margin applied) for the given
// amounts of audio, in seconds, and text, in tokens.
func UsageCostUSD(userAudioSeconds, theresaAudioSeconds float64, textInTokens, textOutTokens int) float64 {
	return userAudioSeconds*AudioInputUSDPerSecond +
		theresaAudioSeconds*AudioOutputUSDPerSecond +
		float64(textInTokens)*TextInputUSDPerToken +
		float64(textOutTokens)*TextOutputUSDPerToken
}

// PriceUSD marks up a raw cost so charging it preserves at least
// ProfitMargin - e.g. a $1.00 cost is priced at $1.25 for a 20% margin
// ($1.00 / (1 - 0.20) = $1.25, of which $0.25 is the 20% profit on $1.25).
func PriceUSD(costUSD float64) float64 {
	return costUSD / (1 - ProfitMargin)
}

// KoboFromUSD converts a USD amount to whole kobo (1/100 naira) at the given
// exchange rate, rounding to the nearest kobo. The only place a USD<->NGN
// conversion happens, since Gemini bills in USD but Theresa charges in NGN.
func KoboFromUSD(usd, usdToNGNRate float64) int64 {
	naira := usd * usdToNGNRate
	return int64(naira*100 + 0.5)
}

// ChargeKobo computes the kobo to deduct for a slice of real usage,
// including the profit margin - the single entry point callers should use
// rather than composing the pieces above by hand.
func ChargeKobo(userAudioSeconds, theresaAudioSeconds float64, textInTokens, textOutTokens int, usdToNGNRate float64) int64 {
	cost := UsageCostUSD(userAudioSeconds, theresaAudioSeconds, textInTokens, textOutTokens)
	price := PriceUSD(cost)
	return KoboFromUSD(price, usdToNGNRate)
}

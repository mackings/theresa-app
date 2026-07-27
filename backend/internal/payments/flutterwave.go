// Package payments wraps Flutterwave's v3 API for buying voice credits.
// Request/response shapes for InitiatePayment and VerifyTransaction are
// taken directly from Flutterwave's official developer docs
// (developer.flutterwave.com/docs/making-payments/standard and the
// transaction-verification page) - not guessed.
//
// One thing worth flagging honestly: Flutterwave's webhook-payload docs
// show two different shapes across pages - an older v3-style shape (numeric
// "id", "tx_ref", status "successful", matching the verify-endpoint response
// below) and what looks like a newer v4/merchant-API shape ("chg_..." string
// ids, "reference", status "succeeded"). Since payments here are initiated
// through the v3 /v3/payments endpoint, the webhook for those should be
// v3-style - but the webhook handler parses defensively for both rather than
// betting everything on one shape, and crediting an account never trusts
// the webhook body alone regardless - it always re-confirms via
// VerifyTransaction, a call whose shape is unambiguous, before crediting.
package payments

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

const baseURL = "https://api.flutterwave.com/v3"

type Client struct {
	secretKey  string
	httpClient *http.Client
}

func NewClient(secretKey string) *Client {
	return &Client{
		secretKey:  secretKey,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

type InitiateRequest struct {
	TxRef         string
	AmountNaira   int64
	RedirectURL   string
	CustomerEmail string
	CustomerName  string
}

// InitiatePayment creates a Flutterwave-hosted checkout link for the given
// amount and returns its URL for the browser to redirect to.
func (c *Client) InitiatePayment(ctx context.Context, req InitiateRequest) (paymentLink string, err error) {
	body := map[string]any{
		"tx_ref":       req.TxRef,
		"amount":       strconv.FormatInt(req.AmountNaira, 10),
		"currency":     "NGN",
		"redirect_url": req.RedirectURL,
		"customer": map[string]any{
			"email": req.CustomerEmail,
			"name":  req.CustomerName,
		},
		"customizations": map[string]any{
			"title": "Theresa Voice Credits",
		},
	}

	raw, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/payments", bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.secretKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var parsed struct {
		Status  string `json:"status"`
		Message string `json:"message"`
		Data    struct {
			Link string `json:"link"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return "", fmt.Errorf("decode flutterwave response: %w", err)
	}
	if parsed.Status != "success" || parsed.Data.Link == "" {
		return "", fmt.Errorf("flutterwave initiate payment failed: %s", parsed.Message)
	}

	return parsed.Data.Link, nil
}

type VerifyResult struct {
	Status        string // "successful" when genuinely paid
	AmountNaira   float64
	Currency      string
	TxRef         string
	TransactionID string
}

// VerifyTransaction independently confirms a transaction's real status
// directly from Flutterwave, using our secret key - the only thing that
// should ever gate crediting an account, never a webhook body by itself.
func (c *Client) VerifyTransaction(ctx context.Context, transactionID string) (*VerifyResult, error) {
	url := fmt.Sprintf("%s/transactions/%s/verify", baseURL, transactionID)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.secretKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var parsed struct {
		Status string `json:"status"`
		Data   struct {
			ID       json.Number `json:"id"`
			TxRef    string      `json:"tx_ref"`
			Amount   float64     `json:"amount"`
			Currency string      `json:"currency"`
			Status   string      `json:"status"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("decode flutterwave verify response: %w", err)
	}

	return &VerifyResult{
		Status:        parsed.Data.Status,
		AmountNaira:   parsed.Data.Amount,
		Currency:      parsed.Data.Currency,
		TxRef:         parsed.Data.TxRef,
		TransactionID: parsed.Data.ID.String(),
	}, nil
}

// VerifyWebhookSignature checks the flutterwave-signature header against an
// HMAC-SHA256 of the raw request body, keyed with the configured webhook
// secret hash - confirms the request came from Flutterwave before any of
// its contents are trusted. Only a fast pre-filter, though: crediting an
// account additionally always requires an independent VerifyTransaction
// call, per the package doc comment above.
func VerifyWebhookSignature(body []byte, signatureHeader, secretHash string) bool {
	if signatureHeader == "" || secretHash == "" {
		return false
	}
	expected := ExpectedWebhookSignature(body, secretHash)
	return hmac.Equal([]byte(expected), []byte(signatureHeader))
}

// VerifyWebhookAuth accepts either of Flutterwave's two real-world webhook
// authentication schemes, since production traffic has been observed using
// the classic one even for payments initiated through the v3 API (contrary
// to the docs, which describe only the newer scheme for v3):
//   - newer: HMAC-SHA256 of the raw body in the "flutterwave-signature"
//     header (VerifyWebhookSignature above).
//   - classic: the secret hash itself, verbatim, echoed back in the
//     "verif-hash" header - no hashing, just a direct equality check.
//
// Accepted if either header is present and matches; both are still only a
// fast pre-filter, never a substitute for the independent VerifyTransaction
// call before crediting anything.
func VerifyWebhookAuth(body []byte, flutterwaveSignature, verifHash, secretHash string) bool {
	if flutterwaveSignature != "" && VerifyWebhookSignature(body, flutterwaveSignature, secretHash) {
		return true
	}
	if verifHash != "" && secretHash != "" && hmac.Equal([]byte(verifHash), []byte(secretHash)) {
		return true
	}
	return false
}

// ExpectedWebhookSignature computes the HMAC-SHA256 hex digest a valid
// webhook for this exact body should carry - exported separately so a
// rejected webhook can log what we computed alongside what was received,
// for diagnosing a real mismatch without exposing the secret hash itself.
func ExpectedWebhookSignature(body []byte, secretHash string) string {
	mac := hmac.New(sha256.New, []byte(secretHash))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

// WebhookTransactionID extracts a transaction id from a Flutterwave webhook
// body, tolerant of the shape differences noted in the package doc comment -
// tries "data.id" as either a number or a string.
func WebhookTransactionID(body []byte) (string, bool) {
	var parsed struct {
		Data struct {
			ID json.RawMessage `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", false
	}
	if len(parsed.Data.ID) == 0 {
		return "", false
	}

	var asNumber json.Number
	if err := json.Unmarshal(parsed.Data.ID, &asNumber); err == nil && asNumber.String() != "" {
		return asNumber.String(), true
	}
	var asString string
	if err := json.Unmarshal(parsed.Data.ID, &asString); err == nil && asString != "" {
		return asString, true
	}
	return "", false
}

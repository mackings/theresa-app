package httpapi

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"theresa/backend/internal/auth"
	"theresa/backend/internal/billing"
	"theresa/backend/internal/config"
	"theresa/backend/internal/models"
	"theresa/backend/internal/payments"
)

const minPurchaseNaira = 1000

// maxTransactionsListed caps the transactions history response - there's no
// pagination UI yet (matching every other list endpoint in this app, e.g.
// SessionHandler.List), and 200 rows is already far more than the profile
// page's history view needs to be useful.
const maxTransactionsListed = 200

type PaymentHandler struct {
	db  *mongo.Database
	cfg config.Config
	flw *payments.Client

	// Unlike login/reset (IP + account), this is IP-only: Initiate is
	// already behind auth.RequireAuth, so an account key would just be the
	// same authenticated user hitting their own limit - no distinct
	// cross-account abuse pattern to separately defend against here. The
	// real cost being guarded against is each call making a real
	// server-to-server request to Flutterwave's API.
	initiateIPLimiter *auth.RateLimiter
}

func NewPaymentHandler(db *mongo.Database, cfg config.Config) *PaymentHandler {
	return &PaymentHandler{
		db:                db,
		cfg:               cfg,
		flw:               payments.NewClient(cfg.FlutterwaveSecretKey),
		initiateIPLimiter: auth.NewRateLimiter(10, 15*time.Minute),
	}
}

// txRefForUser encodes the user id directly into the reference we send to
// Flutterwave - this means the webhook can identify who to credit purely
// from the tx_ref Flutterwave already echoes back, with no separate
// "pending payment" table to look it up in.
func txRefForUser(userID bson.ObjectID) string {
	return fmt.Sprintf("theresa-%s-%d", userID.Hex(), time.Now().UnixNano())
}

func userIDFromTxRef(txRef string) (bson.ObjectID, bool) {
	parts := strings.Split(txRef, "-")
	if len(parts) != 3 || parts[0] != "theresa" {
		return bson.ObjectID{}, false
	}
	id, err := bson.ObjectIDFromHex(parts[1])
	if err != nil {
		return bson.ObjectID{}, false
	}
	return id, true
}

type initiatePaymentRequest struct {
	AmountNaira int64 `json:"amount_naira"`
}

// Initiate creates a Flutterwave-hosted checkout link for the requested
// naira amount and returns it for the browser to redirect to. No database
// write happens here - see the payments package doc comment for why the
// webhook alone (after independent server-side verification) is what
// actually credits an account.
func (h *PaymentHandler) Initiate(w http.ResponseWriter, r *http.Request) {
	if h.cfg.FlutterwaveSecretKey == "" {
		writeError(w, http.StatusServiceUnavailable, "payments are not configured")
		return
	}
	if !h.initiateIPLimiter.Allow(clientIP(r)) {
		writeError(w, http.StatusTooManyRequests, "too many requests, try again later")
		return
	}

	userIDHex, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	userID, err := bson.ObjectIDFromHex(userIDHex)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req initiatePaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.AmountNaira < minPurchaseNaira {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("minimum purchase is %d naira", minPurchaseNaira))
		return
	}

	var user models.User
	if err := h.db.Collection("users").FindOne(r.Context(), bson.M{"_id": userID}).Decode(&user); err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	txRef := txRefForUser(userID)
	link, err := h.flw.InitiatePayment(r.Context(), payments.InitiateRequest{
		TxRef:         txRef,
		AmountNaira:   req.AmountNaira,
		RedirectURL:   h.cfg.FrontendURL + "/payment/callback",
		CustomerEmail: user.Email,
		CustomerName:  user.Name,
	})
	if err != nil {
		log.Printf("flutterwave initiate payment failed: %v", err)
		writeError(w, http.StatusBadGateway, "failed to start payment, please try again")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"payment_link": link})
}

// Webhook receives Flutterwave's payment notifications. Never trusts the
// body alone: checks the request's authentication header first as a fast
// filter (either of Flutterwave's two real-world schemes - see
// payments.VerifyWebhookAuth), then independently re-verifies the
// transaction directly with Flutterwave using our secret key before
// crediting anything - the verified server-to-server response is the only
// source of truth for whether a payment actually succeeded.
func (h *PaymentHandler) Webhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	signature := r.Header.Get("flutterwave-signature")
	verifHash := r.Header.Get("verif-hash")
	if !payments.VerifyWebhookAuth(body, signature, verifHash, h.cfg.FlutterwaveWebhookSecretKey) {
		log.Printf("webhook auth failed, rejecting (flutterwave-signature present=%v, verif-hash present=%v)",
			signature != "", verifHash != "")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	transactionID, ok := payments.WebhookTransactionID(body)
	if !ok {
		log.Printf("webhook body missing a transaction id, ignoring")
		w.WriteHeader(http.StatusOK)
		return
	}

	result, err := h.flw.VerifyTransaction(r.Context(), transactionID)
	if err != nil {
		log.Printf("webhook transaction verify failed: %v", err)
		// Ack anyway - this is our side failing to reach Flutterwave, not a
		// bad webhook. Returning non-200 here would just make Flutterwave
		// retry a call that's likely to fail the same way; there's nothing
		// webhook-retry can fix about our own outbound connectivity.
		w.WriteHeader(http.StatusOK)
		return
	}

	if result.Status != "successful" || result.Currency != "NGN" {
		log.Printf("webhook transaction %s not a successful NGN payment (status=%s currency=%s), ignoring",
			transactionID, result.Status, result.Currency)
		w.WriteHeader(http.StatusOK)
		return
	}

	userID, ok := userIDFromTxRef(result.TxRef)
	if !ok {
		log.Printf("webhook tx_ref %q doesn't match expected format, ignoring", result.TxRef)
		w.WriteHeader(http.StatusOK)
		return
	}

	amountKobo := int64(result.AmountNaira*100 + 0.5)
	_, err = billing.Credit(r.Context(), h.db, userID, amountKobo, result.TxRef, result.TransactionID)
	if err != nil && err != billing.ErrAlreadyCredited {
		log.Printf("crediting user %s for tx %s failed: %v", userID.Hex(), result.TxRef, err)
		// A genuine failure to write the credit - ack anyway so Flutterwave
		// doesn't hammer the endpoint on an infrastructure problem it can't
		// fix by retrying, but this is logged for manual reconciliation.
	}

	w.WriteHeader(http.StatusOK)
}

// Balance returns the authenticated user's current credit balance and free
// trial time remaining - used by the frontend for the balance indicator and
// after returning from the Flutterwave redirect.
func (h *PaymentHandler) Balance(w http.ResponseWriter, r *http.Request) {
	userIDHex, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	userID, err := bson.ObjectIDFromHex(userIDHex)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var user models.User
	if err := h.db.Collection("users").FindOne(r.Context(), bson.M{"_id": userID}).Decode(&user); err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"balance_kobo":                 user.CreditBalanceKobo,
		"free_trial_seconds_remaining": user.FreeTrialSecondsRemaining,
		"percent_remaining":            percentRemaining(user.CreditBalanceKobo, user.CreditCycleStartKobo),
	})
}

// Transactions returns the authenticated user's credit transaction history
// (purchases, voice usage deductions, free trial grants), most recent first
// - the profile page's transactions list.
func (h *PaymentHandler) Transactions(w http.ResponseWriter, r *http.Request) {
	userIDHex, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	userID, err := bson.ObjectIDFromHex(userIDHex)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetLimit(maxTransactionsListed)
	cursor, err := h.db.Collection("credit_transactions").Find(r.Context(), bson.M{"user_id": userID}, opts)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load transactions")
		return
	}

	transactions := []models.CreditTransaction{}
	if err := cursor.All(r.Context(), &transactions); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load transactions")
		return
	}

	writeJSON(w, http.StatusOK, transactions)
}

// percentRemaining is how much of the current top-up cycle is left, for
// surfaces that show usage without ever displaying a raw naira figure (the
// sidebar's persistent indicator) - /credits and the payment callback page
// still show the real balance directly, since showing an amount is the
// whole point there.
func percentRemaining(balanceKobo, cycleStartKobo int64) int {
	if cycleStartKobo <= 0 {
		return 0
	}
	pct := int(float64(balanceKobo) / float64(cycleStartKobo) * 100)
	if pct < 0 {
		return 0
	}
	if pct > 100 {
		return 100
	}
	return pct
}

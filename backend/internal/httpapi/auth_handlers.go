package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"theresa/backend/internal/auth"
	"theresa/backend/internal/billing"
	"theresa/backend/internal/config"
	"theresa/backend/internal/email"
	"theresa/backend/internal/models"
)

type AuthHandler struct {
	db           *mongo.Database
	cfg          config.Config
	email        *email.Client
	limiter      *auth.LoginLimiter
	resetLimiter *auth.LoginLimiter
}

func NewAuthHandler(db *mongo.Database, cfg config.Config, emailClient *email.Client) *AuthHandler {
	return &AuthHandler{
		db:           db,
		cfg:          cfg,
		email:        emailClient,
		limiter:      auth.NewLoginLimiter(5, 15*time.Minute),
		resetLimiter: auth.NewLoginLimiter(3, 15*time.Minute),
	}
}

func (h *AuthHandler) users() *mongo.Collection {
	return h.db.Collection("users")
}

type signupRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

func (h *AuthHandler) Signup(w http.ResponseWriter, r *http.Request) {
	var req signupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))
	if err := auth.ValidateEmail(email); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := auth.ValidatePassword(req.Password); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to process password")
		return
	}

	rawToken, tokenHash, err := auth.GenerateVerificationToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate verification token")
		return
	}

	user := models.User{
		Email:                      email,
		Name:                       strings.TrimSpace(req.Name),
		PasswordHash:               passwordHash,
		EmailVerified:              false,
		VerificationTokenHash:      tokenHash,
		VerificationTokenExpiresAt: time.Now().Add(auth.VerificationTokenTTL),
		CreatedAt:                  time.Now(),
		FreeTrialSecondsRemaining:  billing.FreeTrialSeconds,
	}

	ctx := r.Context()
	result, err := h.users().InsertOne(ctx, user)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			writeError(w, http.StatusConflict, "email already registered")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to create account")
		return
	}
	user.ID = result.InsertedID.(bson.ObjectID)

	verifyURL := h.cfg.FrontendURL + "/verify-email?token=" + rawToken
	if err := h.email.SendVerificationEmail(ctx, user.Email, user.Name, verifyURL); err != nil {
		// Compensate: don't leave an unverifiable, un-retryable account behind.
		_, _ = h.users().DeleteOne(context.Background(), bson.M{"_id": user.ID})
		writeError(w, http.StatusBadGateway, "failed to send verification email, please try again")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"id":             user.ID.Hex(),
		"email":          user.Email,
		"name":           user.Name,
		"email_verified": false,
		"message":        "check your email to verify your account",
	})
}

type verifyEmailRequest struct {
	Token string `json:"token"`
}

func (h *AuthHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	var req verifyEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Token == "" {
		writeError(w, http.StatusBadRequest, "invalid or expired verification link")
		return
	}

	tokenHash := auth.HashToken(req.Token)
	filter := bson.M{
		"verification_token_hash":       tokenHash,
		"verification_token_expires_at": bson.M{"$gt": time.Now()},
	}
	update := bson.M{
		"$set":   bson.M{"email_verified": true},
		"$unset": bson.M{"verification_token_hash": "", "verification_token_expires_at": ""},
	}

	result, err := h.users().UpdateOne(r.Context(), filter, update)
	if err != nil || result.MatchedCount == 0 {
		writeError(w, http.StatusBadRequest, "invalid or expired verification link")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "email verified"})
}

type forgotPasswordRequest struct {
	Email string `json:"email"`
}

// ForgotPassword always responds with the same generic message regardless of
// whether the email is registered - confirming/denying account existence via
// response shape would leak which emails have accounts.
func (h *AuthHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req forgotPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	const genericMessage = "if that email has an account, we've sent a password reset link"

	clientKey := clientIP(r)
	if !h.resetLimiter.Allow(clientKey) {
		writeError(w, http.StatusTooManyRequests, "too many requests, try again later")
		return
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))
	ctx := r.Context()

	var user models.User
	if err := h.users().FindOne(ctx, bson.M{"email": email}).Decode(&user); err != nil {
		writeJSON(w, http.StatusOK, map[string]string{"message": genericMessage})
		return
	}

	rawToken, tokenHash, err := auth.GenerateVerificationToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate reset token")
		return
	}

	_, err = h.users().UpdateOne(ctx, bson.M{"_id": user.ID}, bson.M{
		"$set": bson.M{
			"reset_token_hash":       tokenHash,
			"reset_token_expires_at": time.Now().Add(auth.ResetTokenTTL),
		},
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to process request")
		return
	}

	resetURL := h.cfg.FrontendURL + "/reset-password?token=" + rawToken
	if err := h.email.SendPasswordResetEmail(ctx, user.Email, user.Name, resetURL); err != nil {
		writeError(w, http.StatusBadGateway, "failed to send reset email, please try again")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": genericMessage})
}

type resetPasswordRequest struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req resetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Token == "" {
		writeError(w, http.StatusBadRequest, "invalid or expired reset link")
		return
	}
	if err := auth.ValidatePassword(req.Password); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to process password")
		return
	}

	tokenHash := auth.HashToken(req.Token)
	filter := bson.M{
		"reset_token_hash":       tokenHash,
		"reset_token_expires_at": bson.M{"$gt": time.Now()},
	}
	update := bson.M{
		"$set":   bson.M{"password_hash": passwordHash},
		"$unset": bson.M{"reset_token_hash": "", "reset_token_expires_at": ""},
	}

	result, err := h.users().UpdateOne(r.Context(), filter, update)
	if err != nil || result.MatchedCount == 0 {
		writeError(w, http.StatusBadRequest, "invalid or expired reset link")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "password updated"})
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	clientKey := clientIP(r)
	if !h.limiter.Allow(clientKey) {
		writeError(w, http.StatusTooManyRequests, "too many login attempts, try again later")
		return
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))

	var user models.User
	err := h.users().FindOne(r.Context(), bson.M{"email": email}).Decode(&user)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	if err := auth.VerifyPassword(user.PasswordHash, req.Password); err != nil {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	if !user.EmailVerified {
		writeError(w, http.StatusForbidden, "please verify your email before logging in")
		return
	}

	token, err := auth.MintToken(user.ID.Hex(), h.cfg.JWTSecret, h.cfg.JWTTTL)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create session")
		return
	}

	_, _ = h.users().UpdateOne(r.Context(), bson.M{"_id": user.ID}, bson.M{
		"$set": bson.M{"last_login_at": time.Now()},
	})

	h.setSessionCookie(w, token)
	writeJSON(w, http.StatusOK, map[string]any{
		"id":    user.ID.Hex(),
		"email": user.Email,
		"name":  user.Name,
	})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     h.cfg.JWTCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cfg.Environment == "production",
		SameSite: sameSiteFor(h.cfg.Environment),
		MaxAge:   -1,
	})
	writeJSON(w, http.StatusOK, map[string]string{"message": "logged out"})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
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
	if err := h.users().FindOne(r.Context(), bson.M{"_id": userID}).Decode(&user); err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"id":    user.ID.Hex(),
		"email": user.Email,
		"name":  user.Name,
	})
}

func (h *AuthHandler) setSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     h.cfg.JWTCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cfg.Environment == "production",
		SameSite: sameSiteFor(h.cfg.Environment),
		MaxAge:   int(h.cfg.JWTTTL.Seconds()),
	})
}

// sameSiteFor picks the cookie's SameSite mode based on environment. In
// production the frontend and backend are two separate Render services on
// different *.onrender.com hostnames - onrender.com is itself registered on
// the public suffix list specifically so different customers' subdomains
// aren't treated as "same site," which means SameSite=Lax would silently
// never send this cookie cross-service. SameSite=None (paired with the
// Secure flag, already tied to the same production check) is required for
// a cross-site cookie to be sent at all. Locally both run on "localhost"
// (different ports only), which is already same-site, so Lax is fine there.
func sameSiteFor(environment string) http.SameSite {
	if environment == "production" {
		return http.SameSiteNoneMode
	}
	return http.SameSiteLaxMode
}

func clientIP(r *http.Request) string {
	host := r.RemoteAddr
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		host = host[:idx]
	}
	return host
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

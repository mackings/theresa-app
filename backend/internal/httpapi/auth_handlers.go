package httpapi

import (
	"context"
	"encoding/json"
	"log"
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
	db  *mongo.Database
	cfg config.Config

	email *email.Client

	// Both login and reset are throttled by IP *and* by account, since an
	// IP-only limit doesn't stop a distributed attempt against one target
	// account from many source addresses, and an account-only limit doesn't
	// stop one address hammering many accounts.
	loginIPLimiter      *auth.RateLimiter
	loginAccountLimiter *auth.RateLimiter
	resetIPLimiter      *auth.RateLimiter
	resetAccountLimiter *auth.RateLimiter
}

func NewAuthHandler(db *mongo.Database, cfg config.Config, emailClient *email.Client) *AuthHandler {
	return &AuthHandler{
		db:                  db,
		cfg:                 cfg,
		email:               emailClient,
		loginIPLimiter:      auth.NewRateLimiter(5, 15*time.Minute),
		loginAccountLimiter: auth.NewRateLimiter(5, 15*time.Minute),
		resetIPLimiter:      auth.NewRateLimiter(3, 15*time.Minute),
		resetAccountLimiter: auth.NewRateLimiter(3, 15*time.Minute),
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

	// Sent in the background so a slow (or, on a cold Render instance, very
	// slow) Resend call never blocks the signup response - the account
	// already exists and is real the moment InsertOne succeeds above. A
	// fresh context is required since r.Context() is canceled the instant
	// this handler returns, same pattern as DocumentHandler's background
	// processing goroutine. A failure here is logged, not compensated: there's
	// no resend-verification endpoint yet, so deleting the account would just
	// trade "stuck unverified" for "signup silently vanished" - logging keeps
	// the (rare) failure visible without erasing a real account.
	verifyURL := h.cfg.FrontendURL + "/verify-email?token=" + rawToken
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := h.email.SendVerificationEmail(ctx, user.Email, user.Name, verifyURL); err != nil {
			log.Printf("signup: failed to send verification email to %s: %v", user.Email, err)
		}
	}()

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

	email := strings.ToLower(strings.TrimSpace(req.Email))
	if !h.resetIPLimiter.Allow(clientIP(r)) || !h.resetAccountLimiter.Allow(email) {
		writeError(w, http.StatusTooManyRequests, "too many requests, try again later")
		return
	}

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
		// Revokes every session issued before this reset - a password reset
		// is often prompted by a suspected compromise, so any token an
		// attacker already holds should stop working immediately too, not
		// just future logins requiring the new password.
		"$inc": bson.M{"token_version": 1},
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

	email := strings.ToLower(strings.TrimSpace(req.Email))
	if !h.loginIPLimiter.Allow(clientIP(r)) || !h.loginAccountLimiter.Allow(email) {
		writeError(w, http.StatusTooManyRequests, "too many login attempts, try again later")
		return
	}

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
		h.resendVerificationEmail(user)
		writeError(w, http.StatusForbidden,
			"please verify your email before logging in - we've sent a new verification link, "+
				"check your inbox (and your spam/junk folder if you don't see it soon)")
		return
	}

	token, err := auth.MintToken(user.ID.Hex(), user.TokenVersion, h.cfg.JWTSecret, h.cfg.JWTTTL)
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

// Logout bumps the user's TokenVersion (best-effort, not behind RequireAuth
// since a logout call should still succeed and clear the cookie even with
// an already-expired/invalid token) so the cookie being cleared here is
// immediately revoked everywhere it's still held - a stolen copy of this
// same cookie stops working the instant the real user logs out, rather than
// remaining valid for the rest of its natural multi-day expiry.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(h.cfg.JWTCookieName); err == nil {
		if claims, err := auth.ParseToken(cookie.Value, h.cfg.JWTSecret); err == nil {
			if userID, err := bson.ObjectIDFromHex(claims.UserID); err == nil {
				h.users().UpdateOne(r.Context(), bson.M{"_id": userID}, bson.M{
					"$inc": bson.M{"token_version": 1},
				})
			}
		}
	}

	http.SetCookie(w, &http.Cookie{
		Name:     h.cfg.JWTCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cfg.Environment == "production",
		SameSite: auth.SameSiteFor(h.cfg.Environment),
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
	auth.SetSessionCookie(w, h.cfg.JWTCookieName, token, h.cfg.Environment, h.cfg.JWTTTL)
}

// resendVerificationEmail issues a fresh verification token and emails it,
// exactly like Signup does - called when an unverified user tries to log in,
// since their original link may be long expired, lost, or never arrived, and
// there was previously no way for them to get a new one short of signing up
// again. Best-effort: a failure here still leaves the login attempt correctly
// rejected (unverified accounts can't log in either way), just without a
// fresh email actually landing - logged, not surfaced to the response, same
// as Signup's own background send.
func (h *AuthHandler) resendVerificationEmail(user models.User) {
	rawToken, tokenHash, err := auth.GenerateVerificationToken()
	if err != nil {
		log.Printf("login: failed to generate verification token for %s: %v", user.Email, err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = h.users().UpdateOne(ctx, bson.M{"_id": user.ID}, bson.M{
		"$set": bson.M{
			"verification_token_hash":       tokenHash,
			"verification_token_expires_at": time.Now().Add(auth.VerificationTokenTTL),
		},
	})
	if err != nil {
		log.Printf("login: failed to store resent verification token for %s: %v", user.Email, err)
		return
	}

	verifyURL := h.cfg.FrontendURL + "/verify-email?token=" + rawToken
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := h.email.SendVerificationEmail(ctx, user.Email, user.Name, verifyURL); err != nil {
			log.Printf("login: failed to resend verification email to %s: %v", user.Email, err)
		}
	}()
}

// clientIP extracts the real originating client address. Render sits behind
// Cloudflare, which terminates the actual client connection - r.RemoteAddr
// on every request the app ever sees is Render's own internal proxy address
// (observed in production logs as "[::1]:port" for every single request,
// regardless of who the real caller was), so using it directly as a rate
// limit key would make the limiter effectively global across all users
// instead of per-attacker. Cloudflare's True-Client-IP/CF-Connecting-IP
// headers carry the real address; X-Forwarded-For is the standard fallback.
// Locally (no Cloudflare in front) none of these headers are present, so it
// falls through to RemoteAddr exactly as before.
func clientIP(r *http.Request) string {
	if ip := r.Header.Get("True-Client-IP"); ip != "" {
		return ip
	}
	if ip := r.Header.Get("CF-Connecting-IP"); ip != "" {
		return ip
	}
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if idx := strings.Index(xff, ","); idx != -1 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}
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

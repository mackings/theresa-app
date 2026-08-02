package auth

import (
	"net/http"
	"time"
)

// SessionCookieConfig bundles what RequireAuth needs to reissue a session
// cookie mid-request (sliding expiration) - the same three values
// AuthHandler already uses to set the cookie at login.
type SessionCookieConfig struct {
	Name        string
	TTL         time.Duration
	Environment string
}

// SetSessionCookie writes the session cookie. Shared between AuthHandler
// (login) and RequireAuth (sliding-expiration refresh) so both always agree
// on Secure/SameSite/Path - having two copies drift would be an easy way to
// silently reintroduce the cross-site cookie bugs already fixed once.
func SetSessionCookie(w http.ResponseWriter, name, token, environment string, ttl time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   environment == "production",
		SameSite: SameSiteFor(environment),
		MaxAge:   int(ttl.Seconds()),
	})
}

// SameSiteFor picks the cookie's SameSite mode based on environment. In
// production the frontend and backend are two separate Render services on
// different *.onrender.com hostnames - onrender.com is itself registered on
// the public suffix list specifically so different customers' subdomains
// aren't treated as "same site," which means SameSite=Lax would silently
// never send this cookie cross-service. SameSite=None (paired with the
// Secure flag, already tied to the same production check) is required for
// a cross-site cookie to be sent at all. Locally both run on "localhost"
// (different ports only), which is already same-site, so Lax is fine there.
func SameSiteFor(environment string) http.SameSite {
	if environment == "production" {
		return http.SameSiteNoneMode
	}
	return http.SameSiteLaxMode
}

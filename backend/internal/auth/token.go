package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"time"
)

const VerificationTokenTTL = 24 * time.Hour
const ResetTokenTTL = 1 * time.Hour

// GenerateVerificationToken returns a random raw token (to embed in the
// emailed link) and the SHA-256 hex hash of it (to store in the DB), so a
// database read alone never yields a directly usable token.
func GenerateVerificationToken() (raw string, hash string, err error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", "", err
	}
	raw = hex.EncodeToString(buf)
	return raw, HashToken(raw), nil
}

func HashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

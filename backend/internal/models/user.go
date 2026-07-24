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
}

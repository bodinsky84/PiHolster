package auth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"
)

// Session mirrors the sessions row returned from the store.
type Session struct {
	Token     string
	UserID    int64
	ExpiresAt time.Time
}

// GenerateToken produces a 32-byte cryptographically random token encoded as
// base64url (no padding) suitable for use as a session cookie value.
func GenerateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("auth: generate token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

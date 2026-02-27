package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"
)

// RefreshToken is stored in DB (see migrations). The raw token is never stored.
// Users are stored in the historical `students` table.
type RefreshToken struct {
	ID                int64      `gorm:"primaryKey" json:"id"`
	UserID            int64      `gorm:"not null;index" json:"user_id"`
	TokenHash         string     `gorm:"type:text;not null;uniqueIndex" json:"-"`
	CreatedAt         time.Time  `json:"created_at"`
	ExpiresAt         time.Time  `gorm:"not null;index" json:"expires_at"`
	RevokedAt         *time.Time `gorm:"index" json:"revoked_at"`
	ReplacedByTokenID *int64     `gorm:"" json:"replaced_by_token_id"`
}

func (RefreshToken) TableName() string { return "refresh_tokens" }

func GenerateRefreshToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	// URL-safe, no padding.
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func HashRefreshToken(raw string) (string, error) {
	if raw == "" {
		return "", fmt.Errorf("refresh_token required")
	}
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:]), nil
}

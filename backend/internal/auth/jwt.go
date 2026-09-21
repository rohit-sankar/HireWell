// Package auth implements JWT issuance/verification and the Google OAuth2
// flow. It knows nothing about HTTP or the database -- controllers/services
// call into this package, keeping token mechanics in one auditable place.
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/resume-platform/backend/internal/models"
)

var (
	ErrInvalidToken = errors.New("invalid or expired token")
)

type Claims struct {
	UserID int64       `json:"user_id"`
	Role   models.Role `json:"role"`
	Email  string      `json:"email"`
	jwt.RegisteredClaims
}

type TokenManager struct {
	accessSecret  []byte
	refreshSecret []byte
	accessExpiry  time.Duration
	refreshExpiry time.Duration
}

func NewTokenManager(accessSecret, refreshSecret string, accessExpiry, refreshExpiry time.Duration) *TokenManager {
	return &TokenManager{
		accessSecret:  []byte(accessSecret),
		refreshSecret: []byte(refreshSecret),
		accessExpiry:  accessExpiry,
		refreshExpiry: refreshExpiry,
	}
}

// GenerateAccessToken issues a short-lived JWT carrying user id, role, and
// email. Role-based authorization middleware reads Claims.Role directly
// from this token -- it is never re-derived from a mutable request field.
func (tm *TokenManager) GenerateAccessToken(userID int64, role models.Role, email string) (string, time.Time, error) {
	expiresAt := time.Now().Add(tm.accessExpiry)
	claims := Claims{
		UserID: userID,
		Role:   role,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   email,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(tm.accessSecret)
	return signed, expiresAt, err
}

func (tm *TokenManager) ParseAccessToken(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return tm.accessSecret, nil
	})
	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

// GenerateRefreshToken returns a high-entropy opaque random string (NOT a
// JWT) plus its SHA-256 hash. Only the hash is stored in the database; the
// raw token is sent to the client once and can never be recovered from the
// DB, matching standard refresh-token-rotation hygiene.
func (tm *TokenManager) GenerateRefreshToken() (raw string, hash string, expiresAt time.Time, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return
	}
	raw = hex.EncodeToString(b)
	hash = HashToken(raw)
	expiresAt = time.Now().Add(tm.refreshExpiry)
	return
}

func HashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

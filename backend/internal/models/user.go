package models

import "time"

type Role string

const (
	RoleCandidate Role = "candidate"
	RoleCompany   Role = "company"
	RoleAdmin     Role = "admin"
)

// User is the single authentication identity for every role. Candidate- and
// company-specific profile data lives in their own tables (Candidate /
// Company), joined 1:1 via user_id.
type User struct {
	ID            int64     `json:"id" db:"id"`
	Email         string    `json:"email" db:"email"`
	PasswordHash  *string   `json:"-" db:"password_hash"` // never serialized to JSON
	Role          Role      `json:"role" db:"role"`
	GoogleID      *string   `json:"-" db:"google_id"`
	IsActive      bool      `json:"is_active" db:"is_active"`
	EmailVerified bool      `json:"email_verified" db:"email_verified"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

type RefreshToken struct {
	ID        int64     `json:"id" db:"id"`
	UserID    int64     `json:"user_id" db:"user_id"`
	TokenHash string    `json:"-" db:"token_hash"`
	ExpiresAt time.Time `json:"expires_at" db:"expires_at"`
	Revoked   bool      `json:"revoked" db:"revoked"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

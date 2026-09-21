package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/resume-platform/backend/internal/models"
)

var ErrNotFound = errors.New("record not found")

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, u *models.User) (int64, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO users (email, password_hash, role, google_id, is_active, email_verified)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		u.Email, u.PasswordHash, u.Role, u.GoogleID, u.IsActive, u.EmailVerified,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, email, password_hash, role, google_id, is_active, email_verified, created_at, updated_at
		 FROM users WHERE email = ?`, email)
	return scanUser(row)
}

func (r *UserRepository) FindByID(ctx context.Context, id int64) (*models.User, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, email, password_hash, role, google_id, is_active, email_verified, created_at, updated_at
		 FROM users WHERE id = ?`, id)
	return scanUser(row)
}

func (r *UserRepository) FindByGoogleID(ctx context.Context, googleID string) (*models.User, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, email, password_hash, role, google_id, is_active, email_verified, created_at, updated_at
		 FROM users WHERE google_id = ?`, googleID)
	return scanUser(row)
}

func (r *UserRepository) LinkGoogleID(ctx context.Context, userID int64, googleID string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE users SET google_id = ? WHERE id = ?`, googleID, userID)
	return err
}

func scanUser(row *sql.Row) (*models.User, error) {
	var u models.User
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.GoogleID, &u.IsActive, &u.EmailVerified, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// --- Refresh tokens ---

type RefreshTokenRepository struct {
	db *sql.DB
}

func NewRefreshTokenRepository(db *sql.DB) *RefreshTokenRepository {
	return &RefreshTokenRepository{db: db}
}

func (r *RefreshTokenRepository) Create(ctx context.Context, t *models.RefreshToken) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO refresh_tokens (user_id, token_hash, expires_at, revoked) VALUES (?, ?, ?, ?)`,
		t.UserID, t.TokenHash, t.ExpiresAt, t.Revoked,
	)
	return err
}

func (r *RefreshTokenRepository) FindByHash(ctx context.Context, hash string) (*models.RefreshToken, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, token_hash, expires_at, revoked, created_at FROM refresh_tokens WHERE token_hash = ?`, hash)
	var t models.RefreshToken
	err := row.Scan(&t.ID, &t.UserID, &t.TokenHash, &t.ExpiresAt, &t.Revoked, &t.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *RefreshTokenRepository) Revoke(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `UPDATE refresh_tokens SET revoked = TRUE WHERE id = ?`, id)
	return err
}

func (r *RefreshTokenRepository) RevokeAllForUser(ctx context.Context, userID int64) error {
	_, err := r.db.ExecContext(ctx, `UPDATE refresh_tokens SET revoked = TRUE WHERE user_id = ?`, userID)
	return err
}

package services

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/resume-platform/backend/internal/auth"
	"github.com/resume-platform/backend/internal/models"
	"github.com/resume-platform/backend/internal/repository"
	"github.com/resume-platform/backend/internal/utils"
)

type AuthService struct {
	users         *repository.UserRepository
	refreshTokens *repository.RefreshTokenRepository
	candidates    *repository.CandidateRepository
	companies     *repository.CompanyRepository
	tokens        *auth.TokenManager
}

func NewAuthService(users *repository.UserRepository, refreshTokens *repository.RefreshTokenRepository, candidates *repository.CandidateRepository, companies *repository.CompanyRepository, tm *auth.TokenManager) *AuthService {
	return &AuthService{users: users, refreshTokens: refreshTokens, candidates: candidates, companies: companies, tokens: tm}
}

type RegisterCandidateInput struct {
	Email    string
	Password string
	FullName string
	Phone    string
}

type RegisterCompanyInput struct {
	Email       string
	Password    string
	CompanyName string
	Website     string
}

type AuthResult struct {
	User         *models.User
	AccessToken  string
	AccessExpiry time.Time
	RefreshToken string
}

func (s *AuthService) RegisterCandidate(ctx context.Context, in RegisterCandidateInput) (*AuthResult, error) {
	email := normalizeEmail(in.Email)
	if _, err := s.users.FindByEmail(ctx, email); err == nil {
		return nil, utils.NewConflict("an account with this email already exists")
	} else if !errors.Is(err, repository.ErrNotFound) {
		return nil, utils.NewDatabaseError(err)
	}

	hash, err := utils.HashPassword(in.Password)
	if err != nil {
		return nil, utils.NewInternalError(err)
	}

	user := &models.User{Email: email, PasswordHash: &hash, Role: models.RoleCandidate, IsActive: true}
	userID, err := s.users.Create(ctx, user)
	if err != nil {
		return nil, utils.NewDatabaseError(err)
	}
	user.ID = userID

	var phonePtr *string
	if in.Phone != "" {
		phonePtr = &in.Phone
	}
	_, err = s.candidates.Create(ctx, &models.Candidate{
		UserID: userID, FullName: in.FullName, Phone: phonePtr, Links: models.JSONMap{},
	})
	if err != nil {
		return nil, utils.NewDatabaseError(err)
	}

	hydrated, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return nil, utils.NewDatabaseError(err)
	}
	return s.issueTokens(ctx, hydrated)
}

func (s *AuthService) RegisterCompany(ctx context.Context, in RegisterCompanyInput) (*AuthResult, error) {
	email := normalizeEmail(in.Email)
	if _, err := s.users.FindByEmail(ctx, email); err == nil {
		return nil, utils.NewConflict("an account with this email already exists")
	} else if !errors.Is(err, repository.ErrNotFound) {
		return nil, utils.NewDatabaseError(err)
	}

	hash, err := utils.HashPassword(in.Password)
	if err != nil {
		return nil, utils.NewInternalError(err)
	}

	user := &models.User{Email: email, PasswordHash: &hash, Role: models.RoleCompany, IsActive: true}
	userID, err := s.users.Create(ctx, user)
	if err != nil {
		return nil, utils.NewDatabaseError(err)
	}
	user.ID = userID

	var websitePtr *string
	if in.Website != "" {
		websitePtr = &in.Website
	}
	_, err = s.companies.Create(ctx, &models.Company{
		UserID: userID, CompanyName: in.CompanyName, Website: websitePtr,
	})
	if err != nil {
		return nil, utils.NewDatabaseError(err)
	}

	hydrated, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return nil, utils.NewDatabaseError(err)
	}
	return s.issueTokens(ctx, hydrated)
}

func (s *AuthService) Login(ctx context.Context, email, password string) (*AuthResult, error) {
	user, err := s.users.FindByEmail(ctx, normalizeEmail(email))
	if errors.Is(err, repository.ErrNotFound) {
		return nil, utils.NewUnauthorized("invalid email or password")
	}
	if err != nil {
		return nil, utils.NewDatabaseError(err)
	}
	if user.PasswordHash == nil {
		return nil, utils.NewUnauthorized("this account uses Google sign-in; please continue with Google")
	}
	if !utils.CheckPassword(*user.PasswordHash, password) {
		return nil, utils.NewUnauthorized("invalid email or password")
	}
	if !user.IsActive {
		return nil, utils.NewForbidden("this account has been deactivated")
	}
	return s.issueTokens(ctx, user)
}

// LoginOrRegisterWithGoogle implements the Google OAuth flow's account
// linking logic: if a user with this Google ID exists, log them in; else if
// a user with this verified email exists (they registered with
// password first), link the Google ID to that account; else create a new
// candidate account (Google sign-up defaults to the candidate role --
// companies must register explicitly with company details).
func (s *AuthService) LoginOrRegisterWithGoogle(ctx context.Context, googleID, email, fullName string) (*AuthResult, error) {
	email = normalizeEmail(email)

	user, err := s.users.FindByGoogleID(ctx, googleID)
	if err == nil {
		return s.issueTokens(ctx, user)
	}
	if !errors.Is(err, repository.ErrNotFound) {
		return nil, utils.NewDatabaseError(err)
	}

	user, err = s.users.FindByEmail(ctx, email)
	if err == nil {
		if err := s.users.LinkGoogleID(ctx, user.ID, googleID); err != nil {
			return nil, utils.NewDatabaseError(err)
		}
		return s.issueTokens(ctx, user)
	}
	if !errors.Is(err, repository.ErrNotFound) {
		return nil, utils.NewDatabaseError(err)
	}

	newUser := &models.User{Email: email, Role: models.RoleCandidate, GoogleID: &googleID, EmailVerified: true, IsActive: true}
	userID, err := s.users.Create(ctx, newUser)
	if err != nil {
		return nil, utils.NewDatabaseError(err)
	}
	newUser.ID = userID

	if fullName == "" {
		fullName = strings.Split(email, "@")[0]
	}
	if _, err := s.candidates.Create(ctx, &models.Candidate{UserID: userID, FullName: fullName, Links: models.JSONMap{}}); err != nil {
		return nil, utils.NewDatabaseError(err)
	}

	hydrated, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return nil, utils.NewDatabaseError(err)
	}
	return s.issueTokens(ctx, hydrated)
}

func (s *AuthService) RefreshTokens(ctx context.Context, rawRefreshToken string) (*AuthResult, error) {
	hash := auth.HashToken(rawRefreshToken)
	stored, err := s.refreshTokens.FindByHash(ctx, hash)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, utils.NewUnauthorized("invalid refresh token")
	}
	if err != nil {
		return nil, utils.NewDatabaseError(err)
	}
	if stored.Revoked || time.Now().After(stored.ExpiresAt) {
		return nil, utils.NewUnauthorized("refresh token has expired or been revoked")
	}

	user, err := s.users.FindByID(ctx, stored.UserID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, utils.NewUnauthorized("account no longer exists")
	}
	if err != nil {
		return nil, utils.NewDatabaseError(err)
	}

	// rotate: revoke the used refresh token and issue a brand new pair,
	// preventing replay of an old refresh token.
	if err := s.refreshTokens.Revoke(ctx, stored.ID); err != nil {
		return nil, utils.NewDatabaseError(err)
	}
	return s.issueTokens(ctx, user)
}

func (s *AuthService) Logout(ctx context.Context, rawRefreshToken string) error {
	if rawRefreshToken == "" {
		return nil
	}
	hash := auth.HashToken(rawRefreshToken)
	stored, err := s.refreshTokens.FindByHash(ctx, hash)
	if errors.Is(err, repository.ErrNotFound) {
		return nil // already gone/invalid -- logout is idempotent, not an error
	}
	if err != nil {
		return utils.NewDatabaseError(err)
	}
	if err := s.refreshTokens.Revoke(ctx, stored.ID); err != nil {
		return utils.NewDatabaseError(err)
	}
	return nil
}

func (s *AuthService) Me(ctx context.Context, userID int64) (*models.User, error) {
	user, err := s.users.FindByID(ctx, userID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, utils.NewNotFound("user")
	}
	if err != nil {
		return nil, utils.NewDatabaseError(err)
	}
	return user, nil
}

func (s *AuthService) issueTokens(ctx context.Context, user *models.User) (*AuthResult, error) {
	access, expiry, err := s.tokens.GenerateAccessToken(user.ID, user.Role, user.Email)
	if err != nil {
		return nil, utils.NewInternalError(err)
	}
	rawRefresh, hash, refreshExpiry, err := s.tokens.GenerateRefreshToken()
	if err != nil {
		return nil, utils.NewInternalError(err)
	}
	if err := s.refreshTokens.Create(ctx, &models.RefreshToken{UserID: user.ID, TokenHash: hash, ExpiresAt: refreshExpiry}); err != nil {
		return nil, utils.NewDatabaseError(err)
	}

	return &AuthResult{User: user, AccessToken: access, AccessExpiry: expiry, RefreshToken: rawRefresh}, nil
}

func normalizeEmail(e string) string {
	return strings.ToLower(strings.TrimSpace(e))
}

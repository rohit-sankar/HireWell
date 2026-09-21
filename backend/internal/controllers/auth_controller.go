package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/resume-platform/backend/internal/auth"
	"github.com/resume-platform/backend/internal/config"
	"github.com/resume-platform/backend/internal/middleware"
	"github.com/resume-platform/backend/internal/services"
	"github.com/resume-platform/backend/internal/utils"
	"github.com/resume-platform/backend/internal/validators"
	"golang.org/x/oauth2"
)

type AuthController struct {
	auth         *services.AuthService
	googleConfig *oauth2.Config
	frontendURL  string
}

func NewAuthController(authSvc *services.AuthService, cfg *config.Config) *AuthController {
	return &AuthController{
		auth:         authSvc,
		googleConfig: auth.NewGoogleOAuthConfig(cfg.GoogleClientID, cfg.GoogleClientSecret, cfg.GoogleRedirectURL),
		frontendURL:  cfg.FrontendURL,
	}
}

// refreshCookieName / setRefreshCookie: the refresh token is delivered via
// an httpOnly, SameSite=Lax cookie rather than the JSON body, so it is
// inaccessible to JavaScript (XSS-resistant) while the short-lived access
// token is returned in the body for the client to hold in memory.
const refreshCookieName = "refresh_token"

func setRefreshCookie(c *gin.Context, token string, maxAgeSeconds int) {
	c.SetCookie(refreshCookieName, token, maxAgeSeconds, "/api/v1/auth", "", false, true)
}

func (ac *AuthController) RegisterCandidate(c *gin.Context) {
	var req validators.RegisterCandidateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(utils.NewValidationError("invalid request body", err.Error()))
		return
	}
	result, err := ac.auth.RegisterCandidate(c.Request.Context(), services.RegisterCandidateInput{
		Email: req.Email, Password: req.Password, FullName: req.FullName, Phone: req.Phone,
	})
	if err != nil {
		c.Error(err)
		return
	}
	respondWithAuthResult(c, http.StatusCreated, result)
}

func (ac *AuthController) RegisterCompany(c *gin.Context) {
	var req validators.RegisterCompanyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(utils.NewValidationError("invalid request body", err.Error()))
		return
	}
	result, err := ac.auth.RegisterCompany(c.Request.Context(), services.RegisterCompanyInput{
		Email: req.Email, Password: req.Password, CompanyName: req.CompanyName, Website: req.Website,
	})
	if err != nil {
		c.Error(err)
		return
	}
	respondWithAuthResult(c, http.StatusCreated, result)
}

func (ac *AuthController) Login(c *gin.Context) {
	var req validators.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(utils.NewValidationError("invalid request body", err.Error()))
		return
	}
	result, err := ac.auth.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		c.Error(err)
		return
	}
	respondWithAuthResult(c, http.StatusOK, result)
}

func (ac *AuthController) Refresh(c *gin.Context) {
	rawToken, err := c.Cookie(refreshCookieName)
	if err != nil || rawToken == "" {
		// fall back to body for API clients that can't use cookies (e.g. mobile)
		var req validators.RefreshRequest
		if bindErr := c.ShouldBindJSON(&req); bindErr == nil {
			rawToken = req.RefreshToken
		}
	}
	if rawToken == "" {
		c.Error(utils.NewUnauthorized("no refresh token provided"))
		return
	}

	result, err := ac.auth.RefreshTokens(c.Request.Context(), rawToken)
	if err != nil {
		c.Error(err)
		return
	}
	respondWithAuthResult(c, http.StatusOK, result)
}

func (ac *AuthController) Logout(c *gin.Context) {
	rawToken, _ := c.Cookie(refreshCookieName)
	_ = ac.auth.Logout(c.Request.Context(), rawToken)
	setRefreshCookie(c, "", -1)
	utils.Success(c, http.StatusOK, nil, "logged out")
}

func (ac *AuthController) Me(c *gin.Context) {
	userID, ok := middleware.UserIDFromContext(c)
	if !ok {
		middleware.AbortUnauthenticated(c)
		return
	}
	user, err := ac.auth.Me(c.Request.Context(), userID)
	if err != nil {
		c.Error(err)
		return
	}
	utils.Success(c, http.StatusOK, user, "")
}

func (ac *AuthController) GoogleLogin(c *gin.Context) {
	if ac.googleConfig == nil {
		c.Error(utils.NewBadRequest("Google OAuth is not configured on this server"))
		return
	}
	url := ac.googleConfig.AuthCodeURL("state", oauth2.AccessTypeOffline)
	c.Redirect(http.StatusTemporaryRedirect, url)
}

func (ac *AuthController) GoogleCallback(c *gin.Context) {
	if ac.googleConfig == nil {
		c.Error(utils.NewBadRequest("Google OAuth is not configured on this server"))
		return
	}
	var req validators.GoogleCallbackRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.Error(utils.NewValidationError("missing authorization code", nil))
		return
	}

	info, err := auth.ExchangeGoogleCode(c.Request.Context(), ac.googleConfig, req.Code)
	if err != nil {
		c.Error(utils.NewUnauthorized("google authentication failed: " + err.Error()))
		return
	}

	result, err := ac.auth.LoginOrRegisterWithGoogle(c.Request.Context(), info.Sub, info.Email, info.Name)
	if err != nil {
		c.Error(err)
		return
	}

	setRefreshCookie(c, result.RefreshToken, 7*24*3600)
	// Redirect back to the frontend with the access token as a URL fragment
	// (not a query param, so it never lands in server logs/referrer
	// headers); the frontend reads it once and stores it in memory.
	c.Redirect(http.StatusTemporaryRedirect, ac.frontendURL+"/auth/callback#access_token="+result.AccessToken)
}

func respondWithAuthResult(c *gin.Context, status int, result *services.AuthResult) {
	setRefreshCookie(c, result.RefreshToken, 7*24*3600)
	utils.Success(c, status, gin.H{
		"user":         result.User,
		"access_token": result.AccessToken,
		"expires_at":   result.AccessExpiry,
	}, "")
}

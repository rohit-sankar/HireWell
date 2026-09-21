package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/resume-platform/backend/internal/auth"
	"github.com/resume-platform/backend/internal/models"
	"github.com/resume-platform/backend/internal/utils"
)

const (
	CtxUserID = "auth_user_id"
	CtxRole   = "auth_role"
	CtxEmail  = "auth_email"
)

// RequireAuth validates the Bearer access token and populates the request
// context with the authenticated user's id/role/email. Every protected
// route must use this before any RequireRole check.
func RequireAuth(tm *auth.TokenManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" || !strings.HasPrefix(header, "Bearer ") {
			c.Error(utils.NewUnauthorized("missing or malformed Authorization header"))
			c.Abort()
			return
		}
		tokenString := strings.TrimPrefix(header, "Bearer ")

		claims, err := tm.ParseAccessToken(tokenString)
		if err != nil {
			c.Error(utils.NewUnauthorized("invalid or expired access token"))
			c.Abort()
			return
		}

		c.Set(CtxUserID, claims.UserID)
		c.Set(CtxRole, claims.Role)
		c.Set(CtxEmail, claims.Email)
		c.Next()
	}
}

// OptionalAuth populates the user context when a valid Bearer token is
// supplied, while still allowing anonymous requests through. Public job
// browsing uses this so the same endpoint can safely support mine=true for
// authenticated company users.
func OptionalAuth(tm *auth.TokenManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header != "" && strings.HasPrefix(header, "Bearer ") {
			tokenString := strings.TrimPrefix(header, "Bearer ")
			if claims, err := tm.ParseAccessToken(tokenString); err == nil {
				c.Set(CtxUserID, claims.UserID)
				c.Set(CtxRole, claims.Role)
				c.Set(CtxEmail, claims.Email)
			}
		}
		c.Next()
	}
}

// RequireRole enforces role-based access control. A candidate must never
// reach an HR/company-only handler and vice versa; admin routes are
// similarly gated. This must run after RequireAuth.
func RequireRole(roles ...models.Role) gin.HandlerFunc {
	allowed := make(map[models.Role]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}
	return func(c *gin.Context) {
		roleVal, exists := c.Get(CtxRole)
		if !exists {
			c.Error(utils.NewUnauthorized(""))
			c.Abort()
			return
		}
		role, ok := roleVal.(models.Role)
		if !ok || !allowed[role] {
			c.Error(utils.NewForbidden("this action is not permitted for your account type"))
			c.Abort()
			return
		}
		c.Next()
	}
}

// UserIDFromContext is a small helper controllers use to read the
// authenticated user id after RequireAuth has run.
func UserIDFromContext(c *gin.Context) (int64, bool) {
	v, exists := c.Get(CtxUserID)
	if !exists {
		return 0, false
	}
	id, ok := v.(int64)
	return id, ok
}

func RoleFromContext(c *gin.Context) (models.Role, bool) {
	v, exists := c.Get(CtxRole)
	if !exists {
		return "", false
	}
	role, ok := v.(models.Role)
	return role, ok
}

// AbortUnauthenticated is a convenience for handlers that need to bail out
// early with a consistent 401 when context population unexpectedly fails.
func AbortUnauthenticated(c *gin.Context) {
	utils.Error(c, http.StatusUnauthorized, utils.CodeUnauthorized, "authentication required", nil)
	c.Abort()
}

package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/resume-platform/backend/internal/auth"
	"github.com/resume-platform/backend/internal/models"
	"github.com/rs/zerolog"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func testRouter(tm *auth.TokenManager, roles ...models.Role) *gin.Engine {
	r := gin.New()
	r.Use(ErrorHandler(zerolog.Nop())) // must wrap RequireAuth so c.Error() calls become real responses

	group := r.Group("/protected", RequireAuth(tm))
	if len(roles) > 0 {
		group.Use(RequireRole(roles...))
	}
	group.GET("", func(c *gin.Context) {
		userID, _ := UserIDFromContext(c)
		role, _ := RoleFromContext(c)
		c.JSON(http.StatusOK, gin.H{"user_id": userID, "role": role})
	})
	return r
}

func doRequest(r *gin.Engine, authHeader string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestRequireAuth_RejectsMissingHeader(t *testing.T) {
	tm := auth.NewTokenManager("a", "b", 15*time.Minute, time.Hour)
	r := testRouter(tm)

	w := doRequest(r, "")
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for missing Authorization header, got %d", w.Code)
	}
}

func TestRequireAuth_RejectsMalformedHeader(t *testing.T) {
	tm := auth.NewTokenManager("a", "b", 15*time.Minute, time.Hour)
	r := testRouter(tm)

	w := doRequest(r, "NotBearer sometoken")
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for a non-Bearer Authorization header, got %d", w.Code)
	}
}

func TestRequireAuth_RejectsInvalidToken(t *testing.T) {
	tm := auth.NewTokenManager("a", "b", 15*time.Minute, time.Hour)
	r := testRouter(tm)

	w := doRequest(r, "Bearer this.is.not.a.valid.jwt")
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for an invalid token, got %d", w.Code)
	}
}

func TestRequireAuth_RejectsTokenSignedByDifferentSecret(t *testing.T) {
	tmServer := auth.NewTokenManager("server-secret", "refresh", 15*time.Minute, time.Hour)
	tmAttacker := auth.NewTokenManager("attacker-secret", "refresh", 15*time.Minute, time.Hour)

	forgedToken, _, _ := tmAttacker.GenerateAccessToken(1, models.RoleAdmin, "attacker@example.com")

	r := testRouter(tmServer)
	w := doRequest(r, "Bearer "+forgedToken)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("a token forged with a different secret must be rejected, got status %d", w.Code)
	}
}

func TestRequireAuth_AcceptsValidToken(t *testing.T) {
	tm := auth.NewTokenManager("a", "b", 15*time.Minute, time.Hour)
	token, _, err := tm.GenerateAccessToken(7, models.RoleCandidate, "jane@example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	r := testRouter(tm)
	w := doRequest(r, "Bearer "+token)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for a valid token, got %d: %s", w.Code, w.Body.String())
	}

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if body["user_id"].(float64) != 7 {
		t.Errorf("expected user_id 7 in response, got %v", body["user_id"])
	}
}

func TestRequireRole_BlocksWrongRole(t *testing.T) {
	tm := auth.NewTokenManager("a", "b", 15*time.Minute, time.Hour)
	token, _, _ := tm.GenerateAccessToken(1, models.RoleCandidate, "jane@example.com")

	// route requires company role; candidate token must be rejected
	r := testRouter(tm, models.RoleCompany)
	w := doRequest(r, "Bearer "+token)
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 when a candidate token hits a company-only route, got %d", w.Code)
	}
}

func TestRequireRole_AllowsCorrectRole(t *testing.T) {
	tm := auth.NewTokenManager("a", "b", 15*time.Minute, time.Hour)
	token, _, _ := tm.GenerateAccessToken(1, models.RoleCompany, "hr@example.com")

	r := testRouter(tm, models.RoleCompany)
	w := doRequest(r, "Bearer "+token)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 when a company token hits a company-only route, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRequireRole_AllowsAnyOfMultipleRoles(t *testing.T) {
	tm := auth.NewTokenManager("a", "b", 15*time.Minute, time.Hour)
	candidateToken, _, _ := tm.GenerateAccessToken(1, models.RoleCandidate, "jane@example.com")
	adminToken, _, _ := tm.GenerateAccessToken(2, models.RoleAdmin, "admin@example.com")

	// route allows both candidate and admin
	r := testRouter(tm, models.RoleCandidate, models.RoleAdmin)

	if w := doRequest(r, "Bearer "+candidateToken); w.Code != http.StatusOK {
		t.Errorf("expected candidate to be allowed, got %d", w.Code)
	}
	if w := doRequest(r, "Bearer "+adminToken); w.Code != http.StatusOK {
		t.Errorf("expected admin to be allowed, got %d", w.Code)
	}

	companyToken, _, _ := tm.GenerateAccessToken(3, models.RoleCompany, "hr@example.com")
	if w := doRequest(r, "Bearer "+companyToken); w.Code != http.StatusForbidden {
		t.Errorf("expected company role to be forbidden on a candidate/admin-only route, got %d", w.Code)
	}
}

package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func corsRouter(allowedOrigin string) *gin.Engine {
	r := gin.New()
	r.Use(CORS(allowedOrigin))
	r.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })
	return r
}

func TestCORS_EchoesAllowedOriginWithCredentials(t *testing.T) {
	r := corsRouter("https://app.example.com")
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Origin", "https://app.example.com")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://app.example.com" {
		t.Errorf("expected Access-Control-Allow-Origin to echo the allowed origin, got %q", got)
	}
	if got := w.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Errorf("expected Access-Control-Allow-Credentials: true, got %q", got)
	}
}

func TestCORS_DoesNotEchoDisallowedOrigin(t *testing.T) {
	// This is the actual security property: CORS must NOT be a wildcard
	// reflection of any Origin header -- only the configured frontend
	// origin should ever be allowed, or the credentialed-cookie flow would
	// be exploitable from any attacker-controlled site.
	r := corsRouter("https://app.example.com")
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Origin", "https://evil-attacker.com")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("SECURITY: an untrusted origin must never be echoed back, got %q", got)
	}
	if got := w.Header().Get("Access-Control-Allow-Credentials"); got == "true" {
		t.Error("SECURITY: credentials must not be allowed for a disallowed origin")
	}
}

func TestCORS_HandlesPreflightOptionsRequest(t *testing.T) {
	r := corsRouter("https://app.example.com")
	req := httptest.NewRequest(http.MethodOptions, "/x", nil)
	req.Header.Set("Origin", "https://app.example.com")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204 No Content for an OPTIONS preflight, got %d", w.Code)
	}
}

func TestSecurityHeaders_SetsDefensiveHeaders(t *testing.T) {
	r := gin.New()
	r.Use(SecurityHeaders())
	r.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	checks := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
	}
	for header, expected := range checks {
		if got := w.Header().Get(header); got != expected {
			t.Errorf("expected %s: %s, got %q", header, expected, got)
		}
	}
}

func TestRateLimiter_AllowsUpToLimit(t *testing.T) {
	rl := NewRateLimiter(3)
	r := gin.New()
	r.Use(rl.Middleware())
	r.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })

	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		req.RemoteAddr = "10.0.0.1:1234"
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("request %d: expected 200 within limit, got %d", i+1, w.Code)
		}
	}
}

func TestRateLimiter_BlocksOverLimit(t *testing.T) {
	rl := NewRateLimiter(3)
	r := gin.New()
	r.Use(rl.Middleware())
	r.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })

	var lastCode int
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		req.RemoteAddr = "10.0.0.2:1234"
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		lastCode = w.Code
	}
	if lastCode != http.StatusTooManyRequests {
		t.Errorf("expected the 5th request (over a limit of 3) to be 429, got %d", lastCode)
	}
}

func TestRateLimiter_TracksClientsIndependently(t *testing.T) {
	rl := NewRateLimiter(1)
	r := gin.New()
	r.Use(rl.Middleware())
	r.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })

	// client A uses its one allowed request
	reqA := httptest.NewRequest(http.MethodGet, "/x", nil)
	reqA.RemoteAddr = "10.0.0.3:1111"
	wA := httptest.NewRecorder()
	r.ServeHTTP(wA, reqA)
	if wA.Code != http.StatusOK {
		t.Fatalf("client A's first request should succeed, got %d", wA.Code)
	}

	// client B, a completely different IP, must not be affected by client A's usage
	reqB := httptest.NewRequest(http.MethodGet, "/x", nil)
	reqB.RemoteAddr = "10.0.0.4:2222"
	wB := httptest.NewRecorder()
	r.ServeHTTP(wB, reqB)
	if wB.Code != http.StatusOK {
		t.Errorf("client B should not be rate-limited by client A's requests, got %d", wB.Code)
	}
}

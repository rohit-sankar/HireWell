package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/resume-platform/backend/internal/utils"
)

// CORS restricts cross-origin requests to the configured frontend origin.
// Credentials (cookies / Authorization header) are allowed because the
// refresh-token flow uses an httpOnly cookie.
func CORS(allowedOrigin string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" && origin == allowedOrigin {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
		}
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization, X-Request-ID")
		c.Header("Access-Control-Expose-Headers", "X-Request-ID")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// SecurityHeaders sets a conservative baseline of defensive HTTP headers.
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("X-XSS-Protection", "0") // deprecated header; explicit 0 avoids inconsistent legacy behavior
		c.Next()
	}
}

// RateLimiter is a minimal fixed-window, in-memory, per-IP limiter. It is
// intentionally simple (no Redis dependency) for a single-instance
// deployment; the interface is small enough to swap for a distributed
// limiter later without touching call sites.
type RateLimiter struct {
	mu       sync.Mutex
	requests map[string][]time.Time
	limit    int
	window   time.Duration
}

func NewRateLimiter(requestsPerMinute int) *RateLimiter {
	return &RateLimiter{
		requests: make(map[string][]time.Time),
		limit:    requestsPerMinute,
		window:   time.Minute,
	}
}

func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.ClientIP()
		now := time.Now()

		rl.mu.Lock()
		windowStart := now.Add(-rl.window)
		var recent []time.Time
		for _, t := range rl.requests[key] {
			if t.After(windowStart) {
				recent = append(recent, t)
			}
		}
		if len(recent) >= rl.limit {
			rl.mu.Unlock()
			utils.Error(c, http.StatusTooManyRequests, utils.CodeRateLimited, "too many requests, please slow down", nil)
			c.Abort()
			return
		}
		recent = append(recent, now)
		rl.requests[key] = recent
		rl.mu.Unlock()

		c.Next()
	}
}

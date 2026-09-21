package controllers

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/resume-platform/backend/internal/ai"
)

type HealthController struct {
	db         *sql.DB
	aiProvider ai.Provider
}

func NewHealthController(db *sql.DB, provider ai.Provider) *HealthController {
	return &HealthController{db: db, aiProvider: provider}
}

// Health is a liveness check: "is the process up". It never touches the
// database, so it can't report unhealthy just because a dependency is slow.
func (hc *HealthController) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// Ready is a readiness check: "can this instance actually serve traffic".
// It verifies the database connection is alive and reports which AI
// provider is configured (without leaking the API key).
func (hc *HealthController) Ready(c *gin.Context) {
	dbStatus := "ok"
	httpStatus := http.StatusOK
	if err := hc.db.PingContext(c.Request.Context()); err != nil {
		dbStatus = "unavailable"
		httpStatus = http.StatusServiceUnavailable
	}

	c.JSON(httpStatus, gin.H{
		"status":      map[bool]string{true: "ready", false: "not_ready"}[httpStatus == http.StatusOK],
		"database":    dbStatus,
		"ai_provider": hc.aiProvider.Name(),
	})
}

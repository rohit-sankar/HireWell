package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/resume-platform/backend/internal/utils"
	"github.com/rs/zerolog"
)

// ErrorHandler centralizes conversion of errors into the standard API
// response envelope. Controllers/services call c.Error(err) (or return an
// *utils.AppError which controllers pass to c.Error) instead of writing
// JSON themselves for error cases. This guarantees a single consistent
// shape and guarantees internal details never leak to the client.
func ErrorHandler(logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) == 0 {
			return
		}

		err := c.Errors.Last().Err
		reqID, _ := c.Get(RequestIDKey)

		if appErr, ok := err.(*utils.AppError); ok {
			logEvent := logger.Warn()
			if appErr.Status >= 500 {
				logEvent = logger.Error()
			}
			logEvent.
				Str("request_id", toString(reqID)).
				Str("code", appErr.Code).
				Err(appErr.Err).
				Msg(appErr.Message)

			utils.Error(c, appErr.Status, appErr.Code, appErr.Message, appErr.Details)
			return
		}

		// Unexpected/unclassified error: log full detail server-side, but
		// never leak internals (stack traces, driver errors, file paths) to
		// the client.
		logger.Error().
			Str("request_id", toString(reqID)).
			Err(err).
			Msg("unhandled error")
		utils.Error(c, http.StatusInternalServerError, utils.CodeInternal, "an unexpected error occurred", nil)
	}
}

// Recovery converts panics into 500 responses instead of crashing the
// server, logging the panic value server-side only.
func Recovery(logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				reqID, _ := c.Get(RequestIDKey)
				logger.Error().
					Str("request_id", toString(reqID)).
					Interface("panic", r).
					Msg("recovered from panic")
				utils.Error(c, http.StatusInternalServerError, utils.CodeInternal, "an unexpected error occurred", nil)
				c.Abort()
			}
		}()
		c.Next()
	}
}

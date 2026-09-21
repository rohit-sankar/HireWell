package utils

import (
	"os"
	"time"

	"github.com/rs/zerolog"
)

// NewLogger returns a zerolog.Logger configured for structured (JSON) output
// in production and pretty console output in development. Never log
// secrets, password hashes, JWTs, or raw resume contents through this
// logger -- see middleware/logging.go for what request fields are safe to
// log (method, path, status, latency, request id).
func NewLogger(env string) zerolog.Logger {
	zerolog.TimeFieldFormat = time.RFC3339

	var logger zerolog.Logger
	if env == "production" {
		logger = zerolog.New(os.Stdout).With().Timestamp().Logger()
	} else {
		logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: "15:04:05"}).With().Timestamp().Logger()
	}
	return logger
}

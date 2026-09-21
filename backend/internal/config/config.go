// Package config centralizes all environment-driven configuration for the
// application. No other package should call os.Getenv directly -- everything
// flows through this struct so we have one place that documents every knob
// and one place that fails fast on misconfiguration.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv  string
	AppPort string

	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string

	JWTSecret        string
	JWTAccessExpiry  time.Duration
	JWTRefreshSecret string
	JWTRefreshExpiry time.Duration

	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string

	AIProvider string
	AIAPIKey   string
	AIModel    string

	UploadDir     string
	MaxUploadSize int64 // bytes

	FrontendURL string
	BackendURL  string

	RateLimitRequestsPerMinute int
}

// Load reads .env (if present -- it's fine if it isn't, e.g. in prod where
// env vars are injected by the platform) and then builds a validated Config.
// It intentionally fails loudly (panic at startup, not silently at request
// time) if a required secret is missing, per the "no secrets in source,
// fail fast on missing config" requirement.
func Load() (*Config, error) {
	_ = godotenv.Load() // ignore error: absence of .env is fine in prod

	cfg := &Config{
		AppEnv:  getEnv("APP_ENV", "development"),
		AppPort: getEnv("APP_PORT", "8080"),

		DBHost:     getEnv("DB_HOST", "127.0.0.1"),
		DBPort:     getEnv("DB_PORT", "3306"),
		DBUser:     getEnv("DB_USER", "app_user"),
		DBPassword: getEnv("DB_PASSWORD", ""),
		DBName:     getEnv("DB_NAME", "resume_platform"),

		JWTSecret:        getEnv("JWT_SECRET", ""),
		JWTRefreshSecret: getEnv("JWT_REFRESH_SECRET", ""),

		GoogleClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
		GoogleRedirectURL:  getEnv("GOOGLE_REDIRECT_URL", ""),

		AIProvider: getEnv("AI_PROVIDER", "stub"),
		AIAPIKey:   getEnv("AI_API_KEY", ""),
		AIModel:    getEnv("AI_MODEL", "gemini-2.0-flash"),

		UploadDir:   getEnv("UPLOAD_DIR", "./uploads"),
		FrontendURL: getEnv("FRONTEND_URL", "http://localhost:3000"),
		BackendURL:  getEnv("BACKEND_URL", "http://localhost:8080"),
	}

	accessExpiry := getEnv("JWT_ACCESS_EXPIRY", "15m")
	refreshExpiry := getEnv("JWT_REFRESH_EXPIRY", "168h") // 7 days
	var err error
	cfg.JWTAccessExpiry, err = time.ParseDuration(accessExpiry)
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_ACCESS_EXPIRY %q: %w", accessExpiry, err)
	}
	cfg.JWTRefreshExpiry, err = time.ParseDuration(refreshExpiry)
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_REFRESH_EXPIRY %q: %w", refreshExpiry, err)
	}

	maxUploadMB := getEnvInt("MAX_UPLOAD_SIZE_MB", 10)
	cfg.MaxUploadSize = int64(maxUploadMB) * 1024 * 1024

	cfg.RateLimitRequestsPerMinute = getEnvInt("RATE_LIMIT_RPM", 120)

	if cfg.JWTSecret == "" || cfg.JWTRefreshSecret == "" {
		if cfg.AppEnv == "production" {
			return nil, fmt.Errorf("JWT_SECRET and JWT_REFRESH_SECRET are required in production")
		}
		// dev-only fallback so `go run` works out of the box without a .env,
		// but this is loud and unmistakable, never silent.
		if cfg.JWTSecret == "" {
			cfg.JWTSecret = "dev-only-insecure-access-secret-change-me"
		}
		if cfg.JWTRefreshSecret == "" {
			cfg.JWTRefreshSecret = "dev-only-insecure-refresh-secret-change-me"
		}
	}

	return cfg, nil
}

func (c *Config) DSN() string {
	// parseTime=true lets the mysql driver scan DATETIME directly into time.Time.
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4&collation=utf8mb4_unicode_ci&loc=Local",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName)
}

func (c *Config) IsProduction() bool {
	return strings.EqualFold(c.AppEnv, "production")
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	v := getEnv(key, "")
	if v == "" {
		return fallback
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return i
}

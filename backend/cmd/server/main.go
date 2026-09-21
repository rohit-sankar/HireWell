package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/resume-platform/backend/internal/config"
	"github.com/resume-platform/backend/internal/database"
	"github.com/resume-platform/backend/internal/middleware"
	"github.com/resume-platform/backend/internal/routes"
	"github.com/resume-platform/backend/internal/utils"
)

func main() {
	migrateUp := flag.Bool("migrate-up", false, "run pending migrations then exit")
	migrateDown := flag.Bool("migrate-down", false, "roll back the most recent migration then exit")
	migrationsDir := flag.String("migrations-dir", "migrations", "path to the migrations directory")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "config error:", err)
		os.Exit(1)
	}

	logger := utils.NewLogger(cfg.AppEnv)

	db, err := database.Connect(cfg)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer db.Close()

	if *migrateUp {
		if err := database.MigrateUp(db, *migrationsDir); err != nil {
			logger.Fatal().Err(err).Msg("migration up failed")
		}
		return
	}
	if *migrateDown {
		if err := database.MigrateDown(db, *migrationsDir); err != nil {
			logger.Fatal().Err(err).Msg("migration down failed")
		}
		return
	}

	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(middleware.Recovery(logger))
	router.Use(middleware.RequestID())
	router.Use(middleware.StructuredLogging(logger))
	router.Use(middleware.SecurityHeaders())
	router.Use(middleware.CORS(cfg.FrontendURL))
	router.Use(middleware.NewRateLimiter(cfg.RateLimitRequestsPerMinute).Middleware())
	router.Use(middleware.ErrorHandler(logger))

	if err := routes.Register(router, db, cfg, logger); err != nil {
		logger.Fatal().Err(err).Msg("failed to register routes")
	}

	srv := &http.Server{
		Addr:         ":" + cfg.AppPort,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second, // generous: resume parsing calls an external AI API synchronously
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info().Str("port", cfg.AppPort).Str("env", cfg.AppEnv).Msg("server starting")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal().Err(err).Msg("server failed")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info().Msg("shutting down gracefully...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error().Err(err).Msg("forced shutdown")
	}
}

// Package routes wires every controller to its /api/v1 endpoint, attaching
// the correct RequireAuth / RequireRole middleware per the spec's "a
// candidate must never access HR/company-only resources" rule. This file
// is intentionally the only place route paths + middleware + handler are
// declared together, so the full API surface is auditable in one place.
package routes

import (
	"database/sql"

	"github.com/gin-gonic/gin"
	"github.com/resume-platform/backend/internal/ai"
	"github.com/resume-platform/backend/internal/auth"
	"github.com/resume-platform/backend/internal/config"
	"github.com/resume-platform/backend/internal/controllers"
	"github.com/resume-platform/backend/internal/middleware"
	"github.com/resume-platform/backend/internal/models"
	"github.com/resume-platform/backend/internal/repository"
	"github.com/resume-platform/backend/internal/resume"
	"github.com/resume-platform/backend/internal/services"
	"github.com/resume-platform/backend/internal/storage"
	"github.com/rs/zerolog"
)

func Register(router *gin.Engine, db *sql.DB, cfg *config.Config, logger zerolog.Logger) error {
	// --- repositories ---------------------------------------------------
	userRepo := repository.NewUserRepository(db)
	refreshTokenRepo := repository.NewRefreshTokenRepository(db)
	candidateRepo := repository.NewCandidateRepository(db)
	companyRepo := repository.NewCompanyRepository(db)
	resumeRepo := repository.NewResumeRepository(db)
	jobRepo := repository.NewJobRepository(db)
	applicationRepo := repository.NewApplicationRepository(db)
	matchRepo := repository.NewMatchRepository(db)
	interviewRepo := repository.NewInterviewRepository(db)

	// --- cross-cutting dependencies --------------------------------------
	tokenManager := auth.NewTokenManager(cfg.JWTSecret, cfg.JWTRefreshSecret, cfg.JWTAccessExpiry, cfg.JWTRefreshExpiry)
	fileStorage, err := storage.NewLocalStorage(cfg.UploadDir)
	if err != nil {
		return err
	}
	aiProvider, err := ai.NewProvider(cfg.AIProvider, cfg.AIAPIKey, cfg.AIModel)
	if err != nil {
		return err
	}
	logger.Info().Str("provider", aiProvider.Name()).Msg("AI provider initialized")

	_ = resume.ErrUnsupportedFormat // ensure resume package is linked even if only used transitively via services

	// --- services ---------------------------------------------------------
	authService := services.NewAuthService(userRepo, refreshTokenRepo, candidateRepo, companyRepo, tokenManager)
	candidateService := services.NewCandidateService(candidateRepo, resumeRepo)
	companyService := services.NewCompanyService(companyRepo)
	resumeService := services.NewResumeService(resumeRepo, candidateRepo, fileStorage, aiProvider, cfg.MaxUploadSize, logger)
	jobService := services.NewJobService(jobRepo, companyRepo)
	matchingService := services.NewMatchingService(matchRepo, applicationRepo, jobRepo, resumeRepo, jobService, aiProvider, logger)
	applicationService := services.NewApplicationService(applicationRepo, jobRepo, candidateRepo, jobService, matchingService)
	interviewService := services.NewInterviewService(interviewRepo, applicationRepo, jobRepo, resumeRepo, jobService, aiProvider, logger)

	// --- controllers --------------------------------------------------------
	authController := controllers.NewAuthController(authService, cfg)
	candidateController := controllers.NewCandidateController(candidateService, resumeService)
	companyController := controllers.NewCompanyController(companyService)
	jobController := controllers.NewJobController(jobService)
	applicationController := controllers.NewApplicationController(applicationService, matchingService, jobService)
	interviewController := controllers.NewInterviewController(interviewService, applicationService)
	healthController := controllers.NewHealthController(db, aiProvider)

	// --- health (unversioned, unauthenticated) -----------------------------
	router.GET("/health", healthController.Health)
	router.GET("/ready", healthController.Ready)

	v1 := router.Group("/api/v1")

	// --- auth ---------------------------------------------------------------
	authGroup := v1.Group("/auth")
	{
		authGroup.POST("/register/candidate", authController.RegisterCandidate)
		authGroup.POST("/register/company", authController.RegisterCompany)
		authGroup.POST("/register", authController.RegisterCandidate) // alias per spec's generic /auth/register
		authGroup.POST("/login", authController.Login)
		authGroup.POST("/refresh", authController.Refresh)
		authGroup.POST("/logout", authController.Logout)
		authGroup.GET("/me", middleware.RequireAuth(tokenManager), authController.Me)
		authGroup.GET("/google", authController.GoogleLogin)
		authGroup.GET("/google/callback", authController.GoogleCallback)
	}

	requireAuth := middleware.RequireAuth(tokenManager)
	requireCandidate := middleware.RequireRole(models.RoleCandidate)
	requireCompany := middleware.RequireRole(models.RoleCompany)

	// --- candidates -----------------------------------------------------
	candidateGroup := v1.Group("/candidates")
	{
		me := candidateGroup.Group("/me", requireAuth, requireCandidate)
		me.GET("", candidateController.Me)
		me.PUT("", candidateController.UpdateMe)
		me.PUT("/profile", candidateController.UpdateMe) // alias per spec path
		me.GET("/resume", candidateController.GetResume)
		me.GET("/resumes", candidateController.ListResumes)
		me.POST("/resume", candidateController.UploadResume)
		me.PUT("/resume/:id", candidateController.UpdateResumeProfile)
		me.GET("/applications", applicationController.ListMine)
	}

	// --- companies ------------------------------------------------------
	companyGroup := v1.Group("/companies")
	{
		me := companyGroup.Group("/me", requireAuth, requireCompany)
		me.GET("", companyController.Me)
		me.PUT("", companyController.UpdateMe)
	}

	// --- jobs -------------------------------------------------------------
	jobGroup := v1.Group("/jobs")
	{
		jobGroup.GET("", middleware.OptionalAuth(tokenManager), jobController.List) // public: published jobs, or "mine=true" for a company's own
		jobGroup.GET("/:id", jobController.Get)
		jobGroup.POST("", requireAuth, requireCompany, jobController.Create)
		jobGroup.PUT("/:id", requireAuth, requireCompany, jobController.Update)
		jobGroup.DELETE("/:id", requireAuth, requireCompany, jobController.Delete)
		jobGroup.POST("/:id/publish", requireAuth, requireCompany, jobController.Publish)
		jobGroup.POST("/:id/close", requireAuth, requireCompany, jobController.Close)
		jobGroup.POST("/:id/archive", requireAuth, requireCompany, jobController.Archive)
		jobGroup.PUT("/:id/match-weights", requireAuth, requireCompany, jobController.UpdateMatchWeights)
		jobGroup.POST("/:id/apply", requireAuth, requireCandidate, applicationController.Apply)
		jobGroup.GET("/:id/applications", requireAuth, requireCompany, applicationController.ListForJob)
		jobGroup.GET("/:id/matches", requireAuth, requireCompany, applicationController.GetJobMatches)
	}

	// --- applications -------------------------------------------------------
	applicationGroup := v1.Group("/applications", requireAuth)
	{
		applicationGroup.GET("/:id", applicationController.Get)
		applicationGroup.PUT("/:id/status", requireCompany, applicationController.UpdateStatus)
		applicationGroup.GET("/:id/match", applicationController.GetMatch)
		applicationGroup.POST("/:id/match/recalculate", requireCompany, applicationController.RecalculateMatch)
	}

	// --- interviews -----------------------------------------------------
	interviewGroup := v1.Group("/interviews", requireAuth)
	{
		interviewGroup.POST("", requireCandidate, interviewController.Create)
		interviewGroup.GET("/:id", interviewController.Get)
		interviewGroup.POST("/:id/start", requireCandidate, interviewController.Start)
		interviewGroup.POST("/:id/answer", requireCandidate, interviewController.Answer)
		interviewGroup.POST("/:id/complete", requireCandidate, interviewController.Complete)
		interviewGroup.GET("/:id/result", interviewController.Result)
	}

	// --- reports ----------------------------------------------------------
	// NOTE (documented limitation, see README "Known Limitations"): job- and
	// candidate-level aggregate report endpoints are not implemented in
	// this delivery. GET /jobs/:id/matches (above) already returns every
	// ranked applicant with a full score breakdown, which covers the core
	// "companies receive ranked candidates" requirement; a dedicated
	// analytics/aggregation layer (time-to-hire, funnel conversion, etc.)
	// is future work.

	return nil
}

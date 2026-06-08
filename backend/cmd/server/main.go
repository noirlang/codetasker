// Package main is the entry point for the CodeTasker backend server.
// It wires together all layers (config → database → repository → service →
// controller), configures the Fiber HTTP server with middleware, registers all
// routes, and starts the server with graceful shutdown on SIGINT/SIGTERM.
//
// Startup sequence:
//  1. Load .env file (non-fatal if absent — production uses real env vars).
//  2. Initialise the structured logger (zap production config).
//  3. Load and validate configuration from environment variables.
//  4. Connect to MongoDB and ensure indexes exist.
//  5. Build repository → service → controller dependency graph.
//  6. Configure Fiber with recovery, logger, and CORS middleware.
//  7. Register all HTTP routes.
//  8. Start the server and block until a shutdown signal is received.
//  9. Gracefully shut down: stop accepting new connections, close MongoDB.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/codetasker/backend/internal/config"
	"github.com/codetasker/backend/internal/controller"
	"github.com/codetasker/backend/internal/database"
	"github.com/codetasker/backend/internal/middleware"
	"github.com/codetasker/backend/internal/parser"
	"github.com/codetasker/backend/internal/repository"
	"github.com/codetasker/backend/internal/service"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	fiberlogger "github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

func main() {
	// ── Step 1: Load .env file ───────────────────────────────────────────────
	// godotenv.Load populates os.Getenv from a .env file if present.
	// In production environments, real env vars take precedence over .env values,
	// and the absence of the file is intentional — so we ignore the error.
	if err := godotenv.Load(); err != nil {
		// Not fatal; log the skip so operators understand the behaviour.
		fmt.Println("[INFO] .env file not found, using system environment variables")
	}

	// ── Step 2: Initialise logger ────────────────────────────────────────────
	// zap.NewProduction builds a JSON-format logger suitable for log aggregation
	// systems (Datadog, Loki, CloudWatch, etc.).
	log, err := zap.NewProduction()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialise logger: %v\n", err)
		os.Exit(1)
	}
	defer log.Sync() //nolint:errcheck

	// ── Step 3: Load configuration ───────────────────────────────────────────
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("configuration error", zap.Error(err))
	}
	log.Info("configuration loaded", zap.String("port", cfg.Port), zap.String("db", cfg.DBName))

	// ── Step 4: Connect to MongoDB ───────────────────────────────────────────
	db, err := database.Connect(cfg.MongoURI, cfg.DBName)
	if err != nil {
		log.Fatal("mongodb connection failed", zap.Error(err))
	}
	log.Info("mongodb connected", zap.String("db", cfg.DBName))

	// ── Step 5: Build dependency graph ───────────────────────────────────────
	// Repositories
	userRepo := repository.NewUserRepository(db)
	taskRepo := repository.NewTaskRepository(db)
	syncedRepo := repository.NewSyncedRepository(db)
	collaboratorRepo := repository.NewCollaboratorRepository(db)

	// Services
	authService := service.NewAuthService(cfg, userRepo, log)
	githubService := service.NewGithubService(cfg, userRepo, log)
	todoParser := parser.NewParser()
	taskService := service.NewTaskService(taskRepo, userRepo, todoParser, githubService, log)

	// Controllers
	authCtrl := controller.NewAuthController(authService)
	repoCtrl := controller.NewRepoController(cfg, githubService, syncedRepo, collaboratorRepo, userRepo)
	webhookCtrl := controller.NewWebhookController(taskService)
	taskCtrl := controller.NewTaskController(taskService, githubService, syncedRepo, collaboratorRepo)

	// ── Step 6: Configure Fiber ──────────────────────────────────────────────
	app := fiber.New(fiber.Config{
		// AppName is embedded in the Server header.
		AppName: "CodeTasker API v1.0",
		// ReadTimeout guards against slow-client attacks.
		ReadTimeout: 30 * time.Second,
		// WriteTimeout bounds response time including handler execution.
		WriteTimeout: 30 * time.Second,
		// ErrorHandler provides a consistent JSON shape for unhandled panics and
		// errors returned by handlers.
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			var fiberErr *fiber.Error
			if errors, ok := err.(*fiber.Error); ok {
				fiberErr = errors
				code = fiberErr.Code
			}
			return c.Status(code).JSON(fiber.Map{
				"error":   "server_error",
				"message": err.Error(),
			})
		},
	})

	// Recovery middleware catches panics in handlers and converts them to 500 errors
	// instead of crashing the server process.
	app.Use(recover.New(recover.Config{
		EnableStackTrace: true,
	}))

	// HTTP access logging in Combined Log Format — compatible with most log
	// aggregation pipelines.
	app.Use(fiberlogger.New(fiberlogger.Config{
		Format: "[${time}] ${method} ${path} → ${status} (${latency})\n",
	}))

	// CORS middleware allows the frontend origin to make credentialed requests
	// (needed for the auth_token cookie). All other origins are rejected.
	app.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.FrontendURL,
		AllowCredentials: true,
		AllowMethods:     "GET,POST,PATCH,DELETE,OPTIONS",
		AllowHeaders:     "Content-Type,Authorization",
	}))

	// ── Step 7: Register routes ──────────────────────────────────────────────

	// Health check — no auth required; used by load balancers and readiness probes.
	app.Get("/api/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"service": "codetasker-api",
			"time":    time.Now().UTC().Format(time.RFC3339),
		})
	})

	// Auth routes — publicly accessible (no JWT required).
	authCtrl.RegisterRoutes(app)

	// Webhook routes — protected by HMAC signature verification, not JWT.
	// The VerifyWebhookSignature middleware is applied to this group only.
	webhooks := app.Group("/api/webhooks",
		middleware.VerifyWebhookSignature(cfg.WebhookSecret),
	)
	webhookCtrl.RegisterRoutes(webhooks)

	// All other /api/* routes require a valid JWT.
	protected := app.Group("/api", middleware.Protected(cfg))
	authCtrl.RegisterProtectedRoutes(protected)
	repoCtrl.RegisterRoutes(protected)
	taskCtrl.RegisterRoutes(protected)

	// ── Step 8: Start server ─────────────────────────────────────────────────
	// We start the server in a background goroutine so the main goroutine can
	// block on the OS signal channel for graceful shutdown.
	addr := ":" + cfg.Port
	serverErrors := make(chan error, 1)

	go func() {
		log.Info("server starting", zap.String("addr", addr))
		if err := app.Listen(addr); err != nil {
			serverErrors <- err
		}
	}()

	// ── Step 9: Graceful shutdown ─────────────────────────────────────────────
	// Block until SIGINT (Ctrl-C) or SIGTERM (Docker/Kubernetes stop) is received,
	// or until the server itself returns a fatal error.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		log.Fatal("server error", zap.Error(err))

	case sig := <-quit:
		log.Info("shutdown signal received", zap.String("signal", sig.String()))
	}

	log.Info("shutting down server gracefully…")

	// Give in-flight requests up to 10 seconds to complete.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := app.ShutdownWithContext(shutdownCtx); err != nil {
		log.Error("server shutdown error", zap.Error(err))
	}

	// Close the MongoDB connection pool cleanly.
	if err := db.Disconnect(shutdownCtx); err != nil {
		log.Error("mongodb disconnect error", zap.Error(err))
	}

	log.Info("server stopped")
}

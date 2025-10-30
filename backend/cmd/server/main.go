package main

import (
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/nimbus/backend/internal/config"
	"github.com/nimbus/backend/internal/db"
	"github.com/nimbus/backend/internal/handlers"
	"github.com/nimbus/backend/internal/middleware"
	"github.com/nimbus/backend/internal/repository"
	"github.com/nimbus/backend/internal/services"
	"github.com/nimbus/backend/internal/workers"
)

func main() {
	// Load environment variables
	config.MustLoadEnv()

	// Connect to database
	database, err := db.Connect()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()
	log.Println("✓ Connected to database")

	// Run database migrations
	log.Println("Running database migrations...")
	if err := db.RunMigrations(database); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	log.Println("✓ Database migrations completed")

	// Initialize repositories
	userRepo := repository.NewUserRepository(database)
	serviceRepo := repository.NewServiceRepository(database)
	preferencesRepo := repository.NewPreferencesRepository(database)

	// Initialize services
	authService := services.NewAuthService()

	// Initialize health check service
	healthCheckTimeout := getEnvDuration("HEALTH_CHECK_TIMEOUT", 10*time.Second)
	healthCheckService := services.NewHealthCheckService(serviceRepo, healthCheckTimeout)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(userRepo, authService)
	serviceHandler := handlers.NewServiceHandler(serviceRepo, healthCheckService)
	preferencesHandler := handlers.NewPreferencesHandler(preferencesRepo)
	adminHandler := handlers.NewAdminHandler(userRepo)

	// Create fiber app
	app := fiber.New(fiber.Config{
		AppName: "Nimbus API",
	})

	// Middleware
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins:     os.Getenv("CORS_ORIGINS"),
		AllowCredentials: true,
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
	}))

	// Routes
	api := app.Group("/api")
	v1 := api.Group("/v1")

	// Health check
	v1.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status": "healthy",
			"app":    "nimbus",
		})
	})

	// Auth routes (public)
	auth := v1.Group("/auth")
	auth.Post("/register", authHandler.Register)
	auth.Post("/login", authHandler.Login)
	auth.Post("/logout", authHandler.Logout)

	// Protected auth routes
	authProtected := auth.Group("", middleware.AuthMiddleware(authService, userRepo))
	authProtected.Get("/me", authHandler.GetMe)

	// Service routes (all protected)
	services := v1.Group("/services", middleware.AuthMiddleware(authService, userRepo))
	services.Post("/", serviceHandler.CreateService)
	services.Get("/", serviceHandler.GetServices)
	services.Put("/reorder", serviceHandler.ReorderServices) // Must be before /:id routes
	services.Get("/:id", serviceHandler.GetService)
	services.Put("/:id", serviceHandler.UpdateService)
	services.Delete("/:id", serviceHandler.DeleteService)
	services.Post("/:id/check", serviceHandler.CheckService)

	// User preferences routes (protected)
	preferences := v1.Group("/users/me/preferences", middleware.AuthMiddleware(authService, userRepo))
	preferences.Get("/", preferencesHandler.GetPreferences)
	preferences.Put("/", preferencesHandler.UpdatePreferences)

	// Admin routes (protected, admin only)
	admin := v1.Group("/admin", middleware.AuthMiddleware(authService, userRepo), middleware.AdminOnly())
	admin.Get("/users", adminHandler.GetAllUsers)
	admin.Get("/users/stats", adminHandler.GetUserStats)
	admin.Put("/users/:id/role", adminHandler.UpdateUserRole)
	admin.Delete("/users/:id", adminHandler.DeleteUser)

	// Start health check monitor
	healthCheckInterval := getEnvDuration("HEALTH_CHECK_INTERVAL", 60*time.Second)
	healthMonitor := workers.NewHealthMonitor(healthCheckService, serviceRepo, healthCheckInterval)
	healthMonitor.Start()

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start server in a goroutine
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	go func() {
		log.Printf("Server starting on port %s", port)
		log.Printf("Auth endpoints available:")
		log.Printf("  POST /api/v1/auth/register")
		log.Printf("  POST /api/v1/auth/login")
		log.Printf("  POST /api/v1/auth/logout")
		log.Printf("  GET  /api/v1/auth/me (protected)")
		log.Printf("Service endpoints available:")
		log.Printf("  POST   /api/v1/services (protected)")
		log.Printf("  GET    /api/v1/services (protected)")
		log.Printf("  GET    /api/v1/services/:id (protected)")
		log.Printf("  PUT    /api/v1/services/:id (protected)")
		log.Printf("  DELETE /api/v1/services/:id (protected)")
		log.Printf("  POST   /api/v1/services/:id/check (protected) - Manual health check")
		log.Printf("  PUT    /api/v1/services/reorder (protected) - Reorder services")
		if err := app.Listen(":" + port); err != nil {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal
	<-sigChan
	log.Println("\nReceived shutdown signal, shutting down gracefully...")

	// Stop health monitor
	healthMonitor.Stop()

	// Shutdown Fiber app
	if err := app.Shutdown(); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}

	log.Println("Server stopped")
}

// getEnvDuration reads a duration from environment variable in seconds
func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	valStr := os.Getenv(key)
	if valStr == "" {
		return defaultValue
	}

	seconds, err := strconv.Atoi(valStr)
	if err != nil {
		log.Printf("Invalid value for %s: %s, using default %v", key, valStr, defaultValue)
		return defaultValue
	}

	return time.Duration(seconds) * time.Second
}

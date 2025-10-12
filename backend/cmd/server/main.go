package main

import (
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/joho/godotenv"
	"github.com/nimbus/backend/internal/db"
	"github.com/nimbus/backend/internal/handlers"
	"github.com/nimbus/backend/internal/middleware"
	"github.com/nimbus/backend/internal/repository"
	"github.com/nimbus/backend/internal/services"
)

func main() {
	// Load environment variables
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found")
	}

	// Connect to database
	database, err := db.Connect()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()
	log.Println("✓ Connected to database")

	// Initialize repositories
	userRepo := repository.NewUserRepository(database)

	// Initialize services
	authService := services.NewAuthService()

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(userRepo, authService)

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
	authProtected := auth.Group("", middleware.AuthMiddleware(authService))
	authProtected.Get("/me", authHandler.GetMe)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	log.Printf("Auth endpoints available:")
	log.Printf("  POST /api/v1/auth/register")
	log.Printf("  POST /api/v1/auth/login")
	log.Printf("  POST /api/v1/auth/logout")
	log.Printf("  GET  /api/v1/auth/me (protected)")
	log.Fatal(app.Listen(":" + port))
}

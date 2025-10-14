//go:build dev

package main

import (
	"log"

	"github.com/joho/godotenv"
	"github.com/nimbus/backend/internal/db"
	"github.com/nimbus/backend/internal/repository"
	"github.com/nimbus/backend/internal/seeds"
	"github.com/nimbus/backend/internal/services"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	// Connect to database
	database, err := db.Connect()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()
	log.Println("✓ Connected to database")

	// Initialize repositories and services
	userRepo := repository.NewUserRepository(database)
	authService := services.NewAuthService()

	// Run seeder
	if err := seeds.SeedUsers(userRepo, authService, database); err != nil {
		log.Fatalf("Failed to seed users: %v", err)
	}

	log.Println("\n✓ Seeding completed successfully!")
}

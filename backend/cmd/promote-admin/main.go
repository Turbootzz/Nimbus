package main

import (
	"fmt"
	"log"
	"os"

	"github.com/nimbus/backend/internal/config"
	"github.com/nimbus/backend/internal/db"
	"github.com/nimbus/backend/internal/repository"
)

func main() {
	// Load environment variables
	config.MustLoadEnv()

	// Check if email argument provided
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run cmd/promote-admin/main.go <user-email>")
		fmt.Println("Example: go run cmd/promote-admin/main.go user@example.com")
		os.Exit(1)
	}

	email := os.Args[1]

	// Connect to database
	database, err := db.Connect()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	// Initialize repository
	userRepo := repository.NewUserRepository(database)

	// Get user by email
	user, err := userRepo.GetByEmail(email)
	if err != nil {
		log.Fatalf("User not found: %v", err)
	}

	// Check if already admin
	if user.Role == "admin" {
		fmt.Printf("✓ User %s (%s) is already an admin\n", user.Name, user.Email)
		os.Exit(0)
	}

	// Promote to admin
	if err := userRepo.UpdateRole(user.ID, "admin"); err != nil {
		log.Fatalf("Failed to promote user: %v", err)
	}

	fmt.Printf("✓ Successfully promoted %s (%s) to admin!\n", user.Name, user.Email)
}

package main

import (
	"log"

	"github.com/nimbus/backend/internal/config"
	"github.com/nimbus/backend/internal/db"
)

func main() {
	config.MustLoadEnv()

	// Connect to database
	database, err := db.Connect()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	log.Println("Running database migrations...")

	// Run migrations
	if err := db.RunMigrations(database); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	log.Println("✅ All migrations completed successfully!")
}

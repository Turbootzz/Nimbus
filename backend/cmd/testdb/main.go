package main

import (
	"fmt"
	"os"

	"github.com/nimbus/backend/internal/config"
	"github.com/nimbus/backend/internal/db"
)

func main() {
	// Load environment variables
	config.LoadEnv()

	fmt.Println("🔍 Testing database connection...")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("Host:     %s\n", os.Getenv("DB_HOST"))
	fmt.Printf("Port:     %s\n", os.Getenv("DB_PORT"))
	fmt.Printf("Database: %s\n", os.Getenv("DB_NAME"))
	fmt.Printf("User:     %s\n", os.Getenv("DB_USER"))
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	// Test the connection
	err := db.TestConnection()
	if err != nil {
		fmt.Println("❌ Database connection FAILED")
		fmt.Printf("Error: %v\n", err)
		fmt.Println()
		fmt.Println("Troubleshooting steps:")
		fmt.Println("1. Make sure PostgreSQL is running:")
		fmt.Println("   ps aux | grep postgres")
		fmt.Println()
		fmt.Println("2. Check if the database exists:")
		fmt.Println("   psql -U postgres -c '\\l' | grep nimbus")
		fmt.Println()
		fmt.Println("3. Create the database if needed:")
		fmt.Println("   psql -U postgres")
		fmt.Println("   CREATE DATABASE nimbus;")
		fmt.Println("   \\q")
		fmt.Println()
		fmt.Println("4. Update .env (root) with correct credentials")
		os.Exit(1)
	}

	fmt.Println("✅ Database connection SUCCESS!")
	fmt.Println()
	fmt.Println("Your database is ready for development! 🎉")
}

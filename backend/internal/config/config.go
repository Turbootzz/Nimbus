package config

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

// LoadEnv loads environment variables from .env file with proper error handling
func LoadEnv() error {
	// Try to load .env from current directory, then parent directory
	err := godotenv.Load(".env")
	if err != nil {
		// Fallback to parent directory (for when running from backend/)
		err = godotenv.Load("../.env")
		if err != nil {
			return fmt.Errorf("failed to load .env file: %w (ensure file exists and has valid syntax)", err)
		}
	}

	// Validate critical environment variables
	if err := validateRequiredEnvVars(); err != nil {
		return err
	}

	return nil
}

// MustLoadEnv loads environment variables or exits with clear error message
func MustLoadEnv() {
	if err := LoadEnv(); err != nil {
		log.Fatalf("Failed to load environment: %v", err)
	}
}

// validateRequiredEnvVars ensures critical environment variables are set
func validateRequiredEnvVars() error {
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return fmt.Errorf("JWT_SECRET is required")
	}
	if len(jwtSecret) < 32 {
		return fmt.Errorf("JWT_SECRET must be at least 32 characters for security")
	}

	if os.Getenv("DB_NAME") == "" {
		return fmt.Errorf("DB_NAME is required")
	}

	if os.Getenv("PORT") == "" {
		return fmt.Errorf("PORT is required")
	}

	return nil
}

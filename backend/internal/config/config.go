package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// LoadEnv loads environment variables from .env file with proper error handling
// If no .env file is found, it will continue (for Docker/production deployments)
func LoadEnv() error {
	// Try to load .env from current directory, then parent directory
	err := godotenv.Load(".env")
	if err != nil {
		// Fallback to parent directory (for when running from backend/)
		err = godotenv.Load("../.env")
		if err != nil {
			// .env file is optional - environment variables may be set directly (Docker/production)
			log.Println("No .env file found, using environment variables directly")
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

// validateRequiredEnvVars ensures critical environment variables are set and properly formatted
func validateRequiredEnvVars() error {
	var errors []string

	// Validate JWT_SECRET
	jwtSecret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if jwtSecret == "" {
		errors = append(errors, "JWT_SECRET is required")
	} else if len(jwtSecret) < 32 {
		errors = append(errors, "JWT_SECRET must be at least 32 characters for security")
	}

	// Validate database host
	dbHost := strings.TrimSpace(os.Getenv("DB_HOST"))
	if dbHost == "" {
		errors = append(errors, "DB_HOST is required")
	}

	// Validate database port
	dbPort := strings.TrimSpace(os.Getenv("DB_PORT"))
	if dbPort == "" {
		errors = append(errors, "DB_PORT is required")
	} else {
		if port, err := strconv.Atoi(dbPort); err != nil || port < 1 || port > 65535 {
			errors = append(errors, "DB_PORT must be a valid port number (1-65535)")
		}
	}

	// Validate database name
	dbName := strings.TrimSpace(os.Getenv("DB_NAME"))
	if dbName == "" {
		errors = append(errors, "DB_NAME is required")
	}

	// Validate database user
	dbUser := strings.TrimSpace(os.Getenv("DB_USER"))
	if dbUser == "" {
		errors = append(errors, "DB_USER is required")
	}

	// Validate database password (don't trim - spaces may be intentional)
	dbPassword := os.Getenv("DB_PASSWORD")
	if dbPassword == "" {
		errors = append(errors, "DB_PASSWORD is required")
	}

	// Validate server port
	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		errors = append(errors, "PORT is required")
	} else {
		if p, err := strconv.Atoi(port); err != nil || p < 1 || p > 65535 {
			errors = append(errors, "PORT must be a valid port number (1-65535)")
		}
	}

	// Validate CORS origins (critical for security)
	corsOrigins := strings.TrimSpace(os.Getenv("CORS_ORIGINS"))
	if corsOrigins == "" {
		errors = append(errors, "CORS_ORIGINS is required (security risk if misconfigured)")
	}

	// Return all validation errors
	if len(errors) > 0 {
		return fmt.Errorf("environment validation failed:\n  - %s", strings.Join(errors, "\n  - "))
	}

	return nil
}

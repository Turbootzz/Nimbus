package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
	"github.com/nimbus/backend/internal/models"
)

// Default values for environment variables (Convention over Configuration)
// Note: CORS_ORIGINS is intentionally not in defaults - when unset, same-origin mode is used
var defaults = map[string]string{
	"PORT":    "8080",
	"DB_HOST": "db",
	"DB_PORT": "5432",
	"DB_USER": "nimbus",
	"DB_NAME": "nimbus",
}

// applyDefaults sets default values for environment variables if not already set
func applyDefaults() {
	for key, value := range defaults {
		if os.Getenv(key) == "" {
			os.Setenv(key, value)
			log.Printf("Using default %s=%s", key, value)
		}
	}
}

// GetEnvOrDefault returns the environment variable value or a default
func GetEnvOrDefault(key, defaultValue string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return defaultValue
}

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

	// Apply defaults for optional variables before validation
	applyDefaults()

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

	// CORS_ORIGINS is optional - when not set, same-origin mode is used (no CORS middleware)
	// This enables the unified Docker image where nginx proxies both frontend and backend
	corsOrigins := strings.TrimSpace(os.Getenv("CORS_ORIGINS"))
	if corsOrigins == "" {
		log.Println("CORS_ORIGINS not set - same-origin mode (CORS middleware will be skipped)")
	}

	// Return all validation errors
	if len(errors) > 0 {
		return fmt.Errorf("environment validation failed:\n  - %s", strings.Join(errors, "\n  - "))
	}

	return nil
}

// GetGoogleOAuthConfig returns the Google OAuth configuration
func GetGoogleOAuthConfig() models.OAuthConfig {
	return models.OAuthConfig{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("GOOGLE_REDIRECT_URL"),
	}
}

// GetGitHubOAuthConfig returns the GitHub OAuth configuration
func GetGitHubOAuthConfig() models.OAuthConfig {
	return models.OAuthConfig{
		ClientID:     os.Getenv("GITHUB_CLIENT_ID"),
		ClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("GITHUB_REDIRECT_URL"),
	}
}

// GetDiscordOAuthConfig returns the Discord OAuth configuration
func GetDiscordOAuthConfig() models.OAuthConfig {
	return models.OAuthConfig{
		ClientID:     os.Getenv("DISCORD_CLIENT_ID"),
		ClientSecret: os.Getenv("DISCORD_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("DISCORD_REDIRECT_URL"),
	}
}

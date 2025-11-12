package testutil

import (
	"database/sql"
	"fmt"
	"os"
	"testing"

	_ "github.com/lib/pq"
)

// SetupPostgresTestDB creates a test database for integration testing
// Set INTEGRATION_TEST=true and provide DB credentials via env vars to run
func SetupPostgresTestDB(t *testing.T) (*sql.DB, func()) {
	// Skip if not running integration tests
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=true to run.")
	}

	// Get PostgreSQL connection details from environment
	host := getEnvOrDefault("TEST_DB_HOST", "localhost")
	port := getEnvOrDefault("TEST_DB_PORT", "5432")
	user := getEnvOrDefault("TEST_DB_USER", "nimbus")
	password := getEnvOrDefault("TEST_DB_PASSWORD", "")
	dbName := getEnvOrDefault("TEST_DB_NAME", "nimbus_test")

	// Connect to PostgreSQL
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbName)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		t.Fatalf("Failed to ping test database: %v", err)
	}

	// Cleanup function to drop all tables after test
	cleanup := func() {
		// Drop tables in reverse dependency order
		tables := []string{
			"service_status_logs",
			"services",
			"categories",
			"user_preferences",
			"users",
		}

		for _, table := range tables {
			_, err := db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", table))
			if err != nil {
				t.Logf("Warning: Failed to drop table %s: %v", table, err)
			}
		}

		db.Close()
	}

	return db, cleanup
}

// RunMigrations runs database migrations for testing
// This assumes migrations are in backend/internal/db/migrations
func RunMigrations(t *testing.T, db *sql.DB) {
	// For testing, we'll manually create the schema instead of running full migrations
	// This is faster and more reliable for tests

	// Create users table
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			email VARCHAR(255) UNIQUE NOT NULL,
			name VARCHAR(255) NOT NULL,
			password_hash VARCHAR(255) NOT NULL,
			role VARCHAR(20) DEFAULT 'user',
			last_activity_at TIMESTAMP WITH TIME ZONE,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create users table: %v", err)
	}

	// Create categories table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS categories (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			name VARCHAR(100) NOT NULL,
			color VARCHAR(7) DEFAULT '#6366f1',
			position INTEGER DEFAULT 0,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);

		CREATE INDEX IF NOT EXISTS idx_categories_user_id ON categories(user_id);
		CREATE INDEX IF NOT EXISTS idx_categories_position ON categories(user_id, position);
	`)
	if err != nil {
		t.Fatalf("Failed to create categories table: %v", err)
	}

	// Create services table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS services (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			name VARCHAR(255) NOT NULL,
			url TEXT NOT NULL,
			icon VARCHAR(50) DEFAULT '🔗',
			description TEXT,
			status VARCHAR(20) DEFAULT 'unknown',
			response_time INTEGER,
			position INTEGER DEFAULT 0,
			category_id UUID REFERENCES categories(id) ON DELETE SET NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);

		CREATE INDEX IF NOT EXISTS idx_services_user_id ON services(user_id);
		CREATE INDEX IF NOT EXISTS idx_services_status ON services(status);
		CREATE INDEX IF NOT EXISTS idx_services_position ON services(user_id, position);
		CREATE INDEX IF NOT EXISTS idx_services_category_id ON services(category_id);
	`)
	if err != nil {
		t.Fatalf("Failed to create services table: %v", err)
	}

	// Create user_preferences table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS user_preferences (
			user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
			theme_mode VARCHAR(10) DEFAULT 'dark',
			theme_background TEXT,
			theme_accent_color VARCHAR(7),
			open_in_new_tab BOOLEAN DEFAULT false,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create user_preferences table: %v", err)
	}

	// Create service_status_logs table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS service_status_logs (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			service_id UUID NOT NULL REFERENCES services(id) ON DELETE CASCADE,
			status VARCHAR(20) NOT NULL CHECK(status IN ('online', 'offline', 'unknown')),
			response_time INTEGER,
			error_message TEXT,
			checked_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
		);

		CREATE INDEX IF NOT EXISTS idx_service_status_logs_service_id ON service_status_logs(service_id);
		CREATE INDEX IF NOT EXISTS idx_service_status_logs_checked_at ON service_status_logs(checked_at);
	`)
	if err != nil {
		t.Fatalf("Failed to create service_status_logs table: %v", err)
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

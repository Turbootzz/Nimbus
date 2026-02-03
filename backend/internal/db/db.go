package db

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"time"

	_ "github.com/lib/pq"
)

// buildDBURL constructs a PostgreSQL connection URL from individual components
// with proper URL encoding for special characters in credentials
func buildDBURL(host, port, user, password, dbname string) string {
	u := &url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(user, password),
		Host:     fmt.Sprintf("%s:%s", host, port),
		Path:     "/" + dbname,
		RawQuery: "sslmode=disable",
	}
	return u.String()
}

// Connect creates a connection to the PostgreSQL database
func Connect() (*sql.DB, error) {
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		// Construct URL from individual components
		dbURL = buildDBURL(
			os.Getenv("DB_HOST"),
			os.Getenv("DB_PORT"),
			os.Getenv("DB_USER"),
			os.Getenv("DB_PASSWORD"),
			os.Getenv("DB_NAME"),
		)
	}
	// Note: If DB_URL is provided directly, user is responsible for proper encoding

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

// TestConnection tests if the database connection is working
func TestConnection() error {
	db, err := Connect()
	if err != nil {
		return err
	}
	defer db.Close()

	// Test query
	var result int
	err = db.QueryRow("SELECT 1").Scan(&result)
	if err != nil {
		return fmt.Errorf("test query failed: %w", err)
	}

	if result != 1 {
		return fmt.Errorf("unexpected result: got %d, want 1", result)
	}

	return nil
}

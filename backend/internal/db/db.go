package db

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/lib/pq"
)

// Connect creates a connection to the PostgreSQL database
func Connect() (*sql.DB, error) {
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		// Construct URL from individual components
		host := os.Getenv("DB_HOST")
		port := os.Getenv("DB_PORT")
		user := os.Getenv("DB_USER")
		password := os.Getenv("DB_PASSWORD")
		dbname := os.Getenv("DB_NAME")

		dbURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
			user, password, host, port, dbname)
	}

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

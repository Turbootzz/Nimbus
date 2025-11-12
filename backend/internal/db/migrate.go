package db

import (
	"database/sql"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// RunMigrations runs all pending SQL migrations
func RunMigrations(db *sql.DB) error {
	// Create migrations table if it doesn't exist
	if err := createMigrationsTable(db); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Get all migration files
	migrationsDir := filepath.Join("internal", "db", "migrations")
	files, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	// Filter and sort .up.sql files
	var upFiles []fs.DirEntry
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".up.sql") {
			upFiles = append(upFiles, file)
		}
	}
	sort.Slice(upFiles, func(i, j int) bool {
		return upFiles[i].Name() < upFiles[j].Name()
	})

	// Run each migration
	for _, file := range upFiles {
		migrationName := strings.TrimSuffix(file.Name(), ".up.sql")

		// Check if already applied
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version = $1", migrationName).Scan(&count)
		if err != nil {
			return fmt.Errorf("failed to check migration status: %w", err)
		}

		if count > 0 {
			fmt.Printf("Migration %s already applied, skipping\n", migrationName)
			continue
		}

		// Read and execute migration
		filePath := filepath.Join(migrationsDir, file.Name())
		sqlBytes, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", file.Name(), err)
		}

		fmt.Printf("Running migration: %s\n", migrationName)
		if _, err := db.Exec(string(sqlBytes)); err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", migrationName, err)
		}

		// Record migration
		if _, err := db.Exec("INSERT INTO schema_migrations (version) VALUES ($1)", migrationName); err != nil {
			return fmt.Errorf("failed to record migration %s: %w", migrationName, err)
		}

		fmt.Printf("✓ Migration %s completed\n", migrationName)
	}

	return nil
}

func createMigrationsTable(db *sql.DB) error {
	// Check if schema_migrations exists with the correct schema
	var exists bool
	err := db.QueryRow(`
		SELECT EXISTS (
			SELECT FROM information_schema.tables
			WHERE table_name = 'schema_migrations'
		)
	`).Scan(&exists)

	if err != nil {
		return err
	}

	// If table exists, check if it has the correct schema
	if exists {
		var columnType string
		err := db.QueryRow(`
			SELECT data_type
			FROM information_schema.columns
			WHERE table_name = 'schema_migrations' AND column_name = 'version'
		`).Scan(&columnType)

		// If it has the correct schema (VARCHAR), we're good
		if err == nil && strings.Contains(strings.ToLower(columnType), "varchar") {
			return nil
		}

		// Old schema detected - drop it (but NOT CASCADE!)
		_, _ = db.Exec("DROP TABLE IF EXISTS schema_migrations")
	}

	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMP NOT NULL DEFAULT NOW()
		);
	`
	_, err = db.Exec(query)
	return err
}

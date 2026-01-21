package db

import (
	"database/sql"
	"strings"
)

// IsPostgreSQL detects if the database connection is PostgreSQL
func IsPostgreSQL(db *sql.DB) bool {
	if db == nil {
		return false
	}
	var version string
	if err := db.QueryRow("SELECT version()").Scan(&version); err != nil {
		return false
	}
	return strings.HasPrefix(version, "PostgreSQL")
}

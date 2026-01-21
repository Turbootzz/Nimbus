package db

import (
	"database/sql"
	"strings"
)

// IsPostgreSQL detects if the database connection is PostgreSQL by querying
// the database version. This runs a query on each call, so it should be called
// once at repository initialization time and the result cached.
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

package db

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestIsPostgreSQL_NilDB(t *testing.T) {
	result := IsPostgreSQL(nil)
	if result {
		t.Error("IsPostgreSQL(nil) expected false, got true")
	}
}

func TestIsPostgreSQL_SQLite(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open SQLite database: %v", err)
	}
	defer db.Close()

	result := IsPostgreSQL(db)
	if result {
		t.Error("IsPostgreSQL(sqlite) expected false, got true")
	}
}

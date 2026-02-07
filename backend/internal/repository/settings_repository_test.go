package repository

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func setupSettingsTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	schema := `
		CREATE TABLE IF NOT EXISTS system_settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_by TEXT
		);
	`

	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("Failed to create test tables: %v", err)
	}

	return db
}

func TestSettingsRepository_IsPublicRegistrationEnabled_Default(t *testing.T) {
	db := setupSettingsTestDB(t)
	defer db.Close()

	repo := NewSettingsRepository(db)

	// No setting in DB — should default to true
	enabled, err := repo.IsPublicRegistrationEnabled(context.Background())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !enabled {
		t.Error("Expected registration to be enabled by default")
	}
}

func TestSettingsRepository_IsPublicRegistrationEnabled_SetTrue(t *testing.T) {
	db := setupSettingsTestDB(t)
	defer db.Close()

	_, err := db.Exec(`INSERT INTO system_settings (key, value, updated_at) VALUES (?, ?, ?)`,
		"public_registration_enabled", "true", time.Now())
	if err != nil {
		t.Fatalf("Failed to insert setting: %v", err)
	}

	repo := NewSettingsRepository(db)

	enabled, err := repo.IsPublicRegistrationEnabled(context.Background())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !enabled {
		t.Error("Expected registration to be enabled")
	}
}

func TestSettingsRepository_IsPublicRegistrationEnabled_SetFalse(t *testing.T) {
	db := setupSettingsTestDB(t)
	defer db.Close()

	_, err := db.Exec(`INSERT INTO system_settings (key, value, updated_at) VALUES (?, ?, ?)`,
		"public_registration_enabled", "false", time.Now())
	if err != nil {
		t.Fatalf("Failed to insert setting: %v", err)
	}

	repo := NewSettingsRepository(db)

	enabled, err := repo.IsPublicRegistrationEnabled(context.Background())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if enabled {
		t.Error("Expected registration to be disabled")
	}
}

func TestSettingsRepository_IsPublicRegistrationEnabled_EnvVarOverride(t *testing.T) {
	db := setupSettingsTestDB(t)
	defer db.Close()

	// DB says enabled
	_, err := db.Exec(`INSERT INTO system_settings (key, value, updated_at) VALUES (?, ?, ?)`,
		"public_registration_enabled", "true", time.Now())
	if err != nil {
		t.Fatalf("Failed to insert setting: %v", err)
	}

	// Env var overrides to disabled
	os.Setenv("DISABLE_PUBLIC_REGISTRATION", "true")
	defer os.Unsetenv("DISABLE_PUBLIC_REGISTRATION")

	repo := NewSettingsRepository(db)

	enabled, err := repo.IsPublicRegistrationEnabled(context.Background())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if enabled {
		t.Error("Expected registration to be disabled by env var override")
	}
}

func TestSettingsRepository_IsPublicRegistrationEnabled_EnvVarNotTrue(t *testing.T) {
	db := setupSettingsTestDB(t)
	defer db.Close()

	// DB says enabled
	_, err := db.Exec(`INSERT INTO system_settings (key, value, updated_at) VALUES (?, ?, ?)`,
		"public_registration_enabled", "true", time.Now())
	if err != nil {
		t.Fatalf("Failed to insert setting: %v", err)
	}

	// Env var set but not "true" — should not override
	os.Setenv("DISABLE_PUBLIC_REGISTRATION", "false")
	defer os.Unsetenv("DISABLE_PUBLIC_REGISTRATION")

	repo := NewSettingsRepository(db)

	enabled, err := repo.IsPublicRegistrationEnabled(context.Background())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !enabled {
		t.Error("Expected registration to remain enabled when env var is not 'true'")
	}
}

func TestSettingsRepository_GetAndUpdate(t *testing.T) {
	db := setupSettingsTestDB(t)
	defer db.Close()

	repo := NewSettingsRepository(db)
	ctx := context.Background()

	// Get non-existent setting
	_, err := repo.Get(ctx, "nonexistent")
	if err == nil {
		t.Fatal("Expected error for non-existent setting")
	}

	// Create via Update (upsert)
	userID := "admin-1"
	err = repo.Update(ctx, "test_key", "test_value", &userID)
	if err != nil {
		t.Fatalf("Failed to update setting: %v", err)
	}

	// Get created setting
	setting, err := repo.Get(ctx, "test_key")
	if err != nil {
		t.Fatalf("Failed to get setting: %v", err)
	}
	if setting.Value != "test_value" {
		t.Errorf("Expected value 'test_value', got '%s'", setting.Value)
	}

	// Update existing setting
	err = repo.Update(ctx, "test_key", "updated_value", &userID)
	if err != nil {
		t.Fatalf("Failed to update setting: %v", err)
	}

	setting, err = repo.Get(ctx, "test_key")
	if err != nil {
		t.Fatalf("Failed to get updated setting: %v", err)
	}
	if setting.Value != "updated_value" {
		t.Errorf("Expected value 'updated_value', got '%s'", setting.Value)
	}
}

func TestSettingsRepository_GetAll(t *testing.T) {
	db := setupSettingsTestDB(t)
	defer db.Close()

	repo := NewSettingsRepository(db)
	ctx := context.Background()

	// Empty
	settings, err := repo.GetAll(ctx)
	if err != nil {
		t.Fatalf("Failed to get all settings: %v", err)
	}
	if len(settings) != 0 {
		t.Errorf("Expected 0 settings, got %d", len(settings))
	}

	// Insert two settings
	userID := "admin-1"
	if err := repo.Update(ctx, "key_a", "value_a", &userID); err != nil {
		t.Fatalf("Failed to insert key_a: %v", err)
	}
	if err := repo.Update(ctx, "key_b", "value_b", &userID); err != nil {
		t.Fatalf("Failed to insert key_b: %v", err)
	}

	settings, err = repo.GetAll(ctx)
	if err != nil {
		t.Fatalf("Failed to get all settings: %v", err)
	}
	if len(settings) != 2 {
		t.Errorf("Expected 2 settings, got %d", len(settings))
	}
}

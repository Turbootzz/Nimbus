package services

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/nimbus/backend/internal/repository"

	_ "github.com/mattn/go-sqlite3"
)

func setupEmailTestDB(t *testing.T) *sql.DB {
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

func TestGetSMTPConfig_FromDatabase(t *testing.T) {
	db := setupEmailTestDB(t)
	defer db.Close()

	// Clear env vars
	os.Unsetenv("SMTP_HOST")
	os.Unsetenv("SMTP_PORT")
	os.Unsetenv("SMTP_USERNAME")
	os.Unsetenv("SMTP_PASSWORD")
	os.Unsetenv("SMTP_FROM_EMAIL")
	os.Unsetenv("SMTP_FROM_NAME")

	// Insert DB settings
	settings := map[string]string{
		"smtp_host":       "db.smtp.com",
		"smtp_port":       "465",
		"smtp_username":   "dbuser",
		"smtp_password":   "dbpass",
		"smtp_from_email": "db@example.com",
		"smtp_from_name":  "DBName",
		"smtp_enabled":    "true",
	}
	for k, v := range settings {
		_, err := db.Exec("INSERT INTO system_settings (key, value) VALUES (?, ?)", k, v)
		if err != nil {
			t.Fatalf("Failed to insert setting %s: %v", k, err)
		}
	}

	repo := repository.NewSettingsRepository(db)
	svc := NewEmailService(repo)

	config, err := svc.GetSMTPConfig(context.Background())
	if err != nil {
		t.Fatalf("GetSMTPConfig failed: %v", err)
	}

	if config.Host != "db.smtp.com" {
		t.Errorf("Expected host 'db.smtp.com', got '%s'", config.Host)
	}
	if config.Port != 465 {
		t.Errorf("Expected port 465, got %d", config.Port)
	}
	if config.Username != "dbuser" {
		t.Errorf("Expected username 'dbuser', got '%s'", config.Username)
	}
	if !config.Enabled {
		t.Error("Expected enabled to be true")
	}
}

func TestGetSMTPConfig_FromEnvVars(t *testing.T) {
	db := setupEmailTestDB(t)
	defer db.Close()

	os.Setenv("SMTP_HOST", "env.smtp.com")
	os.Setenv("SMTP_PORT", "587")
	os.Setenv("SMTP_USERNAME", "envuser")
	os.Setenv("SMTP_PASSWORD", "envpass")
	os.Setenv("SMTP_FROM_EMAIL", "env@example.com")
	os.Setenv("SMTP_FROM_NAME", "EnvName")
	defer func() {
		os.Unsetenv("SMTP_HOST")
		os.Unsetenv("SMTP_PORT")
		os.Unsetenv("SMTP_USERNAME")
		os.Unsetenv("SMTP_PASSWORD")
		os.Unsetenv("SMTP_FROM_EMAIL")
		os.Unsetenv("SMTP_FROM_NAME")
	}()

	repo := repository.NewSettingsRepository(db)
	svc := NewEmailService(repo)

	config, err := svc.GetSMTPConfig(context.Background())
	if err != nil {
		t.Fatalf("GetSMTPConfig failed: %v", err)
	}

	if config.Host != "env.smtp.com" {
		t.Errorf("Expected host 'env.smtp.com', got '%s'", config.Host)
	}
	if config.Username != "envuser" {
		t.Errorf("Expected username 'envuser', got '%s'", config.Username)
	}
	if !config.Enabled {
		t.Error("Expected auto-enabled when host is set via env")
	}
}

func TestGetSMTPConfig_DefaultFromName(t *testing.T) {
	db := setupEmailTestDB(t)
	defer db.Close()

	os.Unsetenv("SMTP_HOST")
	os.Unsetenv("SMTP_FROM_NAME")

	repo := repository.NewSettingsRepository(db)
	svc := NewEmailService(repo)

	config, err := svc.GetSMTPConfig(context.Background())
	if err != nil {
		t.Fatalf("GetSMTPConfig failed: %v", err)
	}

	if config.FromName != "Nimbus" {
		t.Errorf("Expected default FromName 'Nimbus', got '%s'", config.FromName)
	}
}

func TestGetSMTPStatus_None(t *testing.T) {
	db := setupEmailTestDB(t)
	defer db.Close()

	os.Unsetenv("SMTP_HOST")

	repo := repository.NewSettingsRepository(db)
	svc := NewEmailService(repo)

	status := svc.GetSMTPStatus(context.Background())
	if status.Source != "none" {
		t.Errorf("Expected source 'none', got '%s'", status.Source)
	}
	if status.Configured {
		t.Error("Expected configured false")
	}
}

func TestGetSMTPStatus_Env(t *testing.T) {
	db := setupEmailTestDB(t)
	defer db.Close()

	os.Setenv("SMTP_HOST", "smtp.test.com")
	defer os.Unsetenv("SMTP_HOST")

	repo := repository.NewSettingsRepository(db)
	svc := NewEmailService(repo)

	status := svc.GetSMTPStatus(context.Background())
	if status.Source != "env" {
		t.Errorf("Expected source 'env', got '%s'", status.Source)
	}
	if !status.Configured {
		t.Error("Expected configured true")
	}
}

func TestGetSMTPStatus_Database(t *testing.T) {
	db := setupEmailTestDB(t)
	defer db.Close()

	os.Unsetenv("SMTP_HOST")

	_, err := db.Exec("INSERT INTO system_settings (key, value) VALUES (?, ?)", "smtp_host", "smtp.db.com")
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	repo := repository.NewSettingsRepository(db)
	svc := NewEmailService(repo)

	status := svc.GetSMTPStatus(context.Background())
	if status.Source != "database" {
		t.Errorf("Expected source 'database', got '%s'", status.Source)
	}
}

func TestSanitizeHeader(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"clean string", "Hello World", "Hello World"},
		{"with CR", "Hello\rWorld", "HelloWorld"},
		{"with LF", "Hello\nWorld", "HelloWorld"},
		{"with CRLF", "Hello\r\nWorld", "HelloWorld"},
		{"injection attempt", "test\r\nBcc: attacker@evil.com", "testBcc: attacker@evil.com"},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeHeader(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeHeader(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestTestConnectionWithConfig_NotConfigured(t *testing.T) {
	db := setupEmailTestDB(t)
	defer db.Close()

	repo := repository.NewSettingsRepository(db)
	svc := NewEmailService(repo)

	err := svc.TestConnectionWithConfig(&SMTPConfig{})
	if err == nil {
		t.Error("Expected error for unconfigured SMTP")
	}
	if err.Error() != "SMTP is not configured" {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestTestConnectionWithConfig_DisabledSMTP(t *testing.T) {
	db := setupEmailTestDB(t)
	defer db.Close()

	repo := repository.NewSettingsRepository(db)
	svc := NewEmailService(repo)

	err := svc.TestConnectionWithConfig(&SMTPConfig{
		Host:    "smtp.example.com",
		Enabled: false,
	})
	if err == nil {
		t.Error("Expected error for disabled SMTP")
	}
}

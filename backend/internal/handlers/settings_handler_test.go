package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/repository"

	_ "github.com/mattn/go-sqlite3"
)

// setupSettingsTestDB creates an in-memory SQLite database for settings testing
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

		CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			email TEXT UNIQUE NOT NULL,
			name TEXT NOT NULL,
			password TEXT,
			role TEXT NOT NULL DEFAULT 'user',
			provider TEXT NOT NULL DEFAULT 'local',
			provider_id TEXT,
			avatar_url TEXT,
			email_verified INTEGER NOT NULL DEFAULT 0,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL,
			last_activity_at TIMESTAMP
		);
	`

	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("Failed to create test tables: %v", err)
	}

	return db
}

func TestSettingsHandler_GetSettings_Empty(t *testing.T) {
	db := setupSettingsTestDB(t)
	defer db.Close()

	settingsRepo := repository.NewSettingsRepository(db)
	handler := NewSettingsHandler(settingsRepo)

	app := fiber.New()
	app.Get("/admin/settings", func(c *fiber.Ctx) error {
		c.Locals("user_id", "admin-1")
		return handler.GetSettings(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/admin/settings", nil)

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// When no settings exist, the response can be either null or empty array
	settings := response["settings"]
	if settings == nil {
		// This is acceptable - no settings exist
		return
	}

	settingsArray, ok := settings.([]interface{})
	if !ok {
		t.Fatal("Expected settings to be array or null")
	}

	if len(settingsArray) != 0 {
		t.Errorf("Expected empty settings array, got %d items", len(settingsArray))
	}
}

func TestSettingsHandler_GetSettings_WithData(t *testing.T) {
	db := setupSettingsTestDB(t)
	defer db.Close()

	// Insert test settings
	_, err := db.Exec(`
		INSERT INTO system_settings (key, value, updated_at)
		VALUES (?, ?, ?)
	`, "public_registration_enabled", "true", time.Now())
	if err != nil {
		t.Fatalf("Failed to insert setting: %v", err)
	}

	settingsRepo := repository.NewSettingsRepository(db)
	handler := NewSettingsHandler(settingsRepo)

	app := fiber.New()
	app.Get("/admin/settings", func(c *fiber.Ctx) error {
		c.Locals("user_id", "admin-1")
		return handler.GetSettings(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/admin/settings", nil)

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	settings, ok := response["settings"].([]interface{})
	if !ok {
		t.Fatal("Expected settings array in response")
	}

	if len(settings) != 1 {
		t.Errorf("Expected 1 setting, got %d", len(settings))
	}
}

func TestSettingsHandler_GetSetting_Found(t *testing.T) {
	db := setupSettingsTestDB(t)
	defer db.Close()

	// Insert test setting
	_, err := db.Exec(`
		INSERT INTO system_settings (key, value, updated_at)
		VALUES (?, ?, ?)
	`, "public_registration_enabled", "true", time.Now())
	if err != nil {
		t.Fatalf("Failed to insert setting: %v", err)
	}

	settingsRepo := repository.NewSettingsRepository(db)
	handler := NewSettingsHandler(settingsRepo)

	app := fiber.New()
	app.Get("/admin/settings/:key", func(c *fiber.Ctx) error {
		c.Locals("user_id", "admin-1")
		return handler.GetSetting(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/admin/settings/public_registration_enabled", nil)

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	var setting models.SystemSetting
	if err := json.NewDecoder(resp.Body).Decode(&setting); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if setting.Key != "public_registration_enabled" {
		t.Errorf("Expected key 'public_registration_enabled', got '%s'", setting.Key)
	}

	if setting.Value != "true" {
		t.Errorf("Expected value 'true', got '%s'", setting.Value)
	}
}

func TestSettingsHandler_GetSetting_NotFound(t *testing.T) {
	db := setupSettingsTestDB(t)
	defer db.Close()

	settingsRepo := repository.NewSettingsRepository(db)
	handler := NewSettingsHandler(settingsRepo)

	app := fiber.New()
	app.Get("/admin/settings/:key", func(c *fiber.Ctx) error {
		c.Locals("user_id", "admin-1")
		return handler.GetSetting(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/admin/settings/nonexistent_key", nil)

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, resp.StatusCode)
	}
}

func TestSettingsHandler_UpdateSetting_Create(t *testing.T) {
	db := setupSettingsTestDB(t)
	defer db.Close()

	settingsRepo := repository.NewSettingsRepository(db)
	handler := NewSettingsHandler(settingsRepo)

	app := fiber.New()
	app.Put("/admin/settings/:key", func(c *fiber.Ctx) error {
		c.Locals("user_id", "admin-1")
		return handler.UpdateSetting(c)
	})

	updateReq := models.UpdateSettingRequest{
		Value: "false",
	}

	bodyJSON, _ := json.Marshal(updateReq)
	req := httptest.NewRequest(http.MethodPut, "/admin/settings/public_registration_enabled", bytes.NewReader(bodyJSON))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	// Verify setting was created
	var value string
	err = db.QueryRow("SELECT value FROM system_settings WHERE key = ?", "public_registration_enabled").Scan(&value)
	if err != nil {
		t.Fatalf("Failed to query setting: %v", err)
	}

	if value != "false" {
		t.Errorf("Expected value 'false', got '%s'", value)
	}
}

func TestSettingsHandler_UpdateSetting_Update(t *testing.T) {
	db := setupSettingsTestDB(t)
	defer db.Close()

	// Insert existing setting
	_, err := db.Exec(`
		INSERT INTO system_settings (key, value, updated_at)
		VALUES (?, ?, ?)
	`, "public_registration_enabled", "true", time.Now())
	if err != nil {
		t.Fatalf("Failed to insert setting: %v", err)
	}

	settingsRepo := repository.NewSettingsRepository(db)
	handler := NewSettingsHandler(settingsRepo)

	app := fiber.New()
	app.Put("/admin/settings/:key", func(c *fiber.Ctx) error {
		c.Locals("user_id", "admin-1")
		return handler.UpdateSetting(c)
	})

	updateReq := models.UpdateSettingRequest{
		Value: "false",
	}

	bodyJSON, _ := json.Marshal(updateReq)
	req := httptest.NewRequest(http.MethodPut, "/admin/settings/public_registration_enabled", bytes.NewReader(bodyJSON))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	// Verify setting was updated
	var value string
	err = db.QueryRow("SELECT value FROM system_settings WHERE key = ?", "public_registration_enabled").Scan(&value)
	if err != nil {
		t.Fatalf("Failed to query setting: %v", err)
	}

	if value != "false" {
		t.Errorf("Expected value 'false', got '%s'", value)
	}
}

func TestSettingsHandler_UpdateSetting_InvalidBooleanValue(t *testing.T) {
	db := setupSettingsTestDB(t)
	defer db.Close()

	settingsRepo := repository.NewSettingsRepository(db)
	handler := NewSettingsHandler(settingsRepo)

	app := fiber.New()
	app.Put("/admin/settings/:key", func(c *fiber.Ctx) error {
		c.Locals("user_id", "admin-1")
		return handler.UpdateSetting(c)
	})

	updateReq := models.UpdateSettingRequest{
		Value: "invalid", // Not "true" or "false"
	}

	bodyJSON, _ := json.Marshal(updateReq)
	req := httptest.NewRequest(http.MethodPut, "/admin/settings/public_registration_enabled", bytes.NewReader(bodyJSON))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status %d for invalid boolean value, got %d", http.StatusBadRequest, resp.StatusCode)
	}
}

func TestSettingsHandler_UpdateSetting_InvalidJSON(t *testing.T) {
	db := setupSettingsTestDB(t)
	defer db.Close()

	settingsRepo := repository.NewSettingsRepository(db)
	handler := NewSettingsHandler(settingsRepo)

	app := fiber.New()
	app.Put("/admin/settings/:key", func(c *fiber.Ctx) error {
		c.Locals("user_id", "admin-1")
		return handler.UpdateSetting(c)
	})

	req := httptest.NewRequest(http.MethodPut, "/admin/settings/public_registration_enabled", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status %d for invalid JSON, got %d", http.StatusBadRequest, resp.StatusCode)
	}
}

func TestSettingsHandler_UpdateSetting_NoAuth(t *testing.T) {
	db := setupSettingsTestDB(t)
	defer db.Close()

	settingsRepo := repository.NewSettingsRepository(db)
	handler := NewSettingsHandler(settingsRepo)

	app := fiber.New()
	// Not setting user_id in locals
	app.Put("/admin/settings/:key", handler.UpdateSetting)

	updateReq := models.UpdateSettingRequest{
		Value: "false",
	}

	bodyJSON, _ := json.Marshal(updateReq)
	req := httptest.NewRequest(http.MethodPut, "/admin/settings/public_registration_enabled", bytes.NewReader(bodyJSON))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status %d for unauthenticated request, got %d", http.StatusUnauthorized, resp.StatusCode)
	}
}

func TestSettingsHandler_GetPublicRegistrationStatus_Enabled(t *testing.T) {
	db := setupSettingsTestDB(t)
	defer db.Close()

	// Insert setting with registration enabled
	_, err := db.Exec(`
		INSERT INTO system_settings (key, value, updated_at)
		VALUES (?, ?, ?)
	`, "public_registration_enabled", "true", time.Now())
	if err != nil {
		t.Fatalf("Failed to insert setting: %v", err)
	}

	settingsRepo := repository.NewSettingsRepository(db)
	handler := NewSettingsHandler(settingsRepo)

	app := fiber.New()
	app.Get("/setup/registration-status", handler.GetPublicRegistrationStatus)

	req := httptest.NewRequest(http.MethodGet, "/setup/registration-status", nil)

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	var response map[string]bool
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if !response["enabled"] {
		t.Error("Expected enabled to be true")
	}
}

func TestSettingsHandler_GetPublicRegistrationStatus_Disabled(t *testing.T) {
	db := setupSettingsTestDB(t)
	defer db.Close()

	// Insert setting with registration disabled
	_, err := db.Exec(`
		INSERT INTO system_settings (key, value, updated_at)
		VALUES (?, ?, ?)
	`, "public_registration_enabled", "false", time.Now())
	if err != nil {
		t.Fatalf("Failed to insert setting: %v", err)
	}

	settingsRepo := repository.NewSettingsRepository(db)
	handler := NewSettingsHandler(settingsRepo)

	app := fiber.New()
	app.Get("/setup/registration-status", handler.GetPublicRegistrationStatus)

	req := httptest.NewRequest(http.MethodGet, "/setup/registration-status", nil)

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	var response map[string]bool
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["enabled"] {
		t.Error("Expected enabled to be false")
	}
}

func TestSettingsHandler_GetPublicRegistrationStatus_NoSetting(t *testing.T) {
	db := setupSettingsTestDB(t)
	defer db.Close()

	// No setting inserted - should default to true
	settingsRepo := repository.NewSettingsRepository(db)
	handler := NewSettingsHandler(settingsRepo)

	app := fiber.New()
	app.Get("/setup/registration-status", handler.GetPublicRegistrationStatus)

	req := httptest.NewRequest(http.MethodGet, "/setup/registration-status", nil)

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	var response map[string]bool
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Default should be true when setting doesn't exist
	if !response["enabled"] {
		t.Error("Expected enabled to default to true when setting doesn't exist")
	}
}

func TestSettingsHandler_UpdateSetting_BlockedByEnvVar(t *testing.T) {
	db := setupSettingsTestDB(t)
	defer db.Close()

	// Set env var to disable registration
	os.Setenv("DISABLE_PUBLIC_REGISTRATION", "true")
	defer os.Unsetenv("DISABLE_PUBLIC_REGISTRATION")

	settingsRepo := repository.NewSettingsRepository(db)
	handler := NewSettingsHandler(settingsRepo)

	app := fiber.New()
	app.Put("/admin/settings/:key", func(c *fiber.Ctx) error {
		c.Locals("user_id", "admin-1")
		return handler.UpdateSetting(c)
	})

	// Try to enable registration while env var blocks it
	updateReq := models.UpdateSettingRequest{
		Value: "true",
	}

	bodyJSON, _ := json.Marshal(updateReq)
	req := httptest.NewRequest(http.MethodPut, "/admin/settings/public_registration_enabled", bytes.NewReader(bodyJSON))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}

	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("Expected status %d when env var blocks re-enable, got %d", http.StatusForbidden, resp.StatusCode)
	}

	var response map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["error"] != "Registration is disabled by server configuration" {
		t.Errorf("Expected server configuration error, got '%s'", response["error"])
	}
}

func TestSettingsHandler_UpdateSetting_AllowDisableWithEnvVar(t *testing.T) {
	db := setupSettingsTestDB(t)
	defer db.Close()

	// Set env var to disable registration
	os.Setenv("DISABLE_PUBLIC_REGISTRATION", "true")
	defer os.Unsetenv("DISABLE_PUBLIC_REGISTRATION")

	settingsRepo := repository.NewSettingsRepository(db)
	handler := NewSettingsHandler(settingsRepo)

	app := fiber.New()
	app.Put("/admin/settings/:key", func(c *fiber.Ctx) error {
		c.Locals("user_id", "admin-1")
		return handler.UpdateSetting(c)
	})

	// Setting to "false" should still work even with env var
	updateReq := models.UpdateSettingRequest{
		Value: "false",
	}

	bodyJSON, _ := json.Marshal(updateReq)
	req := httptest.NewRequest(http.MethodPut, "/admin/settings/public_registration_enabled", bytes.NewReader(bodyJSON))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d when disabling (not blocked), got %d", http.StatusOK, resp.StatusCode)
	}
}

func TestSettingsHandler_GetPublicRegistrationStatus_EnvVarOverride(t *testing.T) {
	db := setupSettingsTestDB(t)
	defer db.Close()

	// Insert setting with registration enabled
	_, err := db.Exec(`
		INSERT INTO system_settings (key, value, updated_at)
		VALUES (?, ?, ?)
	`, "public_registration_enabled", "true", time.Now())
	if err != nil {
		t.Fatalf("Failed to insert setting: %v", err)
	}

	// Env var should override the DB setting
	os.Setenv("DISABLE_PUBLIC_REGISTRATION", "true")
	defer os.Unsetenv("DISABLE_PUBLIC_REGISTRATION")

	settingsRepo := repository.NewSettingsRepository(db)
	handler := NewSettingsHandler(settingsRepo)

	app := fiber.New()
	app.Get("/setup/registration-status", handler.GetPublicRegistrationStatus)

	req := httptest.NewRequest(http.MethodGet, "/setup/registration-status", nil)

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	var response map[string]bool
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["enabled"] {
		t.Error("Expected enabled to be false when DISABLE_PUBLIC_REGISTRATION is set")
	}
}

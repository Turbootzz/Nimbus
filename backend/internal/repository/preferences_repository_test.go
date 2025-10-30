package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/nimbus/backend/internal/models"
)

// setupPreferencesTestDB creates an in-memory SQLite database with users and preferences tables
func setupPreferencesTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Create users table (needed for foreign key)
	usersSchema := `
		CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			email TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL,
			password TEXT NOT NULL,
			role TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL
		);
	`

	if _, err := db.Exec(usersSchema); err != nil {
		t.Fatalf("Failed to create users table: %v", err)
	}

	// Create preferences table
	// Note: SQLite doesn't have UUID type, so we generate IDs manually or use user_id as implicit key
	preferencesSchema := `
		CREATE TABLE IF NOT EXISTS user_preferences (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
			user_id TEXT NOT NULL UNIQUE,
			theme_mode TEXT NOT NULL DEFAULT 'light' CHECK (theme_mode IN ('light', 'dark')),
			theme_background TEXT,
			theme_accent_color TEXT,
			open_in_new_tab BOOLEAN NOT NULL DEFAULT 1,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		);
	`

	if _, err := db.Exec(preferencesSchema); err != nil {
		t.Fatalf("Failed to create preferences table: %v", err)
	}

	// Create test user
	_, err = db.Exec(`
		INSERT INTO users (id, email, name, password, role, created_at, updated_at)
		VALUES ('user-1', 'test@example.com', 'Test User', 'hashed', 'user', ?, ?)
	`, time.Now(), time.Now())
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	return db
}

func TestPreferencesRepository_Create(t *testing.T) {
	db := setupPreferencesTestDB(t)
	defer db.Close()

	lightMode := "light"
	accentColor := "#3B82F6"

	// Insert preferences directly (bypassing Create method due to SQLite RETURNING limitation)
	_, err := db.Exec(`
		INSERT INTO user_preferences (id, user_id, theme_mode, theme_background, theme_accent_color, created_at, updated_at)
		VALUES ('pref-1', 'user-1', ?, NULL, ?, ?, ?)
	`, lightMode, accentColor, time.Now(), time.Now())
	if err != nil {
		t.Fatalf("Failed to insert preferences: %v", err)
	}

	// Verify preferences were created by counting rows
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM user_preferences WHERE user_id = 'user-1'").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count preferences: %v", err)
	}
	if count != 1 {
		t.Errorf("Create() created %d rows, want 1", count)
	}

	// Verify values
	var storedMode, storedColor string
	err = db.QueryRow("SELECT theme_mode, theme_accent_color FROM user_preferences WHERE user_id = 'user-1'").Scan(&storedMode, &storedColor)
	if err != nil {
		t.Fatalf("Failed to retrieve created preferences: %v", err)
	}
	if storedMode != lightMode {
		t.Errorf("ThemeMode = %v, want %v", storedMode, lightMode)
	}
	if storedColor != accentColor {
		t.Errorf("ThemeAccentColor = %v, want %v", storedColor, accentColor)
	}
}

func TestPreferencesRepository_GetByUserID(t *testing.T) {
	db := setupPreferencesTestDB(t)
	defer db.Close()

	repo := NewPreferencesRepository(db)
	ctx := context.Background()

	// Insert test preferences directly
	accentColor := "#3B82F6"
	background := "https://example.com/bg.jpg"
	_, err := db.Exec(`
		INSERT INTO user_preferences (id, user_id, theme_mode, theme_background, theme_accent_color, created_at, updated_at)
		VALUES ('pref-1', 'user-1', 'dark', ?, ?, ?, ?)
	`, background, accentColor, time.Now(), time.Now())
	if err != nil {
		t.Fatalf("Failed to insert test preferences: %v", err)
	}

	tests := []struct {
		name      string
		userID    string
		wantErr   bool
		expectNil bool
	}{
		{
			name:      "Get existing preferences",
			userID:    "user-1",
			wantErr:   false,
			expectNil: false,
		},
		{
			name:      "Get non-existent preferences",
			userID:    "user-999",
			wantErr:   true,
			expectNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			preferences, err := repo.GetByUserID(ctx, tt.userID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetByUserID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.expectNil && preferences != nil {
				t.Errorf("GetByUserID() expected nil preferences, got %v", preferences)
			}
			if !tt.expectNil && preferences == nil {
				t.Error("GetByUserID() expected preferences, got nil")
			}
			if !tt.expectNil && preferences != nil {
				if preferences.UserID != tt.userID {
					t.Errorf("GetByUserID() UserID = %v, want %v", preferences.UserID, tt.userID)
				}
				if preferences.ThemeMode != "dark" {
					t.Errorf("GetByUserID() ThemeMode = %v, want dark", preferences.ThemeMode)
				}
				if preferences.ThemeBackground == nil || *preferences.ThemeBackground != background {
					t.Errorf("GetByUserID() ThemeBackground = %v, want %v", preferences.ThemeBackground, background)
				}
				if preferences.ThemeAccentColor == nil || *preferences.ThemeAccentColor != accentColor {
					t.Errorf("GetByUserID() ThemeAccentColor = %v, want %v", preferences.ThemeAccentColor, accentColor)
				}
			}
		})
	}
}

func TestPreferencesRepository_Update(t *testing.T) {
	db := setupPreferencesTestDB(t)
	defer db.Close()

	repo := NewPreferencesRepository(db)
	ctx := context.Background()

	// Insert initial preferences
	_, err := db.Exec(`
		INSERT INTO user_preferences (id, user_id, theme_mode, theme_background, theme_accent_color, created_at, updated_at)
		VALUES ('pref-1', 'user-1', 'light', NULL, NULL, ?, ?)
	`, time.Now(), time.Now())
	if err != nil {
		t.Fatalf("Failed to insert test preferences: %v", err)
	}

	newAccentColor := "#EF4444"
	newBackground := "https://example.com/new-bg.jpg"

	tests := []struct {
		name    string
		userID  string
		req     *models.PreferencesUpdateRequest
		wantErr bool
	}{
		{
			name:   "Update existing preferences",
			userID: "user-1",
			req: &models.PreferencesUpdateRequest{
				ThemeMode:        "dark",
				ThemeBackground:  &newBackground,
				ThemeAccentColor: &newAccentColor,
			},
			wantErr: false,
		},
		{
			name:   "Update non-existent preferences",
			userID: "user-999",
			req: &models.PreferencesUpdateRequest{
				ThemeMode: "dark",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.Update(ctx, tt.userID, tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Update() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				// Verify update was applied
				preferences, err := repo.GetByUserID(ctx, tt.userID)
				if err != nil {
					t.Fatalf("Failed to retrieve updated preferences: %v", err)
				}
				if preferences.ThemeMode != tt.req.ThemeMode {
					t.Errorf("Update() ThemeMode = %v, want %v", preferences.ThemeMode, tt.req.ThemeMode)
				}
				if tt.req.ThemeBackground != nil {
					if preferences.ThemeBackground == nil || *preferences.ThemeBackground != *tt.req.ThemeBackground {
						t.Errorf("Update() ThemeBackground = %v, want %v", preferences.ThemeBackground, tt.req.ThemeBackground)
					}
				}
				if tt.req.ThemeAccentColor != nil {
					if preferences.ThemeAccentColor == nil || *preferences.ThemeAccentColor != *tt.req.ThemeAccentColor {
						t.Errorf("Update() ThemeAccentColor = %v, want %v", preferences.ThemeAccentColor, tt.req.ThemeAccentColor)
					}
				}
			}
		})
	}
}

func TestPreferencesRepository_Upsert(t *testing.T) {
	db := setupPreferencesTestDB(t)
	defer db.Close()

	repo := NewPreferencesRepository(db)
	ctx := context.Background()

	// Create second test user
	_, err := db.Exec(`
		INSERT INTO users (id, email, name, password, role, created_at, updated_at)
		VALUES ('user-2', 'test2@example.com', 'Test User 2', 'hashed', 'user', ?, ?)
	`, time.Now(), time.Now())
	if err != nil {
		t.Fatalf("Failed to create test user 2: %v", err)
	}

	// Insert initial preferences for user-1
	_, err = db.Exec(`
		INSERT INTO user_preferences (id, user_id, theme_mode, theme_background, theme_accent_color, created_at, updated_at)
		VALUES ('pref-1', 'user-1', 'light', NULL, NULL, ?, ?)
	`, time.Now(), time.Now())
	if err != nil {
		t.Fatalf("Failed to insert test preferences: %v", err)
	}

	accentColor1 := "#3B82F6"
	accentColor2 := "#10B981"

	tests := []struct {
		name    string
		userID  string
		req     *models.PreferencesUpdateRequest
		isNew   bool
		wantErr bool
	}{
		{
			name:   "Upsert for existing user (update)",
			userID: "user-1",
			req: &models.PreferencesUpdateRequest{
				ThemeMode:        "dark",
				ThemeAccentColor: &accentColor1,
			},
			isNew:   false,
			wantErr: false,
		},
		{
			name:   "Upsert for new user (insert)",
			userID: "user-2",
			req: &models.PreferencesUpdateRequest{
				ThemeMode:        "light",
				ThemeAccentColor: &accentColor2,
			},
			isNew:   true,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.Upsert(ctx, tt.userID, tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Upsert() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				// Verify upsert was applied
				preferences, err := repo.GetByUserID(ctx, tt.userID)
				if err != nil {
					t.Fatalf("Failed to retrieve upserted preferences: %v", err)
				}
				if preferences.ThemeMode != tt.req.ThemeMode {
					t.Errorf("Upsert() ThemeMode = %v, want %v", preferences.ThemeMode, tt.req.ThemeMode)
				}
				if tt.req.ThemeAccentColor != nil {
					if preferences.ThemeAccentColor == nil || *preferences.ThemeAccentColor != *tt.req.ThemeAccentColor {
						t.Errorf("Upsert() ThemeAccentColor = %v, want %v", preferences.ThemeAccentColor, tt.req.ThemeAccentColor)
					}
				}
			}
		})
	}

	// Verify user-1 was updated (not duplicated)
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM user_preferences WHERE user_id = 'user-1'").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count preferences: %v", err)
	}
	if count != 1 {
		t.Errorf("Upsert created duplicate row, got %d rows for user-1, want 1", count)
	}
}

func TestPreferencesRepository_Upsert_MultipleUpdates(t *testing.T) {
	db := setupPreferencesTestDB(t)
	defer db.Close()

	repo := NewPreferencesRepository(db)
	ctx := context.Background()

	accentColor1 := "#3B82F6"
	accentColor2 := "#EF4444"
	accentColor3 := "#10B981"

	// First upsert (insert)
	req1 := &models.PreferencesUpdateRequest{
		ThemeMode:        "light",
		ThemeAccentColor: &accentColor1,
	}
	err := repo.Upsert(ctx, "user-1", req1)
	if err != nil {
		t.Fatalf("First Upsert() failed: %v", err)
	}

	// Second upsert (update)
	req2 := &models.PreferencesUpdateRequest{
		ThemeMode:        "dark",
		ThemeAccentColor: &accentColor2,
	}
	err = repo.Upsert(ctx, "user-1", req2)
	if err != nil {
		t.Fatalf("Second Upsert() failed: %v", err)
	}

	// Third upsert (update)
	req3 := &models.PreferencesUpdateRequest{
		ThemeMode:        "light",
		ThemeAccentColor: &accentColor3,
	}
	err = repo.Upsert(ctx, "user-1", req3)
	if err != nil {
		t.Fatalf("Third Upsert() failed: %v", err)
	}

	// Verify only one row exists
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM user_preferences WHERE user_id = 'user-1'").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count preferences: %v", err)
	}
	if count != 1 {
		t.Errorf("Multiple Upserts created %d rows, want 1", count)
	}

	// Verify final state
	preferences, err := repo.GetByUserID(ctx, "user-1")
	if err != nil {
		t.Fatalf("GetByUserID() failed: %v", err)
	}
	if preferences.ThemeMode != "light" {
		t.Errorf("Final ThemeMode = %v, want light", preferences.ThemeMode)
	}
	if preferences.ThemeAccentColor == nil || *preferences.ThemeAccentColor != accentColor3 {
		t.Errorf("Final ThemeAccentColor = %v, want %v", preferences.ThemeAccentColor, accentColor3)
	}
}

func TestPreferencesRepository_NullFields(t *testing.T) {
	db := setupPreferencesTestDB(t)
	defer db.Close()

	repo := NewPreferencesRepository(db)
	ctx := context.Background()

	// Upsert with nil optional fields
	req := &models.PreferencesUpdateRequest{
		ThemeMode:        "dark",
		ThemeBackground:  nil,
		ThemeAccentColor: nil,
	}

	err := repo.Upsert(ctx, "user-1", req)
	if err != nil {
		t.Fatalf("Upsert() failed: %v", err)
	}

	// Retrieve and verify nil fields
	preferences, err := repo.GetByUserID(ctx, "user-1")
	if err != nil {
		t.Fatalf("GetByUserID() failed: %v", err)
	}

	if preferences.ThemeBackground != nil {
		t.Errorf("ThemeBackground = %v, want nil", preferences.ThemeBackground)
	}
	if preferences.ThemeAccentColor != nil {
		t.Errorf("ThemeAccentColor = %v, want nil", preferences.ThemeAccentColor)
	}
	if preferences.ThemeMode != "dark" {
		t.Errorf("ThemeMode = %v, want dark", preferences.ThemeMode)
	}
}

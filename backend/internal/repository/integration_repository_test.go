package repository

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"

	"github.com/nimbus/backend/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/mattn/go-sqlite3"
)

// integrationTestSchema mirrors migration 000026 for SQLite
const integrationTestSchema = `
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY
	);

	CREATE TABLE IF NOT EXISTS integrations (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		kind TEXT NOT NULL,
		name TEXT NOT NULL,
		base_url TEXT NOT NULL,
		auth_type TEXT NOT NULL DEFAULT 'none'
			CHECK (auth_type IN ('none', 'api_key', 'basic', 'token')),
		credentials_enc BLOB,
		verify_tls INTEGER NOT NULL DEFAULT 1,
		options TEXT NOT NULL DEFAULT '{}',
		refresh_seconds INTEGER NOT NULL DEFAULT 60 CHECK (refresh_seconds >= 10),
		last_test_at TIMESTAMP,
		last_test_ok INTEGER,
		last_error TEXT,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	INSERT INTO users (id) VALUES ('user-1'), ('user-2');
`

func setupIntegrationTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	if _, err := db.Exec(integrationTestSchema); err != nil {
		t.Fatalf("Failed to create test tables: %v", err)
	}
	return db
}

func createTestIntegration(t *testing.T, repo *IntegrationRepository, userID, name string, creds []byte) *models.Integration {
	t.Helper()
	integration := &models.Integration{
		UserID:         userID,
		Kind:           "sonarr",
		Name:           name,
		BaseURL:        "http://192.168.1.10:8989",
		AuthType:       models.IntegrationAuthAPIKey,
		VerifyTLS:      true,
		Options:        json.RawMessage(`{"days":7}`),
		RefreshSeconds: 30,
	}
	if err := repo.Create(context.Background(), integration, creds, 50); err != nil {
		t.Fatalf("Failed to create integration: %v", err)
	}
	return integration
}

func TestIntegrationRepository_CreateAndGet(t *testing.T) {
	repo := NewIntegrationRepository(setupIntegrationTestDB(t))
	ctx := context.Background()

	created := createTestIntegration(t, repo, "user-1", "Sonarr", []byte{0x01, 0x02})
	if created.ID == "" || created.CreatedAt.IsZero() || !created.HasCredentials {
		t.Fatalf("Create did not fill ID/CreatedAt/HasCredentials: %+v", created)
	}

	got, err := repo.GetByID(ctx, created.ID, "user-1")
	if err != nil {
		t.Fatalf("GetByID returned error: %v", err)
	}
	if got.Name != "Sonarr" || got.Kind != "sonarr" || got.AuthType != models.IntegrationAuthAPIKey ||
		!got.VerifyTLS || got.RefreshSeconds != 30 || !got.HasCredentials {
		t.Errorf("unexpected integration: %+v", got)
	}
	if string(got.Options) != `{"days":7}` {
		t.Errorf("expected options to round trip, got %s", got.Options)
	}
	if got.LastTestAt != nil || got.LastTestOK != nil || got.LastError != nil {
		t.Errorf("expected no test result yet, got %+v", got)
	}
}

func TestIntegrationRepository_Credentials(t *testing.T) {
	repo := NewIntegrationRepository(setupIntegrationTestDB(t))
	ctx := context.Background()

	withCreds := createTestIntegration(t, repo, "user-1", "With", []byte{0xde, 0xad})
	blob, err := repo.GetCredentials(ctx, withCreds.ID, "user-1")
	if err != nil || !bytes.Equal(blob, []byte{0xde, 0xad}) {
		t.Errorf("expected stored blob, got %x (err %v)", blob, err)
	}

	without := createTestIntegration(t, repo, "user-1", "Without", nil)
	if without.HasCredentials {
		t.Error("expected HasCredentials false for nil blob")
	}
	got, _ := repo.GetByID(ctx, without.ID, "user-1")
	if got.HasCredentials {
		t.Error("nil blob must be stored as NULL, not an empty value")
	}
	blob, err = repo.GetCredentials(ctx, without.ID, "user-1")
	if err != nil || blob != nil {
		t.Errorf("expected nil blob, got %x (err %v)", blob, err)
	}
}

func TestIntegrationRepository_ListByUserID(t *testing.T) {
	repo := NewIntegrationRepository(setupIntegrationTestDB(t))

	createTestIntegration(t, repo, "user-1", "Sonarr", nil)
	createTestIntegration(t, repo, "user-1", "Radarr", nil)
	createTestIntegration(t, repo, "user-2", "Other", nil)

	list, err := repo.ListByUserID(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("ListByUserID returned error: %v", err)
	}
	if len(list) != 2 || list[0].Name != "Radarr" || list[1].Name != "Sonarr" {
		t.Errorf("expected [Radarr Sonarr] for user-1, got %+v", list)
	}

	empty, err := repo.ListByUserID(context.Background(), "nobody")
	if err != nil || empty == nil || len(empty) != 0 {
		t.Errorf("expected empty non-nil list, got %v (err %v)", empty, err)
	}
}

func TestIntegrationRepository_Update(t *testing.T) {
	repo := NewIntegrationRepository(setupIntegrationTestDB(t))
	ctx := context.Background()

	integration := createTestIntegration(t, repo, "user-1", "Sonarr", []byte{0x01})
	integration.Name = "Sonarr 4K"
	integration.BaseURL = "https://sonarr.lan"
	integration.AuthType = models.IntegrationAuthNone
	integration.VerifyTLS = false
	integration.Options = json.RawMessage(`{"days":14}`)
	integration.RefreshSeconds = 120

	if err := repo.Update(ctx, integration, nil); err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	if integration.HasCredentials {
		t.Error("expected nil blob to clear HasCredentials")
	}

	got, _ := repo.GetByID(ctx, integration.ID, "user-1")
	if got.Name != "Sonarr 4K" || got.BaseURL != "https://sonarr.lan" || got.AuthType != models.IntegrationAuthNone ||
		got.VerifyTLS || got.RefreshSeconds != 120 || string(got.Options) != `{"days":14}` || got.HasCredentials {
		t.Errorf("update not persisted: %+v", got)
	}
}

func TestIntegrationRepository_UpdateTestResult(t *testing.T) {
	repo := NewIntegrationRepository(setupIntegrationTestDB(t))
	ctx := context.Background()

	integration := createTestIntegration(t, repo, "user-1", "Sonarr", nil)
	msg := "connection refused"
	if err := repo.UpdateTestResult(ctx, integration.ID, "user-1", false, &msg); err != nil {
		t.Fatalf("UpdateTestResult returned error: %v", err)
	}

	got, _ := repo.GetByID(ctx, integration.ID, "user-1")
	if got.LastTestAt == nil || got.LastTestOK == nil || *got.LastTestOK || got.LastError == nil || *got.LastError != msg {
		t.Errorf("test result not persisted: %+v", got)
	}

	if err := repo.UpdateTestResult(ctx, integration.ID, "user-1", true, nil); err != nil {
		t.Fatalf("UpdateTestResult returned error: %v", err)
	}
	got, _ = repo.GetByID(ctx, integration.ID, "user-1")
	if !*got.LastTestOK || got.LastError != nil {
		t.Errorf("expected success to clear last_error: %+v", got)
	}

	// Update writes the test result as given: kept when set, cleared when nil
	if err := repo.Update(ctx, got, nil); err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	kept, _ := repo.GetByID(ctx, integration.ID, "user-1")
	if kept.LastTestOK == nil || kept.LastTestAt == nil {
		t.Errorf("expected Update to keep the test result: %+v", kept)
	}
	kept.LastTestAt, kept.LastTestOK = nil, nil
	if err := repo.Update(ctx, kept, nil); err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	cleared, _ := repo.GetByID(ctx, integration.ID, "user-1")
	if cleared.LastTestOK != nil || cleared.LastTestAt != nil {
		t.Errorf("expected Update to clear the test result: %+v", cleared)
	}
}

func TestIntegrationRepository_OwnershipIsEnforced(t *testing.T) {
	repo := NewIntegrationRepository(setupIntegrationTestDB(t))
	ctx := context.Background()

	integration := createTestIntegration(t, repo, "user-1", "Sonarr", []byte{0x01})

	if _, err := repo.GetByID(ctx, integration.ID, "user-2"); !errors.Is(err, ErrIntegrationNotFound) {
		t.Errorf("GetByID: expected ErrIntegrationNotFound, got %v", err)
	}
	if _, err := repo.GetCredentials(ctx, integration.ID, "user-2"); !errors.Is(err, ErrIntegrationNotFound) {
		t.Errorf("GetCredentials: expected ErrIntegrationNotFound, got %v", err)
	}
	other := *integration
	other.UserID = "user-2"
	if err := repo.Update(ctx, &other, nil); !errors.Is(err, ErrIntegrationNotFound) {
		t.Errorf("Update: expected ErrIntegrationNotFound, got %v", err)
	}
	if err := repo.UpdateTestResult(ctx, integration.ID, "user-2", true, nil); !errors.Is(err, ErrIntegrationNotFound) {
		t.Errorf("UpdateTestResult: expected ErrIntegrationNotFound, got %v", err)
	}
	if err := repo.Delete(ctx, integration.ID, "user-2"); !errors.Is(err, ErrIntegrationNotFound) {
		t.Errorf("Delete: expected ErrIntegrationNotFound, got %v", err)
	}

	// Still intact for the owner
	if _, err := repo.GetByID(ctx, integration.ID, "user-1"); err != nil {
		t.Errorf("owner lost access: %v", err)
	}
}

func TestIntegrationRepository_Delete(t *testing.T) {
	repo := NewIntegrationRepository(setupIntegrationTestDB(t))
	ctx := context.Background()

	integration := createTestIntegration(t, repo, "user-1", "Sonarr", nil)
	if err := repo.Delete(ctx, integration.ID, "user-1"); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	if _, err := repo.GetByID(ctx, integration.ID, "user-1"); !errors.Is(err, ErrIntegrationNotFound) {
		t.Errorf("expected ErrIntegrationNotFound after delete, got %v", err)
	}
}

func TestIntegrationRepository_LimitReached(t *testing.T) {
	repo := NewIntegrationRepository(setupIntegrationTestDB(t))
	ctx := context.Background()

	for range 2 {
		if err := repo.Create(ctx, &models.Integration{UserID: "user-1", Kind: "k", Name: "n", BaseURL: "http://x", AuthType: "none", RefreshSeconds: 60}, nil, 2); err != nil {
			t.Fatalf("Create under limit returned error: %v", err)
		}
	}
	err := repo.Create(ctx, &models.Integration{UserID: "user-1", Kind: "k", Name: "n", BaseURL: "http://x", AuthType: "none", RefreshSeconds: 60}, nil, 2)
	if !errors.Is(err, ErrIntegrationLimitReached) {
		t.Errorf("expected ErrIntegrationLimitReached, got %v", err)
	}

	list, _ := repo.ListByUserID(ctx, "user-1")
	if len(list) != 2 {
		t.Errorf("rejected create must roll back, found %d rows", len(list))
	}
}

func TestIntegrationRepository_ListInUse(t *testing.T) {
	db := setupIntegrationTestDB(t)
	repo := NewIntegrationRepository(db)
	ctx := context.Background()
	_, err := db.Exec(`CREATE TABLE widgets (id TEXT PRIMARY KEY, integration_id TEXT, enabled INTEGER NOT NULL DEFAULT 1)`)
	require.NoError(t, err)

	used := createTestIntegration(t, repo, "user-1", "Used", nil)
	usedByDisabled := createTestIntegration(t, repo, "user-2", "Disabled widget", nil)
	createTestIntegration(t, repo, "user-1", "Unused", nil)
	_, err = db.Exec(`INSERT INTO widgets (id, integration_id, enabled) VALUES ('w1', ?, 1), ('w2', ?, 1), ('w3', ?, 0), ('w4', NULL, 1)`,
		used.ID, used.ID, usedByDisabled.ID)
	require.NoError(t, err)

	list, err := repo.ListInUse(ctx)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, used.ID, list[0].ID)
}

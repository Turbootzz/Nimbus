package repository

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/nimbus/backend/internal/models"

	_ "github.com/mattn/go-sqlite3"
)

// setupAPITokenTestDB creates an in-memory SQLite database with the api_tokens table
func setupAPITokenTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	schema := `
		CREATE TABLE IF NOT EXISTS api_tokens (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			name TEXT NOT NULL,
			token_hash TEXT NOT NULL UNIQUE,
			token_prefix TEXT NOT NULL,
			read_only INTEGER NOT NULL DEFAULT 1,
			last_used_at TIMESTAMP,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
	`

	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("Failed to create test tables: %v", err)
	}

	return db
}

func createTestToken(t *testing.T, repo *APITokenRepository, userID, name, hash string, readOnly bool) *models.APIToken {
	t.Helper()
	token := &models.APIToken{
		UserID:      userID,
		Name:        name,
		TokenHash:   hash,
		TokenPrefix: "nimbus_abcde",
		ReadOnly:    readOnly,
	}
	if err := repo.Create(context.Background(), token); err != nil {
		t.Fatalf("Failed to create token: %v", err)
	}
	return token
}

func TestAPITokenRepository_CreateAndGet(t *testing.T) {
	db := setupAPITokenTestDB(t)
	defer db.Close()

	repo := NewAPITokenRepository(db)
	ctx := context.Background()

	token := createTestToken(t, repo, "user-1", "Hearth", "hash-1", true)

	if token.ID == "" {
		t.Error("expected Create to set token ID")
	}
	if token.CreatedAt.IsZero() {
		t.Error("expected Create to set CreatedAt")
	}

	got, err := repo.GetByTokenHash(ctx, "hash-1")
	if err != nil {
		t.Fatalf("GetByTokenHash returned error: %v", err)
	}
	if got.ID != token.ID || got.UserID != "user-1" || got.Name != "Hearth" || !got.ReadOnly {
		t.Errorf("unexpected token: %+v", got)
	}
}

func TestAPITokenRepository_GetByTokenHash_NotFound(t *testing.T) {
	db := setupAPITokenTestDB(t)
	defer db.Close()

	repo := NewAPITokenRepository(db)

	_, err := repo.GetByTokenHash(context.Background(), "unknown-hash")
	if !errors.Is(err, ErrAPITokenNotFound) {
		t.Errorf("expected ErrAPITokenNotFound, got %v", err)
	}
}

func TestAPITokenRepository_ListByUserID(t *testing.T) {
	db := setupAPITokenTestDB(t)
	defer db.Close()

	repo := NewAPITokenRepository(db)
	ctx := context.Background()

	createTestToken(t, repo, "user-1", "Token A", "hash-a", true)
	createTestToken(t, repo, "user-1", "Token B", "hash-b", false)
	createTestToken(t, repo, "user-2", "Other", "hash-c", true)

	tokens, err := repo.ListByUserID(ctx, "user-1")
	if err != nil {
		t.Fatalf("ListByUserID returned error: %v", err)
	}
	if len(tokens) != 2 {
		t.Fatalf("expected 2 tokens, got %d", len(tokens))
	}
	for _, tok := range tokens {
		if tok.UserID != "user-1" {
			t.Errorf("got token for wrong user: %+v", tok)
		}
	}
}

func TestAPITokenRepository_Delete(t *testing.T) {
	db := setupAPITokenTestDB(t)
	defer db.Close()

	repo := NewAPITokenRepository(db)
	ctx := context.Background()

	token := createTestToken(t, repo, "user-1", "Hearth", "hash-1", true)

	// Delete scoped to another user must not remove the token
	if err := repo.Delete(ctx, token.ID, "user-2"); !errors.Is(err, ErrAPITokenNotFound) {
		t.Errorf("expected ErrAPITokenNotFound for wrong owner, got %v", err)
	}

	if err := repo.Delete(ctx, token.ID, "user-1"); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}

	if _, err := repo.GetByTokenHash(ctx, "hash-1"); !errors.Is(err, ErrAPITokenNotFound) {
		t.Errorf("expected token to be gone, got %v", err)
	}
}

func TestAPITokenRepository_CountByUserID(t *testing.T) {
	db := setupAPITokenTestDB(t)
	defer db.Close()

	repo := NewAPITokenRepository(db)
	ctx := context.Background()

	createTestToken(t, repo, "user-1", "Token A", "hash-a", true)
	createTestToken(t, repo, "user-1", "Token B", "hash-b", true)

	count, err := repo.CountByUserID(ctx, "user-1")
	if err != nil {
		t.Fatalf("CountByUserID returned error: %v", err)
	}
	if count != 2 {
		t.Errorf("expected count 2, got %d", count)
	}

	count, err = repo.CountByUserID(ctx, "user-2")
	if err != nil {
		t.Fatalf("CountByUserID returned error: %v", err)
	}
	if count != 0 {
		t.Errorf("expected count 0, got %d", count)
	}
}

func TestAPITokenRepository_UpdateLastUsed(t *testing.T) {
	db := setupAPITokenTestDB(t)
	defer db.Close()

	repo := NewAPITokenRepository(db)
	ctx := context.Background()

	token := createTestToken(t, repo, "user-1", "Hearth", "hash-1", true)

	if err := repo.UpdateLastUsed(ctx, token.ID); err != nil {
		t.Fatalf("UpdateLastUsed returned error: %v", err)
	}

	got, err := repo.GetByTokenHash(ctx, "hash-1")
	if err != nil {
		t.Fatalf("GetByTokenHash returned error: %v", err)
	}
	if got.LastUsedAt == nil {
		t.Error("expected LastUsedAt to be set")
	}
}

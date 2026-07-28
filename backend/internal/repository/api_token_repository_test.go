package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/nimbus/backend/internal/models"

	_ "github.com/mattn/go-sqlite3"
)

const apiTokenTestSchema = `
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY
	);

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

	INSERT INTO users (id) VALUES ('user-1'), ('user-2');
`

// setupAPITokenTestDB creates an in-memory SQLite database with the api_tokens table
func setupAPITokenTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	if _, err := db.Exec(apiTokenTestSchema); err != nil {
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
	if err := repo.Create(context.Background(), token, 10); err != nil {
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

func TestAPITokenRepository_ListByUserID_NewestFirst(t *testing.T) {
	db := setupAPITokenTestDB(t)
	defer db.Close()

	repo := NewAPITokenRepository(db)
	ctx := context.Background()

	// Insert directly with explicit timestamps; CURRENT_TIMESTAMP resolution
	// would make same-second inserts an ordering tie
	insert := `INSERT INTO api_tokens (id, user_id, name, token_hash, token_prefix, read_only, created_at) VALUES (?, ?, ?, ?, ?, 1, ?)`
	if _, err := db.Exec(insert, "id-old", "user-1", "Old", "hash-old", "nimbus_old", "2026-01-01 10:00:00"); err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}
	if _, err := db.Exec(insert, "id-new", "user-1", "New", "hash-new", "nimbus_new", "2026-06-01 10:00:00"); err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	tokens, err := repo.ListByUserID(ctx, "user-1")
	if err != nil {
		t.Fatalf("ListByUserID returned error: %v", err)
	}
	if len(tokens) != 2 {
		t.Fatalf("expected 2 tokens, got %d", len(tokens))
	}
	if tokens[0].Name != "New" || tokens[1].Name != "Old" {
		t.Errorf("expected newest first, got order: %s, %s", tokens[0].Name, tokens[1].Name)
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

func TestAPITokenRepository_Create_EnforcesLimit(t *testing.T) {
	db := setupAPITokenTestDB(t)
	defer db.Close()

	repo := NewAPITokenRepository(db)
	ctx := context.Background()

	for i := 0; i < 2; i++ {
		token := &models.APIToken{
			UserID:      "user-1",
			Name:        "Token",
			TokenHash:   fmt.Sprintf("hash-%d", i),
			TokenPrefix: "nimbus_abcde",
			ReadOnly:    true,
		}
		if err := repo.Create(ctx, token, 2); err != nil {
			t.Fatalf("Failed to create token %d: %v", i, err)
		}
	}

	over := &models.APIToken{
		UserID:      "user-1",
		Name:        "Over limit",
		TokenHash:   "hash-over",
		TokenPrefix: "nimbus_abcde",
		ReadOnly:    true,
	}
	if err := repo.Create(ctx, over, 2); !errors.Is(err, ErrAPITokenLimitReached) {
		t.Errorf("expected ErrAPITokenLimitReached, got %v", err)
	}

	// Rejected insert must be rolled back
	tokens, err := repo.ListByUserID(ctx, "user-1")
	if err != nil {
		t.Fatalf("ListByUserID returned error: %v", err)
	}
	if len(tokens) != 2 {
		t.Errorf("expected 2 tokens after rejected create, got %d", len(tokens))
	}

	// Other users are unaffected by user-1's limit
	other := &models.APIToken{
		UserID:      "user-2",
		Name:        "Other user",
		TokenHash:   "hash-other",
		TokenPrefix: "nimbus_abcde",
		ReadOnly:    true,
	}
	if err := repo.Create(ctx, other, 2); err != nil {
		t.Errorf("expected create for other user to succeed, got %v", err)
	}
}

func TestAPITokenRepository_Create_ConcurrentRespectsLimit(t *testing.T) {
	// Single connection: SQLite has no concurrent writers (shared-cache mode
	// returns SQLITE_LOCKED instead of waiting). Goroutines still race on the
	// pool, proving the limit invariant holds under concurrent Create calls;
	// cross-connection serialization on Postgres comes from the user row lock.
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)

	if _, err := db.Exec(apiTokenTestSchema); err != nil {
		t.Fatalf("Failed to create test tables: %v", err)
	}

	repo := NewAPITokenRepository(db)
	const limit = 5
	const attempts = 10

	var wg sync.WaitGroup
	errs := make([]error, attempts)
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			token := &models.APIToken{
				UserID:      "user-1",
				Name:        "Concurrent",
				TokenHash:   fmt.Sprintf("hash-concurrent-%d", i),
				TokenPrefix: "nimbus_abcde",
				ReadOnly:    true,
			}
			errs[i] = repo.Create(context.Background(), token, limit)
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil && !errors.Is(err, ErrAPITokenLimitReached) {
			t.Errorf("create %d returned unexpected error: %v", i, err)
		}
	}

	tokens, err := repo.ListByUserID(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("ListByUserID returned error: %v", err)
	}
	if len(tokens) > limit {
		t.Errorf("limit exceeded under concurrency: got %d tokens, limit %d", len(tokens), limit)
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

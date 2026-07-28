# API Tokens (Personal Access Tokens) Implementation Plan — Issue #177

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Users can create/revoke personal access tokens in Settings and use them as `Authorization: Bearer nimbus_...` for programmatic API access, with an optional read-only flag.

**Architecture:** New `api_tokens` table stores SHA-256 hashes (plaintext shown once at creation). `AuthMiddleware` gains a PAT branch: Bearer tokens with the `nimbus_` prefix are looked up by hash instead of JWT-validated; user identity comes from the DB row. Read-only tokens are rejected on non-GET/HEAD requests in middleware. Token-management routes require session (JWT) auth so a leaked PAT can never mint or revoke tokens. Frontend gets a `/settings/api-tokens` page following the webhooks/notifications CRUD pattern.

**Tech Stack:** Go + Fiber, PostgreSQL (custom migration runner), SQLite in-memory for tests, Next.js App Router + Tailwind, vitest + testing-library.

## Global Constraints

- Commit messages: conventional style (`feat: ...`), **no Co-Authored-By trailer**.
- Handlers → services/repository layering; `context.Context` passed down (`c.Context()`).
- Token format: `nimbus_` + 64 hex chars (32 random bytes). Hash: SHA-256 hex. Display prefix: first 12 chars.
- Max 10 tokens per user (mirrors `maxWebhooksPerUser`).
- Plaintext token appears in exactly one response: the create response.
- Frontend: Server Components default, `'use client'` only where state needed; types in `types/index.ts`; API calls via `lib/api.ts`.
- Run `make ci-check` at the end.

---

### Task 1: Migration + model

**Files:**
- Create: `backend/internal/db/migrations/000025_create_api_tokens.up.sql`
- Create: `backend/internal/db/migrations/000025_create_api_tokens.down.sql`
- Create: `backend/internal/models/api_token.go`

**Interfaces:**
- Produces: `models.APIToken` (fields below), `models.APITokenCreateRequest{Name string; ReadOnly *bool}`, `models.APITokenCreateResponse{Token string; APIToken APIToken}`.

- [ ] **Step 1: Write up migration**

```sql
CREATE TABLE IF NOT EXISTS api_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    token_hash VARCHAR(64) NOT NULL UNIQUE,
    token_prefix VARCHAR(12) NOT NULL,
    read_only BOOLEAN NOT NULL DEFAULT TRUE,
    last_used_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_api_tokens_user_id ON api_tokens(user_id);
```

- [ ] **Step 2: Write down migration**

```sql
DROP TABLE IF EXISTS api_tokens;
```

- [ ] **Step 3: Write model** (`backend/internal/models/api_token.go`)

```go
package models

import "time"

// APIToken is a personal access token for programmatic API access.
// TokenHash is never serialized; the plaintext token is only returned at creation.
type APIToken struct {
	ID          string     `json:"id" db:"id"`
	UserID      string     `json:"user_id" db:"user_id"`
	Name        string     `json:"name" db:"name"`
	TokenHash   string     `json:"-" db:"token_hash"`
	TokenPrefix string     `json:"token_prefix" db:"token_prefix"`
	ReadOnly    bool       `json:"read_only" db:"read_only"`
	LastUsedAt  *time.Time `json:"last_used_at" db:"last_used_at"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
}

// APITokenCreateRequest is the payload for creating a token
type APITokenCreateRequest struct {
	Name     string `json:"name"`
	ReadOnly *bool  `json:"read_only"`
}

// APITokenCreateResponse carries the plaintext token exactly once
type APITokenCreateResponse struct {
	Token    string   `json:"token"`
	APIToken APIToken `json:"api_token"`
}
```

- [ ] **Step 4: Compile** — `cd backend && go build ./...` → OK
- [ ] **Step 5: Commit** — `feat: add api_tokens migration and model`

---

### Task 2: Token utils (generate/hash)

**Files:**
- Create: `backend/internal/utils/api_token.go`
- Test: `backend/internal/utils/api_token_test.go`

**Interfaces:**
- Produces: `utils.GenerateAPIToken() (token, hash, prefix string, err error)`, `utils.HashAPIToken(token string) string`, `utils.IsAPIToken(token string) bool`, `utils.APITokenPrefix = "nimbus_"`.

- [ ] **Step 1: Write failing test** (`api_token_test.go`): token starts with `nimbus_`, length = 7+64, hash equals `HashAPIToken(token)` (64 hex chars), prefix = first 12 chars, two calls differ, `IsAPIToken("nimbus_x")` true / `IsAPIToken("eyJ...")` false.
- [ ] **Step 2: Run** `go test ./internal/utils/ -run APIToken -v` → FAIL (undefined)
- [ ] **Step 3: Implement**

```go
package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// APITokenPrefix marks bearer tokens as personal access tokens (vs JWTs)
const APITokenPrefix = "nimbus_"

// GenerateAPIToken returns a new plaintext token, its SHA-256 hex hash,
// and the 12-char display prefix stored for identification in the UI.
func GenerateAPIToken() (token, hash, prefix string, err error) {
	raw := make([]byte, 32)
	if _, err = rand.Read(raw); err != nil {
		return "", "", "", fmt.Errorf("failed to generate token: %w", err)
	}
	token = APITokenPrefix + hex.EncodeToString(raw)
	return token, HashAPIToken(token), token[:12], nil
}

// HashAPIToken returns the SHA-256 hex digest of a token
func HashAPIToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// IsAPIToken reports whether a bearer token is a personal access token
func IsAPIToken(token string) bool {
	return strings.HasPrefix(token, APITokenPrefix)
}
```

- [ ] **Step 4: Run** → PASS
- [ ] **Step 5: Commit** — `feat: add api token generation helpers`

---

### Task 3: Repository

**Files:**
- Create: `backend/internal/repository/api_token_repository.go`
- Test: `backend/internal/repository/api_token_repository_test.go` (SQLite in-memory, mirror `webhook_repository_test.go` setup)

**Interfaces:**
- Consumes: `models.APIToken`.
- Produces: `NewAPITokenRepository(db *sql.DB) *APITokenRepository` with methods:
  - `Create(ctx, *models.APIToken) error` (generates UUID in Go via `uuid.New().String()` — SQLite has no `gen_random_uuid()`)
  - `GetByTokenHash(ctx, hash string) (*models.APIToken, error)` → `ErrAPITokenNotFound`
  - `ListByUserID(ctx, userID string) ([]models.APIToken, error)` (newest first)
  - `Delete(ctx, id, userID string) error` → `ErrAPITokenNotFound` when 0 rows
  - `UpdateLastUsed(ctx, id string) error`
  - `CountByUserID(ctx, userID string) (int, error)`
  - Sentinel: `var ErrAPITokenNotFound = errors.New("api token not found")`

- [ ] **Step 1: Write failing tests**: create+get roundtrip, get unknown hash → ErrAPITokenNotFound, list scoped per user, delete scoped to owner (other user's delete → ErrAPITokenNotFound), count, UpdateLastUsed sets timestamp. SQLite fixture table:

```sql
CREATE TABLE api_tokens (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    name TEXT NOT NULL,
    token_hash TEXT NOT NULL UNIQUE,
    token_prefix TEXT NOT NULL,
    read_only BOOLEAN NOT NULL DEFAULT TRUE,
    last_used_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

- [ ] **Step 2: Run** `go test ./internal/repository/ -run APIToken -v` → FAIL
- [ ] **Step 3: Implement** repository. Query patterns follow `password_reset_repository.go` (`QueryRowContext`, `$1` placeholders, wrap errors with `fmt.Errorf("...: %w", err)`). `Create` inserts id/user_id/name/token_hash/token_prefix/read_only with `RETURNING created_at`. `Delete` uses `DELETE FROM api_tokens WHERE id = $1 AND user_id = $2` + `RowsAffected`.
- [ ] **Step 4: Run** → PASS
- [ ] **Step 5: Commit** — `feat: add api token repository`

---

### Task 4: Middleware PAT support + RequireSessionAuth

**Files:**
- Modify: `backend/internal/middleware/auth_middleware.go`
- Modify: `backend/cmd/server/main.go` (all `AuthMiddleware(...)`/`OptionalAuthMiddleware(...)` call sites get `apiTokenRepo` arg)
- Test: `backend/internal/middleware/auth_middleware_test.go`

**Interfaces:**
- Consumes: `utils.IsAPIToken`, `utils.HashAPIToken`, `APITokenRepository.GetByTokenHash/UpdateLastUsed`.
- Produces:
  - `AuthMiddleware(authService *services.AuthService, userRepo *repository.UserRepository, apiTokenRepo *repository.APITokenRepository) fiber.Handler`
  - `OptionalAuthMiddleware(...)` same signature change
  - `RequireSessionAuth() fiber.Handler`
  - Locals: existing `user_id`, `email`, `role` + new `auth_method` = `"session"` (JWT/cookie) or `"api_token"`.

- [ ] **Step 1: Write failing tests** (extend `auth_middleware_test.go`, reuse `setupMiddlewareTestDB` + add `api_tokens` fixture table from Task 3):
  - Bearer PAT valid → 200, locals user_id/email/role set from user row, `auth_method` = `api_token`, `last_used_at` updated
  - Bearer PAT unknown → 401
  - Read-only PAT + POST → 403; read-only PAT + GET → 200; non-read-only PAT + POST → 200
  - `RequireSessionAuth` after PAT auth → 403; after cookie JWT auth → 200
  - Existing JWT cookie tests updated to new signature; JWT path sets `auth_method` = `session`
- [ ] **Step 2: Run** `go test ./internal/middleware/ -v` → FAIL
- [ ] **Step 3: Implement.** In both middlewares, after token extraction (cookie → Bearer fallback): if `utils.IsAPIToken(token)` branch to PAT auth instead of JWT validation.

PAT auth (strict variant, in `AuthMiddleware`):

```go
func authenticateAPIToken(c *fiber.Ctx, token string, userRepo *repository.UserRepository, apiTokenRepo *repository.APITokenRepository) error {
	apiToken, err := apiTokenRepo.GetByTokenHash(c.Context(), utils.HashAPIToken(token))
	if err != nil {
		if errors.Is(err, repository.ErrAPITokenNotFound) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid API token"})
		}
		c.Context().Logger().Printf("API token auth DB error: %v", err)
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Authentication service unavailable"})
	}

	if apiToken.ReadOnly && c.Method() != fiber.MethodGet && c.Method() != fiber.MethodHead {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Read-only token cannot modify resources"})
	}

	user, err := userRepo.GetByID(apiToken.UserID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "User not found - invalid token"})
		}
		c.Context().Logger().Printf("API token auth DB error: %v", err)
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Authentication service unavailable"})
	}

	// Best-effort; auth must not fail on this
	if err := apiTokenRepo.UpdateLastUsed(c.Context(), apiToken.ID); err != nil {
		c.Context().Logger().Printf("Failed to update api token last_used_at: %v", err)
	}

	c.Locals("user_id", user.ID)
	c.Locals("email", user.Email)
	c.Locals("role", user.Role)
	c.Locals("auth_method", "api_token")
	return c.Next()
}
```

JWT path: add `c.Locals("auth_method", "session")` next to existing locals. `OptionalAuthMiddleware` PAT branch: same lookup but every failure falls through to `c.Next()` without locals (keep optional semantics).

`RequireSessionAuth`:

```go
// RequireSessionAuth blocks API-token auth. Used on token-management routes
// so a leaked token can never create or revoke tokens.
func RequireSessionAuth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if c.Locals("auth_method") == "api_token" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "API tokens cannot manage API tokens"})
		}
		return c.Next()
	}
}
```

- [ ] **Step 4:** Update every `middleware.AuthMiddleware(authService, userRepo)` / `OptionalAuthMiddleware(...)` call in `cmd/server/main.go` to pass `apiTokenRepo`; construct `apiTokenRepo := repository.NewAPITokenRepository(...)` beside the other repos (same DB handle).
- [ ] **Step 5: Run** `go test ./... && go build ./...` → PASS
- [ ] **Step 6: Commit** — `feat: authenticate personal access tokens in auth middleware`

---

### Task 5: Handler + routes

**Files:**
- Create: `backend/internal/handlers/api_token_handler.go`
- Modify: `backend/cmd/server/main.go` (handler init + routes)
- Test: `backend/internal/handlers/api_token_handler_test.go` (mirror `webhook_handler_test.go` SQLite setup)

**Interfaces:**
- Consumes: `RequireUserID(c)` (helpers.go), `utils.GenerateAPIToken`, repository from Task 3.
- Produces: `NewAPITokenHandler(tokenRepo *repository.APITokenRepository) *APITokenHandler` with `CreateToken`, `ListTokens`, `DeleteToken`. Routes:

```go
apiTokens := v1.Group("/tokens", middleware.AuthMiddleware(authService, userRepo, apiTokenRepo), middleware.RequireSessionAuth())
apiTokens.Post("/", apiTokenHandler.CreateToken)
apiTokens.Get("/", apiTokenHandler.ListTokens)
apiTokens.Delete("/:id<guid>", apiTokenHandler.DeleteToken)
```

- [ ] **Step 1: Write failing tests**: create → 201 with plaintext `token` (prefix `nimbus_`) + `api_token` object; name required → 400; name > 100 chars → 400; default `read_only` = true when omitted; 11th token → 400 (limit); list → tokens without plaintext/hash, only own tokens; delete own → 204; delete other user's / unknown id → 404.
- [ ] **Step 2: Run** `go test ./internal/handlers/ -run APIToken -v` → FAIL
- [ ] **Step 3: Implement** handler. Validation/limit/shape follow `webhook_handler.go::CreateWebhook` (`const maxAPITokensPerUser = 10`). Create: parse body → validate → count check → `utils.GenerateAPIToken()` → `readOnly := true; if req.ReadOnly != nil { readOnly = *req.ReadOnly }` → repo.Create → `c.Status(fiber.StatusCreated).JSON(models.APITokenCreateResponse{Token: token, APIToken: *apiToken})`. Delete: `id := c.Params("id")` (router `<guid>` constraint validates), `errors.Is(err, repository.ErrAPITokenNotFound)` → 404, else 204 `c.SendStatus(fiber.StatusNoContent)`.
- [ ] **Step 4:** Wire `apiTokenHandler := handlers.NewAPITokenHandler(apiTokenRepo)` + routes in `main.go`.
- [ ] **Step 5: Run** `go test ./... && go build ./...` → PASS
- [ ] **Step 6: Commit** — `feat: add api token management endpoints`

---

### Task 6: Frontend types + API client

**Files:**
- Modify: `frontend/types/index.ts`
- Modify: `frontend/lib/api.ts`

**Interfaces:**
- Produces types `ApiToken`, `ApiTokenCreateRequest`, `ApiTokenCreateResponse`; api methods `getApiTokens()`, `createApiToken(data)`, `deleteApiToken(id)`.

- [ ] **Step 1: Add types** (`types/index.ts`):

```ts
export interface ApiToken {
  id: string
  name: string
  token_prefix: string
  read_only: boolean
  last_used_at: string | null
  created_at: string
}

export interface ApiTokenCreateRequest {
  name: string
  read_only: boolean
}

export interface ApiTokenCreateResponse {
  token: string
  api_token: ApiToken
}
```

- [ ] **Step 2: Add api methods** (`lib/api.ts`, next to the Webhooks block, import the new types):

```ts
// API Tokens

async getApiTokens(): Promise<ApiResponse<ApiToken[]>> {
  return this.request<ApiToken[]>('/tokens')
}

async createApiToken(data: ApiTokenCreateRequest): Promise<ApiResponse<ApiTokenCreateResponse>> {
  return this.request<ApiTokenCreateResponse>('/tokens', {
    method: 'POST',
    body: JSON.stringify(data),
  })
}

async deleteApiToken(id: string): Promise<ApiResponse<void>> {
  return this.request<void>(`/tokens/${id}`, { method: 'DELETE' })
}
```

- [ ] **Step 3:** `cd frontend && npx tsc --noEmit` → OK
- [ ] **Step 4: Commit** — `feat: add api token types and client methods`

---

### Task 7: Settings page + nav + frontend test

**Files:**
- Create: `frontend/app/(dashboard)/settings/api-tokens/page.tsx`
- Modify: `frontend/app/(dashboard)/settings/page.tsx` (add nav entry to `settingsSections`)
- Test: `frontend/__tests__/components/ApiTokensPage.test.tsx`

**Interfaces:**
- Consumes: `api.getApiTokens/createApiToken/deleteApiToken`, `ApiToken` type, `Toggle` component (`components/ui/Toggle.tsx`).

- [ ] **Step 1: Write failing test** (mock `@/lib/api` with `vi.mock`, patterns from `__tests__/components/ServicesList.test.tsx`): renders token list from `getApiTokens`; empty state text when no tokens; create flow calls `createApiToken` and reveals plaintext token exactly once; revoke button calls `deleteApiToken`.
- [ ] **Step 2: Run** `npm run test -- ApiTokensPage` → FAIL
- [ ] **Step 3: Implement page** (`'use client'`): follow notifications/webhooks page structure — `useState` for tokens/loading/error, `useEffect` load, card layout (`bg-card border-card-border rounded-lg border`). Content:
  - Create form: name `ThemedInput`, read-only `Toggle` (default on, description "Token can only read data, never modify"), Create button.
  - After create: highlighted box with full plaintext token + Copy button (`navigator.clipboard.writeText`) + warning "You won't be able to see this token again".
  - List: name, `token_prefix` + `…`, read-only badge, created/last-used dates (`toLocaleDateString`, last used "Never" when null), Revoke button with `confirm()` guard.
  - Usage hint: `Authorization: Bearer <token>`.
- [ ] **Step 4:** Add nav entry to `settingsSections` in `settings/page.tsx`: title "API Tokens", description "Create tokens for programmatic API access", href `/settings/api-tokens`, key icon SVG (heroicons key path).
- [ ] **Step 5: Run** `npm run test -- ApiTokensPage` → PASS; `npx tsc --noEmit` → OK
- [ ] **Step 6: Commit** — `feat: add api tokens settings page`

---

### Task 8: Full check

- [ ] **Step 1:** `make ci-check` from repo root → all green (backend fmt/lint/tests/build + frontend ci-check). Fix anything that fails.
- [ ] **Step 2:** Commit fixes if any — `fix: ...`

---

## Self-Review Notes

- Issue coverage: generate/revoke in settings UI ✔ (Task 7), Bearer header ✔ (Task 4), read endpoints ✔ (PAT rides existing AuthMiddleware, so /services, /metrics etc. all work), read-only flag ✔ (middleware method gate). Extra hardening: PATs cannot manage PATs (RequireSessionAuth).
- `OptionalAuthMiddleware` PAT branch keeps optional semantics (falls through on failure) so the Prometheus endpoint gains PAT support without behavior change.
- Type consistency: `APIToken`/`APITokenRepository`/`ErrAPITokenNotFound` naming consistent across tasks; frontend `ApiToken` matches backend JSON tags (`token_prefix`, `read_only`, `last_used_at`).

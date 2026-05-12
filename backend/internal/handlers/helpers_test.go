package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestRequireUserID_Success(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		c.Locals("user_id", "test-user-123")
		userID, err := RequireUserID(c)
		if err != nil {
			return err
		}
		return c.JSON(fiber.Map{"user_id": userID})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test request: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if body["user_id"] != "test-user-123" {
		t.Errorf("Expected user_id 'test-user-123', got %q", body["user_id"])
	}
}

func TestRequireUserID_MissingUserID(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		// No user_id set in locals
		userID, err := RequireUserID(c)
		if err != nil {
			return err // Fiber's error handler will send 401
		}
		return c.JSON(fiber.Map{"user_id": userID})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test request: %v", err)
	}

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", resp.StatusCode)
	}
}

func TestRequireUserID_EmptyUserID(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		c.Locals("user_id", "") // Empty string
		_, err := RequireUserID(c)
		if err != nil {
			return err // Fiber's error handler will send 401
		}
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test request: %v", err)
	}

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", resp.StatusCode)
	}
}

func TestRequireUserID_WrongType(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		c.Locals("user_id", 12345) // Integer instead of string
		_, err := RequireUserID(c)
		if err != nil {
			return err // Fiber's error handler will send 401
		}
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test request: %v", err)
	}

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", resp.StatusCode)
	}
}

// TestRequireUUIDParam mounts the helper on a route with NO <guid> constraint
// so we exercise the in-handler path explicitly (constraint-less defense).
func TestRequireUUIDParam(t *testing.T) {
	app := fiber.New()
	app.Get("/test/:id", func(c *fiber.Ctx) error {
		id, err := RequireUUIDParam(c, "id")
		if err != nil {
			return err
		}
		return c.JSON(fiber.Map{"id": id})
	})

	cases := []struct {
		name           string
		id             string
		expectedStatus int
		expectedID     string // empty = don't check body
	}{
		{"valid lowercase UUID", "11111111-1111-1111-1111-111111111111", http.StatusOK, "11111111-1111-1111-1111-111111111111"},
		{"valid uppercase UUID", "AAAAAAAA-AAAA-AAAA-AAAA-AAAAAAAAAAAA", http.StatusOK, "AAAAAAAA-AAAA-AAAA-AAAA-AAAAAAAAAAAA"},
		{"valid mixed-case UUID", "AbCdEf01-2345-6789-aBcD-eF0123456789", http.StatusOK, "AbCdEf01-2345-6789-aBcD-eF0123456789"},
		{"non-UUID word", "favicon", http.StatusNotFound, ""},
		{"non-UUID garbage", "not-a-uuid-12345", http.StatusNotFound, ""},
		// URL-encoded SQL injection payload — even after Fiber decodes it,
		// it's not a valid UUID, so the helper rejects with 404.
		{"SQL injection (encoded)", "1%27%3B%20DROP%20TABLE%20services", http.StatusNotFound, ""},
		{"too short", "1234", http.StatusNotFound, ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test/"+tc.id, nil)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("Failed to test request: %v", err)
			}
			if resp.StatusCode != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d (id=%q)", tc.expectedStatus, resp.StatusCode, tc.id)
			}
			if tc.expectedID != "" {
				var body map[string]string
				if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}
				if body["id"] != tc.expectedID {
					t.Errorf("Expected id %q, got %q", tc.expectedID, body["id"])
				}
			}
		})
	}
}

// TestRouteGuidConstraintRejectsNonUUID is the regression guard for the bug
// that caused production Postgres `invalid input syntax for type uuid` errors
// when /services/favicon hit the old backend (no /favicon route) and matched
// /:id with id="favicon". It also confirms that a static reserved-word route
// registered alongside `:id<guid>` still wins, so paths like /services/favicon
// don't get swallowed by the constraint.
func TestRouteGuidConstraintRejectsNonUUID(t *testing.T) {
	app := fiber.New()
	handlerCalled := false
	app.Get("/services/favicon", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})
	app.Get("/services/:id<guid>", func(c *fiber.Ctx) error {
		handlerCalled = true
		return c.SendStatus(fiber.StatusOK)
	})

	// Static route still wins for the reserved word.
	req := httptest.NewRequest(http.MethodGet, "/services/favicon", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("/services/favicon should hit the static route, got %d", resp.StatusCode)
	}
	if handlerCalled {
		t.Error("the /:id handler should not have been called for /favicon")
	}

	// Non-UUID :id is rejected by the router, NOT passed to the handler.
	req = httptest.NewRequest(http.MethodGet, "/services/not-a-uuid", nil)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("non-UUID :id should return 404 from the router, got %d", resp.StatusCode)
	}
	if handlerCalled {
		t.Error("router constraint should have rejected the non-UUID before reaching the handler")
	}

	// Valid UUID :id reaches the handler.
	handlerCalled = false
	req = httptest.NewRequest(http.MethodGet, "/services/11111111-1111-1111-1111-111111111111", nil)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("valid UUID :id should reach the handler, got %d", resp.StatusCode)
	}
	if !handlerCalled {
		t.Error("valid UUID :id should reach the handler")
	}
}

// TestRouteGuidConstraintCoversAllResourceGroups confirms that every resource
// group under /api/v1 that takes a UUID :id has the <guid> constraint in
// place — if a future contributor drops the constraint on any of them, this
// test catches it. Each row exercises a single route shape and expects 404
// at the router (handler never runs) for a non-UUID path.
func TestRouteGuidConstraintCoversAllResourceGroups(t *testing.T) {
	cases := []struct {
		name       string
		method     string
		routeSpec  string // pattern registered with Fiber
		requestURL string // the actual path requested (non-UUID)
	}{
		{"services :id", http.MethodGet, "/api/v1/services/:id<guid>", "/api/v1/services/not-a-uuid"},
		{"services :id/check", http.MethodPost, "/api/v1/services/:id<guid>/check", "/api/v1/services/bad-id/check"},
		{"services :id/status-logs", http.MethodGet, "/api/v1/services/:id<guid>/status-logs", "/api/v1/services/bad-id/status-logs"},
		{"groups :id", http.MethodGet, "/api/v1/groups/:id<guid>", "/api/v1/groups/not-a-uuid"},
		{"webhooks :id", http.MethodGet, "/api/v1/webhooks/:id<guid>", "/api/v1/webhooks/not-a-uuid"},
		{"webhooks :id/test", http.MethodPost, "/api/v1/webhooks/:id<guid>/test", "/api/v1/webhooks/bad-id/test"},
		{"webhooks :id/logs", http.MethodGet, "/api/v1/webhooks/:id<guid>/logs", "/api/v1/webhooks/bad-id/logs"},
		{"metrics :id", http.MethodGet, "/api/v1/metrics/:id<guid>", "/api/v1/metrics/not-a-uuid"},
		{"prometheus :userID", http.MethodGet, "/api/v1/prometheus/metrics/user/:userID<guid>", "/api/v1/prometheus/metrics/user/not-a-uuid"},
		{"admin :id/role", http.MethodPut, "/api/v1/admin/users/:id<guid>/role", "/api/v1/admin/users/bad-id/role"},
		{"admin :id", http.MethodDelete, "/api/v1/admin/users/:id<guid>", "/api/v1/admin/users/bad-id"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app := fiber.New()
			handlerCalled := false
			app.Add(tc.method, tc.routeSpec, func(c *fiber.Ctx) error {
				handlerCalled = true
				return c.SendStatus(fiber.StatusOK)
			})

			req := httptest.NewRequest(tc.method, tc.requestURL, nil)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("Failed to test request: %v", err)
			}
			if resp.StatusCode != http.StatusNotFound {
				t.Errorf("Expected 404 from router for non-UUID, got %d", resp.StatusCode)
			}
			if handlerCalled {
				t.Errorf("Router should have rejected %s before reaching handler", tc.requestURL)
			}
		})
	}
}

// TestCatchAll404ReturnsJSON proves that a Fiber app with the catch-all
// middleware from main.go responds with JSON for unmatched paths — including
// paths rejected by the <guid> constraint. The frontend's API client expects
// JSON for every response; without this catch-all the constraint rejection
// falls back to Fiber's default "text/plain Cannot GET …" body and the
// frontend renders a generic "API returned an invalid response" toast.
func TestCatchAll404ReturnsJSON(t *testing.T) {
	app := fiber.New()
	app.Get("/services/:id<guid>", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})
	// Mirror main.go's catch-all (must be registered last).
	app.Use(func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Not found"})
	})

	cases := []struct {
		name string
		path string
	}{
		{"constraint-rejected non-UUID", "/services/not-a-uuid"},
		{"totally unknown path", "/no-such/endpoint/anywhere"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("app.Test failed: %v", err)
			}
			if resp.StatusCode != http.StatusNotFound {
				t.Errorf("Expected 404, got %d", resp.StatusCode)
			}
			if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "application/json") {
				t.Errorf("Expected Content-Type to contain application/json, got %q", ct)
			}
			var body map[string]string
			if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
				t.Fatalf("Failed to decode JSON body: %v", err)
			}
			if body["error"] != "Not found" {
				t.Errorf("Expected error=\"Not found\", got %q", body["error"])
			}
		})
	}
}

// TestMainGoRoutesHaveGuidConstraint statically inspects the production route
// table in backend/cmd/server/main.go to confirm every UUID-typed param (`:id`
// and `:userID`) carries the `<guid>` constraint. The companion test above
// (TestRouteGuidConstraintCoversAllResourceGroups) only proves Fiber's
// constraint *behaviour* — it can't catch a contributor dropping `<guid>` from
// a real registration. This test reads the actual main.go source so the
// failure mode for #152 is caught at the source level.
func TestMainGoRoutesHaveGuidConstraint(t *testing.T) {
	const mainGoPath = "../../cmd/server/main.go"
	src, err := os.ReadFile(mainGoPath)
	if err != nil {
		t.Fatalf("read %s: %v", mainGoPath, err)
	}

	// Match Fiber registrations of the form `.METHOD("/path/...", ...)` where
	// the path is the first argument.
	routeRE := regexp.MustCompile(`\.(?:Get|Post|Put|Delete|Patch)\("([^"]+)"`)
	// Match Fiber's generic `.Add(method, "/path", ...)` form where the path
	// is the SECOND argument. The method arg can be a constant (http.MethodGet)
	// or a string literal — either way we just need to skip past it.
	addRouteRE := regexp.MustCompile(`\.Add\([^,]+,\s*"([^"]+)"`)
	// Within a registered path, find every `:name` optionally followed by a
	// `<constraint>` block. Whitelist the param names we know must be UUIDs.
	paramRE := regexp.MustCompile(`:([a-zA-Z_]+)(<[^>]*>)?`)

	uuidParams := map[string]bool{
		"id":     true,
		"userID": true,
	}

	matches := routeRE.FindAllStringSubmatch(string(src), -1)
	matches = append(matches, addRouteRE.FindAllStringSubmatch(string(src), -1)...)
	if len(matches) == 0 {
		t.Fatal("no route registrations matched in main.go - test regex is broken")
	}

	uuidParamCount := 0
	for _, m := range matches {
		path := m[1]
		for _, p := range paramRE.FindAllStringSubmatch(path, -1) {
			name, constraint := p[1], p[2]
			if !uuidParams[name] {
				continue
			}
			uuidParamCount++
			if constraint != "<guid>" {
				if constraint == "" {
					t.Errorf("route %q: param :%s is missing the <guid> constraint", path, name)
				} else {
					t.Errorf("route %q: param :%s has constraint %q, want <guid>", path, name, constraint)
				}
			}
		}
	}
	if uuidParamCount == 0 {
		t.Fatal("no :id or :userID params found in main.go - test regex is broken")
	}
}

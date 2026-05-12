package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/repository"

	_ "github.com/mattn/go-sqlite3"
)

// setupGroupTestDB creates an in-memory SQLite database for group testing
func setupGroupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	schema := `
		CREATE TABLE IF NOT EXISTS groups (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			name TEXT NOT NULL,
			color TEXT DEFAULT '#6366f1',
			position INTEGER DEFAULT 0,
			is_default INTEGER DEFAULT 0,
			monitoring_enabled INTEGER DEFAULT 1,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_groups_user_id ON groups(user_id);
	`

	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}

	return db
}

// createGroupDirectly inserts a group for testing
func createGroupDirectly(t *testing.T, db *sql.DB, group *models.Group) {
	query := `
		INSERT INTO groups (id, user_id, name, color, position, is_default, monitoring_enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	isDefault := 0
	if group.IsDefault {
		isDefault = 1
	}
	monitoringEnabled := 1
	if !group.MonitoringEnabled {
		monitoringEnabled = 0
	}

	_, err := db.Exec(
		query,
		group.ID,
		group.UserID,
		group.Name,
		group.Color,
		group.Position,
		isDefault,
		monitoringEnabled,
		group.CreatedAt,
		group.UpdatedAt,
	)
	if err != nil {
		t.Fatalf("Failed to create group directly: %v", err)
	}
}

func TestGroupHandler_CreateGroup(t *testing.T) {
	db := setupGroupTestDB(t)
	defer db.Close()

	groupRepo := repository.NewGroupRepository(db)
	handler := NewGroupHandler(groupRepo)

	tests := []struct {
		name           string
		userID         string
		requestBody    models.GroupCreateRequest
		expectedStatus int
		expectError    bool
	}{
		{
			name:   "Missing name",
			userID: "cccccccc-cccc-cccc-cccc-ccccccccccc1",
			requestBody: models.GroupCreateRequest{
				Color: "#ff0000",
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name:   "Invalid color format",
			userID: "cccccccc-cccc-cccc-cccc-ccccccccccc1",
			requestBody: models.GroupCreateRequest{
				Name:  "Test Group",
				Color: "invalid",
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name:   "Name too long",
			userID: "cccccccc-cccc-cccc-cccc-ccccccccccc1",
			requestBody: models.GroupCreateRequest{
				Name:  string(make([]byte, 36)),
				Color: "#ff0000",
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Post("/groups", func(c *fiber.Ctx) error {
				c.Locals("user_id", tt.userID)
				return handler.CreateGroup(c)
			})

			bodyJSON, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/groups", bytes.NewReader(bodyJSON))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req, -1)
			if err != nil {
				t.Fatalf("Failed to execute request: %v", err)
			}

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}
		})
	}
}

func TestGroupHandler_CreateGroup_NoAuth(t *testing.T) {
	db := setupGroupTestDB(t)
	defer db.Close()

	groupRepo := repository.NewGroupRepository(db)
	handler := NewGroupHandler(groupRepo)

	app := fiber.New()
	app.Post("/groups", handler.CreateGroup)

	requestBody := models.GroupCreateRequest{
		Name:  "Test Group",
		Color: "#ff0000",
	}

	bodyJSON, _ := json.Marshal(requestBody)
	req := httptest.NewRequest(http.MethodPost, "/groups", bytes.NewReader(bodyJSON))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status %d for unauthenticated request, got %d", http.StatusUnauthorized, resp.StatusCode)
	}
}

func TestGroupHandler_GetGroups(t *testing.T) {
	db := setupGroupTestDB(t)
	defer db.Close()

	groupRepo := repository.NewGroupRepository(db)
	handler := NewGroupHandler(groupRepo)

	// Create test groups for user-1
	createGroupDirectly(t, db, &models.Group{
		ID:        "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1",
		UserID:    "cccccccc-cccc-cccc-cccc-ccccccccccc1",
		Name:      "Group 1",
		Color:     "#ff0000",
		Position:  0,
		IsDefault: true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	createGroupDirectly(t, db, &models.Group{
		ID:        "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa2",
		UserID:    "cccccccc-cccc-cccc-cccc-ccccccccccc1",
		Name:      "Group 2",
		Color:     "#00ff00",
		Position:  1,
		IsDefault: false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	// Create group for different user
	createGroupDirectly(t, db, &models.Group{
		ID:        "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa3",
		UserID:    "cccccccc-cccc-cccc-cccc-ccccccccccc2",
		Name:      "Group 3",
		Color:     "#0000ff",
		Position:  0,
		IsDefault: true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	app := fiber.New()
	app.Get("/groups", func(c *fiber.Ctx) error {
		c.Locals("user_id", "cccccccc-cccc-cccc-cccc-ccccccccccc1")
		return handler.GetGroups(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/groups", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	// Decode response
	var groups []models.GroupResponse
	if err := json.NewDecoder(resp.Body).Decode(&groups); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Should only get user-1's groups
	if len(groups) != 2 {
		t.Errorf("Expected 2 groups, got %d", len(groups))
	}

	// Verify user isolation
	for _, g := range groups {
		if g.ID == "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa3" {
			t.Error("GetGroups returned another user's group")
		}
	}
}

func TestGroupHandler_GetGroup(t *testing.T) {
	db := setupGroupTestDB(t)
	defer db.Close()

	groupRepo := repository.NewGroupRepository(db)
	handler := NewGroupHandler(groupRepo)

	// Create test groups
	createGroupDirectly(t, db, &models.Group{
		ID:        "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1",
		UserID:    "cccccccc-cccc-cccc-cccc-ccccccccccc1",
		Name:      "Group 1",
		Color:     "#ff0000",
		Position:  0,
		IsDefault: true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	createGroupDirectly(t, db, &models.Group{
		ID:        "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa2",
		UserID:    "cccccccc-cccc-cccc-cccc-ccccccccccc2",
		Name:      "Group 2",
		Color:     "#00ff00",
		Position:  0,
		IsDefault: true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	tests := []struct {
		name           string
		userID         string
		groupID        string
		expectedStatus int
	}{
		{
			name:           "Get own group",
			userID:         "cccccccc-cccc-cccc-cccc-ccccccccccc1",
			groupID:        "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Get another user's group",
			userID:         "cccccccc-cccc-cccc-cccc-ccccccccccc1",
			groupID:        "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa2",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "Get non-existent group",
			userID:         "cccccccc-cccc-cccc-cccc-ccccccccccc1",
			groupID:        "99999999-9999-9999-9999-999999999999",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Get("/groups/:id", func(c *fiber.Ctx) error {
				c.Locals("user_id", tt.userID)
				return handler.GetGroup(c)
			})

			req := httptest.NewRequest(http.MethodGet, "/groups/"+tt.groupID, nil)
			resp, err := app.Test(req, -1)
			if err != nil {
				t.Fatalf("Failed to execute request: %v", err)
			}

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}
		})
	}
}

func TestGroupHandler_UpdateGroup(t *testing.T) {
	db := setupGroupTestDB(t)
	defer db.Close()

	groupRepo := repository.NewGroupRepository(db)
	handler := NewGroupHandler(groupRepo)

	// Create test groups
	createGroupDirectly(t, db, &models.Group{
		ID:        "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1",
		UserID:    "cccccccc-cccc-cccc-cccc-ccccccccccc1",
		Name:      "Group 1",
		Color:     "#ff0000",
		Position:  0,
		IsDefault: true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	createGroupDirectly(t, db, &models.Group{
		ID:        "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa2",
		UserID:    "cccccccc-cccc-cccc-cccc-ccccccccccc2",
		Name:      "Group 2",
		Color:     "#00ff00",
		Position:  0,
		IsDefault: true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	tests := []struct {
		name           string
		userID         string
		groupID        string
		requestBody    models.GroupUpdateRequest
		expectedStatus int
		expectError    bool
	}{
		{
			name:    "Successfully update group",
			userID:  "cccccccc-cccc-cccc-cccc-ccccccccccc1",
			groupID: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1",
			requestBody: models.GroupUpdateRequest{
				Name:  "Updated Name",
				Color: "#00ff00",
			},
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:    "Update only name",
			userID:  "cccccccc-cccc-cccc-cccc-ccccccccccc1",
			groupID: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1",
			requestBody: models.GroupUpdateRequest{
				Name: "Another Name",
			},
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:    "Update only color",
			userID:  "cccccccc-cccc-cccc-cccc-ccccccccccc1",
			groupID: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1",
			requestBody: models.GroupUpdateRequest{
				Color: "#0000ff",
			},
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:    "Invalid color format",
			userID:  "cccccccc-cccc-cccc-cccc-ccccccccccc1",
			groupID: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1",
			requestBody: models.GroupUpdateRequest{
				Color: "not-a-color",
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name:    "Name too long",
			userID:  "cccccccc-cccc-cccc-cccc-ccccccccccc1",
			groupID: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1",
			requestBody: models.GroupUpdateRequest{
				Name: string(make([]byte, 36)), // 36 chars exceeds MaxGroupNameLen (35)
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name:    "Update another user's group",
			userID:  "cccccccc-cccc-cccc-cccc-ccccccccccc1",
			groupID: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa2",
			requestBody: models.GroupUpdateRequest{
				Name: "Hacked",
			},
			expectedStatus: http.StatusForbidden,
			expectError:    true,
		},
		{
			name:    "Update non-existent group",
			userID:  "cccccccc-cccc-cccc-cccc-ccccccccccc1",
			groupID: "99999999-9999-9999-9999-999999999999",
			requestBody: models.GroupUpdateRequest{
				Name: "Updated",
			},
			expectedStatus: http.StatusNotFound,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Put("/groups/:id", func(c *fiber.Ctx) error {
				c.Locals("user_id", tt.userID)
				return handler.UpdateGroup(c)
			})

			bodyJSON, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPut, "/groups/"+tt.groupID, bytes.NewReader(bodyJSON))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req, -1)
			if err != nil {
				t.Fatalf("Failed to execute request: %v", err)
			}

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			// Verify update was applied (if success expected)
			if !tt.expectError && resp.StatusCode == http.StatusOK {
				group, err := groupRepo.GetByID(context.Background(), tt.groupID)
				if err != nil {
					t.Fatalf("Failed to retrieve group: %v", err)
				}
				if tt.requestBody.Name != "" && group.Name != tt.requestBody.Name {
					t.Errorf("Group name = %s, want %s", group.Name, tt.requestBody.Name)
				}
				if tt.requestBody.Color != "" && group.Color != tt.requestBody.Color {
					t.Errorf("Group color = %s, want %s", group.Color, tt.requestBody.Color)
				}
			}
		})
	}
}

func TestGroupHandler_DeleteGroup(t *testing.T) {
	db := setupGroupTestDB(t)
	defer db.Close()

	groupRepo := repository.NewGroupRepository(db)
	handler := NewGroupHandler(groupRepo)

	// Create test groups
	createGroupDirectly(t, db, &models.Group{
		ID:        "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1",
		UserID:    "cccccccc-cccc-cccc-cccc-ccccccccccc1",
		Name:      "Regular Group",
		Color:     "#ff0000",
		Position:  1,
		IsDefault: false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	createGroupDirectly(t, db, &models.Group{
		ID:        "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa2",
		UserID:    "cccccccc-cccc-cccc-cccc-ccccccccccc1",
		Name:      "Default Group",
		Color:     "#6366f1",
		Position:  0,
		IsDefault: true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	createGroupDirectly(t, db, &models.Group{
		ID:        "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa3",
		UserID:    "cccccccc-cccc-cccc-cccc-ccccccccccc2",
		Name:      "Other User Group",
		Color:     "#0000ff",
		Position:  0,
		IsDefault: true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	tests := []struct {
		name           string
		userID         string
		groupID        string
		expectedStatus int
	}{
		{
			name:           "Delete regular group",
			userID:         "cccccccc-cccc-cccc-cccc-ccccccccccc1",
			groupID:        "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Delete default group (should fail)",
			userID:         "cccccccc-cccc-cccc-cccc-ccccccccccc1",
			groupID:        "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa2",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "Delete another user's group",
			userID:         "cccccccc-cccc-cccc-cccc-ccccccccccc1",
			groupID:        "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa3",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Delete non-existent group",
			userID:         "cccccccc-cccc-cccc-cccc-ccccccccccc1",
			groupID:        "99999999-9999-9999-9999-999999999999",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Delete("/groups/:id", func(c *fiber.Ctx) error {
				c.Locals("user_id", tt.userID)
				return handler.DeleteGroup(c)
			})

			req := httptest.NewRequest(http.MethodDelete, "/groups/"+tt.groupID, nil)
			resp, err := app.Test(req, -1)
			if err != nil {
				t.Fatalf("Failed to execute request: %v", err)
			}

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}
		})
	}
}

func TestGroupHandler_ReorderGroups(t *testing.T) {
	db := setupGroupTestDB(t)
	defer db.Close()

	groupRepo := repository.NewGroupRepository(db)
	handler := NewGroupHandler(groupRepo)

	// Create test groups
	groups := []*models.Group{
		{
			ID:        "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1",
			UserID:    "cccccccc-cccc-cccc-cccc-ccccccccccc1",
			Name:      "Group 1",
			Color:     "#ff0000",
			Position:  0,
			IsDefault: true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:        "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa2",
			UserID:    "cccccccc-cccc-cccc-cccc-ccccccccccc1",
			Name:      "Group 2",
			Color:     "#00ff00",
			Position:  1,
			IsDefault: false,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:        "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa3",
			UserID:    "cccccccc-cccc-cccc-cccc-ccccccccccc2",
			Name:      "Group 3",
			Color:     "#0000ff",
			Position:  0,
			IsDefault: true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	for _, g := range groups {
		createGroupDirectly(t, db, g)
	}

	tests := []struct {
		name           string
		userID         string
		requestBody    models.GroupReorderRequest
		expectedStatus int
		expectError    bool
	}{
		{
			name:   "Successfully reorder groups",
			userID: "cccccccc-cccc-cccc-cccc-ccccccccccc1",
			requestBody: models.GroupReorderRequest{
				Groups: []models.GroupPosition{
					{ID: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1", Position: 1},
					{ID: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa2", Position: 0},
				},
			},
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:   "Reorder single group",
			userID: "cccccccc-cccc-cccc-cccc-ccccccccccc1",
			requestBody: models.GroupReorderRequest{
				Groups: []models.GroupPosition{
					{ID: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1", Position: 1}, // Valid position within bounds (user-1 has 2 groups)
				},
			},
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:   "Attempt to reorder another user's group",
			userID: "cccccccc-cccc-cccc-cccc-ccccccccccc1",
			requestBody: models.GroupReorderRequest{
				Groups: []models.GroupPosition{
					{ID: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa3", Position: 0}, // Valid position, but wrong user
				},
			},
			expectedStatus: http.StatusNotFound,
			expectError:    true,
		},
		{
			name:   "Position out of bounds",
			userID: "cccccccc-cccc-cccc-cccc-ccccccccccc1",
			requestBody: models.GroupReorderRequest{
				Groups: []models.GroupPosition{
					{ID: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1", Position: 10}, // Invalid: user-1 only has 2 groups (positions 0-1)
				},
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name:   "Empty group ID",
			userID: "cccccccc-cccc-cccc-cccc-ccccccccccc1",
			requestBody: models.GroupReorderRequest{
				Groups: []models.GroupPosition{
					{ID: "", Position: 0},
				},
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name:   "Negative position",
			userID: "cccccccc-cccc-cccc-cccc-ccccccccccc1",
			requestBody: models.GroupReorderRequest{
				Groups: []models.GroupPosition{
					{ID: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1", Position: -1},
				},
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name:   "Empty groups array",
			userID: "cccccccc-cccc-cccc-cccc-ccccccccccc1",
			requestBody: models.GroupReorderRequest{
				Groups: []models.GroupPosition{},
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name:   "Non-existent group",
			userID: "cccccccc-cccc-cccc-cccc-ccccccccccc1",
			requestBody: models.GroupReorderRequest{
				Groups: []models.GroupPosition{
					{ID: "99999999-9999-9999-9999-999999999999", Position: 0},
				},
			},
			expectedStatus: http.StatusNotFound,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Put("/groups/reorder", func(c *fiber.Ctx) error {
				c.Locals("user_id", tt.userID)
				return handler.ReorderGroups(c)
			})

			bodyJSON, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPut, "/groups/reorder", bytes.NewReader(bodyJSON))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req, -1)
			if err != nil {
				t.Fatalf("Failed to execute request: %v", err)
			}

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			// Verify positions were updated correctly (if success expected)
			if !tt.expectError && resp.StatusCode == http.StatusOK {
				ctx := context.Background()
				for _, gp := range tt.requestBody.Groups {
					group, err := groupRepo.GetByID(ctx, gp.ID)
					if err != nil {
						t.Fatalf("Failed to retrieve group %s: %v", gp.ID, err)
					}
					if group.Position != gp.Position {
						t.Errorf("Group %s position = %d, want %d", gp.ID, group.Position, gp.Position)
					}
				}
			}
		})
	}
}

func TestGroupHandler_ReorderGroups_NoAuth(t *testing.T) {
	db := setupGroupTestDB(t)
	defer db.Close()

	groupRepo := repository.NewGroupRepository(db)
	handler := NewGroupHandler(groupRepo)

	app := fiber.New()
	app.Put("/groups/reorder", handler.ReorderGroups)

	requestBody := models.GroupReorderRequest{
		Groups: []models.GroupPosition{
			{ID: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1", Position: 0},
		},
	}

	bodyJSON, _ := json.Marshal(requestBody)
	req := httptest.NewRequest(http.MethodPut, "/groups/reorder", bytes.NewReader(bodyJSON))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status %d for unauthenticated request, got %d", http.StatusUnauthorized, resp.StatusCode)
	}
}

func TestGroupHandler_ReorderGroups_InvalidJSON(t *testing.T) {
	db := setupGroupTestDB(t)
	defer db.Close()

	groupRepo := repository.NewGroupRepository(db)
	handler := NewGroupHandler(groupRepo)

	app := fiber.New()
	app.Put("/groups/reorder", func(c *fiber.Ctx) error {
		c.Locals("user_id", "cccccccc-cccc-cccc-cccc-ccccccccccc1")
		return handler.ReorderGroups(c)
	})

	req := httptest.NewRequest(http.MethodPut, "/groups/reorder", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status %d for invalid JSON, got %d", http.StatusBadRequest, resp.StatusCode)
	}
}

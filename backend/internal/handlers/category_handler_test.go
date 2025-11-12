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

// setupCategoryTestDB creates an in-memory SQLite database for testing categories
func setupCategoryTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	schema := `
		CREATE TABLE IF NOT EXISTS categories (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			name TEXT NOT NULL,
			color TEXT DEFAULT '#6366f1',
			position INTEGER DEFAULT 0,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL
		);
	`

	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}

	return db
}

// createCategoryDirectly inserts a category for testing
func createCategoryDirectly(t *testing.T, db *sql.DB, category *models.Category) {
	query := `
		INSERT INTO categories (id, user_id, name, color, position, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	_, err := db.Exec(
		query,
		category.ID,
		category.UserID,
		category.Name,
		category.Color,
		category.Position,
		category.CreatedAt,
		category.UpdatedAt,
	)
	if err != nil {
		t.Fatalf("Failed to create category directly: %v", err)
	}
}

func TestCategoryHandler_CreateCategory(t *testing.T) {
	db := setupCategoryTestDB(t)
	defer db.Close()

	categoryRepo := repository.NewCategoryRepository(db)
	handler := NewCategoryHandler(categoryRepo)

	app := fiber.New()
	app.Post("/categories", func(c *fiber.Ctx) error {
		c.Locals("user_id", "user-1")
		return handler.CreateCategory(c)
	})

	tests := []struct {
		name           string
		reqBody        models.CategoryCreateRequest
		wantStatusCode int
	}{
		{
			name: "Create valid category",
			reqBody: models.CategoryCreateRequest{
				Name:  "Work",
				Color: "#6366f1",
			},
			wantStatusCode: fiber.StatusCreated,
		},
		{
			name: "Create category without color (uses default)",
			reqBody: models.CategoryCreateRequest{
				Name: "Personal",
			},
			wantStatusCode: fiber.StatusCreated,
		},
		{
			name: "Create category without name",
			reqBody: models.CategoryCreateRequest{
				Color: "#6366f1",
			},
			wantStatusCode: fiber.StatusBadRequest,
		},
		{
			name: "Create category with invalid hex color",
			reqBody: models.CategoryCreateRequest{
				Name:  "Test",
				Color: "invalid",
			},
			wantStatusCode: fiber.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.reqBody)
			req := httptest.NewRequest(http.MethodPost, "/categories", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}

			if resp.StatusCode != tt.wantStatusCode {
				t.Errorf("Status code = %d, want %d", resp.StatusCode, tt.wantStatusCode)
			}
		})
	}
}

func TestCategoryHandler_GetCategories(t *testing.T) {
	db := setupCategoryTestDB(t)
	defer db.Close()

	categoryRepo := repository.NewCategoryRepository(db)
	handler := NewCategoryHandler(categoryRepo)

	// Create test categories
	categories := []*models.Category{
		{
			ID:        "cat-1",
			UserID:    "user-1",
			Name:      "Work",
			Color:     "#6366f1",
			Position:  0,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:        "cat-2",
			UserID:    "user-1",
			Name:      "Personal",
			Color:     "#f43f5e",
			Position:  1,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	for _, cat := range categories {
		createCategoryDirectly(t, db, cat)
	}

	app := fiber.New()
	app.Get("/categories", func(c *fiber.Ctx) error {
		c.Locals("user_id", "user-1")
		return handler.GetCategories(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/categories", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("Status code = %d, want %d", resp.StatusCode, fiber.StatusOK)
	}

	var result []models.CategoryResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("Got %d categories, want 2", len(result))
	}
}

func TestCategoryHandler_GetCategory(t *testing.T) {
	db := setupCategoryTestDB(t)
	defer db.Close()

	categoryRepo := repository.NewCategoryRepository(db)
	handler := NewCategoryHandler(categoryRepo)

	// Create test category
	category := &models.Category{
		ID:        "cat-1",
		UserID:    "user-1",
		Name:      "Work",
		Color:     "#6366f1",
		Position:  0,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	createCategoryDirectly(t, db, category)

	app := fiber.New()
	app.Get("/categories/:id", func(c *fiber.Ctx) error {
		c.Locals("user_id", "user-1")
		return handler.GetCategory(c)
	})

	tests := []struct {
		name           string
		categoryID     string
		wantStatusCode int
	}{
		{
			name:           "Get existing category",
			categoryID:     "cat-1",
			wantStatusCode: fiber.StatusOK,
		},
		{
			name:           "Get non-existent category",
			categoryID:     "non-existent",
			wantStatusCode: fiber.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/categories/"+tt.categoryID, nil)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}

			if resp.StatusCode != tt.wantStatusCode {
				t.Errorf("Status code = %d, want %d", resp.StatusCode, tt.wantStatusCode)
			}
		})
	}
}

func TestCategoryHandler_UpdateCategory(t *testing.T) {
	db := setupCategoryTestDB(t)
	defer db.Close()

	categoryRepo := repository.NewCategoryRepository(db)
	handler := NewCategoryHandler(categoryRepo)

	// Create test category
	category := &models.Category{
		ID:        "cat-1",
		UserID:    "user-1",
		Name:      "Work",
		Color:     "#6366f1",
		Position:  0,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	createCategoryDirectly(t, db, category)

	app := fiber.New()
	app.Put("/categories/:id", func(c *fiber.Ctx) error {
		c.Locals("user_id", "user-1")
		return handler.UpdateCategory(c)
	})

	tests := []struct {
		name           string
		categoryID     string
		reqBody        models.CategoryUpdateRequest
		wantStatusCode int
	}{
		{
			name:       "Update category successfully",
			categoryID: "cat-1",
			reqBody: models.CategoryUpdateRequest{
				Name:  "Professional",
				Color: "#f43f5e",
			},
			wantStatusCode: fiber.StatusOK,
		},
		{
			name:       "Update non-existent category",
			categoryID: "non-existent",
			reqBody: models.CategoryUpdateRequest{
				Name:  "Test",
				Color: "#6366f1",
			},
			wantStatusCode: fiber.StatusNotFound,
		},
		{
			name:       "Update with empty name",
			categoryID: "cat-1",
			reqBody: models.CategoryUpdateRequest{
				Name:  "",
				Color: "#6366f1",
			},
			wantStatusCode: fiber.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.reqBody)
			req := httptest.NewRequest(http.MethodPut, "/categories/"+tt.categoryID, bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}

			if resp.StatusCode != tt.wantStatusCode {
				t.Errorf("Status code = %d, want %d", resp.StatusCode, tt.wantStatusCode)
			}
		})
	}
}

func TestCategoryHandler_DeleteCategory(t *testing.T) {
	db := setupCategoryTestDB(t)
	defer db.Close()

	categoryRepo := repository.NewCategoryRepository(db)
	handler := NewCategoryHandler(categoryRepo)

	// Create test category
	category := &models.Category{
		ID:        "cat-1",
		UserID:    "user-1",
		Name:      "Work",
		Color:     "#6366f1",
		Position:  0,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	createCategoryDirectly(t, db, category)

	app := fiber.New()
	app.Delete("/categories/:id", func(c *fiber.Ctx) error {
		c.Locals("user_id", "user-1")
		return handler.DeleteCategory(c)
	})

	tests := []struct {
		name           string
		categoryID     string
		wantStatusCode int
	}{
		{
			name:           "Delete existing category",
			categoryID:     "cat-1",
			wantStatusCode: fiber.StatusNoContent,
		},
		{
			name:           "Delete non-existent category",
			categoryID:     "non-existent",
			wantStatusCode: fiber.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodDelete, "/categories/"+tt.categoryID, nil)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}

			if resp.StatusCode != tt.wantStatusCode {
				t.Errorf("Status code = %d, want %d", resp.StatusCode, tt.wantStatusCode)
			}
		})
	}
}

func TestCategoryHandler_ReorderCategories(t *testing.T) {
	db := setupCategoryTestDB(t)
	defer db.Close()

	categoryRepo := repository.NewCategoryRepository(db)
	handler := NewCategoryHandler(categoryRepo)

	// Create test categories
	categories := []*models.Category{
		{
			ID:        "cat-1",
			UserID:    "user-1",
			Name:      "Work",
			Color:     "#6366f1",
			Position:  0,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:        "cat-2",
			UserID:    "user-1",
			Name:      "Personal",
			Color:     "#f43f5e",
			Position:  1,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	for _, cat := range categories {
		createCategoryDirectly(t, db, cat)
	}

	app := fiber.New()
	app.Put("/categories/reorder", func(c *fiber.Ctx) error {
		c.Locals("user_id", "user-1")
		return handler.ReorderCategories(c)
	})

	tests := []struct {
		name           string
		reqBody        models.CategoryReorderRequest
		wantStatusCode int
	}{
		{
			name: "Reorder categories successfully",
			reqBody: models.CategoryReorderRequest{
				Categories: []models.CategoryPosition{
					{ID: "cat-1", Position: 1},
					{ID: "cat-2", Position: 0},
				},
			},
			wantStatusCode: fiber.StatusNoContent,
		},
		{
			name: "Reorder with empty array",
			reqBody: models.CategoryReorderRequest{
				Categories: []models.CategoryPosition{},
			},
			wantStatusCode: fiber.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.reqBody)
			req := httptest.NewRequest(http.MethodPut, "/categories/reorder", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}

			if resp.StatusCode != tt.wantStatusCode {
				t.Errorf("Status code = %d, want %d", resp.StatusCode, tt.wantStatusCode)
			}

			// Verify positions were updated for successful case
			if tt.wantStatusCode == fiber.StatusNoContent {
				for _, cp := range tt.reqBody.Categories {
					cat, err := categoryRepo.GetByID(context.Background(), cp.ID)
					if err != nil {
						t.Errorf("Failed to get category after reorder: %v", err)
						continue
					}
					if cat.Position != cp.Position {
						t.Errorf("Category %s position = %d, want %d", cp.ID, cat.Position, cp.Position)
					}
				}
			}
		})
	}
}

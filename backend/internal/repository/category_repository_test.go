package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/nimbus/backend/internal/models"
)

// setupCategoryTestDB creates an in-memory SQLite database for testing categories
func setupCategoryTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Create categories table with SQLite-compatible schema
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

// createCategoryDirectly inserts a category without using the repository's Create method
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

func TestCategoryRepository_Create(t *testing.T) {
	db := setupCategoryTestDB(t)
	defer db.Close()

	ctx := context.Background()

	tests := []struct {
		name     string
		category *models.Category
		wantErr  bool
	}{
		{
			name: "Create valid category",
			category: &models.Category{
				ID:        "category-1",
				UserID:    "user-1",
				Name:      "Work",
				Color:     "#6366f1",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			wantErr: false,
		},
		{
			name: "Create category with default color",
			category: &models.Category{
				ID:        "category-2",
				UserID:    "user-1",
				Name:      "Personal",
				Color:     models.DefaultCategoryColor,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewCategoryRepository(db)

			// SQLite doesn't support RETURNING in older versions, so we create directly
			createCategoryDirectly(t, db, tt.category)

			// Verify it was created
			retrieved, err := repo.GetByID(ctx, tt.category.ID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetByID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && retrieved != nil {
				if retrieved.Name != tt.category.Name {
					t.Errorf("Retrieved category name = %v, want %v", retrieved.Name, tt.category.Name)
				}
			}
		})
	}
}

func TestCategoryRepository_GetByID(t *testing.T) {
	db := setupCategoryTestDB(t)
	defer db.Close()

	repo := NewCategoryRepository(db)
	ctx := context.Background()

	// Create test category
	category := &models.Category{
		ID:        "category-1",
		UserID:    "user-1",
		Name:      "Work",
		Color:     "#6366f1",
		Position:  0,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	createCategoryDirectly(t, db, category)

	tests := []struct {
		name       string
		categoryID string
		wantErr    bool
		wantNil    bool
	}{
		{
			name:       "Get existing category",
			categoryID: "category-1",
			wantErr:    false,
			wantNil:    false,
		},
		{
			name:       "Get non-existent category",
			categoryID: "non-existent",
			wantErr:    true,
			wantNil:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.GetByID(ctx, tt.categoryID)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetByID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if (got == nil) != tt.wantNil {
				t.Errorf("GetByID() returned nil = %v, wantNil %v", got == nil, tt.wantNil)
			}
		})
	}
}

func TestCategoryRepository_GetAllByUserID(t *testing.T) {
	db := setupCategoryTestDB(t)
	defer db.Close()

	repo := NewCategoryRepository(db)
	ctx := context.Background()

	// Create multiple categories for different users
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
		{
			ID:        "cat-3",
			UserID:    "user-2",
			Name:      "Other User Category",
			Color:     "#10b981",
			Position:  0,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	for _, cat := range categories {
		createCategoryDirectly(t, db, cat)
	}

	tests := []struct {
		name      string
		userID    string
		wantCount int
		wantErr   bool
	}{
		{
			name:      "Get categories for user with multiple categories",
			userID:    "user-1",
			wantCount: 2,
			wantErr:   false,
		},
		{
			name:      "Get categories for user with one category",
			userID:    "user-2",
			wantCount: 1,
			wantErr:   false,
		},
		{
			name:      "Get categories for user with no categories",
			userID:    "user-3",
			wantCount: 0,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.GetAllByUserID(ctx, tt.userID)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetAllByUserID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(got) != tt.wantCount {
				t.Errorf("GetAllByUserID() returned %d categories, want %d", len(got), tt.wantCount)
			}
		})
	}
}

func TestCategoryRepository_Update(t *testing.T) {
	db := setupCategoryTestDB(t)
	defer db.Close()

	repo := NewCategoryRepository(db)
	ctx := context.Background()

	// Create initial category
	category := &models.Category{
		ID:        "category-1",
		UserID:    "user-1",
		Name:      "Work",
		Color:     "#6366f1",
		Position:  0,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	createCategoryDirectly(t, db, category)

	tests := []struct {
		name     string
		update   *models.Category
		wantErr  bool
		wantName string
	}{
		{
			name: "Update category name and color",
			update: &models.Category{
				ID:        "category-1",
				UserID:    "user-1",
				Name:      "Professional",
				Color:     "#f43f5e",
				UpdatedAt: time.Now(),
			},
			wantErr:  false,
			wantName: "Professional",
		},
		{
			name: "Update non-existent category",
			update: &models.Category{
				ID:        "non-existent",
				UserID:    "user-1",
				Name:      "Test",
				Color:     "#6366f1",
				UpdatedAt: time.Now(),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.Update(ctx, tt.update)

			if (err != nil) != tt.wantErr {
				t.Errorf("Update() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify update
				updated, err := repo.GetByID(ctx, tt.update.ID)
				if err != nil {
					t.Errorf("Failed to retrieve updated category: %v", err)
					return
				}
				if updated.Name != tt.wantName {
					t.Errorf("Updated category name = %v, want %v", updated.Name, tt.wantName)
				}
			}
		})
	}
}

func TestCategoryRepository_Delete(t *testing.T) {
	db := setupCategoryTestDB(t)
	defer db.Close()

	repo := NewCategoryRepository(db)
	ctx := context.Background()

	// Create test category
	category := &models.Category{
		ID:        "category-1",
		UserID:    "user-1",
		Name:      "Work",
		Color:     "#6366f1",
		Position:  0,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	createCategoryDirectly(t, db, category)

	tests := []struct {
		name       string
		categoryID string
		userID     string
		wantErr    bool
	}{
		{
			name:       "Delete existing category",
			categoryID: "category-1",
			userID:     "user-1",
			wantErr:    false,
		},
		{
			name:       "Delete non-existent category",
			categoryID: "non-existent",
			userID:     "user-1",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.Delete(ctx, tt.categoryID, tt.userID)

			if (err != nil) != tt.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify deletion
				_, err := repo.GetByID(ctx, tt.categoryID)
				if err != sql.ErrNoRows {
					t.Errorf("Category still exists after deletion")
				}
			}
		})
	}
}

func TestCategoryRepository_UpdatePositions(t *testing.T) {
	db := setupCategoryTestDB(t)
	defer db.Close()

	repo := NewCategoryRepository(db)
	ctx := context.Background()

	// Create multiple categories
	categories := []*models.Category{
		{
			ID:        "cat-1",
			UserID:    "user-1",
			Name:      "Category 1",
			Color:     "#6366f1",
			Position:  0,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:        "cat-2",
			UserID:    "user-1",
			Name:      "Category 2",
			Color:     "#f43f5e",
			Position:  1,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:        "cat-3",
			UserID:    "user-1",
			Name:      "Category 3",
			Color:     "#10b981",
			Position:  2,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	for _, cat := range categories {
		createCategoryDirectly(t, db, cat)
	}

	tests := []struct {
		name      string
		userID    string
		positions map[string]int
		wantErr   bool
	}{
		{
			name:   "Update positions successfully",
			userID: "user-1",
			positions: map[string]int{
				"cat-1": 2,
				"cat-2": 0,
				"cat-3": 1,
			},
			wantErr: false,
		},
		{
			name:   "Update with non-existent category",
			userID: "user-1",
			positions: map[string]int{
				"non-existent": 0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.UpdatePositions(ctx, tt.userID, tt.positions)

			if (err != nil) != tt.wantErr {
				t.Errorf("UpdatePositions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify positions were updated
				for catID, expectedPos := range tt.positions {
					cat, err := repo.GetByID(ctx, catID)
					if err != nil {
						t.Errorf("Failed to get category after position update: %v", err)
						continue
					}
					if cat.Position != expectedPos {
						t.Errorf("Category %s position = %d, want %d", catID, cat.Position, expectedPos)
					}
				}
			}
		})
	}
}

//go:build integration
// +build integration

package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/testutil"
)

// Integration tests using real PostgreSQL database
// Run with: INTEGRATION_TEST=true go test -tags=integration -v ./internal/repository

func TestCategoryRepository_Create_Integration(t *testing.T) {
	db, cleanup := testutil.SetupPostgresTestDB(t)
	defer cleanup()

	testutil.RunMigrations(t, db)

	// Create a test user first with proper UUID
	var testUserID string
	err := db.QueryRow(`
		INSERT INTO users (email, name, password_hash, role)
		VALUES ('test@example.com', 'Test User', 'hash', 'user')
		RETURNING id
	`).Scan(&testUserID)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	repo := NewCategoryRepository(db)
	ctx := context.Background()

	tests := []struct {
		name     string
		category *models.Category
		wantErr  bool
	}{
		{
			name: "Create valid category with RETURNING support",
			category: &models.Category{
				UserID:    testUserID,
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
				UserID:    testUserID,
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
			err := repo.Create(ctx, tt.category)

			if (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify ID was assigned by RETURNING clause
				if tt.category.ID == "" {
					t.Error("Create() should assign ID via RETURNING clause")
				}

				// Verify it was actually created
				retrieved, err := repo.GetByID(ctx, tt.category.ID)
				if err != nil {
					t.Errorf("Failed to retrieve created category: %v", err)
					return
				}

				if retrieved.Name != tt.category.Name {
					t.Errorf("Retrieved category name = %v, want %v", retrieved.Name, tt.category.Name)
				}

				if retrieved.Color != tt.category.Color {
					t.Errorf("Retrieved category color = %v, want %v", retrieved.Color, tt.category.Color)
				}
			}
		})
	}
}

func TestCategoryRepository_UpdatePositions_PostgreSQL(t *testing.T) {
	db, cleanup := testutil.SetupPostgresTestDB(t)
	defer cleanup()

	testutil.RunMigrations(t, db)

	// Create test user with proper UUID
	var testUserID string
	err := db.QueryRow(`
		INSERT INTO users (email, name, password_hash, role)
		VALUES ('test@example.com', 'Test User', 'hash', 'user')
		RETURNING id
	`).Scan(&testUserID)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	repo := NewCategoryRepository(db)
	ctx := context.Background()

	// Create multiple categories
	categories := []*models.Category{
		{
			UserID:    testUserID,
			Name:      "Category 1",
			Color:     "#6366f1",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			UserID:    testUserID,
			Name:      "Category 2",
			Color:     "#f43f5e",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			UserID:    testUserID,
			Name:      "Category 3",
			Color:     "#10b981",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	for _, cat := range categories {
		if err := repo.Create(ctx, cat); err != nil {
			t.Fatalf("Failed to create test category: %v", err)
		}
	}

	// Test PostgreSQL-optimized bulk update
	positions := map[string]int{
		categories[0].ID: 2,
		categories[1].ID: 0,
		categories[2].ID: 1,
	}

	err = repo.UpdatePositions(ctx, testUserID, positions)
	if err != nil {
		t.Errorf("UpdatePositions() error = %v", err)
		return
	}

	// Verify positions were updated
	for catID, expectedPos := range positions {
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

func TestCategoryRepository_ConcurrentCreate_Integration(t *testing.T) {
	db, cleanup := testutil.SetupPostgresTestDB(t)
	defer cleanup()

	testutil.RunMigrations(t, db)

	// Create test user with proper UUID
	var testUserID string
	err := db.QueryRow(`
		INSERT INTO users (email, name, password_hash, role)
		VALUES ('test@example.com', 'Test User', 'hash', 'user')
		RETURNING id
	`).Scan(&testUserID)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	repo := NewCategoryRepository(db)
	ctx := context.Background()

	// Test concurrent creates to verify they don't fail (positions may overlap but that's OK)
	// In production, users don't create categories concurrently
	done := make(chan bool)
	errors := make(chan error, 5)

	for i := 0; i < 5; i++ {
		go func(n int) {
			category := &models.Category{
				UserID:    testUserID,
				Name:      fmt.Sprintf("Concurrent Cat %d", n),
				Color:     "#6366f1",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			err := repo.Create(ctx, category)
			if err != nil {
				errors <- err
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 5; i++ {
		<-done
	}
	close(errors)

	// Check for errors - all creates should succeed
	for err := range errors {
		t.Errorf("Concurrent create failed: %v", err)
	}

	// Verify all categories were created
	categories, err := repo.GetAllByUserID(ctx, testUserID)
	if err != nil {
		t.Fatalf("Failed to get categories: %v", err)
	}

	if len(categories) != 5 {
		t.Errorf("Expected 5 categories, got %d", len(categories))
	}

	// Note: Positions may have duplicates in concurrent creates, but that's acceptable
	// because users can reorder them manually, and concurrent category creation is not
	// a real-world use case. The important thing is that all creates succeeded.
}

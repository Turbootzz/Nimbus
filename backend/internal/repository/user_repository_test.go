package repository

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/nimbus/backend/internal/models"
)

// setupUserTestDB creates an in-memory SQLite database for testing
func setupUserTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Create users table (SQLite syntax)
	usersSchema := `
		CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
			email TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL,
			password TEXT NOT NULL,
			role TEXT NOT NULL DEFAULT 'user',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
	`

	if _, err := db.Exec(usersSchema); err != nil {
		t.Fatalf("Failed to create users table: %v", err)
	}

	return db
}

func TestUserRepository_UpdateRole(t *testing.T) {
	db := setupUserTestDB(t)
	defer db.Close()

	repo := NewUserRepository(db)

	// Create a test user
	user := &models.User{
		Email:     "test@example.com",
		Name:      "Test User",
		Password:  "hashedpassword",
		Role:      "user",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := repo.Create(user); err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	t.Run("Update role from user to admin", func(t *testing.T) {
		err := repo.UpdateRole(user.ID, "admin")
		if err != nil {
			t.Errorf("Failed to update role: %v", err)
		}

		// Verify role was updated
		updatedUser, err := repo.GetByID(user.ID)
		if err != nil {
			t.Errorf("Failed to get updated user: %v", err)
		}

		if updatedUser.Role != "admin" {
			t.Errorf("Expected role 'admin', got '%s'", updatedUser.Role)
		}
	})

	t.Run("Update role from admin to user", func(t *testing.T) {
		err := repo.UpdateRole(user.ID, "user")
		if err != nil {
			t.Errorf("Failed to update role: %v", err)
		}

		// Verify role was updated
		updatedUser, err := repo.GetByID(user.ID)
		if err != nil {
			t.Errorf("Failed to get updated user: %v", err)
		}

		if updatedUser.Role != "user" {
			t.Errorf("Expected role 'user', got '%s'", updatedUser.Role)
		}
	})

	t.Run("Update role for non-existent user", func(t *testing.T) {
		err := repo.UpdateRole("non-existent-id", "admin")
		if err == nil {
			t.Error("Expected error for non-existent user, got nil")
		}
	})
}

func TestUserRepository_Delete(t *testing.T) {
	db := setupUserTestDB(t)
	defer db.Close()

	repo := NewUserRepository(db)

	t.Run("Delete existing user", func(t *testing.T) {
		// Create a test user
		user := &models.User{
			Email:     "delete@example.com",
			Name:      "Delete User",
			Password:  "hashedpassword",
			Role:      "user",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		if err := repo.Create(user); err != nil {
			t.Fatalf("Failed to create test user: %v", err)
		}

		// Delete the user
		err := repo.Delete(user.ID)
		if err != nil {
			t.Errorf("Failed to delete user: %v", err)
		}

		// Verify user was deleted
		_, err = repo.GetByID(user.ID)
		if err == nil {
			t.Error("Expected error when getting deleted user, got nil")
		}
	})

	t.Run("Delete non-existent user", func(t *testing.T) {
		err := repo.Delete("non-existent-id")
		if err == nil {
			t.Error("Expected error for non-existent user, got nil")
		}
	})
}

func TestUserRepository_GetStats(t *testing.T) {
	db := setupUserTestDB(t)
	defer db.Close()

	repo := NewUserRepository(db)

	// Create test users with different roles
	users := []*models.User{
		{
			Email:     "admin1@example.com",
			Name:      "Admin One",
			Password:  "hashedpassword",
			Role:      "admin",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Email:     "admin2@example.com",
			Name:      "Admin Two",
			Password:  "hashedpassword",
			Role:      "admin",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Email:     "user1@example.com",
			Name:      "User One",
			Password:  "hashedpassword",
			Role:      "user",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Email:     "user2@example.com",
			Name:      "User Two",
			Password:  "hashedpassword",
			Role:      "user",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Email:     "user3@example.com",
			Name:      "User Three",
			Password:  "hashedpassword",
			Role:      "user",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	for _, user := range users {
		if err := repo.Create(user); err != nil {
			t.Fatalf("Failed to create test user: %v", err)
		}
	}

	t.Run("Get user statistics", func(t *testing.T) {
		stats, err := repo.GetStats()
		if err != nil {
			t.Fatalf("Failed to get stats: %v", err)
		}

		expectedTotal := 5
		expectedAdmins := 2
		expectedUsers := 3

		if stats["total"] != expectedTotal {
			t.Errorf("Expected total %d, got %d", expectedTotal, stats["total"])
		}

		if stats["admins"] != expectedAdmins {
			t.Errorf("Expected admins %d, got %d", expectedAdmins, stats["admins"])
		}

		if stats["users"] != expectedUsers {
			t.Errorf("Expected users %d, got %d", expectedUsers, stats["users"])
		}
	})

	t.Run("Get stats with empty database", func(t *testing.T) {
		// Create a fresh database
		freshDB := setupUserTestDB(t)
		defer freshDB.Close()

		freshRepo := NewUserRepository(freshDB)

		stats, err := freshRepo.GetStats()
		if err != nil {
			t.Fatalf("Failed to get stats: %v", err)
		}

		if stats["total"] != 0 {
			t.Errorf("Expected total 0, got %d", stats["total"])
		}

		if stats["admins"] != 0 {
			t.Errorf("Expected admins 0, got %d", stats["admins"])
		}

		if stats["users"] != 0 {
			t.Errorf("Expected users 0, got %d", stats["users"])
		}
	})
}

func TestUserRepository_GetAll(t *testing.T) {
	db := setupUserTestDB(t)
	defer db.Close()

	repo := NewUserRepository(db)

	// Create multiple test users
	users := []*models.User{
		{
			Email:     "user1@example.com",
			Name:      "User One",
			Password:  "hashedpassword",
			Role:      "user",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Email:     "admin@example.com",
			Name:      "Admin User",
			Password:  "hashedpassword",
			Role:      "admin",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Email:     "user2@example.com",
			Name:      "User Two",
			Password:  "hashedpassword",
			Role:      "user",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	for _, user := range users {
		if err := repo.Create(user); err != nil {
			t.Fatalf("Failed to create test user: %v", err)
		}
	}

	t.Run("Get all users", func(t *testing.T) {
		allUsers, err := repo.GetAll()
		if err != nil {
			t.Fatalf("Failed to get all users: %v", err)
		}

		if len(allUsers) != 3 {
			t.Errorf("Expected 3 users, got %d", len(allUsers))
		}

		// Verify users are ordered by created_at DESC
		// (most recent first - but SQLite doesn't guarantee order without ORDER BY)
		// Just verify we got all users
		emails := make(map[string]bool)
		for _, user := range allUsers {
			emails[user.Email] = true
		}

		expectedEmails := []string{"user1@example.com", "admin@example.com", "user2@example.com"}
		for _, email := range expectedEmails {
			if !emails[email] {
				t.Errorf("Expected to find user with email %s", email)
			}
		}
	})

	t.Run("Get all users from empty database", func(t *testing.T) {
		freshDB := setupUserTestDB(t)
		defer freshDB.Close()

		freshRepo := NewUserRepository(freshDB)

		allUsers, err := freshRepo.GetAll()
		if err != nil {
			t.Fatalf("Failed to get all users: %v", err)
		}

		if len(allUsers) != 0 {
			t.Errorf("Expected 0 users, got %d", len(allUsers))
		}
	})
}

func TestUserRepository_AdminWorkflow(t *testing.T) {
	db := setupUserTestDB(t)
	defer db.Close()

	repo := NewUserRepository(db)

	t.Run("Complete admin workflow", func(t *testing.T) {
		// 1. Create a regular user
		user := &models.User{
			Email:     "workflow@example.com",
			Name:      "Workflow User",
			Password:  "hashedpassword",
			Role:      "user",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		if err := repo.Create(user); err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}

		// 2. Verify initial stats
		stats, err := repo.GetStats()
		if err != nil {
			t.Fatalf("Failed to get stats: %v", err)
		}

		if stats["users"] != 1 || stats["admins"] != 0 {
			t.Errorf("Expected 1 user and 0 admins, got %d users and %d admins",
				stats["users"], stats["admins"])
		}

		// 3. Promote to admin
		if err := repo.UpdateRole(user.ID, "admin"); err != nil {
			t.Fatalf("Failed to promote user: %v", err)
		}

		// 4. Verify stats updated
		stats, err = repo.GetStats()
		if err != nil {
			t.Fatalf("Failed to get stats: %v", err)
		}

		if stats["users"] != 0 || stats["admins"] != 1 {
			t.Errorf("Expected 0 users and 1 admin, got %d users and %d admins",
				stats["users"], stats["admins"])
		}

		// 5. Verify user appears in GetAll
		allUsers, err := repo.GetAll()
		if err != nil {
			t.Fatalf("Failed to get all users: %v", err)
		}

		if len(allUsers) != 1 {
			t.Fatalf("Expected 1 user, got %d", len(allUsers))
		}

		if allUsers[0].Role != "admin" {
			t.Errorf("Expected role 'admin', got '%s'", allUsers[0].Role)
		}

		// 6. Demote back to user
		if err := repo.UpdateRole(user.ID, "user"); err != nil {
			t.Fatalf("Failed to demote user: %v", err)
		}

		// 7. Delete user
		if err := repo.Delete(user.ID); err != nil {
			t.Fatalf("Failed to delete user: %v", err)
		}

		// 8. Verify stats are now zero
		stats, err = repo.GetStats()
		if err != nil {
			t.Fatalf("Failed to get stats: %v", err)
		}

		if stats["total"] != 0 {
			t.Errorf("Expected total 0, got %d", stats["total"])
		}
	})
}

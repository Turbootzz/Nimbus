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

	// Create users table (SQLite syntax) with OAuth support
	usersSchema := `
		CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
			email TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL,
			password TEXT,
			role TEXT NOT NULL DEFAULT 'user',
			provider TEXT NOT NULL DEFAULT 'local',
			provider_id TEXT,
			avatar_url TEXT,
			email_verified INTEGER NOT NULL DEFAULT 0,
			last_activity_at DATETIME,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(provider, provider_id)
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
		Password:  stringPtr("hashedpassword"),
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
			Password:  stringPtr("hashedpassword"),
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
			Password:  stringPtr("hashedpassword"),
			Role:      "admin",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Email:     "admin2@example.com",
			Name:      "Admin Two",
			Password:  stringPtr("hashedpassword"),
			Role:      "admin",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Email:     "user1@example.com",
			Name:      "User One",
			Password:  stringPtr("hashedpassword"),
			Role:      "user",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Email:     "user2@example.com",
			Name:      "User Two",
			Password:  stringPtr("hashedpassword"),
			Role:      "user",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Email:     "user3@example.com",
			Name:      "User Three",
			Password:  stringPtr("hashedpassword"),
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
			Password:  stringPtr("hashedpassword"),
			Role:      "user",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Email:     "admin@example.com",
			Name:      "Admin User",
			Password:  stringPtr("hashedpassword"),
			Role:      "admin",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Email:     "user2@example.com",
			Name:      "User Two",
			Password:  stringPtr("hashedpassword"),
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
			Password:  stringPtr("hashedpassword"),
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

func TestUserRepository_UpdateLastActivity(t *testing.T) {
	db := setupUserTestDB(t)
	defer db.Close()

	repo := NewUserRepository(db)

	t.Run("Update last activity for existing user", func(t *testing.T) {
		// Create a test user
		user := &models.User{
			Email:     "activity@example.com",
			Name:      "Activity User",
			Password:  stringPtr("hashedpassword"),
			Role:      "user",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		if err := repo.Create(user); err != nil {
			t.Fatalf("Failed to create test user: %v", err)
		}

		// Verify last_activity_at is nil initially
		fetchedUser, err := repo.GetByID(user.ID)
		if err != nil {
			t.Fatalf("Failed to get user: %v", err)
		}

		if fetchedUser.LastActivityAt != nil {
			t.Errorf("Expected last_activity_at to be nil initially, got %v", fetchedUser.LastActivityAt)
		}

		// Update last activity
		if err := repo.UpdateLastActivity(user.ID); err != nil {
			t.Fatalf("Failed to update last activity: %v", err)
		}

		// Verify last_activity_at is now set
		updatedUser, err := repo.GetByID(user.ID)
		if err != nil {
			t.Fatalf("Failed to get updated user: %v", err)
		}

		if updatedUser.LastActivityAt == nil {
			t.Error("Expected last_activity_at to be set after update")
		}

		// Verify timestamp is recent (within last 5 seconds)
		if updatedUser.LastActivityAt != nil {
			timeDiff := time.Since(*updatedUser.LastActivityAt)
			if timeDiff > 5*time.Second {
				t.Errorf("Expected last_activity_at to be recent, but it was %v ago", timeDiff)
			}
		}
	})

	t.Run("Update last activity multiple times", func(t *testing.T) {
		// Create a test user
		user := &models.User{
			Email:     "multiactivity@example.com",
			Name:      "Multi Activity User",
			Password:  stringPtr("hashedpassword"),
			Role:      "user",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		if err := repo.Create(user); err != nil {
			t.Fatalf("Failed to create test user: %v", err)
		}

		// First update
		if err := repo.UpdateLastActivity(user.ID); err != nil {
			t.Fatalf("Failed first update: %v", err)
		}

		firstUser, err := repo.GetByID(user.ID)
		if err != nil {
			t.Fatalf("Failed to get user after first update: %v", err)
		}

		if firstUser.LastActivityAt == nil {
			t.Fatal("Expected last_activity_at to be set after first update")
		}

		firstActivityTime := *firstUser.LastActivityAt

		// Wait to ensure timestamp difference (SQLite has 1-second precision)
		time.Sleep(1 * time.Second)

		// Second update
		if err := repo.UpdateLastActivity(user.ID); err != nil {
			t.Fatalf("Failed second update: %v", err)
		}

		secondUser, err := repo.GetByID(user.ID)
		if err != nil {
			t.Fatalf("Failed to get user after second update: %v", err)
		}

		if secondUser.LastActivityAt == nil {
			t.Fatal("Expected last_activity_at to be set after second update")
		}

		secondActivityTime := *secondUser.LastActivityAt

		// Verify second timestamp is after first
		if !secondActivityTime.After(firstActivityTime) {
			t.Errorf("Expected second activity time (%v) to be after first (%v)",
				secondActivityTime, firstActivityTime)
		}
	})

	t.Run("Update last activity for non-existent user", func(t *testing.T) {
		// This shouldn't return an error - it just won't affect any rows
		err := repo.UpdateLastActivity("non-existent-id")
		if err != nil {
			t.Errorf("UpdateLastActivity should not error for non-existent user, got: %v", err)
		}
	})

	t.Run("Last activity persists across GetByEmail", func(t *testing.T) {
		// Create a test user
		user := &models.User{
			Email:     "persist@example.com",
			Name:      "Persist User",
			Password:  stringPtr("hashedpassword"),
			Role:      "user",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		if err := repo.Create(user); err != nil {
			t.Fatalf("Failed to create test user: %v", err)
		}

		// Update last activity
		if err := repo.UpdateLastActivity(user.ID); err != nil {
			t.Fatalf("Failed to update last activity: %v", err)
		}

		// Fetch by email
		fetchedUser, err := repo.GetByEmail(user.Email)
		if err != nil {
			t.Fatalf("Failed to get user by email: %v", err)
		}

		if fetchedUser.LastActivityAt == nil {
			t.Error("Expected last_activity_at to be returned by GetByEmail")
		}
	})
}

func TestUserRepository_GetFiltered(t *testing.T) {
	db := setupUserTestDB(t)
	defer db.Close()

	repo := NewUserRepository(db)

	// Create test users with various attributes
	users := []*models.User{
		{
			Email:     "alice@example.com",
			Name:      "Alice Admin",
			Password:  stringPtr("hashedpassword"),
			Role:      "admin",
			CreatedAt: time.Now().Add(-3 * time.Hour),
			UpdatedAt: time.Now().Add(-3 * time.Hour),
		},
		{
			Email:     "bob@example.com",
			Name:      "Bob User",
			Password:  stringPtr("hashedpassword"),
			Role:      "user",
			CreatedAt: time.Now().Add(-2 * time.Hour),
			UpdatedAt: time.Now().Add(-2 * time.Hour),
		},
		{
			Email:     "charlie@example.com",
			Name:      "Charlie Developer",
			Password:  stringPtr("hashedpassword"),
			Role:      "user",
			CreatedAt: time.Now().Add(-1 * time.Hour),
			UpdatedAt: time.Now().Add(-1 * time.Hour),
		},
		{
			Email:     "diana@test.com",
			Name:      "Diana Designer",
			Password:  stringPtr("hashedpassword"),
			Role:      "user",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	for _, user := range users {
		if err := repo.Create(user); err != nil {
			t.Fatalf("Failed to create test user %s: %v", user.Email, err)
		}
	}

	t.Run("Get all users without filter", func(t *testing.T) {
		filter := UserFilter{
			Limit: 10,
		}

		result, err := repo.GetFiltered(filter)
		if err != nil {
			t.Fatalf("Failed to get filtered users: %v", err)
		}

		if result.Total != 4 {
			t.Errorf("Expected total 4, got %d", result.Total)
		}

		if len(result.Users) != 4 {
			t.Errorf("Expected 4 users, got %d", len(result.Users))
		}

		if result.Page != 1 {
			t.Errorf("Expected page 1, got %d", result.Page)
		}

		if result.TotalPages != 1 {
			t.Errorf("Expected 1 total page, got %d", result.TotalPages)
		}
	})

	t.Run("Search by name", func(t *testing.T) {
		filter := UserFilter{
			Search: "alice",
			Limit:  10,
		}

		result, err := repo.GetFiltered(filter)
		if err != nil {
			t.Fatalf("Failed to search users: %v", err)
		}

		if result.Total != 1 {
			t.Errorf("Expected 1 result for 'alice', got %d", result.Total)
		}

		if len(result.Users) > 0 && result.Users[0].Email != "alice@example.com" {
			t.Errorf("Expected alice@example.com, got %s", result.Users[0].Email)
		}
	})

	t.Run("Search by email", func(t *testing.T) {
		filter := UserFilter{
			Search: "test.com",
			Limit:  10,
		}

		result, err := repo.GetFiltered(filter)
		if err != nil {
			t.Fatalf("Failed to search users: %v", err)
		}

		if result.Total != 1 {
			t.Errorf("Expected 1 result for 'test.com', got %d", result.Total)
		}

		if len(result.Users) > 0 && result.Users[0].Email != "diana@test.com" {
			t.Errorf("Expected diana@test.com, got %s", result.Users[0].Email)
		}
	})

	t.Run("Search case insensitive", func(t *testing.T) {
		filter := UserFilter{
			Search: "CHARLIE",
			Limit:  10,
		}

		result, err := repo.GetFiltered(filter)
		if err != nil {
			t.Fatalf("Failed to search users: %v", err)
		}

		if result.Total != 1 {
			t.Errorf("Expected 1 result for 'CHARLIE', got %d", result.Total)
		}
	})

	t.Run("Filter by role - admin only", func(t *testing.T) {
		filter := UserFilter{
			Role:  "admin",
			Limit: 10,
		}

		result, err := repo.GetFiltered(filter)
		if err != nil {
			t.Fatalf("Failed to filter by admin role: %v", err)
		}

		if result.Total != 1 {
			t.Errorf("Expected 1 admin, got %d", result.Total)
		}

		if len(result.Users) > 0 && result.Users[0].Role != "admin" {
			t.Errorf("Expected admin role, got %s", result.Users[0].Role)
		}
	})

	t.Run("Filter by role - users only", func(t *testing.T) {
		filter := UserFilter{
			Role:  "user",
			Limit: 10,
		}

		result, err := repo.GetFiltered(filter)
		if err != nil {
			t.Fatalf("Failed to filter by user role: %v", err)
		}

		if result.Total != 3 {
			t.Errorf("Expected 3 users, got %d", result.Total)
		}

		for _, user := range result.Users {
			if user.Role != "user" {
				t.Errorf("Expected user role, got %s", user.Role)
			}
		}
	})

	t.Run("Combine search and role filter", func(t *testing.T) {
		filter := UserFilter{
			Search: "example.com",
			Role:   "user",
			Limit:  10,
		}

		result, err := repo.GetFiltered(filter)
		if err != nil {
			t.Fatalf("Failed with combined filters: %v", err)
		}

		// Should match bob and charlie (both user role with example.com email)
		if result.Total != 2 {
			t.Errorf("Expected 2 results, got %d", result.Total)
		}

		for _, user := range result.Users {
			if user.Role != "user" {
				t.Errorf("Expected user role, got %s", user.Role)
			}
		}
	})

	t.Run("Pagination - first page", func(t *testing.T) {
		filter := UserFilter{
			Limit:  2,
			Offset: 0,
		}

		result, err := repo.GetFiltered(filter)
		if err != nil {
			t.Fatalf("Failed to get first page: %v", err)
		}

		if result.Total != 4 {
			t.Errorf("Expected total 4, got %d", result.Total)
		}

		if len(result.Users) != 2 {
			t.Errorf("Expected 2 users on first page, got %d", len(result.Users))
		}

		if result.Page != 1 {
			t.Errorf("Expected page 1, got %d", result.Page)
		}

		if result.TotalPages != 2 {
			t.Errorf("Expected 2 total pages, got %d", result.TotalPages)
		}
	})

	t.Run("Pagination - second page", func(t *testing.T) {
		filter := UserFilter{
			Limit:  2,
			Offset: 2,
		}

		result, err := repo.GetFiltered(filter)
		if err != nil {
			t.Fatalf("Failed to get second page: %v", err)
		}

		if result.Total != 4 {
			t.Errorf("Expected total 4, got %d", result.Total)
		}

		if len(result.Users) != 2 {
			t.Errorf("Expected 2 users on second page, got %d", len(result.Users))
		}

		if result.Page != 2 {
			t.Errorf("Expected page 2, got %d", result.Page)
		}
	})

	t.Run("Pagination - partial last page", func(t *testing.T) {
		filter := UserFilter{
			Limit:  3,
			Offset: 3,
		}

		result, err := repo.GetFiltered(filter)
		if err != nil {
			t.Fatalf("Failed to get last page: %v", err)
		}

		if len(result.Users) != 1 {
			t.Errorf("Expected 1 user on last page, got %d", len(result.Users))
		}
	})

	t.Run("Default limit of 20", func(t *testing.T) {
		filter := UserFilter{
			// No limit specified
		}

		result, err := repo.GetFiltered(filter)
		if err != nil {
			t.Fatalf("Failed with default limit: %v", err)
		}

		// Should use default limit of 20
		if len(result.Users) != 4 {
			t.Errorf("Expected 4 users (all available), got %d", len(result.Users))
		}
	})

	t.Run("Empty results", func(t *testing.T) {
		filter := UserFilter{
			Search: "nonexistent",
			Limit:  10,
		}

		result, err := repo.GetFiltered(filter)
		if err != nil {
			t.Fatalf("Failed with empty results: %v", err)
		}

		if result.Total != 0 {
			t.Errorf("Expected total 0, got %d", result.Total)
		}

		if len(result.Users) != 0 {
			t.Errorf("Expected 0 users, got %d", len(result.Users))
		}
	})
}

func TestUserRepository_LastActivityIntegration(t *testing.T) {
	db := setupUserTestDB(t)
	defer db.Close()

	repo := NewUserRepository(db)

	t.Run("Full workflow with last activity", func(t *testing.T) {
		// 1. Create user
		user := &models.User{
			Email:     "workflow@example.com",
			Name:      "Workflow User",
			Password:  stringPtr("hashedpassword"),
			Role:      "user",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		if err := repo.Create(user); err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}

		// 2. Update last activity (simulating login)
		if err := repo.UpdateLastActivity(user.ID); err != nil {
			t.Fatalf("Failed to update last activity: %v", err)
		}

		// 3. Verify GetAll includes last activity
		allUsers, err := repo.GetAll()
		if err != nil {
			t.Fatalf("Failed to get all users: %v", err)
		}

		if len(allUsers) != 1 {
			t.Fatalf("Expected 1 user, got %d", len(allUsers))
		}

		if allUsers[0].LastActivityAt == nil {
			t.Error("Expected last_activity_at in GetAll result")
		}

		// 4. Verify GetFiltered includes last activity
		filter := UserFilter{
			Limit: 10,
		}

		result, err := repo.GetFiltered(filter)
		if err != nil {
			t.Fatalf("Failed to get filtered users: %v", err)
		}

		if len(result.Users) != 1 {
			t.Fatalf("Expected 1 user, got %d", len(result.Users))
		}

		if result.Users[0].LastActivityAt == nil {
			t.Error("Expected last_activity_at in GetFiltered result")
		}

		// 5. Update activity again
		time.Sleep(100 * time.Millisecond)
		if err := repo.UpdateLastActivity(user.ID); err != nil {
			t.Fatalf("Failed second activity update: %v", err)
		}

		// 6. Verify ToResponse includes last activity
		fetchedUser, err := repo.GetByID(user.ID)
		if err != nil {
			t.Fatalf("Failed to get user: %v", err)
		}

		response := fetchedUser.ToResponse()
		if response.LastActivityAt == nil {
			t.Error("Expected last_activity_at in user response")
		}
	})
}

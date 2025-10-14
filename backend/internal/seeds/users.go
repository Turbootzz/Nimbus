//go:build dev

package seeds

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/repository"
	"github.com/nimbus/backend/internal/services"
)

// SeedUsers creates test users for development and testing
func SeedUsers(userRepo *repository.UserRepository, authService *services.AuthService, database *sql.DB) error {
	log.Println("\n📦 Seeding test users...")

	// Password for all test users
	password := "password123"
	hashedPassword, err := authService.HashPassword(password)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	now := time.Now()

	// Define test users
	testUsers := []struct {
		email        string
		name         string
		role         string
		createdDays  int // Days ago the account was created
		activityTime *time.Time
	}{
		// Admins (5 users) with various last activity times
		{"alice.admin@test.example.com", "Alice Administrator", "admin", 90, timePtr(now.Add(-5 * time.Second))},
		{"bob.admin@test.example.com", "Bob Manager", "admin", 75, timePtr(now.Add(-45 * time.Second))},
		{"carol.admin@test.example.com", "Carol Supervisor", "admin", 60, timePtr(now.Add(-10 * time.Minute))},
		{"david.admin@test.example.com", "David Director", "admin", 45, timePtr(now.Add(-2 * time.Hour))},
		{"eve.admin@test.example.com", "Eve Executive", "admin", 30, timePtr(now.Add(-6 * time.Hour))},

		// Regular users with various last activity times (25 users)
		{"frank.user@test.example.com", "Frank Developer", "user", 80, timePtr(now.Add(-24 * time.Hour))},
		{"grace.user@test.example.com", "Grace Designer", "user", 70, timePtr(now.Add(-48 * time.Hour))},
		{"henry.user@test.example.com", "Henry Engineer", "user", 65, timePtr(now.Add(-72 * time.Hour))},
		{"iris.user@test.example.com", "Iris Analyst", "user", 55, timePtr(now.Add(-5 * 24 * time.Hour))},
		{"jack.user@test.example.com", "Jack Consultant", "user", 50, timePtr(now.Add(-7 * 24 * time.Hour))},

		{"karen.user@test.example.com", "Karen Specialist", "user", 45, timePtr(now.Add(-14 * 24 * time.Hour))},
		{"leo.user@test.example.com", "Leo Coordinator", "user", 40, timePtr(now.Add(-21 * 24 * time.Hour))},
		{"maria.user@test.example.com", "Maria Tester", "user", 35, timePtr(now.Add(-35 * 24 * time.Hour))},
		{"nathan.user@test.example.com", "Nathan Developer", "user", 30, timePtr(now.Add(-56 * 24 * time.Hour))},
		{"olivia.user@test.example.com", "Olivia Smith", "user", 28, timePtr(now.Add(-84 * 24 * time.Hour))},

		{"peter.user@test.example.com", "Peter Johnson", "user", 25, timePtr(now.Add(-15 * time.Minute))},
		{"quinn.user@test.example.com", "Quinn Williams", "user", 22, timePtr(now.Add(-30 * time.Minute))},
		{"rachel.user@test.example.com", "Rachel Brown", "user", 20, timePtr(now.Add(-1 * time.Hour))},
		{"sam.user@test.example.com", "Sam Davis", "user", 18, timePtr(now.Add(-3 * time.Hour))},
		{"tina.user@test.example.com", "Tina Miller", "user", 15, timePtr(now.Add(-8 * time.Hour))},

		{"uma.user@test.example.com", "Uma Wilson", "user", 12, timePtr(now.Add(-12 * time.Hour))},
		{"victor.user@test.example.com", "Victor Moore", "user", 10, timePtr(now.Add(-18 * time.Hour))},
		{"wendy.user@test.example.com", "Wendy Taylor", "user", 8, timePtr(now.Add(-23 * time.Hour))},
		{"xavier.user@test.example.com", "Xavier Anderson", "user", 6, timePtr(now.Add(-4 * 24 * time.Hour))},
		{"yara.user@test.example.com", "Yara Thomas", "user", 5, timePtr(now.Add(-6 * 24 * time.Hour))},

		// Users who never logged in (NULL last_activity_at)
		{"zack.user@test.example.com", "Zack Jackson", "user", 4, nil},
		{"amy.user@test.example.com", "Amy White", "user", 3, nil},
		{"brian.user@test.example.com", "Brian Harris", "user", 2, nil},
		{"claire.user@test.example.com", "Claire Martin", "user", 1, nil},
		{"derek.user@test.example.com", "Derek Thompson", "user", 0, nil},
	}

	// Create or update users
	created := 0
	updated := 0
	skipped := 0

	for _, testUser := range testUsers {
		// Check if user exists
		existingUser, err := userRepo.GetByEmail(testUser.email)

		if err != nil && err.Error() == "user not found" {
			// User doesn't exist, create new
			createdAt := now.Add(-time.Duration(testUser.createdDays) * 24 * time.Hour)
			user := &models.User{
				Email:          testUser.email,
				Name:           testUser.name,
				Password:       hashedPassword,
				Role:           testUser.role,
				LastActivityAt: testUser.activityTime,
				CreatedAt:      createdAt,
				UpdatedAt:      createdAt,
			}

			if err := userRepo.Create(user); err != nil {
				log.Printf("  ⚠️  Failed to create user %s: %v", testUser.email, err)
				continue
			}

			// Manually update last_activity_at since Create doesn't set it
			if testUser.activityTime != nil {
				if err := updateLastActivity(database, user.ID, *testUser.activityTime); err != nil {
					log.Printf("  ⚠️  Failed to update last activity for %s: %v", testUser.email, err)
				}
			}

			log.Printf("  ✓ Created: %s (%s)", testUser.name, testUser.role)
			created++
		} else if err != nil {
			// Unexpected error
			log.Printf("  ⚠️  Error checking user %s: %v", testUser.email, err)
			skipped++
		} else {
			// User exists, update last_activity_at for testing
			if testUser.activityTime != nil {
				if err := updateLastActivity(database, existingUser.ID, *testUser.activityTime); err != nil {
					log.Printf("  ⚠️  Failed to update last activity for %s: %v", testUser.email, err)
				}
			}
			log.Printf("  → Exists: %s (updated last activity)", testUser.name)
			updated++
		}
	}

	// Print summary
	log.Printf("\n📊 Summary:")
	log.Printf("  ✓ Created: %d users", created)
	log.Printf("  → Updated: %d users", updated)
	log.Printf("  ⊘ Skipped: %d users", skipped)
	log.Printf("  = Total:   %d users", len(testUsers))

	// Print statistics
	stats, err := userRepo.GetStats()
	if err == nil {
		log.Printf("\n📈 Database Statistics:")
		log.Printf("  Total Users: %d", stats["total"])
		log.Printf("  Admins:      %d", stats["admins"])
		log.Printf("  Users:       %d", stats["users"])
	}

	// Print test credentials
	log.Printf("\n🔑 Test Credentials:")
	log.Printf("  Email:    Any test user (e.g., alice.admin@test.example.com)")
	log.Printf("  Password: %s", password)

	return nil
}

// Helper function to create time pointer
func timePtr(t time.Time) *time.Time {
	return &t
}

// updateLastActivity manually updates the last_activity_at field
func updateLastActivity(database *sql.DB, userID string, activityTime time.Time) error {
	query := `UPDATE users SET last_activity_at = $1 WHERE id = $2`
	_, err := database.Exec(query, activityTime, userID)
	return err
}

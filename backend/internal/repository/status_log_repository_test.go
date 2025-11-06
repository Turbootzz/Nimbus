package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/nimbus/backend/internal/models"
)

// setupStatusLogTestDB creates an in-memory SQLite database for testing
func setupStatusLogTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Create services table
	_, err = db.Exec(`
		CREATE TABLE services (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			name TEXT NOT NULL,
			url TEXT NOT NULL,
			icon TEXT DEFAULT '🔗',
			description TEXT,
			status TEXT DEFAULT 'unknown',
			response_time INTEGER,
			position INTEGER,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create services table: %v", err)
	}

	// Create service_status_logs table (use INTEGER for autoincrement in SQLite)
	_, err = db.Exec(`
		CREATE TABLE service_status_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			service_id TEXT NOT NULL,
			status TEXT NOT NULL CHECK(status IN ('online', 'offline', 'unknown')),
			response_time INTEGER,
			error_message TEXT,
			checked_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(service_id) REFERENCES services(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create service_status_logs table: %v", err)
	}

	// Insert test service
	_, err = db.Exec(`
		INSERT INTO services (id, user_id, name, url, status)
		VALUES ('test-service-1', 'test-user-1', 'Test Service', 'http://example.com', 'online')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test service: %v", err)
	}

	return db
}

func TestStatusLogRepository_Create(t *testing.T) {
	db := setupStatusLogTestDB(t)
	defer db.Close()

	repo := NewStatusLogRepository(db)
	ctx := context.Background()

	responseTime := 150
	log := &models.StatusLog{
		ServiceID:    "test-service-1",
		Status:       models.StatusOnline,
		ResponseTime: &responseTime,
		CheckedAt:    time.Now(),
	}

	err := repo.Create(ctx, log)
	if err != nil {
		t.Fatalf("Failed to create status log: %v", err)
	}

	if log.ID == "" {
		t.Error("Expected ID to be set after creation")
	}
}

func TestStatusLogRepository_GetLatestByServiceID(t *testing.T) {
	db := setupStatusLogTestDB(t)
	defer db.Close()

	repo := NewStatusLogRepository(db)
	ctx := context.Background()

	// Insert test logs
	for i := 0; i < 5; i++ {
		responseTime := 100 + i*10
		log := &models.StatusLog{
			ServiceID:    "test-service-1",
			Status:       models.StatusOnline,
			ResponseTime: &responseTime,
			CheckedAt:    time.Now().Add(time.Duration(i) * time.Minute),
		}
		if err := repo.Create(ctx, log); err != nil {
			t.Fatalf("Failed to create test log: %v", err)
		}
	}

	// Get latest 3 logs
	logs, err := repo.GetLatestByServiceID(ctx, "test-service-1", 3)
	if err != nil {
		t.Fatalf("Failed to get latest logs: %v", err)
	}

	if len(logs) != 3 {
		t.Errorf("Expected 3 logs, got %d", len(logs))
	}

	// Verify they're in descending order by checked_at
	for i := 0; i < len(logs)-1; i++ {
		if logs[i].CheckedAt.Before(logs[i+1].CheckedAt) {
			t.Error("Expected logs to be in descending order by checked_at")
		}
	}
}

func TestStatusLogRepository_GetUptimeStats(t *testing.T) {
	db := setupStatusLogTestDB(t)
	defer db.Close()

	repo := NewStatusLogRepository(db)
	ctx := context.Background()

	now := time.Now()
	startTime := now.Add(-1 * time.Hour)

	// Insert test logs: 7 online, 3 offline
	for i := 0; i < 10; i++ {
		status := models.StatusOnline
		responseTime := 100 + i*10

		if i < 3 {
			status = models.StatusOffline
			responseTime = 0
		}

		log := &models.StatusLog{
			ServiceID:    "test-service-1",
			Status:       status,
			ResponseTime: &responseTime,
			CheckedAt:    startTime.Add(time.Duration(i) * time.Minute),
		}
		if err := repo.Create(ctx, log); err != nil {
			t.Fatalf("Failed to create test log: %v", err)
		}
	}

	// Get uptime stats
	stats, err := repo.GetUptimeStats(ctx, "test-service-1", startTime, now)
	if err != nil {
		t.Fatalf("Failed to get uptime stats: %v", err)
	}

	totalChecks := stats["total_checks"].(int)
	onlineCount := stats["online_count"].(int)
	offlineCount := stats["offline_count"].(int)
	uptimePercentage := stats["uptime_percentage"].(float64)

	if totalChecks != 10 {
		t.Errorf("Expected 10 total checks, got %d", totalChecks)
	}

	if onlineCount != 7 {
		t.Errorf("Expected 7 online count, got %d", onlineCount)
	}

	if offlineCount != 3 {
		t.Errorf("Expected 3 offline count, got %d", offlineCount)
	}

	expectedUptime := 70.0
	if uptimePercentage != expectedUptime {
		t.Errorf("Expected uptime percentage %.2f, got %.2f", expectedUptime, uptimePercentage)
	}
}

func TestStatusLogRepository_DeleteOlderThan(t *testing.T) {
	db := setupStatusLogTestDB(t)
	defer db.Close()

	repo := NewStatusLogRepository(db)
	ctx := context.Background()

	now := time.Now()

	// Insert logs: 5 old (> 7 days), 5 recent (< 7 days)
	for i := 0; i < 10; i++ {
		responseTime := 100
		checkedAt := now.Add(time.Duration(-i) * 24 * time.Hour)

		log := &models.StatusLog{
			ServiceID:    "test-service-1",
			Status:       models.StatusOnline,
			ResponseTime: &responseTime,
			CheckedAt:    checkedAt,
		}
		if err := repo.Create(ctx, log); err != nil {
			t.Fatalf("Failed to create test log: %v", err)
		}
	}

	// Delete logs older than 7 days
	cutoffTime := now.Add(-7 * 24 * time.Hour)
	deletedCount, err := repo.DeleteOlderThan(ctx, cutoffTime)
	if err != nil {
		t.Fatalf("Failed to delete old logs: %v", err)
	}

	if deletedCount < 2 {
		t.Errorf("Expected at least 2 deleted logs, got %d", deletedCount)
	}

	// Verify remaining logs
	count, err := repo.CountByServiceID(ctx, "test-service-1")
	if err != nil {
		t.Fatalf("Failed to count logs: %v", err)
	}

	expectedRemaining := 10 - deletedCount
	if count != expectedRemaining {
		t.Errorf("Expected %d remaining logs, got %d", expectedRemaining, count)
	}
}

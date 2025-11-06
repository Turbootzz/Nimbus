package workers

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/repository"
	"github.com/nimbus/backend/internal/services"
)

// setupCleanupTestDB creates an in-memory SQLite database for testing
func setupCleanupTestDB(t *testing.T) *sql.DB {
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

	// Create service_status_logs table (matching production schema with UUID PRIMARY KEY)
	_, err = db.Exec(`
		CREATE TABLE service_status_logs (
			id TEXT PRIMARY KEY,
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
		VALUES ('test-service-1', 'user-1', 'Test Service', 'http://example.com', 'online')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test service: %v", err)
	}

	return db
}

func TestNewMetricsCleanupWorker(t *testing.T) {
	db := setupCleanupTestDB(t)
	defer db.Close()

	statusLogRepo := repository.NewStatusLogRepository(db)
	serviceRepo := repository.NewServiceRepository(db)
	metricsService := services.NewMetricsService(statusLogRepo, serviceRepo)

	worker := NewMetricsCleanupWorker(metricsService)

	if worker == nil {
		t.Fatal("Expected worker to be created")
	}

	if worker.retentionDays != 30 {
		t.Errorf("Expected default retention days 30, got %d", worker.retentionDays)
	}

	expectedInterval := 24 * time.Hour
	if worker.cleanupInterval != expectedInterval {
		t.Errorf("Expected cleanup interval %v, got %v", expectedInterval, worker.cleanupInterval)
	}
}

func TestNewMetricsCleanupWorker_CustomRetention(t *testing.T) {
	db := setupCleanupTestDB(t)
	defer db.Close()

	// Set custom retention days via environment variable
	t.Setenv("METRICS_RETENTION_DAYS", "7")

	statusLogRepo := repository.NewStatusLogRepository(db)
	serviceRepo := repository.NewServiceRepository(db)
	metricsService := services.NewMetricsService(statusLogRepo, serviceRepo)

	worker := NewMetricsCleanupWorker(metricsService)

	if worker.retentionDays != 7 {
		t.Errorf("Expected retention days 7, got %d", worker.retentionDays)
	}
}

func TestMetricsCleanupWorker_RunNow(t *testing.T) {
	db := setupCleanupTestDB(t)
	defer db.Close()

	statusLogRepo := repository.NewStatusLogRepository(db)
	serviceRepo := repository.NewServiceRepository(db)
	metricsService := services.NewMetricsService(statusLogRepo, serviceRepo)

	ctx := context.Background()
	now := time.Now()

	// Insert test logs: mix of old and recent
	for i := 0; i < 20; i++ {
		responseTime := 100
		// Make half of them older than 30 days
		daysAgo := i
		if i >= 10 {
			daysAgo = i + 25 // Make these 35+ days old
		}

		log := &models.StatusLog{
			ID:           uuid.New().String(),
			ServiceID:    "test-service-1",
			Status:       models.StatusOnline,
			ResponseTime: &responseTime,
			CheckedAt:    now.Add(time.Duration(-daysAgo) * 24 * time.Hour),
		}
		if err := statusLogRepo.Create(ctx, log); err != nil {
			t.Fatalf("Failed to create test log: %v", err)
		}
	}

	// Verify initial count
	initialCount, err := statusLogRepo.CountByServiceID(ctx, "test-service-1")
	if err != nil {
		t.Fatalf("Failed to count logs: %v", err)
	}

	if initialCount != 20 {
		t.Errorf("Expected 20 initial logs, got %d", initialCount)
	}

	// Create worker and run cleanup
	worker := NewMetricsCleanupWorker(metricsService)
	err = worker.RunNow()
	if err != nil {
		t.Fatalf("Failed to run cleanup: %v", err)
	}

	// Verify logs were deleted
	remainingCount, err := statusLogRepo.CountByServiceID(ctx, "test-service-1")
	if err != nil {
		t.Fatalf("Failed to count logs after cleanup: %v", err)
	}

	// Should have deleted the 10 logs older than 30 days
	if remainingCount >= initialCount {
		t.Errorf("Expected fewer than %d logs after cleanup, got %d", initialCount, remainingCount)
	}

	if remainingCount > 10 {
		t.Errorf("Expected around 10 or fewer remaining logs, got %d", remainingCount)
	}
}

func TestMetricsCleanupWorker_RunNow_NoOldLogs(t *testing.T) {
	db := setupCleanupTestDB(t)
	defer db.Close()

	statusLogRepo := repository.NewStatusLogRepository(db)
	serviceRepo := repository.NewServiceRepository(db)
	metricsService := services.NewMetricsService(statusLogRepo, serviceRepo)

	ctx := context.Background()
	now := time.Now()

	// Insert only recent logs (last 7 days)
	for i := 0; i < 10; i++ {
		responseTime := 100
		log := &models.StatusLog{
			ID:           uuid.New().String(),
			ServiceID:    "test-service-1",
			Status:       models.StatusOnline,
			ResponseTime: &responseTime,
			CheckedAt:    now.Add(time.Duration(-i) * time.Hour),
		}
		if err := statusLogRepo.Create(ctx, log); err != nil {
			t.Fatalf("Failed to create test log: %v", err)
		}
	}

	// Create worker and run cleanup
	worker := NewMetricsCleanupWorker(metricsService)
	err := worker.RunNow()
	if err != nil {
		t.Fatalf("Failed to run cleanup: %v", err)
	}

	// Verify no logs were deleted
	remainingCount, err := statusLogRepo.CountByServiceID(ctx, "test-service-1")
	if err != nil {
		t.Fatalf("Failed to count logs after cleanup: %v", err)
	}

	if remainingCount != 10 {
		t.Errorf("Expected 10 remaining logs, got %d", remainingCount)
	}
}

func TestMetricsCleanupWorker_StartStop(t *testing.T) {
	db := setupCleanupTestDB(t)
	defer db.Close()

	statusLogRepo := repository.NewStatusLogRepository(db)
	serviceRepo := repository.NewServiceRepository(db)
	metricsService := services.NewMetricsService(statusLogRepo, serviceRepo)

	worker := NewMetricsCleanupWorker(metricsService)

	// Start the worker
	worker.Start()

	// Give it a moment to start
	time.Sleep(100 * time.Millisecond)

	// Stop the worker
	worker.Stop()

	// Give it a moment to stop gracefully
	time.Sleep(100 * time.Millisecond)

	// If we reached here without hanging, the test passes
}

func TestMetricsCleanupWorker_RunCleanup_Integration(t *testing.T) {
	db := setupCleanupTestDB(t)
	defer db.Close()

	statusLogRepo := repository.NewStatusLogRepository(db)
	serviceRepo := repository.NewServiceRepository(db)
	metricsService := services.NewMetricsService(statusLogRepo, serviceRepo)

	ctx := context.Background()
	now := time.Now()

	// Insert logs with varying ages
	testCases := []struct {
		daysAgo int
		count   int
	}{
		{daysAgo: 1, count: 5},   // Recent logs
		{daysAgo: 15, count: 5},  // Mid-age logs
		{daysAgo: 40, count: 10}, // Old logs (should be deleted)
	}

	totalLogs := 0
	for _, tc := range testCases {
		for i := 0; i < tc.count; i++ {
			responseTime := 100
			log := &models.StatusLog{
				ID:           uuid.New().String(),
				ServiceID:    "test-service-1",
				Status:       models.StatusOnline,
				ResponseTime: &responseTime,
				CheckedAt:    now.Add(time.Duration(-tc.daysAgo) * 24 * time.Hour),
			}
			if err := statusLogRepo.Create(ctx, log); err != nil {
				t.Fatalf("Failed to create test log: %v", err)
			}
			totalLogs++
		}
	}

	// Verify initial count
	initialCount, err := statusLogRepo.CountByServiceID(ctx, "test-service-1")
	if err != nil {
		t.Fatalf("Failed to count logs: %v", err)
	}

	if initialCount != int64(totalLogs) {
		t.Errorf("Expected %d initial logs, got %d", totalLogs, initialCount)
	}

	// Run cleanup
	worker := NewMetricsCleanupWorker(metricsService)
	worker.runCleanup()

	// Verify cleanup results
	remainingCount, err := statusLogRepo.CountByServiceID(ctx, "test-service-1")
	if err != nil {
		t.Fatalf("Failed to count logs after cleanup: %v", err)
	}

	// Should have deleted the 10 logs that are 40 days old
	expectedRemaining := int64(10) // 5 + 5 from recent and mid-age logs
	if remainingCount != expectedRemaining {
		t.Errorf("Expected %d remaining logs, got %d", expectedRemaining, remainingCount)
	}
}

func TestMetricsCleanupWorker_InvalidRetentionDays(t *testing.T) {
	db := setupCleanupTestDB(t)
	defer db.Close()

	// Set invalid retention days via environment variable
	t.Setenv("METRICS_RETENTION_DAYS", "invalid")

	statusLogRepo := repository.NewStatusLogRepository(db)
	serviceRepo := repository.NewServiceRepository(db)
	metricsService := services.NewMetricsService(statusLogRepo, serviceRepo)

	worker := NewMetricsCleanupWorker(metricsService)

	// Should fallback to default value (30)
	if worker.retentionDays != 30 {
		t.Errorf("Expected default retention days 30 for invalid env value, got %d", worker.retentionDays)
	}
}

func TestMetricsCleanupWorker_ZeroRetentionDays(t *testing.T) {
	db := setupCleanupTestDB(t)
	defer db.Close()

	// Set zero retention days via environment variable
	t.Setenv("METRICS_RETENTION_DAYS", "0")

	statusLogRepo := repository.NewStatusLogRepository(db)
	serviceRepo := repository.NewServiceRepository(db)
	metricsService := services.NewMetricsService(statusLogRepo, serviceRepo)

	worker := NewMetricsCleanupWorker(metricsService)

	// Should fallback to default value (30) when value is 0 or negative
	if worker.retentionDays != 30 {
		t.Errorf("Expected default retention days 30 for zero value, got %d", worker.retentionDays)
	}
}

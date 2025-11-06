package services

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/repository"
)

// setupMetricsTestDB creates an in-memory SQLite database for testing
func setupMetricsTestDB(t *testing.T) *sql.DB {
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

	// Create service_status_logs table
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

	// Insert test services with all required fields
	_, err = db.Exec(`
		INSERT INTO services (id, user_id, name, url, description, status, position)
		VALUES
			('test-service-1', 'user-1', 'Test Service 1', 'http://example.com', 'Test description', 'online', 0),
			('test-service-2', 'user-1', 'Test Service 2', 'http://example2.com', 'Test description 2', 'offline', 1)
	`)
	if err != nil {
		t.Fatalf("Failed to insert test services: %v", err)
	}

	return db
}

func TestMetricsService_GetServiceMetrics(t *testing.T) {
	// NOTE: This test is skipped because GetServiceMetrics relies on GetAggregatedByInterval
	// which uses PostgreSQL-specific SQL functions (date_trunc, EXTRACT, interval).
	// In production, this works fine with PostgreSQL. For SQLite testing, we test
	// the individual components separately (GetUptimeStats, GetRecentStatusLogs, etc.)
	t.Skip("Skipping due to PostgreSQL-specific SQL in GetAggregatedByInterval - tested in integration tests with real PostgreSQL")
}

func TestMetricsService_GetRecentStatusLogs(t *testing.T) {
	db := setupMetricsTestDB(t)
	defer db.Close()

	statusLogRepo := repository.NewStatusLogRepository(db)
	serviceRepo := repository.NewServiceRepository(db)
	metricsService := NewMetricsService(statusLogRepo, serviceRepo)

	ctx := context.Background()

	// Insert test logs
	for i := 0; i < 10; i++ {
		responseTime := 100 + i*10
		log := &models.StatusLog{
			ServiceID:    "test-service-1",
			Status:       models.StatusOnline,
			ResponseTime: &responseTime,
			CheckedAt:    time.Now().Add(time.Duration(i) * time.Minute),
		}
		if err := statusLogRepo.Create(ctx, log); err != nil {
			t.Fatalf("Failed to create test log: %v", err)
		}
	}

	// Get recent logs
	logs, err := metricsService.GetRecentStatusLogs(ctx, "test-service-1", 5)
	if err != nil {
		t.Fatalf("Failed to get recent status logs: %v", err)
	}

	if len(logs) != 5 {
		t.Errorf("Expected 5 logs, got %d", len(logs))
	}

	// Verify logs are in descending order
	for i := 0; i < len(logs)-1; i++ {
		if logs[i].CheckedAt.Before(logs[i+1].CheckedAt) {
			t.Error("Expected logs to be in descending order by checked_at")
		}
	}
}

func TestMetricsService_GetLast24HoursUptime(t *testing.T) {
	db := setupMetricsTestDB(t)
	defer db.Close()

	statusLogRepo := repository.NewStatusLogRepository(db)
	serviceRepo := repository.NewServiceRepository(db)
	metricsService := NewMetricsService(statusLogRepo, serviceRepo)

	ctx := context.Background()
	now := time.Now()

	// Insert test logs: 8 online, 2 offline in last 24 hours
	for i := 0; i < 10; i++ {
		status := models.StatusOnline
		if i < 2 {
			status = models.StatusOffline
		}

		responseTime := 100
		log := &models.StatusLog{
			ServiceID:    "test-service-1",
			Status:       status,
			ResponseTime: &responseTime,
			CheckedAt:    now.Add(time.Duration(-i) * time.Hour),
		}
		if err := statusLogRepo.Create(ctx, log); err != nil {
			t.Fatalf("Failed to create test log: %v", err)
		}
	}

	// Get last 24 hours uptime
	uptime, err := metricsService.GetLast24HoursUptime(ctx, "test-service-1")
	if err != nil {
		t.Fatalf("Failed to get last 24 hours uptime: %v", err)
	}

	expectedUptime := 80.0
	if uptime != expectedUptime {
		t.Errorf("Expected uptime %.2f%%, got %.2f%%", expectedUptime, uptime)
	}
}

func TestMetricsService_CleanupOldLogs(t *testing.T) {
	db := setupMetricsTestDB(t)
	defer db.Close()

	statusLogRepo := repository.NewStatusLogRepository(db)
	serviceRepo := repository.NewServiceRepository(db)
	metricsService := NewMetricsService(statusLogRepo, serviceRepo)

	ctx := context.Background()
	now := time.Now()

	// Insert logs: some old, some recent
	for i := 0; i < 20; i++ {
		responseTime := 100
		checkedAt := now.Add(time.Duration(-i) * 24 * time.Hour)

		log := &models.StatusLog{
			ServiceID:    "test-service-1",
			Status:       models.StatusOnline,
			ResponseTime: &responseTime,
			CheckedAt:    checkedAt,
		}
		if err := statusLogRepo.Create(ctx, log); err != nil {
			t.Fatalf("Failed to create test log: %v", err)
		}
	}

	// Cleanup logs older than 7 days
	deletedCount, err := metricsService.CleanupOldLogs(ctx, 7)
	if err != nil {
		t.Fatalf("Failed to cleanup old logs: %v", err)
	}

	if deletedCount < 10 {
		t.Errorf("Expected at least 10 deleted logs, got %d", deletedCount)
	}

	// Verify remaining logs
	count, err := statusLogRepo.CountByServiceID(ctx, "test-service-1")
	if err != nil {
		t.Fatalf("Failed to count logs: %v", err)
	}

	expectedRemaining := 20 - deletedCount
	if count != expectedRemaining {
		t.Errorf("Expected %d remaining logs, got %d", expectedRemaining, count)
	}
}

func TestMetricsService_GetPrometheusMetrics(t *testing.T) {
	db := setupMetricsTestDB(t)
	defer db.Close()

	statusLogRepo := repository.NewStatusLogRepository(db)
	serviceRepo := repository.NewServiceRepository(db)
	metricsService := NewMetricsService(statusLogRepo, serviceRepo)

	ctx := context.Background()

	// Update service with response time
	responseTime := 150
	_, err := db.Exec(`UPDATE services SET response_time = ? WHERE id = ?`, responseTime, "test-service-1")
	if err != nil {
		t.Fatalf("Failed to update service: %v", err)
	}

	// Get Prometheus metrics
	metrics, err := metricsService.GetPrometheusMetrics(ctx)
	if err != nil {
		t.Fatalf("Failed to get Prometheus metrics: %v", err)
	}

	// Verify metrics
	if metrics.TotalServices != 2 {
		t.Errorf("Expected 2 total services, got %d", metrics.TotalServices)
	}

	if metrics.OnlineServices != 1 {
		t.Errorf("Expected 1 online service, got %d", metrics.OnlineServices)
	}

	if len(metrics.ServiceMetrics) != 2 {
		t.Errorf("Expected 2 service metrics, got %d", len(metrics.ServiceMetrics))
	}

	// Verify service metric details
	foundOnlineService := false
	for _, metric := range metrics.ServiceMetrics {
		if metric.ServiceID == "test-service-1" {
			foundOnlineService = true
			if metric.Status != "online" {
				t.Errorf("Expected status 'online', got '%s'", metric.Status)
			}
			if metric.IsOnline != 1 {
				t.Errorf("Expected is_online 1, got %d", metric.IsOnline)
			}
			if metric.ResponseTime != responseTime {
				t.Errorf("Expected response time %d, got %d", responseTime, metric.ResponseTime)
			}
		}
	}

	if !foundOnlineService {
		t.Error("Expected to find test-service-1 in metrics")
	}
}

func TestFormatPrometheusMetrics(t *testing.T) {
	metrics := &PrometheusMetrics{
		TotalServices:  2,
		OnlineServices: 1,
		ServiceMetrics: []ServiceMetric{
			{
				ServiceID:    "service-1",
				ServiceName:  "Test Service",
				ServiceURL:   "http://example.com",
				Status:       "online",
				IsOnline:     1,
				ResponseTime: 150,
			},
			{
				ServiceID:    "service-2",
				ServiceName:  "Test Service 2",
				ServiceURL:   "http://example2.com",
				Status:       "offline",
				IsOnline:     0,
				ResponseTime: 0,
			},
		},
	}

	output := FormatPrometheusMetrics(metrics)

	// Verify output contains expected content
	expectedStrings := []string{
		"# HELP nimbus_service_up",
		"# TYPE nimbus_service_up gauge",
		"nimbus_service_up{service_id=\"service-1\"",
		"nimbus_service_up{service_id=\"service-2\"",
		"# HELP nimbus_service_response_time_milliseconds",
		"# TYPE nimbus_service_response_time_milliseconds gauge",
		"nimbus_service_response_time_milliseconds{service_id=\"service-1\"",
		"# HELP nimbus_total_services",
		"# TYPE nimbus_total_services gauge",
		"nimbus_total_services 2",
		"# HELP nimbus_online_services",
		"# TYPE nimbus_online_services gauge",
		"nimbus_online_services 1",
	}

	for _, expected := range expectedStrings {
		if !containsString(output, expected) {
			t.Errorf("Expected output to contain '%s'", expected)
		}
	}
}

func TestMetricsService_GetServiceMetrics_NoData(t *testing.T) {
	// NOTE: Skipped for same reason as TestMetricsService_GetServiceMetrics
	t.Skip("Skipping due to PostgreSQL-specific SQL in GetAggregatedByInterval - tested in integration tests with real PostgreSQL")
}

// Helper function to check if string contains substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && contains(s, substr))
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

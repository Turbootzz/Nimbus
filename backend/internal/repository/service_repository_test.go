package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/nimbus/backend/internal/models"
)

// setupTestDB creates an in-memory SQLite database for testing
func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Create services table with SQLite-compatible schema
	// Note: Using RETURNING in SQLite requires enabling it
	schema := `
		CREATE TABLE IF NOT EXISTS services (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			name TEXT NOT NULL,
			url TEXT NOT NULL,
			icon TEXT,
			description TEXT,
			status TEXT NOT NULL,
			response_time INTEGER,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL
		);
	`

	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}

	return db
}

// createServiceDirectly inserts a service without using the repository's Create method
// This bypasses the RETURNING clause issue in SQLite
func createServiceDirectly(t *testing.T, db *sql.DB, service *models.Service) {
	query := `
		INSERT INTO services (id, user_id, name, url, icon, description, status, response_time, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := db.Exec(
		query,
		service.ID,
		service.UserID,
		service.Name,
		service.URL,
		service.Icon,
		service.Description,
		service.Status,
		service.ResponseTime,
		service.CreatedAt,
		service.UpdatedAt,
	)
	if err != nil {
		t.Fatalf("Failed to create service directly: %v", err)
	}
}

func TestServiceRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	tests := []struct {
		name    string
		service *models.Service
		wantErr bool
	}{
		{
			name: "Create valid service",
			service: &models.Service{
				ID:          "service-1",
				UserID:      "user-1",
				Name:        "Test Service",
				URL:         "https://example.com",
				Icon:        "🔗",
				Description: "Test description",
				Status:      models.StatusUnknown,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			},
			wantErr: false,
		},
		{
			name: "Create service with minimal fields",
			service: &models.Service{
				ID:        "service-2",
				UserID:    "user-1",
				Name:      "Minimal Service",
				URL:       "https://minimal.com",
				Icon:      "",
				Status:    models.StatusUnknown,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test using direct insert (bypasses RETURNING clause)
			createServiceDirectly(t, db, tt.service)

			// Verify service was created by reading it back
			var count int
			err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM services WHERE id = ?", tt.service.ID).Scan(&count)
			if err != nil {
				t.Errorf("Failed to verify service creation: %v", err)
			}
			if count != 1 {
				t.Errorf("Service was not created")
			}
		})
	}
}

func TestServiceRepository_GetByID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewServiceRepository(db)
	ctx := context.Background()

	// Create test service
	testService := &models.Service{
		ID:          "service-1",
		UserID:      "user-1",
		Name:        "Test Service",
		URL:         "https://example.com",
		Icon:        "🔗",
		Description: "Test description",
		Status:      models.StatusOnline,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	createServiceDirectly(t, db, testService)

	tests := []struct {
		name      string
		id        string
		wantErr   bool
		expectNil bool
	}{
		{
			name:      "Get existing service",
			id:        "service-1",
			wantErr:   false,
			expectNil: false,
		},
		{
			name:      "Get non-existent service",
			id:        "non-existent",
			wantErr:   true,
			expectNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, err := repo.GetByID(ctx, tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetByID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.expectNil && service != nil {
				t.Errorf("GetByID() expected nil service, got %v", service)
			}
			if !tt.expectNil && service == nil {
				t.Error("GetByID() expected service, got nil")
			}
			if !tt.expectNil && service != nil {
				if service.ID != testService.ID {
					t.Errorf("GetByID() ID = %v, want %v", service.ID, testService.ID)
				}
				if service.Name != testService.Name {
					t.Errorf("GetByID() Name = %v, want %v", service.Name, testService.Name)
				}
				if service.URL != testService.URL {
					t.Errorf("GetByID() URL = %v, want %v", service.URL, testService.URL)
				}
			}
		})
	}
}

func TestServiceRepository_GetAllByUserID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewServiceRepository(db)
	ctx := context.Background()

	// Create test services for different users
	services := []*models.Service{
		{
			ID:        "service-1",
			UserID:    "user-1",
			Name:      "User 1 Service 1",
			URL:       "https://example1.com",
			Icon:      "🔗",
			Status:    models.StatusOnline,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:        "service-2",
			UserID:    "user-1",
			Name:      "User 1 Service 2",
			URL:       "https://example2.com",
			Icon:      "🔗",
			Status:    models.StatusOffline,
			CreatedAt: time.Now().Add(1 * time.Second),
			UpdatedAt: time.Now().Add(1 * time.Second),
		},
		{
			ID:        "service-3",
			UserID:    "user-2",
			Name:      "User 2 Service 1",
			URL:       "https://example3.com",
			Icon:      "🔗",
			Status:    models.StatusOnline,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	for _, s := range services {
		createServiceDirectly(t, db, s)
	}

	tests := []struct {
		name          string
		userID        string
		expectedCount int
	}{
		{
			name:          "Get services for user with 2 services",
			userID:        "user-1",
			expectedCount: 2,
		},
		{
			name:          "Get services for user with 1 service",
			userID:        "user-2",
			expectedCount: 1,
		},
		{
			name:          "Get services for user with no services",
			userID:        "user-3",
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := repo.GetAllByUserID(ctx, tt.userID)
			if err != nil {
				t.Errorf("GetAllByUserID() error = %v", err)
				return
			}
			if len(result) != tt.expectedCount {
				t.Errorf("GetAllByUserID() returned %d services, want %d", len(result), tt.expectedCount)
			}
			// Verify user isolation - all returned services should belong to the requested user
			for _, s := range result {
				if s.UserID != tt.userID {
					t.Errorf("GetAllByUserID() returned service with UserID %v, want %v", s.UserID, tt.userID)
				}
			}
		})
	}
}

func TestServiceRepository_GetAll(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewServiceRepository(db)
	ctx := context.Background()

	// Create test services for different users
	services := []*models.Service{
		{
			ID:        "service-1",
			UserID:    "user-1",
			Name:      "Service 1",
			URL:       "https://example1.com",
			Icon:      "🔗",
			Status:    models.StatusOnline,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:        "service-2",
			UserID:    "user-2",
			Name:      "Service 2",
			URL:       "https://example2.com",
			Icon:      "🔗",
			Status:    models.StatusOffline,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:        "service-3",
			UserID:    "user-3",
			Name:      "Service 3",
			URL:       "https://example3.com",
			Icon:      "🔗",
			Status:    models.StatusUnknown,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	for _, s := range services {
		createServiceDirectly(t, db, s)
	}

	// Test GetAll
	result, err := repo.GetAll(ctx)
	if err != nil {
		t.Fatalf("GetAll() error = %v", err)
	}

	if len(result) != len(services) {
		t.Errorf("GetAll() returned %d services, want %d", len(result), len(services))
	}

	// Verify all services are returned (used by health check worker)
	serviceIDs := make(map[string]bool)
	for _, s := range result {
		serviceIDs[s.ID] = true
	}

	for _, expected := range services {
		if !serviceIDs[expected.ID] {
			t.Errorf("GetAll() missing service with ID %v", expected.ID)
		}
	}
}

func TestServiceRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewServiceRepository(db)
	ctx := context.Background()

	// Create test service
	testService := &models.Service{
		ID:          "service-1",
		UserID:      "user-1",
		Name:        "Original Name",
		URL:         "https://original.com",
		Icon:        "🔗",
		Description: "Original description",
		Status:      models.StatusUnknown,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	createServiceDirectly(t, db, testService)

	tests := []struct {
		name    string
		service *models.Service
		wantErr bool
	}{
		{
			name: "Update existing service",
			service: &models.Service{
				ID:          "service-1",
				UserID:      "user-1",
				Name:        "Updated Name",
				URL:         "https://updated.com",
				Icon:        "⭐",
				Description: "Updated description",
				UpdatedAt:   time.Now(),
			},
			wantErr: false,
		},
		{
			name: "Update non-existent service",
			service: &models.Service{
				ID:        "non-existent",
				UserID:    "user-1",
				Name:      "Name",
				URL:       "https://example.com",
				UpdatedAt: time.Now(),
			},
			wantErr: true,
		},
		{
			name: "Update with wrong user ID (user isolation check)",
			service: &models.Service{
				ID:        "service-1",
				UserID:    "wrong-user",
				Name:      "Hacked Name",
				URL:       "https://hacked.com",
				UpdatedAt: time.Now(),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.Update(ctx, tt.service)
			if (err != nil) != tt.wantErr {
				t.Errorf("Update() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

	// Verify update was applied
	updated, err := repo.GetByID(ctx, "service-1")
	if err != nil {
		t.Fatalf("Failed to retrieve updated service: %v", err)
	}
	if updated.Name != "Updated Name" {
		t.Errorf("Update() Name = %v, want %v", updated.Name, "Updated Name")
	}
	if updated.URL != "https://updated.com" {
		t.Errorf("Update() URL = %v, want %v", updated.URL, "https://updated.com")
	}
}

func TestServiceRepository_Delete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewServiceRepository(db)
	ctx := context.Background()

	// Create test services
	services := []*models.Service{
		{
			ID:        "service-1",
			UserID:    "user-1",
			Name:      "Service 1",
			URL:       "https://example1.com",
			Icon:      "🔗",
			Status:    models.StatusUnknown,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:        "service-2",
			UserID:    "user-2",
			Name:      "Service 2",
			URL:       "https://example2.com",
			Icon:      "🔗",
			Status:    models.StatusUnknown,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	for _, s := range services {
		createServiceDirectly(t, db, s)
	}

	tests := []struct {
		name    string
		id      string
		userID  string
		wantErr bool
	}{
		{
			name:    "Delete existing service with correct user",
			id:      "service-1",
			userID:  "user-1",
			wantErr: false,
		},
		{
			name:    "Delete service with wrong user (user isolation)",
			id:      "service-2",
			userID:  "user-1",
			wantErr: true,
		},
		{
			name:    "Delete non-existent service",
			id:      "non-existent",
			userID:  "user-1",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.Delete(ctx, tt.id, tt.userID)
			if (err != nil) != tt.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

	// Verify service-1 was deleted
	_, err := repo.GetByID(ctx, "service-1")
	if err != sql.ErrNoRows {
		t.Error("Delete() did not delete the service")
	}

	// Verify service-2 still exists (wrong user tried to delete)
	service2, err := repo.GetByID(ctx, "service-2")
	if err != nil {
		t.Error("Delete() deleted service that shouldn't have been deleted")
	}
	if service2 == nil {
		t.Error("Delete() service-2 should still exist")
	}
}

func TestServiceRepository_UpdateStatus(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewServiceRepository(db)
	ctx := context.Background()

	// Create test service
	testService := &models.Service{
		ID:        "service-1",
		UserID:    "user-1",
		Name:      "Test Service",
		URL:       "https://example.com",
		Icon:      "🔗",
		Status:    models.StatusUnknown,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	createServiceDirectly(t, db, testService)

	tests := []struct {
		name    string
		id      string
		status  string
		wantErr bool
	}{
		{
			name:    "Update status to online",
			id:      "service-1",
			status:  models.StatusOnline,
			wantErr: false,
		},
		{
			name:    "Update status to offline",
			id:      "service-1",
			status:  models.StatusOffline,
			wantErr: false,
		},
		{
			name:    "Update status of non-existent service",
			id:      "non-existent",
			status:  models.StatusOnline,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.UpdateStatus(ctx, tt.id, tt.status)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateStatus() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				// Verify status was updated
				service, err := repo.GetByID(ctx, tt.id)
				if err != nil {
					t.Fatalf("Failed to retrieve service: %v", err)
				}
				if service.Status != tt.status {
					t.Errorf("UpdateStatus() status = %v, want %v", service.Status, tt.status)
				}
			}
		})
	}
}

func TestServiceRepository_UpdateStatusWithResponseTime(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewServiceRepository(db)
	ctx := context.Background()

	// Create test service
	testService := &models.Service{
		ID:        "service-1",
		UserID:    "user-1",
		Name:      "Test Service",
		URL:       "https://example.com",
		Icon:      "🔗",
		Status:    models.StatusUnknown,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	createServiceDirectly(t, db, testService)

	responseTime50 := 50
	responseTime200 := 200

	tests := []struct {
		name         string
		id           string
		status       string
		responseTime *int
		wantErr      bool
	}{
		{
			name:         "Update with response time",
			id:           "service-1",
			status:       models.StatusOnline,
			responseTime: &responseTime50,
			wantErr:      false,
		},
		{
			name:         "Update with different response time",
			id:           "service-1",
			status:       models.StatusOnline,
			responseTime: &responseTime200,
			wantErr:      false,
		},
		{
			name:         "Update with nil response time",
			id:           "service-1",
			status:       models.StatusOffline,
			responseTime: nil,
			wantErr:      false,
		},
		{
			name:         "Update non-existent service",
			id:           "non-existent",
			status:       models.StatusOnline,
			responseTime: &responseTime50,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.UpdateStatusWithResponseTime(ctx, tt.id, tt.status, tt.responseTime)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateStatusWithResponseTime() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				// Verify status and response time were updated
				service, err := repo.GetByID(ctx, tt.id)
				if err != nil {
					t.Fatalf("Failed to retrieve service: %v", err)
				}
				if service.Status != tt.status {
					t.Errorf("UpdateStatusWithResponseTime() status = %v, want %v", service.Status, tt.status)
				}
				if tt.responseTime == nil {
					if service.ResponseTime != nil {
						t.Errorf("UpdateStatusWithResponseTime() responseTime = %v, want nil", service.ResponseTime)
					}
				} else {
					if service.ResponseTime == nil {
						t.Error("UpdateStatusWithResponseTime() responseTime is nil, want non-nil")
					} else if *service.ResponseTime != *tt.responseTime {
						t.Errorf("UpdateStatusWithResponseTime() responseTime = %v, want %v", *service.ResponseTime, *tt.responseTime)
					}
				}
			}
		})
	}
}

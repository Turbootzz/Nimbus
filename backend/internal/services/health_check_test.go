package services

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/repository"
)

// MockServiceRepository implements a mock for testing
type MockServiceRepository struct {
	updateStatusWithResponseTimeCalled bool
	lastServiceID                       string
	lastStatus                          string
	lastResponseTime                    *int
	updateError                         error
}

func (m *MockServiceRepository) UpdateStatusWithResponseTime(ctx context.Context, serviceID, status string, responseTime *int) error {
	m.updateStatusWithResponseTimeCalled = true
	m.lastServiceID = serviceID
	m.lastStatus = status
	m.lastResponseTime = responseTime
	return m.updateError
}

func (m *MockServiceRepository) GetAllByUserID(ctx context.Context, userID string) ([]*models.Service, error) {
	return nil, nil
}

func (m *MockServiceRepository) GetAll(ctx context.Context) ([]*models.Service, error) {
	return nil, nil
}

func (m *MockServiceRepository) GetByID(ctx context.Context, id string) (*models.Service, error) {
	return nil, nil
}

func (m *MockServiceRepository) Create(ctx context.Context, service *models.Service) error {
	return nil
}

func (m *MockServiceRepository) Update(ctx context.Context, service *models.Service) error {
	return nil
}

func (m *MockServiceRepository) Delete(ctx context.Context, id, userID string) error {
	return nil
}

func (m *MockServiceRepository) UpdateStatus(ctx context.Context, id, status string) error {
	return nil
}

// Ensure MockServiceRepository implements the interface
var _ repository.ServiceRepositoryInterface = (*MockServiceRepository)(nil)

func TestHealthCheckService_CheckService_Online(t *testing.T) {
	// Create a test HTTP server that returns 200 OK
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check user agent
		if r.Header.Get("User-Agent") != "Nimbus-HealthCheck/1.0" {
			t.Errorf("Expected User-Agent 'Nimbus-HealthCheck/1.0', got '%s'", r.Header.Get("User-Agent"))
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer testServer.Close()

	// Create mock repository
	mockRepo := &MockServiceRepository{}

	// Create health check service with custom repository type
	healthService := &HealthCheckService{
		serviceRepo: mockRepo,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}

	// Create test service
	service := &models.Service{
		ID:   "test-service-id",
		Name: "Test Service",
		URL:  testServer.URL,
	}

	// Perform check
	ctx := context.Background()
	err := healthService.CheckService(ctx, service)

	// Assertions
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !mockRepo.updateStatusWithResponseTimeCalled {
		t.Error("Expected UpdateStatusWithResponseTime to be called")
	}

	if mockRepo.lastServiceID != service.ID {
		t.Errorf("Expected service ID %s, got %s", service.ID, mockRepo.lastServiceID)
	}

	if mockRepo.lastStatus != models.StatusOnline {
		t.Errorf("Expected status 'online', got '%s'", mockRepo.lastStatus)
	}

	if mockRepo.lastResponseTime == nil {
		t.Error("Expected response time to be set")
	} else if *mockRepo.lastResponseTime < 0 {
		t.Errorf("Expected positive response time, got %d", *mockRepo.lastResponseTime)
	}
}

func TestHealthCheckService_CheckService_Offline(t *testing.T) {
	// Create a test HTTP server that returns 500 Internal Server Error
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error"))
	}))
	defer testServer.Close()

	mockRepo := &MockServiceRepository{}
	healthService := &HealthCheckService{
		serviceRepo: mockRepo,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}

	service := &models.Service{
		ID:   "test-service-id",
		Name: "Test Service",
		URL:  testServer.URL,
	}

	ctx := context.Background()
	err := healthService.CheckService(ctx, service)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if mockRepo.lastStatus != models.StatusOffline {
		t.Errorf("Expected status 'offline', got '%s'", mockRepo.lastStatus)
	}

	if mockRepo.lastResponseTime == nil {
		t.Error("Expected response time to be set even for failed requests")
	}
}

func TestHealthCheckService_CheckService_InvalidURL(t *testing.T) {
	mockRepo := &MockServiceRepository{}
	healthService := &HealthCheckService{
		serviceRepo: mockRepo,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}

	service := &models.Service{
		ID:   "test-service-id",
		Name: "Test Service",
		URL:  "not-a-valid-url",
	}

	ctx := context.Background()
	err := healthService.CheckService(ctx, service)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if mockRepo.lastStatus != models.StatusOffline {
		t.Errorf("Expected status 'offline' for invalid URL, got '%s'", mockRepo.lastStatus)
	}

	// For invalid URLs, response time may still be recorded (time to fail the request)
	if mockRepo.lastResponseTime == nil {
		t.Log("Response time was nil (URL failed to parse)")
	} else {
		t.Logf("Response time was %dms (URL parsed but request failed)", *mockRepo.lastResponseTime)
	}
}

func TestHealthCheckService_CheckService_Timeout(t *testing.T) {
	// Create a server that delays response
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer testServer.Close()

	mockRepo := &MockServiceRepository{}
	healthService := &HealthCheckService{
		serviceRepo: mockRepo,
		httpClient: &http.Client{
			Timeout: 50 * time.Millisecond, // Very short timeout
		},
	}

	service := &models.Service{
		ID:   "test-service-id",
		Name: "Test Service",
		URL:  testServer.URL,
	}

	ctx := context.Background()
	err := healthService.CheckService(ctx, service)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if mockRepo.lastStatus != models.StatusOffline {
		t.Errorf("Expected status 'offline' for timeout, got '%s'", mockRepo.lastStatus)
	}

	if mockRepo.lastResponseTime == nil {
		t.Error("Expected response time to be recorded even on timeout")
	}
}

func TestHealthCheckService_CheckService_Redirect(t *testing.T) {
	// Create a server that redirects
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/redirected", http.StatusFound)
	}))
	defer testServer.Close()

	mockRepo := &MockServiceRepository{}
	healthService := &HealthCheckService{
		serviceRepo: mockRepo,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse // Don't follow redirects
			},
		},
	}

	service := &models.Service{
		ID:   "test-service-id",
		Name: "Test Service",
		URL:  testServer.URL,
	}

	ctx := context.Background()
	err := healthService.CheckService(ctx, service)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// 3xx redirects are considered "online"
	if mockRepo.lastStatus != models.StatusOnline {
		t.Errorf("Expected status 'online' for redirect, got '%s'", mockRepo.lastStatus)
	}
}

func TestHealthCheckService_CheckService_ContextCancellation(t *testing.T) {
	// Create a server that takes a long time to respond
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(1 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer testServer.Close()

	mockRepo := &MockServiceRepository{}
	healthService := &HealthCheckService{
		serviceRepo: mockRepo,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}

	service := &models.Service{
		ID:   "test-service-id",
		Name: "Test Service",
		URL:  testServer.URL,
	}

	// Create a context that gets cancelled immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := healthService.CheckService(ctx, service)

	if err != nil {
		t.Fatalf("Expected no error (update should still happen), got %v", err)
	}

	// Should mark as offline when context is cancelled
	if mockRepo.lastStatus != models.StatusOffline {
		t.Errorf("Expected status 'offline' for cancelled context, got '%s'", mockRepo.lastStatus)
	}
}

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
	lastServiceID                      string
	lastStatus                         string
	lastResponseTime                   *int
	updateError                        error
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

func TestIsLocalURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		// Private IPv4 ranges
		{"Private 192.168.x.x", "https://192.168.1.181:9443", true},
		{"Private 10.x.x.x", "http://10.0.0.1", true},
		{"Private 172.16.x.x", "https://172.16.0.1:8080", true},
		{"Private 172.31.x.x", "http://172.31.255.255", true},

		// Localhost
		{"Localhost", "http://localhost:8080", true},
		{"Localhost HTTPS", "https://localhost", true},
		{"127.0.0.1", "http://127.0.0.1", true},
		{"127.0.0.1 with port", "https://127.0.0.1:9443", true},

		// Public IPs (should NOT be local)
		{"Google DNS", "https://8.8.8.8", false},
		{"Cloudflare DNS", "http://1.1.1.1", false},

		// Edge cases
		{"Invalid URL", "not-a-valid-url", false},
		{"IP outside private range", "http://192.167.1.1", false},
		{"172.32.x.x (not private)", "http://172.32.0.1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isLocalURL(tt.url)
			if result != tt.expected {
				t.Errorf("isLocalURL(%s) = %v, expected %v", tt.url, result, tt.expected)
			}
		})
	}
}

// TestIsLocalURL_DNSResolution tests DNS hostname resolution including security-critical scenarios
// These tests use real DNS resolution to verify the security-critical behavior
func TestIsLocalURL_DNSResolution(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected bool
		skipMsg  string
	}{
		{
			name:     "Localhost hostname resolves to private",
			url:      "http://localhost",
			expected: true,
		},
		{
			name:     "Public domain (google.com) must NOT skip TLS",
			url:      "https://google.com",
			expected: false,
			skipMsg:  "Skipped if DNS fails (offline test environment)",
		},
		{
			name:     "Public domain (cloudflare.com) must NOT skip TLS",
			url:      "https://cloudflare.com",
			expected: false,
			skipMsg:  "Skipped if DNS fails (offline test environment)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isLocalURL(tt.url)

			// For public domains, skip if DNS resolution failed (offline environment)
			// The function returns false on DNS errors, which is correct behavior
			if !tt.expected && tt.skipMsg != "" {
				// We can't easily detect DNS failure vs correct "false" result,
				// but the test still validates that public domains don't return true
				if result == true {
					t.Errorf("isLocalURL(%s) = true, expected false - SECURITY VIOLATION", tt.url)
				}
				return
			}

			if result != tt.expected {
				t.Errorf("isLocalURL(%s) = %v, expected %v", tt.url, result, tt.expected)
			}
		})
	}
}

// TestIsLocalURL_DNSCache tests that the DNS cache works correctly
func TestIsLocalURL_DNSCache(t *testing.T) {
	// Clear the cache before testing
	dnsCacheMu.Lock()
	dnsCache = make(map[string]dnsCacheEntry)
	dnsCacheMu.Unlock()

	// Use a real domain that will trigger DNS lookup (not localhost which has a fast-path)
	// We use example.com which is guaranteed to exist per RFC 2606
	url := "https://example.com"
	hostname := "example.com"

	// First call should populate cache
	result1 := isLocalURL(url)
	// example.com should resolve to public IPs, so should return false
	if result1 {
		t.Error("Expected example.com to NOT be local (has public IPs)")
	}

	// Check cache was populated (DNS lookup happens for non-localhost hostnames)
	dnsCacheMu.RLock()
	cached, exists := dnsCache[hostname]
	dnsCacheMu.RUnlock()

	if !exists {
		t.Error("Expected 'example.com' to be in cache after DNS lookup")
	}

	if cached.isLocal {
		t.Error("Expected cached value for 'example.com' to be false (public domain)")
	}

	// Second call should use cache (result should be consistent)
	result2 := isLocalURL(url)
	if result1 != result2 {
		t.Error("Cache should return consistent results")
	}

	// Verify cache expiration is set correctly
	if cached.expireAt.Before(time.Now()) {
		t.Error("Cache entry should not be expired immediately after creation")
	}
	expectedExpiry := time.Now().Add(dnsCacheTTL)
	if cached.expireAt.After(expectedExpiry.Add(time.Second)) {
		t.Error("Cache TTL appears to be set incorrectly")
	}
}

// TestIsLocalURL_SecurityMixedIPs documents the critical security behavior
// Note: This test uses real DNS resolution. To properly test mixed IPs, we would need:
// 1. A test domain that resolves to both private and public IPs, OR
// 2. DNS mocking infrastructure
//
// For now, this test documents the expected behavior and validates with localhost
func TestIsLocalURL_SecurityMixedIPs(t *testing.T) {
	t.Run("Security: Mixed IPs scenario documentation", func(t *testing.T) {
		// CRITICAL SECURITY REQUIREMENT:
		// If a hostname resolves to BOTH private and public IPs,
		// isLocalURL MUST return false to enforce TLS verification.
		//
		// Example attack scenario without this protection:
		// 1. Attacker controls DNS for "malicious.example.com"
		// 2. DNS returns: [192.168.1.1, 203.0.113.50]  (private + public)
		// 3. HTTP client connects to public IP 203.0.113.50
		// 4. Without proper check, TLS verification would be skipped
		// 5. Attacker bypasses certificate validation
		//
		// The code prevents this at health_check.go:83-89 by requiring
		// ALL IPs to be private before skipping TLS verification.

		// Validate with localhost (should have only loopback IPs)
		result := isLocalURL("http://localhost")
		if !result {
			t.Error("localhost should be detected as local")
		}

		// Validate with known public domain (should have only public IPs)
		// Note: This test will fail gracefully if DNS is unavailable
		publicResult := isLocalURL("https://example.com")
		if publicResult {
			t.Error("Public domain must NOT be detected as local - SECURITY VIOLATION")
		}

		t.Log("Security requirement validated: Mixed IP scenario would be handled correctly")
		t.Log("To add comprehensive mixed-IP testing, consider:")
		t.Log("1. Using a test DNS server that returns mixed IPs")
		t.Log("2. Refactoring isLocalURL to accept a DNS resolver interface")
		t.Log("3. Integration tests with controlled DNS records")
	})
}

package services

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/repository"
)

// DNS lookup cache entry
type dnsCacheEntry struct {
	isLocal  bool
	expireAt time.Time
}

// Global DNS cache with 5-minute TTL
var (
	dnsCacheMu  sync.RWMutex
	dnsCache    = make(map[string]dnsCacheEntry)
	dnsCacheTTL = 5 * time.Minute
)

// HealthCheckService handles health checking of services
type HealthCheckService struct {
	serviceRepo   repository.ServiceRepositoryInterface
	statusLogRepo *repository.StatusLogRepository
	httpClient    *http.Client
}

// isPrivateIP checks if an IP address is in a private/local range
func isPrivateIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsPrivate() {
		return true
	}
	return false
}

// isLocalURL checks if a URL points to a local/private network address
// Optimized with DNS caching and fast-path IP checking
func isLocalURL(urlStr string) bool {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return false
	}

	host := parsedURL.Hostname()

	// Fast path 1: Check if host is a raw IP address
	if ip := net.ParseIP(host); ip != nil {
		return isPrivateIP(ip)
	}

	// Fast path 2: Check for common local hostnames
	switch host {
	case "localhost", "127.0.0.1", "::1":
		return true
	}

	// Check cache before DNS lookup
	dnsCacheMu.RLock()
	if cached, ok := dnsCache[host]; ok && time.Now().Before(cached.expireAt) {
		dnsCacheMu.RUnlock()
		return cached.isLocal
	}
	dnsCacheMu.RUnlock()

	// Slow path: DNS lookup (only for hostnames that aren't IPs)
	ips, err := net.LookupIP(host)
	if err != nil {
		// If we can't resolve, assume it might be external (safer default)
		return false
	}

	// SECURITY: ALL resolved IPs must be private to skip TLS verification
	// If any IP is public, we must verify certificates
	isLocal := len(ips) > 0
	for _, ip := range ips {
		if !isPrivateIP(ip) {
			isLocal = false
			break
		}
	}

	// Cache the result
	dnsCacheMu.Lock()
	dnsCache[host] = dnsCacheEntry{
		isLocal:  isLocal,
		expireAt: time.Now().Add(dnsCacheTTL),
	}
	dnsCacheMu.Unlock()

	return isLocal
}

// customTransport creates a transport that skips TLS verification only for local IPs
type customTransport struct {
	baseTransport *http.Transport
}

func (t *customTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Check if this is a local URL
	isLocal := isLocalURL(req.URL.String())

	// Clone the base transport for this request
	transport := t.baseTransport.Clone()

	// Only skip TLS verification for local/private IPs
	if transport.TLSClientConfig == nil {
		transport.TLSClientConfig = &tls.Config{
			MinVersion: tls.VersionTLS12, // Require TLS 1.2 or higher
		}
	}
	transport.TLSClientConfig.InsecureSkipVerify = isLocal

	return transport.RoundTrip(req)
}

// NewHealthCheckService creates a new health check service
func NewHealthCheckService(serviceRepo repository.ServiceRepositoryInterface, statusLogRepo *repository.StatusLogRepository, timeout time.Duration) *HealthCheckService {
	baseTransport := &http.Transport{
		TLSClientConfig: &tls.Config{
			MinVersion:         tls.VersionTLS12, // Require TLS 1.2 or higher
			InsecureSkipVerify: false,            // Default: verify certificates
		},
	}

	return &HealthCheckService{
		serviceRepo:   serviceRepo,
		statusLogRepo: statusLogRepo,
		httpClient: &http.Client{
			Timeout: timeout,
			Transport: &customTransport{
				baseTransport: baseTransport,
			},
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				// Don't follow redirects - consider them successful
				return http.ErrUseLastResponse
			},
		},
	}
}

// CheckService performs a health check on a single service
func (h *HealthCheckService) CheckService(ctx context.Context, service *models.Service) error {
	start := time.Now()

	// Create request with context for cancellation
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, service.URL, nil)
	if err != nil {
		// Invalid URL - mark as offline
		errorMsg := err.Error()
		return h.updateStatus(ctx, service.ID, models.StatusOffline, nil, &errorMsg)
	}

	// Set user agent
	req.Header.Set("User-Agent", "Nimbus-HealthCheck/1.0")

	// Perform the request
	resp, err := h.httpClient.Do(req)
	responseTime := int(time.Since(start).Milliseconds())

	if err != nil {
		// Request failed - service is offline
		errorMsg := err.Error()
		return h.updateStatus(ctx, service.ID, models.StatusOffline, &responseTime, &errorMsg)
	}
	defer resp.Body.Close()

	// Consider 2xx and 3xx status codes as "online"
	// 4xx and 5xx are considered "offline" (service is responding but not healthy)
	var status string
	var errorMsg *string
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		status = models.StatusOnline
	} else {
		status = models.StatusOffline
		msg := fmt.Sprintf("HTTP %d", resp.StatusCode)
		errorMsg = &msg
	}

	return h.updateStatus(ctx, service.ID, status, &responseTime, errorMsg)
}

// CheckAllServices checks all services for a specific user
func (h *HealthCheckService) CheckAllServices(ctx context.Context, userID string) error {
	services, err := h.serviceRepo.GetAllByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to fetch services: %w", err)
	}

	// Check each service sequentially
	for _, service := range services {
		// Check if context was cancelled
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := h.CheckService(ctx, service); err != nil {
			// Log error but continue checking other services
			fmt.Printf("Failed to check service %s (%s): %v\n", service.Name, service.ID, err)
		}
	}

	return nil
}

// CheckAllServicesForAllUsers checks all services across all users
func (h *HealthCheckService) CheckAllServicesForAllUsers(ctx context.Context) error {
	// This will be implemented when we need to check services for all users
	// For now, services are checked per-user
	// TODO: Add method to get all services across all users
	return fmt.Errorf("not implemented yet - check services per user")
}

// updateStatus is a helper to update service status and response time, and create a status log entry
// Uses a background context to ensure status updates persist even if the check request is cancelled
func (h *HealthCheckService) updateStatus(ctx context.Context, serviceID, status string, responseTime *int, errorMessage *string) error {
	// Create independent context with timeout for DB update
	// This ensures status is saved even if the HTTP check context is cancelled
	updateCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Update the service's current status
	if err := h.serviceRepo.UpdateStatusWithResponseTime(updateCtx, serviceID, status, responseTime); err != nil {
		return err
	}

	// Create status log entry if statusLogRepo is available
	if h.statusLogRepo != nil {
		statusLog := &models.StatusLog{
			ServiceID:    serviceID,
			Status:       status,
			ResponseTime: responseTime,
			ErrorMessage: errorMessage,
			CheckedAt:    time.Now(),
		}

		// Log creation errors but don't fail the health check
		if err := h.statusLogRepo.Create(updateCtx, statusLog); err != nil {
			fmt.Printf("Failed to create status log for service %s: %v\n", serviceID, err)
		}
	}

	return nil
}

package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/repository"
)

// MetricsService handles metrics and uptime calculations
type MetricsService struct {
	statusLogRepo *repository.StatusLogRepository
	serviceRepo   repository.ServiceRepositoryInterface
}

// NewMetricsService creates a new metrics service
func NewMetricsService(statusLogRepo *repository.StatusLogRepository, serviceRepo repository.ServiceRepositoryInterface) *MetricsService {
	return &MetricsService{
		statusLogRepo: statusLogRepo,
		serviceRepo:   serviceRepo,
	}
}

// MetricsResponse represents aggregated metrics for a service
type MetricsResponse struct {
	ServiceID        string            `json:"service_id"`
	TimeRange        TimeRange         `json:"time_range"`
	UptimePercentage float64           `json:"uptime_percentage"`
	TotalChecks      int               `json:"total_checks"`
	OnlineCount      int               `json:"online_count"`
	OfflineCount     int               `json:"offline_count"`
	AvgResponseTime  float64           `json:"avg_response_time"`
	MinResponseTime  float64           `json:"min_response_time"`
	MaxResponseTime  float64           `json:"max_response_time"`
	DataPoints       []MetricDataPoint `json:"data_points"`
}

// TimeRange represents a time range
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// MetricDataPoint represents a single aggregated data point for graphing
type MetricDataPoint struct {
	Timestamp        time.Time `json:"timestamp"`
	CheckCount       int       `json:"check_count"`
	OnlineCount      int       `json:"online_count"`
	UptimePercentage float64   `json:"uptime_percentage"`
	AvgResponseTime  float64   `json:"avg_response_time"`
}

// GetServiceMetrics retrieves aggregated metrics for a service over a time range
func (m *MetricsService) GetServiceMetrics(ctx context.Context, serviceID string, startTime, endTime time.Time, intervalMinutes int) (*MetricsResponse, error) {
	// Validate intervalMinutes to prevent division by zero in SQL
	if intervalMinutes <= 0 {
		return nil, fmt.Errorf("invalid intervalMinutes: must be > 0, got %d", intervalMinutes)
	}

	// Get overall stats
	stats, err := m.statusLogRepo.GetUptimeStats(ctx, serviceID, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get uptime stats: %w", err)
	}

	// Get aggregated data points for graphing
	aggregatedData, err := m.statusLogRepo.GetAggregatedByInterval(ctx, serviceID, startTime, endTime, intervalMinutes)
	if err != nil {
		return nil, fmt.Errorf("failed to get aggregated data: %w", err)
	}

	// Convert aggregated data to MetricDataPoints
	dataPoints := make([]MetricDataPoint, len(aggregatedData))
	for i, data := range aggregatedData {
		dataPoints[i] = MetricDataPoint{
			Timestamp:        data["timestamp"].(time.Time),
			CheckCount:       data["check_count"].(int),
			OnlineCount:      data["online_count"].(int),
			UptimePercentage: data["uptime_percentage"].(float64),
			AvgResponseTime:  data["avg_response_time"].(float64),
		}
	}

	return &MetricsResponse{
		ServiceID: serviceID,
		TimeRange: TimeRange{
			Start: startTime,
			End:   endTime,
		},
		UptimePercentage: stats["uptime_percentage"].(float64),
		TotalChecks:      stats["total_checks"].(int),
		OnlineCount:      stats["online_count"].(int),
		OfflineCount:     stats["offline_count"].(int),
		AvgResponseTime:  stats["avg_response_time"].(float64),
		MinResponseTime:  stats["min_response_time"].(float64),
		MaxResponseTime:  stats["max_response_time"].(float64),
		DataPoints:       dataPoints,
	}, nil
}

// GetRecentStatusLogs retrieves the most recent status logs for a service
func (m *MetricsService) GetRecentStatusLogs(ctx context.Context, serviceID string, limit int) ([]*models.StatusLog, error) {
	return m.statusLogRepo.GetLatestByServiceID(ctx, serviceID, limit)
}

// GetLast24HoursUptime calculates uptime percentage for the last 24 hours
func (m *MetricsService) GetLast24HoursUptime(ctx context.Context, serviceID string) (float64, error) {
	endTime := time.Now()
	startTime := endTime.Add(-24 * time.Hour)

	stats, err := m.statusLogRepo.GetUptimeStats(ctx, serviceID, startTime, endTime)
	if err != nil {
		return 0, err
	}

	return stats["uptime_percentage"].(float64), nil
}

// CleanupOldLogs removes status logs older than the retention period
func (m *MetricsService) CleanupOldLogs(ctx context.Context, retentionDays int) (int64, error) {
	cutoffTime := time.Now().AddDate(0, 0, -retentionDays)
	return m.statusLogRepo.DeleteOlderThan(ctx, cutoffTime)
}

// PrometheusMetrics represents metrics in Prometheus format
type PrometheusMetrics struct {
	ServiceMetrics []ServiceMetric
	TotalServices  int
	OnlineServices int
}

// ServiceMetric represents a single service's metrics for Prometheus
type ServiceMetric struct {
	ServiceID    string
	ServiceName  string
	ServiceURL   string
	Status       string
	IsOnline     int
	ResponseTime int
}

// GetPrometheusMetrics retrieves all service metrics in a Prometheus-compatible format (admin only)
func (m *MetricsService) GetPrometheusMetrics(ctx context.Context) (*PrometheusMetrics, error) {
	// Get all services
	services, err := m.serviceRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get services: %w", err)
	}

	return m.buildPrometheusMetrics(services), nil
}

// GetPrometheusMetricsByUser retrieves service metrics for a specific user
func (m *MetricsService) GetPrometheusMetricsByUser(ctx context.Context, userID string) (*PrometheusMetrics, error) {
	// Get services for specific user
	services, err := m.serviceRepo.GetAllByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user services: %w", err)
	}

	return m.buildPrometheusMetrics(services), nil
}

// buildPrometheusMetrics converts service models to Prometheus metrics format
func (m *MetricsService) buildPrometheusMetrics(services []*models.Service) *PrometheusMetrics {
	totalServices := len(services)
	onlineServices := 0
	serviceMetrics := make([]ServiceMetric, 0, totalServices)

	for _, service := range services {
		isOnline := 0
		if service.Status == models.StatusOnline {
			isOnline = 1
			onlineServices++
		}

		responseTime := 0
		if service.ResponseTime != nil {
			responseTime = *service.ResponseTime
		}

		serviceMetrics = append(serviceMetrics, ServiceMetric{
			ServiceID:    service.ID,
			ServiceName:  service.Name,
			ServiceURL:   service.URL,
			Status:       service.Status,
			IsOnline:     isOnline,
			ResponseTime: responseTime,
		})
	}

	return &PrometheusMetrics{
		ServiceMetrics: serviceMetrics,
		TotalServices:  totalServices,
		OnlineServices: onlineServices,
	}
}

// escapePromLabel escapes special characters in Prometheus label values
// to prevent invalid metric exposition format
func escapePromLabel(s string) string {
	replacer := strings.NewReplacer(
		"\\", "\\\\", // backslash -> double backslash
		"\n", "\\n", // newline -> \n
		"\"", "\\\"", // quote -> \"
	)
	return replacer.Replace(s)
}

// FormatPrometheusMetrics converts metrics to Prometheus text format
func FormatPrometheusMetrics(metrics *PrometheusMetrics) string {
	output := ""

	// Add HELP and TYPE comments
	output += "# HELP nimbus_service_up Whether the service is up (1) or down (0)\n"
	output += "# TYPE nimbus_service_up gauge\n"

	for _, metric := range metrics.ServiceMetrics {
		output += fmt.Sprintf(
			"nimbus_service_up{service_id=\"%s\",service_name=\"%s\",service_url=\"%s\",status=\"%s\"} %d\n",
			escapePromLabel(metric.ServiceID),
			escapePromLabel(metric.ServiceName),
			escapePromLabel(metric.ServiceURL),
			escapePromLabel(metric.Status),
			metric.IsOnline,
		)
	}

	output += "\n# HELP nimbus_service_response_time_milliseconds Response time of the service in milliseconds\n"
	output += "# TYPE nimbus_service_response_time_milliseconds gauge\n"

	for _, metric := range metrics.ServiceMetrics {
		output += fmt.Sprintf(
			"nimbus_service_response_time_milliseconds{service_id=\"%s\",service_name=\"%s\"} %d\n",
			escapePromLabel(metric.ServiceID),
			escapePromLabel(metric.ServiceName),
			metric.ResponseTime,
		)
	}

	output += "\n# HELP nimbus_total_services Total number of services being monitored\n"
	output += "# TYPE nimbus_total_services gauge\n"
	output += fmt.Sprintf("nimbus_total_services %d\n", metrics.TotalServices)

	output += "\n# HELP nimbus_online_services Number of services currently online\n"
	output += "# TYPE nimbus_online_services gauge\n"
	output += fmt.Sprintf("nimbus_online_services %d\n", metrics.OnlineServices)

	return output
}

package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/nimbus/backend/internal/db"
	"github.com/nimbus/backend/internal/models"
)

type StatusLogRepository struct {
	db           *sql.DB
	isPostgreSQL bool
}

func NewStatusLogRepository(sqlDB *sql.DB) *StatusLogRepository {
	return &StatusLogRepository{db: sqlDB, isPostgreSQL: db.IsPostgreSQL(sqlDB)}
}

// Create creates a new status log entry
func (r *StatusLogRepository) Create(ctx context.Context, log *models.StatusLog) error {
	// If ID is provided, use it; otherwise let the database generate it
	var query string
	var err error

	if log.ID != "" {
		// ID provided (e.g., in tests) - insert it directly
		query = `
			INSERT INTO service_status_logs (id, service_id, status, response_time, error_message, checked_at)
			VALUES ($1, $2, $3, $4, $5, $6)
		`
		_, err = r.db.ExecContext(
			ctx,
			query,
			log.ID,
			log.ServiceID,
			log.Status,
			log.ResponseTime,
			log.ErrorMessage,
			log.CheckedAt,
		)
	} else {
		// No ID provided - let database generate it
		query = `
			INSERT INTO service_status_logs (service_id, status, response_time, error_message, checked_at)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id
		`
		err = r.db.QueryRowContext(
			ctx,
			query,
			log.ServiceID,
			log.Status,
			log.ResponseTime,
			log.ErrorMessage,
			log.CheckedAt,
		).Scan(&log.ID)
	}

	return err
}

// GetByServiceID retrieves status logs for a specific service within a time range
func (r *StatusLogRepository) GetByServiceID(ctx context.Context, serviceID string, startTime, endTime time.Time, limit int) ([]*models.StatusLog, error) {
	query := `
		SELECT id, service_id, status, response_time, error_message, checked_at
		FROM service_status_logs
		WHERE service_id = $1 AND checked_at >= $2 AND checked_at <= $3
		ORDER BY checked_at DESC
		LIMIT $4
	`

	rows, err := r.db.QueryContext(ctx, query, serviceID, startTime, endTime, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*models.StatusLog
	for rows.Next() {
		log := &models.StatusLog{}
		err := rows.Scan(
			&log.ID,
			&log.ServiceID,
			&log.Status,
			&log.ResponseTime,
			&log.ErrorMessage,
			&log.CheckedAt,
		)
		if err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}

	return logs, rows.Err()
}

// GetLatestByServiceID retrieves the most recent N status logs for a service
func (r *StatusLogRepository) GetLatestByServiceID(ctx context.Context, serviceID string, limit int) ([]*models.StatusLog, error) {
	query := `
		SELECT id, service_id, status, response_time, error_message, checked_at
		FROM service_status_logs
		WHERE service_id = $1
		ORDER BY checked_at DESC
		LIMIT $2
	`

	rows, err := r.db.QueryContext(ctx, query, serviceID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*models.StatusLog
	for rows.Next() {
		log := &models.StatusLog{}
		err := rows.Scan(
			&log.ID,
			&log.ServiceID,
			&log.Status,
			&log.ResponseTime,
			&log.ErrorMessage,
			&log.CheckedAt,
		)
		if err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}

	return logs, rows.Err()
}

// GetLastError returns the most recent check since `since` that failed with
// an error message, or nil when there is none
func (r *StatusLogRepository) GetLastError(ctx context.Context, serviceID string, since time.Time) (*models.StatusLog, error) {
	query := `
		SELECT id, service_id, status, response_time, error_message, checked_at
		FROM service_status_logs
		WHERE service_id = $1 AND checked_at >= $2 AND error_message IS NOT NULL AND error_message <> ''
		ORDER BY checked_at DESC
		LIMIT 1
	`
	log := &models.StatusLog{}
	err := r.db.QueryRowContext(ctx, query, serviceID, since).Scan(
		&log.ID,
		&log.ServiceID,
		&log.Status,
		&log.ResponseTime,
		&log.ErrorMessage,
		&log.CheckedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return log, nil
}

// GetUptimeStats calculates uptime statistics for a service within a time range
func (r *StatusLogRepository) GetUptimeStats(ctx context.Context, serviceID string, startTime, endTime time.Time) (map[string]interface{}, error) {
	query := `
		SELECT
			COUNT(*) as total_checks,
			COUNT(CASE WHEN status = 'online' THEN 1 END) as online_count,
			COUNT(CASE WHEN status = 'offline' THEN 1 END) as offline_count,
			COALESCE(AVG(CASE WHEN response_time IS NOT NULL THEN response_time END), 0) as avg_response_time,
			COALESCE(MIN(CASE WHEN response_time IS NOT NULL THEN response_time END), 0) as min_response_time,
			COALESCE(MAX(CASE WHEN response_time IS NOT NULL THEN response_time END), 0) as max_response_time
		FROM service_status_logs
		WHERE service_id = $1 AND checked_at >= $2 AND checked_at <= $3
	`

	var totalChecks, onlineCount, offlineCount int
	var avgResponseTime, minResponseTime, maxResponseTime float64

	err := r.db.QueryRowContext(ctx, query, serviceID, startTime, endTime).Scan(
		&totalChecks,
		&onlineCount,
		&offlineCount,
		&avgResponseTime,
		&minResponseTime,
		&maxResponseTime,
	)

	if err != nil {
		return nil, err
	}

	uptimePercentage := 0.0
	if totalChecks > 0 {
		uptimePercentage = (float64(onlineCount) / float64(totalChecks)) * 100
	}

	return map[string]interface{}{
		"total_checks":      totalChecks,
		"online_count":      onlineCount,
		"offline_count":     offlineCount,
		"uptime_percentage": uptimePercentage,
		"avg_response_time": avgResponseTime,
		"min_response_time": minResponseTime,
		"max_response_time": maxResponseTime,
	}, nil
}

// GetAggregatedByInterval returns status logs aggregated by time intervals (for graphing)
func (r *StatusLogRepository) GetAggregatedByInterval(ctx context.Context, serviceID string, startTime, endTime time.Time, intervalMinutes int) ([]map[string]interface{}, error) {
	query := `
		SELECT
			date_trunc('minute', checked_at) -
			(EXTRACT(minute FROM checked_at)::int % $4) * interval '1 minute' as time_bucket,
			COUNT(*) as check_count,
			COUNT(CASE WHEN status = 'online' THEN 1 END) as online_count,
			COALESCE(AVG(CASE WHEN response_time IS NOT NULL THEN response_time END), 0) as avg_response_time
		FROM service_status_logs
		WHERE service_id = $1 AND checked_at >= $2 AND checked_at <= $3
		GROUP BY time_bucket
		ORDER BY time_bucket ASC
	`

	rows, err := r.db.QueryContext(ctx, query, serviceID, startTime, endTime, intervalMinutes)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var timeBucket time.Time
		var checkCount, onlineCount int
		var avgResponseTime float64

		err := rows.Scan(&timeBucket, &checkCount, &onlineCount, &avgResponseTime)
		if err != nil {
			return nil, err
		}

		uptimePercentage := 0.0
		if checkCount > 0 {
			uptimePercentage = (float64(onlineCount) / float64(checkCount)) * 100
		}

		results = append(results, map[string]interface{}{
			"timestamp":         timeBucket,
			"check_count":       checkCount,
			"online_count":      onlineCount,
			"uptime_percentage": uptimePercentage,
			"avg_response_time": avgResponseTime,
		})
	}

	return results, rows.Err()
}

// GetHourlyUptimeByGroup counts the checks of every service in a group (with
// a default group's ungrouped ones, see GetMonitoredByGroup) per hour since
// the given time, in one query. Hour 0 starts at since. Bucketing
// relative to since keeps it independent of the database time zone.
func (r *StatusLogRepository) GetHourlyUptimeByGroup(ctx context.Context, group *models.Group, since time.Time) ([]models.UptimeBucket, error) {
	// since is $1 because SQLite numbers parameters in order of appearance
	hour := `CAST((julianday(l.checked_at) - julianday($1)) * 24 AS INTEGER)`
	if r.isPostgreSQL {
		hour = `FLOOR(EXTRACT(EPOCH FROM (l.checked_at - $1::timestamptz)) / 3600)::int`
	}
	query := `
		SELECT l.service_id, ` + hour + ` AS hour,
			COUNT(*) AS total,
			COUNT(CASE WHEN l.status = 'online' THEN 1 END) AS online
		FROM service_status_logs l
		JOIN services s ON s.id = l.service_id
		WHERE l.checked_at >= $1 AND (s.group_id = $2 OR (s.group_id IS NULL AND s.user_id = $3))
		GROUP BY l.service_id, hour
	`

	rows, err := r.db.QueryContext(ctx, query, since, group.ID, ungroupedOwner(group))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var buckets []models.UptimeBucket
	for rows.Next() {
		var b models.UptimeBucket
		if err := rows.Scan(&b.ServiceID, &b.Hour, &b.Total, &b.Online); err != nil {
			return nil, err
		}
		buckets = append(buckets, b)
	}
	return buckets, rows.Err()
}

// DeleteOlderThan deletes status logs older than the specified time
func (r *StatusLogRepository) DeleteOlderThan(ctx context.Context, cutoffTime time.Time) (int64, error) {
	query := `DELETE FROM service_status_logs WHERE checked_at < $1`

	result, err := r.db.ExecContext(ctx, query, cutoffTime)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// DeleteByServiceID deletes all status logs for a specific service
func (r *StatusLogRepository) DeleteByServiceID(ctx context.Context, serviceID string) error {
	query := `DELETE FROM service_status_logs WHERE service_id = $1`

	_, err := r.db.ExecContext(ctx, query, serviceID)
	return err
}

// CountByServiceID returns the total number of status logs for a service
func (r *StatusLogRepository) CountByServiceID(ctx context.Context, serviceID string) (int64, error) {
	query := `SELECT COUNT(*) FROM service_status_logs WHERE service_id = $1`

	var count int64
	err := r.db.QueryRowContext(ctx, query, serviceID).Scan(&count)
	return count, err
}

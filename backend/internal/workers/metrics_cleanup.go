package workers

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/nimbus/backend/internal/services"
)

// MetricsCleanupWorker handles periodic cleanup of old status logs
type MetricsCleanupWorker struct {
	metricsService  *services.MetricsService
	retentionDays   int
	cleanupInterval time.Duration
	stopChan        chan struct{}
	cleanupTimer    *time.Timer
}

// NewMetricsCleanupWorker creates a new metrics cleanup worker
func NewMetricsCleanupWorker(metricsService *services.MetricsService) *MetricsCleanupWorker {
	// Get retention days from env (default: 30 days)
	retentionDays := 30
	if days := os.Getenv("METRICS_RETENTION_DAYS"); days != "" {
		if parsed, err := strconv.Atoi(days); err == nil && parsed > 0 {
			retentionDays = parsed
		}
	}

	// Run cleanup once per day
	cleanupInterval := 24 * time.Hour

	return &MetricsCleanupWorker{
		metricsService:  metricsService,
		retentionDays:   retentionDays,
		cleanupInterval: cleanupInterval,
		stopChan:        make(chan struct{}),
	}
}

// Start begins the periodic cleanup process
func (w *MetricsCleanupWorker) Start() {
	log.Printf("Starting metrics cleanup worker (retention: %d days, interval: %s)", w.retentionDays, w.cleanupInterval)

	// Run first cleanup after 1 hour (give system time to settle)
	w.cleanupTimer = time.AfterFunc(1*time.Hour, func() {
		w.runCleanup()
	})

	// Start periodic cleanup
	go w.run()
}

// Stop gracefully stops the worker
func (w *MetricsCleanupWorker) Stop() {
	log.Println("Stopping metrics cleanup worker...")

	// Cancel the initial cleanup timer if it hasn't fired yet
	if w.cleanupTimer != nil {
		w.cleanupTimer.Stop()
	}

	close(w.stopChan)
}

// run is the main worker loop
func (w *MetricsCleanupWorker) run() {
	ticker := time.NewTicker(w.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			w.runCleanup()
		case <-w.stopChan:
			log.Println("Metrics cleanup worker stopped")
			return
		}
	}
}

// runCleanup performs the actual cleanup
func (w *MetricsCleanupWorker) runCleanup() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	log.Printf("Running metrics cleanup (deleting logs older than %d days)...", w.retentionDays)

	deletedCount, err := w.metricsService.CleanupOldLogs(ctx, w.retentionDays)
	if err != nil {
		log.Printf("Error during metrics cleanup: %v", err)
		return
	}

	if deletedCount > 0 {
		log.Printf("Metrics cleanup completed: deleted %d old status log entries", deletedCount)
	} else {
		log.Println("Metrics cleanup completed: no old logs to delete")
	}
}

// RunNow triggers an immediate cleanup (useful for testing or manual triggers)
func (w *MetricsCleanupWorker) RunNow() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	deletedCount, err := w.metricsService.CleanupOldLogs(ctx, w.retentionDays)
	if err != nil {
		return fmt.Errorf("cleanup failed: %w", err)
	}

	log.Printf("Manual cleanup completed: deleted %d old status log entries", deletedCount)
	return nil
}

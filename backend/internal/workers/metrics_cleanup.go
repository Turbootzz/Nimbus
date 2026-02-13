package workers

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/nimbus/backend/internal/handlers"
	"github.com/nimbus/backend/internal/repository"
	"github.com/nimbus/backend/internal/services"
)

// MetricsCleanupWorker handles periodic cleanup of old status logs, webhook logs, and expired tokens
type MetricsCleanupWorker struct {
	metricsService    *services.MetricsService
	webhookRepo       *repository.WebhookRepository
	passwordResetRepo *repository.PasswordResetRepository
	retentionDays     int
	cleanupInterval   time.Duration
	stopChan          chan struct{}
	cleanupTimer      *time.Timer
	stopOnce          sync.Once
}

// NewMetricsCleanupWorker creates a new metrics cleanup worker
func NewMetricsCleanupWorker(metricsService *services.MetricsService, webhookRepo *repository.WebhookRepository, passwordResetRepo *repository.PasswordResetRepository) *MetricsCleanupWorker {
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
		metricsService:    metricsService,
		webhookRepo:       webhookRepo,
		passwordResetRepo: passwordResetRepo,
		retentionDays:     retentionDays,
		cleanupInterval:   cleanupInterval,
		stopChan:          make(chan struct{}),
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

// Stop gracefully stops the worker (safe to call multiple times)
func (w *MetricsCleanupWorker) Stop() {
	w.stopOnce.Do(func() {
		log.Println("Stopping metrics cleanup worker...")

		// Cancel the initial cleanup timer if it hasn't fired yet
		if w.cleanupTimer != nil {
			w.cleanupTimer.Stop()
		}

		close(w.stopChan)
	})
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

	log.Printf("Running cleanup (deleting logs older than %d days)...", w.retentionDays)

	var statusDeleted, webhookDeleted int64
	var statusErr, webhookErr error

	// Clean up status logs
	statusDeleted, statusErr = w.metricsService.CleanupOldLogs(ctx, w.retentionDays)
	if statusErr != nil {
		log.Printf("Error during status logs cleanup: %v", statusErr)
	} else if statusDeleted > 0 {
		log.Printf("Status logs cleanup: deleted %d old entries", statusDeleted)
	}

	// Clean up webhook logs
	if w.webhookRepo != nil {
		cutoff := time.Now().AddDate(0, 0, -w.retentionDays)
		webhookDeleted, webhookErr = w.webhookRepo.DeleteLogsOlderThan(ctx, cutoff)
		if webhookErr != nil {
			log.Printf("Error during webhook logs cleanup: %v", webhookErr)
		} else if webhookDeleted > 0 {
			log.Printf("Webhook logs cleanup: deleted %d old entries", webhookDeleted)
		}
	}

	// Clean up expired password reset tokens
	var tokensDeleted int64
	var tokenErr error
	if w.passwordResetRepo != nil {
		tokensDeleted, tokenErr = w.passwordResetRepo.DeleteExpired(ctx)
		if tokenErr != nil {
			log.Printf("Error during password reset token cleanup: %v", tokenErr)
		} else if tokensDeleted > 0 {
			log.Printf("Password reset token cleanup: deleted %d expired tokens", tokensDeleted)
		}
	}

	// Clean up stale forgot-password rate limit entries
	rateLimitRemoved := handlers.CleanupForgotPasswordRateLimit()
	if rateLimitRemoved > 0 {
		log.Printf("Rate limit cleanup: removed %d stale entries", rateLimitRemoved)
	}

	if statusErr == nil && webhookErr == nil && tokenErr == nil && statusDeleted == 0 && webhookDeleted == 0 && tokensDeleted == 0 && rateLimitRemoved == 0 {
		log.Println("Cleanup completed: no old logs to delete")
	}
}

// RunNow triggers an immediate cleanup (useful for testing or manual triggers)
func (w *MetricsCleanupWorker) RunNow() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	statusDeleted, err := w.metricsService.CleanupOldLogs(ctx, w.retentionDays)
	if err != nil {
		return fmt.Errorf("status logs cleanup failed: %w", err)
	}

	var webhookDeleted int64
	if w.webhookRepo != nil {
		cutoff := time.Now().AddDate(0, 0, -w.retentionDays)
		webhookDeleted, err = w.webhookRepo.DeleteLogsOlderThan(ctx, cutoff)
		if err != nil {
			return fmt.Errorf("webhook logs cleanup failed: %w", err)
		}
	}

	log.Printf("Manual cleanup completed: deleted %d status logs, %d webhook logs", statusDeleted, webhookDeleted)
	return nil
}

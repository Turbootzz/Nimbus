package workers

import (
	"log"
	"sync"
	"time"

	"github.com/nimbus/backend/internal/services"
)

// RateLimitCleanupWorker handles periodic cleanup of expired rate limit cache entries
type RateLimitCleanupWorker struct {
	cleanupInterval time.Duration
	stopChan        chan struct{}
	stopOnce        sync.Once
}

// NewRateLimitCleanupWorker creates a new rate limit cache cleanup worker
func NewRateLimitCleanupWorker() *RateLimitCleanupWorker {
	return &RateLimitCleanupWorker{
		cleanupInterval: 10 * time.Minute, // Run every 10 minutes
		stopChan:        make(chan struct{}),
	}
}

// Start begins the periodic cleanup process
func (w *RateLimitCleanupWorker) Start() {
	log.Printf("Starting rate limit cache cleanup worker (interval: %s)", w.cleanupInterval)
	go w.run()
}

// Stop gracefully stops the worker (safe to call multiple times)
func (w *RateLimitCleanupWorker) Stop() {
	w.stopOnce.Do(func() {
		log.Println("Stopping rate limit cache cleanup worker...")
		close(w.stopChan)
	})
}

// run is the main worker loop
func (w *RateLimitCleanupWorker) run() {
	ticker := time.NewTicker(w.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			removed := services.CleanupRateLimitCache()
			if removed > 0 {
				log.Printf("Rate limit cache cleanup: removed %d expired entries", removed)
			}
		case <-w.stopChan:
			log.Println("Rate limit cache cleanup worker stopped")
			return
		}
	}
}

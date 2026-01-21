package workers

import (
	"log"
	"sync"
	"time"

	"github.com/nimbus/backend/internal/services"
)

// DNSCleanupWorker handles periodic cleanup of expired DNS cache entries
type DNSCleanupWorker struct {
	cleanupInterval time.Duration
	stopChan        chan struct{}
	stopOnce        sync.Once
}

// NewDNSCleanupWorker creates a new DNS cache cleanup worker
func NewDNSCleanupWorker() *DNSCleanupWorker {
	return &DNSCleanupWorker{
		cleanupInterval: 10 * time.Minute, // Run every 10 minutes
		stopChan:        make(chan struct{}),
	}
}

// Start begins the periodic cleanup process
func (w *DNSCleanupWorker) Start() {
	log.Printf("Starting DNS cache cleanup worker (interval: %s)", w.cleanupInterval)
	go w.run()
}

// Stop gracefully stops the worker (safe to call multiple times)
func (w *DNSCleanupWorker) Stop() {
	w.stopOnce.Do(func() {
		log.Println("Stopping DNS cache cleanup worker...")
		close(w.stopChan)
	})
}

// run is the main worker loop
func (w *DNSCleanupWorker) run() {
	ticker := time.NewTicker(w.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			removed := services.CleanupDNSCache()
			if removed > 0 {
				log.Printf("DNS cache cleanup: removed %d expired entries", removed)
			}
		case <-w.stopChan:
			log.Println("DNS cache cleanup worker stopped")
			return
		}
	}
}

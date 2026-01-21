package workers

import (
	"testing"
	"time"

	"github.com/nimbus/backend/internal/services"
)

func TestNewDNSCleanupWorker(t *testing.T) {
	worker := NewDNSCleanupWorker()

	if worker == nil {
		t.Fatal("Expected worker to be created")
	}

	expectedInterval := 10 * time.Minute
	if worker.cleanupInterval != expectedInterval {
		t.Errorf("Expected cleanup interval %v, got %v", expectedInterval, worker.cleanupInterval)
	}

	if worker.stopChan == nil {
		t.Error("Expected stopChan to be initialized")
	}
}

func TestDNSCleanupWorker_StartStop(t *testing.T) {
	worker := NewDNSCleanupWorker()

	// Start the worker
	worker.Start()

	// Give it a moment to start
	time.Sleep(100 * time.Millisecond)

	// Stop the worker
	worker.Stop()

	// Give it a moment to stop gracefully
	time.Sleep(100 * time.Millisecond)

	// If we reached here without hanging, the test passes
}

func TestDNSCleanupWorker_StopMultipleTimes(t *testing.T) {
	worker := NewDNSCleanupWorker()

	// Start the worker
	worker.Start()

	// Give it a moment to start
	time.Sleep(100 * time.Millisecond)

	// Call Stop() multiple times - should not panic due to sync.Once
	worker.Stop()
	worker.Stop()
	worker.Stop()

	// Give it a moment to stop gracefully
	time.Sleep(100 * time.Millisecond)

	// If we reached here without panic, the test passes
}

func TestDNSCleanupWorker_CleanupIntegration(t *testing.T) {
	// This test verifies that CleanupDNSCache is callable from the worker
	// The actual cleanup logic is tested in health_check_test.go

	// Calling cleanup directly should not panic and should return a count
	removed := services.CleanupDNSCache()

	// Should return 0 or more (depending on cache state from other tests)
	if removed < 0 {
		t.Errorf("Expected non-negative removed count, got %d", removed)
	}
}

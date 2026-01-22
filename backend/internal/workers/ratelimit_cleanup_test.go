package workers

import (
	"testing"
	"time"
)

func TestNewRateLimitCleanupWorker(t *testing.T) {
	worker := NewRateLimitCleanupWorker()

	if worker == nil {
		t.Fatal("Expected non-nil worker")
	}

	if worker.cleanupInterval != 10*time.Minute {
		t.Errorf("Expected cleanup interval of 10 minutes, got %v", worker.cleanupInterval)
	}

	if worker.stopChan == nil {
		t.Error("Expected non-nil stop channel")
	}
}

func TestRateLimitCleanupWorker_StartStop(t *testing.T) {
	worker := NewRateLimitCleanupWorker()

	// Start the worker
	worker.Start()

	// Give it a moment to start
	time.Sleep(100 * time.Millisecond)

	// Stop the worker
	worker.Stop()

	// Give it a moment to stop
	time.Sleep(100 * time.Millisecond)

	// Verify we can stop multiple times without panic
	worker.Stop()
}

func TestRateLimitCleanupWorker_StopMultipleTimes(t *testing.T) {
	worker := NewRateLimitCleanupWorker()

	worker.Start()
	time.Sleep(100 * time.Millisecond)

	// Multiple stops should be safe
	worker.Stop()
	worker.Stop()
	worker.Stop()

	time.Sleep(100 * time.Millisecond)
}

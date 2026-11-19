package workers

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type fakeAutoAdder struct {
	calls atomic.Int32
}

func (f *fakeAutoAdder) AutoAdd(context.Context) (int, error) {
	// An error must not stop the worker
	if f.calls.Add(1) == 1 {
		return 0, errors.New("docker is down")
	}
	return 1, nil
}

func TestDiscoveryWorker_RunsUntilStopped(t *testing.T) {
	adder := &fakeAutoAdder{}
	worker := NewDiscoveryWorker(adder)
	worker.interval = 5 * time.Millisecond
	worker.Start()

	assert.Eventually(t, func() bool { return adder.calls.Load() >= 3 }, 2*time.Second, 5*time.Millisecond)
	worker.Stop()
	worker.Stop() // safe to call twice

	calls := adder.calls.Load()
	time.Sleep(30 * time.Millisecond)
	assert.Equal(t, calls, adder.calls.Load(), "no runs after Stop")
}

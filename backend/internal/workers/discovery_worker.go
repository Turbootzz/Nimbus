package workers

import (
	"context"
	"log"
	"sync"
	"time"
)

// AutoAdder adds labelled containers when the admin setting allows it
type AutoAdder interface {
	AutoAdd(ctx context.Context) (int, error)
}

// DiscoveryWorker runs Docker discovery's auto-add every minute. With the
// setting off (the default) a run reads that setting and nothing else.
type DiscoveryWorker struct {
	adder    AutoAdder
	interval time.Duration
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
}

func NewDiscoveryWorker(adder AutoAdder) *DiscoveryWorker {
	ctx, cancel := context.WithCancel(context.Background())
	return &DiscoveryWorker{adder: adder, interval: time.Minute, ctx: ctx, cancel: cancel}
}

// Start begins the periodic runs
func (w *DiscoveryWorker) Start() {
	log.Printf("Starting discovery worker (interval: %s)", w.interval)
	w.wg.Add(1)
	go w.run()
}

// Stop ends the worker and waits for a running pass (safe to call more than once)
func (w *DiscoveryWorker) Stop() {
	w.cancel()
	w.wg.Wait()
	log.Println("Discovery worker stopped")
}

func (w *DiscoveryWorker) run() {
	defer w.wg.Done()
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			added, err := w.adder.AutoAdd(w.ctx)
			if err != nil && w.ctx.Err() == nil {
				log.Printf("Discovery worker: %v", err)
			}
			if added > 0 {
				log.Printf("Discovery worker: added %d labelled container(s)", added)
			}
		case <-w.ctx.Done():
			return
		}
	}
}

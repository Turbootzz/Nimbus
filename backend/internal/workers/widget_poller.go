package workers

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/nimbus/backend/internal/services"
)

const (
	pollerTick           = 5 * time.Second
	pollerReloadInterval = 60 * time.Second
	pollerMaxConcurrent  = 8
	pollerMaxFetchTime   = 15 * time.Second
	// A manual refresh is ignored this soon after the last fetch started
	pollerRefreshCooldown = 30 * time.Second
	// After this many failures in a row a source is polled 5x less often
	pollerBackoffAfter  = 3
	pollerBackoffFactor = 5
)

// LiveDataProvider lists the sources to poll and keeps their results
type LiveDataProvider interface {
	Sources(ctx context.Context) ([]services.LiveSource, error)
	Record(src services.LiveSource, payload any, err error)
	Retain(keys map[string]bool)
}

type pollEntry struct {
	src      services.LiveSource
	nextRun  time.Time
	failures int
	running  bool
	lastRun  time.Time
	// rerun asks for another fetch right after the running one, because
	// the source was edited or refreshed meanwhile
	rerun bool
	// removed marks a source that is gone while its fetch still runs; the
	// entry stays until then, so the source never has two fetches at once
	removed bool
}

// runNow makes the entry due at once, or right after its running fetch
func (e *pollEntry) runNow() {
	if e.running {
		e.rerun = true
	} else {
		e.nextRun = time.Time{}
	}
}

// WidgetPoller fetches every widget and integration on its own interval.
// Each fetch has its own timeout and at most 8 run at once, so one slow app
// never delays the others.
type WidgetPoller struct {
	provider LiveDataProvider
	now      func() time.Time

	mu         sync.Mutex
	entries    map[string]*pollEntry
	lastReload time.Time

	sem    chan struct{}
	kick   chan struct{}
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup // the loop and running fetches
}

func NewWidgetPoller(provider LiveDataProvider) *WidgetPoller {
	ctx, cancel := context.WithCancel(context.Background())
	return &WidgetPoller{
		provider: provider,
		now:      time.Now,
		entries:  map[string]*pollEntry{},
		sem:      make(chan struct{}, pollerMaxConcurrent),
		kick:     make(chan struct{}, 1),
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Start begins polling
func (p *WidgetPoller) Start() {
	p.wg.Add(1)
	go p.run()
	log.Println("Widget poller started")
}

// Stop ends polling and waits for running fetches
func (p *WidgetPoller) Stop() {
	log.Println("Stopping widget poller...")
	p.cancel()
	p.wg.Wait()
	log.Println("Widget poller stopped")
}

// Kick reloads the sources soon, e.g. after a widget was added
func (p *WidgetPoller) Kick() {
	select {
	case p.kick <- struct{}{}:
	default:
	}
}

// Refresh fetches one source as soon as possible. It returns false when
// the source was fetched too recently, so a refresh loop can't flood an app.
func (p *WidgetPoller) Refresh(key string) bool {
	p.mu.Lock()
	entry, ok := p.entries[key]
	if ok && !entry.lastRun.IsZero() && p.now().Sub(entry.lastRun) < pollerRefreshCooldown {
		p.mu.Unlock()
		return false
	}
	if ok {
		entry.runNow()
	}
	p.mu.Unlock()
	p.Kick()
	return true
}

func (p *WidgetPoller) run() {
	defer p.wg.Done()
	ticker := time.NewTicker(pollerTick)
	defer ticker.Stop()

	p.reload()
	p.runDue()
	for {
		select {
		case <-p.ctx.Done():
			return
		case <-p.kick:
			p.reload()
			p.runDue()
		case <-ticker.C:
			if p.now().Sub(p.lastReload) >= pollerReloadInterval {
				p.reload()
			}
			p.runDue()
		}
	}
}

// reload syncs the schedule with the current sources. New and edited
// sources are due at once; removed ones are dropped.
func (p *WidgetPoller) reload() {
	ctx, cancel := context.WithTimeout(p.ctx, 10*time.Second)
	defer cancel()
	sources, err := p.provider.Sources(ctx)
	if err != nil {
		log.Printf("Widget poller: failed to load sources: %v", err)
		return
	}

	keys := make(map[string]bool, len(sources))
	p.mu.Lock()
	p.lastReload = p.now()
	for _, src := range sources {
		key := src.Key()
		keys[key] = true
		entry, ok := p.entries[key]
		if !ok {
			p.entries[key] = &pollEntry{src: src}
			continue
		}
		if entry.removed {
			entry.removed = false
			entry.runNow()
		}
		if entry.src.Version != src.Version {
			entry.runNow()
			entry.failures = 0
		}
		entry.src = src
	}
	for key, entry := range p.entries {
		switch {
		case keys[key]:
		case entry.running:
			entry.removed = true
		default:
			delete(p.entries, key)
		}
	}
	p.mu.Unlock()

	p.provider.Retain(keys)
}

// runDue starts a fetch for every due source that isn't running yet
func (p *WidgetPoller) runDue() {
	p.mu.Lock()
	defer p.mu.Unlock()
	now := p.now()
	for key, entry := range p.entries {
		if entry.running || entry.removed || entry.nextRun.After(now) {
			continue
		}
		entry.running = true
		entry.lastRun = now
		p.wg.Add(1)
		go p.fetch(key, entry)
	}
}

func (p *WidgetPoller) fetch(key string, entry *pollEntry) {
	defer p.wg.Done()
	p.mu.Lock()
	src := entry.src
	p.mu.Unlock()

	select {
	case p.sem <- struct{}{}:
		defer func() { <-p.sem }()
	case <-p.ctx.Done():
		return
	}

	ctx, cancel := context.WithTimeout(p.ctx, min(src.Interval, pollerMaxFetchTime))
	payload, err := src.Fetch(ctx)
	cancel()
	if p.ctx.Err() != nil {
		return // shutting down; don't record a cancelled fetch
	}

	p.mu.Lock()
	if entry.removed {
		delete(p.entries, key) // removed while fetching; drop the result
		p.mu.Unlock()
		return
	}
	p.mu.Unlock()
	p.provider.Record(src, payload, err)

	p.mu.Lock()
	defer p.mu.Unlock()
	entry.running = false
	delay := src.Interval
	if err != nil {
		entry.failures++
		if entry.failures >= pollerBackoffAfter {
			delay *= pollerBackoffFactor
		}
	} else {
		entry.failures = 0
	}
	entry.nextRun = p.now().Add(delay)
	if entry.rerun {
		entry.rerun = false
		entry.nextRun = time.Time{}
		p.Kick()
	}
}

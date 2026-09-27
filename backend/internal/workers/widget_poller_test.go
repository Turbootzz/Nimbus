package workers

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nimbus/backend/internal/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recorded struct {
	key     string
	payload any
	err     error
}

// fakeProvider serves a fixed list of sources and records results
type fakeProvider struct {
	mu       sync.Mutex
	sources  []services.LiveSource
	results  []recorded
	retained map[string]bool
}

func (f *fakeProvider) Sources(context.Context) ([]services.LiveSource, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]services.LiveSource(nil), f.sources...), nil
}

func (f *fakeProvider) Record(src services.LiveSource, payload any, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.results = append(f.results, recorded{src.Key(), payload, err})
}

func (f *fakeProvider) Retain(keys map[string]bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.retained = keys
}

func (f *fakeProvider) setSources(sources ...services.LiveSource) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sources = sources
}

func (f *fakeProvider) count(key string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, r := range f.results {
		if r.key == key {
			n++
		}
	}
	return n
}

// fakeClock is advanced by hand
type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

func newTestPoller(t *testing.T, sources ...services.LiveSource) (*WidgetPoller, *fakeProvider, *fakeClock) {
	t.Helper()
	provider := &fakeProvider{sources: sources}
	clock := &fakeClock{now: time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)}
	p := NewWidgetPoller(provider)
	p.now = clock.Now
	t.Cleanup(p.Stop)
	return p, provider, clock
}

// cycle runs one reload and dispatch and waits for the fetches
func (p *WidgetPoller) cycle(reload bool) {
	if reload {
		p.reload()
	}
	p.runDue()
	p.wg.Wait()
}

func source(id string, interval time.Duration, fetch func(context.Context) (any, error)) services.LiveSource {
	return services.LiveSource{Kind: "widget", ID: id, UserID: "u1", Interval: interval, Version: "v1", Fetch: fetch}
}

func ok(payload any) func(context.Context) (any, error) {
	return func(context.Context) (any, error) { return payload, nil }
}

func TestPoller_FetchesEachSourceOnItsOwnInterval(t *testing.T) {
	fast := source("fast", 30*time.Second, ok("f"))
	slow := source("slow", 5*time.Minute, ok("s"))
	p, provider, clock := newTestPoller(t, fast, slow)

	p.cycle(true)
	assert.Equal(t, 1, provider.count("widget:fast"), "new sources are due at once")
	assert.Equal(t, 1, provider.count("widget:slow"))
	assert.Equal(t, map[string]bool{"widget:fast": true, "widget:slow": true}, provider.retained)

	clock.Advance(10 * time.Second)
	p.cycle(false)
	assert.Equal(t, 1, provider.count("widget:fast"), "not due yet")

	clock.Advance(20 * time.Second)
	p.cycle(false)
	assert.Equal(t, 2, provider.count("widget:fast"))
	assert.Equal(t, 1, provider.count("widget:slow"))
}

func TestPoller_BacksOffAfterThreeFailures(t *testing.T) {
	failing := source("bad", time.Minute, func(context.Context) (any, error) { return nil, errors.New("down") })
	p, provider, clock := newTestPoller(t, failing)

	p.cycle(true)
	for range 2 {
		clock.Advance(time.Minute)
		p.cycle(false)
	}
	assert.Equal(t, 3, provider.count("widget:bad"))

	// Third failure: the next try waits 5 minutes instead of 1
	clock.Advance(time.Minute)
	p.cycle(false)
	assert.Equal(t, 3, provider.count("widget:bad"))
	clock.Advance(4 * time.Minute)
	p.cycle(false)
	assert.Equal(t, 4, provider.count("widget:bad"))
	assert.EqualError(t, provider.results[3].err, "down")
}

func TestPoller_SuccessResetsBackoff(t *testing.T) {
	var fail atomic.Bool
	fail.Store(true)
	flaky := source("flaky", time.Minute, func(context.Context) (any, error) {
		if fail.Load() {
			return nil, errors.New("down")
		}
		return "ok", nil
	})
	p, provider, clock := newTestPoller(t, flaky)

	p.cycle(true)
	clock.Advance(time.Minute)
	p.cycle(false)
	clock.Advance(time.Minute)
	fail.Store(false)
	p.cycle(false) // third attempt succeeds
	clock.Advance(time.Minute)
	p.cycle(false)
	assert.Equal(t, 4, provider.count("widget:flaky"), "back to the normal interval")
}

func TestPoller_EditedSourceIsFetchedAtOnce(t *testing.T) {
	src := source("w", 10*time.Minute, ok(1))
	p, provider, clock := newTestPoller(t, src)
	p.cycle(true)

	clock.Advance(time.Second)
	edited := src
	edited.Version = "v2"
	edited.Fetch = ok(2)
	provider.setSources(edited)
	p.cycle(true)

	assert.Equal(t, 2, provider.count("widget:w"))
	assert.Equal(t, 2, provider.results[1].payload)
}

func TestPoller_RemovedSourceStops(t *testing.T) {
	p, provider, clock := newTestPoller(t, source("a", time.Minute, ok(1)), source("b", time.Minute, ok(1)))
	p.cycle(true)

	provider.setSources(source("b", time.Minute, ok(1)))
	clock.Advance(time.Minute)
	p.cycle(true)
	assert.Equal(t, 1, provider.count("widget:a"))
	assert.Equal(t, 2, provider.count("widget:b"))
	assert.Equal(t, map[string]bool{"widget:b": true}, provider.retained)
}

func TestPoller_RefreshRunsNowAndAfterARunningFetch(t *testing.T) {
	release := make(chan struct{})
	var calls atomic.Int32
	src := source("w", time.Hour, func(context.Context) (any, error) {
		if calls.Add(1) == 2 {
			<-release // second fetch hangs until released
		}
		return "ok", nil
	})
	p, provider, clock := newTestPoller(t, src)
	p.cycle(true)

	assert.False(t, p.Refresh("widget:w"), "just fetched: refresh is ignored")
	p.runDue()
	assert.Equal(t, int32(1), calls.Load())

	clock.Advance(pollerRefreshCooldown)
	assert.True(t, p.Refresh("widget:w"))
	p.runDue() // starts the hanging second fetch
	require.Eventually(t, func() bool { return calls.Load() == 2 }, time.Second, time.Millisecond)

	clock.Advance(pollerRefreshCooldown)
	assert.True(t, p.Refresh("widget:w")) // while running: remembered for later
	p.runDue()
	assert.Equal(t, int32(2), calls.Load(), "no second fetch of the same source at once")

	close(release)
	p.wg.Wait()
	p.cycle(false)
	assert.Equal(t, 3, provider.count("widget:w"), "the refresh ran after the running fetch")
	assert.True(t, p.Refresh("widget:missing"), "unknown keys just kick a reload")
}

func TestPoller_SourceRemovedAndReaddedWhileFetching(t *testing.T) {
	release := make(chan struct{})
	var running, maxRunning atomic.Int32
	src := source("w", time.Minute, func(context.Context) (any, error) {
		n := running.Add(1)
		defer running.Add(-1)
		for {
			old := maxRunning.Load()
			if n <= old || maxRunning.CompareAndSwap(old, n) {
				break
			}
		}
		<-release
		return "ok", nil
	})
	p, provider, _ := newTestPoller(t, src)
	p.reload()
	p.runDue()
	require.Eventually(t, func() bool { return running.Load() == 1 }, time.Second, time.Millisecond)

	provider.setSources() // disabled
	p.reload()
	provider.setSources(src) // enabled again
	p.reload()
	p.runDue()
	assert.Equal(t, int32(1), running.Load(), "never two fetches of one source")

	close(release)
	p.wg.Wait()
	p.cycle(false)
	assert.Equal(t, int32(1), maxRunning.Load())
	assert.Equal(t, 2, provider.count("widget:w"), "fetched again once the first finished")

	// Removed for good while fetching: the result is dropped
	block := make(chan struct{})
	gone := source("gone", time.Minute, func(context.Context) (any, error) { <-block; return "late", nil })
	provider.setSources(gone)
	p.reload()
	p.runDue()
	provider.setSources()
	p.reload()
	close(block)
	p.wg.Wait()
	assert.Equal(t, 0, provider.count("widget:gone"))
	p.mu.Lock()
	assert.Empty(t, p.entries)
	p.mu.Unlock()
}

func TestPoller_SlowSourceDoesNotDelayOthers(t *testing.T) {
	hang := make(chan struct{})
	slow := source("slow", time.Minute, func(ctx context.Context) (any, error) {
		select {
		case <-hang:
		case <-ctx.Done():
		}
		return nil, errors.New("too slow")
	})
	sources := []services.LiveSource{slow}
	for _, id := range []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"} {
		sources = append(sources, source(id, time.Minute, ok(id)))
	}
	p, provider, _ := newTestPoller(t, sources...)

	p.reload()
	p.runDue()
	require.Eventually(t, func() bool { return provider.count("widget:j") == 1 && provider.count("widget:a") == 1 }, 2*time.Second, time.Millisecond)
	assert.Equal(t, 0, provider.count("widget:slow"), "still running")
	close(hang)
	p.wg.Wait()
}

func TestPoller_FetchTimeoutIsPerSource(t *testing.T) {
	var deadline time.Duration
	src := source("w", 10*time.Second, func(ctx context.Context) (any, error) {
		d, _ := ctx.Deadline()
		deadline = time.Until(d)
		return nil, nil
	})
	long := source("long", time.Hour, func(ctx context.Context) (any, error) {
		d, _ := ctx.Deadline()
		if time.Until(d) > pollerMaxFetchTime {
			return nil, errors.New("timeout above the maximum")
		}
		return nil, nil
	})
	p, provider, _ := newTestPoller(t, src, long)
	p.cycle(true)

	assert.LessOrEqual(t, deadline, 10*time.Second, "min(interval, 15s)")
	assert.NoError(t, provider.results[0].err)
	assert.NoError(t, provider.results[1].err)
}

func TestPoller_StopDoesNotRecordCancelledFetches(t *testing.T) {
	started := make(chan struct{})
	src := source("w", time.Minute, func(ctx context.Context) (any, error) {
		close(started)
		<-ctx.Done()
		return nil, ctx.Err()
	})
	provider := &fakeProvider{sources: []services.LiveSource{src}}
	p := NewWidgetPoller(provider)
	p.Start()
	<-started
	p.Stop()
	assert.Equal(t, 0, provider.count("widget:w"))
}

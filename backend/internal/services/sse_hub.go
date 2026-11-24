package services

import (
	"errors"
	"sync"
)

const (
	// Events waiting for a slow browser; newer events are dropped when full
	sseBufferSize = 32
	// Open dashboards (tabs, devices) per user
	maxStreamsPerUser = 20
)

// SSE event names
const (
	EventHello         = "hello"
	EventSnapshot      = "snapshot"
	EventServiceStatus = "service_status"
)

var (
	ErrTooManyStreams = errors.New("too many open dashboard streams")
	ErrHubClosed      = errors.New("stream hub is closed")
)

// Event is one server-sent event
type Event struct {
	Name string
	Data any
}

// Subscriber receives the events of one user until it is unsubscribed
type Subscriber struct {
	userID string // who listens; their open streams are limited
	owner  string // whose events arrive
	events chan Event
}

// Events is closed when the subscriber is removed or the hub closes
func (s *Subscriber) Events() <-chan Event {
	return s.events
}

// SSEHub fans events out to the open dashboard streams of each user.
// Publish never blocks, so a slow browser can't hold up the poller.
type SSEHub struct {
	mu     sync.Mutex
	subs   map[string]map[*Subscriber]struct{} // by the user whose events they get
	open   map[string]int                      // open streams per listening user
	closed bool
}

func NewSSEHub() *SSEHub {
	return &SSEHub{subs: map[string]map[*Subscriber]struct{}{}, open: map[string]int{}}
}

// Subscribe opens a stream for a user, with their own events
func (h *SSEHub) Subscribe(userID string) (*Subscriber, error) {
	return h.SubscribeTo(userID, userID)
}

// SubscribeTo opens a stream for a user with the events of ownerID: the
// owner of a shared board the user views. The stream counts against the
// user's own limit, and the caller must filter the events down to what the
// user may see.
func (h *SSEHub) SubscribeTo(userID, ownerID string) (*Subscriber, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return nil, ErrHubClosed
	}
	if h.open[userID] >= maxStreamsPerUser {
		return nil, ErrTooManyStreams
	}
	sub := &Subscriber{userID: userID, owner: ownerID, events: make(chan Event, sseBufferSize)}
	if h.subs[ownerID] == nil {
		h.subs[ownerID] = map[*Subscriber]struct{}{}
	}
	h.subs[ownerID][sub] = struct{}{}
	h.open[userID]++
	return sub, nil
}

// Unsubscribe closes a stream; calling it twice is fine
func (h *SSEHub) Unsubscribe(sub *Subscriber) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.subs[sub.owner][sub]; !ok {
		return
	}
	delete(h.subs[sub.owner], sub)
	if len(h.subs[sub.owner]) == 0 {
		delete(h.subs, sub.owner)
	}
	if h.open[sub.userID]--; h.open[sub.userID] <= 0 {
		delete(h.open, sub.userID)
	}
	close(sub.events)
}

// Publish sends an event to all streams of a user, dropping it for streams
// whose buffer is full
func (h *SSEHub) Publish(userID string, event Event) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for sub := range h.subs[userID] {
		select {
		case sub.events <- event:
		default:
		}
	}
}

// Close ends all streams, so the server can shut down
func (h *SSEHub) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.closed = true
	for _, subs := range h.subs {
		for sub := range subs {
			close(sub.events)
		}
	}
	h.subs = map[string]map[*Subscriber]struct{}{}
	h.open = map[string]int{}
}

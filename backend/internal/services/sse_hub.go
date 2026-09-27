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
	userID string
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
	subs   map[string]map[*Subscriber]struct{}
	closed bool
}

func NewSSEHub() *SSEHub {
	return &SSEHub{subs: map[string]map[*Subscriber]struct{}{}}
}

// Subscribe opens a stream for a user
func (h *SSEHub) Subscribe(userID string) (*Subscriber, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return nil, ErrHubClosed
	}
	if len(h.subs[userID]) >= maxStreamsPerUser {
		return nil, ErrTooManyStreams
	}
	sub := &Subscriber{userID: userID, events: make(chan Event, sseBufferSize)}
	if h.subs[userID] == nil {
		h.subs[userID] = map[*Subscriber]struct{}{}
	}
	h.subs[userID][sub] = struct{}{}
	return sub, nil
}

// Unsubscribe closes a stream; calling it twice is fine
func (h *SSEHub) Unsubscribe(sub *Subscriber) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.subs[sub.userID][sub]; !ok {
		return
	}
	delete(h.subs[sub.userID], sub)
	if len(h.subs[sub.userID]) == 0 {
		delete(h.subs, sub.userID)
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
}

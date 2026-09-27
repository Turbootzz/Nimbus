package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSSEHub_PublishReachesOnlyThatUser(t *testing.T) {
	hub := NewSSEHub()
	a1, err := hub.Subscribe("a")
	require.NoError(t, err)
	a2, err := hub.Subscribe("a")
	require.NoError(t, err)
	b, err := hub.Subscribe("b")
	require.NoError(t, err)

	hub.Publish("a", Event{Name: EventSnapshot, Data: 1})

	assert.Equal(t, Event{Name: EventSnapshot, Data: 1}, <-a1.Events())
	assert.Equal(t, Event{Name: EventSnapshot, Data: 1}, <-a2.Events())
	assert.Empty(t, b.Events())
}

func TestSSEHub_DropsWhenBufferIsFull(t *testing.T) {
	hub := NewSSEHub()
	sub, err := hub.Subscribe("a")
	require.NoError(t, err)

	for i := range sseBufferSize + 10 {
		hub.Publish("a", Event{Name: EventSnapshot, Data: i}) // never blocks
	}
	assert.Len(t, sub.Events(), sseBufferSize)
	assert.Equal(t, 0, (<-sub.Events()).Data, "oldest events are kept, newest dropped")
}

func TestSSEHub_UnsubscribeClosesOnce(t *testing.T) {
	hub := NewSSEHub()
	sub, err := hub.Subscribe("a")
	require.NoError(t, err)

	hub.Unsubscribe(sub)
	hub.Unsubscribe(sub) // no double close panic
	_, open := <-sub.Events()
	assert.False(t, open)
	hub.Publish("a", Event{Name: EventSnapshot}) // no send on closed channel
}

func TestSSEHub_LimitPerUser(t *testing.T) {
	hub := NewSSEHub()
	for range maxStreamsPerUser {
		_, err := hub.Subscribe("a")
		require.NoError(t, err)
	}
	_, err := hub.Subscribe("a")
	assert.ErrorIs(t, err, ErrTooManyStreams)

	_, err = hub.Subscribe("b")
	assert.NoError(t, err, "limit is per user")
}

func TestSSEHub_CloseEndsAllStreams(t *testing.T) {
	hub := NewSSEHub()
	sub, err := hub.Subscribe("a")
	require.NoError(t, err)

	hub.Close()
	_, open := <-sub.Events()
	assert.False(t, open)
	hub.Unsubscribe(sub) // after Close is fine too

	_, err = hub.Subscribe("a")
	assert.ErrorIs(t, err, ErrHubClosed)
}

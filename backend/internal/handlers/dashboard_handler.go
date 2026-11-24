package handlers

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/services"
)

// Comments keep the stream alive; nginx closes idle proxied reads after 60s
const ssePingInterval = 25 * time.Second

type DashboardHandler struct {
	live         *services.LiveDataService
	hub          *services.SSEHub
	pingInterval time.Duration
	boards       *services.BoardService // nil: one dashboard per user
}

// SetBoards lets Data and Stream take ?board=, for a shared board
func (h *DashboardHandler) SetBoards(boards *services.BoardService) {
	h.boards = boards
}

// How long a viewer's stream may wait for the database when it checks what
// the viewer may still see
const viewerCheckTimeout = 5 * time.Second

// viewerEvent returns the event as a viewer of a shared board gets it, and
// false when it is about something that is not on the board (keys)
func viewerEvent(event services.Event, keys map[string]bool) (services.Event, bool) {
	switch data := event.Data.(type) {
	case models.Snapshot:
		event.Data = services.ForViewer(data)
		return event, keys[data.Key()]
	case services.ServiceStatusEvent:
		return event, keys[services.ServiceKey(data.ID)]
	}
	return event, false
}

func NewDashboardHandler(live *services.LiveDataService, hub *services.SSEHub) *DashboardHandler {
	return &DashboardHandler{live: live, hub: hub, pingInterval: ssePingInterval}
}

// Data returns the latest snapshot of every polled widget and integration,
// for the first render before the stream takes over. With ?board=<id> of a
// board someone else shares: only what that board shows.
func (h *DashboardHandler) Data(c *fiber.Ctx) error {
	userID, err := RequireUserID(c)
	if err != nil {
		return err
	}
	board, err := requestBoard(c, h.boards, userID)
	if err != nil {
		return serviceError(c, err, "load dashboard data", boardNotFound)
	}
	// The owner gets everything they have: the status strip shows
	// integrations that are on no board
	if board.CanEdit {
		return c.JSON(h.live.Snapshots(userID))
	}
	keys, err := h.boards.ViewerKeys(c.Context(), board.Board.ID)
	if err != nil {
		return serviceError(c, err, "load dashboard data", boardNotFound)
	}
	return c.JSON(h.live.ViewerSnapshots(board.Board.UserID, keys))
}

// Stream pushes snapshot and service_status events as server-sent events:
// the user's own, or with ?board=<id> of a board someone else shares, those
// of the tiles on that board.
func (h *DashboardHandler) Stream(c *fiber.Ctx) error {
	userID, err := RequireUserID(c)
	if err != nil {
		return err
	}
	board, err := requestBoard(c, h.boards, userID)
	if err != nil {
		return serviceError(c, err, "open dashboard stream", boardNotFound)
	}
	// A viewer listens to the owner's events, filtered to the board (keys)
	ownerID, viewer, boardID := userID, !board.CanEdit, board.ID()
	var keys map[string]bool
	if viewer {
		ownerID = board.Board.UserID
		if keys, err = h.boards.ViewerKeys(c.Context(), boardID); err != nil {
			return serviceError(c, err, "open dashboard stream", boardNotFound)
		}
	}
	sub, err := h.hub.SubscribeTo(userID, ownerID)
	if errors.Is(err, services.ErrTooManyStreams) {
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{"error": "Too many open dashboards"})
	}
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Server is shutting down"})
	}

	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("X-Accel-Buffering", "no") // tell nginx not to buffer

	// The writer runs after this handler returns, so it must not touch c
	ping := h.pingInterval
	c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		defer h.hub.Unsubscribe(sub)
		ticker := time.NewTicker(ping)
		defer ticker.Stop()

		if writeSSE(w, services.EventHello, fiber.Map{}) != nil {
			return
		}
		for {
			select {
			case event, ok := <-sub.Events():
				if !ok {
					return // hub closed
				}
				if viewer {
					if event, ok = viewerEvent(event, keys); !ok {
						continue
					}
				}
				if writeSSE(w, event.Name, event.Data) != nil {
					return
				}
			case <-ticker.C:
				// The board may be unshared, or its tiles moved or added,
				// while the stream is open: look again at every ping, and
				// end the stream when the viewer lost the board
				if viewer {
					if keys = h.viewerKeys(userID, ownerID, boardID); keys == nil {
						return
					}
				}
				// A failed flush means the browser went away
				if _, err := w.WriteString(": ping\n\n"); err != nil {
					return
				}
				if w.Flush() != nil {
					return
				}
			}
		}
	})
	return nil
}

// viewerKeys returns what the user may follow on someone's shared board
// now, and nil when they may not see the board anymore
func (h *DashboardHandler) viewerKeys(userID, ownerID, boardID string) map[string]bool {
	ctx, cancel := context.WithTimeout(context.Background(), viewerCheckTimeout)
	defer cancel()
	board, err := h.boards.Resolve(ctx, userID, boardID)
	if err != nil || board.CanEdit || board.Board.UserID != ownerID {
		return nil
	}
	keys, err := h.boards.ViewerKeys(ctx, boardID)
	if err != nil {
		log.Printf("Failed to check a shared board for its viewer: %v", err)
		return nil
	}
	return keys
}

// writeSSE writes one event and flushes it
func writeSSE(w *bufio.Writer, name string, data any) error {
	raw, err := json.Marshal(data)
	if err != nil {
		log.Printf("Failed to encode %s event: %v", name, err)
		return nil // skip the event, keep the stream
	}
	if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", name, raw); err != nil {
		return err
	}
	return w.Flush()
}

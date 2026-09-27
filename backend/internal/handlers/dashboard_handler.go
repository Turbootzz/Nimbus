package handlers

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/nimbus/backend/internal/services"
)

// Comments keep the stream alive; nginx closes idle proxied reads after 60s
const ssePingInterval = 25 * time.Second

type DashboardHandler struct {
	live         *services.LiveDataService
	hub          *services.SSEHub
	pingInterval time.Duration
}

func NewDashboardHandler(live *services.LiveDataService, hub *services.SSEHub) *DashboardHandler {
	return &DashboardHandler{live: live, hub: hub, pingInterval: ssePingInterval}
}

// Data returns the latest snapshot of every polled widget and integration,
// for the first render before the stream takes over
func (h *DashboardHandler) Data(c *fiber.Ctx) error {
	userID, err := RequireUserID(c)
	if err != nil {
		return err
	}
	return c.JSON(h.live.Snapshots(userID))
}

// Stream pushes snapshot and service_status events as server-sent events
func (h *DashboardHandler) Stream(c *fiber.Ctx) error {
	userID, err := RequireUserID(c)
	if err != nil {
		return err
	}
	sub, err := h.hub.Subscribe(userID)
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
				if writeSSE(w, event.Name, event.Data) != nil {
					return
				}
			case <-ticker.C:
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

package models

import (
	"encoding/json"
	"time"
)

// Snapshot source kinds
const (
	SnapshotSourceWidget      = "widget"
	SnapshotSourceIntegration = "integration"
)

// Snapshot is the latest data of one polled widget or integration.
// Payload is the last good result; Error is set when the latest fetch
// failed. Stale means the payload is older than the source's interval.
type Snapshot struct {
	SourceKind string          `json:"source_kind"`
	SourceID   string          `json:"source_id"`
	UserID     string          `json:"-"`
	Payload    json.RawMessage `json:"payload"`
	Error      string          `json:"error,omitempty"`
	FetchedAt  time.Time       `json:"fetched_at"`
	Stale      bool            `json:"stale"`
}

// Key identifies the source, e.g. "widget:<id>"
func (s *Snapshot) Key() string {
	return SnapshotKey(s.SourceKind, s.SourceID)
}

// SnapshotKey builds the key of a source
func SnapshotKey(kind, id string) string {
	return kind + ":" + id
}

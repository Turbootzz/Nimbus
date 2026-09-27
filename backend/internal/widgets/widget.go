// Package widgets is the framework for dashboard widgets (clock, notes,
// weather, ...). Each type lives in its own file and registers itself in
// init(), so adding a type never touches the router.
package widgets

import (
	"bytes"
	"encoding/json"
	"errors"

	"github.com/nimbus/backend/internal/models"
)

// Categories group types in the add-widget picker
const (
	CategoryGeneral = "general"
)

// WidgetType is implemented once per widget type
type WidgetType interface {
	// Type is the stable id stored in the database (e.g. "clock").
	Type() string
	// Meta describes the type for the UI.
	Meta() Meta
	// Validate checks a config and returns it normalised: defaults filled
	// in, unknown fields dropped. Error messages are shown to the user.
	Validate(config json.RawMessage) (json.RawMessage, error)
}

// Meta describes a widget type. Type is filled in by the registry.
type Meta struct {
	Type         string   `json:"type"`
	Name         string   `json:"name"`
	Category     string   `json:"category"`
	DefaultSize  string   `json:"default_size"`
	AllowedSizes []string `json:"allowed_sizes"`
	// Static widgets render in the browser and are never polled
	Static bool `json:"static"`
	// IntegrationKinds lists the integration kinds the widget can use.
	// Empty means the widget takes no integration.
	IntegrationKinds []string `json:"integration_kinds,omitempty"`
	// MinRefreshSeconds overrides the global minimum (e.g. rate limits)
	MinRefreshSeconds int `json:"min_refresh_seconds,omitempty"`
}

// allSizes is the AllowedSizes of types that fit any card
var allSizes = []string{models.CardSize1x1, models.CardSize2x1, models.CardSize2x2}

var errNotObject = errors.New("config must be a JSON object")

// decodeConfig parses a config object into T
func decodeConfig[T any](config json.RawMessage) (T, error) {
	var cfg T
	if !bytes.HasPrefix(bytes.TrimSpace(config), []byte("{")) {
		return cfg, errNotObject
	}
	if err := json.Unmarshal(config, &cfg); err != nil {
		return cfg, errNotObject
	}
	return cfg, nil
}

// encodeConfig returns the normalised config
func encodeConfig(cfg any) (json.RawMessage, error) {
	return json.Marshal(cfg)
}

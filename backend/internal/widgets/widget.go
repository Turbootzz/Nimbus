// Package widgets is the framework for dashboard widgets (clock, notes,
// weather, ...). Each type lives in its own file and registers itself in
// init(), so adding a type never touches the router.
package widgets

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/nimbus/backend/internal/integrations"
	"github.com/nimbus/backend/internal/models"
)

// Categories group types in the add-widget picker
const (
	CategoryGeneral = "general"
	CategoryInfo    = "info"
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

// Fetcher is implemented by types whose data the backend polls. Types
// without it are static and render in the browser only.
type Fetcher interface {
	// Fetch returns the widget's current data; it is sent to the browser
	// as JSON. Errors are shown to the user.
	Fetch(ctx context.Context, req *FetchRequest) (any, error)
}

// SecretConfig is implemented by types whose config holds secrets. The API
// never returns them: Redact masks them, and Unredact puts the stored values
// back when a client sends a masked value unchanged.
type SecretConfig interface {
	Redact(config json.RawMessage) json.RawMessage
	Unredact(config, stored json.RawMessage) json.RawMessage
}

// ConfigIntegrations is implemented by types whose config lists several
// integrations (of Meta.ConfigIntegrationKinds), like a calendar of Sonarr
// and Radarr. The service checks they are the user's; Fetch gets them in
// FetchRequest.Linked.
type ConfigIntegrations interface {
	IntegrationIDs(config json.RawMessage) []string
	// SetIntegrationIDs returns the config with another list, e.g. without
	// integrations that were deleted
	SetIntegrationIDs(config json.RawMessage, ids []string) (json.RawMessage, error)
}

// recoverInto turns a panic into *err. Goroutines a Fetch starts need it:
// the recover around Fetch can't see them, and a panic would stop the server.
func recoverInto(err *error) {
	if r := recover(); r != nil {
		*err = fmt.Errorf("crashed: %v", r)
	}
}

// LinkedIntegration is one integration a config lists, connected
type LinkedIntegration struct {
	ID   string
	Kind string
	Name string
	Conn *integrations.Conn
}

// FetchRequest is everything a Fetch needs
type FetchRequest struct {
	Config json.RawMessage // normalised by Validate
	// Client is SSRF-safe; always use it
	Client *http.Client
	// Integration is the linked app, nil when the widget has none
	Integration *integrations.Conn
	// Linked are the integrations the config lists (ConfigIntegrations);
	// one that is gone or can't connect is left out of the list
	Linked []LinkedIntegration
	// Unlinked explains the integrations left out of Linked
	Unlinked []string
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
	// DefaultRefreshSeconds overrides the default for new widgets, e.g. for
	// numbers that change by the second
	DefaultRefreshSeconds int `json:"default_refresh_seconds,omitempty"`
	// ConfigIntegrationKinds are the kinds a ConfigIntegrations type lists
	ConfigIntegrationKinds []string `json:"config_integration_kinds,omitempty"`
	// AdminOnly types can only be added and configured by admins, e.g.
	// because they show what the server can read on the LAN
	AdminOnly bool `json:"admin_only,omitempty"`
}

// Common AllowedSizes. Content always fits its card (it scrolls inside), so
// these only rule out sizes that make no sense for a type.
var (
	// serviceSizes are the three sizes services have too
	serviceSizes = []string{models.CardSize1x1, models.CardSize2x1, models.CardSize2x2}
	// allSizes adds the narrow and tall 1x2, for lists and text
	allSizes = []string{models.CardSize1x1, models.CardSize2x1, models.CardSize1x2, models.CardSize2x2}
)

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

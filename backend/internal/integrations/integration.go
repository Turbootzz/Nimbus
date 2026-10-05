// Package integrations is the framework for connections to external apps
// (Sonarr, Proxmox, ...). Each kind lives in its own file and registers
// itself in init(), so adding a kind never touches the router.
package integrations

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"

	"github.com/nimbus/backend/internal/models"
)

// Integration is implemented once per kind.
type Integration interface {
	// Kind is the stable id stored in the database (e.g. "sonarr").
	Kind() string
	// Meta describes the kind for the UI.
	Meta() Meta
	// Test checks that the app is reachable and the credentials work.
	Test(ctx context.Context, conn *Conn) error
	// Fetch returns current data for KPI rows and widgets.
	Fetch(ctx context.Context, conn *Conn) (*Payload, error)
}

// Meta describes a kind for the integration form and KPI rows.
// Kind is filled in by the registry.
type Meta struct {
	Kind        string   `json:"kind"`
	Name        string   `json:"name"`
	Icon        string   `json:"icon,omitempty"` // dashboard-icons slug
	DefaultPort int      `json:"default_port,omitempty"`
	AuthTypes   []string `json:"auth_types"` // first one is the default
	KPIs        []KPI    `json:"kpis,omitempty"`
	// URLHint replaces the example address in the form
	URLHint string `json:"url_hint,omitempty"`
	// AdminOnly kinds can only be set up by admins, e.g. because they
	// expose the host
	AdminOnly bool `json:"admin_only,omitempty"`
	// UnixSocket kinds also take unix:///path as the URL, for the socket
	// the server allows in DOCKER_SOCKET
	UnixSocket bool `json:"unix_socket,omitempty"`
}

// KPI describes one value a kind can show on a service tile.
type KPI struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Unit  string `json:"unit,omitempty"`
}

// Payload is the result of one Fetch. KPIs maps KPI.Key to its value; Data
// holds kind-specific detail for widgets (e.g. a container list).
type Payload struct {
	KPIs map[string]any `json:"kpis,omitempty"`
	Data any            `json:"data,omitempty"`
}

// Conn is everything a kind needs to talk to one configured app.
// Secrets belong in Creds, never in Options.
type Conn struct {
	BaseURL   string // no trailing slash
	Creds     models.IntegrationCredentials
	VerifyTLS bool
	Options   json.RawMessage
	// Client is SSRF-safe and already honours VerifyTLS; always use it.
	Client *http.Client
	// State keeps session data (e.g. a login sid) between calls.
	State *State
}

// State is a small concurrency-safe store for per-integration session data.
// The zero value is ready to use.
type State struct {
	mu     sync.Mutex
	values map[string]string
}

// Get returns a stored value
func (s *State) Get(key string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.values[key]
	return v, ok
}

// Set stores a value
func (s *State) Set(key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.values == nil {
		s.values = make(map[string]string)
	}
	s.values[key] = value
}

// Delete removes a value
func (s *State) Delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.values, key)
}

// Package integrations is the framework for connections to external apps
// (Sonarr, Proxmox, ...). Each kind lives in its own file and registers
// itself in init(), so adding a kind never touches the router.
package integrations

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"
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

// Closer is implemented by kinds that keep a session open on the app
// (Pi-hole has few session seats). Close ends it when the integration is
// deleted.
type Closer interface {
	Close(ctx context.Context, conn *Conn) error
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
	// Options are the kind's own settings, shown in the form in this order
	Options []Option `json:"options,omitempty"`
	// URLHint replaces the example address in the form
	URLHint string `json:"url_hint,omitempty"`
	// Hint is extra help shown under the URL in the form
	Hint string `json:"hint,omitempty"`
	// AdminOnly kinds can only be set up by admins, e.g. because they
	// expose the host
	AdminOnly bool `json:"admin_only,omitempty"`
	// DockerSocket kinds also take unix:///path as the URL, for the
	// socket the server allows in DOCKER_SOCKET
	DockerSocket bool `json:"docker_socket,omitempty"`
}

// KPI describes one value a kind can show on a service tile.
type KPI struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Unit  string `json:"unit,omitempty"`
	// Option names the option that must be set for this KPI to apply
	Option string `json:"option,omitempty"`
}

// Option describes one setting of a kind (a text field in the form). The
// service checks submitted options against it; kinds read them with
// Conn.Option. Never use options for secrets.
type Option struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Help        string `json:"help,omitempty"`
	Placeholder string `json:"placeholder,omitempty"`
	Required    bool   `json:"required,omitempty"`
	// MaxLength in characters; 0 means DefaultOptionMaxLength
	MaxLength int `json:"max_length,omitempty"`
	// Pattern is a regular expression the whole value must match. Keep it
	// simple: the browser checks it too.
	Pattern string `json:"pattern,omitempty"`
}

// DefaultOptionMaxLength caps options that set no MaxLength
const DefaultOptionMaxLength = 200

// MaxLen is the longest value the option takes, in characters
func (o Option) MaxLen() int {
	if o.MaxLength > 0 {
		return o.MaxLength
	}
	return DefaultOptionMaxLength
}

// Matches reports whether value fits the option's pattern
func (o Option) Matches(value string) bool {
	if o.Pattern == "" {
		return true
	}
	// Register compiled it once, so it is valid
	return regexp.MustCompile(`^(?:` + o.Pattern + `)$`).MatchString(value)
}

// Payload is the result of one Fetch. KPIs maps KPI.Key to its value; Data
// holds kind-specific detail for widgets (e.g. a container list).
type Payload struct {
	KPIs map[string]any `json:"kpis,omitempty"`
	// KPIDefs replaces Meta().KPIs on this integration's tile when the
	// KPIs to show, or their labels, depend on its options or live data
	KPIDefs []KPI `json:"kpi_defs,omitempty"`
	Data    any   `json:"data,omitempty"`
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

// Option returns a text option, "" when it is not set
func (c *Conn) Option(key string) string {
	var options map[string]any
	// Options were validated when saved; anything else reads as not set
	_ = json.Unmarshal(c.Options, &options)
	value, _ := options[key].(string)
	return value
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

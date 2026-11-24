package models

// SettingDiscoveryAutoAdd is the system setting that lets Nimbus add
// containers labelled nimbus.enable=true on its own
const SettingDiscoveryAutoAdd = "discovery_auto_add"

// DiscoveredContainer is a container that can become a service. It carries
// only values parsed from the nimbus.* and homepage.* labels, never the
// labels themselves.
type DiscoveredContainer struct {
	// Key identifies the container across re-creates: "<integration id>/<container name>"
	Key             string `json:"key"`
	IntegrationID   string `json:"integration_id"`
	IntegrationName string `json:"integration_name"`
	Container       string `json:"container"`
	Image           string `json:"image"`
	State           string `json:"state"`
	// Enabled reports the label nimbus.enable=true
	Enabled bool   `json:"enabled"`
	Name    string `json:"name"`
	URL     string `json:"url"`
	// URLGuessed is true when the URL comes from a published port instead
	// of a label
	URLGuessed  bool   `json:"url_guessed"`
	IconURL     string `json:"icon_url"`
	Group       string `json:"group"`
	Description string `json:"description"`
	// Integration is the connection the nimbus.widget label asks for
	Integration *DiscoveredIntegration `json:"integration,omitempty"`
	Ignored     bool                   `json:"ignored"`
}

// DiscoveredIntegration suggests an integration to set up. It has no
// credentials: the user adds those in the integration form.
type DiscoveredIntegration struct {
	Kind    string `json:"kind"`
	Name    string `json:"name"`
	BaseURL string `json:"base_url"`
}

// DiscoveryError is a Docker integration that could not be read
type DiscoveryError struct {
	IntegrationID   string `json:"integration_id"`
	IntegrationName string `json:"integration_name"`
	Error           string `json:"error"`
}

// DiscoveryResult is the response of GET /discovery/docker
type DiscoveryResult struct {
	// Integrations is the number of Docker integrations that were read
	Integrations int                   `json:"integrations"`
	Containers   []DiscoveredContainer `json:"containers"`
	Errors       []DiscoveryError      `json:"errors"`
}

// DiscoveryAddRequest turns a discovered container into a service
type DiscoveryAddRequest struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	URL         string `json:"url"`
	IconURL     string `json:"icon_url"`
	Group       string `json:"group"` // by name; created when missing
	Description string `json:"description"`
}

// DiscoveryKeyRequest names a discovered container
type DiscoveryKeyRequest struct {
	Key string `json:"key"`
}

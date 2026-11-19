// Package dashdoc defines the dashboard document: the JSON shape a dashboard
// is exported to and imported from (docs/CONFIGURATION.md). It is pure data
// plus structural validation, so importers can build a Document from another
// tool's config without touching the database.
//
// The entry points that read and write a user's dashboard are
// services.DashboardTransferService.Export, Preview and Apply.
package dashdoc

import "encoding/json"

// Version is the document version this server reads and writes
const Version = 1

// Limits of one document. Integrations and widgets match the per-user limits.
const (
	MaxBoards       = 20
	MaxBoardNameLen = 50
	MaxGroups       = 100
	MaxServices     = 500
	MaxWidgets      = 200
	MaxIntegrations = 50
	MaxPosition     = 1_000_000
	MaxRefLen       = 100
)

// Mode is how a document is applied to the dashboard a user already has
type Mode string

const (
	// ModeMerge adds the document's tiles after the existing ones and
	// leaves the preferences alone
	ModeMerge Mode = "merge"
	// ModeReplace deletes the user's groups, services, widgets and
	// integrations first, and applies the document's preferences
	ModeReplace Mode = "replace"
	// ModeSync makes the dashboard match the document while keeping the rows
	// they share (services.DashboardTransferService.Sync). It is how a config
	// file is applied and can't be asked for over HTTP.
	ModeSync Mode = "sync"
)

// ParseMode reads a mode from user input
func ParseMode(s string) (Mode, bool) {
	switch Mode(s) {
	case ModeMerge, ModeReplace:
		return Mode(s), true
	}
	return "", false
}

// Document is a whole dashboard. Entities point at each other with refs:
// strings that only mean something inside the document, never database IDs.
// A ref is only required on an entity that something points at.
//
// Groups, services and widgets are on the user's default board unless their
// board_ref names one of Boards. A document works on those boards only: the
// default board and the ones it lists.
type Document struct {
	Version int `json:"version"`
	// Boards are the user's other boards; a dashboard with one board has none
	Boards       []Board       `json:"boards,omitempty"`
	Groups       []Group       `json:"groups"`
	Integrations []Integration `json:"integrations"`
	Services     []Service     `json:"services"`
	Widgets      []Widget      `json:"widgets"`
	Preferences  *Preferences  `json:"preferences,omitempty"`
	// OmittedImages counts the uploaded images an export left out because
	// they were missing or too large. Ignored on import.
	OmittedImages int `json:"omitted_images,omitempty"`
}

// Board is a board next to the default one. An import finds the board the
// user already has by its slug, and creates it otherwise.
type Board struct {
	Ref  string `json:"ref"`
	Name string `json:"name"`
	// Slug is the URL name (/board/<slug>); omitted, it follows the name
	Slug string `json:"slug,omitempty"`
	// Shared shows the board to every user. Only an admin's import shares
	// a board; anyone else gets a private one.
	Shared bool `json:"shared,omitempty"`
}

// Group is a service group. Groups are ordered as listed.
type Group struct {
	Ref   string `json:"ref,omitempty"`
	Name  string `json:"name"`
	Color string `json:"color,omitempty"`
	// BoardRef is the board the group is on; omitted means the default board
	BoardRef string `json:"board_ref,omitempty"`
	// Default marks the default group of its board; a merge maps it onto
	// the default group that board already has
	Default           bool  `json:"default,omitempty"`
	MonitoringEnabled *bool `json:"monitoring_enabled,omitempty"`
	// TileStyle is "cards" or "chips"; omitted means cards
	TileStyle string `json:"tile_style,omitempty"`
}

// Integration is a connection to an app, without its credentials
type Integration struct {
	Ref            string          `json:"ref,omitempty"`
	Kind           string          `json:"kind"`
	Name           string          `json:"name"`
	BaseURL        string          `json:"base_url"`
	AuthType       string          `json:"auth_type,omitempty"`
	VerifyTLS      *bool           `json:"verify_tls,omitempty"`
	Options        json.RawMessage `json:"options,omitempty"`
	RefreshSeconds *int            `json:"refresh_seconds,omitempty"`
	// NeedsCredentials tells the reader that credentials must be entered
	// after an import. Informational: an import derives it from auth_type.
	NeedsCredentials bool `json:"needs_credentials,omitempty"`
	// Credentials are only read from a config file (NIMBUS_CONFIG_FILE): an
	// import over HTTP refuses them and an export never writes them
	Credentials *Credentials `json:"credentials,omitempty"`
}

// Credentials are the secrets of an integration; which fields are used
// depends on its auth type
type Credentials struct {
	APIKey   string `json:"api_key,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Token    string `json:"token,omitempty"`
}

// Service is a link tile. Its icon is an emoji (Icon), an image URL
// (IconURL) or an uploaded image embedded as base64 (IconData).
type Service struct {
	Ref               string `json:"ref,omitempty"`
	Name              string `json:"name"`
	URL               string `json:"url"`
	Description       string `json:"description,omitempty"`
	Icon              string `json:"icon,omitempty"`
	IconType          string `json:"icon_type,omitempty"`
	IconURL           string `json:"icon_url,omitempty"`
	IconData          string `json:"icon_data,omitempty"`
	CardSize          string `json:"card_size,omitempty"`
	Position          int    `json:"position"`
	GroupRef          string `json:"group_ref,omitempty"`
	IntegrationRef    string `json:"integration_ref,omitempty"`
	MonitoringEnabled *bool  `json:"monitoring_enabled,omitempty"`
	// BoardRef is the board the tile is on, the same as its group's;
	// omitted means the default board
	BoardRef string `json:"board_ref,omitempty"`
}

// Widget is a tile that is not a link. Config is type-specific; where a
// config lists integrations (the calendar), it lists their refs.
type Widget struct {
	Ref            string          `json:"ref,omitempty"`
	Type           string          `json:"type"`
	Title          string          `json:"title,omitempty"`
	Config         json.RawMessage `json:"config,omitempty"`
	CardSize       string          `json:"card_size,omitempty"`
	Position       int             `json:"position"`
	RefreshSeconds *int            `json:"refresh_seconds,omitempty"`
	Enabled        *bool           `json:"enabled,omitempty"`
	GroupRef       string          `json:"group_ref,omitempty"`
	IntegrationRef string          `json:"integration_ref,omitempty"`
	// BoardRef is the board the tile is on, the same as its group's;
	// omitted means the default board
	BoardRef string `json:"board_ref,omitempty"`
	// NeedsSecrets tells the reader that the config holds masked secrets
	// (header values, private feed URLs) that must be entered again.
	// Informational: an import looks at the config itself.
	NeedsSecrets bool `json:"needs_secrets,omitempty"`
}

// Preferences are the user's look and layout settings. An omitted field
// keeps the value the user has, except GroupDisplay. They are only applied
// by ModeReplace.
type Preferences struct {
	ThemeMode *string `json:"theme_mode,omitempty"`
	// ThemeBackground is an image URL; "" clears the background
	ThemeBackground *string `json:"theme_background,omitempty"`
	// WallpaperData is an uploaded background image embedded as base64
	WallpaperData string `json:"wallpaper_data,omitempty"`
	// ThemeAccentColor is #RRGGBB; "" goes back to the default accent
	ThemeAccentColor      *string `json:"theme_accent_color,omitempty"`
	OpenInNewTab          *bool   `json:"open_in_new_tab,omitempty"`
	EnableCardResizing    *bool   `json:"enable_card_resizing,omitempty"`
	EnableServiceGrouping *bool   `json:"enable_service_grouping,omitempty"`
	CardScale             *string `json:"card_scale,omitempty"`
	ViewMode              *string `json:"view_mode,omitempty"`
	WallpaperBlur         *int    `json:"wallpaper_blur,omitempty"`
	WallpaperDim          *int    `json:"wallpaper_dim,omitempty"`
	CardOpacity           *int    `json:"card_opacity,omitempty"`
	CardBlur              *int    `json:"card_blur,omitempty"`
	LayoutMode            *string `json:"layout_mode,omitempty"`
	// GroupDisplay is "tabs" or "sections". Omitted follows the layout
	// (tabs in classic, sections in canvas) instead of keeping the value
	// the user has: that is a setting of its own, and an export has to
	// carry it.
	GroupDisplay *string      `json:"group_display,omitempty"`
	StatusStrip  *StatusStrip `json:"status_strip,omitempty"`
	// The theme settings of docs/THEMING.md
	ThemePreset *string `json:"theme_preset,omitempty"`
	ThemeFont   *string `json:"theme_font,omitempty"`
	KpiMono     *bool   `json:"kpi_mono,omitempty"`
	IconSource  *string `json:"icon_source,omitempty"`
	// CustomCSS is the user's own CSS, checked on import like a normal
	// save (utils.ValidateCustomCSS); "" clears it
	CustomCSS *string `json:"custom_css,omitempty"`
}

// StatusStrip is the row of KPI chips at the top of the dashboard
type StatusStrip struct {
	Enabled bool   `json:"enabled"`
	Chips   []Chip `json:"chips"`
}

// Chip sources
const (
	ChipSourceIntegration = "integration"
	ChipSourceWidget      = "widget"
)

// Chip shows one KPI of the integration or widget that Ref points at
type Chip struct {
	Source string `json:"source"`
	Ref    string `json:"ref"`
	KPI    string `json:"kpi"`
}

// Count is what an import does to one kind of entity
type Count struct {
	Created int `json:"created"`
	// Removed is how many existing rows a replace or sync deletes
	Removed int `json:"removed"`
	// Updated is how many existing rows a sync changes
	Updated int `json:"updated,omitempty"`
}

// Summary is the result of an import, or of its dry run
type Summary struct {
	Mode         Mode  `json:"mode"`
	DryRun       bool  `json:"dry_run"`
	Groups       Count `json:"groups"`
	Services     Count `json:"services"`
	Widgets      Count `json:"widgets"`
	Integrations Count `json:"integrations"`
	// Preferences is true when the document's preferences are applied
	Preferences bool `json:"preferences"`
	// NeedsCredentials counts imported integrations that wait for credentials
	NeedsCredentials int `json:"needs_credentials"`
	// NeedsSecrets counts imported widgets whose secrets must be entered again
	NeedsSecrets int `json:"needs_secrets"`
	// CustomCSS is the custom CSS the import applies, as the server read
	// it, so the user can read it before a file of someone else styles
	// their pages
	CustomCSS string `json:"custom_css,omitempty"`
	// StatusPage is set by a replace while the public status page shows one
	// of the user's groups, which the replace removes
	StatusPage *StatusPageImpact `json:"status_page,omitempty"`
}

// StatusPageImpact is what a replace does to the public status page
type StatusPageImpact struct {
	// Group is the name of the group the page shows
	Group string `json:"group"`
	// Kept: the document has a group with that name, which the page then
	// shows. Otherwise the page goes offline.
	Kept bool `json:"kept"`
}

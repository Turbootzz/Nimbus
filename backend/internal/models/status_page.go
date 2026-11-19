package models

import "time"

// Overall states of the public status page
const (
	StatusPageOperational = "operational" // every service is online
	StatusPageDegraded    = "degraded"    // some services are offline
	StatusPageOutage      = "outage"      // every service is offline
	StatusPageUnknown     = "unknown"     // nothing has been checked yet
)

const (
	DefaultStatusPageTitle = "Status"
	MaxStatusPageTitleLen  = 80
)

// StatusPageSettings is the admin configuration of the public status page
type StatusPageSettings struct {
	Enabled   bool   `json:"enabled"`
	GroupID   string `json:"group_id"`   // One of the saving admin's own groups
	Title     string `json:"title"`      // Empty shows DefaultStatusPageTitle
	Logo      string `json:"logo"`       // File name of an uploaded service icon, or empty
	ShowLinks bool   `json:"show_links"` // Expose service URLs to visitors
}

// UptimeBucket is the number of checks of one service in one hour
type UptimeBucket struct {
	ServiceID string
	Hour      int // Hours since the start of the queried range
	Total     int
	Online    int
}

// PublicStatusPage is everything an anonymous visitor may see. Add fields
// with care: nothing here is behind authentication.
type PublicStatusPage struct {
	Title     string                `json:"title"`
	Logo      string                `json:"logo,omitempty"`
	Overall   string                `json:"overall"`
	UpdatedAt time.Time             `json:"updated_at"`
	Services  []PublicStatusService `json:"services"`
}

// PublicStatusService is one row of the public status page. Uptime values
// are percentages; nil means there were no checks in that period.
type PublicStatusService struct {
	Name          string     `json:"name"`
	Icon          string     `json:"icon"`
	IconType      string     `json:"icon_type"`
	IconImagePath string     `json:"icon_image_path,omitempty"`
	Status        string     `json:"status"`
	URL           string     `json:"url,omitempty"` // Only with ShowLinks
	Uptime24h     *float64   `json:"uptime_24h"`
	Uptime7d      *float64   `json:"uptime_7d"`
	Bars24h       []*float64 `json:"bars_24h"` // 24 bars of one hour, oldest first
	Bars7d        []*float64 `json:"bars_7d"`  // 28 bars of six hours, oldest first
}

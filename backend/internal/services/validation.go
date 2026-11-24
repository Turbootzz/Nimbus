package services

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/utils"
)

// Refresh interval bounds shared by integrations and widgets
const (
	minRefreshSeconds = 10
	maxRefreshSeconds = 86400
)

// ValidationError marks bad user input. Its message is safe to show.
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string { return e.Message }

func invalid(format string, args ...any) error {
	return &ValidationError{Message: fmt.Sprintf(format, args...)}
}

// NewGroupFromRequest validates a group create request and returns the group
// to store, without its owner
func NewGroupFromRequest(req *models.GroupCreateRequest) (*models.Group, error) {
	if req.Name == "" {
		return nil, invalid("Name is required")
	}
	if len(req.Name) > models.MaxGroupNameLen {
		return nil, invalid("Name must be %d characters or less", models.MaxGroupNameLen)
	}

	color := req.Color
	if color == "" {
		color = models.DefaultGroupColor
	}
	if !models.IsValidHexColor(color) {
		return nil, invalid("Invalid color format. Must be hex format #RRGGBB")
	}

	tileStyle := req.TileStyle
	if tileStyle == "" {
		tileStyle = models.TileStyleCards
	}
	if !models.IsValidTileStyle(tileStyle) {
		return nil, invalid("Invalid tile style. Must be cards or chips")
	}

	// Default monitoring to enabled
	monitoringEnabled := true
	if req.MonitoringEnabled != nil {
		monitoringEnabled = *req.MonitoringEnabled
	}

	now := time.Now()
	return &models.Group{
		Name:              req.Name,
		Color:             color,
		MonitoringEnabled: monitoringEnabled,
		TileStyle:         tileStyle,
		CreatedAt:         now,
		UpdatedAt:         now,
	}, nil
}

// NewServiceFromRequest validates a service create request and returns the
// service to store, without its owner, group and integration: the caller
// checks that those belong to the user
func NewServiceFromRequest(req *models.ServiceCreateRequest) (*models.Service, error) {
	if req.Name == "" || req.URL == "" {
		return nil, invalid("Name and URL are required")
	}
	if utf8.RuneCountInString(req.Name) > models.MaxServiceNameLen {
		return nil, invalid("Name must be %d characters or less", models.MaxServiceNameLen)
	}
	if utf8.RuneCountInString(req.Icon) > models.MaxServiceIconLen {
		return nil, invalid("Icon must be %d characters or less", models.MaxServiceIconLen)
	}

	// http(s) only: the URL becomes a link on the tile and a health check target
	if err := utils.ValidateServiceURL(req.URL); err != nil {
		return nil, invalid("Invalid URL: %s", err)
	}

	iconType := req.IconType
	if iconType == "" {
		iconType = models.IconTypeEmoji
	}
	if iconType != models.IconTypeEmoji && iconType != models.IconTypeImageUpload && iconType != models.IconTypeImageURL {
		return nil, invalid("Invalid icon_type. Must be 'emoji', 'image_upload', or 'image_url'")
	}

	// Set default emoji if emoji type and no icon provided
	icon := req.Icon
	if iconType == models.IconTypeEmoji && icon == "" {
		icon = models.DefaultIcon
	}

	iconImagePath := strings.TrimSpace(req.IconImagePath)
	if iconType == models.IconTypeImageUpload {
		if iconImagePath == "" {
			return nil, invalid("icon_image_path is required for image_upload type")
		}
		// Prevent path traversal attacks - only allow filename, no path separators
		if strings.Contains(iconImagePath, "..") || strings.Contains(iconImagePath, "/") || strings.Contains(iconImagePath, "\\") {
			return nil, invalid("icon_image_path must be a filename only, no path separators allowed")
		}
	}
	if iconType == models.IconTypeImageURL {
		if iconImagePath == "" {
			return nil, invalid("icon_image_path (URL) is required for image_url type")
		}
		// Validate URL format (no SSRF risk since browser renders the URL, not the server)
		if err := utils.ValidateIconURL(iconImagePath); err != nil {
			return nil, invalid("Invalid image URL: %s", err.Error())
		}
	}

	cardSize := req.CardSize
	if cardSize == "" {
		cardSize = models.DefaultCardSize
	}
	if !models.IsValidCardSize(cardSize) {
		return nil, invalid("Invalid card_size. Must be '1x1', '2x1', or '2x2'")
	}

	// Default monitoring to enabled
	monitoringEnabled := true
	if req.MonitoringEnabled != nil {
		monitoringEnabled = *req.MonitoringEnabled
	}

	now := time.Now()
	return &models.Service{
		Name:              req.Name,
		URL:               req.URL,
		Icon:              icon,
		IconType:          iconType,
		IconImagePath:     iconImagePath,
		Description:       req.Description,
		Status:            models.StatusUnknown, // Initial status
		CardSize:          cardSize,
		MonitoringEnabled: monitoringEnabled,
		CreatedAt:         now,
		UpdatedAt:         now,
	}, nil
}

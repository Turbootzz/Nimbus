package services

import (
	"cmp"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/repository"
	"github.com/nimbus/backend/internal/widgets"
)

const (
	maxWidgetsPerUser           = 200
	defaultWidgetRefreshSeconds = 300
	maxWidgetTitleRunes         = 100
	maxWidgetConfigBytes        = 64 * 1024
	maxReorderTiles             = 1000
)

// ErrRefreshTooSoon means the widget was fetched moments ago
var ErrRefreshTooSoon = errors.New("refreshed too recently")

// GroupGetter is the part of GroupRepository the widget service needs
type GroupGetter interface {
	GetByID(ctx context.Context, id string) (*models.Group, error)
}

// WidgetService handles widget CRUD and the shared tile order
type WidgetService struct {
	repo         repository.WidgetRepositoryInterface
	groups       GroupGetter
	integrations repository.IntegrationRepositoryInterface
	poller       Poller
}

func NewWidgetService(repo repository.WidgetRepositoryInterface, groups GroupGetter, integrations repository.IntegrationRepositoryInterface) *WidgetService {
	return &WidgetService{repo: repo, groups: groups, integrations: integrations}
}

// SetPoller lets the service tell the widget poller about changes
func (s *WidgetService) SetPoller(p Poller) {
	s.poller = p
}

// Types lists the widget types the user can add
func (s *WidgetService) Types(isAdmin bool) []widgets.Meta {
	types := widgets.Types()
	if !isAdmin {
		types = slices.DeleteFunc(types, func(m widgets.Meta) bool { return m.AdminOnly })
	}
	return types
}

// List returns the user's widgets with secrets in their config masked
func (s *WidgetService) List(ctx context.Context, userID string) ([]models.Widget, error) {
	list, err := s.repo.ListByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	for i := range list {
		redact(&list[i])
	}
	return list, nil
}

// Get returns a widget with secrets in its config masked
func (s *WidgetService) Get(ctx context.Context, id, userID string) (*models.Widget, error) {
	widget, err := s.repo.GetByID(ctx, id, userID)
	if err != nil {
		return nil, err
	}
	redact(widget)
	return widget, nil
}

// redact masks secrets in the config, for widgets sent to a client
func redact(widget *models.Widget) {
	if widgetType, ok := widgets.Get(widget.Type); ok {
		if secret, ok := widgetType.(widgets.SecretConfig); ok {
			widget.Config = secret.Redact(widget.Config)
		}
	}
}

func (s *WidgetService) Delete(ctx context.Context, id, userID string) error {
	if err := s.repo.Delete(ctx, id, userID); err != nil {
		return err
	}
	kick(s.poller)
	return nil
}

// Refresh asks the poller to fetch a widget's data now
func (s *WidgetService) Refresh(ctx context.Context, id, userID string) error {
	widget, err := s.repo.GetByID(ctx, id, userID)
	if err != nil {
		return err
	}
	widgetType, ok := widgets.Get(widget.Type)
	if _, fetches := widgetType.(widgets.Fetcher); !ok || !fetches {
		return invalid("This widget has no data to refresh")
	}
	if !widget.Enabled {
		return invalid("This widget is disabled")
	}
	if s.poller != nil && !s.poller.Refresh(models.SnapshotKey(models.SnapshotSourceWidget, widget.ID)) {
		return ErrRefreshTooSoon
	}
	return nil
}

// Create validates the request and adds the widget at the end of the grid
func (s *WidgetService) Create(ctx context.Context, userID string, isAdmin bool, req *models.WidgetRequest) (*models.Widget, error) {
	widget, err := s.apply(ctx, userID, isAdmin, req, nil)
	if err != nil {
		return nil, err
	}
	if err := s.repo.Create(ctx, widget, maxWidgetsPerUser); err != nil {
		if errors.Is(err, repository.ErrWidgetLimitReached) {
			return nil, invalid("Maximum widget limit reached (%d)", maxWidgetsPerUser)
		}
		return nil, err
	}
	kick(s.poller)
	redact(widget)
	return widget, nil
}

// Update applies the request on top of the stored widget
func (s *WidgetService) Update(ctx context.Context, id, userID string, isAdmin bool, req *models.WidgetRequest) (*models.Widget, error) {
	current, err := s.repo.GetByID(ctx, id, userID)
	if err != nil {
		return nil, err
	}
	widget, err := s.apply(ctx, userID, isAdmin, req, current)
	if err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, widget); err != nil {
		return nil, err
	}
	kick(s.poller)
	redact(widget)
	return widget, nil
}

// ReorderTiles saves new positions for services and widgets at once
func (s *WidgetService) ReorderTiles(ctx context.Context, userID string, req *models.TileReorderRequest) error {
	if len(req.Tiles) == 0 {
		return invalid("At least one tile is required")
	}
	if len(req.Tiles) > maxReorderTiles {
		return invalid("At most %d tiles can be reordered at once", maxReorderTiles)
	}

	seen := make(map[models.TilePosition]bool, len(req.Tiles))
	for _, tile := range req.Tiles {
		if tile.Kind != models.TileKindService && tile.Kind != models.TileKindWidget {
			return invalid("Tile kind must be service or widget")
		}
		if _, err := uuid.Parse(tile.ID); err != nil {
			return invalid("Tile ID must be a UUID")
		}
		if tile.Position < 0 || tile.Position > repository.MaxTilePosition {
			return invalid("Position must be between 0 and %d", repository.MaxTilePosition)
		}
		key := models.TilePosition{ID: strings.ToLower(tile.ID), Kind: tile.Kind}
		if seen[key] {
			return invalid("Each tile can only appear once")
		}
		seen[key] = true
	}
	return s.repo.ReorderTiles(ctx, userID, req.Tiles)
}

// apply validates req on top of current (nil on create) and returns the
// widget to store. On update, omitted fields keep their stored value.
func (s *WidgetService) apply(ctx context.Context, userID string, isAdmin bool, req *models.WidgetRequest, current *models.Widget) (*models.Widget, error) {
	w := models.Widget{UserID: userID, Enabled: true}
	if current != nil {
		w = *current
	}

	if typ := strings.TrimSpace(req.Type); typ != "" {
		if current != nil && typ != current.Type {
			return nil, invalid("Type cannot be changed")
		}
		w.Type = typ
	}
	widgetType, ok := widgets.Get(w.Type)
	if !ok {
		return nil, invalid("Unknown widget type")
	}
	meta := widgetType.Meta()
	// Moving or resizing is fine; adding or changing what it fetches is not
	if meta.AdminOnly && !isAdmin && (current == nil || hasConfig(req.Config)) {
		return nil, invalid("Only admins can set up %s widgets", meta.Name)
	}

	if req.Title != nil {
		w.Title = strings.TrimSpace(*req.Title)
	}
	if utf8.RuneCountInString(w.Title) > maxWidgetTitleRunes {
		return nil, invalid("Title must be %d characters or less", maxWidgetTitleRunes)
	}

	if current == nil {
		w.CardSize = meta.DefaultSize
	}
	if req.CardSize != nil {
		w.CardSize = *req.CardSize
	}
	if !slices.Contains(meta.AllowedSizes, w.CardSize) {
		return nil, invalid("Card size %q is not available for %s", w.CardSize, meta.Name)
	}

	minRefresh := max(minRefreshSeconds, meta.MinRefreshSeconds)
	if current == nil {
		w.RefreshSeconds = max(cmp.Or(meta.DefaultRefreshSeconds, defaultWidgetRefreshSeconds), minRefresh)
	}
	if req.RefreshSeconds != nil {
		w.RefreshSeconds = *req.RefreshSeconds
	}
	if w.RefreshSeconds < minRefresh || w.RefreshSeconds > maxRefreshSeconds {
		return nil, invalid("Refresh interval must be between %d and %d seconds", minRefresh, maxRefreshSeconds)
	}

	if req.Enabled != nil {
		w.Enabled = *req.Enabled
	}

	if req.GroupID != nil {
		groupID, err := s.resolveGroup(ctx, userID, *req.GroupID)
		if err != nil {
			return nil, err
		}
		w.GroupID = groupID
	}

	// A deleted integration leaves integration_id NULL; only check it when it
	// is set, so the widget can still be moved or resized in the meantime
	if current == nil || req.IntegrationID != nil {
		if req.IntegrationID != nil {
			w.IntegrationID = nil
			if id := strings.TrimSpace(*req.IntegrationID); id != "" {
				w.IntegrationID = &id
			}
		}
		if err := s.checkIntegration(ctx, userID, meta, w.IntegrationID); err != nil {
			return nil, err
		}
	}

	config, err := s.validateConfig(widgetType, req.Config, current)
	if err != nil {
		return nil, err
	}
	w.Config = config

	return &w, nil
}

// validateConfig returns the normalised config. A missing config keeps the
// stored one on update and is validated as {} on create, so types with
// required fields reject it.
func (s *WidgetService) validateConfig(widgetType widgets.WidgetType, config json.RawMessage, current *models.Widget) (json.RawMessage, error) {
	if !hasConfig(config) {
		if current != nil {
			return current.Config, nil
		}
		config = json.RawMessage(`{}`)
	}
	if len(config) > maxWidgetConfigBytes {
		return nil, invalid("Config is too large")
	}
	// Masked secrets that come back unchanged keep their stored value
	if secret, ok := widgetType.(widgets.SecretConfig); ok && current != nil {
		config = secret.Unredact(config, current.Config)
	}
	normalised, err := widgetType.Validate(config)
	if err != nil {
		return nil, invalid("Invalid config: %s", err.Error())
	}
	return normalised, nil
}

func hasConfig(config json.RawMessage) bool {
	return len(config) > 0 && string(config) != "null"
}

// resolveGroup checks that a group belongs to the user; "" means no group
func (s *WidgetService) resolveGroup(ctx context.Context, userID, groupID string) (*string, error) {
	groupID = strings.TrimSpace(groupID)
	if groupID == "" {
		return nil, nil
	}
	if _, err := uuid.Parse(groupID); err != nil {
		return nil, invalid("Group not found")
	}
	group, err := s.groups.GetByID(ctx, groupID)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && group.UserID != userID) {
		return nil, invalid("Group not found")
	}
	if err != nil {
		return nil, err
	}
	return &group.ID, nil
}

// checkIntegration enforces the type's integration rules and ownership
func (s *WidgetService) checkIntegration(ctx context.Context, userID string, meta widgets.Meta, integrationID *string) error {
	if len(meta.IntegrationKinds) == 0 {
		if integrationID != nil {
			return invalid("%s widgets don't use an integration", meta.Name)
		}
		return nil
	}
	if integrationID == nil {
		return invalid("%s widgets need an integration", meta.Name)
	}
	if _, err := uuid.Parse(*integrationID); err != nil {
		return invalid("Integration not found")
	}
	integration, err := s.integrations.GetByID(ctx, *integrationID, userID)
	if errors.Is(err, repository.ErrIntegrationNotFound) {
		return invalid("Integration not found")
	}
	if err != nil {
		return err
	}
	if !slices.Contains(meta.IntegrationKinds, integration.Kind) {
		return invalid("%s widgets can't use a %s integration", meta.Name, integration.Kind)
	}
	return nil
}

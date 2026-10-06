package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"strconv"
	"time"

	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/repository"
	"github.com/nimbus/backend/internal/utils"
	"github.com/nimbus/backend/internal/widgets"
)

// Longest a single widget or integration fetch may take
const maxFetchTimeout = 15 * time.Second

// Poller is the part of the widget poller that services call after changes
type Poller interface {
	// Kick reloads the list of sources soon
	Kick()
	// Refresh fetches one source (by snapshot key) as soon as possible; it
	// returns false when the source was fetched too recently
	Refresh(key string) bool
}

func kick(p Poller) {
	if p != nil {
		p.Kick()
	}
}

// LiveSource is one widget or integration the poller fetches
type LiveSource struct {
	Kind     string // models.SnapshotSource*
	ID       string
	UserID   string
	Interval time.Duration
	// Version changes when the source is edited, so it is fetched again at once
	Version string
	Fetch   func(ctx context.Context) (any, error)
}

// Key is the snapshot key of the source
func (s LiveSource) Key() string {
	return models.SnapshotKey(s.Kind, s.ID)
}

// ServiceStatusEvent is sent to the dashboard after every health check
type ServiceStatusEvent struct {
	ID           string `json:"id"`
	Status       string `json:"status"`
	ResponseTime *int   `json:"response_time,omitempty"`
}

// LiveDataService provides what the poller fetches and keeps the results:
// in memory for the dashboard, in the database for restarts, and pushed to
// open dashboards over SSE.
type LiveDataService struct {
	widgetRepo         repository.WidgetRepositoryInterface
	integrationRepo    repository.IntegrationRepositoryInterface
	integrationService *IntegrationService
	snapshots          repository.SnapshotRepositoryInterface
	cache              *DashboardCache
	hub                *SSEHub
	now                func() time.Time
}

func NewLiveDataService(
	widgetRepo repository.WidgetRepositoryInterface,
	integrationRepo repository.IntegrationRepositoryInterface,
	integrationService *IntegrationService,
	snapshots repository.SnapshotRepositoryInterface,
	hub *SSEHub,
) *LiveDataService {
	return &LiveDataService{
		widgetRepo:         widgetRepo,
		integrationRepo:    integrationRepo,
		integrationService: integrationService,
		snapshots:          snapshots,
		cache:              NewDashboardCache(),
		hub:                hub,
		now:                time.Now,
	}
}

// Sources lists what to poll: enabled widgets that fetch data, and the
// integrations those widgets use
func (s *LiveDataService) Sources(ctx context.Context) ([]LiveSource, error) {
	integrationList, err := s.integrationRepo.ListInUse(ctx)
	if err != nil {
		return nil, err
	}
	widgetList, err := s.widgetRepo.ListEnabled(ctx)
	if err != nil {
		return nil, err
	}

	sources := make([]LiveSource, 0, len(integrationList)+len(widgetList))
	byID := make(map[string]*models.Integration, len(integrationList))
	for i := range integrationList {
		integration := &integrationList[i]
		byID[integration.ID] = integration
		sources = append(sources, s.integrationSource(integration))
	}
	for _, widget := range widgetList {
		widgetType, ok := widgets.Get(widget.Type)
		if !ok {
			continue
		}
		fetcher, ok := widgetType.(widgets.Fetcher)
		if !ok {
			continue
		}
		var integration *models.Integration
		if widget.IntegrationID != nil {
			integration = byID[*widget.IntegrationID]
		}
		sources = append(sources, s.widgetSource(widget, widgetType.Meta(), fetcher, integration))
	}
	return sources, nil
}

func (s *LiveDataService) integrationSource(integration *models.Integration) LiveSource {
	return LiveSource{
		Kind:     models.SnapshotSourceIntegration,
		ID:       integration.ID,
		UserID:   integration.UserID,
		Interval: time.Duration(integration.RefreshSeconds) * time.Second,
		Version:  integrationVersion(integration),
		Fetch: func(ctx context.Context) (any, error) {
			return s.integrationService.Fetch(ctx, integration)
		},
	}
}

func (s *LiveDataService) widgetSource(widget models.Widget, meta widgets.Meta, fetcher widgets.Fetcher, integration *models.Integration) LiveSource {
	// Only what changes the fetched data counts; a move or resize doesn't
	parts := []string{widget.Type, string(widget.Config), strconv.Itoa(widget.RefreshSeconds)}
	if integration != nil {
		parts = append(parts, integrationVersion(integration))
	}
	return LiveSource{
		Kind:     models.SnapshotSourceWidget,
		ID:       widget.ID,
		UserID:   widget.UserID,
		Interval: time.Duration(widget.RefreshSeconds) * time.Second,
		Version:  fingerprint(parts...),
		Fetch: func(ctx context.Context) (any, error) {
			client := utils.NewSafeClient(widgets.VerifiesTLS(widget.Config), maxFetchTimeout)
			req := &widgets.FetchRequest{Config: widget.Config, Client: client}
			defer req.Client.CloseIdleConnections()

			if len(meta.IntegrationKinds) > 0 {
				if integration == nil {
					return nil, errors.New("this widget has no integration")
				}
				conn, err := s.integrationService.Conn(ctx, integration)
				if err != nil {
					return nil, err
				}
				defer conn.Client.CloseIdleConnections()
				req.Integration = conn
				payload, err := callWidgetFetch(ctx, widget.Type, fetcher, req)
				if err != nil {
					return nil, errors.New(redactSecrets(err.Error(), conn.Creds))
				}
				return payload, nil
			}
			return callWidgetFetch(ctx, widget.Type, fetcher, req)
		},
	}
}

func callWidgetFetch(ctx context.Context, typ string, fetcher widgets.Fetcher, req *widgets.FetchRequest) (payload any, err error) {
	defer recoverCrash("widget "+typ, &err)
	return fetcher.Fetch(ctx, req)
}

// integrationVersion changes when the integration is edited (test results
// don't touch updated_at)
func integrationVersion(integration *models.Integration) string {
	return integration.ID + "@" + strconv.FormatInt(integration.UpdatedAt.UnixNano(), 10)
}

// fingerprint is a short hash of the parts
func fingerprint(parts ...string) string {
	h := sha256.New()
	for _, part := range parts {
		h.Write([]byte(part))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil)[:12])
}

// Record keeps the result of a fetch and pushes it to the user's open
// dashboards. A failed fetch keeps the last good payload and marks it stale.
func (s *LiveDataService) Record(src LiveSource, payload any, fetchErr error) {
	snap := models.Snapshot{
		SourceKind: src.Kind,
		SourceID:   src.ID,
		UserID:     src.UserID,
		FetchedAt:  s.now().UTC(),
	}
	if fetchErr == nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			fetchErr = errors.New("the data could not be encoded")
		} else {
			snap.Payload = raw
		}
	}
	if fetchErr != nil {
		if prev, ok := s.cache.Get(src.UserID, src.Key()); ok && prev.Payload != nil {
			snap.Payload, snap.FetchedAt = prev.Payload, prev.FetchedAt
		}
		snap.Error = fetchErr.Error()
		snap.Stale = true
	}

	s.cache.Set(snap)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.snapshots.Upsert(ctx, &snap); err != nil {
		log.Printf("Failed to save snapshot %s: %v", src.Key(), err)
	}
	s.hub.Publish(src.UserID, Event{Name: EventSnapshot, Data: snap})
}

// Retain forgets snapshots of sources that are no longer polled
func (s *LiveDataService) Retain(keys map[string]bool) {
	s.cache.Retain(keys)
}

// Warm loads the stored snapshots into memory at boot, marked stale until
// the poller fetches them again
func (s *LiveDataService) Warm(ctx context.Context) error {
	snapshots, err := s.snapshots.ListAll(ctx)
	if err != nil {
		return err
	}
	for _, snap := range snapshots {
		snap.Stale = true
		s.cache.Set(snap)
	}
	return nil
}

// Snapshots returns the latest data of all the user's polled sources
func (s *LiveDataService) Snapshots(userID string) []models.Snapshot {
	return s.cache.ForUser(userID)
}

// PruneSnapshots deletes stored snapshots of deleted widgets and integrations
func (s *LiveDataService) PruneSnapshots(ctx context.Context) (int64, error) {
	return s.snapshots.PruneOrphans(ctx)
}

// PublishServiceStatus pushes a health check result to open dashboards
func (s *LiveDataService) PublishServiceStatus(userID, serviceID, status string, responseTime *int) {
	s.hub.Publish(userID, Event{
		Name: EventServiceStatus,
		Data: ServiceStatusEvent{ID: serviceID, Status: status, ResponseTime: responseTime},
	})
}

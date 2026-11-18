package services

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/repository"
)

// System setting keys of the status page; all start with StatusPageKeyPrefix
const (
	StatusPageKeyPrefix   = "status_page_"
	statusPageEnabledKey  = StatusPageKeyPrefix + "enabled"
	statusPageGroupKey    = StatusPageKeyPrefix + "group_id"
	statusPageTitleKey    = StatusPageKeyPrefix + "title"
	statusPageLogoKey     = StatusPageKeyPrefix + "logo"
	statusPageShowLinkKey = StatusPageKeyPrefix + "show_links"
)

const (
	statusPageCacheTTL = 30 * time.Second
	uptimeHours        = 7 * 24
	hoursPer7dBar      = 6
)

// Names the upload handler generates for uploaded images
var uploadedImageName = regexp.MustCompile(`^[0-9a-f]{32}\.(jpg|png|gif|webp)$`)

// StatusPageService holds the settings of the public status page and builds
// what anonymous visitors get to see
type StatusPageService struct {
	settingsRepo  *repository.SettingsRepository
	groupRepo     *repository.GroupRepository
	serviceRepo   *repository.ServiceRepository
	statusLogRepo *repository.StatusLogRepository
	userRepo      *repository.UserRepository
	logoDir       string
	now           func() time.Time

	// The page is the same for every visitor, so one cached copy keeps
	// anonymous traffic away from the database. A disabled page (nil) is
	// cached too.
	mu       sync.Mutex
	cached   *models.PublicStatusPage
	cachedAt time.Time
	hasCache bool
}

// NewStatusPageService creates the service; logoDir is where uploaded logos live
func NewStatusPageService(
	settingsRepo *repository.SettingsRepository,
	groupRepo *repository.GroupRepository,
	serviceRepo *repository.ServiceRepository,
	statusLogRepo *repository.StatusLogRepository,
	userRepo *repository.UserRepository,
	logoDir string,
) *StatusPageService {
	return &StatusPageService{
		settingsRepo:  settingsRepo,
		groupRepo:     groupRepo,
		serviceRepo:   serviceRepo,
		statusLogRepo: statusLogRepo,
		userRepo:      userRepo,
		logoDir:       logoDir,
		now:           time.Now,
	}
}

// Settings returns the stored configuration; missing keys mean "off"
func (s *StatusPageService) Settings(ctx context.Context) (models.StatusPageSettings, error) {
	all, err := s.settingsRepo.GetAll(ctx)
	if err != nil {
		return models.StatusPageSettings{}, err
	}
	values := map[string]string{}
	for _, setting := range all {
		values[setting.Key] = setting.Value
	}
	return models.StatusPageSettings{
		Enabled:   values[statusPageEnabledKey] == "true",
		GroupID:   values[statusPageGroupKey],
		Title:     values[statusPageTitleKey],
		Logo:      values[statusPageLogoKey],
		ShowLinks: values[statusPageShowLinkKey] == "true",
	}, nil
}

// Save validates and stores the configuration on behalf of an admin
func (s *StatusPageService) Save(ctx context.Context, userID string, settings models.StatusPageSettings) (models.StatusPageSettings, error) {
	settings.Title = strings.TrimSpace(settings.Title)
	if utf8.RuneCountInString(settings.Title) > models.MaxStatusPageTitleLen {
		return settings, invalid("Title must be at most %d characters", models.MaxStatusPageTitleLen)
	}

	if settings.GroupID == "" {
		if settings.Enabled {
			return settings, invalid("Choose a group before enabling the status page")
		}
	} else if err := s.requireOwnGroup(ctx, userID, settings.GroupID); err != nil {
		return settings, err
	}

	if settings.Logo != "" {
		if !uploadedImageName.MatchString(settings.Logo) {
			return settings, invalid("Logo must be an uploaded image")
		}
		if _, err := os.Stat(filepath.Join(s.logoDir, settings.Logo)); err != nil {
			return settings, invalid("Logo must be an uploaded image")
		}
	}

	values := map[string]string{
		statusPageEnabledKey:  boolSetting(settings.Enabled),
		statusPageGroupKey:    settings.GroupID,
		statusPageTitleKey:    settings.Title,
		statusPageLogoKey:     settings.Logo,
		statusPageShowLinkKey: boolSetting(settings.ShowLinks),
	}
	if err := s.settingsRepo.UpdateBatch(ctx, values, &userID); err != nil {
		return settings, err
	}
	s.Invalidate()
	return settings, nil
}

// requireOwnGroup rejects a group that is missing or someone else's, with
// one message so group IDs cannot be probed
func (s *StatusPageService) requireOwnGroup(ctx context.Context, userID, groupID string) error {
	const message = "Choose one of your own groups"
	if _, err := uuid.Parse(groupID); err != nil {
		return invalid(message)
	}
	group, err := s.groupRepo.GetByID(ctx, groupID)
	if errors.Is(err, sql.ErrNoRows) {
		return invalid(message)
	}
	if err != nil {
		return err
	}
	if group.UserID != userID {
		return invalid(message)
	}
	return nil
}

func boolSetting(v bool) string {
	if v {
		return "true"
	}
	return "false"
}

// Invalidate drops the cached page so a settings change shows at once
func (s *StatusPageService) Invalidate() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.hasCache = false
}

// Public returns the page for anonymous visitors, or nil when it is disabled.
// The result is cached for 30 seconds.
func (s *StatusPageService) Public(ctx context.Context) (*models.PublicStatusPage, error) {
	// ponytail: one lock around the build, so concurrent visitors wait for a
	// single query instead of each running it. Fine for one cheap page.
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now()
	if s.hasCache && now.Sub(s.cachedAt) < statusPageCacheTTL {
		return s.cached, nil
	}

	page, err := s.build(ctx, now)
	if err != nil {
		return nil, err
	}
	s.cached, s.cachedAt, s.hasCache = page, now, true
	return page, nil
}

// build reads everything from the database; nil means the page is off
func (s *StatusPageService) build(ctx context.Context, now time.Time) (*models.PublicStatusPage, error) {
	settings, err := s.Settings(ctx)
	if err != nil {
		return nil, err
	}
	if !settings.Enabled || settings.GroupID == "" {
		return nil, nil
	}

	// The group can be deleted and its owner demoted after the page was set
	// up; either one turns the page off
	group, err := s.groupRepo.GetByID(ctx, settings.GroupID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	owner, err := s.userRepo.GetByID(group.UserID)
	if errors.Is(err, repository.ErrUserNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if owner.Role != "admin" {
		return nil, nil
	}

	services, err := s.serviceRepo.GetMonitoredByGroupID(ctx, group.ID)
	if err != nil {
		return nil, err
	}

	// Hour buckets end with the running hour, so the bars line up on the clock
	since := now.Truncate(time.Hour).Add(-(uptimeHours - 1) * time.Hour)
	buckets, err := s.statusLogRepo.GetHourlyUptimeByGroup(ctx, group.ID, since)
	if err != nil {
		return nil, err
	}
	hourly := map[string]*[uptimeHours]uptimeCount{}
	for _, b := range buckets {
		if b.Hour < 0 || b.Hour >= uptimeHours {
			continue
		}
		if hourly[b.ServiceID] == nil {
			hourly[b.ServiceID] = &[uptimeHours]uptimeCount{}
		}
		hourly[b.ServiceID][b.Hour] = uptimeCount{total: b.Total, online: b.Online}
	}

	page := &models.PublicStatusPage{
		Title:     settings.Title,
		Logo:      settings.Logo,
		UpdatedAt: now.UTC(),
		Services:  make([]models.PublicStatusService, 0, len(services)),
	}
	if page.Title == "" {
		page.Title = models.DefaultStatusPageTitle
	}

	online, offline := 0, 0
	for _, service := range services {
		switch service.Status {
		case models.StatusOnline:
			online++
		case models.StatusOffline:
			offline++
		}
		page.Services = append(page.Services, publicStatusService(service, hourly[service.ID], settings.ShowLinks))
	}
	page.Overall = overallStatus(online, offline)
	return page, nil
}

// overallStatus sums up the services; unchecked ones do not count
func overallStatus(online, offline int) string {
	switch {
	case online+offline == 0:
		return models.StatusPageUnknown
	case offline == 0:
		return models.StatusPageOperational
	case online == 0:
		return models.StatusPageOutage
	default:
		return models.StatusPageDegraded
	}
}

// publicStatusService copies only what a visitor may see
func publicStatusService(service *models.Service, hours *[uptimeHours]uptimeCount, showLinks bool) models.PublicStatusService {
	if hours == nil {
		hours = &[uptimeHours]uptimeCount{}
	}
	last24 := hours[uptimeHours-24:]

	out := models.PublicStatusService{
		Name:          service.Name,
		Icon:          service.Icon,
		IconType:      service.IconType,
		IconImagePath: service.IconImagePath,
		Status:        service.Status,
		Uptime24h:     sumUptime(last24).percentage(),
		Uptime7d:      sumUptime(hours[:]).percentage(),
		Bars24h:       uptimeBars(last24, 1),
		Bars7d:        uptimeBars(hours[:], hoursPer7dBar),
	}
	if out.Icon == "" {
		out.Icon = models.DefaultIcon
	}
	if showLinks {
		out.URL = service.URL
	} else if service.IconType == models.IconTypeImageURL {
		// A remote icon often lives on the service itself, so its URL would
		// give the address away; fall back to the emoji
		out.IconType = models.IconTypeEmoji
		out.IconImagePath = ""
	}
	return out
}

type uptimeCount struct{ total, online int }

// percentage is the uptime with two decimals, nil without checks
func (u uptimeCount) percentage() *float64 {
	if u.total == 0 {
		return nil
	}
	pct := math.Round(float64(u.online)/float64(u.total)*10000) / 100
	return &pct
}

func sumUptime(hours []uptimeCount) uptimeCount {
	var sum uptimeCount
	for _, h := range hours {
		sum.total += h.total
		sum.online += h.online
	}
	return sum
}

// uptimeBars folds hour buckets into bars of perBar hours
func uptimeBars(hours []uptimeCount, perBar int) []*float64 {
	bars := make([]*float64, 0, len(hours)/perBar)
	for i := 0; i+perBar <= len(hours); i += perBar {
		bars = append(bars, sumUptime(hours[i:i+perBar]).percentage())
	}
	return bars
}

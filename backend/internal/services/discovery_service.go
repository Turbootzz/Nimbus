package services

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/nimbus/backend/internal/integrations"
	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/repository"
	"github.com/nimbus/backend/internal/utils"
)

const (
	dockerKind = "docker"
	// Limits of the add request; labels are cut to fit when parsed
	discoveredNameMax        = 255
	discoveredDescriptionMax = 500
	discoveredURLMax         = 2048
	// How long a Docker integration's container list is reused, so the
	// dashboard pill does not ask Docker on every page load
	containerCacheTTL = 30 * time.Second
	// Most containers one auto-add run adds; the rest follow on later runs
	autoAddMaxPerRun = 20
)

// ErrDiscoveryHandled means the container was added or ignored already
var ErrDiscoveryHandled = errors.New("container was added or ignored already")

// Docker container names: [a-zA-Z0-9][a-zA-Z0-9_.-]*
var containerNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,254}$`)

// DiscoveryService offers the containers behind Docker integrations as
// services. Besides the container list it never requests anything: URLs and
// icons from labels are only validated and stored.
type DiscoveryService struct {
	integrations *IntegrationService
	repo         *repository.DiscoveryRepository
	services     *repository.ServiceRepository
	groups       *repository.GroupRepository
	settings     *repository.SettingsRepository
	// iconBase is the dashboard-icons URL; empty leaves icons out
	iconBase string
	// managed tells whose services a config file decides; nil: nobody's
	managed func(userID string) bool

	// ponytail: in-process locks, Nimbus runs as one process. groupMu keeps
	// two adds from creating the same group twice (group names have no unique
	// index); use a database lock if Nimbus ever runs as several instances.
	groupMu sync.Mutex
	cacheMu sync.Mutex
	cache   map[string]cachedContainers
}

// cachedContainers is one Docker integration's answer, error included: an
// unreachable host is not asked again on every page load either
type cachedContainers struct {
	containers []integrations.DiscoveredContainer
	err        error
	at         time.Time
}

func NewDiscoveryService(
	integrationService *IntegrationService,
	repo *repository.DiscoveryRepository,
	serviceRepo *repository.ServiceRepository,
	groupRepo *repository.GroupRepository,
	settingsRepo *repository.SettingsRepository,
	iconBase string,
) *DiscoveryService {
	return &DiscoveryService{
		integrations: integrationService,
		repo:         repo,
		services:     serviceRepo,
		groups:       groupRepo,
		settings:     settingsRepo,
		iconBase:     iconBase,
		cache:        map[string]cachedContainers{},
	}
}

// Discover lists the containers of all the user's Docker integrations that
// are not a service yet; ignored ones are included and marked. A failing
// integration is reported without hiding the others. socketHost is the host
// Nimbus was reached on, the best guess for containers behind a unix socket.
// refresh asks Docker again instead of reusing a recent answer.
func (s *DiscoveryService) Discover(ctx context.Context, userID, socketHost string, refresh bool) (*models.DiscoveryResult, error) {
	all, err := s.integrations.List(ctx, userID)
	if err != nil {
		return nil, err
	}
	handled, err := s.repo.List(ctx, userID)
	if err != nil {
		return nil, err
	}

	result := &models.DiscoveryResult{Containers: []models.DiscoveredContainer{}, Errors: []models.DiscoveryError{}}
	for i := range all {
		integration := &all[i]
		if integration.Kind != dockerKind {
			continue
		}
		result.Integrations++
		containers, err := s.containers(ctx, integration, refresh)
		if err != nil {
			result.Errors = append(result.Errors, models.DiscoveryError{
				IntegrationID: integration.ID, IntegrationName: integration.Name, Error: err.Error(),
			})
			continue
		}
		for _, container := range containers {
			candidate := s.candidate(integration, container, socketHost)
			serviceID, isHandled := handled[candidate.Key]
			if serviceID != nil {
				continue // already a service
			}
			candidate.Ignored = isHandled
			result.Containers = append(result.Containers, candidate)
		}
	}

	// Labelled first, then running, then by name
	rank := func(c models.DiscoveredContainer) int {
		switch {
		case c.Enabled:
			return 0
		case c.State == "running":
			return 1
		}
		return 2
	}
	slices.SortStableFunc(result.Containers, func(a, b models.DiscoveredContainer) int {
		return cmp.Or(cmp.Compare(rank(a), rank(b)), strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name)))
	})
	return result, nil
}

// containers returns one Docker integration's containers, from the cache
// unless fresh is set or the answer is older than containerCacheTTL
func (s *DiscoveryService) containers(ctx context.Context, integration *models.Integration, fresh bool) ([]integrations.DiscoveredContainer, error) {
	// The address is part of the key, so an edited integration is asked anew.
	// ponytail: entries of deleted integrations stay until restart (one small
	// list each); sweep them if that ever matters.
	key := integration.ID + " " + integration.BaseURL
	s.cacheMu.Lock()
	cached, ok := s.cache[key]
	s.cacheMu.Unlock()
	if ok && !fresh && time.Since(cached.at) < containerCacheTTL {
		return cached.containers, cached.err
	}
	containers, err := s.fetchContainers(ctx, integration)
	if ctx.Err() == nil { // a cancelled request says nothing about Docker
		s.cacheMu.Lock()
		s.cache[key] = cachedContainers{containers: containers, err: err, at: time.Now()}
		s.cacheMu.Unlock()
	}
	return containers, err
}

// fetchContainers asks one Docker integration for its containers. Errors are redacted.
func (s *DiscoveryService) fetchContainers(ctx context.Context, integration *models.Integration) ([]integrations.DiscoveredContainer, error) {
	conn, err := s.integrations.Conn(ctx, integration)
	if err != nil {
		return nil, err
	}
	defer conn.Client.CloseIdleConnections()

	ctx, cancel := context.WithTimeout(ctx, s.integrations.timeout)
	defer cancel()
	containers, err := integrations.DockerDiscover(ctx, conn)
	if err != nil {
		return nil, errors.New(redactSecrets(err.Error(), conn.Creds))
	}
	return containers, nil
}

// candidate turns a container into what the tray offers
func (s *DiscoveryService) candidate(integration *models.Integration, container integrations.DiscoveredContainer, socketHost string) models.DiscoveredContainer {
	labels := container.Labels
	candidate := models.DiscoveredContainer{
		Key:             discoveryKey(integration.ID, container.Name),
		IntegrationID:   integration.ID,
		IntegrationName: integration.Name,
		Container:       container.Name,
		Image:           container.Image,
		State:           container.State,
		Enabled:         labels.Enabled,
		Name:            cmp.Or(labels.Name, container.Name),
		Group:           labels.Group,
		Description:     labels.Description,
		IconURL:         s.iconURL(labels.Icon, container.Image),
	}
	if validServiceURL(labels.URL) {
		candidate.URL = labels.URL
	} else if guess := guessURL(integration.BaseURL, socketHost, container.Port); validServiceURL(guess) {
		candidate.URL, candidate.URLGuessed = guess, true
	}
	if impl, ok := integrations.Get(labels.Widget); ok && candidate.URL != "" {
		candidate.Integration = &models.DiscoveredIntegration{Kind: labels.Widget, Name: impl.Meta().Name, BaseURL: candidate.URL}
	}
	return candidate
}

// iconURL is the icon label when it is an http(s) URL, else the
// dashboard-icons image for the label's slug or one guessed from the image
func (s *DiscoveryService) iconURL(icon, image string) string {
	if utils.ValidateIconURL(icon) == nil {
		return icon
	}
	slug := cmp.Or(integrations.IconSlug(icon), integrations.ImageIconSlug(image))
	if slug == "" || s.iconBase == "" {
		return ""
	}
	return s.iconBase + "/" + slug + ".svg"
}

// validServiceURL allows only http(s) URLs with a host that is no cloud
// metadata endpoint: labels come from whoever can start containers, and the
// URL ends up as a link and as a health check target
func validServiceURL(rawURL string) bool {
	return len(rawURL) <= discoveredURLMax && utils.ValidateServiceURL(rawURL) == nil
}

// guessURL builds a URL from a published port, "" without one. The host is
// the Docker integration's own (a socket proxy or Portainer usually runs on
// the Docker host); behind a unix socket the containers run next to Nimbus,
// so it is the host Nimbus was reached on.
func guessURL(baseURL, socketHost string, port int) string {
	if port <= 0 {
		return ""
	}
	host := cmp.Or(socketHost, "localhost")
	if !strings.HasPrefix(baseURL, unixScheme) {
		parsed, err := url.Parse(baseURL)
		if err != nil || parsed.Hostname() == "" {
			return ""
		}
		host = parsed.Hostname()
	}
	switch port {
	case 80:
		return "http://" + hostOnly(host)
	case 443:
		return "https://" + hostOnly(host)
	}
	return "http://" + net.JoinHostPort(host, strconv.Itoa(port))
}

// hostOnly puts an IPv6 address in brackets
func hostOnly(host string) string {
	if strings.Contains(host, ":") {
		return "[" + host + "]"
	}
	return host
}

func discoveryKey(integrationID, container string) string {
	return integrationID + "/" + container
}

// checkKey makes sure key names a container of one of the user's Docker integrations
func (s *DiscoveryService) checkKey(ctx context.Context, userID, key string) error {
	integrationID, container, _ := strings.Cut(key, "/")
	// Only the form Discover returns: a lowercase UUID
	if parsed, err := uuid.Parse(integrationID); err != nil || parsed.String() != integrationID || !containerNamePattern.MatchString(container) {
		return invalid("Unknown container")
	}
	integration, err := s.integrations.Get(ctx, integrationID, userID)
	if errors.Is(err, repository.ErrIntegrationNotFound) || (err == nil && integration.Kind != dockerKind) {
		return invalid("Unknown container")
	}
	return err
}

// Add creates a service from a discovered container, in the group with the
// given name (matched ignoring case, created when missing). The container is
// not offered again after that.
func (s *DiscoveryService) Add(ctx context.Context, userID string, req *models.DiscoveryAddRequest) (*models.Service, error) {
	name, group := strings.TrimSpace(req.Name), strings.TrimSpace(req.Group)
	serviceURL, iconURL := strings.TrimSpace(req.URL), strings.TrimSpace(req.IconURL)
	switch {
	case name == "":
		return nil, invalid("Name is required")
	case utf8.RuneCountInString(name) > discoveredNameMax:
		return nil, invalid("Name must be %d characters or less", discoveredNameMax)
	case !validServiceURL(serviceURL):
		return nil, invalid("URL must be an http:// or https:// address")
	case iconURL != "" && (len(iconURL) > discoveredURLMax || utils.ValidateIconURL(iconURL) != nil):
		return nil, invalid("Icon must be an http:// or https:// address")
	case len(group) > models.MaxGroupNameLen:
		return nil, invalid("Group must be %d characters or less", models.MaxGroupNameLen)
	case utf8.RuneCountInString(req.Description) > discoveredDescriptionMax:
		return nil, invalid("Description must be %d characters or less", discoveredDescriptionMax)
	}
	if err := s.checkKey(ctx, userID, req.Key); err != nil {
		return nil, err
	}

	// The claim decides who adds the container when two try at once
	claimed, err := s.repo.Claim(ctx, userID, req.Key, true)
	if err != nil {
		return nil, err
	}
	if !claimed {
		return nil, ErrDiscoveryHandled
	}
	service, err := s.createService(ctx, userID, name, serviceURL, iconURL, group, req.Description)
	// The claim is settled even when the request or the worker was cancelled
	// meanwhile: a claim left behind would hide the container as ignored
	settle := context.WithoutCancel(ctx)
	if err != nil {
		if releaseErr := s.repo.Unclaim(settle, userID, req.Key); releaseErr != nil {
			log.Printf("discovery: %v", releaseErr)
		}
		return nil, err
	}
	if err := s.repo.SetService(settle, userID, req.Key, service.ID); err != nil {
		// The service exists and the claim keeps the container from being
		// offered again; only the link between the two is missing
		log.Printf("discovery: %v", err)
	}
	return service, nil
}

func (s *DiscoveryService) createService(ctx context.Context, userID, name, serviceURL, iconURL, group, description string) (*models.Service, error) {
	groupID, err := s.groupID(ctx, userID, group)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	service := &models.Service{
		UserID:            userID,
		Name:              name,
		URL:               serviceURL,
		Icon:              models.DefaultIcon,
		IconType:          models.IconTypeEmoji,
		Description:       description,
		Status:            models.StatusUnknown,
		CardSize:          models.DefaultCardSize,
		GroupID:           groupID,
		MonitoringEnabled: true,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	if iconURL != "" {
		service.IconType, service.IconImagePath = models.IconTypeImageURL, iconURL
	}
	if err := s.services.Create(ctx, service); err != nil {
		return nil, fmt.Errorf("failed to create service: %w", err)
	}
	return service, nil
}

// groupID finds the user's group with this name, ignoring case, and creates
// it when there is none. No name means no group.
func (s *DiscoveryService) groupID(ctx context.Context, userID, name string) (*string, error) {
	if name == "" {
		return nil, nil
	}
	s.groupMu.Lock()
	defer s.groupMu.Unlock()
	groups, err := s.groups.GetAllByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list groups: %w", err)
	}
	for _, group := range groups {
		if strings.EqualFold(group.Name, name) {
			return &group.ID, nil
		}
	}
	now := time.Now()
	group := &models.Group{UserID: userID, Name: name, Color: models.DefaultGroupColor, MonitoringEnabled: true, CreatedAt: now, UpdatedAt: now}
	if err := s.groups.Create(ctx, group); err != nil {
		return nil, fmt.Errorf("failed to create group: %w", err)
	}
	return &group.ID, nil
}

// Ignore hides a container from the tray until it is un-ignored
func (s *DiscoveryService) Ignore(ctx context.Context, userID, key string) error {
	if err := s.checkKey(ctx, userID, key); err != nil {
		return err
	}
	_, err := s.repo.Claim(ctx, userID, key, false)
	return err
}

// Unignore offers an ignored container again
func (s *DiscoveryService) Unignore(ctx context.Context, userID, key string) error {
	return s.repo.Release(ctx, userID, key)
}

// SetManaged tells AutoAdd whose dashboard a config file owns
// (NIMBUS_CONFIG_FILE). It adds nothing there: the file would remove the
// service at its next sync, and a read-only dashboard takes no additions.
func (s *DiscoveryService) SetManaged(managed func(userID string) bool) {
	s.managed = managed
}

// AutoAddEnabled reports the admin setting; off unless it is "true"
func (s *DiscoveryService) AutoAddEnabled(ctx context.Context) (bool, error) {
	setting, err := s.settings.Get(ctx, models.SettingDiscoveryAutoAdd)
	if errors.Is(err, repository.ErrSettingNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return setting.Value == "true", nil
}

// AutoAdd adds every container labelled nimbus.enable=true that has a nimbus.url
// label, for each admin's Docker integrations, when the admin setting is on.
// It never adds a guessed URL, an ignored container, or one it added before
// (also not after that service was deleted), and nothing to a dashboard a
// config file owns (SetManaged). Returns how many were added,
// at most autoAddMaxPerRun.
func (s *DiscoveryService) AutoAdd(ctx context.Context) (int, error) {
	enabled, err := s.AutoAddEnabled(ctx)
	if err != nil || !enabled {
		return 0, err
	}
	all, err := s.repo.AdminIntegrations(ctx, dockerKind)
	if err != nil {
		return 0, err
	}

	added := 0
	handledByUser := map[string]map[string]*string{}
	for i := range all {
		integration := &all[i]
		if s.managed != nil && s.managed(integration.UserID) {
			continue
		}
		handled, ok := handledByUser[integration.UserID]
		if !ok {
			if handled, err = s.repo.List(ctx, integration.UserID); err != nil {
				return added, err
			}
			handledByUser[integration.UserID] = handled
		}
		containers, err := s.containers(ctx, integration, true)
		if err != nil {
			log.Printf("discovery: integration %s: %v", integration.ID, err)
			continue
		}
		for _, container := range containers {
			if added >= autoAddMaxPerRun || ctx.Err() != nil {
				return added, ctx.Err()
			}
			candidate := s.candidate(integration, container, "")
			if _, isHandled := handled[candidate.Key]; isHandled || !candidate.Enabled || !container.Labels.NimbusURL || candidate.URLGuessed || candidate.URL == "" {
				continue
			}
			_, err := s.Add(ctx, integration.UserID, &models.DiscoveryAddRequest{
				Key: candidate.Key, Name: candidate.Name, URL: candidate.URL, IconURL: candidate.IconURL,
				Group: candidate.Group, Description: candidate.Description,
			})
			switch {
			case err == nil:
				added++
			case errors.Is(err, ErrDiscoveryHandled):
			default:
				log.Printf("discovery: could not add container %q of integration %s: %v", container.Name, integration.ID, err)
			}
		}
	}
	return added, nil
}

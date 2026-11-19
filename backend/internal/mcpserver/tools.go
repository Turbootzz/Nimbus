package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"slices"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/repository"
	svc "github.com/nimbus/backend/internal/services"
	widgettypes "github.com/nimbus/backend/internal/widgets"
)

// Tool results are their own types, so a field added to a model never
// reaches an assistant by accident.

// ServiceInfo is a service as the tools return it
type ServiceInfo struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	URL               string `json:"url"`
	Description       string `json:"description,omitempty"`
	Group             string `json:"group,omitempty"`
	Status            string `json:"status" jsonschema:"online, offline, unknown (not checked yet) or not_monitored"`
	ResponseTimeMs    *int   `json:"response_time_ms,omitempty"`
	MonitoringEnabled bool   `json:"monitoring_enabled" jsonschema:"false means Nimbus does not health check this service, because of its own setting or its group's"`
}

// ServiceList is a list of services with a status count
type ServiceList struct {
	Services     []ServiceInfo `json:"services"`
	Total        int           `json:"total"`
	Online       int           `json:"online"`
	Offline      int           `json:"offline"`
	Unknown      int           `json:"unknown"`
	NotMonitored int           `json:"not_monitored"`
}

// statusNotMonitored is the status of a service Nimbus does not check: its
// stored status is stale
const statusNotMonitored = "not_monitored"

// CheckInfo is one past health check
type CheckInfo struct {
	Status         string    `json:"status"`
	ResponseTimeMs *int      `json:"response_time_ms,omitempty"`
	Error          string    `json:"error,omitempty"`
	CheckedAt      time.Time `json:"checked_at"`
}

// ServiceStatus is a service with its latest health checks
type ServiceStatus struct {
	Service      ServiceInfo `json:"service"`
	RecentChecks []CheckInfo `json:"recent_checks" jsonschema:"latest health checks, newest first"`
}

// Uptime is the reliability of a service over a time range
type Uptime struct {
	Service           ServiceInfo `json:"service"`
	Range             string      `json:"range"`
	UptimePercentage  float64     `json:"uptime_percentage"`
	TotalChecks       int         `json:"total_checks" jsonschema:"0 means there is no data for this range"`
	OnlineChecks      int         `json:"online_checks"`
	OfflineChecks     int         `json:"offline_checks"`
	AvgResponseTimeMs float64     `json:"avg_response_time_ms"`
	MinResponseTimeMs float64     `json:"min_response_time_ms"`
	MaxResponseTimeMs float64     `json:"max_response_time_ms"`
}

// WidgetData is a widget with the data it currently shows
type WidgetData struct {
	ID        string     `json:"id"`
	Type      string     `json:"type"`
	Title     string     `json:"title"`
	Enabled   bool       `json:"enabled"`
	Config    any        `json:"config,omitempty" jsonschema:"widget settings with secrets masked"`
	Data      any        `json:"data,omitempty" jsonschema:"latest data the widget fetched, if it fetches any"`
	Error     string     `json:"error,omitempty" jsonschema:"why the latest fetch failed"`
	FetchedAt *time.Time `json:"fetched_at,omitempty"`
	Stale     bool       `json:"stale,omitempty" jsonschema:"data is older than the widget's refresh interval"`
}

// IntegrationInfo is a connected app, without its URL, options or credentials
type IntegrationInfo struct {
	ID         string     `json:"id"`
	Kind       string     `json:"kind"`
	Name       string     `json:"name"`
	Status     string     `json:"status" jsonschema:"ok, failing or untested"`
	LastTestAt *time.Time `json:"last_test_at,omitempty"`
}

// IntegrationList is the result of list_integrations
type IntegrationList struct {
	Integrations []IntegrationInfo `json:"integrations"`
}

// Tool inputs

type listServicesInput struct {
	Group string `json:"group,omitempty" jsonschema:"only services in this group (name or ID)"`
	Board string `json:"board,omitempty" jsonschema:"board to read (name, slug or ID); default: the default board"`
}

type serviceInput struct {
	Service string `json:"service" jsonschema:"service ID or name"`
	Board   string `json:"board,omitempty" jsonschema:"board the service is on (name, slug or ID); default: the default board"`
}

type uptimeInput struct {
	Service string `json:"service" jsonschema:"service ID or name"`
	Range   string `json:"range,omitempty" jsonschema:"1h, 24h, 7d or 30d (default 24h)"`
	Board   string `json:"board,omitempty" jsonschema:"board the service is on (name, slug or ID); default: the default board"`
}

type searchInput struct {
	Q     string `json:"q" jsonschema:"text to look for in the name, description, URL or group"`
	Board string `json:"board,omitempty" jsonschema:"board to search (name, slug or ID); default: the default board"`
}

type idInput struct {
	ID    string `json:"id" jsonschema:"ID from the nimbus://dashboard resource"`
	Board string `json:"board,omitempty" jsonschema:"board the widget is on (name, slug or ID); default: the default board"`
}

type recheckInput struct {
	ID    string `json:"id" jsonschema:"service ID or name"`
	Board string `json:"board,omitempty" jsonschema:"board the service is on (name, slug or ID); default: the default board"`
}

type noInput struct{}

// uptimeRanges are the ranges get_uptime accepts
var uptimeRanges = map[string]time.Duration{
	"1h":  time.Hour,
	"24h": 24 * time.Hour,
	"7d":  7 * 24 * time.Hour,
	"30d": 30 * 24 * time.Hour,
}

const recentChecks = 5

type tools struct {
	deps Deps
}

func (t *tools) register(server *mcp.Server) {
	readOnly := &mcp.ToolAnnotations{ReadOnlyHint: true}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_services",
		Description: "List the services on the dashboard with their current status, optionally limited to one group.",
		Annotations: readOnly,
	}, t.listServices)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_service_status",
		Description: "Current status of one service and its latest health checks, including the error of failed checks.",
		Annotations: readOnly,
	}, t.getServiceStatus)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_uptime",
		Description: "Uptime percentage and response times of one service over the last 1h, 24h, 7d or 30d.",
		Annotations: readOnly,
	}, t.getUptime)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "search_services",
		Description: "Find services whose name, description, URL or group contains the given text.",
		Annotations: readOnly,
	}, t.searchServices)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_widget_data",
		Description: "Settings and latest data of one dashboard widget. Widget IDs are listed in the " + DashboardURI + " resource and in the error for an unknown ID.",
		Annotations: readOnly,
	}, t.getWidgetData)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_integrations",
		Description: "List the apps connected to Nimbus and whether their connection works. Never includes URLs or credentials.",
		Annotations: readOnly,
	}, t.listIntegrations)
	notDestructive := false
	mcp.AddTool(server, &mcp.Tool{
		Name:        "recheck_service",
		Description: "Run a health check for one service now and return its new status. Needs a token with the write:services scope. A service that is not monitored (status not_monitored: monitoring is off for it or its group) cannot be checked.",
		Annotations: &mcp.ToolAnnotations{DestructiveHint: &notDestructive, IdempotentHint: true},
	}, t.recheckService)

	server.AddResource(&mcp.Resource{
		URI:         DashboardURI,
		Name:        "dashboard",
		Title:       "Nimbus dashboard",
		Description: "Short summary of the dashboard: service status, groups, widgets and integrations.",
		MIMEType:    "text/markdown",
	}, t.readDashboard)
}

// userServices loads the services of one of the caller's boards (board: a
// name, slug or ID; "" is the default board) and the names of its groups.
// Services are always loaded by owner or by a board the caller may see,
// never by ID alone. They are shown as on the dashboard: monitored only when
// their group is too, and ungrouped ones in the board's default group.
func (t *tools) userServices(ctx context.Context, userID, board string) ([]*models.Service, map[string]string, error) {
	var list []*models.Service
	var groups []*models.Group
	content, err := t.board(ctx, userID, board)
	switch {
	case err != nil:
		return nil, nil, err
	case content != nil:
		list, groups = content.Services, content.Groups
	default:
		if list, err = t.deps.Services.GetAllByUserID(ctx, userID); err != nil {
			return nil, nil, internalError("load services", err)
		}
		if groups, err = t.deps.Groups.GetAllByUserID(ctx, userID); err != nil {
			return nil, nil, internalError("load groups", err)
		}
	}
	groupNames := make(map[string]string, len(groups))
	paused := map[string]bool{}
	var defaultGroup *string
	for _, group := range groups {
		groupNames[group.ID] = group.Name
		paused[group.ID] = !group.MonitoringEnabled
		if group.IsDefault {
			defaultGroup = &group.ID
		}
	}
	for _, service := range list {
		if service.GroupID == nil {
			// Its own flag alone decides, as for the health checker
			service.GroupID = defaultGroup
		} else if paused[*service.GroupID] {
			service.MonitoringEnabled = false
		}
	}
	return list, groupNames, nil
}

// userWidgets loads the widgets of one of the caller's boards, with the
// secrets in their config masked, and the latest data of what it shows
func (t *tools) userWidgets(ctx context.Context, userID, board string) ([]models.Widget, []models.Snapshot, error) {
	content, err := t.board(ctx, userID, board)
	if err != nil {
		return nil, nil, err
	}
	if content != nil {
		return content.Widgets, content.Snapshots, nil
	}
	widgets, err := t.deps.Widgets.List(ctx, userID)
	if err != nil {
		return nil, nil, internalError("load widgets", err)
	}
	return widgets, t.deps.Snapshots.Snapshots(userID), nil
}

// board reads the board ref means. It returns nil without boards (Deps.
// Boards unset): the caller then has one dashboard.
func (t *tools) board(ctx context.Context, userID, ref string) (*svc.BoardContent, error) {
	if t.deps.Boards == nil {
		if strings.TrimSpace(ref) != "" {
			return nil, errors.New("this server has no boards")
		}
		return nil, nil
	}
	content, err := t.deps.Boards.Read(ctx, userID, ref)
	if errors.Is(err, repository.ErrBoardNotFound) {
		boards, listErr := t.deps.Boards.List(ctx, userID)
		if listErr != nil {
			return nil, internalError("load boards", listErr)
		}
		return nil, fmt.Errorf("no board %q; boards: %s", ref, strings.Join(boardLines(boards, userID), "; "))
	}
	if err != nil {
		return nil, internalError("load the board", err)
	}
	return content, nil
}

// boardLines describes boards as "<name> (<slug>)"
func boardLines(boards []*models.Board, userID string) []string {
	lines := make([]string, len(boards))
	for i, board := range boards {
		lines[i] = fmt.Sprintf("%s (%s)", board.Name, board.Slug)
		switch {
		case board.UserID != userID:
			lines[i] += ", shared with you, read-only"
		case board.IsDefault:
			lines[i] += ", default"
		case board.IsShared:
			lines[i] += ", shared with everyone"
		}
	}
	return lines
}

// internalError logs the cause and returns a message that is safe to show
func internalError(action string, err error) error {
	log.Printf("MCP: failed to %s: %v", action, err)
	return fmt.Errorf("failed to %s", action)
}

func serviceInfo(service *models.Service, groupNames map[string]string) ServiceInfo {
	info := ServiceInfo{
		ID:                service.ID,
		Name:              service.Name,
		URL:               service.URL,
		Description:       service.Description,
		Status:            service.Status,
		ResponseTimeMs:    service.ResponseTime,
		MonitoringEnabled: service.MonitoringEnabled,
	}
	if !service.MonitoringEnabled {
		info.Status, info.ResponseTimeMs = statusNotMonitored, nil
	}
	if service.GroupID != nil {
		info.Group = groupNames[*service.GroupID]
	}
	return info
}

func serviceList(services []*models.Service, groupNames map[string]string) ServiceList {
	list := ServiceList{Services: make([]ServiceInfo, 0, len(services)), Total: len(services)}
	for _, service := range services {
		info := serviceInfo(service, groupNames)
		list.Services = append(list.Services, info)
		switch info.Status {
		case models.StatusOnline:
			list.Online++
		case models.StatusOffline:
			list.Offline++
		case statusNotMonitored:
			list.NotMonitored++
		default:
			list.Unknown++
		}
	}
	return list
}

// findService resolves an ID or a name (case-insensitive) to one service
func findService(services []*models.Service, ref string) (*models.Service, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return nil, errors.New("a service ID or name is required")
	}
	var matches []*models.Service
	for _, service := range services {
		if service.ID == ref {
			return service, nil
		}
		if strings.EqualFold(service.Name, ref) {
			matches = append(matches, service)
		}
	}
	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("no service with ID or name %q; use search_services or list_services to find it", ref)
	case 1:
		return matches[0], nil
	}
	ids := make([]string, len(matches))
	for i, match := range matches {
		ids[i] = match.ID
	}
	return nil, fmt.Errorf("%d services are named %q; use one of these IDs: %s", len(matches), ref, strings.Join(ids, ", "))
}

func (t *tools) listServices(ctx context.Context, req *mcp.CallToolRequest, in listServicesInput) (*mcp.CallToolResult, ServiceList, error) {
	who, err := callerOf(req.Extra)
	if err != nil {
		return nil, ServiceList{}, err
	}
	services, groupNames, err := t.userServices(ctx, who.userID, in.Board)
	if err != nil {
		return nil, ServiceList{}, err
	}

	if group := strings.TrimSpace(in.Group); group != "" {
		groupID := ""
		for id, name := range groupNames {
			if id == group || strings.EqualFold(name, group) {
				groupID = id
			}
		}
		if groupID == "" {
			names := make([]string, 0, len(groupNames))
			for _, name := range groupNames {
				names = append(names, name)
			}
			slices.Sort(names)
			return nil, ServiceList{}, fmt.Errorf("no group %q; groups: %s", group, strings.Join(names, ", "))
		}
		services = slices.DeleteFunc(services, func(s *models.Service) bool {
			return s.GroupID == nil || *s.GroupID != groupID
		})
	}

	return nil, serviceList(services, groupNames), nil
}

func (t *tools) searchServices(ctx context.Context, req *mcp.CallToolRequest, in searchInput) (*mcp.CallToolResult, ServiceList, error) {
	who, err := callerOf(req.Extra)
	if err != nil {
		return nil, ServiceList{}, err
	}
	q := strings.ToLower(strings.TrimSpace(in.Q))
	if q == "" {
		return nil, ServiceList{}, errors.New("q is required")
	}
	services, groupNames, err := t.userServices(ctx, who.userID, in.Board)
	if err != nil {
		return nil, ServiceList{}, err
	}

	services = slices.DeleteFunc(services, func(s *models.Service) bool {
		group := ""
		if s.GroupID != nil {
			group = groupNames[*s.GroupID]
		}
		for _, field := range []string{s.Name, s.Description, s.URL, group} {
			if strings.Contains(strings.ToLower(field), q) {
				return false
			}
		}
		return true
	})
	return nil, serviceList(services, groupNames), nil
}

func (t *tools) getServiceStatus(ctx context.Context, req *mcp.CallToolRequest, in serviceInput) (*mcp.CallToolResult, ServiceStatus, error) {
	who, err := callerOf(req.Extra)
	if err != nil {
		return nil, ServiceStatus{}, err
	}
	services, groupNames, err := t.userServices(ctx, who.userID, in.Board)
	if err != nil {
		return nil, ServiceStatus{}, err
	}
	service, err := findService(services, in.Service)
	if err != nil {
		return nil, ServiceStatus{}, err
	}

	logs, err := t.deps.StatusLogs.GetLatestByServiceID(ctx, service.ID, recentChecks)
	if err != nil {
		return nil, ServiceStatus{}, internalError("load status history", err)
	}
	status := ServiceStatus{Service: serviceInfo(service, groupNames), RecentChecks: make([]CheckInfo, 0, len(logs))}
	for _, entry := range logs {
		check := CheckInfo{Status: entry.Status, ResponseTimeMs: entry.ResponseTime, CheckedAt: entry.CheckedAt}
		switch {
		case entry.ErrorMessage == nil:
		case service.UserID != who.userID:
			// A board shared with the caller: the error may quote the owner's network
			check.Error = svc.ViewerCheckError
		default:
			check.Error = *entry.ErrorMessage
		}
		status.RecentChecks = append(status.RecentChecks, check)
	}
	return nil, status, nil
}

func (t *tools) getUptime(ctx context.Context, req *mcp.CallToolRequest, in uptimeInput) (*mcp.CallToolResult, Uptime, error) {
	who, err := callerOf(req.Extra)
	if err != nil {
		return nil, Uptime{}, err
	}
	if in.Range == "" {
		in.Range = "24h"
	}
	span, ok := uptimeRanges[in.Range]
	if !ok {
		return nil, Uptime{}, fmt.Errorf("range must be 1h, 24h, 7d or 30d, not %q", in.Range)
	}
	services, groupNames, err := t.userServices(ctx, who.userID, in.Board)
	if err != nil {
		return nil, Uptime{}, err
	}
	service, err := findService(services, in.Service)
	if err != nil {
		return nil, Uptime{}, err
	}

	end := time.Now()
	stats, err := t.deps.StatusLogs.GetUptimeStats(ctx, service.ID, end.Add(-span), end)
	if err != nil {
		return nil, Uptime{}, internalError("load uptime", err)
	}
	// The repository returns a map; a missing key reads as zero
	count := func(key string) int { v, _ := stats[key].(int); return v }
	number := func(key string) float64 { v, _ := stats[key].(float64); return v }
	return nil, Uptime{
		Service:           serviceInfo(service, groupNames),
		Range:             in.Range,
		UptimePercentage:  number("uptime_percentage"),
		TotalChecks:       count("total_checks"),
		OnlineChecks:      count("online_count"),
		OfflineChecks:     count("offline_count"),
		AvgResponseTimeMs: number("avg_response_time"),
		MinResponseTimeMs: number("min_response_time"),
		MaxResponseTimeMs: number("max_response_time"),
	}, nil
}

func (t *tools) getWidgetData(ctx context.Context, req *mcp.CallToolRequest, in idInput) (*mcp.CallToolResult, WidgetData, error) {
	who, err := callerOf(req.Extra)
	if err != nil {
		return nil, WidgetData{}, err
	}
	widgets, snapshots, err := t.userWidgets(ctx, who.userID, in.Board)
	if err != nil {
		return nil, WidgetData{}, err
	}
	index := slices.IndexFunc(widgets, func(w models.Widget) bool { return w.ID == strings.TrimSpace(in.ID) })
	if index < 0 {
		if len(widgets) == 0 {
			return nil, WidgetData{}, errors.New("the dashboard has no widgets")
		}
		return nil, WidgetData{}, fmt.Errorf("no widget with ID %q; widgets: %s", in.ID, strings.Join(widgetLines(widgets), "; "))
	}
	widget := widgets[index]

	data := WidgetData{ID: widget.ID, Type: widget.Type, Title: widget.Title, Enabled: widget.Enabled, Config: widgettypes.StripURLSecrets(decodeJSON(widget.Config))}
	key := models.SnapshotKey(models.SnapshotSourceWidget, widget.ID)
	for _, snap := range snapshots {
		if snap.Key() != key {
			continue
		}
		fetchedAt := snap.FetchedAt
		data.Data = decodeJSON(snap.Payload)
		data.Error = snap.Error
		data.FetchedAt = &fetchedAt
		data.Stale = snap.Stale
	}
	return nil, data, nil
}

// decodeJSON turns stored JSON into a value the result can embed
func decodeJSON(raw json.RawMessage) any {
	var value any
	if len(raw) == 0 || json.Unmarshal(raw, &value) != nil {
		return nil
	}
	return value
}

// widgetLines describes widgets as "<id> (<type>: <title>)"
func widgetLines(widgets []models.Widget) []string {
	lines := make([]string, len(widgets))
	for i, widget := range widgets {
		lines[i] = widget.ID + " (" + widget.Type
		if widget.Title != "" {
			lines[i] += ": " + widget.Title
		}
		lines[i] += ")"
	}
	return lines
}

func integrationStatus(integration *models.Integration) string {
	switch {
	case integration.LastTestOK == nil:
		return "untested"
	case *integration.LastTestOK:
		return "ok"
	}
	return "failing"
}

func (t *tools) listIntegrations(ctx context.Context, req *mcp.CallToolRequest, _ noInput) (*mcp.CallToolResult, IntegrationList, error) {
	who, err := callerOf(req.Extra)
	if err != nil {
		return nil, IntegrationList{}, err
	}
	integrations, err := t.deps.Integrations.List(ctx, who.userID)
	if err != nil {
		return nil, IntegrationList{}, internalError("load integrations", err)
	}

	// Only what identifies the connection: the REST API keeps integrations
	// away from tokens, so no URL, options or error text here.
	list := IntegrationList{Integrations: make([]IntegrationInfo, 0, len(integrations))}
	for i := range integrations {
		list.Integrations = append(list.Integrations, IntegrationInfo{
			ID:         integrations[i].ID,
			Kind:       integrations[i].Kind,
			Name:       integrations[i].Name,
			Status:     integrationStatus(&integrations[i]),
			LastTestAt: integrations[i].LastTestAt,
		})
	}
	return nil, list, nil
}

func (t *tools) recheckService(ctx context.Context, req *mcp.CallToolRequest, in recheckInput) (*mcp.CallToolResult, ServiceStatus, error) {
	who, err := callerOf(req.Extra)
	if err != nil {
		return nil, ServiceStatus{}, err
	}
	if !slices.Contains(who.scopes, models.ScopeWriteServices) {
		return nil, ServiceStatus{}, errors.New("this API token is read-only; recheck_service needs the write:services scope")
	}
	services, _, err := t.userServices(ctx, who.userID, in.Board)
	if err != nil {
		return nil, ServiceStatus{}, err
	}
	service, err := findService(services, in.ID)
	if err != nil {
		return nil, ServiceStatus{}, err
	}
	if service.UserID != who.userID {
		return nil, ServiceStatus{}, fmt.Errorf("%q is on a board that is shared with you; only its owner can run a check", service.Name)
	}
	if !service.MonitoringEnabled {
		return nil, ServiceStatus{}, fmt.Errorf("monitoring is disabled for %q or its group, so it cannot be checked", service.Name)
	}

	if err := t.deps.Health.CheckService(ctx, service); err != nil {
		return nil, ServiceStatus{}, internalError("run the health check", err)
	}
	// Read the result back like any other status request
	return t.getServiceStatus(ctx, req, serviceInput{Service: service.ID, Board: in.Board})
}

func (t *tools) readDashboard(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	who, err := callerOf(req.Extra)
	if err != nil {
		return nil, err
	}
	services, groupNames, err := t.userServices(ctx, who.userID, "")
	if err != nil {
		return nil, err
	}
	widgets, _, err := t.userWidgets(ctx, who.userID, "")
	if err != nil {
		return nil, err
	}
	integrations, err := t.deps.Integrations.List(ctx, who.userID)
	if err != nil {
		return nil, internalError("load integrations", err)
	}

	list := serviceList(services, groupNames)
	var b strings.Builder
	fmt.Fprintf(&b, "# Nimbus dashboard\n\n%d services: %d online, %d offline, %d unknown, %d not monitored.\n", list.Total, list.Online, list.Offline, list.Unknown, list.NotMonitored)

	if list.Offline > 0 {
		b.WriteString("\n## Offline\n")
		for _, service := range list.Services {
			if service.Status == models.StatusOffline {
				fmt.Fprintf(&b, "- %s (%s)\n", service.Name, service.URL)
			}
		}
	}

	perGroup := map[string]int{}
	for _, service := range list.Services {
		perGroup[service.Group]++
	}
	if len(perGroup) > 0 {
		b.WriteString("\n## Groups\n")
		names := make([]string, 0, len(perGroup))
		for name := range perGroup {
			names = append(names, name)
		}
		slices.Sort(names)
		for _, name := range names {
			label := name
			if label == "" {
				label = "(no group)"
			}
			fmt.Fprintf(&b, "- %s: %d services\n", label, perGroup[name])
		}
	}

	if len(widgets) > 0 {
		b.WriteString("\n## Widgets\n")
		for _, line := range widgetLines(widgets) {
			fmt.Fprintf(&b, "- %s\n", line)
		}
	}

	// Only worth a section when there is more than the board above
	if t.deps.Boards != nil {
		boards, err := t.deps.Boards.List(ctx, who.userID)
		if err != nil {
			return nil, internalError("load boards", err)
		}
		if len(boards) > 1 {
			b.WriteString("\n## Boards\nThe above is the default board. Pass \"board\" to a tool to read another one.\n")
			for _, line := range boardLines(boards, who.userID) {
				fmt.Fprintf(&b, "- %s\n", line)
			}
		}
	}

	if len(integrations) > 0 {
		b.WriteString("\n## Integrations\n")
		for i := range integrations {
			fmt.Fprintf(&b, "- %s (%s): %s\n", integrations[i].Name, integrations[i].Kind, integrationStatus(&integrations[i]))
		}
	}

	return &mcp.ReadResourceResult{
		// One user's dashboard: nothing in between may cache it for another
		Cacheable: mcp.Cacheable{CacheScope: "private"},
		Contents:  []*mcp.ResourceContents{{URI: DashboardURI, MIMEType: "text/markdown", Text: b.String()}},
	}, nil
}

package services

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/nimbus/backend/internal/dashdoc"
	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/repository"
	widgettypes "github.com/nimbus/backend/internal/widgets"
)

// storedDashboard is the dashboard a user has, and the rows the entities of
// a document were matched to (by their index in the document)
type storedDashboard struct {
	groups       []*models.Group
	integrations []*models.Integration
	services     []*models.Service
	widgets      []*models.Widget

	group       map[int]*models.Group
	integration map[int]*models.Integration
	service     map[int]*models.Service
	widget      map[int]*models.Widget

	// boards are the document's boards: a row only matches an entity on
	// its own board (nil without boards)
	boards *docBoards
	// otherWidgets counts the user's widgets on boards the document
	// doesn't work on, and usedElsewhere holds the integrations tiles there
	// use: both stay whatever the document says
	otherWidgets  int
	usedElsewhere map[string]bool
}

// The ID of the row entity i of the document was matched to, or a new one.
// A nil storedDashboard matches nothing.

func (d *storedDashboard) groupID(i int) string {
	if d != nil && d.group[i] != nil {
		return d.group[i].ID
	}
	return uuid.New().String()
}

func (d *storedDashboard) integrationID(i int) string {
	if d != nil && d.integration[i] != nil {
		return d.integration[i].ID
	}
	return uuid.New().String()
}

func (d *storedDashboard) serviceID(i int) string {
	if d != nil && d.service[i] != nil {
		return d.service[i].ID
	}
	return uuid.New().String()
}

func (d *storedDashboard) widgetID(i int) string {
	if row := d.widgetRow(i); row != nil {
		return row.ID
	}
	return uuid.New().String()
}

func (d *storedDashboard) widgetRow(i int) *models.Widget {
	if d == nil {
		return nil
	}
	return d.widget[i]
}

// pair matches document entities to stored rows with the same key: the k-th
// entity with a key gets the k-th row with it. An empty key matches nothing,
// and neither does an entity or a row that is matched already.
func pair[D, R any](matched map[int]*R, taken map[*R]bool, docs []D, rows []*R, docKey func(D) string, rowKey func(*R) string) {
	byKey := map[string][]*R{}
	for _, row := range rows {
		if key := rowKey(row); key != "" && !taken[row] {
			byKey[key] = append(byKey[key], row)
		}
	}
	for i, doc := range docs {
		key := docKey(doc)
		if matched[i] != nil || key == "" || len(byKey[key]) == 0 {
			continue
		}
		row := byKey[key][0]
		matched[i], taken[row], byKey[key] = row, true, byKey[key][1:]
	}
}

// key joins the fields an entity is recognised by
func key(fields ...string) string {
	for i, field := range fields {
		fields[i] = strings.TrimSpace(field)
	}
	return strings.Join(fields, "\x00")
}

// matchGroups matches the document's groups to the user's own. The default
// group is only ever matched to the default group (byDefault; a merge does
// that itself), the others by name.
func (d *storedDashboard) matchGroups(doc *dashdoc.Document, byDefault bool) {
	d.group = map[int]*models.Group{}
	taken := map[*models.Group]bool{}
	// Every key starts with the board: each board has its own groups
	flag := func(isDefault bool, board string) string {
		if isDefault {
			return key("default", board)
		}
		return ""
	}
	name := func(isDefault bool, board, name string) string {
		if isDefault {
			return ""
		}
		return key(board, name)
	}
	b := d.boards
	if byDefault {
		pair(d.group, taken, doc.Groups, d.groups,
			func(g dashdoc.Group) string { return flag(g.Default, b.id(g.BoardRef)) },
			func(g *models.Group) string { return flag(g.IsDefault, b.of(g.BoardID)) })
	}
	pair(d.group, taken, doc.Groups, d.groups,
		// Without a default group to match, a sync may make a group the default
		func(g dashdoc.Group) string { return name(g.Default && !byDefault, b.id(g.BoardRef), g.Name) },
		func(g *models.Group) string { return name(g.IsDefault, b.of(g.BoardID), g.Name) })
}

// match finds the rows the document's entities already have. They are
// recognised by what a user would call them: an integration by its kind and
// name, a service by its name and URL (or its name alone, so changing the
// URL keeps its history), a widget by its type and title; groups and tiles
// only on the board they are on.
func (d *storedDashboard) match(doc *dashdoc.Document) {
	d.matchGroups(doc, true)
	b := d.boards

	d.matchIntegrations(doc, d.integrations)

	d.service = map[int]*models.Service{}
	takenServices := map[*models.Service]bool{}
	pair(d.service, takenServices, doc.Services, d.services,
		func(s dashdoc.Service) string { return key(b.id(s.BoardRef), s.Name, s.URL) },
		func(s *models.Service) string { return key(b.of(s.BoardID), s.Name, s.URL) })
	pair(d.service, takenServices, doc.Services, d.services,
		func(s dashdoc.Service) string { return key(b.id(s.BoardRef), s.Name) },
		func(s *models.Service) string { return key(b.of(s.BoardID), s.Name) })

	// Widgets of the same type and title are told apart by their position
	// first, so the file's order doesn't swap their rows on every sync
	d.widget = map[int]*models.Widget{}
	takenWidgets := map[*models.Widget]bool{}
	pair(d.widget, takenWidgets, doc.Widgets, d.widgets,
		func(w dashdoc.Widget) string { return key(b.id(w.BoardRef), w.Type, w.Title, strconv.Itoa(w.Position)) },
		func(w *models.Widget) string { return key(b.of(w.BoardID), w.Type, w.Title, strconv.Itoa(w.Position)) })
	pair(d.widget, takenWidgets, doc.Widgets, d.widgets,
		func(w dashdoc.Widget) string { return key(b.id(w.BoardRef), w.Type, w.Title) },
		func(w *models.Widget) string { return key(b.of(w.BoardID), w.Type, w.Title) })
}

// matchIntegrations matches the document's integrations to rows by kind and name
func (d *storedDashboard) matchIntegrations(doc *dashdoc.Document, rows []*models.Integration) {
	d.integration = map[int]*models.Integration{}
	pair(d.integration, map[*models.Integration]bool{}, doc.Integrations, rows,
		func(i dashdoc.Integration) string { return key(i.Kind, i.Name) },
		func(i *models.Integration) string { return key(i.Kind, i.Name) })
}

// loadGroups loads the user's groups on the boards the document works on
func (s *DashboardTransferService) loadGroups(ctx context.Context, userID string, boards *docBoards) (*storedDashboard, error) {
	groups, err := s.Groups.GetAllByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list groups: %w", err)
	}
	groups = slices.DeleteFunc(groups, func(g *models.Group) bool { return !boards.manages(g.BoardID) })
	return &storedDashboard{groups: groups, boards: boards, usedElsewhere: map[string]bool{}}, nil
}

// loadStored loads what the user has on the boards the document works on,
// and all their integrations. What is on their other boards is left out, so
// a sync neither matches nor prunes it.
func (s *DashboardTransferService) loadStored(ctx context.Context, userID string, boards *docBoards) (*storedDashboard, error) {
	stored, err := s.loadGroups(ctx, userID, boards)
	if err != nil {
		return nil, err
	}
	elsewhere := func(boardID string, integrationID *string) bool {
		if boards.manages(boardID) {
			return false
		}
		if integrationID != nil {
			stored.usedElsewhere[*integrationID] = true
		}
		return true
	}
	integrations, err := s.Integrations.ListByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	for i := range integrations {
		stored.integrations = append(stored.integrations, &integrations[i])
	}
	if stored.services, err = s.Services.GetAllByUserID(ctx, userID); err != nil {
		return nil, fmt.Errorf("failed to list services: %w", err)
	}
	stored.services = slices.DeleteFunc(stored.services, func(service *models.Service) bool {
		return elsewhere(service.BoardID, service.IntegrationID)
	})
	widgets, err := s.Widgets.ListByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	for i := range widgets {
		if elsewhere(widgets[i].BoardID, widgets[i].IntegrationID) {
			stored.otherWidgets++
			// A calendar there may list integrations in its config too
			for _, id := range widgettypes.ConfigIntegrationIDs(widgets[i].Type, widgets[i].Config) {
				stored.usedElsewhere[id] = true
			}
			continue
		}
		stored.widgets = append(stored.widgets, &widgets[i])
	}
	return stored, nil
}

// matchGroupNames is the matching of a merge with ImportOptions.MatchGroups.
// The default group is left to the merge itself, which maps it already.
func (s *DashboardTransferService) matchGroupNames(ctx context.Context, userID string, doc *dashdoc.Document, boards *docBoards) (*storedDashboard, error) {
	stored, err := s.loadGroups(ctx, userID, boards)
	if err != nil {
		return nil, err
	}
	stored.matchGroups(doc, false)
	return stored, nil
}

// matchKeptIntegrations is the matching of a replace: it keeps the
// integrations tiles on other boards use, and the document's integration of
// the same kind and name uses the kept one instead of being a second
func (s *DashboardTransferService) matchKeptIntegrations(ctx context.Context, userID string, doc *dashdoc.Document, boards *docBoards) (*storedDashboard, error) {
	stored, err := s.loadStored(ctx, userID, boards)
	if err != nil {
		return nil, err
	}
	var kept []*models.Integration
	for _, integration := range stored.integrations {
		if stored.usedElsewhere[integration.ID] {
			kept = append(kept, integration)
		}
	}
	stored.matchIntegrations(doc, kept)
	return stored, nil
}

// useKeptIntegrations takes the integrations a replace matched out of the
// plan: the document's tiles point at the kept row, which stays as it is.
// The file doesn't get to change its URL: its saved credentials go there.
func (p *importPlan) useKeptIntegrations(stored *storedDashboard) {
	if stored == nil {
		return
	}
	kept := map[string]bool{}
	for _, integration := range stored.integration {
		kept[integration.ID] = true
	}
	rows := p.rows.Integrations[:0]
	for _, integration := range p.rows.Integrations {
		switch {
		case !kept[integration.ID]:
			rows = append(rows, integration)
		case integration.AuthType != models.IntegrationAuthNone:
			p.created.NeedsCredentials-- // the kept one has its own
		}
	}
	p.rows.Integrations = rows
	p.created.Integrations.Created = len(rows)
}

// useStoredGroups takes the groups a merge matched out of the plan: their
// tiles point at the stored group already, which stays as it is
func (p *importPlan) useStoredGroups(stored *storedDashboard) {
	if stored == nil {
		return
	}
	existing := map[string]bool{}
	for _, group := range stored.group {
		existing[group.ID] = true
	}
	kept := p.rows.Groups[:0]
	for _, group := range p.rows.Groups {
		if !existing[group.ID] {
			kept = append(kept, group)
		}
	}
	p.rows.Groups = kept
	p.created.Groups.Created = len(kept)
}

// refuseCredentials keeps credentials out of an import over HTTP: only a
// config file, which the server's operator writes, may carry them
func refuseCredentials(doc *dashdoc.Document) error {
	var issues dashdoc.Issues
	for i, integration := range doc.Integrations {
		if integration.Credentials != nil {
			issues.Add(dashdoc.Path("integrations", i, "credentials"), "credentials can only be given in a config file; enter them after the import")
		}
	}
	return issues.Err()
}

// sameAsJSON compares two values by the JSON they give, so the formatting of
// a stored config or of options doesn't matter
func sameAsJSON(a, b any) bool {
	var decoded [2]any
	for i, value := range []any{a, b} {
		raw, err := json.Marshal(value)
		if err != nil || json.Unmarshal(raw, &decoded[i]) != nil {
			return false
		}
	}
	return reflect.DeepEqual(decoded[0], decoded[1])
}

func byID[R any](rows []*R, id func(*R) string) map[string]*R {
	out := make(map[string]*R, len(rows))
	for _, row := range rows {
		out[id(row)] = row
	}
	return out
}

// Sync makes the user's dashboard match doc while keeping the rows both
// have: a row the document still describes keeps its ID, and with it its
// status history, snapshots and stored credentials. Rows are recognised by
// their natural keys (see storedDashboard.match). Only rows that differ are
// written, so syncing a document twice changes nothing the second time.
//
// With prune, rows the document doesn't have are removed and its preferences
// are applied: the document is the whole dashboard. Without it, the document
// only adds and updates. The user's default group is never removed, and a
// list the document leaves out (or leaves null) is not pruned: only an
// explicit empty list removes everything of a kind, so a file that lost a
// section does not empty the dashboard.
//
// Unlike Apply, Sync takes integration credentials from the document; an
// integration without them keeps the ones it has. It is how a config file is
// applied (internal/configfile) and is not reachable over HTTP.
func (s *DashboardTransferService) Sync(ctx context.Context, userID string, doc *dashdoc.Document, prune bool) (*dashdoc.Summary, error) {
	if err := dashdoc.Validate(doc); err != nil {
		return nil, err
	}
	user, err := s.Users.GetByID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	boards, err := s.resolveBoards(ctx, user, doc)
	if err != nil {
		return nil, err
	}
	stored, err := s.loadStored(ctx, userID, boards)
	if err != nil {
		return nil, err
	}
	stored.match(doc)

	// Preferences are only planned for a replace
	mode := dashdoc.ModeSync
	if prune {
		mode = dashdoc.ModeReplace
	}
	p, err := s.plan(ctx, user, doc, mode, stored, boards)
	if err != nil {
		return nil, err
	}

	run := &syncRun{service: s, ctx: ctx, userID: userID, plan: p, summary: dashdoc.Summary{Mode: dashdoc.ModeSync, NeedsSecrets: p.needsSecrets}}
	run.changes.Credentials = map[string][]byte{}
	run.changes.Create.Boards = boards.changed()
	run.prune.groups, run.prune.integrations = prune && doc.Groups != nil, prune && doc.Integrations != nil
	run.prune.services, run.prune.widgets = prune && doc.Services != nil, prune && doc.Widgets != nil
	summary, err := run.apply(stored)
	if err != nil {
		// Images went to disk first; nothing points at them now
		for _, name := range run.savedIcons {
			s.Images.Remove(ImageIcon, name)
		}
		if run.savedWallpaper != "" {
			s.Images.Remove(ImageWallpaper, run.savedWallpaper)
		}
		return nil, limitIssue(err)
	}
	return summary, nil
}

// syncRun is one Sync: what it writes and what it does once that is stored
type syncRun struct {
	service *DashboardTransferService
	ctx     context.Context
	userID  string
	// prune tells which kinds of rows are removed when the document doesn't
	// have them
	prune   struct{ groups, integrations, services, widgets bool }
	plan    *importPlan
	changes repository.DashboardChanges
	summary dashdoc.Summary

	// savedIcons and savedWallpaper are removed again when the sync fails,
	// staleIcons when it succeeds
	savedIcons, staleIcons []string
	savedWallpaper         string
	// reconnect are the integrations whose session no longer applies
	reconnect []*models.Integration
}

func (r *syncRun) apply(stored *storedDashboard) (*dashdoc.Summary, error) {
	s := r.service
	r.groups(stored)
	if err := r.integrations(stored); err != nil {
		return nil, err
	}
	if err := r.services(stored); err != nil {
		return nil, err
	}
	r.widgets(stored)

	// Without prune, the rows the document doesn't have stay; widgets on
	// boards it doesn't work on always do
	kept := repository.DashboardState{Widgets: stored.otherWidgets}
	if !r.prune.integrations {
		kept.Integrations = len(stored.integrations) - (len(r.plan.rows.Integrations) - r.summary.Integrations.Created)
	}
	if !r.prune.widgets {
		kept.Widgets += len(stored.widgets) - (len(r.plan.rows.Widgets) - r.summary.Widgets.Created)
	}
	if err := r.plan.rows.CheckLimits(kept, dashboardLimits); err != nil {
		return nil, err
	}

	releaseWallpaper, err := r.preferences()
	if err != nil {
		return nil, err
	}
	defer releaseWallpaper()

	if r.changes.Empty() {
		return &r.summary, nil
	}

	// Prepared before the write: ending a session needs the old credentials
	closers := make([]func(), 0, len(r.reconnect))
	for _, integration := range r.reconnect {
		if closeSession := s.IntegrationService.sessionCloser(r.ctx, integration); closeSession != nil {
			closers = append(closers, closeSession)
		}
	}
	if err := s.Dashboard.Sync(r.ctx, r.userID, &r.changes, dashboardLimits); err != nil {
		return nil, err
	}

	// Committed. The rest mirrors what the normal update and delete paths do.
	for _, icon := range r.staleIcons {
		s.Images.Remove(ImageIcon, icon)
	}
	for _, integration := range r.reconnect {
		s.IntegrationService.states.Delete(integration.ID)
	}
	for _, closeSession := range closers {
		closeSession()
	}
	releaseWallpaper()
	if r.changes.Create.Preferences != nil && r.changes.Create.Preferences.ThemeBackground.IsSet() {
		s.Images.PruneWallpapers(r.ctx)
	}
	kick(s.poller)
	return &r.summary, nil
}

func (r *syncRun) groups(stored *storedDashboard) {
	have := byID(stored.groups, func(g *models.Group) string { return g.ID })
	next := 0
	for _, group := range stored.groups {
		next = max(next, group.Position+1)
	}
	planned := map[string]bool{}
	for i, group := range r.plan.rows.Groups {
		planned[group.ID] = true
		old := have[group.ID]
		if old == nil {
			// A replace orders the groups as the document does; otherwise
			// new groups go behind the ones the user has
			group.Position = i
			if !r.prune.groups {
				group.Position, next = next, next+1
			}
			r.changes.Create.Groups = append(r.changes.Create.Groups, group)
			r.summary.Groups.Created++
			continue
		}
		updated := *old
		updated.Name, updated.Color, updated.MonitoringEnabled, updated.IsDefault = group.Name, group.Color, group.MonitoringEnabled, group.IsDefault
		updated.TileStyle = group.TileStyle
		if r.prune.groups {
			updated.Position = i
		}
		if updated != *old {
			updated.UpdatedAt = time.Now()
			r.changes.Update.Groups = append(r.changes.Update.Groups, &updated)
			r.summary.Groups.Updated++
		}
	}
	for _, group := range stored.groups {
		if r.prune.groups && !planned[group.ID] && !group.IsDefault {
			r.changes.Delete.Groups = append(r.changes.Delete.Groups, group.ID)
			r.summary.Groups.Removed++
		}
	}
}

func (r *syncRun) integrations(stored *storedDashboard) error {
	s := r.service.IntegrationService
	have := byID(stored.integrations, func(i *models.Integration) string { return i.ID })
	planned := map[string]bool{}
	for _, integration := range r.plan.rows.Integrations {
		planned[integration.ID] = true
		given := r.plan.credentials[integration.ID]
		needsCredentials := integration.AuthType != models.IntegrationAuthNone
		old := have[integration.ID]
		if old == nil {
			blob, err := s.encryptCredentials(given, integration.ID)
			if err != nil {
				return err
			}
			if blob != nil {
				r.changes.Credentials[integration.ID] = blob
			} else if needsCredentials {
				r.summary.NeedsCredentials++
			}
			r.changes.Create.Integrations = append(r.changes.Create.Integrations, integration)
			r.summary.Integrations.Created++
			continue
		}

		updated := *old
		updated.Name, updated.BaseURL, updated.AuthType = integration.Name, integration.BaseURL, integration.AuthType
		updated.VerifyTLS, updated.RefreshSeconds = integration.VerifyTLS, integration.RefreshSeconds
		if !sameJSON(old.Options, integration.Options) {
			updated.Options = integration.Options
		}
		reconnect := connectionChanged(old, &updated)

		blob, err := r.service.Integrations.GetCredentials(r.ctx, old.ID, r.userID)
		if err != nil {
			return err
		}
		// The credentials to store, when they are not the stored ones
		var replace *[]byte
		switch {
		case !needsCredentials || (given == nil && updated.AuthType != old.AuthType):
			// Credentials of another auth type are of no use
			if blob != nil {
				replace = new([]byte)
			}
		case given != nil:
			current, err := s.decryptCredentials(blob, old.ID)
			if blob == nil || err != nil || current != *given {
				encrypted, err := s.encryptCredentials(given, old.ID)
				if err != nil {
					return err
				}
				replace = &encrypted
			}
		}
		if replace != nil {
			r.changes.Credentials[old.ID], blob = *replace, *replace
		}
		if needsCredentials && blob == nil {
			r.summary.NeedsCredentials++
		}

		if reconnect || replace != nil {
			// The last test result and the session are of the old connection
			updated.LastTestAt, updated.LastTestOK, updated.LastError = nil, nil, nil
			r.reconnect = append(r.reconnect, old)
		}
		if reconnect || replace != nil || updated.Name != old.Name || updated.RefreshSeconds != old.RefreshSeconds {
			r.changes.Update.Integrations = append(r.changes.Update.Integrations, &updated)
			r.summary.Integrations.Updated++
		}
	}
	for _, integration := range stored.integrations {
		// An integration a tile on another board uses is not the document's to remove
		if r.prune.integrations && !planned[integration.ID] && !stored.usedElsewhere[integration.ID] {
			r.changes.Delete.Integrations = append(r.changes.Delete.Integrations, integration.ID)
			r.reconnect = append(r.reconnect, integration)
			r.summary.Integrations.Removed++
		}
	}
	return nil
}

func (r *syncRun) services(stored *storedDashboard) error {
	images := r.service.Images
	have := byID(stored.services, func(s *models.Service) string { return s.ID })
	uploaded := func(service *models.Service) bool {
		return service.IconType == models.IconTypeImageUpload && service.IconImagePath != ""
	}
	planned := map[string]bool{}
	for index, service := range r.plan.rows.Services {
		planned[service.ID] = true
		old := have[service.ID]
		if icon := r.plan.icons[index]; icon != nil {
			// The same image again keeps the file the service has
			if current, err := loadIcon(images, old); err == nil && bytes.Equal(current, icon) {
				service.IconImagePath = old.IconImagePath
			} else {
				name, _, err := images.Save(ImageIcon, icon)
				if err != nil {
					return fmt.Errorf("failed to store icon: %w", err)
				}
				r.savedIcons = append(r.savedIcons, name)
				service.IconImagePath = name
			}
		}
		if old == nil {
			r.changes.Create.Services = append(r.changes.Create.Services, service)
			r.summary.Services.Created++
			continue
		}
		updated := *old
		updated.Name, updated.URL, updated.Description = service.Name, service.URL, service.Description
		updated.Icon, updated.IconType, updated.IconImagePath = service.Icon, service.IconType, service.IconImagePath
		updated.CardSize, updated.Position, updated.MonitoringEnabled = service.CardSize, service.Position, service.MonitoringEnabled
		updated.GroupID, updated.IntegrationID = service.GroupID, service.IntegrationID
		if !sameAsJSON(updated, old) {
			updated.UpdatedAt = time.Now()
			r.changes.Update.Services = append(r.changes.Update.Services, &updated)
			r.summary.Services.Updated++
			if uploaded(old) && old.IconImagePath != updated.IconImagePath {
				r.staleIcons = append(r.staleIcons, old.IconImagePath)
			}
		}
	}
	for _, service := range stored.services {
		if r.prune.services && !planned[service.ID] {
			r.changes.Delete.Services = append(r.changes.Delete.Services, service.ID)
			r.summary.Services.Removed++
			if uploaded(service) {
				r.staleIcons = append(r.staleIcons, service.IconImagePath)
			}
		}
	}
	return nil
}

// loadIcon reads the uploaded icon of a stored service
func loadIcon(images ImageStore, service *models.Service) ([]byte, error) {
	if service == nil || service.IconType != models.IconTypeImageUpload || service.IconImagePath == "" {
		return nil, errors.New("no uploaded icon")
	}
	return images.Load(ImageIcon, service.IconImagePath)
}

func (r *syncRun) widgets(stored *storedDashboard) {
	have := byID(stored.widgets, func(w *models.Widget) string { return w.ID })
	planned := map[string]bool{}
	for _, widget := range r.plan.rows.Widgets {
		planned[widget.ID] = true
		old := have[widget.ID]
		if old == nil {
			r.changes.Create.Widgets = append(r.changes.Create.Widgets, widget)
			r.summary.Widgets.Created++
			continue
		}
		updated := *old
		updated.Title, updated.Config, updated.CardSize = widget.Title, widget.Config, widget.CardSize
		updated.Position, updated.RefreshSeconds, updated.Enabled = widget.Position, widget.RefreshSeconds, widget.Enabled
		updated.GroupID, updated.IntegrationID = widget.GroupID, widget.IntegrationID
		if !sameAsJSON(updated, old) {
			r.changes.Update.Widgets = append(r.changes.Update.Widgets, &updated)
			r.summary.Widgets.Updated++
		}
	}
	for _, widget := range stored.widgets {
		if r.prune.widgets && !planned[widget.ID] {
			r.changes.Delete.Widgets = append(r.changes.Delete.Widgets, widget.ID)
			r.summary.Widgets.Removed++
		}
	}
}

// preferences puts the planned preferences in the changes when they differ
// from the ones the user has. The returned func releases a stored wallpaper
// (see ImageStore.Save).
func (r *syncRun) preferences() (release func(), err error) {
	s, request := r.service, r.plan.rows.Preferences
	release = func() {}
	if request == nil {
		return release, nil
	}
	current, err := s.Preferences.GetByUserID(r.ctx, r.userID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("failed to get preferences: %w", err)
	}

	if wallpaper := r.plan.wallpaper; wallpaper != nil {
		// The same image again keeps the file the user has
		var stored []byte
		if current != nil && current.ThemeBackground != nil && isUploadedPath(*current.ThemeBackground) {
			stored, _ = s.Images.Load(ImageWallpaper, *current.ThemeBackground)
		}
		if !bytes.Equal(stored, wallpaper) {
			path, releaseSaved, err := s.Images.Save(ImageWallpaper, wallpaper)
			if err != nil {
				return nil, fmt.Errorf("failed to store wallpaper: %w", err)
			}
			release, r.savedWallpaper = releaseSaved, path
			request.ThemeBackground = models.NullableString{Value: &path, Set: true}
		}
	}

	if current != nil && sameAsJSON(withPreferences(*current, request), current) {
		return release, nil
	}
	r.changes.Create.Preferences = request
	r.summary.Preferences = true
	return release, nil
}

// withPreferences is p after request is applied to it
func withPreferences(p models.UserPreferences, request *models.PreferencesUpdateRequest) *models.UserPreferences {
	set(&p.ThemeMode, request.ThemeMode)
	set(&p.OpenInNewTab, request.OpenInNewTab)
	set(&p.EnableCardResizing, request.EnableCardResizing)
	set(&p.EnableServiceGrouping, request.EnableServiceGrouping)
	set(&p.CardScale, request.CardScale)
	set(&p.ViewMode, request.ViewMode)
	set(&p.WallpaperBlur, request.WallpaperBlur)
	set(&p.WallpaperDim, request.WallpaperDim)
	set(&p.CardOpacity, request.CardOpacity)
	set(&p.CardBlur, request.CardBlur)
	set(&p.LayoutMode, request.LayoutMode)
	set(&p.StatusStrip, request.StatusStrip)
	set(&p.ThemePreset, request.ThemePreset)
	set(&p.ThemeFont, request.ThemeFont)
	set(&p.KpiMono, request.KpiMono)
	set(&p.IconSource, request.IconSource)
	set(&p.CustomCSS, request.CustomCSS)
	// Not given, a document's group display follows the layout again
	if request.GroupDisplay != nil || request.FollowLayout {
		p.GroupDisplay = request.GroupDisplay
	}
	if request.ThemeBackground.IsSet() {
		p.ThemeBackground = request.ThemeBackground.Value
	}
	if request.ThemeAccentColor.IsSet() {
		p.ThemeAccentColor = request.ThemeAccentColor.Value
	}
	return &p
}

func set[T any](field *T, value *T) {
	if value != nil {
		*field = *value
	}
}

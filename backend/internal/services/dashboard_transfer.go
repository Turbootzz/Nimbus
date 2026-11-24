package services

import (
	"bytes"
	"cmp"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"slices"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/nimbus/backend/internal/dashdoc"
	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/repository"
	"github.com/nimbus/backend/internal/utils"
	"github.com/nimbus/backend/internal/widgets"
)

const (
	// MaxDocumentSize is the largest document an import takes
	// (handlers.MaxImportSize); an export leaves out images to stay under it
	MaxDocumentSize = 10 * 1024 * 1024
	// maxEmbeddedBytes is how many bytes of images one export loads: as
	// base64, more could never fit in a document
	maxEmbeddedBytes = MaxDocumentSize / 4 * 3
	// pendingIcon stands in for the file name of an embedded icon until the
	// image is stored
	pendingIcon = "pending"
)

// ImageKind tells an ImageStore which uploads are meant
type ImageKind int

const (
	ImageIcon ImageKind = iota
	ImageWallpaper
)

// ImageStore reads and writes uploaded images. Service icons are named by
// their file name, wallpapers by their URL path (as theme_background has it).
type ImageStore interface {
	// Check validates an image (size, content type) without storing it;
	// its error is shown to the user
	Check(kind ImageKind, data []byte) error
	// Save stores an image and returns its name. A stored wallpaper is only
	// safe from cleanup until release is called.
	Save(kind ImageKind, data []byte) (name string, release func(), err error)
	Load(kind ImageKind, name string) ([]byte, error)
	Remove(kind ImageKind, name string)
	// PruneWallpapers deletes wallpapers no preferences point at anymore
	PruneWallpapers(ctx context.Context)
}

// UserGetter is the part of UserRepository the transfer needs
type UserGetter interface {
	GetByID(id string) (*models.User, error)
}

// DashboardTransferDeps is what a DashboardTransferService works with
type DashboardTransferDeps struct {
	Dashboard *repository.DashboardRepository
	// Boards is nil for one dashboard per user: documents then have no boards
	Boards             *repository.BoardRepository
	Groups             *repository.GroupRepository
	Services           repository.ServiceRepositoryInterface
	Widgets            repository.WidgetRepositoryInterface
	Integrations       repository.IntegrationRepositoryInterface
	Preferences        *repository.PreferencesRepository
	Users              UserGetter
	IntegrationService *IntegrationService
	Images             ImageStore
}

// StatusPage is the public status page as a replace import sees it. The page
// shows one group, and a replace removes every group of the user and creates
// new ones: the page follows its group to the new one with the same name.
type StatusPage interface {
	Settings(ctx context.Context) (models.StatusPageSettings, error)
	UseGroup(ctx context.Context, userID, groupID string) error
}

// DashboardTransferService exports a user's dashboard as a dashdoc.Document
// and imports one. It is the single way a document gets into the database:
//
//   - Export(ctx, userID) reads the dashboard into a document.
//   - Apply(ctx, userID, doc, mode) validates a document and writes it, in
//     one transaction: all of it or nothing.
//   - Preview is Apply without the writes, for showing what would happen.
//
// Apply takes no request: it looks up whether the user is an admin itself,
// so it can also run at boot or from an importer. Everything it creates goes
// through the checks a normal create runs (admin-only kinds and types, URLs,
// widget configs, per-user limits).
type DashboardTransferService struct {
	DashboardTransferDeps
	poller     Poller
	statusPage StatusPage
}

func NewDashboardTransferService(deps DashboardTransferDeps) *DashboardTransferService {
	return &DashboardTransferService{DashboardTransferDeps: deps}
}

// SetPoller lets an import start polling what it created
func (s *DashboardTransferService) SetPoller(p Poller) {
	s.poller = p
}

// SetStatusPage lets a replace keep the status page on its group
func (s *DashboardTransferService) SetStatusPage(page StatusPage) {
	s.statusPage = page
}

// Export reads the user's dashboard into a document. Its content only
// depends on what the dashboard shows, never on IDs or timestamps, so
// exporting an imported export gives the same document. Credentials and
// widget secrets are left out.
func (s *DashboardTransferService) Export(ctx context.Context, userID string) (*dashdoc.Document, error) {
	groups, err := s.Groups.GetAllByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list groups: %w", err)
	}
	integrationList, err := s.Integrations.ListByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	serviceList, err := s.Services.GetAllByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list services: %w", err)
	}
	widgetList, err := s.Widgets.ListByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	doc := &dashdoc.Document{
		Version:      dashdoc.Version,
		Groups:       make([]dashdoc.Group, 0, len(groups)),
		Integrations: make([]dashdoc.Integration, 0, len(integrationList)),
		Services:     make([]dashdoc.Service, 0, len(serviceList)),
		Widgets:      make([]dashdoc.Widget, 0, len(widgetList)),
	}
	budget := maxEmbeddedBytes
	// Everything is on the default board unless it says otherwise, so a
	// dashboard with one board exports as it did before boards existed
	boardRefs, err := s.exportBoards(ctx, userID, doc)
	if err != nil {
		return nil, err
	}

	// Sorted by content: the order the database returns ties in changes
	// with every import
	slices.SortStableFunc(groups, func(a, b *models.Group) int {
		return cmp.Or(cmp.Compare(a.Position, b.Position), cmp.Compare(a.Name, b.Name), cmp.Compare(a.Color, b.Color))
	})
	groupRefs, names := map[string]string{}, refNames{}
	for _, group := range groups {
		ref := names.next(group.Name, "group")
		groupRefs[group.ID] = ref
		doc.Groups = append(doc.Groups, dashdoc.Group{
			Ref:               ref,
			Name:              group.Name,
			Color:             group.Color,
			BoardRef:          boardRefs[group.BoardID],
			Default:           group.IsDefault,
			MonitoringEnabled: &group.MonitoringEnabled,
			TileStyle:         group.TileStyle,
		})
	}

	slices.SortStableFunc(integrationList, func(a, b models.Integration) int {
		return cmp.Or(cmp.Compare(a.Name, b.Name), cmp.Compare(a.Kind, b.Kind), cmp.Compare(a.BaseURL, b.BaseURL))
	})
	integrationRefs, names := map[string]string{}, refNames{}
	for _, integration := range integrationList {
		ref := names.next(integration.Name, integration.Kind)
		integrationRefs[integration.ID] = ref
		doc.Integrations = append(doc.Integrations, dashdoc.Integration{
			Ref:              ref,
			Kind:             integration.Kind,
			Name:             integration.Name,
			BaseURL:          integration.BaseURL,
			AuthType:         integration.AuthType,
			VerifyTLS:        &integration.VerifyTLS,
			Options:          integration.Options,
			RefreshSeconds:   &integration.RefreshSeconds,
			NeedsCredentials: integration.AuthType != models.IntegrationAuthNone,
		})
	}

	slices.SortStableFunc(serviceList, func(a, b *models.Service) int {
		return cmp.Or(cmp.Compare(a.Position, b.Position), cmp.Compare(a.Name, b.Name), cmp.Compare(a.URL, b.URL))
	})
	names = refNames{}
	for _, service := range serviceList {
		out := dashdoc.Service{
			Ref:               names.next(service.Name, "service"),
			Name:              service.Name,
			URL:               service.URL,
			Description:       service.Description,
			Icon:              service.Icon,
			IconType:          service.IconType,
			CardSize:          service.CardSize,
			Position:          service.Position,
			GroupRef:          refOf(groupRefs, service.GroupID),
			IntegrationRef:    refOf(integrationRefs, service.IntegrationID),
			MonitoringEnabled: &service.MonitoringEnabled,
			BoardRef:          boardRefs[service.BoardID],
		}
		switch service.IconType {
		case models.IconTypeImageURL:
			out.IconURL = service.IconImagePath
		case models.IconTypeImageUpload:
			if out.IconData = s.embed(ImageIcon, service.IconImagePath, &budget); out.IconData == "" {
				// Without its image the service falls back to its emoji
				out.IconType = models.IconTypeEmoji
				doc.OmittedImages++
			}
		}
		doc.Services = append(doc.Services, out)
	}

	slices.SortStableFunc(widgetList, func(a, b models.Widget) int {
		return cmp.Or(cmp.Compare(a.Position, b.Position), cmp.Compare(a.Type, b.Type),
			cmp.Compare(a.Title, b.Title), cmp.Compare(string(a.Config), string(b.Config)))
	})
	widgetRefs, names := map[string]string{}, refNames{}
	for _, widget := range widgetList {
		widgetType, known := widgets.Get(widget.Type)
		// A widget that lost its integration shows nothing and no import
		// would take it; leaving it out keeps the export importable
		if !known || (len(widgetType.Meta().IntegrationKinds) > 0 && refOf(integrationRefs, widget.IntegrationID) == "") {
			continue
		}
		out := dashdoc.Widget{
			Type:           widget.Type,
			Title:          widget.Title,
			Config:         widget.Config,
			CardSize:       widget.CardSize,
			Position:       widget.Position,
			RefreshSeconds: &widget.RefreshSeconds,
			Enabled:        &widget.Enabled,
			GroupRef:       refOf(groupRefs, widget.GroupID),
			IntegrationRef: refOf(integrationRefs, widget.IntegrationID),
			BoardRef:       boardRefs[widget.BoardID],
		}
		validate := widgetType.Validate
		if secret, ok := widgetType.(widgets.SecretConfig); ok {
			out.Config = secret.Redact(out.Config)
		}
		if masked, ok := widgetType.(widgets.MaskedConfig); ok {
			out.NeedsSecrets = masked.HasMasks(out.Config)
			validate = masked.ValidateMasked
		}
		if lister, ok := widgetType.(widgets.ConfigIntegrations); ok {
			// A config lists integration IDs; the document lists refs
			ids, refs := lister.IntegrationIDs(out.Config), []string{}
			for _, id := range ids {
				if ref, ok := integrationRefs[id]; ok {
					refs = append(refs, ref)
				}
			}
			if out.Config, err = lister.SetIntegrationIDs(out.Config, refs); err != nil {
				return nil, fmt.Errorf("failed to export widget %s: %w", widget.ID, err)
			}
			// All of them deleted: left out too when nothing else is shown
			if len(ids) > 0 && len(refs) == 0 {
				if _, err := validate(out.Config); err != nil {
					continue
				}
			}
		}
		out.Ref = names.next(widget.Title, widget.Type)
		widgetRefs[widget.ID] = out.Ref
		doc.Widgets = append(doc.Widgets, out)
	}

	// Old reorders stored positions a document can't hold; keep the order
	positions := make([]*int, 0, len(doc.Services)+len(doc.Widgets))
	tooHigh := false
	for i := range doc.Services {
		positions = append(positions, &doc.Services[i].Position)
	}
	for i := range doc.Widgets {
		positions = append(positions, &doc.Widgets[i].Position)
	}
	for _, position := range positions {
		tooHigh = tooHigh || *position > dashdoc.MaxPosition
	}
	if tooHigh {
		rankPositions(positions)
	}

	preferences, err := s.Preferences.GetByUserID(ctx, userID)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		// Never saved: nothing to carry over
	case err != nil:
		return nil, fmt.Errorf("failed to get preferences: %w", err)
	default:
		doc.Preferences = s.exportPreferences(preferences, integrationRefs, widgetRefs, &budget, &doc.OmittedImages)
	}
	if err := fitImages(doc); err != nil {
		return nil, err
	}
	return doc, nil
}

// fitImages leaves embedded images out, the wallpaper first and then icons
// from the last service, until the document fits in an import as the browser
// saves it: indented (DashboardExport.tsx). Text alone that is too large
// stays too large.
func fitImages(doc *dashdoc.Document) error {
	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to measure the export: %w", err)
	}
	// The margin covers omitted_images growing
	size := len(raw) + 64
	omit := func(data *string) {
		size -= len(*data)
		*data = ""
		doc.OmittedImages++
	}
	if prefs := doc.Preferences; size > MaxDocumentSize && prefs != nil && prefs.WallpaperData != "" {
		// Left out, so an import keeps the wallpaper the user has
		omit(&prefs.WallpaperData)
	}
	for i := len(doc.Services) - 1; i >= 0 && size > MaxDocumentSize; i-- {
		if service := &doc.Services[i]; service.IconData != "" {
			omit(&service.IconData)
			service.IconType = models.IconTypeEmoji
		}
	}
	return nil
}

func (s *DashboardTransferService) exportPreferences(p *models.UserPreferences, integrationRefs, widgetRefs map[string]string, budget, omitted *int) *dashdoc.Preferences {
	out := &dashdoc.Preferences{
		ThemeMode:             &p.ThemeMode,
		ThemeAccentColor:      new(string),
		Language:              new(string),
		OpenInNewTab:          &p.OpenInNewTab,
		EnableCardResizing:    &p.EnableCardResizing,
		EnableServiceGrouping: &p.EnableServiceGrouping,
		CardScale:             &p.CardScale,
		ViewMode:              &p.ViewMode,
		WallpaperBlur:         &p.WallpaperBlur,
		WallpaperDim:          &p.WallpaperDim,
		CardOpacity:           &p.CardOpacity,
		CardBlur:              &p.CardBlur,
		LayoutMode:            &p.LayoutMode,
		GroupDisplay:          p.GroupDisplay,
		StatusStrip:           &dashdoc.StatusStrip{Enabled: p.StatusStrip.Enabled, Chips: []dashdoc.Chip{}},
		ThemePreset:           &p.ThemePreset,
		ThemeFont:             &p.ThemeFont,
		KpiMono:               &p.KpiMono,
		IconSource:            &p.IconSource,
		CustomCSS:             &p.CustomCSS,
	}
	if p.ThemeAccentColor != nil {
		out.ThemeAccentColor = p.ThemeAccentColor
	}
	if p.Language != nil {
		out.Language = p.Language
	}

	switch background := p.ThemeBackground; {
	case background == nil || *background == "":
		out.ThemeBackground = new(string)
	case isUploadedPath(*background):
		// Left out when it can't be embedded, so an import keeps the
		// wallpaper the user has instead of clearing it
		if out.WallpaperData = s.embed(ImageWallpaper, *background, budget); out.WallpaperData == "" {
			*omitted++
		}
	default:
		out.ThemeBackground = background
	}

	for _, chip := range p.StatusStrip.Chips {
		refs := integrationRefs
		if chip.Source == dashdoc.ChipSourceWidget {
			refs = widgetRefs
		}
		// A chip of something deleted since shows nothing; drop it
		if ref, ok := refs[chip.ID]; ok {
			out.StatusStrip.Chips = append(out.StatusStrip.Chips, dashdoc.Chip{Source: chip.Source, Ref: ref, KPI: chip.KPI})
		}
	}
	return out
}

// isUploadedPath tells an uploaded wallpaper (a local path) from an image URL
func isUploadedPath(background string) bool {
	return strings.HasPrefix(background, "/")
}

// embed returns a stored image as base64, or "" when it is missing or no
// longer fits in the budget
func (s *DashboardTransferService) embed(kind ImageKind, name string, budget *int) string {
	data, err := s.Images.Load(kind, name)
	if err != nil || len(data) == 0 || len(data) > *budget {
		return ""
	}
	*budget -= len(data)
	return base64.StdEncoding.EncodeToString(data)
}

func refOf(refs map[string]string, id *string) string {
	if id == nil {
		return ""
	}
	return refs[*id]
}

// refNames hands out readable, unique refs: "sonarr", "sonarr-2", ...
type refNames map[string]bool

func (n refNames) next(name, fallback string) string {
	base := slug(name)
	if base == "" {
		base = slug(fallback)
	}
	ref := base
	for i := 2; n[ref]; i++ {
		ref = fmt.Sprintf("%s-%d", base, i)
	}
	n[ref] = true
	return ref
}

// slug lowercases s and turns everything but letters and digits into dashes
func slug(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case b.Len() > 0 && !strings.HasSuffix(b.String(), "-"):
			b.WriteByte('-')
		}
		if b.Len() >= 60 {
			break
		}
	}
	return strings.Trim(b.String(), "-")
}

// Apply validates doc and writes it to the user's dashboard with the given
// mode (see dashdoc.ModeMerge and dashdoc.ModeReplace). It is atomic: when
// anything fails, the dashboard is as it was. A document that is not valid
// gives a *dashdoc.ValidationError and changes nothing.
func (s *DashboardTransferService) Apply(ctx context.Context, userID string, doc *dashdoc.Document, mode dashdoc.Mode) (*dashdoc.Summary, error) {
	return s.Import(ctx, userID, doc, ImportOptions{Mode: mode})
}

// Preview does everything Apply does except writing, and returns the
// summary Apply would return
func (s *DashboardTransferService) Preview(ctx context.Context, userID string, doc *dashdoc.Document, mode dashdoc.Mode) (*dashdoc.Summary, error) {
	return s.Import(ctx, userID, doc, ImportOptions{Mode: mode, DryRun: true})
}

// ImportOptions is how Import applies a document
type ImportOptions struct {
	Mode dashdoc.Mode
	// DryRun writes nothing (Preview)
	DryRun bool
	// MatchGroups makes a merge put the tiles of a document group into the
	// group the user already has with that name, instead of creating a
	// second one. The import wizard maps groups with it.
	MatchGroups bool
}

// Import is Apply or Preview with all options
func (s *DashboardTransferService) Import(ctx context.Context, userID string, doc *dashdoc.Document, opts ImportOptions) (*dashdoc.Summary, error) {
	return s.run(ctx, userID, doc, opts)
}

var dashboardLimits = repository.DashboardLimits{Integrations: maxIntegrationsPerUser, Widgets: maxWidgetsPerUser}

func (s *DashboardTransferService) run(ctx context.Context, userID string, doc *dashdoc.Document, opts ImportOptions) (*dashdoc.Summary, error) {
	mode, dryRun := opts.Mode, opts.DryRun
	if _, ok := dashdoc.ParseMode(string(mode)); !ok {
		return nil, invalid("Mode must be merge or replace")
	}
	if err := dashdoc.Validate(doc); err != nil {
		return nil, err
	}
	if raw, err := json.Marshal(doc); err != nil || hasNUL(raw) {
		return nil, (dashdoc.Issues{{Path: "document", Message: `must be valid JSON without the character \u0000, which can not be stored`}}).Err()
	}
	if err := refuseCredentials(doc); err != nil {
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
	var stored *storedDashboard
	if opts.MatchGroups && mode == dashdoc.ModeMerge {
		if stored, err = s.matchGroupNames(ctx, userID, doc, boards); err != nil {
			return nil, err
		}
	}
	// Planned before the transaction: validating URLs looks up their hosts
	p, err := s.plan(ctx, user, doc, mode, stored, boards)
	if err != nil {
		return nil, err
	}
	p.useStoredGroups(stored)
	replace := mode == dashdoc.ModeReplace
	var move *statusPageMove
	// The page shows an admin's group: a non-admin's replace never moves it
	if replace && user.Role == "admin" {
		if move, err = s.statusPageMove(ctx, userID, p.rows.Groups, boards); err != nil {
			return nil, err
		}
	}

	if dryRun {
		before, err := s.Dashboard.Scope(ctx, userID, boards.listed())
		if err != nil {
			return nil, err
		}
		kept := before.State
		kept.Widgets += before.Kept.Widgets
		if replace {
			kept = before.Kept
		}
		if err := limitIssue(p.rows.CheckLimits(kept, dashboardLimits)); err != nil {
			return nil, err
		}
		return move.note(p.summary(mode, true, before)), nil
	}

	// Images go to disk first; whatever fails after that removes them again
	var saved []func()
	undo := func() {
		for _, remove := range saved {
			remove()
		}
	}
	for index, data := range p.icons {
		name, _, err := s.Images.Save(ImageIcon, data)
		if err != nil {
			undo()
			return nil, fmt.Errorf("failed to store icon: %w", err)
		}
		saved = append(saved, func() { s.Images.Remove(ImageIcon, name) })
		p.rows.Services[index].IconImagePath = name
	}
	releaseWallpaper := func() {}
	if p.wallpaper != nil {
		path, release, err := s.Images.Save(ImageWallpaper, p.wallpaper)
		if err != nil {
			undo()
			return nil, fmt.Errorf("failed to store wallpaper: %w", err)
		}
		// Held until the preferences point at it, or the sweep would take it
		releaseWallpaper = release
		defer release()
		saved = append(saved, func() { s.Images.Remove(ImageWallpaper, path) })
		p.rows.Preferences.ThemeBackground = models.NullableString{Value: &path, Set: true}
	}

	forgetIntegrations := func() {}
	if replace {
		if forgetIntegrations, err = s.IntegrationService.forgetUser(ctx, userID); err != nil {
			undo()
			return nil, err
		}
	}

	applied, err := s.Dashboard.Apply(ctx, userID, replace, &p.rows, dashboardLimits)
	if err != nil {
		undo()
		return nil, limitIssue(err)
	}

	// Committed. The rest mirrors what the normal delete and save paths do.
	for _, icon := range applied.RemovedIcons {
		s.Images.Remove(ImageIcon, icon)
	}
	forgetIntegrations()
	releaseWallpaper()
	if p.rows.Preferences != nil && p.rows.Preferences.ThemeBackground.IsSet() {
		s.Images.PruneWallpapers(ctx)
	}
	kick(s.poller)
	if move != nil && move.to != nil {
		if err := s.statusPage.UseGroup(ctx, userID, move.to.ID); err != nil {
			log.Printf("Import: failed to keep the status page on group %q: %v", move.from, err)
			move.to = nil // it shows the removed group: offline
		}
	}
	return move.note(p.summary(mode, false, applied.Scope)), nil
}

// statusPageMove is where a replace takes the status page: from the name of
// the user's group it shows to the planned group with that name, nil if none
type statusPageMove struct {
	from    string
	to      *models.Group
	enabled bool
}

// statusPageMove returns nil when the page shows none of the user's groups
// on a board the document replaces
func (s *DashboardTransferService) statusPageMove(ctx context.Context, userID string, planned []*models.Group, boards *docBoards) (*statusPageMove, error) {
	if s.statusPage == nil {
		return nil, nil
	}
	settings, err := s.statusPage.Settings(ctx)
	if err != nil || settings.GroupID == "" {
		return nil, err
	}
	groups, err := s.Groups.GetAllByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list groups: %w", err)
	}
	i := slices.IndexFunc(groups, func(g *models.Group) bool { return g.ID == settings.GroupID })
	if i < 0 || !boards.manages(groups[i].BoardID) {
		return nil, nil
	}
	move := &statusPageMove{from: groups[i].Name, enabled: settings.Enabled}
	board := boards.of(groups[i].BoardID)
	if j := slices.IndexFunc(planned, func(g *models.Group) bool { return g.BoardID == board && key(g.Name) == key(move.from) }); j >= 0 {
		move.to = planned[j]
	}
	return move, nil
}

// note tells the summary what happens to a page that is on
func (m *statusPageMove) note(summary *dashdoc.Summary) *dashdoc.Summary {
	if m != nil && m.enabled {
		summary.StatusPage = &dashdoc.StatusPageImpact{Group: m.from, Kept: m.to != nil}
	}
	return summary
}

// hasNUL reports a \u0000 escape in JSON that is not itself escaped.
// Postgres stores that character in neither text nor JSONB.
func hasNUL(raw []byte) bool {
	const escape = `\u0000`
	for from := 0; ; {
		at := bytes.Index(raw[from:], []byte(escape))
		if at < 0 {
			return false
		}
		at += from
		backslashes := 0
		for at-backslashes > 0 && raw[at-backslashes-1] == '\\' {
			backslashes++
		}
		if backslashes%2 == 0 {
			return true
		}
		from = at + 1
	}
}

// limitIssue turns a per-user limit error into a validation error
func limitIssue(err error) error {
	var issues dashdoc.Issues
	switch {
	case errors.Is(err, repository.ErrIntegrationLimitReached):
		issues.Add("integrations", "the import would go over the limit of %d integrations", maxIntegrationsPerUser)
	case errors.Is(err, repository.ErrWidgetLimitReached):
		issues.Add("widgets", "the import would go over the limit of %d widgets", maxWidgetsPerUser)
	case errors.Is(err, repository.ErrBoardLimitReached):
		issues.Add("boards", "the import would go over the limit of %d boards", models.MaxBoardsPerUser)
	default:
		return err
	}
	return issues.Err()
}

// importPlan is a validated document, ready to be written
type importPlan struct {
	rows repository.DashboardRows
	// icons are the embedded icons to store, by index in rows.Services
	icons map[int][]byte
	// wallpaper is the embedded wallpaper to store, if any
	wallpaper []byte
	// defaultGroupBoards are the boards the document has a default group
	// for: a merge maps each onto the default group its board has
	defaultGroupBoards []string
	// credentials are the ones a config file gives, by integration ID
	credentials      map[string]*models.IntegrationCredentials
	created          dashdoc.Summary
	needsCredentials int
	needsSecrets     int
}

func (p *importPlan) summary(mode dashdoc.Mode, dryRun bool, before *repository.DashboardScope) *dashdoc.Summary {
	summary := p.created
	summary.Mode, summary.DryRun = mode, dryRun
	if mode == dashdoc.ModeReplace {
		summary.Groups.Removed = before.State.Groups
		summary.Services.Removed = before.State.Services
		summary.Widgets.Removed = before.State.Widgets
		summary.Integrations.Removed = before.State.Integrations - before.Kept.Integrations
		return &summary
	}
	for _, board := range p.defaultGroupBoards {
		if before.DefaultGroups[cmp.Or(board, before.DefaultBoardID)] != "" {
			summary.Groups.Created--
		}
	}
	return &summary
}

// importedGroups and importedIntegrations let the widget checks find what
// the same document creates, before it is in the database
type importedGroups map[string]*models.Group

func (m importedGroups) GetByID(_ context.Context, id string) (*models.Group, error) {
	if group, ok := m[id]; ok {
		return group, nil
	}
	return nil, sql.ErrNoRows
}

type importedIntegrations map[string]*models.Integration

func (m importedIntegrations) GetByID(_ context.Context, id, userID string) (*models.Integration, error) {
	if integration, ok := m[id]; ok && integration.UserID == userID {
		return integration, nil
	}
	return nil, repository.ErrIntegrationNotFound
}

var preferencesValidator = validator.New()

// plan turns a structurally valid document into rows with new IDs. Every
// entity goes through the checks its normal create runs; all problems are
// collected into one *dashdoc.ValidationError. An entity that stored has a
// row for gets the ID of that row (nil: none has).
//
// boards are the document's boards (resolveBoards); nil without boards.
func (s *DashboardTransferService) plan(ctx context.Context, user *models.User, doc *dashdoc.Document, mode dashdoc.Mode, stored *storedDashboard, boards *docBoards) (*importPlan, error) {
	isAdmin := user.Role == "admin"
	var issues dashdoc.Issues
	p := &importPlan{icons: map[int][]byte{}, credentials: map[string]*models.IntegrationCredentials{}}
	if boards != nil {
		p.rows.Boards = boards.rows
	}

	// report records a validation error at path; anything else is returned
	report := func(path string, err error) error {
		var vErr *ValidationError
		if !errors.As(err, &vErr) {
			return err
		}
		issues.Add(path, "%s", vErr.Message)
		return nil
	}

	groups, groupIDs := importedGroups{}, map[string]string{}
	for i, in := range doc.Groups {
		group, err := NewGroupFromRequest(&models.GroupCreateRequest{Name: in.Name, Color: in.Color, MonitoringEnabled: in.MonitoringEnabled, TileStyle: in.TileStyle})
		if err != nil {
			if err := report(dashdoc.Path("groups", i, ""), err); err != nil {
				return nil, err
			}
			continue
		}
		group.ID, group.UserID, group.IsDefault = stored.groupID(i), user.ID, in.Default
		group.BoardID = boards.id(in.BoardRef)
		groups[group.ID], groupIDs[in.Ref] = group, group.ID
		if in.Default {
			p.defaultGroupBoards = append(p.defaultGroupBoards, group.BoardID)
		}
		p.rows.Groups = append(p.rows.Groups, group)
	}

	integrationList, integrationIDs := importedIntegrations{}, map[string]string{}
	for i, in := range doc.Integrations {
		integration, err := s.IntegrationService.buildImported(&models.IntegrationRequest{
			Kind:           in.Kind,
			Name:           in.Name,
			BaseURL:        in.BaseURL,
			AuthType:       in.AuthType,
			VerifyTLS:      in.VerifyTLS,
			Options:        in.Options,
			RefreshSeconds: in.RefreshSeconds,
		}, isAdmin)
		if err != nil {
			if err := report(dashdoc.Path("integrations", i, ""), err); err != nil {
				return nil, err
			}
			continue
		}
		integration.ID, integration.UserID = stored.integrationID(i), user.ID
		integrationList[integration.ID], integrationIDs[in.Ref] = integration, integration.ID
		switch {
		case in.Credentials != nil:
			credentials, err := resolveCredentials(integration.AuthType, (*models.IntegrationCredentials)(in.Credentials), nil)
			if err != nil {
				if err := report(dashdoc.Path("integrations", i, "credentials"), err); err != nil {
					return nil, err
				}
				continue
			}
			p.credentials[integration.ID] = credentials
		case integration.AuthType != models.IntegrationAuthNone:
			p.needsCredentials++
		}
		p.rows.Integrations = append(p.rows.Integrations, integration)
	}

	for i, in := range doc.Services {
		req := &models.ServiceCreateRequest{
			Name:              in.Name,
			URL:               in.URL,
			Icon:              in.Icon,
			IconType:          in.IconType,
			IconImagePath:     in.IconURL,
			Description:       in.Description,
			CardSize:          in.CardSize,
			MonitoringEnabled: in.MonitoringEnabled,
		}
		var icon []byte
		if in.IconData != "" || in.IconType == models.IconTypeImageUpload {
			// The file name is ours to choose; a document can't name one
			var problem string
			if icon, problem = s.decodeImage(ImageIcon, in.IconData); problem != "" {
				issues.Add(dashdoc.Path("services", i, "icon_data"), "%s", problem)
				continue
			}
			req.IconType, req.IconImagePath = models.IconTypeImageUpload, pendingIcon
		}
		service, err := NewServiceFromRequest(req)
		if err != nil {
			if err := report(dashdoc.Path("services", i, ""), err); err != nil {
				return nil, err
			}
			continue
		}
		service.ID, service.UserID, service.Position = stored.serviceID(i), user.ID, in.Position
		service.BoardID = boards.id(in.BoardRef)
		service.GroupID = idOf(groupIDs, in.GroupRef)
		service.IntegrationID = idOf(integrationIDs, in.IntegrationRef)
		if icon != nil {
			p.icons[len(p.rows.Services)] = icon
		}
		p.rows.Services = append(p.rows.Services, service)
	}

	// The widget service checks groups and integrations by ID, so it gets
	// the ones this document creates
	widgetService := &WidgetService{groups: groups, integrations: integrationList, allowMasked: true}
	widgetIDs := map[string]string{}
	for i, in := range doc.Widgets {
		req := &models.WidgetRequest{
			Type:           in.Type,
			Title:          &in.Title,
			GroupID:        idOf(groupIDs, in.GroupRef),
			IntegrationID:  idOf(integrationIDs, in.IntegrationRef),
			Config:         in.Config,
			RefreshSeconds: in.RefreshSeconds,
			Enabled:        in.Enabled,
		}
		if in.CardSize != "" {
			req.CardSize = &in.CardSize
		}
		widgetType, _ := widgets.Get(strings.TrimSpace(in.Type))
		if lister, ok := widgetType.(widgets.ConfigIntegrations); ok && hasConfig(in.Config) {
			config, problem := configWithIDs(lister, in.Config, integrationIDs)
			if problem != "" {
				issues.Add(dashdoc.Path("widgets", i, "config"), "%s", problem)
				continue
			}
			req.Config = config
		}
		if current := stored.widgetRow(i); current != nil && hasConfig(req.Config) {
			// A secret that is still masked keeps the value the widget has
			if secret, ok := widgetType.(widgets.SecretConfig); ok {
				req.Config = secret.Unredact(req.Config, current.Config)
			}
		}
		widget, err := widgetService.apply(ctx, user.ID, isAdmin, req, nil)
		if err != nil {
			if err := report(dashdoc.Path("widgets", i, ""), err); err != nil {
				return nil, err
			}
			continue
		}
		widget.ID, widget.Position = stored.widgetID(i), in.Position
		widget.BoardID = boards.id(in.BoardRef)
		widgetIDs[in.Ref] = widget.ID
		if masked, ok := widgetType.(widgets.MaskedConfig); ok && masked.HasMasks(widget.Config) {
			p.needsSecrets++
		}
		p.rows.Widgets = append(p.rows.Widgets, widget)
	}

	// A merge adds tiles and keeps the look the user has
	if doc.Preferences != nil && mode == dashdoc.ModeReplace {
		p.rows.Preferences, p.wallpaper = s.planPreferences(doc.Preferences, integrationIDs, widgetIDs, &issues)
	}

	if err := issues.Err(); err != nil {
		return nil, err
	}
	if mode == dashdoc.ModeMerge {
		// So a merge can put them right behind the user's tiles
		positions := make([]*int, 0, len(p.rows.Services)+len(p.rows.Widgets))
		for _, service := range p.rows.Services {
			positions = append(positions, &service.Position)
		}
		for _, widget := range p.rows.Widgets {
			positions = append(positions, &widget.Position)
		}
		rankPositions(positions)
	}
	p.created = dashdoc.Summary{
		Groups:           dashdoc.Count{Created: len(p.rows.Groups)},
		Services:         dashdoc.Count{Created: len(p.rows.Services)},
		Widgets:          dashdoc.Count{Created: len(p.rows.Widgets)},
		Integrations:     dashdoc.Count{Created: len(p.rows.Integrations)},
		Preferences:      p.rows.Preferences != nil,
		NeedsCredentials: p.needsCredentials,
		NeedsSecrets:     p.needsSecrets,
	}
	if prefs := p.rows.Preferences; prefs != nil && prefs.CustomCSS != nil {
		p.created.CustomCSS = *prefs.CustomCSS
	}
	return p, nil
}

// rankPositions renumbers tile positions 0, 1, 2, ... in their order (equal
// positions stay equal)
func rankPositions(positions []*int) {
	sorted := make([]int, len(positions))
	for i, position := range positions {
		sorted[i] = *position
	}
	slices.Sort(sorted)
	sorted = slices.Compact(sorted)
	for _, position := range positions {
		*position, _ = slices.BinarySearch(sorted, *position)
	}
}

// decodeImage decodes and validates an embedded image; problem is "" when
// it is fine
func (s *DashboardTransferService) decodeImage(kind ImageKind, encoded string) (data []byte, problem string) {
	if encoded == "" {
		return nil, "the image is missing"
	}
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, "must be base64"
	}
	if err := s.Images.Check(kind, data); err != nil {
		return nil, err.Error()
	}
	return data, ""
}

// configWithIDs replaces the integration refs a widget config lists by the
// IDs of the integrations the document creates
func configWithIDs(lister widgets.ConfigIntegrations, config []byte, integrationIDs map[string]string) (out []byte, problem string) {
	refs := lister.IntegrationIDs(config)
	ids := make([]string, 0, len(refs))
	for _, ref := range refs {
		id, ok := integrationIDs[ref]
		if !ok {
			return nil, fmt.Sprintf("no integration has the ref %q", ref)
		}
		ids = append(ids, id)
	}
	out, err := lister.SetIntegrationIDs(config, ids)
	if err != nil {
		return nil, "config must be a JSON object"
	}
	return out, ""
}

func idOf(ids map[string]string, ref string) *string {
	if id, ok := ids[ref]; ok && ref != "" {
		return &id
	}
	return nil
}

// planPreferences builds the preferences update of a document and validates
// it like the preferences endpoint does. It also returns the embedded
// wallpaper, which becomes the background once it is stored.
func (s *DashboardTransferService) planPreferences(in *dashdoc.Preferences, integrationIDs, widgetIDs map[string]string, issues *dashdoc.Issues) (*models.PreferencesUpdateRequest, []byte) {
	req := &models.PreferencesUpdateRequest{
		ThemeMode:             in.ThemeMode,
		OpenInNewTab:          in.OpenInNewTab,
		EnableCardResizing:    in.EnableCardResizing,
		EnableServiceGrouping: in.EnableServiceGrouping,
		CardScale:             in.CardScale,
		ViewMode:              in.ViewMode,
		WallpaperBlur:         in.WallpaperBlur,
		WallpaperDim:          in.WallpaperDim,
		CardOpacity:           in.CardOpacity,
		CardBlur:              in.CardBlur,
		LayoutMode:            in.LayoutMode,
		GroupDisplay:          in.GroupDisplay,
		FollowLayout:          in.GroupDisplay == nil,
		ThemePreset:           in.ThemePreset,
		ThemeFont:             in.ThemeFont,
		KpiMono:               in.KpiMono,
		IconSource:            in.IconSource,
		CustomCSS:             in.CustomCSS,
	}
	// A file can come from someone else: its CSS gets the same check as
	// CSS saved in the theme settings
	if in.CustomCSS != nil {
		if err := utils.ValidateCustomCSS(*in.CustomCSS); err != nil {
			issues.Add("preferences.custom_css", "%s", err.Error())
		}
	}

	var wallpaper []byte
	switch {
	case in.WallpaperData != "":
		var problem string
		if wallpaper, problem = s.decodeImage(ImageWallpaper, in.WallpaperData); problem != "" {
			issues.Add("preferences.wallpaper_data", "%s", problem)
		}
	case in.ThemeBackground != nil && *in.ThemeBackground == "":
		req.ThemeBackground = models.NullableString{Set: true}
	case in.ThemeBackground != nil:
		// An image URL only; the path of an uploaded file is never taken
		// from a document
		if err := utils.ValidateIconURL(*in.ThemeBackground); err != nil {
			issues.Add("preferences.theme_background", "must be a valid HTTP or HTTPS URL")
		}
		req.ThemeBackground = models.NullableString{Value: in.ThemeBackground, Set: true}
	}

	switch {
	case in.ThemeAccentColor == nil:
	case *in.ThemeAccentColor == "":
		req.ThemeAccentColor = models.NullableString{Set: true}
	default:
		// What the preferences endpoint takes, within the 7 characters the
		// column holds
		if preferencesValidator.Var(*in.ThemeAccentColor, "hexcolor,max=7") != nil {
			issues.Add("preferences.theme_accent_color", "must be a valid hex color (e.g., #3B82F6)")
		}
		req.ThemeAccentColor = models.NullableString{Value: in.ThemeAccentColor, Set: true}
	}

	switch {
	case in.Language == nil:
	case *in.Language == "":
		req.Language = models.NullableString{Set: true}
	default:
		if !models.ValidLanguage(*in.Language) {
			issues.Add("preferences.language", "must be one of: %s", models.Languages)
		}
		req.Language = models.NullableString{Value: in.Language, Set: true}
	}

	if in.StatusStrip != nil {
		strip := &models.StatusStrip{Enabled: in.StatusStrip.Enabled, Chips: []models.StatusChip{}}
		for _, chip := range in.StatusStrip.Chips {
			ids := integrationIDs
			if chip.Source == dashdoc.ChipSourceWidget {
				ids = widgetIDs
			}
			// The ref exists (dashdoc.Validate); its entity may have failed
			// its own checks, which is reported there
			if id, ok := ids[chip.Ref]; ok {
				strip.Chips = append(strip.Chips, models.StatusChip{Source: chip.Source, ID: id, KPI: chip.KPI})
			}
		}
		req.StatusStrip = strip
	}

	var fieldErrors validator.ValidationErrors
	if err := preferencesValidator.Struct(req); errors.As(err, &fieldErrors) {
		for _, fieldError := range fieldErrors {
			field := strings.TrimPrefix(fieldError.Namespace(), "PreferencesUpdateRequest.")
			issues.Add("preferences", "%s is invalid (%s)", field, fieldError.Tag())
		}
	}
	return req, wallpaper
}

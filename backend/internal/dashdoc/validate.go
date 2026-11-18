package dashdoc

import (
	"fmt"
	"strings"
)

// Issue is one problem in a document. Path names the entity and field, like
// "services[2].group_ref".
type Issue struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

// ValidationError lists everything wrong with a document. The messages are
// safe to show to the user.
type ValidationError struct {
	Issues []Issue
}

func (e *ValidationError) Error() string {
	parts := make([]string, len(e.Issues))
	for i, issue := range e.Issues {
		parts[i] = issue.Path + ": " + issue.Message
	}
	return "invalid dashboard document: " + strings.Join(parts, "; ")
}

// Issues collects problems while a document is checked
type Issues []Issue

// Add records a problem at path
func (i *Issues) Add(path, format string, args ...any) {
	*i = append(*i, Issue{Path: path, Message: fmt.Sprintf(format, args...)})
}

// Err returns a *ValidationError, or nil when nothing was recorded
func (i Issues) Err() error {
	if len(i) == 0 {
		return nil
	}
	return &ValidationError{Issues: i}
}

// Path builds "list[index].field"; field may be empty
func Path(list string, index int, field string) string {
	path := fmt.Sprintf("%s[%d]", list, index)
	if field != "" {
		path += "." + field
	}
	return path
}

// Validate checks the structure of a document: its version, sizes, refs and
// positions. What a field may hold (URLs, widget configs, admin-only types)
// is checked on import by the same code that checks a normal create.
func Validate(doc *Document) error {
	var issues Issues
	if doc.Version != Version {
		issues.Add("version", "unsupported version %d; this server reads version %d", doc.Version, Version)
		return issues.Err()
	}

	checkSize(&issues, "groups", len(doc.Groups), MaxGroups)
	checkSize(&issues, "integrations", len(doc.Integrations), MaxIntegrations)
	checkSize(&issues, "services", len(doc.Services), MaxServices)
	checkSize(&issues, "widgets", len(doc.Widgets), MaxWidgets)
	if len(issues) > 0 {
		return issues.Err()
	}

	groups, integrations, widgets, services := refSet{}, refSet{}, refSet{}, refSet{}
	defaults := 0
	for i, group := range doc.Groups {
		groups.add(&issues, Path("groups", i, "ref"), group.Ref)
		if group.Default {
			if defaults++; defaults > 1 {
				issues.Add(Path("groups", i, "default"), "only one group can be the default")
			}
		}
	}
	for i, integration := range doc.Integrations {
		integrations.add(&issues, Path("integrations", i, "ref"), integration.Ref)
	}
	for i, service := range doc.Services {
		services.add(&issues, Path("services", i, "ref"), service.Ref)
		groups.need(&issues, Path("services", i, "group_ref"), service.GroupRef, "group")
		integrations.need(&issues, Path("services", i, "integration_ref"), service.IntegrationRef, "integration")
		checkPosition(&issues, Path("services", i, "position"), service.Position)
	}
	for i, widget := range doc.Widgets {
		widgets.add(&issues, Path("widgets", i, "ref"), widget.Ref)
		groups.need(&issues, Path("widgets", i, "group_ref"), widget.GroupRef, "group")
		integrations.need(&issues, Path("widgets", i, "integration_ref"), widget.IntegrationRef, "integration")
		checkPosition(&issues, Path("widgets", i, "position"), widget.Position)
	}

	if prefs := doc.Preferences; prefs != nil {
		if prefs.ThemeBackground != nil && prefs.WallpaperData != "" {
			issues.Add("preferences.wallpaper_data", "give theme_background or wallpaper_data, not both")
		}
		if prefs.StatusStrip != nil {
			for i, chip := range prefs.StatusStrip.Chips {
				path := Path("preferences.status_strip.chips", i, "ref")
				switch chip.Source {
				case ChipSourceIntegration:
					integrations.require(&issues, path, chip.Ref, "integration")
				case ChipSourceWidget:
					widgets.require(&issues, path, chip.Ref, "widget")
				default:
					issues.Add(Path("preferences.status_strip.chips", i, "source"), "must be %q or %q", ChipSourceIntegration, ChipSourceWidget)
				}
			}
		}
	}
	return issues.Err()
}

func checkSize(issues *Issues, list string, n, limit int) {
	if n > limit {
		issues.Add(list, "at most %d %s are allowed, the document has %d", limit, list, n)
	}
}

func checkPosition(issues *Issues, path string, position int) {
	if position < 0 || position > MaxPosition {
		issues.Add(path, "must be between 0 and %d", MaxPosition)
	}
}

// refSet is the refs of one kind of entity
type refSet map[string]bool

// add registers an entity's own ref; empty means it has none
func (s refSet) add(issues *Issues, path, ref string) {
	switch {
	case ref == "":
	case len(ref) > MaxRefLen:
		issues.Add(path, "must be %d characters or less", MaxRefLen)
	case s[ref]:
		issues.Add(path, "ref %q is used twice", ref)
	default:
		s[ref] = true
	}
}

// need checks an optional pointer to an entity
func (s refSet) need(issues *Issues, path, ref, kind string) {
	if ref != "" {
		s.require(issues, path, ref, kind)
	}
}

// require checks a pointer to an entity that must be set
func (s refSet) require(issues *Issues, path, ref, kind string) {
	if !s[ref] {
		issues.Add(path, "no %s has the ref %q", kind, ref)
	}
}

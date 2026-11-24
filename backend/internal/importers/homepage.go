package importers

import (
	"errors"

	"github.com/nimbus/backend/internal/dashdoc"
)

// Homepage converts a services.yaml (or bookmarks.yaml) of gethomepage.dev:
// a list of groups, each a list of services. A service's widget becomes a
// Nimbus integration when Nimbus has that kind; its key or password is not
// taken over.
func Homepage(data []byte) (dashdoc.Document, []string, error) {
	tree, err := parse(data)
	if err != nil {
		return dashdoc.Document{}, nil, err
	}
	groups, ok := tree.([]any)
	if !ok {
		return dashdoc.Document{}, nil, errors.New("this is not a Homepage services.yaml: it must be a list of groups")
	}
	b := newBuilder()
	b.plainIcons = true
	for _, group := range groups {
		// A group is a map with one key: its name
		for _, name := range sortedKeys(asMap(group)) {
			b.homepageGroup(asString(name), asMap(group)[name], 0)
		}
	}
	return b.result()
}

// maxGroupDepth stops at groups nested deeper than any real config has
const maxGroupDepth = 8

func (b *builder) homepageGroup(group string, entries any, depth int) {
	if len(asList(entries)) == 0 {
		b.emptyGroup(group)
	}
	for _, item := range asList(entries) {
		for _, name := range sortedKeys(asMap(item)) {
			switch value := asMap(item)[name].(type) {
			case map[string]any:
				b.homepageService(group, asString(name), value)
			case []any:
				// A bookmark is a list with one map; a nested group a list of
				// services. Nimbus groups are flat, so it becomes its own group.
				if len(value) > 0 && asString(asMap(value[0])["href"]) != "" {
					b.homepageService(group, asString(name), asMap(value[0]))
				} else if depth < maxGroupDepth {
					b.homepageGroup(asString(name), value, depth+1)
				}
			default:
				b.warn("%s: skipped, it has no settings", name)
			}
		}
	}
}

func (b *builder) homepageService(group, name string, settings map[string]any) {
	// One widget, or several: the first one Nimbus knows is used
	widgets := asList(settings["widgets"])
	if widget := asMap(settings["widget"]); widget != nil {
		widgets = append([]any{widget}, widgets...)
	}
	e := entry{
		name:        name,
		url:         asString(settings["href"]),
		description: asString(settings["description"]),
		icon:        asString(settings["icon"]),
		group:       group,
	}
	for _, widget := range widgets {
		app := asString(asMap(widget)["type"])
		if app == "" {
			continue
		}
		if e.app == "" || kindOf(e.app) == "" {
			e.app, e.appURL = app, asString(asMap(widget)["url"])
		}
		if e.url == "" {
			// A service without a link still has its widget's address
			e.url = asString(asMap(widget)["url"])
		}
	}
	b.service(e)
}

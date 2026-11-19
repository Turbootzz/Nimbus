package importers

import (
	"errors"
	"strings"

	"github.com/nimbus/backend/internal/dashdoc"
)

// Dashy converts a conf.yml of Dashy: its sections and their items. Dashy's
// widgets are not converted.
func Dashy(data []byte) (dashdoc.Document, []string, error) {
	tree, err := parse(data)
	if err != nil {
		return dashdoc.Document{}, nil, err
	}
	sections, ok := asMap(tree)["sections"].([]any)
	if !ok {
		return dashdoc.Document{}, nil, errors.New("this is not a Dashy conf.yml: it has no sections")
	}
	b := newBuilder()
	skippedWidgets := 0
	for _, section := range sections {
		skippedWidgets += len(asList(asMap(section)["widgets"]))
		items := asList(asMap(section)["items"])
		if len(items) == 0 {
			// A group is only made for the links in it
			b.warn("Section %q: skipped, it has no items", asString(asMap(section)["name"]))
		}
		for _, item := range items {
			settings := asMap(item)
			icon := asString(settings["icon"])
			// Dashy fetches or draws these itself; Nimbus shows its default
			if icon == "generative" || strings.HasPrefix(icon, "favicon") {
				icon = ""
			}
			b.service(entry{
				name:        asString(settings["title"]),
				url:         asString(settings["url"]),
				description: asString(settings["description"]),
				icon:        icon,
				group:       asString(asMap(section)["name"]),
			})
		}
	}
	if skippedWidgets > 0 {
		b.warn("%d Dashy widgets are not imported; add the Nimbus widgets you want", skippedWidgets)
	}
	return b.result()
}

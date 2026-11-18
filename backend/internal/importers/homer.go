package importers

import (
	"errors"

	"github.com/nimbus/backend/internal/dashdoc"
)

// Homer converts a config.yml of Homer: its service groups and their items.
// An item with a type Nimbus has an integration for (PiHole, Sonarr, ...)
// gets that integration. The navbar links are not converted.
func Homer(data []byte) (dashdoc.Document, []string, error) {
	tree, err := parse(data)
	if err != nil {
		return dashdoc.Document{}, nil, err
	}
	groups, ok := asMap(tree)["services"].([]any)
	if !ok {
		return dashdoc.Document{}, nil, errors.New("this is not a Homer config.yml: it has no services list")
	}
	b := newBuilder()
	for _, group := range groups {
		for _, item := range asList(asMap(group)["items"]) {
			settings := asMap(item)
			// Font Awesome icons (icon) can't be shown; an image (logo) can
			b.service(entry{
				name:        asString(settings["name"]),
				url:         asString(settings["url"]),
				description: asString(settings["subtitle"]),
				icon:        asString(settings["logo"]),
				group:       asString(asMap(group)["name"]),
				app:         asString(settings["type"]),
				appURL:      asString(settings["endpoint"]),
			})
		}
	}
	if links := asList(asMap(tree)["links"]); len(links) > 0 {
		b.warn("%d navbar links are not imported", len(links))
	}
	return b.result()
}

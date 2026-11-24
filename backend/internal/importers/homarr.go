package importers

import (
	"errors"
	"sort"
	"strings"

	"github.com/nimbus/backend/internal/dashdoc"
)

// Homarr converts a board config of Homarr 0.x (the JSON files in
// data/configs): its apps become services, grouped by their category. Its
// widgets (weather, calendar, ...) are not converted.
func Homarr(data []byte) (dashdoc.Document, []string, error) {
	tree, err := parse(data)
	if err != nil {
		return dashdoc.Document{}, nil, err
	}
	root := asMap(tree)
	apps, ok := root["apps"].([]any)
	if !ok {
		return dashdoc.Document{}, nil, errors.New("this is not a Homarr board config: it has no apps (only the JSON configs of Homarr 0.x are supported)")
	}
	b := newBuilder()

	// Categories in their own order; apps outside one get no group
	categories := asList(root["categories"])
	sort.SliceStable(categories, func(i, j int) bool {
		return number(asMap(categories[i])["position"]) < number(asMap(categories[j])["position"])
	})
	names := map[string]string{}
	for _, category := range categories {
		id, name := asString(asMap(category)["id"]), asString(asMap(category)["name"])
		if id != "" && name != "" {
			names[id] = name
			b.group(name)
		}
	}

	// Apps as they are laid out on the largest screen: by row, then column
	sort.SliceStable(apps, func(i, j int) bool {
		yi, xi := homarrPlace(apps[i])
		yj, xj := homarrPlace(apps[j])
		return yi < yj || (yi == yj && xi < xj)
	})
	for _, item := range apps {
		app := asMap(item)
		behaviour, area := asMap(app["behaviour"]), asMap(app["area"])
		// The address to open; the app's own URL is the one Homarr reaches it at
		link := asString(behaviour["externalUrl"])
		if link == "" {
			link = asString(app["url"])
		}
		e := entry{
			name:        asString(app["name"]),
			url:         link,
			description: asString(behaviour["tooltipDescription"]),
			icon:        asString(asMap(app["appearance"])["iconUrl"]),
			app:         asString(asMap(app["integration"])["type"]),
			appURL:      asString(app["url"]),
		}
		if asString(area["type"]) == "category" {
			e.group = names[asString(asMap(area["properties"])["id"])]
		}
		b.service(e)
	}

	if widgets := asList(root["widgets"]); len(widgets) > 0 {
		types := map[string]any{}
		for _, widget := range widgets {
			if widgetType := asString(asMap(widget)["type"]); widgetType != "" {
				types[widgetType] = nil
			}
		}
		b.warn("%d Homarr widgets are not imported (%s); add the Nimbus widgets you want", len(widgets), strings.Join(sortedKeys(types), ", "))
	}
	return b.result()
}

// homarrPlace returns where an app sits on the largest layout it has
func homarrPlace(app any) (y, x float64) {
	shape := asMap(asMap(app)["shape"])
	for _, size := range []string{"lg", "md", "sm"} {
		if location := asMap(asMap(shape[size])["location"]); location != nil {
			return number(location["y"]), number(location["x"])
		}
	}
	return 0, 0
}

// number reads a JSON or YAML number
func number(value any) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	}
	return 0
}

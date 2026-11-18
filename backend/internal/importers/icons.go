package importers

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/nimbus/backend/internal/dashdoc"
	"github.com/nimbus/backend/internal/models"
)

// Where the icon sets that dashboards refer to by name are served. These are
// the URLs Homepage itself builds (src/components/resolvedicon.jsx in
// gethomepage/homepage); dashboard-icons is also what Nimbus' own icon
// lookup uses. Nothing is fetched here: the browser loads the image, and a
// name that doesn't exist falls back to the default icon there.
const (
	dashboardIconsURL = "https://cdn.jsdelivr.net/gh/homarr-labs/dashboard-icons/"
	selfhstIconsURL   = "https://cdn.jsdelivr.net/gh/selfhst/icons@main/"
	materialIconsURL  = "https://cdn.jsdelivr.net/npm/@mdi/svg@latest/svg/"
	// Simple Icons' own CDN serves an icon in its brand colour, or in the
	// colour given, which stays visible on a dark dashboard
	simpleIconsURL = "https://cdn.simpleicons.org/"
)

var (
	iconName  = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*$`)
	iconColor = regexp.MustCompile(`-#([0-9a-fA-F]{3,8})$`)
)

// iconURL resolves an icon as dashboards write it: a URL, a prefixed name
// (si-, mdi-, sh-, hl-) or, with plain, a dashboard-icons name. ok is false
// when it is something else.
func iconURL(icon string, plain bool) (string, bool) {
	if _, ok := webURL(icon); ok {
		return icon, true
	}
	lower := strings.ToLower(icon)
	prefix, name, prefixed := strings.Cut(lower, "-")
	if !prefixed {
		prefix, name = "", lower
	}
	// Homepage allows a colour behind mdi- and si- names
	color := ""
	if match := iconColor.FindStringSubmatch(name); match != nil && (prefix == "mdi" || prefix == "si") {
		color, name = match[1], strings.TrimSuffix(name, match[0])
	}
	// A name may say which image format it wants
	extension := ""
	for _, ext := range []string{"svg", "png", "webp"} {
		if strings.HasSuffix(name, "."+ext) {
			extension, name = ext, strings.TrimSuffix(name, "."+ext)
		}
	}
	valid := iconName.MatchString(name)

	switch {
	case prefix == "mdi" && valid:
		return materialIconsURL + name + ".svg", true
	case prefix == "si" && valid:
		if color != "" {
			return simpleIconsURL + name + "/" + color, true
		}
		return simpleIconsURL + name, true
	case prefix == "sh" && valid:
		if extension == "" {
			extension = "png"
		}
		return selfhstIconsURL + extension + "/" + name + "." + extension, true
	case prefix == "hl" && valid:
		return dashboardIconsURL + "svg/" + name + ".svg", true
	}

	// Not a prefix after all: "home-assistant.png"
	if prefixed {
		name = prefix + "-" + name
	}
	if plain && iconName.MatchString(name) {
		if extension == "" {
			extension = "png"
		}
		return dashboardIconsURL + extension + "/" + name + "." + extension, true
	}
	return "", false
}

// isEmoji reports a short text without letters or digits, like 🚀
func isEmoji(icon string) bool {
	if icon == "" || utf8.RuneCountInString(icon) > 8 {
		return false
	}
	for _, r := range icon {
		if r < 0x2000 {
			return false
		}
	}
	return true
}

// icon sets the icon of a service from what another dashboard calls it. An
// icon Nimbus can't show leaves the default one, with a warning.
func (b *builder) icon(service *dashdoc.Service, icon string) {
	icon = strings.TrimSpace(icon)
	switch {
	case icon == "":
	case isEmoji(icon):
		service.Icon = icon
	default:
		if resolved, ok := iconURL(icon, b.plainIcons); ok {
			service.IconType, service.IconURL = models.IconTypeImageURL, resolved
			return
		}
		b.warn("%s: the icon %q can't be imported (a local file or an icon set Nimbus doesn't know); the default icon is used", service.Name, icon)
	}
}

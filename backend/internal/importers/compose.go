package importers

import (
	"errors"
	"strconv"
	"strings"
	"unicode"

	"github.com/nimbus/backend/internal/dashdoc"
)

// Compose converts a docker-compose file: every service that publishes a TCP
// port becomes a link to http://host:port, named after the service, with the
// dashboard-icons icon of its image. A compose file doesn't say which machine
// it runs on, so the caller gives the host.
//
// Homepage's labels (homepage.name, .href, .icon, .description, .group) are
// used when a service has them.
func Compose(data []byte, host string) (dashdoc.Document, []string, error) {
	tree, err := parse(data)
	if err != nil {
		return dashdoc.Document{}, nil, err
	}
	services, ok := asMap(tree)["services"].(map[string]any)
	if !ok {
		return dashdoc.Document{}, nil, errors.New("this is not a docker-compose file: it has no services")
	}
	if host == "" {
		host = "localhost"
	}
	b := newBuilder()
	b.plainIcons = true
	for _, name := range sortedKeys(services) {
		settings := asMap(services[name])
		labels := composeLabels(settings["labels"])
		e := entry{
			name:        labels["homepage.name"],
			url:         labels["homepage.href"],
			description: labels["homepage.description"],
			icon:        labels["homepage.icon"],
			group:       labels["homepage.group"],
		}
		if e.name == "" {
			e.name = displayName(asString(name))
		}
		if e.icon == "" {
			if slug := imageSlug(asString(settings["image"])); slug != "" {
				// The SVG set, like the icons Nimbus looks up itself
				e.icon = dashboardIconsURL + "svg/" + slug + ".svg"
			}
		}
		if e.url == "" {
			port := publishedPort(asList(settings["ports"]))
			if port == 0 {
				b.warn("%s: skipped, it publishes no TCP port", name)
				continue
			}
			e.url = portURL(host, port)
		}
		b.service(e)
	}
	return b.result()
}

// composeLabels reads labels in either form: a map, or a list of key=value
func composeLabels(value any) map[string]string {
	labels := map[string]string{}
	for key, item := range asMap(value) {
		labels[key] = asString(item)
	}
	for _, item := range asList(value) {
		if key, label, ok := strings.Cut(asString(item), "="); ok {
			labels[strings.TrimSpace(key)] = strings.TrimSpace(label)
		}
	}
	return labels
}

// displayName makes "home-assistant" into "Home Assistant"
func displayName(service string) string {
	words := strings.FieldsFunc(service, func(r rune) bool { return r == '-' || r == '_' || r == '.' })
	for i, word := range words {
		runes := []rune(word)
		runes[0] = unicode.ToUpper(runes[0])
		words[i] = string(runes)
	}
	return strings.Join(words, " ")
}

// imageSlug returns the name of an image without registry, organisation,
// tag and digest: "lscr.io/linuxserver/sonarr:latest@sha256:..." is "sonarr"
func imageSlug(image string) string {
	image, _, _ = strings.Cut(image, "@")
	image = image[strings.LastIndex(image, "/")+1:]
	image, _, _ = strings.Cut(image, ":")
	image = strings.ToLower(image)
	if !iconName.MatchString(image) {
		return ""
	}
	return image
}

// publishedPort returns the first TCP port a service publishes on the host,
// or 0. Ports come as "8080:80", "127.0.0.1:8080:80/tcp", "8080-8090:80-90",
// or as a map with published and target; a bare "80" is not published on a
// port anyone can know.
func publishedPort(ports []any) int {
	for _, item := range ports {
		published, protocol := "", ""
		if long := asMap(item); long != nil {
			published, protocol = asString(long["published"]), asString(long["protocol"])
		} else {
			short, proto, _ := strings.Cut(asString(item), "/")
			protocol = proto
			// [ip:]published:target; the IP may be a bracketed IPv6 address
			parts := strings.Split(short[strings.LastIndex(short, "]")+1:], ":")
			if len(parts) >= 2 {
				published = parts[len(parts)-2]
			}
		}
		if protocol != "" && !strings.EqualFold(protocol, "tcp") {
			continue
		}
		// The first port of a range
		published, _, _ = strings.Cut(published, "-")
		if port, err := strconv.Atoi(published); err == nil && port > 0 && port <= 65535 {
			return port
		}
	}
	return 0
}

// portURL is where a published port is reached; the usual TLS ports get https
func portURL(host string, port int) string {
	switch port {
	case 80:
		return "http://" + host
	case 443:
		return "https://" + host
	case 8443, 9443:
		return "https://" + host + ":" + strconv.Itoa(port)
	}
	return "http://" + host + ":" + strconv.Itoa(port)
}

package handlers

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/net/html"
)

const (
	faviconFetchTimeout = 10 * time.Second
	faviconMaxHTMLBytes = 512 * 1024 // cap HTML parse to avoid huge pages
	// Use a Chrome-on-macOS UA. A descriptive bot UA gets us 403'd by
	// Cloudflare-protected sites (e.g. claude.ai), even though the well-known
	// favicon paths themselves are served — we want the HTML too so we can
	// find <link> tags.
	faviconUserAgent    = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	faviconMaxRedirects = 5
	// dashboardIconsDefaultBase is the canonical jsDelivr URL for the curated
	// homarr-labs/dashboard-icons (Apache 2.0) SVG collection. For a homelab
	// dashboard this is almost always crisper than scraping origin sites, which
	// often expose only small favicons or live behind LAN IPs with no useful
	// icon at all. The base is overridable via DASHBOARD_ICONS_BASE_URL — useful
	// for self-hosted mirrors, airgapped deployments, or setting it to empty to
	// disable the curated lookup entirely (in which case only origin scraping
	// runs).
	dashboardIconsDefaultBase = "https://cdn.jsdelivr.net/gh/homarr-labs/dashboard-icons/svg"
	dashboardIconsEnvVar      = "DASHBOARD_ICONS_BASE_URL"
)

// dashboardIconsBase returns the configured base URL (no trailing slash) for
// the curated icon lookup. An empty value (env explicitly set to empty)
// disables curated lookup.
func dashboardIconsBase() string {
	if v, ok := os.LookupEnv(dashboardIconsEnvVar); ok {
		return strings.TrimRight(strings.TrimSpace(v), "/")
	}
	return dashboardIconsDefaultBase
}

// blockedFaviconHosts are hosts we refuse to fetch from even though Nimbus is
// otherwise permissive about private addresses (homelab services often live
// on RFC1918). These hosts expose credentials on cloud VMs, so an authenticated
// user must not be able to coax the server into hitting them — directly or via
// a redirect from an attacker-controlled public URL.
var blockedFaviconHosts = map[string]struct{}{
	"169.254.169.254":          {}, // AWS / Azure / GCP / DO IMDS
	"fd00:ec2::254":            {}, // AWS IMDS over IPv6
	"100.100.100.200":          {}, // Alibaba Cloud metadata
	"metadata.google.internal": {}, // GCP DNS alias for 169.254.169.254
}

func isBlockedFaviconHost(host string) bool {
	_, blocked := blockedFaviconHosts[strings.ToLower(host)]
	return blocked
}

// errBlockedFaviconRedirect is returned by CheckRedirect when the favicon
// client is asked to follow a redirect to a metadata-style host. It is wrapped
// in url.Error by net/http so callers should errors.Is it.
var errBlockedFaviconRedirect = errors.New("redirect to blocked host")

// faviconHTTPClient is shared across requests so the connection pool is reused.
// CheckRedirect both caps the chain and refuses redirects to metadata hosts.
var faviconHTTPClient = &http.Client{
	Timeout: faviconFetchTimeout,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= faviconMaxRedirects {
			return fmt.Errorf("stopped after %d redirects", faviconMaxRedirects)
		}
		if isBlockedFaviconHost(req.URL.Hostname()) {
			return errBlockedFaviconRedirect
		}
		return nil
	},
}

// FetchServiceFavicon resolves an icon image for a service and stores it under
// the same directory as uploaded icons, returning the bare filename. It tries,
// in order:
//  1. A curated SVG from homarr-labs/dashboard-icons matched by the `name`
//     query param (slug = lowercased, spaces → hyphens). This is the primary
//     path for homelab use — a "plex" service running on a LAN IP has no
//     useful favicon at the origin, but plex.svg exists in the curated set.
//  2. Origin-site scraping using the `url` query param: <link> tags from HTML,
//     then well-known paths (/favicon.svg, /apple-touch-icon.png, /favicon.ico).
//
// At least one of `name` or `url` must be provided. Server-side fetching
// sidesteps CORS and lets icons persist even if the origin/CDN later goes
// offline.
func (h *ServiceHandler) FetchServiceFavicon(c *fiber.Ctx) error {
	if _, err := RequireUserID(c); err != nil {
		return err
	}

	name := strings.TrimSpace(c.Query("name"))
	rawURL := strings.TrimSpace(c.Query("url"))
	if name == "" && rawURL == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "name or url query parameter is required",
		})
	}

	var pageURL *url.URL
	if rawURL != "" {
		parsed, err := url.ParseRequestURI(rawURL)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid URL. Must include http:// or https:// scheme",
			})
		}
		if isBlockedFaviconHost(parsed.Hostname()) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Refusing to fetch from cloud metadata host",
			})
		}
		pageURL = parsed
	}

	if err := os.MkdirAll(UploadDir, 0o755); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create upload directory",
		})
	}

	ctx, cancel := context.WithTimeout(c.Context(), faviconFetchTimeout)
	defer cancel()

	candidates := buildIconCandidates(ctx, name, pageURL)
	var lastErr error
	for _, candidate := range candidates {
		filename, err := downloadAndSaveFavicon(ctx, candidate)
		if err == nil {
			return c.Status(fiber.StatusOK).JSON(fiber.Map{
				"icon_image_path": filename,
				"message":         "Icon fetched successfully",
			})
		}
		lastErr = err
	}

	// Log the upstream detail server-side for ops debugging, but show the
	// user a friendly, context-aware message that doesn't leak CDN URLs or
	// status codes (those imply a problem the user can't act on).
	if lastErr != nil {
		urlStr := ""
		if pageURL != nil {
			urlStr = pageURL.String()
		}
		log.Printf("FetchServiceFavicon: all candidates failed (name=%q, url=%q): %v",
			name, urlStr, lastErr)
	}
	return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
		"error": iconNotFoundMessage(name, pageURL),
	})
}

// iconNotFoundMessage builds a friendly error string based on which inputs
// were tried, without exposing the underlying CDN or HTTP status.
func iconNotFoundMessage(name string, pageURL *url.URL) string {
	hasName := strings.TrimSpace(name) != ""
	hasURL := pageURL != nil
	switch {
	case hasName && hasURL:
		return fmt.Sprintf("Could not automatically fetch an icon for %q or from that URL", strings.TrimSpace(name))
	case hasName:
		return fmt.Sprintf("Could not automatically fetch an icon for %q", strings.TrimSpace(name))
	default:
		return "Could not automatically fetch an icon from that URL"
	}
}

// buildIconCandidates returns the full ordered list of URLs to try, starting
// with the curated dashboard-icons lookup (when a name is provided and the
// curated base is configured) and then origin-site candidates (when a URL is
// provided).
func buildIconCandidates(ctx context.Context, name string, pageURL *url.URL) []*url.URL {
	candidates := dashboardIconsCandidates(name)
	if pageURL != nil {
		candidates = append(candidates, resolveFaviconCandidates(ctx, pageURL)...)
	}
	return candidates
}

// dashboardIconsCandidates returns dashboard-icons URLs to try for a service
// name, in order of specificity:
//  1. Full kebab-case slug of the whole name ("My Plex Server" → "my-plex-server")
//  2. Each whitespace-separated token ("my", "plex", "server")
//
// Duplicates are removed. The token fallback exists because dashboard-icons
// is indexed by brand name (plex.svg), so "Plex Media Server" — which a user
// would naturally type — needs to fall through to the bare "plex" lookup.
// Returns nil when curated lookup is disabled or the name is empty.
func dashboardIconsCandidates(name string) []*url.URL {
	base := dashboardIconsBase()
	if base == "" {
		return nil
	}
	seen := make(map[string]struct{})
	var out []*url.URL
	addSlug := func(slug string) {
		if slug == "" {
			return
		}
		if _, dup := seen[slug]; dup {
			return
		}
		seen[slug] = struct{}{}
		if u, err := url.Parse(fmt.Sprintf("%s/%s.svg", base, slug)); err == nil {
			out = append(out, u)
		}
	}
	addSlug(serviceNameSlug(name))
	for _, token := range strings.Fields(strings.ToLower(name)) {
		addSlug(serviceNameSlug(token))
	}
	return out
}

// serviceNameSlug turns a service name into the kebab-case lookup key used by
// homarr-labs/dashboard-icons. "Home Assistant" → "home-assistant". Returns ""
// if the slug would be unsafe to drop into a URL path.
func serviceNameSlug(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	if name == "" {
		return ""
	}
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-':
			b.WriteRune(r)
		case r == ' ' || r == '_':
			b.WriteRune('-')
		}
		// All other characters (slashes, dots, accented letters, etc.) are
		// dropped: the curated icon repo uses ASCII kebab-case filenames only.
	}
	slug := strings.Trim(b.String(), "-")
	return slug
}

// resolveFaviconCandidates returns the URLs to try in order:
//  1. <link rel="icon|shortcut icon|apple-touch-icon"> hrefs from the page HTML
//     (sorted by parseFaviconLinksFromPage, largest first)
//  2. <scheme>://<host>/favicon.svg — modern well-known path; sites behind
//     anti-bot protection (e.g. claude.ai) block the HTML but still serve the
//     SVG, and SVG is the crispest option when it exists
//  3. <scheme>://<host>/apple-touch-icon.png — iOS's well-known path, often
//     present even when not linked in HTML (e.g. github.com hosts a 120x120
//     here but only advertises a 32x32 in its <link> tags)
//  4. <scheme>://<host>/apple-touch-icon-precomposed.png — pre-iOS-7 variant
//  5. <scheme>://<host>/favicon.ico — last resort
func resolveFaviconCandidates(ctx context.Context, pageURL *url.URL) []*url.URL {
	candidates := parseFaviconLinksFromPage(ctx, pageURL)

	wellKnown := func(path string) *url.URL {
		u := *pageURL
		u.Path = path
		u.RawQuery = ""
		u.Fragment = ""
		return &u
	}
	fallbacks := []*url.URL{
		wellKnown("/favicon.svg"),
		wellKnown("/apple-touch-icon.png"),
		wellKnown("/apple-touch-icon-precomposed.png"),
		wellKnown("/favicon.ico"),
	}

	// Dedupe: don't append a fallback we already parsed from the HTML.
	seen := make(map[string]struct{}, len(candidates))
	for _, c := range candidates {
		seen[c.String()] = struct{}{}
	}
	for _, fb := range fallbacks {
		if _, dup := seen[fb.String()]; !dup {
			candidates = append(candidates, fb)
			seen[fb.String()] = struct{}{}
		}
	}
	return candidates
}

// faviconCandidate carries the resolved URL plus a score derived from the
// <link sizes> attribute (and rel type as a fallback). Higher score = bigger
// icon, so we try the crispest candidate first and fall back if it fails.
type faviconCandidate struct {
	url   *url.URL
	score int
}

// parseFaviconLinksFromPage fetches pageURL, parses the HTML, and returns the
// absolute URLs of any <link rel="icon"|"shortcut icon"|"apple-touch-icon">,
// sorted largest-first so the user gets a crisp icon when the site provides
// multiple sizes. Returns an empty slice on any fetch/parse failure so the
// caller falls back to /favicon.ico.
func parseFaviconLinksFromPage(ctx context.Context, pageURL *url.URL) []*url.URL {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL.String(), nil)
	if err != nil {
		return nil
	}
	req.Header.Set("User-Agent", faviconUserAgent)
	resp, err := faviconHTTPClient.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil
	}
	doc, err := html.Parse(io.LimitReader(resp.Body, faviconMaxHTMLBytes))
	if err != nil {
		return nil
	}

	var candidates []faviconCandidate
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "link" {
			var rel, href, sizes, typ string
			for _, a := range n.Attr {
				switch strings.ToLower(a.Key) {
				case "rel":
					rel = strings.ToLower(a.Val)
				case "href":
					href = a.Val
				case "sizes":
					sizes = a.Val
				case "type":
					typ = strings.ToLower(a.Val)
				}
			}
			if href != "" && isIconRel(rel) {
				if u, err := url.Parse(strings.TrimSpace(href)); err == nil {
					candidates = append(candidates, faviconCandidate{
						url:   pageURL.ResolveReference(u),
						score: faviconSizeScore(rel, sizes, typ),
					})
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	// Stable sort so equal-score candidates retain document order.
	sort.SliceStable(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})

	out := make([]*url.URL, len(candidates))
	for i, c := range candidates {
		out[i] = c.url
	}
	return out
}

// faviconSizeScore turns a <link>'s sizes/type attributes into a comparable
// "px" value so we can pick the crispest icon. SVG (vector) wins over any
// raster. Falls back to a sensible default based on the rel type when no
// sizes attribute is set.
//
// Examples:
//
//	type="image/svg+xml"     → 1024 (vector beats any raster)
//	sizes="any"              → 1024 (vector convention)
//	sizes="192x192"          → 192
//	sizes="16x16 32x32"      → 32 (largest pair wins)
//	rel=apple-touch-icon     → 180 (Apple default)
//	rel=icon, no sizes       → 32  (typical favicon)
func faviconSizeScore(rel, sizes, typ string) int {
	if strings.TrimSpace(typ) == "image/svg+xml" {
		return 1024
	}
	if s := strings.TrimSpace(strings.ToLower(sizes)); s != "" {
		if s == "any" {
			return 1024
		}
		best := 0
		for _, pair := range strings.Fields(s) {
			parts := strings.Split(pair, "x")
			if len(parts) != 2 {
				continue
			}
			w, errW := strconv.Atoi(parts[0])
			h, errH := strconv.Atoi(parts[1])
			if errW != nil || errH != nil {
				continue
			}
			if w > best {
				best = w
			}
			if h > best {
				best = h
			}
		}
		if best > 0 {
			return best
		}
	}
	// No usable sizes attribute - infer from rel.
	for _, part := range strings.Fields(rel) {
		if part == "apple-touch-icon" || part == "apple-touch-icon-precomposed" {
			return 180
		}
	}
	return 32
}

// isIconRel reports whether a rel attribute denotes a favicon. Handles "icon",
// the legacy "shortcut icon" (matched via the "icon" token), and Apple variants.
func isIconRel(rel string) bool {
	for _, part := range strings.Fields(rel) {
		switch part {
		case "icon", "apple-touch-icon", "apple-touch-icon-precomposed":
			return true
		}
	}
	return false
}

// downloadAndSaveFavicon GETs target, validates the body as an allowed image
// type, and writes it under UploadDir via the shared saveValidatedImage helper.
func downloadAndSaveFavicon(ctx context.Context, target *url.URL) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", faviconUserAgent)
	resp, err := faviconHTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status %d from %s", resp.StatusCode, target.String())
	}
	return saveValidatedImage(resp.Body, UploadDir, MaxUploadSize, FaviconAllowedMimeTypes)
}

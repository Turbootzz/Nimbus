package handlers

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/net/html"
)

const (
	faviconFetchTimeout = 10 * time.Second
	faviconMaxHTMLBytes = 512 * 1024 // cap HTML parse to avoid huge pages
	faviconUserAgent    = "Nimbus-Favicon-Fetcher/1.0"
	faviconMaxRedirects = 5
)

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

// FetchServiceFavicon downloads the favicon for the service URL provided in
// the `url` query param, stores it under the same directory as uploaded icons,
// and returns the bare filename so the frontend can render it like any other
// uploaded image. Server-side fetch sidesteps CORS and lets the icon persist
// even if the origin site later goes offline.
func (h *ServiceHandler) FetchServiceFavicon(c *fiber.Ctx) error {
	if _, err := RequireUserID(c); err != nil {
		return err
	}

	rawURL := strings.TrimSpace(c.Query("url"))
	if rawURL == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "url query parameter is required",
		})
	}

	pageURL, err := url.ParseRequestURI(rawURL)
	if err != nil || (pageURL.Scheme != "http" && pageURL.Scheme != "https") || pageURL.Host == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid URL. Must include http:// or https:// scheme",
		})
	}
	if isBlockedFaviconHost(pageURL.Hostname()) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Refusing to fetch from cloud metadata host",
		})
	}

	if err := os.MkdirAll(UploadDir, 0o755); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create upload directory",
		})
	}

	ctx, cancel := context.WithTimeout(c.Context(), faviconFetchTimeout)
	defer cancel()

	var lastErr error
	for _, candidate := range resolveFaviconCandidates(ctx, pageURL) {
		filename, err := downloadAndSaveFavicon(ctx, candidate)
		if err == nil {
			return c.Status(fiber.StatusOK).JSON(fiber.Map{
				"icon_image_path": filename,
				"message":         "Favicon fetched successfully",
			})
		}
		lastErr = err
	}

	msg := "Could not fetch a favicon from that URL"
	if lastErr != nil {
		msg = fmt.Sprintf("Could not fetch favicon: %s", lastErr.Error())
	}
	return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": msg})
}

// resolveFaviconCandidates returns the URLs to try in order:
//  1. <link rel="icon|shortcut icon|apple-touch-icon"> hrefs from the page HTML
//  2. <scheme>://<host>/favicon.ico as the final fallback
func resolveFaviconCandidates(ctx context.Context, pageURL *url.URL) []*url.URL {
	candidates := parseFaviconLinksFromPage(ctx, pageURL)

	rootFavicon := *pageURL
	rootFavicon.Path = "/favicon.ico"
	rootFavicon.RawQuery = ""
	rootFavicon.Fragment = ""

	rootStr := rootFavicon.String()
	for _, c := range candidates {
		if c.String() == rootStr {
			return candidates
		}
	}
	return append(candidates, &rootFavicon)
}

// parseFaviconLinksFromPage fetches pageURL, parses the HTML, and returns the
// absolute URLs of any <link rel="icon"|"shortcut icon"|"apple-touch-icon">.
// Returns an empty slice on any fetch/parse failure so the caller falls back
// to /favicon.ico.
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

	var out []*url.URL
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "link" {
			var rel, href string
			for _, a := range n.Attr {
				switch strings.ToLower(a.Key) {
				case "rel":
					rel = strings.ToLower(a.Val)
				case "href":
					href = a.Val
				}
			}
			if href != "" && isIconRel(rel) {
				if u, err := url.Parse(strings.TrimSpace(href)); err == nil {
					out = append(out, pageURL.ResolveReference(u))
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return out
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

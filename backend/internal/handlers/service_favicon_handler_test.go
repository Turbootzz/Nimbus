package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// validPNG is a 1x1 transparent PNG used to stand in for a real favicon.
var validPNG = []byte{
	0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
	0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
	0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
	0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53,
	0xDE, 0x00, 0x00, 0x00, 0x0C, 0x49, 0x44, 0x41,
	0x54, 0x08, 0xD7, 0x63, 0xF8, 0xCF, 0xC0, 0x00,
	0x00, 0x03, 0x01, 0x01, 0x00, 0x18, 0xDD, 0x8D,
	0xB4, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4E,
	0x44, 0xAE, 0x42, 0x60, 0x82,
}

// minimalICO is the smallest byte sequence detectContentType recognizes as
// image/x-icon — 00 00 01 00 magic plus enough payload bytes to satisfy the
// 4-byte check. Real ICOs are larger; the sniffer doesn't care.
var minimalICO = []byte{0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x10, 0x10}

// newFaviconTestApp wires the handler behind a stub that sets user_id, so
// RequireUserID succeeds without spinning up the full auth middleware.
func newFaviconTestApp(t *testing.T) *fiber.App {
	t.Helper()
	h := &ServiceHandler{}
	app := fiber.New()
	app.Get("/services/favicon", func(c *fiber.Ctx) error {
		c.Locals("user_id", "test-user")
		return h.FetchServiceFavicon(c)
	})
	return app
}

// readFaviconResponse decodes the handler's JSON response.
func readFaviconResponse(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	var out map[string]any
	require.NoError(t, json.Unmarshal(body, &out))
	return out
}

func cleanUploadDir(t *testing.T) {
	t.Helper()
	t.Cleanup(func() { os.RemoveAll(UploadDir) })
}

func TestFetchServiceFavicon_LinkTagInHTML(t *testing.T) {
	cleanUploadDir(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, `<!doctype html><html><head><link rel="icon" href="/icon.png"></head><body></body></html>`)
		case "/icon.png":
			w.Header().Set("Content-Type", "image/png")
			w.Write(validPNG)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	app := newFaviconTestApp(t)
	req := httptest.NewRequest(http.MethodGet, "/services/favicon?url="+server.URL, nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	body := readFaviconResponse(t, resp)
	filename, _ := body["icon_image_path"].(string)
	require.NotEmpty(t, filename, "expected icon_image_path in response: %v", body)
	assert.True(t, strings.HasSuffix(filename, ".png"), "filename should keep png ext: %s", filename)

	_, err = os.Stat(filepath.Join(UploadDir, filename))
	assert.NoError(t, err, "expected file to be written to disk")
}

func TestFetchServiceFavicon_FallsBackToRootFavicon(t *testing.T) {
	cleanUploadDir(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			// HTML with no link tag - forces fallback to /favicon.ico
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, `<!doctype html><html><head><title>no icon</title></head><body></body></html>`)
		case "/favicon.ico":
			w.Header().Set("Content-Type", "image/png") // declared type ignored, sniffing decides
			w.Write(validPNG)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	app := newFaviconTestApp(t)
	req := httptest.NewRequest(http.MethodGet, "/services/favicon?url="+server.URL, nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	body := readFaviconResponse(t, resp)
	filename, _ := body["icon_image_path"].(string)
	require.NotEmpty(t, filename)
}

func TestFetchServiceFavicon_AllSourcesFail(t *testing.T) {
	cleanUploadDir(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer server.Close()

	app := newFaviconTestApp(t)
	req := httptest.NewRequest(http.MethodGet, "/services/favicon?url="+server.URL, nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadGateway, resp.StatusCode)
}

func TestFetchServiceFavicon_RejectsMissingURL(t *testing.T) {
	app := newFaviconTestApp(t)
	req := httptest.NewRequest(http.MethodGet, "/services/favicon", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestFetchServiceFavicon_RejectsNonHTTPScheme(t *testing.T) {
	app := newFaviconTestApp(t)
	req := httptest.NewRequest(http.MethodGet, "/services/favicon?url=file:///etc/passwd", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestFetchServiceFavicon_RelativeHrefResolution(t *testing.T) {
	cleanUploadDir(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/sub/":
			// Relative href that must be resolved against the page URL, not the host root.
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, `<!doctype html><html><head><link rel="icon" href="icon.png"></head></html>`)
		case "/sub/icon.png":
			w.Header().Set("Content-Type", "image/png")
			w.Write(validPNG)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	app := newFaviconTestApp(t)
	req := httptest.NewRequest(http.MethodGet, "/services/favicon?url="+server.URL+"/sub/", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode, "relative href should resolve against page path")
}

func TestFetchServiceFavicon_ICOFavicon(t *testing.T) {
	cleanUploadDir(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, `<!doctype html><html><head></head></html>`)
		case "/favicon.ico":
			// Browsers commonly serve .ico as image/x-icon - exercise the new allow list.
			w.Header().Set("Content-Type", "image/x-icon")
			w.Write(minimalICO)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	app := newFaviconTestApp(t)
	req := httptest.NewRequest(http.MethodGet, "/services/favicon?url="+server.URL, nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	body := readFaviconResponse(t, resp)
	filename, _ := body["icon_image_path"].(string)
	require.NotEmpty(t, filename)
	assert.True(t, strings.HasSuffix(filename, ".ico"), "filename should keep ico ext: %s", filename)
}

func TestFetchServiceFavicon_LinkTargetGarbageFallsBackToRoot(t *testing.T) {
	cleanUploadDir(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, `<!doctype html><html><head><link rel="icon" href="/broken-icon"></head></html>`)
		case "/broken-icon":
			// Page links to it, but the body is not a recognized image - sniffing rejects it.
			w.Header().Set("Content-Type", "image/png")
			w.Write([]byte("not really an image"))
		case "/favicon.ico":
			w.Header().Set("Content-Type", "image/png")
			w.Write(validPNG)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	app := newFaviconTestApp(t)
	req := httptest.NewRequest(http.MethodGet, "/services/favicon?url="+server.URL, nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode, "should fall through to /favicon.ico when linked icon is not a valid image")
}

func TestFetchServiceFavicon_RejectsCloudMetadataHost(t *testing.T) {
	app := newFaviconTestApp(t)
	for _, host := range []string{
		"http://169.254.169.254/latest/meta-data/",
		"http://metadata.google.internal/",
		"http://100.100.100.200/",
	} {
		req := httptest.NewRequest(http.MethodGet, "/services/favicon?url="+host, nil)
		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode, "host %s should be rejected", host)
	}
}

func TestIsBlockedFaviconHost(t *testing.T) {
	cases := map[string]bool{
		"169.254.169.254":          true,
		"METADATA.GOOGLE.INTERNAL": true, // lookup is case-insensitive
		"100.100.100.200":          true,
		"github.com":               false,
		"192.168.1.5":              false, // private but not metadata - homelab use case
		"127.0.0.1":                false,
		"":                         false,
	}
	for host, want := range cases {
		assert.Equal(t, want, isBlockedFaviconHost(host), "host=%q", host)
	}
}

func TestIsIconRel(t *testing.T) {
	cases := map[string]bool{
		"icon":                         true,
		"shortcut icon":                true,
		"ICON":                         false, // caller lowercases before passing in
		"apple-touch-icon":             true,
		"apple-touch-icon-precomposed": true,
		"stylesheet":                   false,
		"shortcut":                     false,
		"":                             false,
	}
	for rel, want := range cases {
		assert.Equal(t, want, isIconRel(rel), "rel=%q", rel)
	}
}

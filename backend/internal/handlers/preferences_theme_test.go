package handlers

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/nimbus/backend/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPreferences_ThemeDefaults(t *testing.T) {
	app, _ := setupPreferencesApp(t)

	// Without a row, and with one that only holds something else
	status, prefs, _ := send(t, app, httptest.NewRequest("GET", "/preferences", nil))
	require.Equal(t, fiber.StatusOK, status)
	assert.Equal(t, "default", prefs.ThemePreset)
	assert.Equal(t, "default", prefs.ThemeFont)
	assert.Equal(t, "dashboard-icons-svg", prefs.IconSource)

	status, prefs, _ = putJSON(t, app, "/preferences", `{"theme_mode":"dark"}`)
	require.Equal(t, fiber.StatusOK, status)
	assert.Equal(t, "default", prefs.ThemePreset)
	assert.Equal(t, "default", prefs.ThemeFont)
	assert.False(t, prefs.KpiMono)
	assert.Equal(t, "dashboard-icons-svg", prefs.IconSource)
	assert.Empty(t, prefs.CustomCSS)
}

func TestPreferences_ThemeSettingsPersist(t *testing.T) {
	app, _ := setupPreferencesApp(t)

	status, _, _ := putJSON(t, app, "/preferences", `{
		"theme_preset": "nord", "theme_font": "jetbrains-mono", "kpi_mono": true,
		"icon_source": "selfhst", "custom_css": ".glass-card { border-radius: 0 }"
	}`)
	require.Equal(t, fiber.StatusOK, status)

	// Another change leaves them alone
	status, prefs, _ := putJSON(t, app, "/preferences", `{"theme_mode":"dark"}`)
	require.Equal(t, fiber.StatusOK, status)
	assert.Equal(t, "nord", prefs.ThemePreset)
	assert.Equal(t, "jetbrains-mono", prefs.ThemeFont)
	assert.True(t, prefs.KpiMono)
	assert.Equal(t, "selfhst", prefs.IconSource)
	assert.Equal(t, ".glass-card { border-radius: 0 }", prefs.CustomCSS)

	// "" clears the CSS
	status, prefs, _ = putJSON(t, app, "/preferences", `{"custom_css":""}`)
	require.Equal(t, fiber.StatusOK, status)
	assert.Empty(t, prefs.CustomCSS)
}

func TestPreferences_ThemeValidation(t *testing.T) {
	tests := map[string]struct {
		body  string
		field string
	}{
		"unknown preset":      {`{"theme_preset":"solarized"}`, "ThemePreset"},
		"unknown font":        {`{"theme_font":"comic-sans"}`, "ThemeFont"},
		"unknown icon source": {`{"icon_source":"https://evil.example"}`, "IconSource"},
		"css with import":     {`{"custom_css":"@import 'https://evil.example/x.css';"}`, "custom_css"},
		"css with a url":      {`{"custom_css":"body{background:url(https://evil.example/a.png)}"}`, "custom_css"},
		"css breaking out":    {`{"custom_css":"</style><script>alert(1)</script>"}`, "custom_css"},
		"css over the cap":    {`{"custom_css":"` + strings.Repeat("a", utils.MaxCustomCSSBytes+1) + `"}`, "custom_css"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			app, _ := setupPreferencesApp(t)
			status, _, raw := putJSON(t, app, "/preferences", tt.body)
			require.Equal(t, fiber.StatusBadRequest, status)
			var body struct {
				Fields map[string]string `json:"fields"`
			}
			require.NoError(t, json.Unmarshal([]byte(raw), &body))
			assert.Contains(t, body.Fields, tt.field)

			// Nothing was stored
			_, prefs, _ := send(t, app, httptest.NewRequest("GET", "/preferences", nil))
			assert.Empty(t, prefs.CustomCSS)
			assert.Equal(t, "default", prefs.ThemePreset)
		})
	}
}

// A rejected field rejects the whole request, so an imported theme is never
// half applied
func TestPreferences_ThemeImportIsAllOrNothing(t *testing.T) {
	app, _ := setupPreferencesApp(t)
	status, _, _ := putJSON(t, app, "/preferences",
		`{"theme_preset":"dracula","theme_mode":"dark","custom_css":"@import 'x.css';"}`)
	require.Equal(t, fiber.StatusBadRequest, status)

	_, prefs, _ := send(t, app, httptest.NewRequest("GET", "/preferences", nil))
	assert.Equal(t, "default", prefs.ThemePreset)
	assert.Equal(t, "auto", prefs.ThemeMode)
}

// Each user reads and writes only their own CSS
func TestPreferences_CustomCSSIsPerUser(t *testing.T) {
	app, db := setupPreferencesApp(t)
	_, err := db.Exec(`INSERT INTO user_preferences (user_id, custom_css, created_at, updated_at)
		VALUES ('user-2', 'body { color: red }', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
	require.NoError(t, err)

	_, prefs, _ := send(t, app, httptest.NewRequest("GET", "/preferences", nil))
	assert.Empty(t, prefs.CustomCSS)

	status, _, _ := putJSON(t, app, "/preferences", `{"custom_css":"body { color: blue }"}`)
	require.Equal(t, fiber.StatusOK, status)
	var theirs string
	require.NoError(t, db.QueryRow(`SELECT custom_css FROM user_preferences WHERE user_id = 'user-2'`).Scan(&theirs))
	assert.Equal(t, "body { color: red }", theirs)
}

func TestPreferences_Language(t *testing.T) {
	app, _ := setupPreferencesApp(t)

	// Without a row it follows the browser, and says so with null
	status, _, raw := send(t, app, httptest.NewRequest("GET", "/preferences", nil))
	require.Equal(t, fiber.StatusOK, status)
	assert.Contains(t, raw, `"language":null`)

	status, prefs, _ := putJSON(t, app, "/preferences", `{"language":"nl"}`)
	require.Equal(t, fiber.StatusOK, status)
	assert.Equal(t, new("nl"), prefs.Language)

	// Another change leaves it alone
	status, prefs, _ = putJSON(t, app, "/preferences", `{"theme_mode":"dark"}`)
	require.Equal(t, fiber.StatusOK, status)
	assert.Equal(t, new("nl"), prefs.Language)
	_, prefs, _ = send(t, app, httptest.NewRequest("GET", "/preferences", nil))
	assert.Equal(t, new("nl"), prefs.Language)

	// null goes back to the browser's language
	status, _, raw = putJSON(t, app, "/preferences", `{"language":null}`)
	require.Equal(t, fiber.StatusOK, status)
	assert.Contains(t, raw, `"language":null`)
}

func TestPreferences_LanguageValidation(t *testing.T) {
	for _, body := range []string{`{"language":"xx"}`, `{"language":""}`, `{"language":"EN"}`, `{"language":"en-US"}`, `{"language":"en nl"}`} {
		t.Run(body, func(t *testing.T) {
			app, _ := setupPreferencesApp(t)
			status, _, _ := putJSON(t, app, "/preferences", `{"language":"de"}`)
			require.Equal(t, fiber.StatusOK, status)

			status, _, raw := putJSON(t, app, "/preferences", body)
			require.Equal(t, fiber.StatusBadRequest, status)
			assert.JSONEq(t, `{"error":"Validation failed","fields":{"language":"language must be one of: en nl de fr es"}}`, raw)

			// Nothing was stored
			_, prefs, _ := send(t, app, httptest.NewRequest("GET", "/preferences", nil))
			assert.Equal(t, new("de"), prefs.Language)
		})
	}
}

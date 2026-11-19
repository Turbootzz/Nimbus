package widgets

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Every type with secrets must take part in a dashboard import
func TestSecretConfigTypesAcceptMaskedConfigs(t *testing.T) {
	for typ, w := range registry {
		if _, ok := w.(SecretConfig); ok {
			assert.Implements(t, (*MaskedConfig)(nil), w, typ)
		}
	}
}

func TestCustomAPIMaskedConfig(t *testing.T) {
	var called bool
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	defer server.Close()

	w, _ := Get("custom_api")
	masked := w.(MaskedConfig)
	config := json.RawMessage(`{"url":"` + server.URL + `","headers":[{"name":"X-Key","value":"********"}],"fields":[{"label":"A","path":"a"}]}`)

	_, err := w.Validate(config)
	require.ErrorContains(t, err, "enter the value of header 1 again")
	stored, err := masked.ValidateMasked(config)
	require.NoError(t, err)
	assert.True(t, masked.HasMasks(stored))

	// The rest of the config is still checked
	_, err = masked.ValidateMasked(json.RawMessage(`{"url":"ftp://x","headers":[{"name":"X-Key","value":"********"}],"fields":[{"label":"A","path":"a"}]}`))
	assert.ErrorContains(t, err, "invalid URL")

	// The mask is never sent as if it were the secret
	_, err = w.(Fetcher).Fetch(context.Background(), &FetchRequest{Config: stored, Client: server.Client()})
	assert.ErrorContains(t, err, "enter its header values again")
	assert.False(t, called)

	real, err := w.Validate(json.RawMessage(`{"url":"` + server.URL + `","headers":[{"name":"X-Key","value":"k"}],"fields":[{"label":"A","path":"a"}]}`))
	require.NoError(t, err)
	assert.False(t, masked.HasMasks(real))
}

func TestCalendarMaskedConfig(t *testing.T) {
	w, _ := Get("calendar")
	masked := w.(MaskedConfig)
	config := json.RawMessage(`{"ical_urls":["https://calendar.example.test/********"]}`)

	stored, err := masked.ValidateMasked(config)
	require.NoError(t, err)
	assert.True(t, masked.HasMasks(stored))
	assert.False(t, masked.HasMasks(json.RawMessage(`{"ical_urls":["https://calendar.example.test/a.ics"]}`)))

	// The feed is reported as failed without a request to its host
	_, err = w.(Fetcher).Fetch(context.Background(), &FetchRequest{Config: stored, Client: http.DefaultClient})
	assert.ErrorContains(t, err, "enter the calendar URL again")
}

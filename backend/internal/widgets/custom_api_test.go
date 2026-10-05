package widgets

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCustomAPIValidate(t *testing.T) {
	field := `{"label":"Users","path":"users.length"}`
	runConfigCases(t, "custom_api", []configCase{
		{"minimal", `{"url":" http://192.168.1.10/api ","fields":[` + field + `]}`,
			`{"url":"http://192.168.1.10/api","headers":[],"fields":[{"label":"Users","path":"users.length","unit":""}],"verify_tls":true}`, ""},
		{"trimmed, empty rows dropped", `{"url":"http://10.0.0.2","verify_tls":false,
			"headers":[{"name":" X-Api-Key ","value":" secret "},{"name":" ","value":""}],
			"fields":[{"label":" CPU ","path":" $.cpu ","unit":" % "},{"label":"","path":"","unit":""}]}`,
			`{"url":"http://10.0.0.2","headers":[{"name":"X-Api-Key","value":"secret"}],"fields":[{"label":"CPU","path":"$.cpu","unit":"%"}],"verify_tls":false}`, ""},
		{"no url", `{"fields":[` + field + `]}`, "", "invalid URL"},
		{"not http", `{"url":"file:///etc/passwd","fields":[` + field + `]}`, "", "invalid URL"},
		{"cloud metadata", `{"url":"http://169.254.169.254/latest","fields":[` + field + `]}`, "", "invalid URL"},
		{"no fields", `{"url":"http://10.0.0.2"}`, "", "add 1 to 4 values"},
		{"too many fields", `{"url":"http://10.0.0.2","fields":[` + strings.Repeat(field+",", 4) + field + `]}`, "", "add 1 to 4 values"},
		{"missing label", `{"url":"http://10.0.0.2","fields":[{"path":"a"}]}`, "", "value 1 needs a label"},
		{"duplicate label", `{"url":"http://10.0.0.2","fields":[{"label":"A","path":"a"},{"label":"A","path":"b"}]}`, "", `label "A" is used twice`},
		{"bad path", `{"url":"http://10.0.0.2","fields":[{"label":"A","path":"a..b"}]}`, "", "value 1: path"},
		{"empty path", `{"url":"http://10.0.0.2","fields":[{"label":"A","path":" "}]}`, "", "path is empty"},
		{"long unit", `{"url":"http://10.0.0.2","fields":[{"label":"A","path":"a","unit":"abcdefghijk"}]}`, "", "unit must be"},
		{"bad header name", `{"url":"http://10.0.0.2","headers":[{"name":"X Key","value":"v"}],"fields":[` + field + `]}`, "", "header 1 needs a valid name"},
		{"host header", `{"url":"http://10.0.0.2","headers":[{"name":"host","value":"a.lan"}],"fields":[` + field + `]}`, "", "put the host in the URL"},
		{"masked value", `{"url":"http://10.0.0.2","headers":[{"name":"X-Key","value":"********"}],"fields":[` + field + `]}`, "", "enter the value of header 1 again"},
		{"header injection", `{"url":"http://10.0.0.2","headers":[{"name":"X-Key","value":"a\r\nHost: evil"}],"fields":[` + field + `]}`, "", "header 1 has an invalid value"},
	})
}

func TestCustomAPIRedact(t *testing.T) {
	w, _ := Get("custom_api")
	secret := w.(SecretConfig)
	stored := json.RawMessage(`{"url":"http://10.0.0.2","headers":[{"name":"X-Api-Key","value":"k1"},{"name":"X-Empty","value":""}],"fields":[{"label":"A","path":"a","unit":""}],"verify_tls":true}`)

	shown := secret.Redact(stored)
	assert.NotContains(t, string(shown), "k1")
	assert.JSONEq(t, `[{"name":"X-Api-Key","value":"********"},{"name":"X-Empty","value":""}]`, headersOf(t, shown))

	// Sent back unchanged: the stored value returns. Matching ignores case.
	back := secret.Unredact(json.RawMessage(strings.Replace(string(shown), "X-Api-Key", "x-api-key", 1)), stored)
	assert.JSONEq(t, `[{"name":"x-api-key","value":"k1"},{"name":"X-Empty","value":""}]`, headersOf(t, back))

	// A new value wins; a renamed header stays masked, so Validate asks for it
	edited := `{"url":"http://10.0.0.2","headers":[{"name":"X-Api-Key","value":"k2"},{"name":"X-Other","value":"********"}],"fields":[{"label":"A","path":"a"}]}`
	back = secret.Unredact(json.RawMessage(edited), stored)
	assert.JSONEq(t, `[{"name":"X-Api-Key","value":"k2"},{"name":"X-Other","value":"********"}]`, headersOf(t, back))
	_, err := w.Validate(back)
	assert.ErrorContains(t, err, "enter the value of header 2 again")
}

func headersOf(t *testing.T, config json.RawMessage) string {
	t.Helper()
	var cfg struct {
		Headers json.RawMessage `json:"headers"`
	}
	require.NoError(t, json.Unmarshal(config, &cfg))
	return string(cfg.Headers)
}

func fetchCustomAPI(t *testing.T, handler http.HandlerFunc, fields string) (any, error) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	w, _ := Get("custom_api")
	config := `{"url":"` + server.URL + `/stats?key=secret","headers":[{"name":"X-Api-Key","value":"k1"}],"fields":` + fields + `}`
	return w.(Fetcher).Fetch(context.Background(), &FetchRequest{Config: json.RawMessage(config), Client: http.DefaultClient})
}

func TestCustomAPIFetch(t *testing.T) {
	got, err := fetchCustomAPI(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "k1", r.Header.Get("X-Api-Key"))
		assert.Equal(t, "application/json", r.Header.Get("Accept"))
		_, _ = w.Write([]byte("\xef\xbb\xbf" + `{"users":[1,2,3],"name":"nas","up":true,"none":null}`))
	}, `[{"label":"Users","path":"users.length"},{"label":"Name","path":"name"},{"label":"Up","path":"up"},{"label":"None","path":"none"}]`)
	require.NoError(t, err, "a byte order mark is fine")
	raw, err := json.Marshal(got)
	require.NoError(t, err)
	assert.JSONEq(t, `{"kpis":{"Users":3,"Name":"nas","Up":true,"None":null}}`, string(raw))

	// A missing key leaves the other values live
	got, err = fetchCustomAPI(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"a":1}`))
	}, `[{"label":"A","path":"a"},{"label":"Errors","path":"errors"}]`)
	require.NoError(t, err)
	raw, _ = json.Marshal(got)
	assert.JSONEq(t, `{"kpis":{"A":1},"missing":["Errors"]}`, string(raw))
}

func TestCustomAPIFetchHeaders(t *testing.T) {
	w, _ := Get("custom_api")
	fetch := func(url, headers string) error {
		config := `{"url":"` + url + `","headers":` + headers + `,"fields":[{"label":"A","path":"a"}]}`
		_, err := w.(Fetcher).Fetch(context.Background(), &FetchRequest{Config: json.RawMessage(config), Client: http.DefaultClient})
		return err
	}

	var gotKey, gotAccept string
	other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey, gotAccept = r.Header.Get("X-Api-Key"), r.Header.Get("Accept")
		_, _ = w.Write([]byte(`{"a":1}`))
	}))
	t.Cleanup(other.Close)
	redirect := httptest.NewServer(http.RedirectHandler(other.URL, http.StatusFound))
	t.Cleanup(redirect.Close)

	// The key never follows a redirect to another host
	err := fetch(redirect.URL, `[{"name":"X-Api-Key","value":"k1"}]`)
	assert.ErrorContains(t, err, "redirected to another host")
	assert.Empty(t, gotKey)

	// Without headers a redirect is fine, and a custom Accept replaces ours
	require.NoError(t, fetch(redirect.URL, `[]`))
	require.NoError(t, fetch(other.URL, `[{"name":"Accept","value":"application/vnd.app+json"}]`))
	assert.Equal(t, "application/vnd.app+json", gotAccept)
}

func TestCustomAPIFetchErrors(t *testing.T) {
	ok := func(body string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(body)) }
	}
	cases := map[string]struct {
		handler http.HandlerFunc
		fields  string
		wantErr string
	}{
		"nothing found": {ok(`{"a":1}`), `[{"label":"B","path":"b"}]`, "nothing found at b"},
		"object":        {ok(`{"a":{"b":1}}`), `[{"label":"A","path":"a"}]`, "A: a is a list or object"},
		"list":          {ok(`{"a":[1]}`), `[{"label":"A","path":"a"}]`, "is a list or object"},
		"not json":      {ok(`<html>`), `[{"label":"A","path":"a"}]`, "not valid JSON"},
		"bad status": {func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusUnauthorized) },
			`[{"label":"A","path":"a"}]`, "unexpected status 401"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := fetchCustomAPI(t, tc.handler, tc.fields)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
		})
	}

	// A failed request never shows the URL, whose query may hold a key
	w, _ := Get("custom_api")
	_, err := w.(Fetcher).Fetch(context.Background(), &FetchRequest{
		Config: json.RawMessage(`{"url":"http://127.0.0.1:1/x?key=secret","fields":[{"label":"A","path":"a"}]}`),
		Client: http.DefaultClient,
	})
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "secret")
}

func TestVerifiesTLS(t *testing.T) {
	assert.True(t, VerifiesTLS(json.RawMessage(`{}`)))
	assert.True(t, VerifiesTLS(json.RawMessage(`{"verify_tls":true}`)))
	assert.True(t, VerifiesTLS(json.RawMessage(`{"verify_tls":"no"}`)), "only an explicit false turns it off")
	assert.False(t, VerifiesTLS(json.RawMessage(`{"verify_tls":false}`)))
}

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
			`{"url":"http://192.168.1.10/api","headers":[],"fields":[{"label":"Users","path":"users.length","unit":""}]}`, ""},
		{"trimmed", `{"url":"http://10.0.0.2","headers":[{"name":" X-Api-Key ","value":" secret "}],"fields":[{"label":" CPU ","path":" $.cpu ","unit":" % "}]}`,
			`{"url":"http://10.0.0.2","headers":[{"name":"X-Api-Key","value":"secret"}],"fields":[{"label":"CPU","path":"$.cpu","unit":"%"}]}`, ""},
		{"no url", `{"fields":[` + field + `]}`, "", "invalid URL"},
		{"not http", `{"url":"file:///etc/passwd","fields":[` + field + `]}`, "", "invalid URL"},
		{"cloud metadata", `{"url":"http://169.254.169.254/latest","fields":[` + field + `]}`, "", "invalid URL"},
		{"no fields", `{"url":"http://10.0.0.2"}`, "", "add 1 to 4 values"},
		{"too many fields", `{"url":"http://10.0.0.2","fields":[` + strings.Repeat(field+",", 4) + field + `]}`, "", "add 1 to 4 values"},
		{"missing label", `{"url":"http://10.0.0.2","fields":[{"path":"a"}]}`, "", "value 1 needs a label"},
		{"bad path", `{"url":"http://10.0.0.2","fields":[{"label":"A","path":"a..b"}]}`, "", "value 1: path"},
		{"empty path", `{"url":"http://10.0.0.2","fields":[{"label":"A","path":" "}]}`, "", "path is empty"},
		{"long unit", `{"url":"http://10.0.0.2","fields":[{"label":"A","path":"a","unit":"abcdefghijk"}]}`, "", "unit must be"},
		{"bad header name", `{"url":"http://10.0.0.2","headers":[{"name":"X Key","value":"v"}],"fields":[` + field + `]}`, "", "header 1 needs a valid name"},
		{"header injection", `{"url":"http://10.0.0.2","headers":[{"name":"X-Key","value":"a\r\nHost: evil"}],"fields":[` + field + `]}`, "", "header 1 has an invalid value"},
	})
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
		_, _ = w.Write([]byte(`{"users":[1,2,3],"big":9007199254740993,"name":"nas","up":true,"none":null}`))
	}, `[{"label":"Users","path":"users.length"},{"label":"Big","path":"big"},{"label":"Name","path":"name"},{"label":"Up","path":"up"}]`)
	require.NoError(t, err)
	raw, err := json.Marshal(got)
	require.NoError(t, err)
	assert.JSONEq(t, `{"kpis":{"0":3,"1":9007199254740993,"2":"nas","3":true}}`, string(raw))
	assert.Contains(t, string(raw), "9007199254740993", "big integers stay exact")

	got, err = fetchCustomAPI(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"none":null}`))
	}, `[{"label":"None","path":"none"}]`)
	require.NoError(t, err)
	raw, _ = json.Marshal(got)
	assert.JSONEq(t, `{"kpis":{"0":null}}`, string(raw))
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
		"missing path": {ok(`{"a":1}`), `[{"label":"B","path":"b"}]`, "B: nothing found at b"},
		"object":       {ok(`{"a":{"b":1}}`), `[{"label":"A","path":"a"}]`, "A: a is a list or object"},
		"list":         {ok(`{"a":[1]}`), `[{"label":"A","path":"a"}]`, "is a list or object"},
		"not json":     {ok(`<html>`), `[{"label":"A","path":"a"}]`, "not valid JSON"},
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

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

func TestWeatherValidate(t *testing.T) {
	runConfigCases(t, "weather", []configCase{
		{"defaults to metric", `{"latitude":52.37,"longitude":4.89,"location":" Amsterdam "}`,
			`{"latitude":52.37,"longitude":4.89,"location":"Amsterdam","units":"metric"}`, ""},
		{"imperial", `{"latitude":0,"longitude":0,"units":"imperial"}`,
			`{"latitude":0,"longitude":0,"location":"","units":"imperial"}`, ""},
		{"no location", `{}`, "", "pick a location"},
		{"latitude out of range", `{"latitude":91,"longitude":0}`, "", "latitude must be between"},
		{"longitude out of range", `{"latitude":0,"longitude":-181}`, "", "longitude between"},
		{"bad units", `{"latitude":0,"longitude":0,"units":"kelvin"}`, "", "metric or imperial"},
	})
}

// fakeOpenMeteo points openMeteoURL at a test server for one test
func fakeOpenMeteo(t *testing.T, handler http.HandlerFunc) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	old := openMeteoURL
	openMeteoURL = server.URL
	t.Cleanup(func() { openMeteoURL = old })
}

func fetchWeather(t *testing.T, config string) (any, error) {
	t.Helper()
	w, _ := Get("weather")
	return w.(Fetcher).Fetch(context.Background(), &FetchRequest{Config: json.RawMessage(config), Client: http.DefaultClient})
}

const openMeteoFixture = `{
	"current": {"temperature_2m": 12.4, "apparent_temperature": 10.1, "weather_code": 3, "wind_speed_10m": 14.2, "is_day": 1},
	"current_units": {"temperature_2m": "°C", "wind_speed_10m": "km/h"},
	"daily": {
		"time": ["2026-09-27", "2026-09-28"],
		"weather_code": [3, 61],
		"temperature_2m_max": [14.1, 13.0],
		"temperature_2m_min": [8.2, 9.5]
	}
}`

func TestWeatherFetch(t *testing.T) {
	var query string
	fakeOpenMeteo(t, func(w http.ResponseWriter, r *http.Request) {
		query = r.URL.RawQuery
		assert.Equal(t, "/v1/forecast", r.URL.Path)
		_, _ = w.Write([]byte(openMeteoFixture))
	})

	got, err := fetchWeather(t, `{"latitude":52.37,"longitude":4.89,"units":"metric"}`)
	require.NoError(t, err)
	assert.Contains(t, query, "latitude=52.3700&longitude=4.8900")
	assert.NotContains(t, query, "fahrenheit")

	raw, err := json.Marshal(got)
	require.NoError(t, err)
	assert.JSONEq(t, `{
		"temperature": 12.4, "feels_like": 10.1, "weather_code": 3, "wind_speed": 14.2, "is_day": true,
		"temperature_unit": "°C", "wind_unit": "km/h",
		"daily": [
			{"date": "2026-09-27", "weather_code": 3, "max": 14.1, "min": 8.2},
			{"date": "2026-09-28", "weather_code": 61, "max": 13.0, "min": 9.5}
		]
	}`, string(raw))

	_, err = fetchWeather(t, `{"latitude":52.37,"longitude":4.89,"units":"imperial"}`)
	require.NoError(t, err)
	assert.Contains(t, query, "temperature_unit=fahrenheit&wind_speed_unit=mph")
}

func TestWeatherFetchErrors(t *testing.T) {
	cases := map[string]struct {
		handler http.HandlerFunc
		wantErr string
	}{
		"bad status": {func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusTooManyRequests) }, "unexpected status 429"},
		"not json":   {func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("<html>")) }, "not valid JSON"},
		"too large": {func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"x":"` + strings.Repeat("a", maxFetchBodyBytes) + `"}`))
		}, "larger than 1 MB"},
		"short daily arrays": {func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"daily":{"time":["a","b"],"weather_code":[1],"temperature_2m_max":[1,2],"temperature_2m_min":[1,2]}}`))
		}, ""},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			fakeOpenMeteo(t, tc.handler)
			got, err := fetchWeather(t, `{"latitude":1,"longitude":1}`)
			if tc.wantErr == "" {
				require.NoError(t, err)
				assert.Len(t, got.(weatherPayload).Daily, 1, "stops at the shortest array")
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
		})
	}
}

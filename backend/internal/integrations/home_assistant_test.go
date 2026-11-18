package integrations

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/nimbus/backend/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHomeAssistant(t *testing.T) {
	impl := registered(t, "home_assistant")
	auth := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Authorization") != "Bearer ha-token" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			next(w, r)
		}
	}
	server := fakeApp(t, map[string]http.HandlerFunc{
		"GET /api/": auth(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, `{"message":"API running."}`) }),
		"GET /api/states": auth(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, `[
				{"entity_id":"person.anna","state":"home","attributes":{"x":[1,2]}},
				{"entity_id":"person.bob","state":"not_home"},
				{"entity_id":"light.kitchen","state":"on"},
				{"entity_id":"light.hall","state":"off"},
				{"entity_id":"light.desk","state":"on"},
				{"entity_id":"switch.fan","state":"on"},
				{"entity_id":"sensor.temp","state":"on"},
				{"entity_id":"sensor.living_temp","state":"21.5","attributes":{"friendly_name":"Living room","unit_of_measurement":"°C"}},
				{"entity_id":"alarm_control_panel.home","state":"armed_away","attributes":{"friendly_name":["odd"],"unit_of_measurement":null}}
			]`)
		}),
	})
	ctx := context.Background()
	conn := newTestConn(server.URL, models.IntegrationCredentials{Token: "ha-token"})

	require.NoError(t, impl.Test(ctx, conn))
	payload, err := impl.Fetch(ctx, conn)
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"people_home": 1, "lights_on": 2, "switches_on": 1}, payload.KPIs)
	assert.Nil(t, payload.KPIDefs, "the kind's own KPIs apply")
	assert.ErrorIs(t, impl.Test(ctx, newTestConn(server.URL, models.IntegrationCredentials{Token: "x"})), errRejected)
	assertUnreachable(t, impl, "/api/")

	// An extra entity goes by its friendly name
	living := KPI{Key: "entity_1", Label: "Living room", Unit: "°C"}
	conn.Options = json.RawMessage(`{"entity_1":"sensor.living_temp"}`)
	payload, err = impl.Fetch(ctx, conn)
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"people_home": 1, "lights_on": 2, "switches_on": 1, "entity_1": 21.5}, payload.KPIs)
	assert.Equal(t, []KPI{living}, payload.KPIDefs)

	// Without a usable friendly name the entity ID is the label
	conn.Options = json.RawMessage(`{"entity_1":"sensor.living_temp","entity_2":"alarm_control_panel.home"}`)
	payload, err = impl.Fetch(ctx, conn)
	require.NoError(t, err)
	assert.Equal(t, map[string]any{
		"people_home": 1, "lights_on": 2, "switches_on": 1, "entity_1": 21.5, "entity_2": "armed_away",
	}, payload.KPIs)
	assert.Equal(t, []KPI{living, {Key: "entity_2", Label: "alarm_control_panel.home"}}, payload.KPIDefs)

	// The same entity twice fills both places
	conn.Options = json.RawMessage(`{"entity_1":"sensor.living_temp","entity_2":"sensor.living_temp"}`)
	payload, err = impl.Fetch(ctx, conn)
	require.NoError(t, err)
	assert.Equal(t, 21.5, payload.KPIs["entity_2"])
	assert.Equal(t, []KPI{living, {Key: "entity_2", Label: "Living room", Unit: "°C"}}, payload.KPIDefs)

	// An entity Home Assistant doesn't have keeps its place, without a
	// value; its label is the entity ID, cut like any other
	gone := "sensor." + strings.Repeat("x", 248)
	conn.Options = json.RawMessage(`{"entity_1":"` + gone + `"}`)
	payload, err = impl.Fetch(ctx, conn)
	require.NoError(t, err)
	assert.NotContains(t, payload.KPIs, "entity_1")
	assert.Equal(t, []KPI{{Key: "entity_1", Label: gone[:40]}}, payload.KPIDefs)
}

func TestHomeAssistantValue(t *testing.T) {
	assert.Equal(t, 21.5, homeAssistantValue("21.5", true))
	assert.Equal(t, "unavailable", homeAssistantValue("unavailable", true))
	assert.Equal(t, "NaN", homeAssistantValue("NaN", true), "not a number JSON can carry")
	assert.Equal(t, "2024.10", homeAssistantValue("2024.10", false), "without a unit it is text, like a version")
	assert.Len(t, homeAssistantValue(strings.Repeat("é", 300), false), 48, "long text is cut by characters")
}

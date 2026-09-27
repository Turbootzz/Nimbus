package integrations

import (
	"context"
	"fmt"
	"net/http"
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
				{"entity_id":"sensor.temp","state":"on"}
			]`)
		}),
	})
	ctx := context.Background()
	conn := newTestConn(server.URL, models.IntegrationCredentials{Token: "ha-token"})

	require.NoError(t, impl.Test(ctx, conn))
	payload, err := impl.Fetch(ctx, conn)
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"people_home": 1, "lights_on": 2, "switches_on": 1}, payload.KPIs)
	assert.ErrorIs(t, impl.Test(ctx, newTestConn(server.URL, models.IntegrationCredentials{Token: "x"})), errRejected)
}

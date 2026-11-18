package integrations

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/nimbus/backend/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProxmox(t *testing.T) {
	impl := registered(t, "proxmox")
	auth := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Authorization") != "PVEAPIToken=nimbus@pve!ro=abc-123" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			next(w, r)
		}
	}
	server := fakeApp(t, map[string]http.HandlerFunc{
		"GET /api2/json/version": auth(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, `{"data":{"version":"8.2"}}`) }),
		"GET /api2/json/cluster/resources": auth(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, `{"data":[
				{"type":"qemu","node":"pve1","status":"running","template":0},
				{"type":"qemu","node":"pve2","status":"stopped","template":0},
				{"type":"qemu","node":"pve1","status":"stopped","template":1},
				{"type":"lxc","node":"pve2","status":"running"},
				{"type":"node","node":"pve1","status":"online","cpu":0.5,"maxcpu":4,"mem":4,"maxmem":16},
				{"type":"node","node":"pve2","status":"online","cpu":0.25,"maxcpu":4,"mem":4,"maxmem":16},
				{"type":"node","node":"pve3","status":"offline","cpu":0,"maxcpu":8,"mem":0,"maxmem":32},
				{"type":"storage","node":"pve1","status":"available"},
				{"type":"storage","node":"pve4","status":"available"}
			]}`)
		}),
	})
	ctx := context.Background()

	// The token works with or without the PVEAPIToken= prefix
	for _, token := range []string{"nimbus@pve!ro=abc-123", "PVEAPIToken=nimbus@pve!ro=abc-123"} {
		conn := newTestConn(server.URL, models.IntegrationCredentials{Token: token})
		require.NoError(t, impl.Test(ctx, conn))
		payload, err := impl.Fetch(ctx, conn)
		require.NoError(t, err)
		assert.Equal(t, map[string]any{"vms": "1/2", "containers": "1/1", "cpu": 38, "memory": 25}, payload.KPIs)
	}
	assert.ErrorIs(t, impl.Test(ctx, newTestConn(server.URL, models.IntegrationCredentials{Token: "bad"})), errRejected)

	// The node option narrows everything to one node
	conn := newTestConn(server.URL, models.IntegrationCredentials{Token: "nimbus@pve!ro=abc-123"})
	conn.Options = json.RawMessage(`{"node":"PVE2"}`)
	require.NoError(t, impl.Test(ctx, conn))
	payload, err := impl.Fetch(ctx, conn)
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"vms": "0/1", "containers": "1/1", "cpu": 25, "memory": 25}, payload.KPIs)

	// An offline node is still a node; a misspelled one is not
	conn.Options = json.RawMessage(`{"node":"pve3"}`)
	payload, err = impl.Fetch(ctx, conn)
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"vms": "0/0", "containers": "0/0", "cpu": 0, "memory": 0}, payload.KPIs)
	// A token that may not see a node itself still sees what runs on it
	conn.Options = json.RawMessage(`{"node":"pve4"}`)
	require.NoError(t, impl.Test(ctx, conn))
	conn.Options = json.RawMessage(`{"node":"pve9"}`)
	assert.EqualError(t, impl.Test(ctx, conn), `node "pve9" was not found in the cluster`)

	assertUnreachable(t, impl, "/api2/json/version")
}

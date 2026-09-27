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
				{"type":"qemu","status":"running","template":0},
				{"type":"qemu","status":"stopped","template":0},
				{"type":"qemu","status":"stopped","template":1},
				{"type":"lxc","status":"running"},
				{"type":"node","status":"online","cpu":0.5,"maxcpu":4,"mem":4,"maxmem":16},
				{"type":"node","status":"online","cpu":0.25,"maxcpu":4,"mem":4,"maxmem":16},
				{"type":"node","status":"offline","cpu":0,"maxcpu":8,"mem":0,"maxmem":32},
				{"type":"storage","status":"available"}
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
}

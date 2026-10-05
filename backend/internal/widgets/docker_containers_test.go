package widgets

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nimbus/backend/internal/integrations"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDockerContainersFetch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/containers/json", r.URL.Path)
		_, _ = w.Write([]byte(`[
			{"Names":["/web"],"Image":"nginx","State":"running","Status":"Up 2 days"},
			{"Names":["/old"],"Image":"redis","State":"exited","Status":"Exited (0) 1 day ago"},
			{"Names":["/loop"],"Image":"app","State":"restarting","Status":"Restarting (1) 5 seconds ago"}
		]`))
	}))
	t.Cleanup(server.Close)
	conn := &integrations.Conn{BaseURL: server.URL, Client: http.DefaultClient, State: &integrations.State{}}
	w, _ := Get("docker_containers")
	fetch := func(config string) (dockerContainersPayload, error) {
		got, err := w.(Fetcher).Fetch(context.Background(), &FetchRequest{Config: json.RawMessage(config), Integration: conn})
		if err != nil {
			return dockerContainersPayload{}, err
		}
		return got.(dockerContainersPayload), nil
	}

	all, err := fetch(`{}`)
	require.NoError(t, err)
	assert.Equal(t, 1, all.Running)
	assert.Equal(t, 3, all.Total)
	assert.Len(t, all.Containers, 3)

	running, err := fetch(`{"hide_stopped":true}`)
	require.NoError(t, err)
	// A restarting container is not stopped, so it stays visible
	assert.Equal(t, []string{"web", "loop"}, []string{running.Containers[0].Name, running.Containers[1].Name})
	assert.Len(t, running.Containers, 2)
	assert.Equal(t, 3, running.Total, "the total still counts stopped ones")

	_, err = w.(Fetcher).Fetch(context.Background(), &FetchRequest{Config: json.RawMessage(`{}`)})
	assert.EqualError(t, err, "this widget has no Docker integration")
}

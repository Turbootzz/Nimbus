package integrations

import (
	"cmp"
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/nimbus/backend/internal/models"
)

// Docker reads the Docker Engine API, through the socket the server allows
// in DOCKER_SOCKET or a socket proxy over HTTP. It only ever sends GETs.
// Admin only: the container list shows what runs on the host.
type docker struct{}

func init() { Register(docker{}) }

func (docker) Kind() string { return "docker" }

func (docker) Meta() Meta {
	return Meta{
		Name:        "Docker",
		Icon:        "docker",
		DefaultPort: 2375,
		AuthTypes:   []string{models.IntegrationAuthNone},
		KPIs: []KPI{
			{Key: "running", Label: "Running"},
			{Key: "stopped", Label: "Stopped"},
			{Key: "total", Label: "Total"},
		},
		URLHint:      "unix:///var/run/docker.sock",
		AdminOnly:    true,
		DockerSocket: true,
	}
}

// DockerContainer is one container, for the containers widget
type DockerContainer struct {
	ID     string `json:"id"` // short id, unique on the host
	Name   string `json:"name"`
	Image  string `json:"image"`
	State  string `json:"state"`  // running, exited, paused, ...
	Status string `json:"status"` // e.g. "Up 3 days"
}

// Stopped reports whether the container is not running and not about to:
// paused and restarting containers are neither running nor stopped
func (c DockerContainer) Stopped() bool {
	return c.State == "exited" || c.State == "created" || c.State == "dead"
}

// DockerContainers lists all containers: running ones first, then by name
func DockerContainers(ctx context.Context, conn *Conn) ([]DockerContainer, error) {
	type apiContainer struct {
		ID     string `json:"Id"`
		Names  []string
		Image  string
		State  string
		Status string
	}
	containers := []DockerContainer{}
	err := eachJSON(ctx, conn, "/containers/json?all=1", nil, func(c apiContainer) {
		name := ""
		if len(c.Names) > 0 {
			name = strings.TrimPrefix(c.Names[0], "/")
		}
		containers = append(containers, DockerContainer{ID: shortID(c.ID), Name: name, Image: c.Image, State: c.State, Status: c.Status})
	})
	if err != nil {
		return nil, err
	}
	notRunning := func(c DockerContainer) int {
		if c.State == "running" {
			return 0
		}
		return 1
	}
	slices.SortFunc(containers, func(a, b DockerContainer) int {
		return cmp.Or(cmp.Compare(notRunning(a), notRunning(b)), strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name)))
	})
	return containers, nil
}

func (docker) Test(ctx context.Context, conn *Conn) error {
	var version struct{ APIVersion string }
	if err := getJSON(ctx, conn, "/version", nil, &version); err != nil {
		return err
	}
	if version.APIVersion == "" {
		return errors.New("/version did not answer like the Docker API")
	}
	return nil
}

func (docker) Fetch(ctx context.Context, conn *Conn) (*Payload, error) {
	containers, err := DockerContainers(ctx, conn)
	if err != nil {
		return nil, err
	}
	running, stopped := 0, 0
	for _, c := range containers {
		switch {
		case c.State == "running":
			running++
		case c.Stopped():
			stopped++
		}
	}
	return &Payload{KPIs: map[string]any{
		"running": running,
		"stopped": stopped,
		"total":   len(containers),
	}}, nil
}

// shortID is the 12 character id the docker CLI shows
func shortID(id string) string {
	return id[:min(len(id), 12)]
}

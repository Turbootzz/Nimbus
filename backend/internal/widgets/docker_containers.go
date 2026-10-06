package widgets

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/nimbus/backend/internal/integrations"
)

type dockerContainersConfig struct {
	HideStopped bool `json:"hide_stopped"`
}

type dockerContainersPayload struct {
	Containers []integrations.DockerContainer `json:"containers"`
	Running    int                            `json:"running"`
	Total      int                            `json:"total"`
}

// dockerContainers lists the containers of a Docker integration
type dockerContainers struct{}

func init() { Register(dockerContainers{}) }

func (dockerContainers) Type() string { return "docker_containers" }

func (dockerContainers) Meta() Meta {
	return Meta{
		Name:                  "Docker containers",
		Category:              CategoryInfo,
		DefaultSize:           "2x2",
		AllowedSizes:          allSizes,
		IntegrationKinds:      []string{"docker"},
		DefaultRefreshSeconds: 60,
		// The Docker integration it needs is admin only
		AdminOnly: true,
	}
}

func (dockerContainers) Validate(config json.RawMessage) (json.RawMessage, error) {
	cfg, err := decodeConfig[dockerContainersConfig](config)
	if err != nil {
		return nil, err
	}
	return encodeConfig(cfg)
}

func (dockerContainers) Fetch(ctx context.Context, req *FetchRequest) (any, error) {
	if req.Integration == nil {
		return nil, errors.New("this widget has no Docker integration")
	}
	cfg, err := decodeConfig[dockerContainersConfig](req.Config)
	if err != nil {
		return nil, err
	}
	containers, err := integrations.DockerContainers(ctx, req.Integration)
	if err != nil {
		return nil, err
	}

	payload := dockerContainersPayload{Containers: []integrations.DockerContainer{}, Total: len(containers)}
	for _, c := range containers {
		if c.State == "running" {
			payload.Running++
		}
		if !cfg.HideStopped || !c.Stopped() {
			payload.Containers = append(payload.Containers, c)
		}
	}
	return payload, nil
}

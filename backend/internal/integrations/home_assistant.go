package integrations

import (
	"context"
	"net/http"
	"strings"

	"github.com/nimbus/backend/internal/models"
)

type homeAssistant struct{}

func init() { Register(homeAssistant{}) }

func (homeAssistant) Kind() string { return "home_assistant" }

func (homeAssistant) Meta() Meta {
	return Meta{
		Name:        "Home Assistant",
		Icon:        "home-assistant",
		DefaultPort: 8123,
		// A long-lived access token from the user's profile page
		AuthTypes: []string{models.IntegrationAuthToken},
		KPIs: []KPI{
			{Key: "people_home", Label: "People home"},
			{Key: "lights_on", Label: "Lights on"},
			{Key: "switches_on", Label: "Switches on"},
		},
	}
}

func homeAssistantAuth(conn *Conn) authFunc {
	return func(r *http.Request) { r.Header.Set("Authorization", "Bearer "+conn.Creds.Token) }
}

func (homeAssistant) Test(ctx context.Context, conn *Conn) error {
	var status struct {
		Message string `json:"message"`
	}
	return getJSON(ctx, conn, "/api/", homeAssistantAuth(conn), &status)
}

func (homeAssistant) Fetch(ctx context.Context, conn *Conn) (*Payload, error) {
	var people, lights, switches int
	// The states list can be large; count while reading instead of keeping it
	err := eachJSON(ctx, conn, "/api/states", homeAssistantAuth(conn), func(s struct {
		EntityID string `json:"entity_id"`
		State    string `json:"state"`
	}) {
		domain, _, _ := strings.Cut(s.EntityID, ".")
		switch {
		case domain == "person" && s.State == "home":
			people++
		case domain == "light" && s.State == "on":
			lights++
		case domain == "switch" && s.State == "on":
			switches++
		}
	})
	if err != nil {
		return nil, err
	}
	return &Payload{KPIs: map[string]any{
		"people_home": people,
		"lights_on":   lights,
		"switches_on": switches,
	}}, nil
}

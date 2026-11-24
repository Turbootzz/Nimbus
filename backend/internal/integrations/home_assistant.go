package integrations

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/nimbus/backend/internal/models"
)

type homeAssistant struct{}

func init() { Register(homeAssistant{}) }

func (homeAssistant) Kind() string { return "home_assistant" }

// The options that each name one extra entity, which are also the keys of
// their KPIs
var homeAssistantEntityOptions = []string{"entity_1", "entity_2"}

// A friendly name or entity ID longer than this is cut on the tile
const homeAssistantLabelLength = 40

func (homeAssistant) Meta() Meta {
	meta := Meta{
		Name:        "Home Assistant",
		Icon:        "home-assistant",
		DefaultPort: 8123,
		// A long-lived access token from the user's profile page
		AuthTypes: []string{models.IntegrationAuthToken},
		// A tile has four numbers: with two extra entities the last of
		// these gives way
		KPIs: []KPI{
			{Key: "people_home", Label: "People home"},
			{Key: "lights_on", Label: "Lights on"},
			{Key: "switches_on", Label: "Switches on"},
		},
	}
	for i, key := range homeAssistantEntityOptions {
		label := fmt.Sprintf("Entity %d", i+1)
		// Fetch replaces the label with the entity's friendly name
		meta.KPIs = append(meta.KPIs, KPI{Key: key, Label: label, Option: key})
		meta.Options = append(meta.Options, Option{
			Key:         key,
			Label:       "Extra " + strings.ToLower(label),
			Help:        "Shows this entity's state on the tile. A tile has four numbers, so a second entity replaces Switches on.",
			Placeholder: "sensor.living_room_temperature",
			MaxLength:   255,
			Pattern:     `[a-z0-9_]+\.[a-z0-9_]+`,
		})
	}
	return meta
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

type homeAssistantState struct {
	EntityID string `json:"entity_id"`
	State    string `json:"state"`
	// Only decoded for the extra entities
	Attributes json.RawMessage `json:"attributes"`
}

func (homeAssistant) Fetch(ctx context.Context, conn *Conn) (*Payload, error) {
	// The KPIs of the extra entities, by entity ID (one entity can be both)
	extras := map[string][]*KPI{}
	var defs []*KPI
	for _, key := range homeAssistantEntityOptions {
		if id := conn.Option(key); id != "" {
			// Until Fetch finds its friendly name the entity ID is the label
			kpi := &KPI{Key: key, Label: clip(id, homeAssistantLabelLength)}
			extras[id] = append(extras[id], kpi)
			defs = append(defs, kpi)
		}
	}

	values := map[string]any{}
	var people, lights, switches int
	// The states list can be large; count while reading instead of keeping it
	err := eachJSON(ctx, conn, "/api/states", homeAssistantAuth(conn), func(s homeAssistantState) {
		if kpis := extras[s.EntityID]; len(kpis) > 0 {
			var attributes struct {
				FriendlyName any `json:"friendly_name"`
				Unit         any `json:"unit_of_measurement"`
			}
			// Attributes are free-form; odd ones just give no name or unit
			_ = json.Unmarshal(s.Attributes, &attributes)
			name, _ := attributes.FriendlyName.(string)
			unit, _ := attributes.Unit.(string)
			for _, kpi := range kpis {
				if name != "" {
					kpi.Label = clip(name, homeAssistantLabelLength)
				}
				kpi.Unit = clip(unit, 12)
				values[kpi.Key] = homeAssistantValue(s.State, unit != "")
			}
		}
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
	values["people_home"], values["lights_on"], values["switches_on"] = people, lights, switches

	payload := &Payload{KPIs: values}
	// The extra entities go by their own names
	for _, kpi := range defs {
		payload.KPIDefs = append(payload.KPIDefs, *kpi)
	}
	return payload, nil
}

// homeAssistantValue turns the state of an entity with a unit into a
// number, so the tile formats it like other numbers. Without a unit a state
// like "2024.10" (a version) or "0123" is text and stays as it is.
func homeAssistantValue(state string, hasUnit bool) any {
	if number, err := strconv.ParseFloat(state, 64); hasUnit && err == nil && !math.IsInf(number, 0) && !math.IsNaN(number) {
		return number
	}
	return clip(state, 24)
}

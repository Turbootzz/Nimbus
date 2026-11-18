package integrations

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"slices"
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

// Fixed KPIs, in tile order; with two extra entities the last one gives way
var homeAssistantFixedKPIs = []KPI{
	{Key: "people_home", Label: "People home"},
	{Key: "lights_on", Label: "Lights on"},
	{Key: "switches_on", Label: "Switches on"},
}

func (homeAssistant) Meta() Meta {
	meta := Meta{
		Name:        "Home Assistant",
		Icon:        "home-assistant",
		DefaultPort: 8123,
		// A long-lived access token from the user's profile page
		AuthTypes: []string{models.IntegrationAuthToken},
		KPIs:      slices.Clone(homeAssistantFixedKPIs),
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
	// Extra entities by entity ID, each with its KPI as far as it is known
	extras := map[string]*KPI{}
	var extraKPIs []*KPI
	for _, key := range homeAssistantEntityOptions {
		id := conn.Option(key)
		if id == "" || extras[id] != nil {
			continue
		}
		extras[id] = &KPI{Key: key, Label: id}
		extraKPIs = append(extraKPIs, extras[id])
	}

	kpis := map[string]any{}
	var people, lights, switches int
	// The states list can be large; count while reading instead of keeping it
	err := eachJSON(ctx, conn, "/api/states", homeAssistantAuth(conn), func(s homeAssistantState) {
		if kpi := extras[s.EntityID]; kpi != nil {
			kpis[kpi.Key] = homeAssistantValue(s.State)
			var attributes struct {
				FriendlyName any `json:"friendly_name"`
				Unit         any `json:"unit_of_measurement"`
			}
			// Attributes are free-form; odd ones just give no name or unit
			_ = json.Unmarshal(s.Attributes, &attributes)
			if name, _ := attributes.FriendlyName.(string); name != "" {
				kpi.Label = clip(name, 40)
			}
			unit, _ := attributes.Unit.(string)
			kpi.Unit = clip(unit, 12)
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
	kpis["people_home"], kpis["lights_on"], kpis["switches_on"] = people, lights, switches

	payload := &Payload{KPIs: kpis}
	if len(extraKPIs) > 0 {
		// A tile has four numbers: the extra entities come last and push
		// out the last fixed ones
		payload.KPIDefs = slices.Clone(homeAssistantFixedKPIs[:min(len(homeAssistantFixedKPIs), 4-len(extraKPIs))])
		for _, kpi := range extraKPIs {
			payload.KPIDefs = append(payload.KPIDefs, *kpi)
		}
	}
	return payload, nil
}

// homeAssistantValue turns a state into a number when it is one, so the
// tile formats it like other numbers
func homeAssistantValue(state string) any {
	if number, err := strconv.ParseFloat(state, 64); err == nil && !math.IsInf(number, 0) && !math.IsNaN(number) {
		return number
	}
	return clip(state, 24)
}

package integrations

import (
	"bytes"
	"context"
	"encoding/json"
	"math"
	"net/http"

	"github.com/nimbus/backend/internal/models"
)

// Pi-hole v6 (v5 used a different API and is not supported)
type pihole struct{}

func init() { Register(pihole{}) }

func (pihole) Kind() string { return "pihole" }

func (pihole) Meta() Meta {
	return Meta{
		Name:        "Pi-hole",
		Icon:        "pi-hole",
		DefaultPort: 80,
		// The app password; none when Pi-hole has no password set
		AuthTypes: []string{models.IntegrationAuthToken, models.IntegrationAuthNone},
		KPIs: []KPI{
			{Key: "queries", Label: "Queries"},
			{Key: "blocked", Label: "Blocked"},
			{Key: "blocked_percent", Label: "Blocked", Unit: "%"},
			{Key: "gravity", Label: "Blocklist"},
		},
	}
}

// piholeLogin opens a session and returns its sid ("" without a password)
func piholeLogin(ctx context.Context, conn *Conn) (string, error) {
	if conn.Creds.Token == "" {
		return "", nil
	}
	body, err := json.Marshal(map[string]string{"password": conn.Creds.Token})
	if err != nil {
		return "", err
	}
	var auth struct {
		Session struct {
			Valid bool   `json:"valid"`
			SID   string `json:"sid"`
		} `json:"session"`
	}
	err = doRequest(ctx, conn, http.MethodPost, "/api/auth", bytes.NewReader(body),
		func(r *http.Request) { r.Header.Set("Content-Type", "application/json") },
		func(resp *http.Response) error { return json.NewDecoder(resp.Body).Decode(&auth) })
	if err != nil {
		return "", err
	}
	if !auth.Session.Valid {
		return "", errRejected
	}
	return auth.Session.SID, nil
}

func piholeAuth(sid string) authFunc {
	return func(r *http.Request) {
		if sid != "" {
			r.Header.Set("X-FTL-SID", sid)
		}
	}
}

func (pihole) Test(ctx context.Context, conn *Conn) error {
	sid, err := piholeLogin(ctx, conn)
	if err != nil {
		return err
	}
	var version map[string]any
	err = getJSON(ctx, conn, "/api/info/version", piholeAuth(sid), &version)
	if sid != "" {
		// Pi-hole has few session seats; give this one back
		_ = doRequest(ctx, conn, http.MethodDelete, "/api/auth", nil, piholeAuth(sid), nil)
	}
	return err
}

func (pihole) Fetch(ctx context.Context, conn *Conn) (*Payload, error) {
	var summary struct {
		Queries struct {
			Total          int64   `json:"total"`
			Blocked        int64   `json:"blocked"`
			PercentBlocked float64 `json:"percent_blocked"`
		} `json:"queries"`
		Gravity struct {
			DomainsBeingBlocked int64 `json:"domains_being_blocked"`
		} `json:"gravity"`
	}
	err := withSession(conn, "sid",
		func() (string, error) { return piholeLogin(ctx, conn) },
		func(sid string) error { return getJSON(ctx, conn, "/api/stats/summary", piholeAuth(sid), &summary) })
	if err != nil {
		return nil, err
	}
	return &Payload{KPIs: map[string]any{
		"queries":         summary.Queries.Total,
		"blocked":         summary.Queries.Blocked,
		"blocked_percent": math.Round(summary.Queries.PercentBlocked*10) / 10,
		"gravity":         summary.Gravity.DomainsBeingBlocked,
	}}, nil
}

package integrations

import (
	"context"
	"math"
	"net/http"

	"github.com/nimbus/backend/internal/models"
)

type adguard struct{}

func init() { Register(adguard{}) }

func (adguard) Kind() string { return "adguard" }

func (adguard) Meta() Meta {
	return Meta{
		Name:        "AdGuard Home",
		Icon:        "adguard-home",
		DefaultPort: 3000,
		AuthTypes:   []string{models.IntegrationAuthBasic, models.IntegrationAuthNone},
		KPIs: []KPI{
			{Key: "queries", Label: "Queries"},
			{Key: "blocked", Label: "Blocked"},
			{Key: "blocked_percent", Label: "Blocked", Unit: "%"},
			{Key: "latency", Label: "Latency", Unit: "ms"},
		},
	}
}

func adguardAuth(conn *Conn) authFunc {
	return func(r *http.Request) {
		if conn.Creds.Username != "" {
			r.SetBasicAuth(conn.Creds.Username, conn.Creds.Password)
		}
	}
}

func (adguard) Test(ctx context.Context, conn *Conn) error {
	var status struct {
		Running bool `json:"running"`
	}
	return getJSON(ctx, conn, "/control/status", adguardAuth(conn), &status)
}

func (adguard) Fetch(ctx context.Context, conn *Conn) (*Payload, error) {
	var stats struct {
		Queries    int64   `json:"num_dns_queries"`
		Blocked    int64   `json:"num_blocked_filtering"`
		AvgSeconds float64 `json:"avg_processing_time"`
	}
	if err := getJSON(ctx, conn, "/control/stats", adguardAuth(conn), &stats); err != nil {
		return nil, err
	}
	return &Payload{KPIs: map[string]any{
		"queries":         stats.Queries,
		"blocked":         stats.Blocked,
		"blocked_percent": percent(float64(stats.Blocked), float64(stats.Queries)),
		"latency":         math.Round(stats.AvgSeconds*1000*10) / 10,
	}}, nil
}

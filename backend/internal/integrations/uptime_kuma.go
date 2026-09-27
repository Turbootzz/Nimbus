package integrations

import (
	"bufio"
	"context"
	"net/http"
	"strings"

	"github.com/nimbus/backend/internal/models"
)

// Uptime Kuma has no REST API; its Prometheus /metrics endpoint is the
// stable way in. It takes an API key as the basic auth password.
type uptimeKuma struct{}

func init() { Register(uptimeKuma{}) }

func (uptimeKuma) Kind() string { return "uptime_kuma" }

func (uptimeKuma) Meta() Meta {
	return Meta{
		Name:        "Uptime Kuma",
		Icon:        "uptime-kuma",
		DefaultPort: 3001,
		// None when /metrics is open (auth disabled in Uptime Kuma)
		AuthTypes: []string{models.IntegrationAuthAPIKey, models.IntegrationAuthNone},
		KPIs: []KPI{
			{Key: "up", Label: "Up"},
			{Key: "down", Label: "Down"},
			{Key: "up_percent", Label: "Up", Unit: "%"},
		},
	}
}

func uptimeKumaAuth(conn *Conn) authFunc {
	return func(r *http.Request) {
		if conn.Creds.APIKey != "" {
			r.SetBasicAuth("", conn.Creds.APIKey)
		}
	}
}

// uptimeKumaCounts reads monitor_status lines: 1 is up, 0 down, 2 pending
// and 3 maintenance (not counted)
func uptimeKumaCounts(ctx context.Context, conn *Conn) (up, down int, err error) {
	err = doRequest(ctx, conn, http.MethodGet, "/metrics", nil, uptimeKumaAuth(conn), func(resp *http.Response) error {
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(line, "monitor_status{") {
				continue
			}
			switch line[strings.LastIndexByte(line, ' ')+1:] {
			case "1":
				up++
			case "0":
				down++
			}
		}
		return scanner.Err()
	})
	return up, down, err
}

func (uptimeKuma) Test(ctx context.Context, conn *Conn) error {
	_, _, err := uptimeKumaCounts(ctx, conn)
	return err
}

func (uptimeKuma) Fetch(ctx context.Context, conn *Conn) (*Payload, error) {
	up, down, err := uptimeKumaCounts(ctx, conn)
	if err != nil {
		return nil, err
	}
	return &Payload{KPIs: map[string]any{
		"up":         up,
		"down":       down,
		"up_percent": percent(float64(up), float64(up+down)),
	}}, nil
}

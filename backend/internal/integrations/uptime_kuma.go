package integrations

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strings"

	"github.com/nimbus/backend/internal/models"
)

// Uptime Kuma has no REST API. There are two stable ways in: its Prometheus
// /metrics endpoint, which takes an API key as the basic auth password, and
// the public endpoints behind a status page (option status_page_slug).
type uptimeKuma struct{}

const uptimeKumaSlug = "status_page_slug"

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
			// The share of counted monitors that is up right now
			{Key: "up_percent", Label: "Up share", Unit: "%", Without: uptimeKumaSlug},
			// Real uptime, which only a status page reports, takes its place
			{Key: "uptime", Label: "Uptime 24h", Unit: "%", Option: uptimeKumaSlug},
		},
		Options: []Option{{
			Key:         uptimeKumaSlug,
			Label:       "Status page slug",
			Help:        "Reads this public status page instead of /metrics. It needs no API key: choose None under Sign in with.",
			Placeholder: "homelab",
			MaxLength:   100,
			Pattern:     `[A-Za-z0-9]+(-[A-Za-z0-9]+)*`,
		}},
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
// and 3 maintenance (not counted). A response without the monitor_status
// metric is not Uptime Kuma, even with a 200.
func uptimeKumaCounts(ctx context.Context, conn *Conn) (up, down int, err error) {
	err = doRequest(ctx, conn, http.MethodGet, "/metrics", nil, uptimeKumaAuth(conn), func(resp *http.Response) error {
		found := false
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			// The TYPE line is there even when no monitors exist yet
			if strings.HasPrefix(line, "# TYPE monitor_status ") {
				found = true
			}
			if !strings.HasPrefix(line, "monitor_status{") {
				continue
			}
			found = true
			switch line[strings.LastIndexByte(line, ' ')+1:] {
			case "1":
				up++
			case "0":
				down++
			}
		}
		if err := scanner.Err(); err != nil {
			return err
		}
		if !found {
			return errors.New("no monitor_status metric")
		}
		return nil
	})
	return up, down, err
}

// uptimeKumaStatusPage reads the monitors of a public status page: their
// latest status, counted like uptimeKumaCounts, and their average uptime
// over the last 24 hours as a percentage. The API key is not sent.
func uptimeKumaStatusPage(ctx context.Context, conn *Conn, slug string) (up, down int, uptime float64, err error) {
	// Uptime Kuma lowercases slugs
	slug = url.PathEscape(strings.ToLower(slug))

	// An unknown slug is a 404 here, but an empty 200 on the heartbeat
	// endpoint, so this request is what proves the page exists
	var page struct {
		Config *struct{} `json:"config"`
	}
	path := "/api/status-page/" + slug
	if err = getJSON(ctx, conn, path, nil, &page); err != nil {
		return 0, 0, 0, err
	}
	if page.Config == nil {
		return 0, 0, 0, fmt.Errorf("%s returned an unexpected response", path)
	}

	var beats struct {
		// Per monitor id, oldest first
		HeartbeatList map[string][]struct {
			Status int `json:"status"`
		} `json:"heartbeatList"`
		// "<monitor id>_24" to a ratio from 0 to 1
		UptimeList map[string]float64 `json:"uptimeList"`
	}
	if err = getJSON(ctx, conn, "/api/status-page/heartbeat/"+slug, nil, &beats); err != nil {
		return 0, 0, 0, err
	}
	for _, list := range beats.HeartbeatList {
		if len(list) == 0 {
			continue
		}
		switch list[len(list)-1].Status {
		case 1:
			up++
		case 0:
			down++
		}
	}
	if len(beats.UptimeList) > 0 {
		var sum float64
		for _, ratio := range beats.UptimeList {
			sum += ratio
		}
		// Two decimals: 99.95 and 100 are not the same uptime
		uptime = math.Round(sum/float64(len(beats.UptimeList))*10000) / 100
	}
	return up, down, uptime, nil
}

func (k uptimeKuma) Test(ctx context.Context, conn *Conn) error {
	_, err := k.Fetch(ctx, conn)
	return err
}

func (uptimeKuma) Fetch(ctx context.Context, conn *Conn) (*Payload, error) {
	slug := conn.Option(uptimeKumaSlug)
	if slug == "" {
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

	up, down, uptime, err := uptimeKumaStatusPage(ctx, conn, slug)
	if err != nil {
		return nil, err
	}
	return &Payload{KPIs: map[string]any{"up": up, "down": down, "uptime": uptime}}, nil
}

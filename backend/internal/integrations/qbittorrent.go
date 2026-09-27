package integrations

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/nimbus/backend/internal/models"
)

type qbittorrent struct{}

func init() { Register(qbittorrent{}) }

func (qbittorrent) Kind() string { return "qbittorrent" }

func (qbittorrent) Meta() Meta {
	return Meta{
		Name:        "qBittorrent",
		Icon:        "qbittorrent",
		DefaultPort: 8080,
		// None when the Web UI skips login for this address
		AuthTypes: []string{models.IntegrationAuthBasic, models.IntegrationAuthNone},
		KPIs: []KPI{
			{Key: "leech", Label: "Leech"},
			{Key: "download", Label: "Down"},
			{Key: "seed", Label: "Seed"},
			{Key: "upload", Label: "Up"},
		},
	}
}

// qbittorrentLogin opens a session and returns its cookie as name=value
// ("" without a username). Newer versions name it QBT_SID_<port>.
func qbittorrentLogin(ctx context.Context, conn *Conn) (string, error) {
	if conn.Creds.Username == "" {
		return "", nil
	}
	form := url.Values{"username": {conn.Creds.Username}, "password": {conn.Creds.Password}}
	var session string
	err := doRequest(ctx, conn, http.MethodPost, "/api/v2/auth/login", strings.NewReader(form.Encode()),
		func(r *http.Request) { r.Header.Set("Content-Type", "application/x-www-form-urlencoded") },
		func(resp *http.Response) error {
			for _, cookie := range resp.Cookies() {
				if cookie.Name == "SID" || strings.HasPrefix(cookie.Name, "QBT_SID") {
					session = cookie.Name + "=" + cookie.Value
				}
			}
			return nil
		})
	if err != nil {
		return "", err
	}
	// A wrong password answers 200 "Fails." without a cookie
	if session == "" {
		return "", errRejected
	}
	return session, nil
}

func qbittorrentAuth(session string) authFunc {
	return func(r *http.Request) {
		if name, value, ok := strings.Cut(session, "="); ok {
			r.AddCookie(&http.Cookie{Name: name, Value: value})
		}
	}
}

func (qbittorrent) Test(ctx context.Context, conn *Conn) error {
	sid, err := qbittorrentLogin(ctx, conn)
	if err != nil {
		return err
	}
	// The version is plain text, not JSON
	err = doRequest(ctx, conn, http.MethodGet, "/api/v2/app/version", nil, qbittorrentAuth(sid),
		func(resp *http.Response) error {
			_, err := io.Copy(io.Discard, resp.Body)
			return err
		})
	if sid != "" {
		_ = doRequest(ctx, conn, http.MethodPost, "/api/v2/auth/logout", nil, qbittorrentAuth(sid), nil)
	}
	return err
}

func (qbittorrent) Fetch(ctx context.Context, conn *Conn) (*Payload, error) {
	var transfer struct {
		Download int64 `json:"dl_info_speed"`
		Upload   int64 `json:"up_info_speed"`
	}
	var leech, seed int
	err := withSession(conn, "sid",
		func() (string, error) { return qbittorrentLogin(ctx, conn) },
		func(sid string) error {
			if err := getJSON(ctx, conn, "/api/v2/transfer/info", qbittorrentAuth(sid), &transfer); err != nil {
				return err
			}
			leech, seed = 0, 0
			return eachJSON(ctx, conn, "/api/v2/torrents/info", qbittorrentAuth(sid), func(t struct {
				Progress float64 `json:"progress"`
			}) {
				if t.Progress < 1 {
					leech++
				} else {
					seed++
				}
			})
		})
	if err != nil {
		return nil, err
	}
	return &Payload{KPIs: map[string]any{
		"leech":    leech,
		"download": formatRate(transfer.Download),
		"seed":     seed,
		"upload":   formatRate(transfer.Upload),
	}}, nil
}

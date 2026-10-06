# Integrations

An integration connects Nimbus to one of your apps. Link it to a service and the service tile shows a few live numbers from that app, like Sonarr's download queue. Nimbus fetches them on the server, so your browser never talks to the app directly and the credentials stay on the server, encrypted with `ENCRYPTION_KEY`.

Add integrations under **Settings, Integrations**, then pick one under **Live numbers** when you edit a service.

## Supported apps

| App | Numbers on the tile | Sign in with | Notes |
|---|---|---|---|
| AdGuard Home | queries, blocked, blocked %, latency | username and password | |
| Docker | running, stopped, total containers | none | Admins only. The URL is `unix:///var/run/docker.sock` (needs `DOCKER_SOCKET`, see [CONFIGURATION.md](CONFIGURATION.md#docker-socket)) or a socket proxy like `http://docker-socket-proxy:2375`. Also feeds the Docker containers widget. |
| Home Assistant | people home, lights on, switches on | long-lived access token | Create the token on your Home Assistant profile page. |
| Jellyfin | movies, series, episodes, streams | API key | Dashboard, API Keys. |
| Pi-hole | queries, blocked, blocked %, blocklist size | app password, or none | Pi-hole v6 only. v5 used a different API. |
| Proxmox VE | VMs running/total, LXC running/total, CPU %, memory % | API token | Enter the token as `user@pve!tokenid=secret`. The `PVEAuditor` role is enough. Turn off TLS verification for a self-signed certificate. |
| qBittorrent | downloading, download speed, seeding, upload speed | username and password, or none | |
| Radarr | wanted, queued, movies | API key | Settings, General. Also feeds the calendar widget (releases). |
| Sonarr | wanted, queued, series | API key | Settings, General. Also feeds the calendar widget (episodes). |
| Uptime Kuma | monitors up, down, up % | API key, or none | Reads `/metrics`. Create the key under Settings, API Keys. Pending and maintenance monitors are not counted. |

The URL is the address the Nimbus server uses to reach the app, including a sub path if the app runs behind a reverse proxy (for example `https://proxy.lan/sonarr`).

## Adding an integration

Each app is one Go file in `backend/internal/integrations/` that registers itself. Nothing else needs to change: the settings page, the service form and the tile all read the app's metadata.

1. Copy `backend/internal/integrations/_template.go.txt` to `<app>.go` and `_template_test.go.txt` to `<app>_test.go`. The template is a working kind with a test. Then make the type implement `Integration` for your app:
   - `Kind()`: a stable id, stored in the database (`"pihole"`). Never change it later.
   - `Meta()`: the name, the [dashboard-icons](https://github.com/walkxcode/dashboard-icons) slug, the default port, the auth types it accepts (the first is the default) and its KPIs. The KPIs are the numbers on the tile, in display order, at most four.
   - `Test(ctx, conn)`: a cheap call that proves the URL and the credentials work.
   - `Fetch(ctx, conn)`: returns a `Payload` whose `KPIs` map has one value per KPI key. Values can be numbers or short text like `"3/5"`.
2. Register it in `init()`: `func init() { Register(myApp{}) }`.
3. Add `<app>_test.go` with an `httptest` server that answers like the real app: a working fetch, rejected credentials, and anything special (sessions, odd responses).
4. Add a row to the table above.

### Rules

- Use `conn.Client` for every request. It blocks cloud metadata addresses and honours the TLS setting.
- Use the helpers in `http.go`: `getJSON`, `eachJSON` for big lists (they are read item by item), and `withSession` for apps that log in first. Store session ids in `conn.State`, which lives as long as the integration.
- Put credentials in headers or the request body, never in the URL. Errors are shown to the user, and Go puts the URL in its errors.
- Don't log credentials or responses.
- Keep `Fetch` to a few requests. It runs on every refresh, by default every minute.

## Testing locally

```bash
cd backend
go test ./internal/integrations/
```

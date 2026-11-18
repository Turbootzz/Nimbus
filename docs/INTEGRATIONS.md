# Integrations

An integration connects Nimbus to one of your apps. Link it to a service and the service tile shows a few live numbers from that app, like Sonarr's download queue. Nimbus fetches them on the server, so your browser never talks to the app directly and the credentials stay on the server, encrypted with `ENCRYPTION_KEY`.

Add integrations under **Settings, Integrations**, then pick one under **Live numbers** when you edit a service.

## Supported apps

| App | Numbers on the tile | Sign in with | Notes |
|---|---|---|---|
| AdGuard Home | queries, blocked, blocked %, latency | username and password | |
| Docker | running, stopped, total containers | none, or API key (Portainer) | Admins only. The URL is `unix:///var/run/docker.sock` (needs `DOCKER_SOCKET`, see [CONFIGURATION.md](CONFIGURATION.md#docker-socket)), a socket proxy like `http://docker-socket-proxy:2375`, or [Portainer](#docker-through-portainer). Also feeds the Docker containers widget. |
| Home Assistant | people home, lights on, switches on, up to two entities of your choice | long-lived access token | Create the token on your Home Assistant profile page. See [extra entities](#home-assistant-extra-entities). |
| Jellyfin | movies, series, episodes, streams | API key | Dashboard, API Keys. |
| Pi-hole | queries, blocked, blocked %, blocklist size | app password, or none | Pi-hole v6 only. v5 used a different API. |
| Proxmox VE | VMs running/total, LXC running/total, CPU %, memory % | API token | Enter the token as `user@pve!tokenid=secret`. The `PVEAuditor` role is enough. Turn off TLS verification for a self-signed certificate. Fill in **Node** to count one node instead of the whole cluster. |
| qBittorrent | downloading, download speed, seeding, upload speed | username and password, or none | |
| Radarr | wanted, queued, movies | API key | Settings, General. Also feeds the calendar widget (releases). |
| Sonarr | wanted, queued, series | API key | Settings, General. Also feeds the calendar widget (episodes). |
| Uptime Kuma | monitors up, down, and the share that is up or the 24 hour uptime | API key, or none | Reads `/metrics` or a public status page, see [Uptime Kuma](#uptime-kuma). Pending and maintenance monitors are not counted. |

The URL is the address the Nimbus server uses to reach the app, including a sub path if the app runs behind a reverse proxy (for example `https://proxy.lan/sonarr`).

Some apps have settings of their own, such as the Proxmox node. They show in the integration form under the credentials.

**Verify TLS certificate** is on by default. Turning it off makes a self-signed certificate work, but Nimbus then can't tell the app from an impostor on the network, who would get the credentials. Only turn it off on a network you trust. If the app redirects Nimbus to another host or port, the credentials are not sent along (an upgrade from http to https on the same host keeps them).

### Proxmox VE

By default the tile counts the whole cluster. Enter a node name (as shown in the Proxmox tree, for example `pve`) under **Node** to count only the VMs, containers, CPU and memory of that node. A name the cluster doesn't have fails the connection test.

### Uptime Kuma

Uptime Kuma has no official REST API. Nimbus can read it in two ways:

| | Metrics (default) | Status page |
|---|---|---|
| Set up | Create a key under Settings, API Keys and choose **Sign in with: API key**. Choose **None** if authentication is disabled in Uptime Kuma. | Enter the slug of a published status page under **Status page slug** (the last part of `https://kuma.lan/status/<slug>`) and choose **Sign in with: None**. |
| Reads | `GET /metrics` | `GET /api/status-page/<slug>` and `GET /api/status-page/heartbeat/<slug>`, both public |
| Monitors | all of them | the ones on that status page |
| Tile | Up, Down, Up share (% of the counted monitors that is up now) | Up, Down, Uptime 24h (average uptime of those monitors over the last 24 hours) |

Status page mode needs no key and Nimbus sends none. Uptime Kuma caches these endpoints, so a change can take a minute (up to five for the monitor list) to show.

### Home Assistant extra entities

Enter an entity ID such as `sensor.living_room_temperature` under **Extra entity 1** or **2** to show its state on the tile, with its friendly name as the label and its unit. A state is formatted as a number when the entity has a unit. A tile has room for four numbers: one extra entity is added as the fourth, a second one takes the place of **Switches on**. All of them stay available for the status strip.

### Docker through Portainer

If you already run Portainer, Nimbus can read Docker through it: no socket proxy and no socket mount on the Nimbus host.

1. In Portainer, open **My account, Access tokens** and add a token. Copy it; Portainer shows it once.
2. Open the environment and read its id from the address bar: `#!/<id>/docker/...`.
3. In Nimbus, add a Docker integration with the URL `https://<portainer>:9443/api/endpoints/<id>/docker`, choose **Sign in with: API key** and paste the token.

Turn off **Verify TLS certificate** if Portainer still has its self-signed certificate. Nimbus sends the token in the `X-API-Key` header, never in the URL. Each Portainer environment is its own integration.

The token can do everything its Portainer user can. Nimbus only sends `GET` requests (the Docker integration's connection refuses anything else), but a token of a separate, non-admin user with access to only that environment limits it further; Nimbus then shows the containers that user may see. Prefer Portainer's `https://` port: over `http://` the token travels unencrypted.

## Adding an integration

Each app is one Go file in `backend/internal/integrations/` that registers itself. Nothing else needs to change: the settings page, the service form and the tile all read the app's metadata.

1. Copy `backend/internal/integrations/_template.go.txt` to `<app>.go` and `_template_test.go.txt` to `<app>_test.go`. The template is a working kind with a test. Then make the type implement `Integration` for your app:
   - `Kind()`: a stable id, stored in the database (`"pihole"`). Never change it later.
   - `Meta()`: the name, the [dashboard-icons](https://github.com/walkxcode/dashboard-icons) slug, the default port, the auth types it accepts (the first is the default) and its KPIs. The KPIs are the numbers on the tile, in display order; the tile shows the first four. Optional: `URLHint` (the example address in the form), `Hint` (a line of help under the URL field) and `Options` (see [Options](#options)).
   - `Test(ctx, conn)`: a cheap call that proves the URL and the credentials work.
   - `Fetch(ctx, conn)`: returns a `Payload` whose `KPIs` map has one value per KPI key. Values can be numbers or short text like `"3/5"`.
2. Register it in `init()`: `func init() { Register(myApp{}) }`.
3. Add `<app>_test.go` with an `httptest` server that answers like the real app: a working fetch, rejected credentials, an unreachable host (`assertUnreachable`), and anything special (options, sessions, odd responses).
4. Add a row to the table above.

### Options

Options are a kind's own settings, like a node name or a status page slug. Declare them in `Meta().Options` and the rest follows: the form shows a text field for each, the server checks what is submitted and stores it with the integration.

```go
Options: []Option{{
    Key:         "node",              // the key in the stored JSON
    Label:       "Node",              // the field's label
    Help:        "Only count this node. Leave empty for the whole cluster.",
    Placeholder: "pve",               // also the example in the error for a bad value
    Type:        "",                  // or OptionNumber for a number field
    Required:    false,
    MaxLength:   63,                  // characters; 200 when left out
    Pattern:     `[A-Za-z0-9\-]+`,    // the whole value must match
}},
```

- Read one with `conn.Option("node")`. It returns `""` when the option is not set. Values are trimmed text, also for a number: parse it with `strconv`.
- The server rejects keys the kind doesn't declare, values that are too long or don't match `Pattern`, and a missing `Required` option. The browser checks `Pattern` too, so keep it to what Go and JavaScript share and write a literal `-` inside a class as `\-`.
- Escape an option wherever it goes: `url.PathEscape` in a path, `url.QueryEscape` in a query.
- Options are stored in plain text and sent back to the browser. Secrets belong in the credentials.
- A KPI that only exists when an option is set names it: `{Key: "uptime", Label: "Uptime 24h", Option: "status_page_slug"}`. One that gives way when an option is set names it in `Without`. Such a KPI is left out of the tile and the status strip for integrations it does not apply to, and `Fetch` leaves its value out.
- A tile shows four numbers. When more apply, the KPIs with an `Option` stay and the others give way, from the end.
- When a label or unit depends on live data, `Fetch` returns it in `Payload.KPIDefs`: a KPI with the same `Key` as one in `Meta().KPIs`, whose label and unit then show on that integration's tile and in the status strip. Home Assistant does this to label an entity by its friendly name. Keep such text short (see `clip`): it comes from another system.

### Rules

- Use `conn.Client` for every request. It blocks cloud metadata addresses, honours the TLS setting and keeps credentials away from redirects to another host or port.
- A kind that must never change anything on the app sets `ReadOnly: true` in `Meta()`: its client then refuses everything but `GET` and `HEAD`.
- Use the helpers in `http.go`: `getJSON`, `eachJSON` for big lists (they are read item by item), and `withSession` for apps that log in first. Store session ids in `conn.State`, which lives as long as the integration.
- An app with few session seats (like Pi-hole) should also implement `Closer`: `Close` logs the kept session out when the integration is deleted.
- Put credentials in headers or the request body, never in the URL. Errors are shown to the user, and Go puts the URL in its errors.
- Don't log credentials or responses.
- Keep `Fetch` to a few requests. It runs on every refresh, by default every minute.

## Testing locally

```bash
cd backend
go test ./internal/integrations/
```

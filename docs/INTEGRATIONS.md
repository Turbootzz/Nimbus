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

**Verify TLS certificate** is on by default. Turning it off makes a self-signed certificate work, but Nimbus then can't tell the app from an impostor on the network, who would get the credentials. Only turn it off on a network you trust. If the app redirects Nimbus to another host or port, the credentials are not sent along (only an upgrade from `http://` on port 80 to `https://` on port 443 of the same host keeps them). An app that redirects from one non-standard port to another, for example `http://nas.lan:5000` to `https://nas.lan:5001`, therefore fails to connect: enter its final `https://` URL.

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

## Docker discovery

Nimbus can offer the containers behind your Docker integrations as services, so you don't type them in one by one. It needs a [Docker integration](#supported-apps) and is for admins only, like that integration: the container list shows what runs on the host.

Open **Services**. The **Discovered** tray lists every container that is not a service yet:

- **Add** creates the service with its name, address, icon, description and group. The group is found by name, ignoring case, and created when you have none with that name.
- **Review** is shown instead of Add when the address is a guess (or missing). It opens the name, address and group so you can correct them first. **Edit** does the same for a labelled container.
- **Ignore** hides a container. It stays hidden across restarts and when the container is re-created; **Ignored** at the bottom of the tray lists them with **Un-ignore**.
- **Refresh** asks Docker again.

A container you added is not offered again, also not after you rename the service or change its address. When you delete that service, the container moves to **Ignored**; un-ignore it to add it again. Containers are remembered by integration and container name, so a renamed container is offered as a new one, and deleting and re-adding the Docker integration offers everything again.

With several Docker integrations, the tray shows the containers of all of them, each with the name of its integration. If one can't be reached, its error is shown and the others still load.

### Labels

Labels are optional. Without them a container is listed under its container name, with a guessed address and icon.

| Label | What it sets |
|-------|--------------|
| `nimbus.enable` | `true` marks the container as labelled: it is listed first and may be [auto-added](#auto-add). `false` hides it from discovery. |
| `nimbus.name` | Service name. Default: the container name. |
| `nimbus.url` | Address of the service. Must start with `http://` or `https://`. |
| `nimbus.icon` | A [dashboard-icons](https://github.com/homarr-labs/dashboard-icons) name such as `sonarr`, or an `http(s)://` image URL. |
| `nimbus.group` | Group to put the service in (35 characters at most). |
| `nimbus.description` | Description on the tile. |
| `nimbus.widget` | An integration kind: `adguard`, `home_assistant`, `jellyfin`, `pihole`, `proxmox`, `qbittorrent`, `radarr`, `sonarr` or `uptime_kuma`. After you add the container, the tray links to the integration form with the kind and address filled in. Check that address before you enter credentials: it comes from the label. You still enter the credentials there, and link the integration to the service under **Live numbers**. Nothing is stored before you save that form. |

Already using [Homepage](https://gethomepage.dev)? `homepage.name`, `homepage.href`, `homepage.icon`, `homepage.group` and `homepage.description` are read when the `nimbus.*` label for that field is missing. Homepage has no `enable` label, so such containers are filled in but not auto-added. Auto-add also takes no `homepage.href`: add `nimbus.enable=true` and `nimbus.url` for that. Homepage's other icon sets (`mdi-`, `si-`, `sh-`) are not used; the icon is guessed from the image instead.

```yaml
services:
  sonarr:
    image: lscr.io/linuxserver/sonarr:latest
    container_name: sonarr
    labels:
      nimbus.enable: "true"
      nimbus.name: Sonarr
      nimbus.url: https://sonarr.example.lan
      nimbus.icon: sonarr
      nimbus.group: Media
      nimbus.description: TV series
      nimbus.widget: sonarr
```

### Guessed address and icon

Without `nimbus.url` (or `homepage.href`), the address is built from the container's lowest published TCP port and marked **guess**:

- Docker integration over `http(s)://` (socket proxy or Portainer): `http://<host of the integration URL>:<port>`.
- Docker integration over the unix socket: `http://<host you opened Nimbus on>:<port>`, because the containers run next to Nimbus.

Port 80 gives `http://<host>` and 443 `https://<host>`. A container without a published port has no address; fill one in under **Review**. The guess is wrong when you reach the service through a reverse proxy, or when the socket proxy is addressed by its Compose name: set `nimbus.url` then.

The icon is the dashboard-icons image named after the image, without registry, organisation, tag and digest: `lscr.io/linuxserver/sonarr:latest` gives `sonarr`. If no such icon exists, the tile shows the default emoji. Icons load from the address in `DASHBOARD_ICONS_BASE_URL`; setting that to empty turns them off.

### Auto-add

**Settings, Admin, Docker Discovery, Auto-add labelled containers** is off by default. When on, Nimbus checks every minute and adds containers that have both `nimbus.enable=true` and a valid `nimbus.url`, to the dashboard of the admin who owns the Docker integration. It never adds:

- unlabelled containers, or containers with only `homepage.*` labels;
- an address from `homepage.href` or a guessed one;
- a container you ignored;
- a container it added before, also after you deleted that service.

### Security

- Discovery sends one request per Docker integration: the container list (`GET /containers/json`). It never opens the addresses or icons it finds.
- Labels often hold secrets (reverse-proxy passwords, tokens). Nimbus reads only the labels in the table above and returns nothing else, not in this API and not in the Docker containers widget.
- Whoever can start containers can set labels. Addresses and icon URLs other than `http://` and `https://` are dropped, and label text is shown as plain text. Still, with auto-add on, anyone who can start a container on that host can put a link on your dashboard. Like every service, an added container is health-checked: the Nimbus server sends a `GET` to its address every interval (it does not follow redirects and keeps only the status and response time). Addresses in your own network are allowed, as for any service; cloud metadata addresses (such as `169.254.169.254`) are refused, both as a service address and by the health checker when a name resolves to one. One run adds at most 20 containers. Leave auto-add off if people you don't trust with that can start containers: that is usually only whoever controls the Docker host, but a socket proxy or Portainer can give label rights to more people.
- Prefer a [socket proxy](https://github.com/Tecnativa/docker-socket-proxy) with only `CONTAINERS=1` over mounting the socket. If you mount it, mount it `:ro` and read [Docker socket](CONFIGURATION.md#docker-socket): `:ro` alone does not make the Docker API read-only.

```yaml
services:
  docker-socket-proxy:
    image: tecnativa/docker-socket-proxy
    environment:
      CONTAINERS: 1 # read-only container list, nothing else
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
    restart: unless-stopped
```

Then add a Docker integration with the URL `http://docker-socket-proxy:2375` (Nimbus must be on the same Docker network) and leave `DOCKER_SOCKET` unset.

### API

All routes need an admin's session; API tokens are refused.

| Route | Does |
|-------|------|
| `GET /api/v1/discovery/docker` | Lists the containers that are not a service yet, with `ignored` set on the ignored ones, and an `errors` entry per integration that failed. Docker's answer is reused for 30 seconds; `?refresh=true` asks again. |
| `POST /api/v1/discovery/docker/services` | Adds one: `{"key", "name", "url", "icon_url", "group", "description"}`. `409` when it was added or ignored already. |
| `POST /api/v1/discovery/docker/ignored` | Ignores `{"key"}`. |
| `DELETE /api/v1/discovery/docker/ignored?key=` | Un-ignores. |

## Adding an integration

Each app is one Go file in `backend/internal/integrations/` that registers itself. Nothing else needs to change: the settings page, the service form and the tile all read the app's metadata.

1. Copy `backend/internal/integrations/_template.go.txt` to `<app>.go` and `_template_test.go.txt` to `<app>_test.go`. The template is a working kind with a test. Then make the type implement `Integration` for your app:
   - `Kind()`: a stable id, stored in the database (`"pihole"`). Never change it later.
   - `Meta()`: the name, the [dashboard-icons](https://github.com/walkxcode/dashboard-icons) slug, the default port, the auth types it accepts (the first is the default) and its KPIs. The KPIs are the numbers on the tile, in display order; the tile shows four of them (see [Options](#options) for which, when more apply). Optional: `URLHint` (the example address in the form), `Hint` (a line of help under the URL field) and `Options` (see [Options](#options)).
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

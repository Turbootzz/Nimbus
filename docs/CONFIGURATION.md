# Advanced Configuration Guide

This guide covers all configuration options for Nimbus, including OAuth setup, Prometheus integration, and production deployment.

---

## Table of Contents

- [Environment Variables](#environment-variables)
  - [Required Variables](#required-variables)
  - [Database Configuration](#database-configuration)
  - [Server Configuration](#server-configuration)
  - [Authentication](#authentication)
  - [Encryption Key](#encryption-key)
  - [Health Checks](#health-checks)
  - [Metrics & Monitoring](#metrics--monitoring)
- [OAuth Setup](#oauth-setup)
  - [Google OAuth](#google-oauth)
  - [GitHub OAuth](#github-oauth)
  - [Discord OAuth](#discord-oauth)
- [API Tokens](#api-tokens)
- [Prometheus Integration](#prometheus-integration)
- [Dashboard Export and Import](#dashboard-export-and-import)
  - [Import from another dashboard](#import-from-another-dashboard)
- [Public Status Page](#public-status-page)
- [Config File](#config-file)
- [Production Deployment](#production-deployment)

---

## Environment Variables

Nimbus uses **convention over configuration**. Most variables have sensible defaults — you only need to set secrets.

### Required Variables

**None!** Nimbus works out of the box with sensible defaults.

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_PASSWORD` | `nimbus-default-password` | PostgreSQL password |
| `JWT_SECRET` | *auto-generated* | Authentication secret (persisted in uploads volume) |
| `ENCRYPTION_KEY` | *auto-generated* | Encrypts integration credentials and notification tokens (persisted in uploads volume) |

> **Security note:** The default database password is safe because PostgreSQL is only accessible within the Docker network (not exposed externally). For shared hosting environments, set a custom password.

**For production**, set custom secrets:
```bash
# Create .env file with secure passwords
cat > .env << EOF
DB_PASSWORD=$(openssl rand -base64 24 | tr -dc 'a-zA-Z0-9' | head -c 24)
JWT_SECRET=$(openssl rand -base64 32 | tr -dc 'a-zA-Z0-9' | head -c 32)
ENCRYPTION_KEY=$(openssl rand -base64 32)
EOF
```

> **⚠️ Password Character Warning:** Avoid special characters (`%`, `@`, `&`, `*`, `#`, `^`) in `DB_PASSWORD`. These can cause connection issues with PostgreSQL. Use alphanumeric characters and `-` or `_` only.

### Database Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_HOST` | `db` | PostgreSQL hostname (use `db` for Docker, `localhost` for local dev) |
| `DB_PORT` | `5432` | PostgreSQL port |
| `DB_NAME` | `nimbus` | Database name |
| `DB_USER` | `nimbus` | Database username |
| `DB_PASSWORD` | *required* | Database password |
| `DB_URL` | - | Alternative: full connection string (overrides individual vars) |

**Example `DB_URL`:**
```
postgres://nimbus:password@localhost:5432/nimbus?sslmode=disable
```

### Server Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | Backend API port |
| `CORS_ORIGINS` | *none* | Comma-separated allowed origins (optional in unified mode) |
| `COOKIE_SECURE` | `false` | Set to `true` for HTTPS |
| `COOKIE_DOMAIN` | - | Cookie domain restriction |

> **Note:** When using the unified `turboot/nimbus` Docker image, `CORS_ORIGINS` is not needed — the built-in nginx proxy handles same-origin requests.

### Frontend API URL

| Variable | Default | Description |
|----------|---------|-------------|
| `NEXT_PUBLIC_API_URL` | *auto-detect* | Backend API URL for browser requests |

**Values:**
- `same-origin` — Use relative `/api/v1` path (unified Docker image)
- `http://server:8080` — Full URL (separate containers)
- *empty/unset* — Auto-detect based on browser location

> **Note:** The unified image sets this automatically. Only configure manually for separate container deployments.

| Variable | Default | Description |
|----------|---------|-------------|
| `INTERNAL_API_URL` | *see below* | Backend API URL for requests the frontend server makes itself (the public status page), including `/api/v1` |

Without it the frontend server uses `NEXT_PUBLIC_API_URL` when that is a full URL, and otherwise `http://127.0.0.1:8080/api/v1`, which is right for the unified image. With separate containers, set it to an address the frontend container can reach, such as `http://backend:8080/api/v1`.

**Production CORS example (separate containers only):**
```bash
CORS_ORIGINS=https://nimbus.example.com,https://www.nimbus.example.com
```

### Authentication

| Variable | Default | Description |
|----------|---------|-------------|
| `JWT_SECRET` | *required* | Secret for signing JWT tokens (min 32 chars) |
| `JWT_EXPIRY` | `24h` | Token expiration (e.g., `24h`, `7d`, `30d`) |
| `BCRYPT_COST` | `10` | Password hashing cost (10-12 recommended) |
| `API_TOKEN_RATE_LIMIT` | `600` | Requests per minute one [API token](#api-tokens) may make |

### Encryption Key

| Variable | Default | Description |
|----------|---------|-------------|
| `ENCRYPTION_KEY` | *auto-generated* (Docker and `make dev-backend`) | Base64 encoded 32-byte key that encrypts integration credentials and [notification](NOTIFICATIONS.md) tokens (AES-256-GCM) |

Generate one with `openssl rand -base64 32`. The unified image creates it on first start and saves it in `uploads/.secrets/generated.env`, next to `JWT_SECRET`. A key you set yourself always wins over the generated one. For local development, `make dev-backend` adds one to your `.env`.

**Keep this key safe.** API keys and passwords of your integrations are stored encrypted with it. If the key is lost or changed, Nimbus can't read them anymore and you have to enter them again. Back up the uploads volume, or set your own key and store it somewhere safe. Nimbus refuses to start without a valid key.

### Docker socket

| Variable | Default | Description |
|----------|---------|-------------|
| `DOCKER_SOCKET` | *(none)* | Path of the Docker socket the Docker integration may use, e.g. `/var/run/docker.sock`. Without it, only a socket proxy over `http://` or Portainer works. |

To use the socket, mount it and set the variable:

```yaml
    environment:
      DOCKER_SOCKET: /var/run/docker.sock
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
```

`:ro` does not make the Docker API read-only: anything that can reach the socket can start and stop containers. Nimbus only ever sends `GET` requests over it, and only admins can add a Docker integration. A [socket proxy](https://github.com/Tecnativa/docker-socket-proxy) with only `CONTAINERS=1` limits it further; then use its `http://` address as the integration URL and leave `DOCKER_SOCKET` unset.

The same integration feeds [Docker discovery](INTEGRATIONS.md#docker-discovery), which offers your containers as services and reads `nimbus.*` labels.

Portainer needs no socket and no `DOCKER_SOCKET` either: use `https://<portainer>:9443/api/endpoints/<id>/docker` as the integration URL and sign in with a Portainer access token as the API key. See [Docker through Portainer](INTEGRATIONS.md#docker-through-portainer).

### Health Checks

| Variable | Default | Description |
|----------|---------|-------------|
| `HEALTH_CHECK_INTERVAL` | `60` | Seconds between health checks |
| `HEALTH_CHECK_TIMEOUT` | `10` | Request timeout in seconds |

**Smart TLS Verification:**
Nimbus automatically handles self-signed certificates for local/private IPs:
- Public services → Full certificate verification
- Local services (192.168.x.x, 10.x.x.x, etc.) → Skips verification

No configuration needed — works with Portainer, Proxmox, and other homelab services.

### Metrics & Monitoring

| Variable | Default | Description |
|----------|---------|-------------|
| `METRICS_RETENTION_DAYS` | `30` | Days to retain status logs |
| `PROMETHEUS_API_KEY` | - | API key for Prometheus endpoint |

---

## OAuth Setup

OAuth providers are optional. Leave the variables empty to disable a provider.

### Google OAuth

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create a new project or select existing
3. Navigate to **APIs & Services** → **Credentials**
4. Click **Create Credentials** → **OAuth client ID**
5. Select **Web application**
6. Add authorized redirect URI: `http://localhost:3000/api/v1/auth/oauth/google/callback`

```bash
GOOGLE_CLIENT_ID=your-client-id.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=your-client-secret
GOOGLE_REDIRECT_URL=http://localhost:3000/api/v1/auth/oauth/google/callback
```

### GitHub OAuth

1. Go to [GitHub Developer Settings](https://github.com/settings/developers)
2. Click **New OAuth App**
3. Set Authorization callback URL: `http://localhost:3000/api/v1/auth/oauth/github/callback`

```bash
GITHUB_CLIENT_ID=your-client-id
GITHUB_CLIENT_SECRET=your-client-secret
GITHUB_REDIRECT_URL=http://localhost:3000/api/v1/auth/oauth/github/callback
```

### Discord OAuth

1. Go to [Discord Developer Portal](https://discord.com/developers/applications)
2. Create a new application
3. Go to **OAuth2** → **General**
4. Add redirect: `http://localhost:3000/api/v1/auth/oauth/discord/callback`

```bash
DISCORD_CLIENT_ID=your-client-id
DISCORD_CLIENT_SECRET=your-client-secret
DISCORD_REDIRECT_URL=http://localhost:3000/api/v1/auth/oauth/discord/callback
```

### OAuth State Secret (Optional)

For additional CSRF protection:

```bash
OAUTH_STATE_SECRET=your-oauth-state-secret
```

If not set, falls back to `JWT_SECRET`.

---

## API Tokens

Personal access tokens let scripts, other dashboards and AI assistants use the Nimbus API
without your password. Create them under **Settings → API Tokens**. A token is shown once,
starts with `nimbus_` and is sent as a bearer token:

```bash
curl -H "Authorization: Bearer nimbus_..." http://localhost:3000/api/v1/services
```

### Scopes

Each token only gets the scopes you select.

| Scope | Grants |
|-------|--------|
| `read:services` | Read services and groups |
| `read:metrics` | Read uptime metrics, status history and the Prometheus export |
| `read:widgets` | Read widgets (secrets masked) and the dashboard data |
| `write:services` | Create, change, delete, reorder and recheck services and groups. Includes `read:services` |
| `mcp` | Use the [MCP endpoint](MCP.md) at `/mcp` |

- `GET` requests need the read scope of what they read.
- A token can only change services and groups, and needs `write:services` for that. Widgets
  and the dashboard layout are read-only for tokens.
- A request outside the token's scopes gets `403`.
- Everything else never accepts a token, whatever its scopes: account settings, preferences,
  uploads, the icon lookup, webhooks, integrations, token management, the live dashboard
  stream and the admin pages. A leaked token cannot create tokens, read webhook URLs or
  integration credentials, or make Nimbus send requests to an address of its choosing other
  than the health check of a service.
- A token never has admin rights, also when its owner is an admin.
- `mcp` only opens `/mcp`. It gives no access to the REST API.

### Expiry and revocation

A token can expire after 7, 30, 90 or 365 days, or never. An expired or revoked token gets
`401`. Creating and revoking a token is recorded in the activity log (the
`user_activity_logs` table).

### Rate limit

Each token may make `API_TOKEN_RATE_LIMIT` requests per minute (default `600`), counted over
the REST API and `/mcp` together. Above that Nimbus answers `429` with a `Retry-After` header.
Browser sessions are not limited.

### Tokens created before scopes

Tokens made before scopes existed keep working: a read-only token has the three read scopes,
a full-access token also has `write:services`. They are bound by the rules above, so they
no longer reach webhooks, preferences or uploads, a full-access token no longer edits
widgets, and an admin's token no longer has admin rights. None of them has `mcp`; create a
new token for that.

---

## Prometheus Integration

Export service metrics in Prometheus format for monitoring and alerting.

### 1. Generate an API Key

```bash
openssl rand -hex 32
```

### 2. Add to Environment

```bash
PROMETHEUS_API_KEY=your-generated-key
```

### 3. Create `prometheus.yml`

```yaml
global:
  scrape_interval: 30s

scrape_configs:
  - job_name: 'nimbus'
    scrape_interval: 30s
    metrics_path: '/api/v1/prometheus/metrics/user/YOUR_USER_ID'
    authorization:
      type: Bearer
      credentials: 'YOUR_API_KEY'
    static_configs:
      - targets: ['host.docker.internal:3000']
```

### 4. Deploy Prometheus

```bash
docker run -d \
  --name prometheus \
  -p 9090:9090 \
  --add-host=host.docker.internal:host-gateway \
  -v $(pwd)/prometheus.yml:/etc/prometheus/prometheus.yml:ro \
  prom/prometheus:latest
```

### Using an API token instead

`PROMETHEUS_API_KEY` is one key for the whole instance and can read every user's metrics. An
[API token](#api-tokens) with the `read:metrics` scope is the safer choice: it only reads your
own metrics (also when you are an admin), can expire and can be revoked on its own.
Put the token in `credentials` and leave `PROMETHEUS_API_KEY` unset. Both keep working, so
you can switch whenever you like. To scrape several users with one credential, keep
`PROMETHEUS_API_KEY`.

### Available Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `nimbus_service_up` | Gauge | Service status (1=up, 0=down) |
| `nimbus_service_response_time_ms` | Gauge | Response time in milliseconds |
| `nimbus_service_last_check_timestamp` | Gauge | Unix timestamp of last check |

---

## Dashboard Export and Import

**Settings → Export & Import** downloads your dashboard as one JSON file and loads such a file
again: as a backup, to move to another Nimbus, or to share a layout. The same file is what
`GET /api/v1/export` returns and `POST /api/v1/import` takes.

### What is in the file

| In the file | Not in the file |
|---|---|
| Groups, services, widgets, integrations | Integration credentials (API keys, passwords, tokens) |
| Uploaded service icons and wallpaper, embedded as base64 | Secrets in widget configs: Custom API header values and calendar feed addresses are masked |
| Preferences: theme, accent color, layout, wallpaper settings, status strip | Health history, metrics, webhooks, API tokens, your account |

After an import, integrations that use credentials show **Needs credentials** under
Settings → Integrations and are not polled until you enter them. Widgets with masked secrets
show an error until you edit them and enter the value again.

An export embeds at most 6 MB of images, so the file stays under the 10 MB upload limit. Images
that are missing or do not fit are left out and counted in `omitted_images`; such a service
falls back to its emoji, and an import keeps the wallpaper you have.

### Merge or replace

| Mode | What it does |
|---|---|
| `merge` | Adds the file's groups, services, widgets and integrations to what you have. New tiles go behind your existing ones, the file's default group becomes your own default group, and your preferences are kept. Importing the same file twice adds everything twice. |
| `replace` | Deletes **all** your groups, services, widgets and integrations and creates the file's instead. Saved integration credentials and the status and response-time history of your services go with them; webhook logs of the old services stay but no longer point at a service. Preferences in the file overwrite yours; fields the file leaves out keep their value, except `group_display` (see below). |

An import is all or nothing: if any part of the file is refused, nothing changes. It only ever
touches the dashboard of the user who imports. Everything in the file is checked like a normal
create, so a non-admin can not import admin-only things (Docker integrations, Custom API,
Docker or System stats widgets), and the limits of 50 integrations and 200 widgets per user
apply.

An export leaves out widgets that no import would take: a widget whose integration was deleted,
or a calendar left without any source. If an import still refuses an entry, the error names it
(`widgets[3]`); remove or fix it in the file and import again.

URLs are exported as you entered them. A key in a service, Custom API, RSS or embed URL is in
the file; check for those before you share it.

### Import from another dashboard

**Settings → Export & Import → Import from another dashboard** takes the config of another
dashboard, or a docker-compose file, and adds what it describes to your dashboard. Paste the
config or choose the file, look at the services, groups, integrations and warnings it gives,
choose for each group whether it becomes a new group or goes into one you have, and import.
It is a `merge`: nothing you have is changed or deleted.

| From | What to paste | What is taken over |
|---|---|---|
| [Homepage](https://gethomepage.dev) | `services.yaml` or `bookmarks.yaml` | Groups (a nested group becomes its own group), services with their link, description and icon. A service widget becomes an integration when Nimbus has that app. |
| [Homarr](https://homarr.dev) 0.x | A board config, `data/configs/<board>.json` | Categories as groups, apps as services, an app's integration when Nimbus has that app. Homarr's widgets are not converted. Homarr 1.x has no documented board export yet and is not supported. |
| [Homer](https://github.com/bastienwirtz/homer) | `config.yml` | Service groups and their items; an item's `type` (PiHole, Sonarr, ...) becomes an integration when Nimbus has that app. Navbar links are not converted. |
| [Dashy](https://dashy.to) | `conf.yml` | Sections as groups, items as services. Dashy's widgets are not converted. |
| docker-compose | `docker-compose.yml` | Every service that publishes a TCP port: named after the service, with the icon of its image (registry, organisation, tag and digest are ignored), linking to the first published port. The [Docker discovery labels](INTEGRATIONS.md#labels) are used when present: `nimbus.*` (`name`, `url`, `icon`, `group`, `description`, `widget`), else `homepage.*` (`name`, `href`, `icon`, `description`, `group`); `nimbus.enable=false` skips the service. |

Good to know:

- **Secrets are never taken over.** API keys, passwords and tokens in the config are dropped, and
  a `${VAR}` or `{{HOMEPAGE_VAR_...}}` is never resolved. Imported integrations show **Needs
  credentials** until you enter them under Settings → Integrations.
- **Integrations** are created for AdGuard Home, Home Assistant, Jellyfin, Pi-hole, Proxmox VE,
  qBittorrent, Radarr, Sonarr and Uptime Kuma. Other widget types are listed as a warning and
  the service is imported as a plain link.
- **Icons**: a URL is kept; `si-`, `mdi-`, `sh-` and `hl-` names and plain
  [dashboard-icons](https://github.com/homarr-labs/dashboard-icons) names become the URL of that
  icon on its CDN. Font Awesome icons and files local to the other dashboard can't be taken
  over; those services get the default icon and a warning. An icon name that doesn't exist only
  shows in the browser, where the default icon takes its place.
- **docker-compose host**: a compose file doesn't say which machine it runs on. The wizard asks
  for the host and defaults to the one you opened Nimbus on; links become `http://<host>:<port>`
  (`https` for 443, 8443 and 9443).
- An entry the import would refuse (no URL, a URL that is not `http(s)`, a placeholder like
  `host.or.ip:port`) is skipped with a warning instead of failing the import.

The wizard uses two API calls. The first converts and writes nothing:

```bash
# format: homepage, homarr, homer, dashy or compose; left out, it is detected
# host: only for compose
POST /api/v1/import/convert?format=compose&host=nas.lan
```

The body is the config as text (at most 1 MB); the response is
`{"format": "...", "document": {...}, "warnings": ["..."]}`. The document is then imported like
any other, with `match_groups=1` so a group in the document goes into your group of the same
name instead of becoming a second one:

```bash
POST /api/v1/import?mode=merge&match_groups=1&dry_run=1
```

`match_groups` only looks at your own groups and only applies to a `merge`. A name matches when
it is the same apart from spaces around it; upper and lower case count, so `media` does not go
into `Media`. Your default group is not matched by its name: a document group goes into it when
it has `default: true`, with or without `match_groups`.

### API

Both endpoints need a browser session; API tokens are refused, as for the integrations API.

```bash
# Export
GET /api/v1/export

# Preview: validates and returns the counts, changes nothing
POST /api/v1/import?mode=replace&dry_run=1

# Import
POST /api/v1/import?mode=merge
```

The import body is the document itself (`Content-Type: application/json` is required, at most
10 MB). The response of an import and of its dry run is the same summary:

```json
{
  "mode": "replace",
  "dry_run": true,
  "groups": { "created": 2, "removed": 1 },
  "services": { "created": 12, "removed": 9 },
  "widgets": { "created": 4, "removed": 0 },
  "integrations": { "created": 3, "removed": 2 },
  "preferences": true,
  "needs_credentials": 3,
  "needs_secrets": 1
}
```

A file that is not valid gives `400` with every problem and where it is:

```json
{
  "error": "The dashboard document is not valid",
  "details": [
    { "path": "services[2].group_ref", "message": "no group has the ref \"media\"" },
    { "path": "widgets[0]", "message": "Only admins can set up Custom API widgets" }
  ]
}
```

### Document shape (version 1)

Entities point at each other with **refs**: strings that only mean something inside the file.
There are no database IDs. A ref must be unique among entities of its kind and is only required
on an entity that something points at. Unknown fields and any other `version` are refused.

```json
{
  "version": 1,
  "groups": [
    { "ref": "media", "name": "Media", "color": "#0ea5e9", "default": true, "monitoring_enabled": true, "tile_style": "cards" }
  ],
  "integrations": [
    {
      "ref": "sonarr",
      "kind": "sonarr",
      "name": "Sonarr",
      "base_url": "http://192.168.1.10:8989",
      "auth_type": "api_key",
      "verify_tls": true,
      "options": {},
      "refresh_seconds": 60,
      "needs_credentials": true
    }
  ],
  "services": [
    {
      "ref": "sonarr",
      "name": "Sonarr",
      "url": "http://192.168.1.10:8989",
      "description": "Series",
      "icon": "📺",
      "icon_type": "image_upload",
      "icon_data": "iVBORw0KGgo...",
      "card_size": "2x1",
      "position": 0,
      "group_ref": "media",
      "integration_ref": "sonarr",
      "monitoring_enabled": true
    }
  ],
  "widgets": [
    {
      "ref": "upcoming",
      "type": "calendar",
      "title": "Upcoming",
      "config": { "integrations": ["sonarr"], "ical_urls": ["https://calendar.example.com/********"], "days": 14 },
      "card_size": "2x2",
      "position": 1,
      "refresh_seconds": 900,
      "enabled": true,
      "group_ref": "media",
      "needs_secrets": true
    }
  ],
  "preferences": {
    "theme_mode": "dark",
    "theme_accent_color": "#3B82F6",
    "wallpaper_data": "/9j/4AAQSkZJRg...",
    "wallpaper_blur": 4,
    "wallpaper_dim": 20,
    "card_opacity": 80,
    "card_blur": 12,
    "layout_mode": "canvas",
    "group_display": "sections",
    "theme_preset": "nord",
    "theme_font": "inter",
    "kpi_mono": false,
    "icon_source": "dashboard-icons-svg",
    "custom_css": ".glass-card { border-radius: 1rem; }",
    "card_scale": "medium",
    "view_mode": "grid",
    "open_in_new_tab": true,
    "enable_card_resizing": true,
    "enable_service_grouping": true,
    "status_strip": {
      "enabled": true,
      "chips": [{ "source": "integration", "ref": "sonarr", "kpi": "queued" }]
    }
  }
}
```

| Field | Notes |
|---|---|
| `groups` | Ordered as listed. At most one has `default: true`. `color` is `#RRGGBB`. `tile_style` is `cards` or `chips`; left out, the group gets cards. |
| `integrations[].kind` | A kind from [INTEGRATIONS.md](INTEGRATIONS.md). `auth_type` defaults to the kind's first one. `needs_credentials` is informational. |
| `services[].icon_type` | `emoji` (uses `icon`), `image_url` (uses `icon_url`) or `image_upload` (uses `icon_data`: base64 of a JPEG, PNG, GIF, WebP, ICO or SVG up to 512 KB). The file can not name a file on the server. |
| `services[].position`, `widgets[].position` | Services and widgets share one order. `replace` keeps the positions; `merge` keeps their order and puts them behind your tiles. |
| `widgets[].type`, `config` | A type and its config from [WIDGETS.md](WIDGETS.md). Where a config lists integrations (the calendar), it lists their refs. `needs_secrets` is informational. |
| `preferences` | Optional, as is every field in it. `theme_background` is an image URL, `""` for none; `wallpaper_data` is an uploaded image as base64 (JPEG, PNG, GIF or WebP up to 8 MB). Give one or neither. `theme_accent_color: ""` selects the default accent. `group_display` is `tabs` or `sections` and is the one field that does not keep your value when it is left out: without it, groups follow the layout (tabs in classic, sections in canvas). |
| `preferences.theme_preset`, `theme_font`, `kpi_mono`, `icon_source`, `custom_css` | The theme settings of [THEMING.md](THEMING.md), with the values listed there. Files from before these settings don't have them and keep yours. `custom_css` is checked like CSS you save in the theme settings (no `@import`, `url()` only with an inline `data:` image, at most 20 KB) and refuses the import when it fails; `""` removes your CSS. A replace shows the file's CSS before you confirm: read it when the file is not your own. |
| `status_strip.chips[]` | `source` is `integration` or `widget`; `ref` points at one in the file; `kpi` is the KPI key, or the value's label for a Custom API widget. |
| `omitted_images` | Written by an export, ignored by an import. |

Only `version` is required at the top level; a file with just services is a valid import. For
the limits: 100 groups, 500 services, 200 widgets and 50 integrations per file.
## Public Status Page

Nimbus can share the uptime of one group of services on a page that needs no account: `/status`. It is off by default, and while it is off both the page and its API answer `404`, like any address that does not exist.

Turn it on under **Settings → Admin → Public status page**:

| Setting | Description |
|---------|-------------|
| Enable the status page | Makes `/status` reachable for anyone who can reach your Nimbus |
| Group to show | One of **your own** groups (groups are personal). Only services with monitoring on appear |
| Title | Heading and browser title of the page; `Status` when empty |
| Logo | Optional image next to the title (JPG, PNG, GIF or WEBP, up to 512 KB) |
| Show links | Off by default. When on, each service name links to its URL and icons loaded from a URL are shown. Only `http(s)` URLs are shown, and without the user name and password a URL may carry |

What visitors can see is deliberately little: the title and logo, and per service its name, icon, current status and uptime over the last 24 hours and 7 days. Service URLs, descriptions, response times, error messages, integration data and everything outside the chosen group stay private. An icon that is loaded from a URL would give away where it lives, so without **Show links** such a service shows its emoji instead; uploaded icons are always shown.

Good to know:

- The bars come from the stored status logs: 24 bars of one hour and 28 bars of six hours. Grey means there were no checks in that period.
- The page data is cached for 30 seconds and the page refreshes itself every minute. Saving the settings takes effect at once.
- Each visitor address (for IPv6: each /64 network) may make 300 requests per 15 minutes; a page load or refresh counts as two. Behind your own reverse proxy, let it send `X-Forwarded-For` (most do by default); without it all visitors share one address, and with it one limit. Forwarding headers are only believed from a loopback or private address, so whoever reaches Nimbus from such an address (somebody on your own network, or everybody when your Docker setup hides visitor addresses) can pose as another address. The same goes for everybody when the frontend container is reachable without a reverse proxy in front of it: it passes on whatever the visitor sends.
- The page always follows the visitor's light or dark system setting. Your own theme, accent colour and wallpaper are not used on it.
- With separate containers and no bundled nginx, a switched off `/status` carries an `x-middleware-rewrite` response header that other missing pages do not have. Let your reverse proxy remove it if that matters to you; the unified image does.
- Search engines are asked not to index the page (`noindex`).
- The page turns itself off when the chosen group is deleted or its owner is no longer an admin.

The data is also available as JSON at `GET /api/v1/public/status`.

---

## Config File

Prefer your dashboard in Git? Point `NIMBUS_CONFIG_FILE` at a JSON or YAML file and Nimbus keeps
one admin's dashboard in sync with it. Nimbus stays a normal multi-user app: the file describes
the dashboard of its **owner**, everyone else is untouched.

| Variable | Default | Description |
|----------|---------|-------------|
| `NIMBUS_CONFIG_FILE` | *(none)* | Path of the config file inside the container. Unset means the mode is off. |
| `NIMBUS_CONFIG_MODE` | `replace` | `replace`: each list in the file (`groups`, `integrations`, `services`, `widgets`) is all there is of that kind, what it doesn't have is deleted, and the file's `preferences` are applied. A list the file leaves out is left alone. `merge`: the file only adds and updates its own entries; everything else, and your preferences, stay. |
| `NIMBUS_CONFIG_READONLY` | `false` | `true` locks the owner's groups, services, widgets and integrations against edits in the UI and the API (and the preferences, when the file sets them in `replace` mode). |

### The file

The file is a [dashboard document](#document-shape-version-1), the same shape an export
downloads, with two additions:

- `owner`: the email address of an existing **admin**, written exactly as that admin signs in
  with it (upper and lower case count). Nimbus never creates users from the file.
  On a fresh volume it waits (and logs that it does) until that admin exists, for example after
  the setup page or through `INITIAL_ADMIN_EMAIL`, and applies the file within a minute of that;
  what the admin added by hand in that minute is removed again in `replace` mode. The owner is
  looked up again at every check: when the account is deleted or is no longer an admin, the file
  is no longer applied and a read-only lock lifts, until an admin with that address exists again.
- `credentials` on an integration: `api_key`, `token`, or `username` and `password`, depending
  on its auth type. Only a config file may carry them; the import in the UI and API refuses
  them, and an export never contains them. An integration without `credentials` keeps the ones
  it has.

A string value may use `${VAR}` to take a value from the environment, so secrets stay out of
Git. A variable that is unset or empty is an error that names the variable; a secret never
silently becomes empty. Write `$${VAR}` for a literal `${VAR}`. Values are replaced after the
file is parsed, so the content of a variable can never change the structure of the file, and a
value is not looked at again: a variable that holds `${OTHER}` gives that text.

Any variable of the Nimbus process can be used, also `JWT_SECRET` or the database password, and
its value ends up wherever the file puts it, a service description included. Whoever can write
the file is trusted like whoever sets the environment. Nothing else resolves variables: not the
import in the UI or API, and not the converters for other dashboards.

```yaml
# nimbus.yaml
owner: admin@example.com
version: 1

groups:
  - ref: media
    name: Media
  - ref: network
    name: Network
    color: "#22c55e"

integrations:
  - ref: sonarr
    kind: sonarr
    name: Sonarr
    base_url: http://sonarr:8989
    credentials:
      api_key: ${SONARR_API_KEY}
  - ref: pihole
    kind: pihole
    name: Pi-hole
    base_url: http://192.168.1.2
    credentials:
      token: ${PIHOLE_PASSWORD}

services:
  - name: Sonarr
    url: https://sonarr.example.com
    icon_type: image_url
    icon_url: https://cdn.jsdelivr.net/gh/homarr-labs/dashboard-icons/svg/sonarr.svg
    position: 0
    group_ref: media
    integration_ref: sonarr
  - name: Pi-hole
    url: http://192.168.1.2/admin
    icon: "🛡️"
    position: 1
    group_ref: network
    integration_ref: pihole

widgets:
  - type: clock
    position: 2
  - ref: power
    type: custom_api
    title: Power
    position: 3
    config:
      url: http://192.168.1.30/api/meter
      headers:
        - name: Authorization
          value: Bearer ${METER_TOKEN}
      fields:
        - label: Now
          path: power
          unit: W

preferences:
  theme_mode: dark
  layout_mode: canvas
  status_strip:
    enabled: true
    chips:
      - source: integration
        ref: sonarr
        kpi: queued
```

Mount it read-only and pass the secrets through the environment:

```yaml
# docker-compose.yml
services:
  nimbus:
    image: turboot/nimbus:latest
    environment:
      NIMBUS_CONFIG_FILE: /config/nimbus.yaml
      NIMBUS_CONFIG_READONLY: "true"
      # The owner, created on a fresh volume
      INITIAL_ADMIN_EMAIL: admin@example.com
      INITIAL_ADMIN_PASSWORD: ${NIMBUS_ADMIN_PASSWORD}
      # Used by ${...} in nimbus.yaml
      SONARR_API_KEY: ${SONARR_API_KEY}
      PIHOLE_PASSWORD: ${PIHOLE_PASSWORD}
      METER_TOKEN: ${METER_TOKEN}
    volumes:
      - uploads_data:/app/backend/uploads
      # The directory, not the file: many editors replace a file when they
      # save it, which a mount of the file itself would not see
      - ./config:/config:ro
```

With the file in Git and the secrets in the environment, a new volume gets the same dashboard
again: groups, services, widgets, integrations with working credentials, and preferences.
The quickest way to start is to export your dashboard, add `owner`, and put the credentials
back as `${VAR}`. A YAML file works the same as the exported JSON.

### How the file is applied

The file is read at start and checked for changes every 60 seconds; it is applied when its
content changed and has stayed the same for two seconds, so a file that is still being written
is not applied half. Applying is a **sync**, not an import: Nimbus recognises the entries it
already has and updates them in place, so they keep their status history, widget data and
credentials.

| Entry | Recognised by |
|---|---|
| Group | its name; the group with `default: true` is always your default group |
| Integration | `kind` and `name` |
| Service | `name` and `url`, otherwise `name` alone (so a new URL keeps the history) |
| Widget | `type` and `title` |

Names, titles and URLs must be the same apart from spaces around them; upper and lower case
count, so `sonarr` is not `Sonarr`. Entries that share a key are matched in order: the first
one in the file is the first one you have. Changing what an entry is recognised by (renaming a
service, group, integration or widget) removes the old one and creates a new one, so a renamed
service starts its history over. Anything else is an update in place: a new URL for a service,
moving it to another group, another icon or position. `ref`s are only used inside the file and
can be renamed freely.

In `replace` mode, what the file doesn't list is **deleted**, with its status history. That
goes per list: a list that is in the file (`services:` with entries, or an explicit
`services: []`) is everything there is of that kind, and a list that is left out or left empty
(`services:` with nothing behind it) keeps all its entries (one that sat in a group or used an
integration the file removes loses only that link). So a file that lost a section, or
one with only `preferences`, never empties the dashboard; write `[]` when you really want none.
Taking single entries out of a list does delete them: keep the file in Git.

Only what differs is written. Applying a file that describes the dashboard as it is changes
nothing, so a restart is free. Your default group is never removed, even when the file doesn't
list it.

What you change in the UI between two applies stays until the file changes or the server
restarts; then the file wins for everything it describes. Set `NIMBUS_CONFIG_READONLY=true` to
rule that out: edits by the owner are then refused with `409` and the message that the
dashboard is managed by a config file, and the UI shows a banner and hides or disables its
edit controls (edit mode, add buttons, the discovered-containers tray, editing integrations,
saving a service, importing).
Health checks, widget refreshes, connection tests and the import preview keep working. The lock
applies once the owner is known, so not while Nimbus waits for that admin or when the very
first read of the file fails.

### Together with other features

- **Public status page.** The status page shows one of the admin's groups. A sync keeps a
  group that the file still lists in place, so the page keeps working through renames of other
  entries. When the file no longer has that group (or renames it) and `replace` removes it,
  the status page turns itself off until an admin picks a group again.
- **Docker discovery.** Nimbus remembers which containers it added by the service they became.
  When a sync removes or recreates such a service, that container is not offered again; it
  shows under the ignored containers, where it can be offered again. Auto-add (the admin
  setting) skips the owner of the config file in `replace` mode and whenever the dashboard is
  read-only: the file decides which services there are, and what auto-add put there would be
  removed at the next sync. In `merge` mode without read-only, discovery works as usual.
- **API tokens.** A token of the owner with `write:services` is locked like the owner's
  browser session; reads and `recheck_service` over MCP keep working.

### When something is wrong

A file that can't be read, parsed or validated never stops the server and changes nothing. The
error is logged with what is wrong (the entry and field, or the missing variable), the
dashboard stays as the last good file left it, and the file is applied again as soon as you
fix it:

```
Config file: not applied, the dashboard stays as it is: environment variable not set or empty: SONARR_API_KEY
Config file: applied for admin@example.com: groups +0 ~0 -0, services +1 ~1 -0, widgets +0 ~0 -0, integrations +0 ~1 -0 (created, updated, removed)
```

The log never repeats what a variable held, nor any other value of the file of four characters
or more. The line about an applied file
holds counts only, and an error names the entry (`widgets[2]`) and says what is wrong; where
that explanation would quote a value, a feed URL for example, it is replaced by
`(not shown: the message quotes a value of the file)`. Unknown fields are refused, like in an
import, so a typo is reported instead of ignored; the name of such a field, or of a YAML
anchor that doesn't exist, is the one thing from the file an error does show.

---

## Production Deployment

### Unified Image (Recommended)

The `turboot/nimbus` unified image is the easiest way to deploy. It includes nginx, frontend, and backend in a single container:

```yaml
services:
  nimbus:
    image: turboot/nimbus:latest
    environment:
      DB_PASSWORD: ${DB_PASSWORD}
      JWT_SECRET: ${JWT_SECRET}
      ENCRYPTION_KEY: ${ENCRYPTION_KEY}
      COOKIE_SECURE: "true"  # Enable for HTTPS
    ports:
      - "3000:3000"
```

### HTTPS with Reverse Proxy

For HTTPS, place a reverse proxy (Traefik, Caddy, nginx) in front of the unified container:

```nginx
server {
    listen 443 ssl;
    server_name nimbus.yourdomain.com;

    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    location / {
        proxy_pass http://localhost:3000;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### Separate Containers (Legacy)

The separate `nimbus-backend` and `nimbus-frontend` images from `docker-compose.deprecated.yml` no longer get updates. Use the unified image instead; the [Migration Guide](MIGRATION.md) explains how to switch.

## Complete Example

### Minimal `.env` (Recommended)

```bash
DB_PASSWORD=your-secure-password
JWT_SECRET=your-32-character-minimum-secret-key
ENCRYPTION_KEY=output-of-openssl-rand-base64-32
```

### Full `.env` (All Options)

```bash
# Required
DB_PASSWORD=your-secure-password
JWT_SECRET=your-32-character-minimum-secret-key
ENCRYPTION_KEY=output-of-openssl-rand-base64-32

# Database (all have defaults)
# DB_HOST=db
# DB_PORT=5432
# DB_USER=nimbus
# DB_NAME=nimbus

# Server (all have defaults)
# PORT=8080
# CORS_ORIGINS=http://localhost:3000  # Not needed for unified image

# OAuth (optional)
GOOGLE_CLIENT_ID=
GOOGLE_CLIENT_SECRET=
GOOGLE_REDIRECT_URL=http://localhost:3000/api/v1/auth/oauth/google/callback

GITHUB_CLIENT_ID=
GITHUB_CLIENT_SECRET=
GITHUB_REDIRECT_URL=http://localhost:3000/api/v1/auth/oauth/github/callback

DISCORD_CLIENT_ID=
DISCORD_CLIENT_SECRET=
DISCORD_REDIRECT_URL=http://localhost:3000/api/v1/auth/oauth/discord/callback

# Prometheus (optional)
PROMETHEUS_API_KEY=

# API tokens: requests per minute per token (optional)
# API_TOKEN_RATE_LIMIT=600

# Production
# COOKIE_SECURE=true
# CORS_ORIGINS=https://nimbus.yourdomain.com
```

---

## Troubleshooting

### OAuth callback errors

If OAuth login fails with a redirect error:

1. Ensure redirect URLs match exactly (including protocol and port)
2. For production, update all OAuth redirect URLs to use HTTPS
3. Check that `FRONTEND_URL` is set correctly for the post-login redirect

### API connection errors (separate containers only)

If using `docker-compose.deprecated.yml` and seeing "Cannot reach API server":

1. **Check `NEXT_PUBLIC_API_URL`** — use your actual server IP, not `http://backend:8080`
2. **Check `CORS_ORIGINS`** — must include your frontend URL
3. **Restart containers** after changing `.env`

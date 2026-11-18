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
- [Public Status Page](#public-status-page)
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

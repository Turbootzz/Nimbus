<div align="center">

# ☁️ Nimbus

### Your Homelab, Beautifully Organized

[![License: AGPL v3](https://img.shields.io/badge/License-AGPL_v3-blue.svg)](https://www.gnu.org/licenses/agpl-3.0)
[![Go Version](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![Next.js](https://img.shields.io/badge/Next.js-16-black?logo=next.js&logoColor=white)](https://nextjs.org/)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?logo=docker&logoColor=white)](https://www.docker.com/)

A modern, self-hosted dashboard for monitoring and managing your homelab services.
Multi-user support, real-time health checks, beautiful themes, and Prometheus metrics.

<br />

<img src="docs/images/dashboard-preview.png" alt="Nimbus Dashboard" width="800" />

<br />

[Quick Start](#-quick-start) · [Features](#-features) · [Configuration](docs/CONFIGURATION.md) · [Contributing](#-contributing)

</div>

---

## ✨ Features

<table>
<tr>
<td width="50%">

**🔐 Authentication & Security**
- Local accounts with JWT
- OAuth2 (Google, GitHub, Discord)
- Role-based access control
- Admin panel for user management

**📊 Service Monitoring**
- Real-time health checks
- Response time tracking
- Smart self-signed cert handling
- Status history & uptime graphs
- Public status page: share the uptime of one group at `/status`, no account needed ([docs](docs/CONFIGURATION.md#public-status-page))
- Offline alerts: webhook, Discord, Slack, ntfy, Gotify, Telegram, Pushover ([docs](docs/NOTIFICATIONS.md))
- Hover a tile (or long-press it) for 24h uptime, last check and last error

</td>
<td width="50%">

**🎨 Personalization**
- Custom wallpapers per user (upload or URL) with blur, dim and glass cards
- Canvas layout: the dashboard fills the page under a top bar, no sidebar. The default for new users; switch in Theme settings
- Groups as tabs or as stacked sections that collapse (sections are the default on the canvas)
- Compact chips per group for a quick-launch row of links
- Status strip: key numbers from your apps and an "Operational / N down" pill at the top
- Boards: more than one dashboard per user, a board an admin shares read-only with everyone, and a chrome-less kiosk view for wall screens (`?kiosk=1`) ([docs](docs/BOARDS.md))
- Search: press `/` or Ctrl+K to find and open a service
- Installable app (PWA): add Nimbus to your home screen or desktop; your services list stays readable offline. See [docs/PWA.md](docs/PWA.md)
- Light/dark mode toggle
- In English, Dutch, German, French or Spanish: your browser's language, or the one you pick in your profile (machine-translated for now, corrections welcome: [docs](docs/TRANSLATING.md))
- Accent color themes
- Theme presets (Catppuccin, Nord, Dracula, Gruvbox, Tokyo Night), self-hosted fonts, custom CSS and theme export/import ([docs](docs/THEMING.md))
- Drag & drop service tiles

**📈 Metrics & Integration**
- Configurable check intervals
- Prometheus metrics export
- Dashboard export and import (JSON), for backups and moving between instances
- Scoped API tokens and an MCP endpoint for AI assistants
- Mobile responsive design

</td>
</tr>
</table>

### 🧩 Widgets & integrations (2.0, `dev` image)

Add widgets next to your services with **Add Widget** on the dashboard: Clock, Note (Markdown), Bookmarks, Embed, Weather, RSS, Calendar, Custom API, System stats and Docker containers. Widget data is fetched by the server and pushed to the dashboard live. See [docs/WIDGETS.md](docs/WIDGETS.md).

Integrations show live numbers from your apps on a service tile. Add one under **Settings, Integrations**, then pick it under **Live numbers** when you edit the service. Credentials are stored encrypted with `ENCRYPTION_KEY`.

| App | Status |
|---|---|
| AdGuard Home, Docker, Home Assistant, Jellyfin, Pi-hole v6, Proxmox VE, qBittorrent, Radarr, Sonarr, Uptime Kuma | native |
| Portainer, Prowlarr, SABnzbd, Immich, Tautulli, Jellyseerr / Overseerr, TrueNAS SCALE, Nextcloud, Speedtest Tracker, Beszel, Nginx Proxy Manager, Kavita, Mealie, Gotify, Gluetun, Traefik, Paperless-ngx, Navidrome, Grafana | planned, [up for grabs](https://github.com/Turbootzz/Nimbus/issues/196) |

With a Docker integration, admins get a **Discovered** tray on the Services page that turns containers into services in one click, using `nimbus.*` (or Homepage's `homepage.*`) labels where present. See [Docker discovery](docs/INTEGRATIONS.md#docker-discovery).

What each app shows and how it signs in is in [docs/INTEGRATIONS.md](docs/INTEGRATIONS.md). Adding one is a single Go file plus a test; the same page explains how.

---

## 🔀 Switch from Homepage/Homarr in one minute

Already have a dashboard? Bring it along:

1. Open **Settings → Export & Import → Import from another dashboard**.
2. Paste your config or choose the file: Homepage `services.yaml`, a Homarr 0.x board JSON (Homarr 1.x is not supported yet), Homer `config.yml`, Dashy `conf.yml`, or a `docker-compose.yml`.
3. Check the preview, choose which groups it goes into, and press **Import**.

Groups, links, descriptions and icons come along, and apps Nimbus has an integration for (Sonarr, Radarr, Pi-hole, ...) get one. API keys and passwords are never copied: enter them under **Settings → Integrations** afterwards. Anything that can't be taken over is listed before you import, and nothing you already have is changed. Details per dashboard are in [docs/CONFIGURATION.md](docs/CONFIGURATION.md#import-from-another-dashboard).

Rather keep the dashboard in Git? Set `NIMBUS_CONFIG_FILE` and Nimbus follows a YAML or JSON file, with secrets from the environment: see [Config File](docs/CONFIGURATION.md#config-file).

---

## 🚀 Quick Start

Deploy Nimbus in under 30 seconds with Docker. **Zero configuration required!**

### 1. Create `docker-compose.yml`

```bash
mkdir nimbus && cd nimbus
curl -O https://raw.githubusercontent.com/Turbootzz/Nimbus/main/docker-compose.yml
```

Or create it manually:

<details>
<summary>docker-compose.yml</summary>

```yaml
services:
  db:
    image: turboot/nimbus-postgres:18
    container_name: nimbus-db
    restart: unless-stopped
    environment:
      POSTGRES_DB: nimbus
      POSTGRES_USER: nimbus
      POSTGRES_PASSWORD: ${DB_PASSWORD:-nimbus-default-password}
      PGDATA: /var/lib/postgresql/data
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U nimbus -d nimbus"]
      interval: 10s
      timeout: 5s
      retries: 5

  nimbus:
    image: turboot/nimbus:latest
    container_name: nimbus
    restart: unless-stopped
    environment:
      DB_PASSWORD: ${DB_PASSWORD:-nimbus-default-password}
      JWT_SECRET: ${JWT_SECRET:-}
      ENCRYPTION_KEY: ${ENCRYPTION_KEY:-}
    volumes:
      - uploads_data:/app/backend/uploads
    ports:
      - "3000:3000"
    depends_on:
      db:
        condition: service_healthy

volumes:
  postgres_data:
  uploads_data:
```

</details>

### 2. Start Nimbus

```bash
docker-compose up -d
```

### 3. Open your browser

Navigate to **http://localhost:3000** and create your first account!

> **Note:** Secrets are auto-generated on first run. For production, see [Configuration](docs/CONFIGURATION.md) to set custom passwords.
>
> **Still on the separate `nimbus-backend` and `nimbus-frontend` images?** They no longer get updates. Switching to the unified image takes about 5 minutes and keeps all your data: see the [Migration Guide](docs/MIGRATION.md).

### Image tags

| Tag | What you get |
|-----|--------------|
| `latest` | Stable release (recommended) |
| `1.4.0`, `1.4`, `1` | A pinned version, for controlled upgrades and rollbacks |
| `beta`, `2.0.0-beta.1` | Pre-releases for testers |
| `dev` | Every commit on the `develop` branch; can break at any time |

> **Trying `dev` or `beta`?** Use [`docker-compose.dev.yml`](docker-compose.dev.yml) (`dev` by default, `NIMBUS_TAG=beta` for the pre-release). It runs on port 3001 with its own database, so pre-release migrations never touch your production data. Going back to `latest` on a database that ran `dev` or `beta` is not supported.

---

## ⚙️ Configuration

Nimbus uses **convention over configuration** — sensible defaults are applied automatically.

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_PASSWORD` | `nimbus-default-password` | PostgreSQL password |
| `JWT_SECRET` | *auto-generated* | Auth secret (persisted in volume) |
| `ENCRYPTION_KEY` | *auto-generated* | Encrypts integration credentials and notification tokens (persisted in volume, back it up) |
| `DOCKER_SOCKET` | *(none)* | Docker socket the Docker integration may use, e.g. `/var/run/docker.sock` |
| `NIMBUS_CONFIG_FILE` | *(none)* | Keep an admin's dashboard in sync with a YAML or JSON file ([Config File](docs/CONFIGURATION.md#config-file)); with `NIMBUS_CONFIG_MODE` and `NIMBUS_CONFIG_READONLY` |
| `DB_HOST` | `db` | Database hostname |
| `DB_PORT` | `5432` | Database port |
| `DB_USER` | `nimbus` | Database username |
| `DB_NAME` | `nimbus` | Database name |

### 🤖 AI assistants (MCP)

Nimbus has a [Model Context Protocol](https://modelcontextprotocol.io) endpoint at `/mcp`, so Claude, Open WebUI or another MCP client can answer "what is down?" or "how was Jellyfin's uptime this week?" from your dashboard. It is read-only unless you allow more, and Nimbus itself holds no AI keys.

1. Create a token with the **MCP** scope under **Settings → API Tokens**.
2. Point your client at it, for example Claude Code:

```bash
claude mcp add --transport http nimbus http://localhost:3000/mcp \
  --header "Authorization: Bearer nimbus_your_token"
```

Setup for Claude Desktop, Open WebUI and Home Assistant, the tools and the security notes are in [docs/MCP.md](docs/MCP.md). The same tokens give scripts scoped access to the REST API, see [API Tokens](docs/CONFIGURATION.md#api-tokens).

**For production**, set custom secrets in a `.env` file:
```bash
DB_PASSWORD=your-secure-password
JWT_SECRET=your-32-char-secret
```

**Need OAuth, Prometheus, or custom domains?** See the [Advanced Configuration Guide](docs/CONFIGURATION.md).

---

## ☁️ Don't want to self-host?

[**Nimbus Cloud**](https://nimbusapp.dev) is a managed hosting option for €5/month — same Nimbus, zero setup.

- No Docker, no server — just sign up and go
- Automatic SSL certificates & updates
- Your own subdomain (`you.nimbusapp.dev`)
- All features included, same as self-hosted

Nimbus is and will always be free and open source. Nimbus Cloud is simply for those who prefer a hosted solution.

[**Get Started →**](https://nimbusapp.dev)

---

## 💻 Local Development

### Prerequisites
- Node.js 24+ / Go 1.26+ / PostgreSQL

### Quick Start

```bash
# Clone and setup
git clone https://github.com/Turbootzz/Nimbus.git
cd nimbus
make setup

# Create 'nimbus' database in PostgreSQL
# Update .env with your credentials

# Start development
make dev-backend    # Terminal 1
make dev-frontend   # Terminal 2
```

Run `make help` for all available commands.

---

## 🤝 Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'feat: add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

---

## 📚 Documentation

| Document | Description |
|----------|-------------|
| [Configuration Guide](docs/CONFIGURATION.md) | All environment variables, OAuth setup, API tokens, Prometheus |
| [README.md](README.md) | Information about Nimbus |
| [DEVELOPMENT.md](docs/DEVELOPMENT.md) | 5-minute development setup |
| [WIDGETS.md](docs/WIDGETS.md) | Widget types, RSS and custom API paths |
| [INTEGRATIONS.md](docs/INTEGRATIONS.md) | Supported apps and how to add one |
| [NOTIFICATIONS.md](docs/NOTIFICATIONS.md) | Alerts through webhooks, Discord, Slack, ntfy, Gotify, Telegram and Pushover |
| [BOARDS.md](docs/BOARDS.md) | Several boards, the shared board and kiosk mode for wall screens |
| [MCP.md](docs/MCP.md) | Connect Claude, Open WebUI or Home Assistant to your dashboard |
| [PWA.md](docs/PWA.md) | Install Nimbus as an app, offline behaviour |
| [THEMING.md](docs/THEMING.md) | Theme presets, fonts, icon source, custom CSS and theme files |
| [TRANSLATING.md](docs/TRANSLATING.md) | Languages, improving a translation, adding a language |

---

## 📋 Roadmap

- [x] JWT & OAuth2 authentication
- [x] Real-time health monitoring
- [x] User themes & customization
- [x] Admin panel & RBAC
- [x] Prometheus metrics export
- [x] Mobile responsive design
- [x] Service groups
- [x] Card resizing & dashboard scaling
- [x] List view mode
- [x] Custom service icons (image uploads)
- [x] Uptime webhook notifications
- [x] Optional landing page
- [x] Zero-config Docker deployment
- [ ] Widgets and app integrations (in progress on `dev`)
- [x] PWA support (installable, offline services list)

---

## 🎨 Icon attributions

When you click "Auto-fetch icon", Nimbus may download an icon from one of the following open-source icon collections via the jsDelivr CDN. The icon is cached in your `uploads/service-icons/` volume; the origin is never re-fetched for rendering.

- [Dashboard Icons](https://github.com/homarr-labs/dashboard-icons) — Apache License 2.0
- [selfh.st icons](https://github.com/selfhst/icons) — CC BY 4.0, when you pick it as the icon source in Theme settings

The fonts you can pick in Theme settings ship with Nimbus: [Inter](https://github.com/rsms/inter) and [JetBrains Mono](https://github.com/JetBrains/JetBrainsMono), both under the SIL Open Font License 1.1 (licences in `frontend/styles/fonts/`).

You can point the curated lookup at a self-hosted mirror or disable it entirely with the `DASHBOARD_ICONS_BASE_URL` env var (see [.env.example](.env.example)).

## 📄 License

[GNU Affero General Public License v3](LICENSE)

---

<div align="center">

**Inspired by** [Dashy](https://github.com/Lissy93/dashy), [Homarr](https://github.com/ajnart/homarr), and [Homer](https://github.com/bastienwirtz/homer)

Made for the homelab community

</div>
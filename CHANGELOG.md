# Changelog

All notable changes to Nimbus are listed here.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and Nimbus uses [Semantic Versioning](https://semver.org/spec/v2.0.0.html). Dependency updates are left out; the [GitHub releases](https://github.com/Turbootzz/Nimbus/releases) list every pull request.

## [Unreleased]

Boards, sections, discovery, importers, a public status page, theming, more notification targets, scoped API tokens and an MCP endpoint. Existing users keep their layout, groups and theme until they change them.

### Added

- **Boards**: more than one dashboard per user, a board an admin shares read-only with every user, and a kiosk view for wall screens with `?kiosk=1`. See [docs/BOARDS.md](docs/BOARDS.md). (#230)
- **Sections**: show groups as stacked sections that collapse, instead of tabs. Choose under Settings → Theme → "Show groups as". (#200)
- **Chips**: a group can show its services as compact chips (icon, name and status dot) for a quick-launch row. (#201)
- **Hover detail**: hover, focus or long-press a tile for its 24h uptime, a sparkline, the last check and the last error. (#202)
- **Canvas layout**: a weather chip in the top bar when you have a weather widget, "Edit layout" in the user menu, a single column on phones, and no flash of the classic layout on first load. (#221)
- **Edit from the dashboard**: in edit mode, the pencil on a tile opens the service form in a dialog. (#271)
- **Docker discovery**: admins get a **Discovered** tray on the Services page that turns containers into services, using `nimbus.*` or `homepage.*` labels, and an optional setting to add labelled containers automatically (off by default). See [docs/INTEGRATIONS.md](docs/INTEGRATIONS.md#docker-discovery). (#224)
- **Import from another dashboard**: Homepage, Homarr 0.x, Homer, Dashy and docker-compose files, with a preview and group mapping under Settings → Export & Import. Passwords and API keys are never copied. (#225)
- **Export and import** your dashboard as one JSON file under Settings → Export & Import, merged into or replacing what you have. Integrations are exported without credentials. (#198)
- **Config file**: set `NIMBUS_CONFIG_FILE` to keep an admin's dashboard in sync with a YAML or JSON file, with `${VAR}` for secrets and an optional read-only lock. See [Config File](docs/CONFIGURATION.md#config-file). (#226)
- **Public status page** at `/status`: an admin shares the status and uptime of one group, no login needed. Off by default. See [Public Status Page](docs/CONFIGURATION.md#public-status-page). (#231)
- **Theming**: presets (Catppuccin Mocha and Latte, Nord, Dracula, Gruvbox Dark and Light, Tokyo Night), fonts (System, Inter and JetBrains Mono, shipped with Nimbus), an icon source (Dashboard Icons or selfh.st), custom CSS and theme files. See [docs/THEMING.md](docs/THEMING.md). (#232)
- **Translations**: Dutch, German, French and Spanish for the dashboard, settings and sign-in pages, taken from the browser or chosen in settings. (#234)
- **Notifications** to ntfy, Gotify, Telegram and Pushover, next to webhooks, Discord and Slack, with a **Verify TLS certificate** toggle for self-signed servers. See [docs/NOTIFICATIONS.md](docs/NOTIFICATIONS.md). (#233)
- **Scoped API tokens**: give each token scopes and an optional expiry; requests are rate limited per token. See [API Tokens](docs/CONFIGURATION.md#api-tokens). (#227)
- **MCP endpoint** at `/mcp`: Claude, Open WebUI or another MCP client reads your dashboard with a token that has the `mcp` scope. Setup in [docs/MCP.md](docs/MCP.md). (#228, #229)
- **Installable app (PWA)**: install Nimbus from the user menu; an offline page shows the services from the last sync. Needs HTTPS or localhost. See [docs/PWA.md](docs/PWA.md). (#203)
- **Docker through Portainer**: the Docker integration takes a Portainer environment URL and an API key. (#267)
- **Integration options**: Proxmox VE limited to one node, Uptime Kuma read from a public status page, and up to two extra Home Assistant entities. (#216, #218, #219)
- **Helm chart** in [`charts/nimbus`](charts/nimbus) with optional built-in PostgreSQL, and a Portainer template and app store listing files in [`distribution/`](distribution/README.md) (not submitted to the stores yet). (#235)

### Changed

- New accounts start with sections on the canvas and glass cards (75% opacity, 12px blur). Existing accounts keep tabs and their card look. (#200, #221)
- An integration that needs credentials and has none is not contacted until you enter them. (#198)
- Uptime Kuma no longer shows two numbers labelled "Up": metrics mode shows Up, Down and Up share. (#218)
- Home Assistant states without a unit show as text instead of a number. (#219)
- Webhook errors give the reason the server sent (for example `HTTP 400: Bad Request: chat not found`) and no longer contain the URL. (#233)

### Fixed

- Buttons and text that sit directly on a wallpaper, such as Edit, Add Widget and the group tabs, could be unreadable. They get a card surface now. (#270)
- Hover backgrounds never showed, because their colour was not defined. (#270)
- Closing the emoji picker on the edit service page saved the form. (#271)
- Uploaded profile pictures did not load. (#293)
- With many group tabs the Add button went out of view; the tabs now scroll by wheel, touch and keyboard. (#293)
- The group dialog closes with Esc and keeps focus inside, and screen readers now read out the label of every switch. (#293)
- The integration form kept the previous app's error after you picked another app. (#293)
- Renaming an integration without authentication cleared its last test result. (#284)
- A service name over 255 characters or an icon over 50 gave a server error instead of a validation message. (#198)
- Behind the bundled nginx every visitor was seen as 127.0.0.1, in the activity log and in the forgot-password limit. (#286)
- `docker-compose.yml` did not pass `PROMETHEUS_API_KEY` to the container. (#286)
- Sign Out in the sidebar now goes through the API client; in the unified image it posted to an address built from `NEXT_PUBLIC_API_URL`, which there is `same-origin`. (#289)

### Security

- Requests to integrations and webhooks drop their credentials when a redirect leaves the service (another scheme, host or port), and a redirect that would carry the request body there is refused. (#284)
- Webhooks use the same safe HTTP client as integrations: cloud metadata addresses are blocked and redirects stop after 5. LAN and loopback targets still work. (#233)
- Service URLs must be `http` or `https`. Any scheme, `javascript:` included, was accepted before. (#224)
- Health checks refuse cloud metadata addresses such as 169.254.169.254. LAN addresses stay allowed. (#224)
- The Docker integration is read-only over HTTP too, as it already was over the socket, so a Portainer token can't change anything. (#284)
- Notification tokens are stored encrypted with `ENCRYPTION_KEY` and never sent back to the browser. (#233)
- API tokens are deny-by-default: a token only reaches what its scopes allow, and never acts as an admin. (#227)

### Upgrading from 2.0

1. **Back up your database.** Migrations `000035` to `000042` run on start and only go forward. Your dashboard stays as it was: groups stay tabs, cards keep their look, the theme stays Default, and everything you have moves to a default board called "Home".
2. **API tokens have scopes now.** Existing tokens keep their read access (full tokens also `write:services`), but can no longer reach webhooks, preferences, uploads, widget writes or the live stream, and an admin's token no longer reads other users' Prometheus metrics. No existing token gets the `mcp` scope. Each token is limited to `API_TOKEN_RATE_LIMIT` requests per minute (default 600). See [Tokens created before scopes](docs/CONFIGURATION.md#tokens-created-before-scopes).
3. **Service URLs must be `http` or `https`.** A service with another scheme keeps working and can still be moved, resized or renamed; changing its URL needs an `http` or `https` one.
4. **Redirects to another address lose credentials.** An integration whose URL redirects to another host, port or scheme (apex to `www`, or `http://nas:5000` to `https://nas:5001`) gets a 401 now, and a webhook URL that redirects that way no longer delivers. Use the final URL. Plain `http` on port 80 to `https` on port 443 of the same host still counts as the same address.
5. **Webhooks use the safe HTTP client.** Cloud metadata addresses are blocked and redirects stop after 5 (was 10). LAN and loopback targets and `HTTP(S)_PROXY` still work.
6. **Public status page on the default group.** Testers who turned it on with a `dev` or `beta` image: when the page shows your default group, it now also lists your ungrouped, monitored services.
7. **Optional: config file.** `NIMBUS_CONFIG_FILE`, `NIMBUS_CONFIG_MODE` and `NIMBUS_CONFIG_READONLY` are new. In the default `replace` mode, items missing from the file's lists are removed at the next check, also ones added in the UI. See [Config File](docs/CONFIGURATION.md#config-file).

## [2.0.0] - 2026-10-06

Widgets and integrations. Existing users see no change until they turn something on: the layout stays classic, and there is no wallpaper and no widgets until they add them.

### Added

- **Widgets** sit in the same grid as your services: clock, weather, markdown, bookmarks, iframe, RSS, calendar (Sonarr/Radarr and iCal feeds), and for admins custom API, system stats and Docker containers. See [docs/WIDGETS.md](docs/WIDGETS.md).
- **Integrations**: link a service to an app and its card shows the key numbers. Supported apps: Sonarr, Radarr, qBittorrent, Pi-hole v6, AdGuard Home, Proxmox VE, Jellyfin, Home Assistant, Uptime Kuma, and Docker (admins only). Integrations belong to the user who adds them. See [docs/INTEGRATIONS.md](docs/INTEGRATIONS.md).
- **Live dashboard**: widgets, integrations and service status update over a server-sent event stream instead of polling.
- **Wallpaper**: upload an image or use a URL, with blur, dim, and see-through glass cards on every page.
- **Canvas layout**: the dashboard fills the page under a top bar with a clock, without the sidebar. New users start on it. Existing users keep the classic layout and can switch in Settings → Theme.
- **Status strip**: up to six key numbers from your integrations and custom API widgets at the top of the dashboard, plus an "Operational / N down" pill.
- **Search**: press `/` or Ctrl/⌘+K to find and open a service.
- `dev` and `beta` image tags for testers, with [`docker-compose.dev.yml`](docker-compose.dev.yml) to run them next to a production install.

### Changed

- The separate `nimbus-backend` and `nimbus-frontend` images are no longer published. Use the unified `turboot/nimbus` image; [docs/MIGRATION.md](docs/MIGRATION.md) explains how to switch.

### Fixed

- The gap between the sidebar and the page is gone.

### Security

- Integration credentials are stored encrypted with `ENCRYPTION_KEY`.
- The Docker integration is for admins only, and Nimbus only uses a Docker socket when `DOCKER_SOCKET` is set.

### Upgrading from 1.3

1. **Back up your database.** Migrations `000026` to `000034` run on start and only go forward.
2. **Keep the uploads volume.** `ENCRYPTION_KEY` encrypts integration credentials. The Docker image generates it on first start and stores it in the uploads volume (`uploads/.secrets/generated.env`). If you run Nimbus without Docker, set it yourself with `openssl rand -base64 32`. If the key is lost, integration credentials have to be entered again.
3. **Optional: `DOCKER_SOCKET`.** To use the Docker integration, mount the socket read-only and set `DOCKER_SOCKET` to its path. Otherwise, point the integration at a socket proxy. See [docs/CONFIGURATION.md](docs/CONFIGURATION.md).
4. **Use the `turboot/nimbus` image.** The separate backend and frontend images are no longer published.

## [1.3.1] - 2026-08-20

### Added

- `SMTP_TLS_MODE` (`starttls`, `tls` or `none`) and `SMTP_TLS_SKIP_VERIFY`, so outgoing mail also works with a relay that has no TLS.

### Fixed

- The Postgres image moves a database that an older `postgres:18` image created one level too deep (`PGDATA/data`) back to where it belongs.

## [1.3.0] - 2026-08-05

### Added

- Personal access tokens for using the API from scripts.

## [1.2.3] - 2026-06-28

### Added

- Delete your own account.
- Auto-fetch a service icon from its name or URL.
- Sign in with any OpenID Connect provider (Keycloak, Authentik, Dex and others). Thanks to [@caraar12345](https://github.com/caraar12345).

### Changed

- The OAuth providers share one code path.

### Fixed

- Profile pictures were broken; remote avatar URLs no longer go through image optimization.
- Services lost their group after a card size change.
- Avatar files were left behind on disk when an account was deleted.
- After creating a service you return to the page you started from.
- A non-UUID `:id` in an API path returned a 500 instead of a 400.

## [1.2.2] - 2026-03-24

### Fixed

- The sidebar broke when you closed it and refreshed the page.
- Uploaded icons did not load on local installs because of CORS.

## [1.2.1] - 2026-02-13

### Added

- Password reset by email, with SMTP settings in the environment or the admin panel.
- Support for the hosted Nimbus Cloud: `DISABLE_PUBLIC_REGISTRATION` and an initial admin account from `INITIAL_ADMIN_EMAIL` / `INITIAL_ADMIN_PASSWORD`.

### Changed

- A service is only marked offline for webhook notifications after a configurable number of retries with a delay, to avoid false alarms.
- The desktop sidebar can be collapsed.

## [1.2.0] - 2026-02-03

### Added

- Setup wizard on first start.
- Exclude a service from monitoring.
- Migration guide for the old separate-container compose file.

### Fixed

- A Postgres setup that could lose data after an image update. The database now runs on `turboot/nimbus-postgres:18` with `PGDATA` set, and existing data is moved on start.
- New services did not always land in the chosen group on another device.
- Database passwords with special characters.

## [1.1.0] - 2026-01-22

### Added

- Unified Docker image `turboot/nimbus` with frontend, backend and nginx in one container.
- Service groups.
- Card resizing.
- Dashboard scaling and a list view.
- Webhook notifications when a service goes down or comes back.

### Changed

- Clearer errors when the API can't be reached.

### Fixed

- Uploaded service images were not saved properly.

## [1.0.2] - 2025-12-28

### Added

- Optional landing page.

### Changed

- Simpler self-hosting: fewer environment variables, a shorter `docker-compose.yml`, and new configuration and development guides.

## [1.0.1] - 2025-12-27

### Fixed

- "Remember me" did not work when signing in with OAuth.

## [1.0.0] - 2025-11-19

First stable release.

### Added

- Local accounts with JWT sessions and "remember me", plus sign-in with Google, GitHub or Discord.
- Admin and user roles, with an admin panel to manage users.
- Services with a name, URL, description and icon (emoji, upload or image URL), drag and drop ordering, and a choice to open links in the same or a new tab.
- Health checks with online/offline status, response times, uptime history and graphs. Self-signed certificates and `.local` hosts work.
- Prometheus metrics export with a configurable retention period.
- Light, dark and automatic theme, background image by URL, and accent colors, per user.
- Mobile layout.
- Docker images for amd64 and arm64 with a docker-compose setup on PostgreSQL 18.

[Unreleased]: https://github.com/Turbootzz/Nimbus/compare/v2.0.0...develop
[2.0.0]: https://github.com/Turbootzz/Nimbus/compare/v1.3.1...v2.0.0
[1.3.1]: https://github.com/Turbootzz/Nimbus/compare/v1.3.0...v1.3.1
[1.3.0]: https://github.com/Turbootzz/Nimbus/compare/v1.2.3...v1.3.0
[1.2.3]: https://github.com/Turbootzz/Nimbus/compare/v1.2.2...v1.2.3
[1.2.2]: https://github.com/Turbootzz/Nimbus/compare/v1.2.1...v1.2.2
[1.2.1]: https://github.com/Turbootzz/Nimbus/compare/v1.2.0...v1.2.1
[1.2.0]: https://github.com/Turbootzz/Nimbus/compare/v1.1.0...v1.2.0
[1.1.0]: https://github.com/Turbootzz/Nimbus/compare/v1.0.2...v1.1.0
[1.0.2]: https://github.com/Turbootzz/Nimbus/compare/v1.0.1...v1.0.2
[1.0.1]: https://github.com/Turbootzz/Nimbus/compare/v1.0.0...v1.0.1
[1.0.0]: https://github.com/Turbootzz/Nimbus/releases/tag/v1.0.0

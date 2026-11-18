# Changelog

All notable changes to Nimbus are listed here.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and Nimbus uses [Semantic Versioning](https://semver.org/spec/v2.0.0.html). Dependency updates are left out; the [GitHub releases](https://github.com/Turbootzz/Nimbus/releases) list every pull request.

## [Unreleased]

Changes merged to `develop` since 2.0.0 are added here before the next release.

### Added

- _Nothing listed yet._

### Changed

- _Nothing listed yet._

### Fixed

- _Nothing listed yet._

### Security

- _Nothing listed yet._

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

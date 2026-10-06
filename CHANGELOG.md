# Changelog

## 2.0.0

Widgets and integrations. Existing users see no change until they turn something on: the layout stays classic, and there is no wallpaper and no widgets until they add them.

### New

- **Widgets** sit in the same grid as your services: clock, weather, markdown, bookmarks, iframe, RSS, calendar (Sonarr/Radarr and iCal feeds), and for admins custom API, system stats and Docker containers. See [docs/WIDGETS.md](docs/WIDGETS.md).
- **Integrations**: link a service to an app and its card shows the key numbers. Supported apps: Sonarr, Radarr, qBittorrent, Pi-hole v6, AdGuard Home, Proxmox VE, Jellyfin, Home Assistant, Uptime Kuma, and Docker (admins only). Integrations belong to the user who adds them. Credentials are stored encrypted. See [docs/INTEGRATIONS.md](docs/INTEGRATIONS.md).
- **Live dashboard**: widgets, integrations and service status update over a server-sent event stream instead of polling.
- **Wallpaper**: upload an image or use a URL, with blur, dim, and see-through glass cards on every page.
- **Canvas layout**: the dashboard fills the page under a top bar with a clock, without the sidebar. New users start on it. Existing users keep the classic layout and can switch in Settings → Theme.
- **Status strip**: up to six key numbers from your integrations and custom API widgets at the top of the dashboard, plus an "Operational / N down" pill.
- **Search**: press `/` or Ctrl/⌘+K to find and open a service.

### Upgrading from 1.3

1. **Back up your database.** Migrations `000026` to `000034` run on start and only go forward.
2. **Keep the uploads volume.** `ENCRYPTION_KEY` encrypts integration credentials. The Docker image generates it on first start and stores it in the uploads volume (`uploads/.secrets/generated.env`). If you run Nimbus without Docker, set it yourself with `openssl rand -base64 32`. If the key is lost, integration credentials have to be entered again.
3. **Optional: `DOCKER_SOCKET`.** To use the Docker integration, mount the socket read-only and set `DOCKER_SOCKET` to its path. Otherwise, point the integration at a socket proxy. See [docs/CONFIGURATION.md](docs/CONFIGURATION.md).
4. **Use the `turboot/nimbus` image.** The separate backend and frontend images are no longer published.

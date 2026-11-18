# Roadmap

Where Nimbus is heading. The work is grouped in four phases, each a [GitHub milestone](https://github.com/Turbootzz/Nimbus/milestones). The phase names are not release numbers: a feature ships in the first release after it is merged. What has shipped is in the [changelog](CHANGELOG.md).

Want something that is not here? Open an idea in [Discussions](https://github.com/Turbootzz/Nimbus/discussions), or use the [integration request](https://github.com/Turbootzz/Nimbus/issues/new?template=integration_request.yml) form for a new app.

| Phase | Theme | Status |
|---|---|---|
| [v1.3 Quick wins](https://github.com/Turbootzz/Nimbus/milestone/1) | Small features with a visible effect | Partly shipped in 2.0.0, rest in progress |
| [v2.0 Widgets & integrations](https://github.com/Turbootzz/Nimbus/milestone/2) | Widgets, live numbers from your apps, canvas layout | Released as [2.0.0](https://github.com/Turbootzz/Nimbus/releases/tag/v2.0.0) on 2026-10-06 |
| [v2.1 Discovery & MCP](https://github.com/Turbootzz/Nimbus/milestone/3) | Less typing to set up, access for AI assistants | In progress |
| [v2.2 Polish & reach](https://github.com/Turbootzz/Nimbus/milestone/4) | Boards, theming, languages, app stores | Mostly planned |

"In progress" below means there is an open pull request; it has not shipped yet.

## v1.3 Quick wins

[Milestone](https://github.com/Turbootzz/Nimbus/milestone/1)

Shipped in 2.0.0:

- Command palette and search with `/` or Ctrl/⌘+K ([#197](https://github.com/Turbootzz/Nimbus/issues/197))
- Wallpaper upload with blur, dim and glass cards ([#199](https://github.com/Turbootzz/Nimbus/issues/199))

In progress:

- Dashboard export and import as JSON ([#198](https://github.com/Turbootzz/Nimbus/issues/198), [PR #285](https://github.com/Turbootzz/Nimbus/pull/285))
- Groups as stacked sections, next to tabs ([#200](https://github.com/Turbootzz/Nimbus/issues/200), [PR #287](https://github.com/Turbootzz/Nimbus/pull/287))
- Compact chip tiles for quick-launch groups ([#201](https://github.com/Turbootzz/Nimbus/issues/201), [PR #287](https://github.com/Turbootzz/Nimbus/pull/287))
- Card hover detail with a 24h uptime sparkline and the last error ([#202](https://github.com/Turbootzz/Nimbus/issues/202), [PR #287](https://github.com/Turbootzz/Nimbus/pull/287))
- PWA support: install prompt and offline page ([#203](https://github.com/Turbootzz/Nimbus/issues/203), [PR #289](https://github.com/Turbootzz/Nimbus/pull/289))
- Roadmap, changelog and Discussions ([#204](https://github.com/Turbootzz/Nimbus/issues/204))

## v2.0 Widgets & integrations

[Milestone](https://github.com/Turbootzz/Nimbus/milestone/2) · [epic #195](https://github.com/Turbootzz/Nimbus/issues/195)

Shipped in 2.0.0:

- Integration framework with encrypted credentials and a connection test ([#205](https://github.com/Turbootzz/Nimbus/issues/205))
- Widget poller, cache and live updates over server-sent events ([#206](https://github.com/Turbootzz/Nimbus/issues/206))
- Mixed grid of services and widgets with an add-widget picker ([#207](https://github.com/Turbootzz/Nimbus/issues/207))
- Widgets: clock, weather, custom API, iframe, RSS, markdown, bookmarks ([#208](https://github.com/Turbootzz/Nimbus/issues/208)), system stats and Docker containers ([#209](https://github.com/Turbootzz/Nimbus/issues/209)), calendar ([#210](https://github.com/Turbootzz/Nimbus/issues/210))
- Integrations: Sonarr ([#211](https://github.com/Turbootzz/Nimbus/issues/211)), Radarr ([#212](https://github.com/Turbootzz/Nimbus/issues/212)), qBittorrent ([#213](https://github.com/Turbootzz/Nimbus/issues/213)), Pi-hole v6 ([#214](https://github.com/Turbootzz/Nimbus/issues/214)), AdGuard Home ([#215](https://github.com/Turbootzz/Nimbus/issues/215)), Proxmox VE ([#216](https://github.com/Turbootzz/Nimbus/issues/216)), Docker ([#217](https://github.com/Turbootzz/Nimbus/issues/217)), Uptime Kuma ([#218](https://github.com/Turbootzz/Nimbus/issues/218)), Home Assistant ([#219](https://github.com/Turbootzz/Nimbus/issues/219)), Jellyfin ([#220](https://github.com/Turbootzz/Nimbus/issues/220))
- Canvas layout ([#221](https://github.com/Turbootzz/Nimbus/issues/221))
- Status strip with key numbers ([#222](https://github.com/Turbootzz/Nimbus/issues/222))
- Contributor guide and template for new integrations ([#223](https://github.com/Turbootzz/Nimbus/issues/223))

In progress:

- Docker through Portainer with an API key ([#267](https://github.com/Turbootzz/Nimbus/issues/267), [PR #281](https://github.com/Turbootzz/Nimbus/pull/281))
- Per-app options, finishing Proxmox VE, Uptime Kuma and Home Assistant ([#216](https://github.com/Turbootzz/Nimbus/issues/216), [#218](https://github.com/Turbootzz/Nimbus/issues/218), [#219](https://github.com/Turbootzz/Nimbus/issues/219), [PR #284](https://github.com/Turbootzz/Nimbus/pull/284))
- Finishing the canvas layout with sections ([#221](https://github.com/Turbootzz/Nimbus/issues/221), [PR #287](https://github.com/Turbootzz/Nimbus/pull/287))

## v2.1 Discovery & MCP

[Milestone](https://github.com/Turbootzz/Nimbus/milestone/3)

In progress:

- Edit services from the dashboard ([#271](https://github.com/Turbootzz/Nimbus/issues/271), [PR #282](https://github.com/Turbootzz/Nimbus/pull/282))
- Find services from Docker containers and their labels (`nimbus.*` and `homepage.*`) ([#224](https://github.com/Turbootzz/Nimbus/issues/224), [PR #288](https://github.com/Turbootzz/Nimbus/pull/288))
- Scoped API tokens ([#227](https://github.com/Turbootzz/Nimbus/issues/227), [PR #286](https://github.com/Turbootzz/Nimbus/pull/286))
- MCP server at `/mcp` ([#228](https://github.com/Turbootzz/Nimbus/issues/228), [PR #286](https://github.com/Turbootzz/Nimbus/pull/286)), with a guide for Claude, Open WebUI and Home Assistant ([#229](https://github.com/Turbootzz/Nimbus/issues/229))

Planned:

- Importers for Homepage, Homarr, Homer, Dashy and docker-compose files ([#225](https://github.com/Turbootzz/Nimbus/issues/225))
- Config-file mode with `NIMBUS_CONFIG_FILE` ([#226](https://github.com/Turbootzz/Nimbus/issues/226))
- More integrations from the community wishlist ([#196](https://github.com/Turbootzz/Nimbus/issues/196)): pick one and open a pull request

## v2.2 Polish & reach

[Milestone](https://github.com/Turbootzz/Nimbus/milestone/4)

In progress:

- Notifications to ntfy, Gotify, Telegram and Pushover ([#233](https://github.com/Turbootzz/Nimbus/issues/233), [PR #283](https://github.com/Turbootzz/Nimbus/pull/283))

Planned:

- Multiple boards, a shared board from the admin, kiosk mode ([#230](https://github.com/Turbootzz/Nimbus/issues/230))
- Public status page ([#231](https://github.com/Turbootzz/Nimbus/issues/231))
- Theme presets, font choice and custom CSS ([#232](https://github.com/Turbootzz/Nimbus/issues/232))
- Translations: Dutch, German, French and Spanish ([#234](https://github.com/Turbootzz/Nimbus/issues/234))
- App store listings and a Helm chart ([#235](https://github.com/Turbootzz/Nimbus/issues/235)). The chart is in [`charts/nimbus`](charts/nimbus) and the listing files are in [`distribution/`](distribution/README.md); submitting them to the stores is still to do.
- Research: SQLite as a second database backend ([#236](https://github.com/Turbootzz/Nimbus/issues/236))

# Widgets

Widgets are tiles that sit in the dashboard grid next to your services. Add one with **Add Widget** on the dashboard, then move and resize it in edit mode like a service.

Static widgets (clock, note, bookmarks, embed) render in your browser. The others are fetched by the Nimbus server on an interval and pushed to open dashboards live, so a slow or unreachable source never blocks the page. When a fetch fails, the tile keeps the last good data and shows a warning.

## Widget types

| Widget | Shows | Sizes | Fetched by |
|---|---|---|---|
| Clock | Time and date in any timezone | 1x1, 2x1, 2x2 | browser |
| Note | Markdown text | all | browser |
| Bookmarks | A compact list of links | all | browser |
| Embed | A web page in an iframe | 1x2, 2x2 | browser |
| Weather | Current weather and a forecast from [Open-Meteo](https://open-meteo.com) (no key needed) | 1x1, 2x1, 2x2 | server, at most every 10 minutes |
| RSS | The latest items from up to three feeds | all | server, at most every 5 minutes |
| Calendar | Upcoming episodes (Sonarr), movie releases (Radarr) and iCal events | 2x1, 1x2, 2x2 | server, at most every 5 minutes |
| Custom API | Up to four values from any JSON API (admins only) | 1x1, 2x1, 2x2 | server |
| System stats | CPU, memory and disk use and uptime of the Nimbus server (admins only) | all | server, every 30 seconds by default |
| Docker containers | The containers of a [Docker integration](INTEGRATIONS.md) and their state (admins only) | all | server, every minute by default |

## RSS

Paste the feed URL of up to three feeds. RSS 2.0, RSS 1.0, Atom and JSON Feed work. Items from all feeds are merged, newest first, and items without a date go last. If one feed fails, the others still show, with a note on the tile.

## Calendar

Pick up to four Sonarr and Radarr integrations, and add up to five iCal feeds (the secret iCal address of a Google, Outlook or Nextcloud calendar works). At 2x2 the tile shows this month with a colored dot per source on each day, and the agenda below it; click a day to see only that day. Smaller tiles show the agenda only. The agenda covers the next 14 days by default (up to 60).

Repeating iCal events (`RRULE`) only show on their first date for now. Events with a time zone (`TZID`) or in UTC are shown in your browser's time; all-day events keep their date. If one source fails, the others still show, with a note on the tile.

## Custom API

The custom API widget is for apps without a native [integration](INTEGRATIONS.md). The server GETs the URL, which must return JSON, and picks up to four values out of the response. Only admins can add one or change what it fetches, because it shows whatever the server can read on your network.

Each value has a label, a path and an optional unit. Paths use dots for keys and brackets for list items:

| Path | Picks |
|---|---|
| `stats.cpu` | the `cpu` key inside `stats` |
| `items[0].name` | `name` of the first item |
| `items[-1].name` | `name` of the last item |
| `items.length` | the number of items |
| `$[0].id` | `id` of the first item when the response is a list |

Each label can be used once. A path must point at a value (a number, text, true/false or null), not at an object or a list. A path that finds nothing shows `-` with the path in its tooltip, since APIs often leave out empty keys; when no path finds anything, the tile shows an error.

Headers are optional, for example `X-Api-Key` for apps that want a key, or `Accept` to ask for another format. The API never sends a saved header value back: the form shows `********`, and the value is kept as long as you leave it. Header values are stored with the widget, **not encrypted**; for apps Nimbus has an integration for, use the integration, which keeps its credentials encrypted. Headers are never sent along when the API redirects to another host.

## System stats

Shows the machine Nimbus runs on. Pick the disk by giving a path on it (`/` by default). Docker doesn't isolate CPU, memory and uptime, so inside a container those are the host's numbers; for a host disk, mount it into the container and give that path. CPU use is measured between two refreshes. Only admins can add it, since it reads any path on the server.

## Docker containers

Lists the containers of a Docker integration: running ones first, then by name, with their status on wide tiles. Turn on **Hide stopped containers** to leave out exited and created ones; paused and restarting containers stay, and the count above the list still includes all of them.

## URLs the server fetches

RSS feeds and custom API URLs are fetched by the Nimbus server, so they can point at hosts on your LAN (`http://192.168.1.10:8080`). Turn off **Verify TLS certificate** for a host with a self-signed certificate. Cloud metadata addresses are blocked, responses are capped at 1 MB, and errors never include the URL, since a query may hold a key.

## Adding a widget type

Each type is one Go file in `backend/internal/widgets/` that registers itself in `init()`, plus a renderer and a config form in `frontend/components/widgets/` registered in `registry.ts`. Types that fetch data implement `Fetcher` and must use `req.Client`, the SSRF-safe client.

A type that reads one integration sets `Meta.IntegrationKinds` and gets it in `req.Integration`; the add-widget form shows a picker. A type that lists several in its config (like the calendar) sets `Meta.ConfigIntegrationKinds` and implements `ConfigIntegrations`: the server checks they are the user's and of those kinds, and `Fetch` gets them connected in `req.Linked`.

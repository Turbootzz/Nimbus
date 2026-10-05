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
| Custom API | Up to four values from any JSON API | 1x1, 2x1, 2x2 | server |

## RSS

Paste the feed URL of up to three feeds. RSS 2.0, RSS 1.0, Atom and JSON Feed work. Items from all feeds are merged, newest first, and items without a date go last. If one feed fails, the others still show, with a note on the tile.

## Custom API

The custom API widget is for apps without a native [integration](INTEGRATIONS.md). The server GETs the URL, which must return JSON, and picks up to four values out of the response.

Each value has a label, a path and an optional unit. Paths use dots for keys and brackets for list items:

| Path | Picks |
|---|---|
| `stats.cpu` | the `cpu` key inside `stats` |
| `items[0].name` | `name` of the first item |
| `items[-1].name` | `name` of the last item |
| `items.length` | the number of items |
| `$[0].id` | `id` of the first item when the response is a list |

A path must point at a value (a number, text, true/false or null), not at an object or a list. A path that finds nothing shows as an error on the tile, so a typo is easy to spot.

Headers are optional, for example `X-Api-Key` for apps that want a key. Header values are stored with the widget and are **not encrypted**. For apps Nimbus has an integration for, use the integration, which keeps its credentials encrypted.

## URLs the server fetches

RSS feeds and custom API URLs are fetched by the Nimbus server, so they can point at hosts on your LAN (`http://192.168.1.10:8080`). Cloud metadata addresses are blocked, responses are capped at 1 MB, and errors never include the URL, since a query may hold a key.

## Adding a widget type

Each type is one Go file in `backend/internal/widgets/` that registers itself in `init()`, plus a renderer and a config form in `frontend/components/widgets/` registered in `registry.ts`. Types that fetch data implement `Fetcher` and must use `req.Client`, the SSRF-safe client.

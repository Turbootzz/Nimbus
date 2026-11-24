# Install Nimbus as an app

Nimbus is a Progressive Web App: you can install it on a phone or desktop and it opens in its
own window, without the browser bars. When the server cannot be reached, it shows the services
list from your last visit.

## Requirements

- Nimbus must be served over **HTTPS** (or opened on `localhost`). Browsers do not allow
  installing or offline use on plain `http://` addresses such as `http://192.168.1.10:3000`.
  Put it behind a reverse proxy with a certificate, see
  [Production Deployment](CONFIGURATION.md#production-deployment).
- The unified Docker image, where the app and the API share one address. This is the default.

## Installing

| Browser                                   | How                                                                                   |
| ----------------------------------------- | ------------------------------------------------------------------------------------- |
| Chrome, Edge, Brave (desktop and Android) | Open the user menu (top right) and choose **Install app**                             |
| Safari on iPhone and iPad                 | Share button, then **Add to Home Screen**                                             |
| Safari on macOS                           | **File**, then **Add to Dock**                                                        |
| Firefox                                   | Desktop Firefox cannot install web apps; on Android use the browser menu, **Install** |

**Install app** only shows up when the browser offers an install and Nimbus is not installed
yet. Safari and Firefox do not tell a page whether it can be installed, so the entry is hidden
there; use the browser's own menu.

The app takes its name from `NEXT_PUBLIC_APP_NAME` (a build argument of the image, default
`Nimbus`).

## Offline behaviour

- **While online nothing is served from a cache.** Every API request goes to the server and its
  answer, errors included, is shown as is. Only build files with a hashed name (scripts, styles,
  fonts) and the icons are served from the cache first.
- **When the server cannot be reached**, opening Nimbus shows an offline page with the services
  you had at the last sync, when that was, and their links. It is read-only and statuses are not
  updated. As soon as the connection is back, the page returns to the app by itself.
- Kept for offline use, on this device only: your services, groups, widgets, the latest widget
  data and your preferences. Never kept: sign-in requests, API tokens, webhooks, integrations,
  admin pages and anything that changes data.
- **Shared devices:** the saved data is removed when you sign out, when the login page is
  opened, when anyone signs in or registers, and when a different user turns out to be signed
  in. A session that expires on its own leaves the data in place until Nimbus is opened online
  again: offline, Nimbus cannot tell who is looking, so the offline page shows the last synced
  list to whoever opens it. Sign out on a device you share.

## Updates

After you update Nimbus, an installed app picks up the new version the next time it loads a
page. Nothing needs to be reinstalled.

## Troubleshooting

- **No "Install app" entry:** check that the address starts with `https://`, that you are not
  already in the installed app, and that the browser is Chromium based.
- **Stale app after switching between a production build and `make dev-frontend` on
  `localhost:3000`:** the dev server removes the service worker on its first page load; reload
  once more.

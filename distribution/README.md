# Distribution

Files for listing Nimbus in homelab app stores. Each folder is laid out like the repository it goes to, so you can copy it over as it is. Nothing here has been submitted yet.

Every listing runs the unified image `turboot/nimbus` with PostgreSQL, and follows the same rules as [`docker-compose.yml`](../docker-compose.yml):

- The database is set with `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_NAME` and `DB_PASSWORD`. Not `DB_URL`: the entrypoint waits for the database and checks it with the separate variables.
- `/app/backend/uploads` is persistent. `JWT_SECRET` and `ENCRYPTION_KEY` are left empty, so Nimbus generates them into `uploads/.secrets/`. A store-generated secret would not do: `ENCRYPTION_KEY` has to be base64 of 32 bytes.
- Only the database password comes from the store, and only letters and digits.
- The image tag is pinned to `2.0.0` where the store asks for a pinned version.

| Target | Files | Checked with | Installed on the real platform |
|---|---|---|---|
| Kubernetes | [`charts/nimbus`](../charts/nimbus) | `helm lint`, `helm template` (all toggles), kubeconform | No |
| Portainer | [`portainer/templates.json`](portainer/templates.json) | JSON parse only | No |
| Dockge | [`docker-compose.yml`](../docker-compose.yml) | Already the supported setup | No |
| Unraid | [`unraid/`](unraid) | `xmllint` only | No |
| Runtipi | [`runtipi/apps/nimbus`](runtipi/apps/nimbus) | Store JSON schemas, `docker compose config` | No |
| CasaOS / ZimaOS | [`casaos/Apps/Nimbus`](casaos/Apps/Nimbus) | Store's `validate_compose.py` | No |
| Umbrel | [`umbrel/nimbus`](umbrel/nimbus) | Store's `npm run lint:apps -- nimbus --check-images` | No |
| TrueNAS | [`truenas/ix-dev/community/nimbus`](truenas/ix-dev/community/nimbus) | Catalog validator, render with the catalog library, container run with the rendered capabilities | No |

"No" in the last column means exactly that: no listing has been installed through its store. Do that once before you open the pull request.

## For every store

Still to do by hand:

- **Screenshots.** The images in [`docs/images`](../docs/images) are from before 2.0. Take new ones first; every store wants two to five.
- **Icon.** [`logo-square.png`](../frontend/public/images/logo-square.png) is 272×272. Umbrel wants an SVG and most stores want at least 512×512, so export a larger one.
- **Licence.** GitHub does not recognise the `LICENSE` file as AGPL-3.0 (the API reports no licence), because the file starts with a project notice. Some reviewers check this.

## Kubernetes (Helm)

```bash
helm install nimbus ./charts/nimbus \
  --namespace nimbus --create-namespace \
  --set secrets.dbPassword="$(openssl rand -hex 16)"
```

- `postgresql.enabled=true` (default) runs one `turboot/nimbus-postgres:18` StatefulSet. Set it to `false` and fill `externalDatabase.*` to use your own server; that user must be allowed to `CREATE EXTENSION pgcrypto`.
- `secrets.existingSecret` takes a Secret with the keys `db-password` and, optionally, `jwt-secret` and `encryption-key`.
- The uploads PVC is kept on `helm uninstall`, because it holds the generated `ENCRYPTION_KEY`.
- One replica only. Nimbus keeps live state in memory.

Still to do: publish the chart (GitHub Pages with chart-releaser, or `helm push` to an OCI registry) and list it on Artifact Hub. Not tested on a cluster.

## Portainer

`portainer/templates.json` is an app template file in the v3 format with one stack template that deploys [`docker-compose.yml`](../docker-compose.yml) from this repository.

- Use it: Portainer → Settings → App Templates → URL `https://raw.githubusercontent.com/Turbootzz/Nimbus/main/distribution/portainer/templates.json`. Portainer takes one URL, so this replaces the default list. The URL works once this file is on `main`.
- Reach more people: add the template object to a community list. Not tested in Portainer.

## Dockge

Dockge has no template format; it manages plain compose files. Create a stack named `nimbus` and paste [`docker-compose.yml`](../docker-compose.yml).

## Unraid Community Applications

Two templates, because an Unraid template is one container: `unraid/nimbus-postgres.xml` (install first) and `unraid/nimbus.xml`. Containers on Unraid's default bridge can't find each other by name, so the Nimbus template asks for the server's IP as database host. On a custom Docker network `nimbus-postgres` works as host.

To submit:

1. Create a public templates repository (for example `Turbootzz/unraid-templates`) and copy both files into it.
2. Change `<TemplateURL>` in both files to the raw URL in that repository.
3. Use the same value for the icon, or a larger PNG.
4. Request inclusion through the Community Applications submission form; the pinned topic in the Community Applications forum has the current link and the rules. You need an Unraid forum account, and a support thread there is expected.

Untested on Unraid. Check on a real server that both containers start and that the Nimbus tile opens the WebUI.

## Runtipi

`runtipi/apps/nimbus` holds `config.json`, `docker-compose.json` (dynamic compose, schema v2), the legacy `docker-compose.yml`, and `metadata/` with the description and logo.

The official `runtipi/runtipi-appstore` no longer accepts new apps (see its README). The options are:

- **Your own app store**: a repository with this `apps/` folder at its root. Users add its URL in Runtipi's app store settings.
- **A community store**: open a pull request with `apps/nimbus` to one of the stores listed in the Runtipi discussions.

Port `8114` is free in the official store today; a community store may need another. Untested in Runtipi.

## CasaOS / ZimaOS

`casaos/Apps/Nimbus` holds `docker-compose.yml` with the top-level `x-casaos` block and `icon.png`.

To submit: fork `IceWhaleTech/CasaOS-AppStore`, copy the folder to `Apps/Nimbus`, add `thumbnail.png` and `screenshot-1.png` to `screenshot-3.png`, list them under `x-casaos.thumbnail` and `x-casaos.screenshot_link` the way the other apps do, run `./scripts/build_dist.sh`, and open a pull request.

- The `icon` URL points at the store repository and resolves after the merge.
- CasaOS has no generated secrets, so the database password is fixed in the file, as in our own compose file. The database is not published on the host.
- The store's build step (`build_dist.sh`) was not run here. Untested in CasaOS.

## Umbrel

`umbrel/nimbus` holds `umbrel-app.yml`, `docker-compose.yml`, `exports.sh` (derives the database password) and the empty `data/` folders.

To submit: fork `getumbrel/umbrel-apps`, copy the folder to `nimbus/`, run `npm run lint:apps -- nimbus --check-images`, test it on umbrelOS, and open a pull request. Then put the pull request URL in `submission`. Attach the icon and three to five screenshots to the pull request; the Umbrel team hosts them and fills `gallery`.

- Both images are pinned by multi-arch digest. Update tag and digest together (`docker buildx imagetools inspect turboot/nimbus:<tag>`).
- Port `3023` is free in the store today; the linter checks it.
- Umbrel's login stays in front of Nimbus. API calls with a personal access token and Prometheus scrapes are blocked by it until you add the paths to `PROXY_AUTH_WHITELIST`.
- The Docker socket integration is not available: store apps may not mount the socket.

Untested on umbrelOS.

## TrueNAS

`truenas/ix-dev/community/nimbus` is a community-train app for `truenas/apps`. It uses the catalog library's Postgres 18 (stock `postgres` image, as the library requires) instead of `turboot/nimbus-postgres`; Nimbus works with both. The first-start log still mentions `nimbus-postgres`, which can be ignored there.

To submit: fork `truenas/apps`, copy the folder to `ix-dev/community/nimbus`, and on Linux run:

```bash
./.github/scripts/port_validation.py
./.github/scripts/generate_metadata.py --train community --app nimbus
./.github/scripts/ci.py --train community --app nimbus --test-file basic-values.yaml
```

Then open a pull request and attach the icon and screenshots; the reviewer uploads them and gives you the final URLs for `app.yaml`.

- `item.yaml` is normally generated by `generate_metadata.py`. Here it was written by hand to mirror `app.yaml`: the script's render step fails in the validation container under amd64 emulation on an Apple Silicon Mac. Regenerate it.
- Default port `30003` was the next free one on 2026-11-18. Run `port_validation.py` again.
- The Nimbus container runs as root with the capabilities `CHOWN`, `DAC_OVERRIDE`, `SETGID` and `SETUID`. Without `DAC_OVERRIDE` nginx does not start.
- The catalog prefers images from `ghcr.io`; Nimbus is on Docker Hub only.

The full `ci.py` deploy test was not run here. Untested on TrueNAS.

## For a new Nimbus release

Change the version in: `charts/nimbus/Chart.yaml` (`appVersion`, and bump `version`), `runtipi/apps/nimbus` (`config.json` `version` and `tipi_version`, both compose files), `casaos/Apps/Nimbus/docker-compose.yml` (image, `version`, `update_at`, `release_notes`), `umbrel/nimbus` (`version`, `releaseNotes`, image tag and digest), `truenas/.../app.yaml` (`app_version`, bump `version`) and `ix_values.yaml` (tag and digest).

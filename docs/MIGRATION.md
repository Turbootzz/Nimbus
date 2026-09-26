# Moving to the unified image

The separate `nimbus-backend` and `nimbus-frontend` images no longer get updates. Your current setup keeps working, but new features and security fixes only ship in the unified image, `turboot/nimbus`.

Switching takes about 5 minutes. The unified stack uses the same database and the same volumes, so your users, services, settings and uploaded icons all stay. Everyone stays logged in as long as `JWT_SECRET` keeps the same value.

## What changes

| | Old | New |
|---|---|---|
| Containers | `nimbus-db`, `nimbus-backend`, `nimbus-frontend` | `nimbus-db`, `nimbus` |
| Ports | 3000 for the web app, 8080 for the API | 3000 for everything |
| `NEXT_PUBLIC_API_URL`, `CORS_ORIGINS` | Required | Not needed |
| `DB_PASSWORD`, `JWT_SECRET` | | Keep the same values |

## 1. Make a backup

Run this on the machine where Nimbus runs:

```bash
docker exec nimbus-db pg_dump -U nimbus --clean --if-exists nimbus > nimbus-backup.sql
docker cp nimbus-backend:/app/uploads ./uploads-backup
```

Also keep a copy of your settings: `cp .env .env.backup` if you use a `.env` file, or write down the environment variables of your Portainer stack.

## 2. Switch

Did you add your own lines under `environment:` in the old compose file, such as OAuth, SMTP or `FRONTEND_URL`? Copy them to the `nimbus` service in the new file. Docker only passes on variables that are listed there, so a value in `.env` or in Portainer alone is not enough.

### Docker Compose

Run these commands in the folder that holds your `docker-compose.deprecated.yml`. The folder matters: Docker names the volumes after it, so starting from another folder gives you an empty database. If you started the old stack with `docker-compose` (with a hyphen), use that command here too.

```bash
# Stop the old stack. Don't add -v, that deletes your data.
docker compose -f docker-compose.deprecated.yml down

# Get the new compose file and start it. Your .env is reused.
curl -O https://raw.githubusercontent.com/Turbootzz/Nimbus/main/docker-compose.yml
docker compose up -d
```

### Portainer

Edit your existing stack instead of creating a new one. The stack name decides the volume names, so a new stack starts with an empty database.

1. Open your Nimbus stack and stop it. Updating a running stack fails because the old frontend still uses port 3000.
2. In the **Editor**, replace everything with the contents of [docker-compose.yml](https://raw.githubusercontent.com/Turbootzz/Nimbus/main/docker-compose.yml).
3. Keep `DB_PASSWORD` and `JWT_SECRET` in the environment variables. You can remove `NEXT_PUBLIC_API_URL` and `CORS_ORIGINS`.
4. Turn on **Prune services** and click **Update the stack**. If your Portainer has no prune option, delete the old `nimbus-backend` and `nimbus-frontend` containers afterwards.

## 3. Check that it works

Open Nimbus on port 3000 and log in. Your services and icons should all be there. From the command line:

```bash
curl http://localhost:3000/api/v1/health
# {"app":"nimbus","database":"connected","status":"healthy"}
```

## 4. Update what pointed at port 8080

- **Reverse proxy**: send all traffic to port 3000. The separate `/api` route to port 8080 is no longer needed.
- **OAuth or OIDC login**: if a redirect URL uses port 8080, change it to 3000, both at your provider and in the `*_REDIRECT_URL` variable.
- **Prometheus**: scrape `nimbus:3000` instead of `nimbus-backend:8080`. The paths stay the same.

## Troubleshooting

**Nimbus looks brand new, with no users or services.** The new stack created fresh volumes. Your data is still in the old ones (`docker volume ls`). Stop the new stack and start it from the same folder, or under the same Portainer stack name, as before.

**Port 3000 is already in use.** The old frontend is still running. Stop the old stack first. If the `nimbus` container still won't start, remove it with `docker rm -f nimbus` and start the stack again.

**Everyone is logged out.** `JWT_SECRET` changed. Put the old value back from your backup.

**Database connection failed.** `DB_PASSWORD` doesn't match the old one. Use the value from your backup.

## Going back

Stop the new stack and start the old one:

```bash
docker compose down
docker compose -f docker-compose.deprecated.yml up -d
```

If the old version shows database errors, the unified image has already updated your database. Load your backup to undo that:

```bash
docker stop nimbus-backend
docker exec -i nimbus-db psql -U nimbus nimbus < nimbus-backup.sql
docker start nimbus-backend
```

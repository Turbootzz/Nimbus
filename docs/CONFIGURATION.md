# Advanced Configuration Guide

This guide covers all configuration options for Nimbus, including OAuth setup, Prometheus integration, and production deployment.

---

## Table of Contents

- [Environment Variables](#environment-variables)
  - [Required Variables](#required-variables)
  - [Database Configuration](#database-configuration)
  - [Server Configuration](#server-configuration)
  - [Authentication](#authentication)
  - [Encryption Key](#encryption-key)
  - [Health Checks](#health-checks)
  - [Metrics & Monitoring](#metrics--monitoring)
- [OAuth Setup](#oauth-setup)
  - [Google OAuth](#google-oauth)
  - [GitHub OAuth](#github-oauth)
  - [Discord OAuth](#discord-oauth)
- [Prometheus Integration](#prometheus-integration)
- [Production Deployment](#production-deployment)

---

## Environment Variables

Nimbus uses **convention over configuration**. Most variables have sensible defaults — you only need to set secrets.

### Required Variables

**None!** Nimbus works out of the box with sensible defaults.

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_PASSWORD` | `nimbus-default-password` | PostgreSQL password |
| `JWT_SECRET` | *auto-generated* | Authentication secret (persisted in uploads volume) |
| `ENCRYPTION_KEY` | *auto-generated* | Encrypts integration credentials (persisted in uploads volume) |

> **Security note:** The default database password is safe because PostgreSQL is only accessible within the Docker network (not exposed externally). For shared hosting environments, set a custom password.

**For production**, set custom secrets:
```bash
# Create .env file with secure passwords
cat > .env << EOF
DB_PASSWORD=$(openssl rand -base64 24 | tr -dc 'a-zA-Z0-9' | head -c 24)
JWT_SECRET=$(openssl rand -base64 32 | tr -dc 'a-zA-Z0-9' | head -c 32)
ENCRYPTION_KEY=$(openssl rand -base64 32)
EOF
```

> **⚠️ Password Character Warning:** Avoid special characters (`%`, `@`, `&`, `*`, `#`, `^`) in `DB_PASSWORD`. These can cause connection issues with PostgreSQL. Use alphanumeric characters and `-` or `_` only.

### Database Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_HOST` | `db` | PostgreSQL hostname (use `db` for Docker, `localhost` for local dev) |
| `DB_PORT` | `5432` | PostgreSQL port |
| `DB_NAME` | `nimbus` | Database name |
| `DB_USER` | `nimbus` | Database username |
| `DB_PASSWORD` | *required* | Database password |
| `DB_URL` | - | Alternative: full connection string (overrides individual vars) |

**Example `DB_URL`:**
```
postgres://nimbus:password@localhost:5432/nimbus?sslmode=disable
```

### Server Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | Backend API port |
| `CORS_ORIGINS` | *none* | Comma-separated allowed origins (optional in unified mode) |
| `COOKIE_SECURE` | `false` | Set to `true` for HTTPS |
| `COOKIE_DOMAIN` | - | Cookie domain restriction |

> **Note:** When using the unified `turboot/nimbus` Docker image, `CORS_ORIGINS` is not needed — the built-in nginx proxy handles same-origin requests.

### Frontend API URL

| Variable | Default | Description |
|----------|---------|-------------|
| `NEXT_PUBLIC_API_URL` | *auto-detect* | Backend API URL for browser requests |

**Values:**
- `same-origin` — Use relative `/api/v1` path (unified Docker image)
- `http://server:8080` — Full URL (separate containers)
- *empty/unset* — Auto-detect based on browser location

> **Note:** The unified image sets this automatically. Only configure manually for separate container deployments.

**Production CORS example (separate containers only):**
```bash
CORS_ORIGINS=https://nimbus.example.com,https://www.nimbus.example.com
```

### Authentication

| Variable | Default | Description |
|----------|---------|-------------|
| `JWT_SECRET` | *required* | Secret for signing JWT tokens (min 32 chars) |
| `JWT_EXPIRY` | `24h` | Token expiration (e.g., `24h`, `7d`, `30d`) |
| `BCRYPT_COST` | `10` | Password hashing cost (10-12 recommended) |

### Encryption Key

| Variable | Default | Description |
|----------|---------|-------------|
| `ENCRYPTION_KEY` | *auto-generated* (Docker and `make dev-backend`) | Base64 encoded 32-byte key that encrypts integration credentials (AES-256-GCM) |

Generate one with `openssl rand -base64 32`. The unified image creates it on first start and saves it in `uploads/.secrets/generated.env`, next to `JWT_SECRET`. A key you set yourself always wins over the generated one. For local development, `make dev-backend` adds one to your `.env`.

**Keep this key safe.** API keys and passwords of your integrations are stored encrypted with it. If the key is lost or changed, Nimbus can't read them anymore and you have to enter them again. Back up the uploads volume, or set your own key and store it somewhere safe. Nimbus refuses to start without a valid key.

### Health Checks

| Variable | Default | Description |
|----------|---------|-------------|
| `HEALTH_CHECK_INTERVAL` | `60` | Seconds between health checks |
| `HEALTH_CHECK_TIMEOUT` | `10` | Request timeout in seconds |

**Smart TLS Verification:**
Nimbus automatically handles self-signed certificates for local/private IPs:
- Public services → Full certificate verification
- Local services (192.168.x.x, 10.x.x.x, etc.) → Skips verification

No configuration needed — works with Portainer, Proxmox, and other homelab services.

### Metrics & Monitoring

| Variable | Default | Description |
|----------|---------|-------------|
| `METRICS_RETENTION_DAYS` | `30` | Days to retain status logs |
| `PROMETHEUS_API_KEY` | - | API key for Prometheus endpoint |

---

## OAuth Setup

OAuth providers are optional. Leave the variables empty to disable a provider.

### Google OAuth

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create a new project or select existing
3. Navigate to **APIs & Services** → **Credentials**
4. Click **Create Credentials** → **OAuth client ID**
5. Select **Web application**
6. Add authorized redirect URI: `http://localhost:3000/api/v1/auth/oauth/google/callback`

```bash
GOOGLE_CLIENT_ID=your-client-id.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=your-client-secret
GOOGLE_REDIRECT_URL=http://localhost:3000/api/v1/auth/oauth/google/callback
```

### GitHub OAuth

1. Go to [GitHub Developer Settings](https://github.com/settings/developers)
2. Click **New OAuth App**
3. Set Authorization callback URL: `http://localhost:3000/api/v1/auth/oauth/github/callback`

```bash
GITHUB_CLIENT_ID=your-client-id
GITHUB_CLIENT_SECRET=your-client-secret
GITHUB_REDIRECT_URL=http://localhost:3000/api/v1/auth/oauth/github/callback
```

### Discord OAuth

1. Go to [Discord Developer Portal](https://discord.com/developers/applications)
2. Create a new application
3. Go to **OAuth2** → **General**
4. Add redirect: `http://localhost:3000/api/v1/auth/oauth/discord/callback`

```bash
DISCORD_CLIENT_ID=your-client-id
DISCORD_CLIENT_SECRET=your-client-secret
DISCORD_REDIRECT_URL=http://localhost:3000/api/v1/auth/oauth/discord/callback
```

### OAuth State Secret (Optional)

For additional CSRF protection:

```bash
OAUTH_STATE_SECRET=your-oauth-state-secret
```

If not set, falls back to `JWT_SECRET`.

---

## Prometheus Integration

Export service metrics in Prometheus format for monitoring and alerting.

### 1. Generate an API Key

```bash
openssl rand -hex 32
```

### 2. Add to Environment

```bash
PROMETHEUS_API_KEY=your-generated-key
```

### 3. Create `prometheus.yml`

```yaml
global:
  scrape_interval: 30s

scrape_configs:
  - job_name: 'nimbus'
    scrape_interval: 30s
    metrics_path: '/api/v1/prometheus/metrics/user/YOUR_USER_ID'
    authorization:
      type: Bearer
      credentials: 'YOUR_API_KEY'
    static_configs:
      - targets: ['host.docker.internal:3000']
```

### 4. Deploy Prometheus

```bash
docker run -d \
  --name prometheus \
  -p 9090:9090 \
  --add-host=host.docker.internal:host-gateway \
  -v $(pwd)/prometheus.yml:/etc/prometheus/prometheus.yml:ro \
  prom/prometheus:latest
```

### Available Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `nimbus_service_up` | Gauge | Service status (1=up, 0=down) |
| `nimbus_service_response_time_ms` | Gauge | Response time in milliseconds |
| `nimbus_service_last_check_timestamp` | Gauge | Unix timestamp of last check |

---

## Production Deployment

### Unified Image (Recommended)

The `turboot/nimbus` unified image is the easiest way to deploy. It includes nginx, frontend, and backend in a single container:

```yaml
services:
  nimbus:
    image: turboot/nimbus:latest
    environment:
      DB_PASSWORD: ${DB_PASSWORD}
      JWT_SECRET: ${JWT_SECRET}
      ENCRYPTION_KEY: ${ENCRYPTION_KEY}
      COOKIE_SECURE: "true"  # Enable for HTTPS
    ports:
      - "3000:3000"
```

### HTTPS with Reverse Proxy

For HTTPS, place a reverse proxy (Traefik, Caddy, nginx) in front of the unified container:

```nginx
server {
    listen 443 ssl;
    server_name nimbus.yourdomain.com;

    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    location / {
        proxy_pass http://localhost:3000;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### Separate Containers (Legacy)

The separate `nimbus-backend` and `nimbus-frontend` images from `docker-compose.deprecated.yml` no longer get updates. Use the unified image instead; the [Migration Guide](MIGRATION.md) explains how to switch.

## Complete Example

### Minimal `.env` (Recommended)

```bash
DB_PASSWORD=your-secure-password
JWT_SECRET=your-32-character-minimum-secret-key
ENCRYPTION_KEY=output-of-openssl-rand-base64-32
```

### Full `.env` (All Options)

```bash
# Required
DB_PASSWORD=your-secure-password
JWT_SECRET=your-32-character-minimum-secret-key
ENCRYPTION_KEY=output-of-openssl-rand-base64-32

# Database (all have defaults)
# DB_HOST=db
# DB_PORT=5432
# DB_USER=nimbus
# DB_NAME=nimbus

# Server (all have defaults)
# PORT=8080
# CORS_ORIGINS=http://localhost:3000  # Not needed for unified image

# OAuth (optional)
GOOGLE_CLIENT_ID=
GOOGLE_CLIENT_SECRET=
GOOGLE_REDIRECT_URL=http://localhost:3000/api/v1/auth/oauth/google/callback

GITHUB_CLIENT_ID=
GITHUB_CLIENT_SECRET=
GITHUB_REDIRECT_URL=http://localhost:3000/api/v1/auth/oauth/github/callback

DISCORD_CLIENT_ID=
DISCORD_CLIENT_SECRET=
DISCORD_REDIRECT_URL=http://localhost:3000/api/v1/auth/oauth/discord/callback

# Prometheus (optional)
PROMETHEUS_API_KEY=

# Production
# COOKIE_SECURE=true
# CORS_ORIGINS=https://nimbus.yourdomain.com
```

---

## Troubleshooting

### OAuth callback errors

If OAuth login fails with a redirect error:

1. Ensure redirect URLs match exactly (including protocol and port)
2. For production, update all OAuth redirect URLs to use HTTPS
3. Check that `FRONTEND_URL` is set correctly for the post-login redirect

### API connection errors (separate containers only)

If using `docker-compose.deprecated.yml` and seeing "Cannot reach API server":

1. **Check `NEXT_PUBLIC_API_URL`** — use your actual server IP, not `http://backend:8080`
2. **Check `CORS_ORIGINS`** — must include your frontend URL
3. **Restart containers** after changing `.env`

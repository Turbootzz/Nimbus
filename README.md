<div align="center">

# ☁️ Nimbus

### Your Homelab, Beautifully Organized

[![License: AGPL v3](https://img.shields.io/badge/License-AGPL_v3-blue.svg)](https://www.gnu.org/licenses/agpl-3.0)
[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![Next.js](https://img.shields.io/badge/Next.js-16-black?logo=next.js&logoColor=white)](https://nextjs.org/)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?logo=docker&logoColor=white)](https://www.docker.com/)

A modern, self-hosted dashboard for monitoring and managing your homelab services.
Multi-user support, real-time health checks, beautiful themes, and Prometheus metrics.

<br />

<img src="docs/images/dashboard-preview.png" alt="Nimbus Dashboard" width="800" />

<br />

[Quick Start](#-quick-start) · [Features](#-features) · [Configuration](docs/CONFIGURATION.md) · [Contributing](#-contributing)

</div>

---

## ✨ Features

<table>
<tr>
<td width="50%">

**🔐 Authentication & Security**
- Local accounts with JWT
- OAuth2 (Google, GitHub, Discord)
- Role-based access control
- Admin panel for user management

**📊 Service Monitoring**
- Real-time health checks
- Response time tracking
- Smart self-signed cert handling
- Status history & uptime graphs

</td>
<td width="50%">

**🎨 Personalization**
- Custom backgrounds per user
- Light/dark mode toggle
- Accent color themes
- Drag & drop service tiles

**📈 Metrics & Integration**
- Configurable check intervals
- Prometheus metrics export
- Mobile responsive design

</td>
</tr>
</table>

---

## 🚀 Quick Start

Deploy Nimbus in under 30 seconds with Docker. **Zero configuration required!**

### 1. Create `docker-compose.yml`

```bash
mkdir nimbus && cd nimbus
curl -O https://raw.githubusercontent.com/Turbootzz/Nimbus/main/docker-compose.yml
```

Or create it manually:

<details>
<summary>docker-compose.yml</summary>

```yaml
services:
  db:
    image: turboot/nimbus-postgres:18
    container_name: nimbus-db
    restart: unless-stopped
    environment:
      POSTGRES_DB: nimbus
      POSTGRES_USER: nimbus
      POSTGRES_PASSWORD: ${DB_PASSWORD:-nimbus-default-password}
      PGDATA: /var/lib/postgresql/data
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U nimbus -d nimbus"]
      interval: 10s
      timeout: 5s
      retries: 5

  nimbus:
    image: turboot/nimbus:latest
    container_name: nimbus
    restart: unless-stopped
    environment:
      DB_PASSWORD: ${DB_PASSWORD:-nimbus-default-password}
      JWT_SECRET: ${JWT_SECRET:-}
    volumes:
      - uploads_data:/app/backend/uploads
    ports:
      - "3000:3000"
    depends_on:
      db:
        condition: service_healthy

volumes:
  postgres_data:
  uploads_data:
```

</details>

### 2. Start Nimbus

```bash
docker-compose up -d
```

### 3. Open your browser

Navigate to **http://localhost:3000** and create your first account!

> **Note:** Secrets are auto-generated on first run. For production, see [Configuration](docs/CONFIGURATION.md) to set custom passwords.
>
> **Upgrading from separate container images?** See our [Migration Guide](docs/MIGRATION.md) for step-by-step instructions. PostgreSQL data migrates automatically!

---

## ⚙️ Configuration

Nimbus uses **convention over configuration** — sensible defaults are applied automatically.

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_PASSWORD` | `nimbus-default-password` | PostgreSQL password |
| `JWT_SECRET` | *auto-generated* | Auth secret (persisted in volume) |
| `DB_HOST` | `db` | Database hostname |
| `DB_PORT` | `5432` | Database port |
| `DB_USER` | `nimbus` | Database username |
| `DB_NAME` | `nimbus` | Database name |

**For production**, set custom secrets in a `.env` file:
```bash
DB_PASSWORD=your-secure-password
JWT_SECRET=your-32-char-secret
```

**Need OAuth, Prometheus, or custom domains?** See the [Advanced Configuration Guide](docs/CONFIGURATION.md).

<details>
<summary><b>Advanced: Separate Container Deployment</b></summary>

For users who prefer separate frontend/backend containers (e.g., for custom reverse proxy setups), use `docker-compose.deprecated.yml`:

```bash
docker-compose -f docker-compose.deprecated.yml up -d
```

This requires configuring `CORS_ORIGINS` and `NEXT_PUBLIC_API_URL` manually.
</details>

---

## 💻 Local Development

### Prerequisites
- Node.js 24+ / Go 1.25+ / PostgreSQL

### Quick Start

```bash
# Clone and setup
git clone https://github.com/Turbootzz/Nimbus.git
cd nimbus
make setup

# Create 'nimbus' database in PostgreSQL
# Update .env with your credentials

# Start development
make dev-backend    # Terminal 1
make dev-frontend   # Terminal 2
```

Run `make help` for all available commands.

---

## 🤝 Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'feat: add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

---

## 📚 Documentation

| Document | Description |
|----------|-------------|
| [Configuration Guide](docs/CONFIGURATION.md) | All environment variables, OAuth setup, Prometheus |
| [README.md](README.md) | Information about Nimbus |
| [DEVELOPMENT.md](docs/DEVELOPMENT.md) | 5-minute development setup |

---

## 📋 Roadmap

- [x] JWT & OAuth2 authentication
- [x] Real-time health monitoring
- [x] User themes & customization
- [x] Admin panel & RBAC
- [x] Prometheus metrics export
- [x] Mobile responsive design
- [x] Service groups
- [x] Card resizing & dashboard scaling
- [x] List view mode
- [x] Custom service icons (image uploads)
- [x] Uptime webhook notifications
- [x] Optional landing page
- [x] Zero-config Docker deployment
- [ ] Widget/plugin system
- [ ] PWA support

---

## 📄 License

[GNU Affero General Public License v3](LICENSE)

---

<div align="center">

**Inspired by** [Dashy](https://github.com/Lissy93/dashy), [Homarr](https://github.com/ajnart/homarr), and [Homer](https://github.com/bastienwirtz/homer)

Made for the homelab community

</div>
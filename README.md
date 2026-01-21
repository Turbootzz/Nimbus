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

Deploy Nimbus in under 2 minutes with Docker. **Only 2 environment variables required!**

### 1. Make a directory

```bash
mkdir nimbus && cd nimbus
```

### 2. Create a `.env` file

```bash
# Generate a secure JWT secret
JWT_SECRET=$(openssl rand -base64 32)

# Create .env with required variables
cat > .env << EOF
DB_PASSWORD=your-secure-database-password
JWT_SECRET=$JWT_SECRET
EOF
```

### 3. Create `docker-compose.yml`

```yaml
services:
  db:
    image: postgres:18-alpine
    container_name: nimbus-db
    restart: unless-stopped
    environment:
      POSTGRES_DB: nimbus
      POSTGRES_USER: nimbus
      POSTGRES_PASSWORD: ${DB_PASSWORD}
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U nimbus -d nimbus"]
      interval: 10s
      timeout: 5s
      retries: 5

  backend:
    image: turboot/nimbus-backend:latest
    container_name: nimbus-backend
    restart: unless-stopped
    environment:
      DB_PASSWORD: ${DB_PASSWORD}
      JWT_SECRET: ${JWT_SECRET}
      CORS_ORIGINS: ${CORS_ORIGINS:-http://localhost:3000}
    volumes:
      - uploads_data:/app/uploads
    ports:
      - "8080:8080"
    depends_on:
      db:
        condition: service_healthy

  frontend:
    image: turboot/nimbus-frontend:latest
    container_name: nimbus-frontend
    restart: unless-stopped
    environment:
      JWT_SECRET: ${JWT_SECRET}
    ports:
      - "3000:3000"
    depends_on:
      - backend

volumes:
  postgres_data:
  uploads_data:
```

### 4. Start Nimbus

```bash
docker-compose up -d
```

### 5. Open your browser

Navigate to **http://localhost:3000** and create your first account!

---

## ⚙️ Configuration

Nimbus uses **convention over configuration** — sensible defaults are applied automatically.

Variables can be set in your `.env` file or directly in `docker-compose.yml`.

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_PASSWORD` | *required* | PostgreSQL password |
| `JWT_SECRET` | *required* | Auth secret (32+ chars) |
| `PORT` | `8080` | Backend API port |
| `DB_HOST` | `db` | Database hostname |
| `DB_PORT` | `5432` | Database port |
| `DB_USER` | `nimbus` | Database username |
| `DB_NAME` | `nimbus` | Database name |
| `CORS_ORIGINS` | `http://localhost:3000` | Allowed origins |

**Need OAuth, Prometheus, or custom domains?** See the [Advanced Configuration Guide](docs/CONFIGURATION.md).

---

## 💻 Local Development

### Prerequisites
- Node.js 20+ / Go 1.21+ / PostgreSQL

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
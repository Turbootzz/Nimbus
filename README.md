# Nimbus - Self-Hosted Homelab Dashboard

A modern, customizable dashboard for your homelab and personal servers with per-user personalization, health monitoring, and role-based access control.

## Features

### MVP Features
- **Authentication & Accounts**: User registration/login with JWT-based authentication
- **Role-Based Access Control**: Admin and user roles with different permissions
- **Dashboard UI**: Grid-based service tiles with icons and health status
- **Health Monitoring**: Automatic service availability checking with visual indicators
- **User Personalization**: Custom backgrounds, themes, and light/dark mode per user
- **Configuration Management**: Add/edit services via web UI or JSON import/export

### Tech Stack
- **Frontend**: Next.js 14 + React + TypeScript + Tailwind CSS
- **Backend**: Go + Fiber framework
- **Database**: PostgreSQL
- **Deployment**: Docker + Docker Compose

## Getting Started

### Prerequisites
- Node.js 20+
- Go 1.21+
- PostgreSQL (with pgAdmin for database management)
- Docker (optional, for production deployment only)

### 🚀 Quick Start

```bash
# 1. One-time setup
make setup

# 2. Create 'nimbus' database in PostgreSQL

# 3. Update backend/.env with your PostgreSQL password

# 4. Test database connection
make testdb

# 5. Start development servers
make dev-backend    # Terminal 1
make dev-frontend   # Terminal 2
```

**Access your app:**
- Frontend: http://localhost:3000
- Backend API: http://localhost:8080/api/v1/health

**Need help?** Run `make` or `make help` to see all available commands.

### 🐳 Production Deployment with Docker

**For production or testing the full stack:**

1. Clone the repository:
```bash
git clone https://github.com/yourusername/nimbus.git
cd nimbus
```

2. Copy environment variables:
```bash
cp .env.example .env
# Edit .env with your configuration
```

3. Start the stack:
```bash
docker-compose up -d --build
```

4. Access the application:
- Frontend: http://localhost:3000
- Backend API: http://localhost:8080/api/v1/health

### Database Migrations

The backend uses golang-migrate for database migrations:

```bash
# Create a new migration
make migrate-create name=create_users_table

# Run migrations
make migrate-up

# Rollback migrations
make migrate-down
```

## Documentation

- **[README.md](README.md)** - This file - project overview and quick reference
- **[QUICKSTART.md](QUICKSTART.md)** - 5-minute setup guide
- **[TOOLING.md](TOOLING.md)** - Why we use Makefiles and npm scripts
- **[CLAUDE.md](CLAUDE.md)** - Project guidelines and coding conventions
- **[DEPRECATED.md](DEPRECATED.md)** - Old scripts and migration guide

## Project Structure

```
nimbus/
├── Makefile                 # Root-level development commands
├── backend/
│   ├── cmd/
│   │   └── server/
│   │       └── main.go         # Application entry point
│   ├── internal/
│   │   ├── config/             # Configuration management
│   │   ├── db/                 # Database connection and migrations
│   │   ├── handlers/           # HTTP request handlers
│   │   ├── middleware/         # HTTP middleware (auth, CORS, etc.)
│   │   ├── models/             # Data models
│   │   ├── repository/         # Database operations
│   │   ├── services/           # Business logic
│   │   └── utils/              # Utility functions
│   ├── go.mod
│   └── Makefile
├── frontend/
│   ├── app/                    # Next.js app router pages
│   ├── components/             # React components
│   ├── hooks/                  # Custom React hooks
│   ├── lib/                    # Utility functions and API client
│   ├── types/                  # TypeScript type definitions
│   ├── public/                 # Static assets
│   └── package.json
├── docker/
│   ├── backend.Dockerfile
│   ├── frontend.Dockerfile
│   └── nginx/
│       └── nginx.conf
├── docker-compose.yml
└── README.md
```

## API Documentation

### Authentication Endpoints
- `POST /api/v1/auth/register` - Register new user
- `POST /api/v1/auth/login` - User login
- `POST /api/v1/auth/refresh` - Refresh JWT token
- `POST /api/v1/auth/logout` - User logout

### Service Management (Coming Soon)
- `GET /api/v1/services` - List all services
- `POST /api/v1/services` - Create new service (admin only)
- `PUT /api/v1/services/:id` - Update service (admin only)
- `DELETE /api/v1/services/:id` - Delete service (admin only)

### Health Monitoring (Coming Soon)
- `GET /api/v1/health/services` - Get service health statuses
- `GET /api/v1/health/services/:id` - Get specific service health history

## Environment Variables

See `.env.example` for all available configuration options. Key variables include:

- `DB_*` - Database connection settings
- `JWT_SECRET` - Secret key for JWT tokens (change in production!)
- `CORS_ORIGINS` - Allowed CORS origins
- `NEXT_PUBLIC_API_URL` - Backend API URL for frontend

## Development Commands

### Quick Reference

Run `make` or `make help` to see all available commands.

### Common Commands

```bash
# Setup (first time)
make setup          # Copy .env files, install dependencies

# Development
make dev-backend    # Start backend (or: cd backend && make dev)
make dev-frontend   # Start frontend (or: cd frontend && npm run dev)

# Testing
make testdb         # Test database connection

# Utilities
make kill-ports     # Kill stuck processes on ports 8080/3000
make clean          # Clean build artifacts
```

### Backend Commands

```bash
cd backend

make dev        # Run development server
make build      # Build production binary
make test       # Run tests
make testdb     # Test database connection
make fmt        # Format code
make lint       # Run linter
```

### Frontend Commands

```bash
cd frontend

npm run dev     # Start development server
npm run build   # Build for production
npm run start   # Start production server
npm run lint    # Run linter
```

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'feat: add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request (our PR template will guide you)

## Roadmap

- [x] Initial project setup
- [x] User authentication (JWT)
- [x] CI/CD pipeline (GitHub Actions)
- [x] Database migrations
- [x] Auth pages (login, register)
- [ ] Dashboard layout with sidebar
- [ ] Service management CRUD
- [ ] Health monitoring system
- [ ] User theme customization
- [ ] Role-based access control (admin features)
- [ ] Docker deployment
- [ ] Admin configuration UI
- [ ] Service status history graphs
- [ ] OAuth2 login support
- [ ] Widget/plugin system
- [ ] Mobile responsive design
- [ ] PWA support

## License

This project is licensed under the GNU Affero General License - see the LICENSE file for details.

## Acknowledgments

- Inspired by Dashy, Homarr, and Homer
- Built with modern web technologies
- Designed for the homelab community/general use
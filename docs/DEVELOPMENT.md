# Development Guide

Complete guide for setting up and developing Nimbus locally.

---

## Table of Contents

- [Prerequisites](#prerequisites)
- [Quick Setup](#quick-setup)
- [Project Structure](#project-structure)
- [Make Commands](#make-commands)
- [Database Setup](#database-setup)
- [Running the Application](#running-the-application)
- [Testing](#testing)
- [Code Quality](#code-quality)
- [Troubleshooting](#troubleshooting)

---

## Prerequisites

| Requirement | Version | Notes |
|-------------|---------|-------|
| **Node.js** | 20+ | For frontend development |
| **Go** | 1.21+ | For backend development |
| **PostgreSQL** | 15+ | Database (18 recommended) |
| **Make** | Any | Build automation |
| **Docker** | Optional | For production testing |

### Recommended Tools

- **pgAdmin** - PostgreSQL GUI management
- **VS Code** - With Go and TypeScript extensions
- **Postman** or **Insomnia** - API testing

---

## Quick Setup

```bash
# 1. Clone the repository
git clone https://github.com/Turbootzz/Nimbus.git
cd nimbus

# 2. Run setup (copies .env files, installs dependencies)
make setup

# 3. Create database in PostgreSQL
# Using pgAdmin or psql:
# CREATE DATABASE nimbus;

# 4. Configure environment
# Edit .env with your PostgreSQL credentials

# 5. Test database connection
make testdb

# 6. Run migrations
make migrate

# 7. Start development servers
make dev-backend    # Terminal 1
make dev-frontend   # Terminal 2
```

**Access your app:**
- Frontend: http://localhost:3000
- Backend API: http://localhost:8080/api/v1/health

---

## Project Structure

```
nimbus/
├── Makefile                 # Root-level commands
├── .env                     # Backend environment variables
├── .env.example             # Example configuration
│
├── backend/
│   ├── Makefile             # Backend-specific commands
│   ├── cmd/
│   │   ├── server/          # Main application entry
│   │   ├── migrate/         # Database migration runner
│   │   ├── seed/            # Test data seeder
│   │   └── testdb/          # Database connection tester
│   ├── internal/
│   │   ├── config/          # Configuration management
│   │   ├── db/              # Database & migrations
│   │   ├── handlers/        # HTTP request handlers
│   │   ├── middleware/      # Auth, CORS, logging
│   │   ├── models/          # Data models
│   │   ├── repository/      # Database operations
│   │   ├── services/        # Business logic
│   │   └── workers/         # Background tasks
│   └── go.mod
│
├── frontend/
│   ├── app/                 # Next.js App Router pages
│   ├── components/          # React components
│   ├── lib/                 # Utilities & API client
│   ├── types/               # TypeScript definitions
│   ├── .env.local           # Frontend environment
│   └── package.json
│
└── docs/
    ├── CONFIGURATION.md     # Environment variables
    └── DEVELOPMENT.md       # This file
```

---

## Make Commands

Run `make` or `make help` to see all available commands.

### Root Commands (from project root)

| Command | Description |
|---------|-------------|
| `make setup` | First-time setup (copy .env, install deps) |
| `make dev-backend` | Start backend development server |
| `make dev-frontend` | Start frontend development server |
| `make testdb` | Test database connection |
| `make migrate` | Run database migrations |
| `make migrate-down` | Rollback last migration |
| `make seed` | Seed database with test users |
| `make ci-check` | Run all CI checks locally |
| `make kill-ports` | Kill processes on 8080/3000 |
| `make clean` | Clean build artifacts |
| `make install` | Install all dependencies |

### Backend Commands (from `backend/`)

| Command | Description |
|---------|-------------|
| `make dev` | Run development server |
| `make build` | Build production binary |
| `make test` | Run all tests |
| `make testdb` | Test database connection |
| `make fmt` | Format Go code |
| `make fmt-check` | Check formatting (CI) |
| `make vet` | Run go vet |
| `make ci-check` | Run all backend CI checks |
| `make deps` | Download & tidy dependencies |
| `make migrate-up` | Run migrations |
| `make migrate-create name=xxx` | Create new migration |
| `make seed` | Seed test data |
| `make clean` | Remove build artifacts |

### Frontend Commands (from `frontend/`)

| Command | Description |
|---------|-------------|
| `npm run dev` | Start development server |
| `npm run build` | Build for production |
| `npm run start` | Start production server |
| `npm run lint` | Run ESLint |
| `npm run format` | Format with Prettier |
| `npm run ci-check` | Run all frontend CI checks |

---

## Database Setup

### Option 1: Using pgAdmin

1. Open pgAdmin
2. Right-click on Databases → Create → Database
3. Name: `nimbus`
4. Save

### Option 2: Using psql

```bash
psql -U postgres
CREATE DATABASE nimbus;
\q
```

### Configure Connection

Edit `.env` in the project root:

```bash
DB_HOST=localhost      # Use 'localhost' for local dev
DB_PORT=5432
DB_NAME=nimbus
DB_USER=postgres       # Your PostgreSQL user
DB_PASSWORD=password   # Your PostgreSQL password
```

### Verify Connection

```bash
make testdb
```

### Run Migrations

```bash
make migrate
```

Migrations run automatically when starting the backend server.

### Creating New Migrations

```bash
cd backend
make migrate-create name=add_user_preferences
```

This creates timestamped migration files in `backend/internal/db/migrations/`.

---

## Running the Application

### Development Mode

Start both servers in separate terminals:

```bash
# Terminal 1 - Backend (Go)
make dev-backend

# Terminal 2 - Frontend (Next.js)
make dev-frontend
```

### Alternative: Direct Commands

```bash
# Backend
cd backend && go run cmd/server/main.go

# Frontend
cd frontend && npm run dev
```

### Environment Files

| File | Purpose |
|------|---------|
| `.env` | Backend configuration (root directory) |
| `frontend/.env.local` | Frontend configuration |

**Important:** The `JWT_SECRET` must match in both files!

---

## Testing

### Run All Tests

```bash
# Full CI check (recommended before commits)
make ci-check
```

### Backend Tests Only

```bash
cd backend
make test           # Standard tests
make ci-check       # Tests with race detector
```

### Frontend Tests

```bash
cd frontend
npm run lint        # Linting
npm run build       # Type checking + build
```

### Test Database Connection

```bash
make testdb
```

---

## Code Quality

### Formatting

```bash
# Backend (Go)
cd backend && make fmt

# Frontend (TypeScript/JavaScript)
cd frontend && npm run format
```

### Linting

```bash
# Backend
cd backend && make vet

# Frontend
cd frontend && npm run lint
```

### Pre-commit Checks

Run before committing:

```bash
make ci-check
```

This runs:
- Go formatting check
- Go vet
- Go tests with race detector
- Frontend linting
- Frontend build

---

## Troubleshooting

### Port Already in Use

```bash
make kill-ports
```

Or manually:
```bash
lsof -ti:8080 | xargs kill -9
lsof -ti:3000 | xargs kill -9
```

### Database Connection Failed

1. Verify PostgreSQL is running
2. Check credentials in `.env`
3. Ensure database `nimbus` exists
4. Run `make testdb` to diagnose

### Module Not Found (Go)

```bash
cd backend
go mod download
go mod tidy
```

### Node Modules Issues

```bash
cd frontend
rm -rf node_modules package-lock.json
npm install
```

### Migrations Failed

Check the error message. Common issues:
- Database doesn't exist
- Wrong credentials
- Migration syntax error

To reset (development only):
```bash
# Drop and recreate database
psql -U postgres -c "DROP DATABASE nimbus;"
psql -U postgres -c "CREATE DATABASE nimbus;"
make migrate
```

### JWT Errors

Ensure `JWT_SECRET` matches in both:
- `.env` (backend)
- `frontend/.env.local`

Minimum 32 characters required.

---

## API Endpoints

### Authentication
| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/auth/register` | Register new user |
| POST | `/api/v1/auth/login` | User login |
| POST | `/api/v1/auth/logout` | User logout |
| GET | `/api/v1/auth/me` | Get current user |

### Services
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/services` | List all services |
| POST | `/api/v1/services` | Create service |
| GET | `/api/v1/services/:id` | Get service |
| PUT | `/api/v1/services/:id` | Update service |
| DELETE | `/api/v1/services/:id` | Delete service |
| PUT | `/api/v1/services/reorder` | Reorder services |
| POST | `/api/v1/services/:id/check` | Manual health check |

### Health
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/health` | API health check |

---

## Tips

1. **Hot Reload**: Both frontend and backend support hot reload in development
2. **Database GUI**: Use pgAdmin to inspect data during development
3. **API Testing**: Use the health endpoint to verify backend is running
4. **Logs**: Backend logs show in Terminal 1, frontend in Terminal 2
5. **VS Code**: Install Go and TypeScript extensions for best experience

---

## Next Steps

- [Configuration Guide](CONFIGURATION.md) - All environment variables
- [README](../README.md) - Project overview

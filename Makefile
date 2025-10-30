.PHONY: help setup dev-backend dev-frontend testdb migrate migrate-down seed kill-ports clean install

# Default target
help:
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo "Nimbus Development Commands"
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo ""
	@echo "Setup (first time):"
	@echo "  make setup        Setup project (copy .env, install dependencies)"
	@echo ""
	@echo "Development:"
	@echo "  make dev-backend  Start backend server (or: cd backend && make dev)"
	@echo "  make dev-frontend Start frontend server (or: cd frontend && npm run dev)"
	@echo ""
	@echo "Database:"
	@echo "  make testdb       Test database connection"
	@echo "  make migrate      Run database migrations (up)"
	@echo "  make migrate-down Rollback last migration"
	@echo "  make seed         Seed database with test users"
	@echo ""
	@echo "Formatting:"
	@echo "  cd backend && make fmt           Format Go code"
	@echo "  cd frontend && npm run format    Format frontend code with Prettier"
	@echo ""
	@echo "CI/CD:"
	@echo "  make ci-check     Run all CI checks locally (format, lint, tests, build)"
	@echo ""
	@echo "Utilities:"
	@echo "  make kill-ports   Kill processes on ports 8080 and 3000"
	@echo "  make clean        Clean build artifacts"
	@echo ""
	@echo "Typical workflow:"
	@echo "  1. make setup              # First time only"
	@echo "  2. Create 'nimbus' database in pgAdmin"
	@echo "  3. Update .env (root) with your PostgreSQL credentials"
	@echo "  4. make testdb             # Verify database connection"
	@echo "  5. make migrate            # Run database migrations"
	@echo "  6. make dev-backend        # Terminal 1"
	@echo "  7. make dev-frontend       # Terminal 2"
	@echo ""

# One-time setup
setup:
	@echo "🔧 Setting up Nimbus..."
	@echo ""
	@if [ ! -f .env ]; then \
		cp .env.example .env && echo "✓ Created .env"; \
	else \
		echo "ℹ .env already exists"; \
	fi
	@if [ ! -f frontend/.env.local ]; then \
		cp frontend/.env.local.example frontend/.env.local && echo "✓ Created frontend/.env.local"; \
	else \
		echo "ℹ frontend/.env.local already exists"; \
	fi
	@echo ""
	@echo "📦 Installing backend dependencies..."
	@cd backend && go mod download
	@echo "✓ Backend dependencies installed"
	@echo ""
	@echo "📦 Installing frontend dependencies..."
	@cd frontend && npm install
	@echo "✓ Frontend dependencies installed"
	@echo ""
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo "✅ Setup complete!"
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo ""
	@echo "⚠️  Next steps:"
	@echo "1. Create 'nimbus' database in pgAdmin"
	@echo "2. Update .env (root) with your PostgreSQL credentials"
	@echo "3. Update frontend/.env.local to match JWT_SECRET from .env"
	@echo "4. Run: make testdb"
	@echo ""

# Install dependencies only
install:
	@echo "📦 Installing dependencies..."
	@cd backend && go mod download
	@cd frontend && npm install
	@echo "✓ Done"

# Start backend
dev-backend:
	@cd backend && make dev

# Start frontend
dev-frontend:
	@cd frontend && npm run dev

# Test database connection
testdb:
	@cd backend && make testdb

# Run database migrations
migrate:
	@cd backend && make migrate-up

# Rollback database migrations
migrate-down:
	@cd backend && make migrate-down

# Seed database with test data
seed:
	@cd backend && make seed

# Kill stuck processes on development ports
kill-ports:
	@echo "🔍 Checking for processes on ports 8080 and 3000..."
	@-lsof -ti:8080 | xargs kill -9 2>/dev/null && echo "✓ Killed process on port 8080" || echo "✗ No process on port 8080"
	@-lsof -ti:3000 | xargs kill -9 2>/dev/null && echo "✓ Killed process on port 3000" || echo "✗ No process on port 3000"

# Clean build artifacts
clean:
	@echo "🧹 Cleaning build artifacts..."
	@cd backend && make clean
	@cd frontend && rm -rf .next node_modules/.cache
	@echo "✓ Done"

# Run all CI checks locally (same as GitHub Actions)
ci-check:
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo "🚀 Running CI Checks (Backend + Frontend)"
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo ""
	@echo "📦 Backend checks..."
	@cd backend && make ci-check
	@echo ""
	@echo "📦 Frontend checks..."
	@cd frontend && npm run ci-check
	@echo ""
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo "✅ All CI checks passed!"
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
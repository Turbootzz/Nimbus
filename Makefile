.PHONY: help setup dev-backend dev-frontend testdb kill-ports clean install

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
	@echo "Testing:"
	@echo "  make testdb       Test database connection"
	@echo ""
	@echo "Utilities:"
	@echo "  make kill-ports   Kill processes on ports 8080 and 3000"
	@echo "  make clean        Clean build artifacts"
	@echo ""
	@echo "Typical workflow:"
	@echo "  1. make setup              # First time only"
	@echo "  2. Create 'nimbus' database in pgAdmin"
	@echo "  3. Update backend/.env with your PostgreSQL credentials"
	@echo "  4. make testdb             # Verify database connection"
	@echo "  5. make dev-backend        # Terminal 1"
	@echo "  6. make dev-frontend       # Terminal 2"
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
	@if [ ! -f backend/.env ]; then \
		cp backend/.env.example backend/.env && echo "✓ Created backend/.env"; \
	else \
		echo "ℹ backend/.env already exists"; \
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
	@echo "2. Update backend/.env with your PostgreSQL password"
	@echo "3. Run: make testdb"
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
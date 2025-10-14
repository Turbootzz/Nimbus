.PHONY: dev build test clean migrate testdb fmt vet ci-check

# Development
dev:
	go run cmd/server/main.go

# Test database connection
testdb:
	go run cmd/testdb/main.go

# Seed database with test data (dev only - excluded from production builds)
seed: migrate-up
	@echo "🌱 Seeding database with test users..."
	@go run -tags dev cmd/seed/main.go

# Build
build:
	go build -o bin/nimbus cmd/server/main.go

# Test
test:
	go test -v ./...

# Clean
clean:
	rm -rf bin/

# Format
fmt:
	go fmt ./...

# Format check (fail if not formatted)
fmt-check:
	@echo "🔍 Checking Go formatting..."
	@if [ "$$(gofmt -s -l . | wc -l)" -gt 0 ]; then \
		echo "❌ Go code is not formatted. Run 'make fmt' to fix:"; \
		gofmt -s -l .; \
		exit 1; \
	fi
	@echo "✓ Go code is properly formatted"

# Go vet
vet:
	@echo "🔍 Running go vet..."
	@go vet ./...
	@echo "✓ go vet passed"

# Run all CI checks locally
ci-check: fmt-check vet
	@echo "🔍 Running tests with race detector..."
	@go test -v -race ./...
	@echo "✓ All CI checks passed!"

# Install dependencies
deps:
	go mod download
	go mod tidy

# Database migrations
# Load .env file and use DB_URL directly
migrate-up:
	@if [ -f .env ]; then \
		set -a; . ./.env; set +a; \
		migrate -path internal/db/migrations -database "$$DB_URL" up || exit 1; \
	else \
		echo "Error: .env file not found"; \
		exit 1; \
	fi

migrate-down:
	@if [ -f .env ]; then \
		set -a; . ./.env; set +a; \
		migrate -path internal/db/migrations -database "$$DB_URL" down || exit 1; \
	else \
		echo "Error: .env file not found"; \
		exit 1; \
	fi

migrate-create:
	migrate create -ext sql -dir internal/db/migrations -seq $(name)
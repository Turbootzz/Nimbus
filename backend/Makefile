.PHONY: dev build test clean migrate migrate-up migrate-down migrate-create testdb fmt vet ci-check

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
# Uses Go-based migration runner (same as dev server)
migrate: migrate-up

migrate-up:
	@go run cmd/migrate/main.go

migrate-down:
	@echo "⚠️  Note: migrate-down is not supported by the Go migration runner"
	@echo "Migrations run automatically on server start (cmd/server/main.go)"
	@echo "To rollback, modify/delete migration files and restart the server"

migrate-create:
	@echo "Creating new migration with name: $(name)"
	@migrate create -ext sql -dir internal/db/migrations -seq $(name)
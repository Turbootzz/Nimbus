.PHONY: dev build test clean migrate testdb

# Development
dev:
	go run cmd/server/main.go

# Test database connection
testdb:
	go run cmd/testdb/main.go

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

# Lint
lint:
	golangci-lint run

# Install dependencies
deps:
	go mod download
	go mod tidy

# Database migrations
migrate-up:
	migrate -path internal/db/migrations -database "postgresql://$$DB_URL" up

migrate-down:
	migrate -path internal/db/migrations -database "postgresql://$$DB_URL" down

migrate-create:
	migrate create -ext sql -dir internal/db/migrations -seq $(name)
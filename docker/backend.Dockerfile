# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum* ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o nimbus cmd/server/main.go

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN addgroup -g 1001 -S nimbus && \
    adduser -u 1001 -S nimbus -G nimbus

WORKDIR /home/nimbus

# Copy the binary from builder
COPY --from=builder /app/nimbus .

# Set ownership
RUN chown nimbus:nimbus /home/nimbus/nimbus

# Switch to non-root user
USER nimbus

# Expose port
EXPOSE 8080

# Run the binary
CMD ["./nimbus"]
# Nimbus Unified Docker Image
# Combines frontend, backend, and nginx into a single container
# Usage: docker build -t nimbus .

# =============================================================================
# Stage 1: Build Go backend
# =============================================================================
FROM golang:1.25-alpine AS backend-builder

WORKDIR /build

RUN apk add --no-cache git ca-certificates

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server ./cmd/server

# =============================================================================
# Stage 2: Build Next.js frontend
# =============================================================================
FROM node:24-alpine AS frontend-builder

WORKDIR /build

# Build arguments for optional customization
ARG NEXT_PUBLIC_APP_NAME
ARG NEXT_PUBLIC_SITE_URL
ENV NEXT_PUBLIC_APP_NAME=$NEXT_PUBLIC_APP_NAME
ENV NEXT_PUBLIC_SITE_URL=$NEXT_PUBLIC_SITE_URL
# Explicit same-origin mode for unified image (nginx proxy handles /api/*)
ENV NEXT_PUBLIC_API_URL="same-origin"

COPY frontend/package*.json ./
RUN npm ci

COPY frontend/ .
RUN npm run build

# =============================================================================
# Stage 3: Production runtime
# =============================================================================
FROM alpine:3.21

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache \
    nginx \
    nodejs \
    npm \
    supervisor \
    ca-certificates \
    tzdata \
    netcat-openbsd \
    wget \
    postgresql-client

# Create required directories
RUN mkdir -p /app/backend /app/frontend /app/backend/uploads/service-icons /app/backend/uploads/avatars \
    && mkdir -p /run/nginx /etc/supervisor/conf.d

# Copy backend binary and migrations
COPY --from=backend-builder /build/server /app/backend/server
COPY --from=backend-builder /build/internal/db/migrations /app/backend/internal/db/migrations

# Copy frontend build (only compiled assets and package files needed for production)
COPY --from=frontend-builder /build/.next /app/frontend/.next
COPY --from=frontend-builder /build/public /app/frontend/public
COPY --from=frontend-builder /build/package*.json /app/frontend/

# Install frontend production dependencies
WORKDIR /app/frontend
RUN npm ci --omit=dev && npm cache clean --force
WORKDIR /app

# Copy configuration files
COPY docker/nginx.conf /etc/nginx/nginx.conf
COPY docker/supervisord.conf /etc/supervisor/conf.d/supervisord.conf
COPY docker/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

EXPOSE 3000

ENV NODE_ENV=production

# Health check - verifies nginx can reach backend
HEALTHCHECK --interval=30s --timeout=10s --start-period=60s --retries=3 \
    CMD wget -q -O /dev/null http://localhost:3000/api/v1/health || exit 1

ENTRYPOINT ["/entrypoint.sh"]

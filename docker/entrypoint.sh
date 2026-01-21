#!/bin/sh
set -e

# Set default environment variables
export DB_HOST="${DB_HOST:-db}"
export DB_PORT="${DB_PORT:-5432}"
export DB_USER="${DB_USER:-nimbus}"
export DB_NAME="${DB_NAME:-nimbus}"
export PORT="${PORT:-8080}"

# Same-origin mode: empty NEXT_PUBLIC_API_URL means relative /api/v1 paths
export NEXT_PUBLIC_API_URL="${NEXT_PUBLIC_API_URL:-}"

# Wait for database
if [ "${WAIT_FOR_DB:-true}" = "true" ]; then
    echo "Waiting for database at ${DB_HOST}:${DB_PORT}..."
    timeout=30
    while [ $timeout -gt 0 ]; do
        if nc -z "${DB_HOST}" "${DB_PORT}" 2>/dev/null; then
            echo "Database is ready!"
            break
        fi
        timeout=$((timeout - 1))
        sleep 1
    done
    if [ $timeout -eq 0 ]; then
        echo "Database not available after 30s, starting anyway..."
    fi
fi

# Create upload directories
mkdir -p /app/backend/uploads/service-icons
mkdir -p /app/backend/uploads/avatars

echo "Starting Nimbus..."
exec /usr/bin/supervisord -c /etc/supervisor/conf.d/supervisord.conf

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

# Secrets file for persisting auto-generated secrets
SECRETS_DIR="/app/backend/uploads/.secrets"
SECRETS_FILE="${SECRETS_DIR}/generated.env"

# Create secrets directory with restrictive permissions
mkdir -p "${SECRETS_DIR}"
chmod 700 "${SECRETS_DIR}"

# Load previously generated secrets if they exist
if [ -f "${SECRETS_FILE}" ]; then
    . "${SECRETS_FILE}"
fi

# Auto-generate JWT_SECRET if not provided
if [ -z "${JWT_SECRET}" ]; then
    # Read extra bytes to ensure we have enough after base64 encoding and filtering
    export JWT_SECRET=$(head -c 64 /dev/urandom | base64 | tr -dc 'a-zA-Z0-9' | head -c 48)
    # Only append if not already in file (prevents duplicates on partial failures)
    if ! grep -q "^JWT_SECRET=" "${SECRETS_FILE}" 2>/dev/null; then
        echo "JWT_SECRET=${JWT_SECRET}" >> "${SECRETS_FILE}"
        chmod 600 "${SECRETS_FILE}"
    fi
    echo "INFO: JWT_SECRET auto-generated and saved for persistence"
fi

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

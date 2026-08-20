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

# Check for data loss (existing install but empty database)
SETUP_MARKER="${SECRETS_DIR}/.nimbus-initialized"
if [ -f "$SETUP_MARKER" ]; then
    # This is an existing installation - verify database has data
    echo "Checking database integrity..."

    # Skip check if DB_PASSWORD is not set
    if [ -z "${DB_PASSWORD}" ]; then
        echo "WARNING: DB_PASSWORD not set, skipping integrity check"
        USER_COUNT="skipped"
    else
        # Retry loop for database readiness (postgres may need time after connection is ready)
        USER_COUNT="error"
        PSQL_ERR=""
        PSQL_ERR_FILE=$(mktemp)
        for i in 1 2 3; do
            if PSQL_OUT=$(PGPASSWORD="${DB_PASSWORD}" psql -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d "${DB_NAME}" -tAc "SELECT COUNT(*) FROM users;" 2>"$PSQL_ERR_FILE"); then
                USER_COUNT="$PSQL_OUT"
                break
            fi
            PSQL_ERR=$(cat "$PSQL_ERR_FILE")
            sleep 2
        done
        rm -f "$PSQL_ERR_FILE"
    fi

    if [ "$USER_COUNT" = "error" ]; then
        echo ""
        echo "============================================================"
        echo "  ERROR: Cannot query the Nimbus database!"
        echo "============================================================"
        echo ""
        echo "  psql said: ${PSQL_ERR}"
        echo ""
        echo "  The database container is likely down or failing to start."
        echo "  Check its logs first: docker logs nimbus-db"
        echo ""
        echo "  If those logs show 'directory exists but is not empty',"
        echo "  pull the latest db image and restart - Nimbus migrates"
        echo "  your data automatically:"
        echo ""
        echo "    docker-compose pull db && docker-compose up -d"
        echo ""
        echo "  To skip this check: SKIP_DB_CHECK=true"
        echo "============================================================"
        echo ""

        if [ "${SKIP_DB_CHECK}" != "true" ]; then
            exit 1
        fi
    elif [ "$USER_COUNT" = "0" ]; then
        echo ""
        echo "============================================================"
        echo "  ERROR: Database appears empty but Nimbus was initialized!"
        echo "============================================================"
        echo ""
        echo "  This usually means your PostgreSQL is misconfigured."
        echo "  Your data may still be recoverable!"
        echo ""
        echo "  REQUIRED: Update your docker-compose.yml:"
        echo ""
        echo "    db:"
        echo "      image: turboot/nimbus-postgres:18   # NOT postgres:18-alpine"
        echo "      environment:"
        echo "        PGDATA: /var/lib/postgresql/data  # REQUIRED"
        echo ""
        echo "  After updating, restart: docker-compose down && docker-compose up -d"
        echo ""
        echo "  To skip this check: SKIP_DB_CHECK=true"
        echo "============================================================"
        echo ""

        if [ "${SKIP_DB_CHECK}" != "true" ]; then
            exit 1
        fi
    elif [ "$USER_COUNT" != "skipped" ]; then
        echo "Database OK ($USER_COUNT users found)"
    fi
else
    # First run - show requirements and create marker
    echo ""
    echo "============================================================"
    echo "  Nimbus First-Time Setup"
    echo "============================================================"
    echo ""
    echo "  Make sure your docker-compose.yml uses:"
    echo ""
    echo "    db:"
    echo "      image: turboot/nimbus-postgres:18"
    echo "      environment:"
    echo "        PGDATA: /var/lib/postgresql/data"
    echo ""
    echo "  Using postgres:18-alpine directly may cause data loss!"
    echo "============================================================"
    echo ""
    touch "$SETUP_MARKER"
fi

# Create upload directories
mkdir -p /app/backend/uploads/service-icons
mkdir -p /app/backend/uploads/avatars

echo "Starting Nimbus..."
exec /usr/bin/supervisord -c /etc/supervisor/conf.d/supervisord.conf

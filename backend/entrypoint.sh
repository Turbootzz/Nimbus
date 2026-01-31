#!/bin/sh
set -e

# Set default environment variables
DB_HOST="${DB_HOST:-db}"
DB_PORT="${DB_PORT:-5432}"
DB_USER="${DB_USER:-nimbus}"
DB_NAME="${DB_NAME:-nimbus}"

# Marker file location (in uploads volume for persistence)
SETUP_MARKER="/app/uploads/.nimbus-initialized"

# Wait for database
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

# Check for data loss (existing install but empty database)
if [ -f "$SETUP_MARKER" ]; then
    echo "Checking database integrity..."

    # Skip check if DB_PASSWORD is not set
    if [ -z "${DB_PASSWORD}" ]; then
        echo "WARNING: DB_PASSWORD not set, skipping integrity check"
        USER_COUNT="skipped"
    else
        # Retry loop for database readiness
        USER_COUNT="error"
        for i in 1 2 3; do
            USER_COUNT=$(PGPASSWORD="${DB_PASSWORD}" psql -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d "${DB_NAME}" -tAc "SELECT COUNT(*) FROM users;" 2>/dev/null || echo "error")
            [ "$USER_COUNT" != "error" ] && break
            sleep 2
        done
    fi

    if [ "$USER_COUNT" = "0" ] || [ "$USER_COUNT" = "error" ]; then
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

echo "Starting Nimbus backend..."
exec ./server

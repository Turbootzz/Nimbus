#!/bin/sh
set -e

# Auto-migrate data from legacy PostgreSQL 18 location if needed
if [ -f /var/lib/postgresql/18/docker/PG_VERSION ] && [ ! -f "$PGDATA/PG_VERSION" ]; then
    echo "Nimbus: Migrating PostgreSQL data from legacy location..."
    file_count=$(ls -1A /var/lib/postgresql/18/docker 2>/dev/null | wc -l)
    mv /var/lib/postgresql/18/docker/* "$PGDATA/" 2>/dev/null || true
    mv /var/lib/postgresql/18/docker/.[!.]* "$PGDATA/" 2>/dev/null || true
    rm -rf /var/lib/postgresql/18
    chown -R postgres:postgres "$PGDATA"
    echo "Nimbus: Migration complete! Moved $file_count items."
fi

# Remove problematic symlink if present
[ -L "$PGDATA/data" ] && rm "$PGDATA/data"

# Run original postgres entrypoint
exec docker-entrypoint.sh "$@"

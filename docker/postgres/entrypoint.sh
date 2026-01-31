#!/bin/sh
set -e

# Validate PGDATA is set to the correct path (not the postgres:18-alpine default)
EXPECTED_PGDATA="/var/lib/postgresql/data"
if [ "$PGDATA" != "$EXPECTED_PGDATA" ]; then
    echo ""
    echo "============================================================"
    echo "  Nimbus: docker-compose.yml update required!"
    echo "============================================================"
    echo ""
    echo "  Your docker-compose.yml is outdated. Please add PGDATA"
    echo "  to your db service environment:"
    echo ""
    echo "    db:"
    echo "      image: turboot/nimbus-postgres:18"
    echo "      environment:"
    echo "        PGDATA: /var/lib/postgresql/data   # <-- add this"
    echo "        ..."
    echo ""
    echo "  Current PGDATA: $PGDATA"
    echo "  Expected PGDATA: $EXPECTED_PGDATA"
    echo ""
    echo "  Then run: docker-compose up -d"
    echo ""
    echo "============================================================"
    echo ""
    exit 1
fi

# Ensure PGDATA directory exists
mkdir -p "$PGDATA"

LEGACY_DIR="/var/lib/postgresql/18/docker"

# Auto-migrate data from legacy PostgreSQL 18 location if needed
if [ -f "$LEGACY_DIR/PG_VERSION" ] && [ ! -f "$PGDATA/PG_VERSION" ]; then
    echo "Nimbus: Migrating PostgreSQL data from legacy location..."
    src_count=$(ls -1A "$LEGACY_DIR" | wc -l)
    initial_dest_count=$(ls -1A "$PGDATA" | wc -l)

    # Move regular files/dirs
    if ! mv "$LEGACY_DIR"/* "$PGDATA/"; then
        echo "Nimbus: ERROR - Failed to move files from $LEGACY_DIR"
        exit 1
    fi

    # Move hidden files (may not exist, so check first)
    for f in "$LEGACY_DIR"/.[!.]*; do
        if [ -e "$f" ]; then
            mv "$f" "$PGDATA/" || { echo "Nimbus: ERROR - Failed to move $f"; exit 1; }
        fi
    done

    # Verify migration succeeded by checking delta
    dest_count=$(ls -1A "$PGDATA" | wc -l)
    moved_count=$((dest_count - initial_dest_count))
    if [ "$moved_count" -lt "$src_count" ]; then
        echo "Nimbus: ERROR - Migration verification failed (expected $src_count items, moved $moved_count)"
        exit 1
    fi

    # Safe to remove source now
    rm -rf /var/lib/postgresql/18
    chown -R postgres:postgres "$PGDATA"
    echo "Nimbus: Migration complete! Moved $src_count items."
fi

# Remove problematic symlink if present
[ -L "$PGDATA/data" ] && rm "$PGDATA/data"

# Run original postgres entrypoint
exec docker-entrypoint.sh "$@"

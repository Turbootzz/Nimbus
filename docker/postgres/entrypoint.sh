#!/bin/sh
set -e

# Overridable in tests (NIMBUS_ENTRYPOINT_TEST); defaults match the real container
NIMBUS_PG_ROOT="${NIMBUS_PG_ROOT:-/var/lib/postgresql}"
EXPECTED_PGDATA="/var/lib/postgresql/data"

# Validate PGDATA is set to the correct path (not the postgres:18-alpine default)
nimbus_validate_pgdata() {
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
}

# Detect a cluster left behind by older image layouts and move it to $PGDATA.
# Known legacy locations:
# 1. <pg root>/18/docker           - postgres:18 default PGDATA, outside volume
# 2. $PGDATA/18/docker             - same, but nested inside the volume
# 3. $PGDATA/data                  - created by old postgres:18 images that shipped
#                                    a /var/lib/postgresql/data -> . compat symlink;
#                                    mounting the volume there landed it on the parent,
#                                    so the cluster was initialized one level deeper
nimbus_migrate_legacy_data() {
    mkdir -p "$PGDATA"

    # Remove stale compat symlink (data -> .) copied in from old postgres:18 images
    [ -L "$PGDATA/data" ] && rm "$PGDATA/data"

    LEGACY_DIR=""
    if [ -f "$NIMBUS_PG_ROOT/18/docker/PG_VERSION" ]; then
        LEGACY_DIR="$NIMBUS_PG_ROOT/18/docker"
    elif [ -f "$PGDATA/18/docker/PG_VERSION" ]; then
        LEGACY_DIR="$PGDATA/18/docker"
    elif [ -f "$PGDATA/data/PG_VERSION" ]; then
        LEGACY_DIR="$PGDATA/data"
    fi

    if [ -n "$LEGACY_DIR" ] && [ ! -f "$PGDATA/PG_VERSION" ]; then
        echo "Nimbus: Migrating PostgreSQL data from legacy location ($LEGACY_DIR)..."

        # Refuse to merge into existing entries - a collision means $PGDATA
        # holds leftovers of another (partial) cluster; moving would interleave them
        for entry in "$LEGACY_DIR"/* "$LEGACY_DIR"/.[!.]*; do
            [ -e "$entry" ] || continue
            name=$(basename "$entry")
            [ "$PGDATA/$name" = "$LEGACY_DIR" ] && continue
            if [ -e "$PGDATA/$name" ]; then
                echo "Nimbus: ERROR - Cannot migrate: $PGDATA already contains '$name'"
                echo "Nimbus: Resolve manually, nothing was moved."
                exit 1
            fi
        done

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

        # Safe to remove the emptied legacy dir now; only remove its parent
        # (the 18/ dir) when nothing else lives there
        rm -rf "$LEGACY_DIR"
        case "$LEGACY_DIR" in
            */18/docker) rmdir "$(dirname "$LEGACY_DIR")" 2>/dev/null || true ;;
        esac
        if [ "$(id -u)" = "0" ]; then
            chown -R postgres:postgres "$PGDATA"
        fi
        echo "Nimbus: Migration complete! Moved $src_count items."
    elif [ -f "$PGDATA/PG_VERSION" ] && [ -f "$PGDATA/data/PG_VERSION" ]; then
        echo "Nimbus: WARNING - Found clusters at both \$PGDATA and \$PGDATA/data; using \$PGDATA."
    fi
}

# Skip execution when sourced by tests
if [ -z "${NIMBUS_ENTRYPOINT_TEST:-}" ]; then
    nimbus_validate_pgdata
    nimbus_migrate_legacy_data

    # Run original postgres entrypoint
    exec docker-entrypoint.sh "$@"
fi

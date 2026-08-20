#!/bin/sh
# Tests for docker/postgres/entrypoint.sh legacy data migration logic
# Run with: sh docker/postgres/entrypoint.test.sh

TESTS_PASSED=0
TESTS_FAILED=0

RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
ENTRYPOINT="${SCRIPT_DIR}/entrypoint.sh"

pass() {
    TESTS_PASSED=$((TESTS_PASSED + 1))
    printf "${GREEN}✓ PASS${NC}: %s\n" "$1"
}

fail() {
    TESTS_FAILED=$((TESTS_FAILED + 1))
    printf "${RED}✗ FAIL${NC}: %s\n" "$1"
}

# Helper: set up a fresh fake volume dir and source the entrypoint functions
setup() {
    TEST_DIR=$(mktemp -d)
    PG_ROOT="${TEST_DIR}/var-lib-postgresql"
    PGDATA="${PG_ROOT}/data"
    mkdir -p "$PG_ROOT"
    export PGDATA
    export NIMBUS_ENTRYPOINT_TEST=1
    export NIMBUS_PG_ROOT="$PG_ROOT"
    # shellcheck disable=SC1090
    . "$ENTRYPOINT"
    set +e  # sourcing the entrypoint enables set -e; tests need failures to be observable
}

teardown() {
    rm -rf "$TEST_DIR"
}

# Helper: create a minimal fake postgres cluster in a directory
make_cluster() {
    mkdir -p "$1/base" "$1/global"
    echo "18" > "$1/PG_VERSION"
    echo "# config" > "$1/postgresql.conf"
    touch "$1/.hidden_marker"
}

# =============================================================================
# Test 1: Cluster nested at $PGDATA/data (postgres:18 compat-symlink era)
# is migrated up to $PGDATA
# =============================================================================
test_migrates_nested_data_dir() {
    setup
    make_cluster "$PGDATA/data"

    nimbus_migrate_legacy_data

    if [ -f "$PGDATA/PG_VERSION" ] && [ -d "$PGDATA/base" ] && [ ! -d "$PGDATA/data" ]; then
        pass "cluster nested at \$PGDATA/data is migrated to \$PGDATA"
    else
        fail "cluster nested at \$PGDATA/data should be migrated to \$PGDATA"
    fi
    teardown
}

# =============================================================================
# Test 2: Hidden files survive the nested-data migration
# =============================================================================
test_migrates_hidden_files() {
    setup
    make_cluster "$PGDATA/data"

    nimbus_migrate_legacy_data

    if [ -f "$PGDATA/.hidden_marker" ]; then
        pass "hidden files are migrated from \$PGDATA/data"
    else
        fail "hidden files should be migrated from \$PGDATA/data"
    fi
    teardown
}

# =============================================================================
# Test 3: Cluster at $PGDATA/18/docker is migrated (existing behavior)
# =============================================================================
test_migrates_nested_18_docker() {
    setup
    make_cluster "$PGDATA/18/docker"

    nimbus_migrate_legacy_data

    if [ -f "$PGDATA/PG_VERSION" ] && [ ! -d "$PGDATA/18" ]; then
        pass "cluster at \$PGDATA/18/docker is migrated to \$PGDATA"
    else
        fail "cluster at \$PGDATA/18/docker should be migrated to \$PGDATA"
    fi
    teardown
}

# =============================================================================
# Test 4: Cluster at <pg root>/18/docker (outside volume) is migrated
# =============================================================================
test_migrates_root_18_docker() {
    setup
    mkdir -p "$PGDATA"
    make_cluster "$PG_ROOT/18/docker"

    nimbus_migrate_legacy_data

    if [ -f "$PGDATA/PG_VERSION" ] && [ ! -d "$PG_ROOT/18" ]; then
        pass "cluster at <pg root>/18/docker is migrated to \$PGDATA"
    else
        fail "cluster at <pg root>/18/docker should be migrated to \$PGDATA"
    fi
    teardown
}

# =============================================================================
# Test 5: Existing cluster at $PGDATA root is left untouched
# =============================================================================
test_existing_root_cluster_untouched() {
    setup
    make_cluster "$PGDATA"
    make_cluster "$PGDATA/data"

    nimbus_migrate_legacy_data

    if [ -f "$PGDATA/PG_VERSION" ] && [ -f "$PGDATA/data/PG_VERSION" ]; then
        pass "existing root cluster is not overwritten by nested data"
    else
        fail "existing root cluster should be left untouched"
    fi
    teardown
}

# =============================================================================
# Test 6: Stale compat symlink $PGDATA/data -> . is removed
# =============================================================================
test_removes_stale_symlink() {
    setup
    make_cluster "$PGDATA"
    ln -s . "$PGDATA/data"

    nimbus_migrate_legacy_data

    if [ ! -L "$PGDATA/data" ] && [ -f "$PGDATA/PG_VERSION" ]; then
        pass "stale \$PGDATA/data symlink is removed, cluster intact"
    else
        fail "stale \$PGDATA/data symlink should be removed without touching cluster"
    fi
    teardown
}

# =============================================================================
# Test 7: Fresh empty PGDATA does nothing and does not error
# =============================================================================
test_fresh_empty_pgdata() {
    setup

    if nimbus_migrate_legacy_data && [ -d "$PGDATA" ] && [ ! -f "$PGDATA/PG_VERSION" ]; then
        pass "fresh empty PGDATA is a no-op"
    else
        fail "fresh empty PGDATA should be a no-op"
    fi
    teardown
}

# =============================================================================
# Test 8: Migration aborts on destination collision without moving anything
# =============================================================================
test_collision_aborts_migration() {
    setup
    make_cluster "$PGDATA/data"
    mkdir -p "$PGDATA/base"
    echo "keep" > "$PGDATA/base/leftover"

    RESULT=$( (nimbus_migrate_legacy_data) 2>&1 )
    STATUS=$?

    if [ "$STATUS" -ne 0 ] && [ -f "$PGDATA/data/PG_VERSION" ] && [ -f "$PGDATA/base/leftover" ] && [ ! -f "$PGDATA/PG_VERSION" ]; then
        pass "destination collision aborts migration, nothing moved"
    else
        fail "destination collision should abort migration untouched (status=$STATUS)"
    fi
    teardown
}

# =============================================================================
# Test 9: Cleanup keeps siblings next to 18/docker
# =============================================================================
test_cleanup_preserves_18_siblings() {
    setup
    make_cluster "$PGDATA/18/docker"
    echo "keep" > "$PGDATA/18/sibling"

    nimbus_migrate_legacy_data

    if [ -f "$PGDATA/PG_VERSION" ] && [ ! -d "$PGDATA/18/docker" ] && [ -f "$PGDATA/18/sibling" ]; then
        pass "cleanup preserves sibling files under 18/"
    else
        fail "cleanup should preserve sibling files under 18/"
    fi
    teardown
}

# =============================================================================
# Test 10: Dot-dot-prefixed entries and dangling symlinks are migrated
# =============================================================================
test_migrates_odd_entries() {
    setup
    make_cluster "$PGDATA/data"
    echo "keep" > "$PGDATA/data/..legacy_marker"
    ln -s /nonexistent "$PGDATA/data/dangling"

    nimbus_migrate_legacy_data

    if [ -f "$PGDATA/..legacy_marker" ] && [ -L "$PGDATA/dangling" ] && [ ! -d "$PGDATA/data" ]; then
        pass "..-prefixed entries and dangling symlinks are migrated"
    else
        fail "..-prefixed entries and dangling symlinks should be migrated"
    fi
    teardown
}

# =============================================================================
# Test 11: Dangling destination symlink triggers collision abort
# =============================================================================
test_dangling_dest_symlink_collides() {
    setup
    make_cluster "$PGDATA/data"
    ln -s /nonexistent "$PGDATA/base"

    RESULT=$( (nimbus_migrate_legacy_data) 2>&1 )
    STATUS=$?

    if [ "$STATUS" -ne 0 ] && [ -f "$PGDATA/data/PG_VERSION" ]; then
        pass "dangling destination symlink aborts migration"
    else
        fail "dangling destination symlink should abort migration (status=$STATUS)"
    fi
    teardown
}

# =============================================================================
# Test 12: PGDATA validation rejects wrong path
# =============================================================================
test_pgdata_validation() {
    setup
    RESULT=$(PGDATA="/wrong/path" sh -c ". '$ENTRYPOINT' && nimbus_validate_pgdata" 2>&1)
    STATUS=$?

    if [ "$STATUS" -ne 0 ] && echo "$RESULT" | grep -q "PGDATA"; then
        pass "wrong PGDATA is rejected with an explanatory message"
    else
        fail "wrong PGDATA should be rejected (status=$STATUS)"
    fi
    teardown
}

# =============================================================================
# Run all tests
# =============================================================================
echo "Running docker/postgres/entrypoint.sh tests..."
echo "================================"

test_migrates_nested_data_dir
test_migrates_hidden_files
test_migrates_nested_18_docker
test_migrates_root_18_docker
test_existing_root_cluster_untouched
test_removes_stale_symlink
test_fresh_empty_pgdata
test_collision_aborts_migration
test_cleanup_preserves_18_siblings
test_migrates_odd_entries
test_dangling_dest_symlink_collides
test_pgdata_validation

echo "================================"
echo "Results: ${TESTS_PASSED} passed, ${TESTS_FAILED} failed"

if [ "${TESTS_FAILED}" -gt 0 ]; then
    exit 1
fi

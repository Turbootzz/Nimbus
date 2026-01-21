#!/bin/sh
# Tests for entrypoint.sh JWT auto-generation logic
# Run with: sh docker/entrypoint.test.sh

set -e

TESTS_PASSED=0
TESTS_FAILED=0

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m' # No Color

# Setup test environment
TEST_DIR=$(mktemp -d)
SECRETS_DIR="${TEST_DIR}/.secrets"
SECRETS_FILE="${SECRETS_DIR}/generated.env"

cleanup() {
    rm -rf "${TEST_DIR}"
}
trap cleanup EXIT

pass() {
    TESTS_PASSED=$((TESTS_PASSED + 1))
    printf "${GREEN}✓ PASS${NC}: %s\n" "$1"
}

fail() {
    TESTS_FAILED=$((TESTS_FAILED + 1))
    printf "${RED}✗ FAIL${NC}: %s\n" "$1"
}

# Helper: simulate the JWT generation logic from entrypoint.sh
run_jwt_logic() {
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
    fi
}

# =============================================================================
# Test 1: JWT_SECRET is generated when not provided
# =============================================================================
test_jwt_generated_when_empty() {
    unset JWT_SECRET
    rm -rf "${SECRETS_DIR}"

    run_jwt_logic

    if [ -n "${JWT_SECRET}" ]; then
        pass "JWT_SECRET is generated when not provided"
    else
        fail "JWT_SECRET should be generated when not provided"
    fi
}

# =============================================================================
# Test 2: JWT_SECRET is persisted to file
# =============================================================================
test_jwt_persisted_to_file() {
    unset JWT_SECRET
    rm -rf "${SECRETS_DIR}"

    run_jwt_logic

    if [ -f "${SECRETS_FILE}" ] && grep -q "^JWT_SECRET=" "${SECRETS_FILE}"; then
        pass "JWT_SECRET is persisted to secrets file"
    else
        fail "JWT_SECRET should be persisted to secrets file"
    fi
}

# =============================================================================
# Test 3: JWT_SECRET is loaded from existing file
# =============================================================================
test_jwt_loaded_from_file() {
    unset JWT_SECRET
    rm -rf "${SECRETS_DIR}"
    mkdir -p "${SECRETS_DIR}"
    echo "JWT_SECRET=test-secret-from-file-12345678901234567890" > "${SECRETS_FILE}"

    run_jwt_logic

    if [ "${JWT_SECRET}" = "test-secret-from-file-12345678901234567890" ]; then
        pass "JWT_SECRET is loaded from existing file"
    else
        fail "JWT_SECRET should be loaded from existing file (got: ${JWT_SECRET})"
    fi
}

# =============================================================================
# Test 4: Provided JWT_SECRET is not overwritten
# =============================================================================
test_provided_jwt_not_overwritten() {
    export JWT_SECRET="user-provided-secret-key-1234567890123456"
    rm -rf "${SECRETS_DIR}"

    run_jwt_logic

    if [ "${JWT_SECRET}" = "user-provided-secret-key-1234567890123456" ]; then
        pass "Provided JWT_SECRET is not overwritten"
    else
        fail "Provided JWT_SECRET should not be overwritten"
    fi
}

# =============================================================================
# Test 5: No duplicate entries on multiple runs
# =============================================================================
test_no_duplicate_entries() {
    unset JWT_SECRET
    rm -rf "${SECRETS_DIR}"

    # Run twice
    run_jwt_logic
    FIRST_SECRET="${JWT_SECRET}"

    # Simulate partial failure: unset JWT_SECRET but keep file
    unset JWT_SECRET
    run_jwt_logic

    COUNT=$(grep -c "^JWT_SECRET=" "${SECRETS_FILE}" 2>/dev/null || echo "0")

    if [ "${COUNT}" = "1" ]; then
        pass "No duplicate JWT_SECRET entries after multiple runs"
    else
        fail "Should have exactly 1 JWT_SECRET entry, found ${COUNT}"
    fi
}

# =============================================================================
# Test 6: Secrets directory has correct permissions (700)
# =============================================================================
test_secrets_dir_permissions() {
    unset JWT_SECRET
    rm -rf "${SECRETS_DIR}"

    run_jwt_logic

    # Get permissions (works on both Linux and macOS)
    if [ "$(uname)" = "Darwin" ]; then
        PERMS=$(stat -f "%Lp" "${SECRETS_DIR}")
    else
        PERMS=$(stat -c "%a" "${SECRETS_DIR}")
    fi

    if [ "${PERMS}" = "700" ]; then
        pass "Secrets directory has correct permissions (700)"
    else
        fail "Secrets directory should have 700 permissions, got ${PERMS}"
    fi
}

# =============================================================================
# Test 7: Secrets file has correct permissions (600)
# =============================================================================
test_secrets_file_permissions() {
    unset JWT_SECRET
    rm -rf "${SECRETS_DIR}"

    run_jwt_logic

    # Get permissions (works on both Linux and macOS)
    if [ "$(uname)" = "Darwin" ]; then
        PERMS=$(stat -f "%Lp" "${SECRETS_FILE}")
    else
        PERMS=$(stat -c "%a" "${SECRETS_FILE}")
    fi

    if [ "${PERMS}" = "600" ]; then
        pass "Secrets file has correct permissions (600)"
    else
        fail "Secrets file should have 600 permissions, got ${PERMS}"
    fi
}

# =============================================================================
# Test 8: Generated JWT_SECRET has correct length (48 chars)
# =============================================================================
test_jwt_length() {
    unset JWT_SECRET
    rm -rf "${SECRETS_DIR}"

    run_jwt_logic

    LEN=${#JWT_SECRET}

    if [ "${LEN}" -eq 48 ]; then
        pass "Generated JWT_SECRET has correct length (48 chars)"
    else
        fail "Generated JWT_SECRET should be 48 chars, got ${LEN}"
    fi
}

# =============================================================================
# Test 9: Generated JWT_SECRET is alphanumeric only
# =============================================================================
test_jwt_alphanumeric() {
    unset JWT_SECRET
    rm -rf "${SECRETS_DIR}"

    run_jwt_logic

    # Check if JWT_SECRET contains only alphanumeric characters
    if echo "${JWT_SECRET}" | grep -q "^[a-zA-Z0-9]*$"; then
        pass "Generated JWT_SECRET is alphanumeric only"
    else
        fail "Generated JWT_SECRET should be alphanumeric only"
    fi
}

# =============================================================================
# Run all tests
# =============================================================================
echo "Running entrypoint.sh tests..."
echo "================================"

test_jwt_generated_when_empty
test_jwt_persisted_to_file
test_jwt_loaded_from_file
test_provided_jwt_not_overwritten
test_no_duplicate_entries
test_secrets_dir_permissions
test_secrets_file_permissions
test_jwt_length
test_jwt_alphanumeric

echo "================================"
echo "Results: ${TESTS_PASSED} passed, ${TESTS_FAILED} failed"

if [ "${TESTS_FAILED}" -gt 0 ]; then
    exit 1
fi

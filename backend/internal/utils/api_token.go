package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// APITokenPrefix marks bearer tokens as personal access tokens (vs JWTs)
const APITokenPrefix = "nimbus_"

// GenerateAPIToken returns a new plaintext token, its SHA-256 hex hash,
// and the 12-char display prefix stored for identification in the UI.
func GenerateAPIToken() (token, hash, prefix string, err error) {
	raw := make([]byte, 32)
	if _, err = rand.Read(raw); err != nil {
		return "", "", "", fmt.Errorf("failed to generate token: %w", err)
	}
	token = APITokenPrefix + hex.EncodeToString(raw)
	return token, HashAPIToken(token), token[:12], nil
}

// HashAPIToken returns the SHA-256 hex digest of a token
func HashAPIToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// IsAPIToken reports whether a bearer token is a personal access token
func IsAPIToken(token string) bool {
	return strings.HasPrefix(token, APITokenPrefix)
}

package utils

import (
	"strings"
	"testing"
)

func TestGenerateAPIToken(t *testing.T) {
	token, hash, prefix, err := GenerateAPIToken()
	if err != nil {
		t.Fatalf("GenerateAPIToken returned error: %v", err)
	}

	if !strings.HasPrefix(token, APITokenPrefix) {
		t.Errorf("token %q does not start with %q", token, APITokenPrefix)
	}

	// "nimbus_" + 64 hex chars (32 random bytes)
	expectedLen := len(APITokenPrefix) + 64
	if len(token) != expectedLen {
		t.Errorf("token length = %d, want %d", len(token), expectedLen)
	}

	if hash != HashAPIToken(token) {
		t.Errorf("hash %q does not match HashAPIToken(token) %q", hash, HashAPIToken(token))
	}
	if len(hash) != 64 {
		t.Errorf("hash length = %d, want 64", len(hash))
	}

	if prefix != token[:12] {
		t.Errorf("prefix %q is not the first 12 chars of token", prefix)
	}
}

func TestGenerateAPIToken_Unique(t *testing.T) {
	token1, _, _, err := GenerateAPIToken()
	if err != nil {
		t.Fatalf("GenerateAPIToken returned error: %v", err)
	}
	token2, _, _, err := GenerateAPIToken()
	if err != nil {
		t.Fatalf("GenerateAPIToken returned error: %v", err)
	}
	if token1 == token2 {
		t.Error("two generated tokens are identical")
	}
}

func TestIsAPIToken(t *testing.T) {
	if !IsAPIToken("nimbus_abc123") {
		t.Error("expected nimbus_-prefixed token to be recognized as API token")
	}
	if IsAPIToken("eyJhbGciOiJIUzI1NiJ9.payload.sig") {
		t.Error("JWT should not be recognized as API token")
	}
	if IsAPIToken("") {
		t.Error("empty string should not be recognized as API token")
	}
}

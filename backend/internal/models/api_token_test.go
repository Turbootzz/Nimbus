package models

import (
	"testing"
	"time"
)

func TestAPIToken_Allows(t *testing.T) {
	tests := []struct {
		name      string
		scopes    []string
		readScope string
		write     bool
		want      bool
	}{
		{"read scope reads", []string{ScopeReadServices}, ScopeReadServices, false, true},
		{"read scope does not write", []string{ScopeReadServices}, ScopeReadServices, true, false},
		{"wrong area", []string{ScopeReadMetrics}, ScopeReadServices, false, false},
		{"write implies read:services", []string{ScopeWriteServices}, ScopeReadServices, false, true},
		{"write writes services", []string{ScopeWriteServices}, ScopeReadServices, true, true},
		{"write does not imply read:widgets", []string{ScopeWriteServices}, ScopeReadWidgets, false, false},
		{"write never reaches widgets", []string{ScopeWriteServices, ScopeReadWidgets}, ScopeReadWidgets, true, false},
		{"write never reaches metrics", APITokenScopes, ScopeReadMetrics, true, false},
		{"mcp grants no REST access", []string{ScopeMCP}, ScopeReadServices, false, false},
		{"closed area", APITokenScopes, "", false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := &APIToken{Scopes: tt.scopes}
			if got := token.Allows(tt.readScope, tt.write); got != tt.want {
				t.Errorf("Allows(%q, %v) = %v, want %v", tt.readScope, tt.write, got, tt.want)
			}
		})
	}
}

func TestAPIToken_Expired(t *testing.T) {
	now := time.Now()
	past, future := now.Add(-time.Second), now.Add(time.Second)

	if (&APIToken{}).Expired(now) {
		t.Error("a token without expiry never expires")
	}
	if !(&APIToken{ExpiresAt: &past}).Expired(now) {
		t.Error("expected a past expiry to be expired")
	}
	if !(&APIToken{ExpiresAt: &now}).Expired(now) {
		t.Error("expected the expiry instant itself to be expired")
	}
	if (&APIToken{ExpiresAt: &future}).Expired(now) {
		t.Error("expected a future expiry to be valid")
	}
}

func TestLegacyScopes(t *testing.T) {
	for _, readOnly := range []bool{true, false} {
		token := &APIToken{Scopes: LegacyScopes(readOnly)}
		if token.HasScope(ScopeMCP) {
			t.Errorf("legacy tokens never get mcp (read_only=%v)", readOnly)
		}
		if token.HasScope(ScopeWriteServices) == readOnly {
			t.Errorf("write scope mismatch for read_only=%v", readOnly)
		}
		for _, scope := range ReadScopes() {
			if !token.HasScope(scope) {
				t.Errorf("legacy token misses %s (read_only=%v)", scope, readOnly)
			}
		}
	}
}

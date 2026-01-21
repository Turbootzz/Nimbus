package utils

import (
	"os"
	"testing"
)

func TestGetCookieConfig_Defaults(t *testing.T) {
	// Clear env vars
	os.Unsetenv("COOKIE_SECURE")
	os.Unsetenv("COOKIE_DOMAIN")

	config := GetCookieConfig()

	if !config.Secure {
		t.Error("GetCookieConfig() Secure should default to true when COOKIE_SECURE is unset")
	}
	if config.Domain != "" {
		t.Errorf("GetCookieConfig() Domain expected empty string, got %q", config.Domain)
	}
}

func TestGetCookieConfig_SecureFalse(t *testing.T) {
	os.Setenv("COOKIE_SECURE", "false")
	defer os.Unsetenv("COOKIE_SECURE")

	config := GetCookieConfig()

	if config.Secure {
		t.Error("GetCookieConfig() Secure should be false when COOKIE_SECURE=false")
	}
}

func TestGetCookieConfig_SecureTrue(t *testing.T) {
	os.Setenv("COOKIE_SECURE", "true")
	defer os.Unsetenv("COOKIE_SECURE")

	config := GetCookieConfig()

	if !config.Secure {
		t.Error("GetCookieConfig() Secure should be true when COOKIE_SECURE=true")
	}
}

func TestGetCookieConfig_Domain(t *testing.T) {
	os.Setenv("COOKIE_DOMAIN", ".example.com")
	defer os.Unsetenv("COOKIE_DOMAIN")

	config := GetCookieConfig()

	if config.Domain != ".example.com" {
		t.Errorf("GetCookieConfig() Domain expected '.example.com', got %q", config.Domain)
	}
}

func TestNewAuthCookie(t *testing.T) {
	config := CookieConfig{Secure: true, Domain: ".example.com"}
	cookie := NewAuthCookie("test-token", 3600, config)

	if cookie.Name != "auth_token" {
		t.Errorf("Cookie Name expected 'auth_token', got %q", cookie.Name)
	}
	if cookie.Value != "test-token" {
		t.Errorf("Cookie Value expected 'test-token', got %q", cookie.Value)
	}
	if cookie.Path != "/" {
		t.Errorf("Cookie Path expected '/', got %q", cookie.Path)
	}
	if cookie.Domain != ".example.com" {
		t.Errorf("Cookie Domain expected '.example.com', got %q", cookie.Domain)
	}
	if !cookie.HTTPOnly {
		t.Error("Cookie HTTPOnly expected true, got false")
	}
	if !cookie.Secure {
		t.Error("Cookie Secure expected true, got false")
	}
	if cookie.SameSite != "Lax" {
		t.Errorf("Cookie SameSite expected 'Lax', got %q", cookie.SameSite)
	}
	if cookie.MaxAge != 3600 {
		t.Errorf("Cookie MaxAge expected 3600, got %d", cookie.MaxAge)
	}
}

func TestNewAuthCookie_SessionCookie(t *testing.T) {
	config := CookieConfig{Secure: false, Domain: ""}
	cookie := NewAuthCookie("session-token", 0, config)

	if cookie.MaxAge != 0 {
		t.Errorf("Session cookie MaxAge expected 0, got %d", cookie.MaxAge)
	}
	if cookie.Secure {
		t.Error("Cookie Secure expected false for dev config")
	}
}

func TestClearAuthCookie(t *testing.T) {
	config := CookieConfig{Secure: true, Domain: ".example.com"}
	cookie := ClearAuthCookie(config)

	if cookie.Name != "auth_token" {
		t.Errorf("Cookie Name expected 'auth_token', got %q", cookie.Name)
	}
	if cookie.Value != "" {
		t.Errorf("Cookie Value expected empty string, got %q", cookie.Value)
	}
	if cookie.MaxAge != -1 {
		t.Errorf("Cookie MaxAge expected -1 for deletion, got %d", cookie.MaxAge)
	}
	if cookie.Domain != ".example.com" {
		t.Errorf("Cookie Domain expected '.example.com', got %q", cookie.Domain)
	}
	if !cookie.HTTPOnly {
		t.Error("Cookie HTTPOnly expected true, got false")
	}
	if !cookie.Secure {
		t.Error("Cookie Secure expected true, got false")
	}
}

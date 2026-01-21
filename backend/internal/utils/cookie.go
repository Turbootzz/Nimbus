package utils

import (
	"os"

	"github.com/gofiber/fiber/v2"
)

// CookieConfig contains shared cookie configuration from environment
type CookieConfig struct {
	Secure bool
	Domain string
}

// GetCookieConfig returns cookie configuration from environment variables.
// This should be called once at handler initialization time (not per-request)
// as it reads from environment variables.
func GetCookieConfig() CookieConfig {
	return CookieConfig{
		Secure: os.Getenv("COOKIE_SECURE") != "false", // Default to true unless explicitly "false"
		Domain: os.Getenv("COOKIE_DOMAIN"),
	}
}

// NewAuthCookie creates a standardized auth cookie
func NewAuthCookie(token string, maxAge int, config CookieConfig) *fiber.Cookie {
	return &fiber.Cookie{
		Name:     "auth_token",
		Value:    token,
		Path:     "/",
		Domain:   config.Domain,
		HTTPOnly: true,
		Secure:   config.Secure,
		SameSite: "Lax",
		MaxAge:   maxAge,
	}
}

// ClearAuthCookie returns a cookie that will clear the auth token
func ClearAuthCookie(config CookieConfig) *fiber.Cookie {
	return &fiber.Cookie{
		Name:     "auth_token",
		Value:    "",
		Path:     "/",
		Domain:   config.Domain,
		HTTPOnly: true,
		Secure:   config.Secure,
		SameSite: "Lax",
		MaxAge:   -1, // Delete the cookie
	}
}

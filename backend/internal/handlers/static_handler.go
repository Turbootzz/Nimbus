package handlers

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v2"
)

type StaticHandler struct{}

func NewStaticHandler() *StaticHandler {
	return &StaticHandler{}
}

// ServeServiceIcon serves uploaded service icon images
func (h *StaticHandler) ServeServiceIcon(c *fiber.Ctx) error {
	filename := c.Params("filename")
	if filename == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Filename is required",
		})
	}

	// Prevent directory traversal attacks
	filename = filepath.Base(filename)

	// Construct full path
	filePath := filepath.Join(UploadDir, filename)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "File not found",
		})
	}

	setImageResponseHeaders(c, filename)
	return c.SendFile(filePath)
}

// ServeAvatar serves uploaded user avatar images
func (h *StaticHandler) ServeAvatar(c *fiber.Ctx) error {
	filename := c.Params("filename")
	if filename == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Filename is required",
		})
	}

	// Prevent directory traversal attacks
	filename = filepath.Base(filename)

	// Construct full path
	filePath := filepath.Join(AvatarUploadDir, filename)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "File not found",
		})
	}

	setImageResponseHeaders(c, filename)
	return c.SendFile(filePath)
}

// setImageResponseHeaders writes the Content-Type, caching, and (for SVG) the
// hardening headers that prevent inline scripts in the SVG from executing if a
// user opens the file URL directly. We render icons in <img> context where
// scripts are inert by default, but defense in depth keeps us safe if the URL
// is loaded via <object>/<iframe> or opened in a new tab.
func setImageResponseHeaders(c *fiber.Ctx, filename string) {
	ext := filepath.Ext(filename)
	contentType := getContentTypeFromExtension(ext)
	c.Set("Content-Type", contentType)
	c.Set("Cache-Control", "public, max-age=31536000") // 1 year
	c.Set("X-Content-Type-Options", "nosniff")
	if contentType == "image/svg+xml" {
		// default-src 'none' blocks scripts, fetches, plugins; style-src
		// 'unsafe-inline' is needed because legitimate SVGs commonly use
		// inline <style>. sandbox isolates the response from same-origin.
		c.Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; sandbox")
	}
}

// getContentTypeFromExtension returns content type for file extension.
// Extensions are lowercased so e.g. "foo.SVG" or "logo.PNG" still match —
// case-insensitive filesystems (macOS, Windows) can serve such filenames.
func getContentTypeFromExtension(ext string) string {
	switch strings.ToLower(ext) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".ico":
		return "image/x-icon"
	case ".svg":
		return "image/svg+xml"
	default:
		return "application/octet-stream"
	}
}

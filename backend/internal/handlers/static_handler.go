package handlers

import (
	"os"
	"path/filepath"

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

	// Determine content type based on file extension
	ext := filepath.Ext(filename)
	contentType := getContentTypeFromExtension(ext)

	// Set content type header
	c.Set("Content-Type", contentType)
	c.Set("Cache-Control", "public, max-age=31536000") // Cache for 1 year

	// Serve the file
	return c.SendFile(filePath)
}

// getContentTypeFromExtension returns content type for file extension
func getContentTypeFromExtension(ext string) string {
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	default:
		return "application/octet-stream"
	}
}

package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v2"
)

const (
	MaxUploadSize    = 2 * 1024 * 1024 // 2MB
	UploadDir        = "uploads/service-icons"
	AllowedMimeTypes = "image/jpeg,image/png,image/gif,image/webp"
)

type UploadHandler struct{}

func NewUploadHandler() *UploadHandler {
	return &UploadHandler{}
}

// UploadServiceIcon handles service icon image uploads
func (h *UploadHandler) UploadServiceIcon(c *fiber.Ctx) error {
	// Ensure upload directory exists
	if err := os.MkdirAll(UploadDir, 0755); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create upload directory",
		})
	}

	// Get the uploaded file
	file, err := c.FormFile("icon")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No file uploaded",
		})
	}

	// Check file size
	if file.Size > MaxUploadSize {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("File size exceeds maximum allowed size of %d bytes", MaxUploadSize),
		})
	}

	// Validate file type
	contentType := file.Header.Get("Content-Type")
	if !isAllowedMimeType(contentType) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("Invalid file type. Allowed types: %s", AllowedMimeTypes),
		})
	}

	// Open the file
	src, err := file.Open()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to open uploaded file",
		})
	}
	defer src.Close()

	// Read first 512 bytes to detect actual content type (prevents MIME type spoofing)
	buffer := make([]byte, 512)
	n, err := src.Read(buffer)
	if err != nil && err != io.EOF {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to read file",
		})
	}

	// Validate actual content type
	detectedType := detectContentType(buffer[:n])
	if !isAllowedMimeType(detectedType) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "File content does not match allowed image types",
		})
	}

	// Reset read position
	if _, err := src.Seek(0, 0); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to process file",
		})
	}

	// Generate unique filename
	ext := filepath.Ext(file.Filename)
	if ext == "" {
		ext = getExtensionFromMimeType(contentType)
	}
	filename := generateUniqueFilename() + ext

	// Full path
	filePath := filepath.Join(UploadDir, filename)

	// Create destination file
	dst, err := os.Create(filePath)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to save file",
		})
	}
	defer dst.Close()

	// Copy file content
	if _, err := io.Copy(dst, src); err != nil {
		os.Remove(filePath) // Clean up on error
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to save file",
		})
	}

	// Return the relative path (without "uploads/" prefix for API consistency)
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"icon_image_path": filename, // Just the filename, path will be constructed in frontend
		"message":         "File uploaded successfully",
	})
}

// generateUniqueFilename creates a random filename
func generateUniqueFilename() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp if random fails
		return fmt.Sprintf("%d", os.Getpid())
	}
	return hex.EncodeToString(bytes)
}

// isAllowedMimeType checks if the mime type is allowed
func isAllowedMimeType(mimeType string) bool {
	allowed := strings.Split(AllowedMimeTypes, ",")
	for _, a := range allowed {
		if strings.TrimSpace(a) == mimeType {
			return true
		}
	}
	return false
}

// detectContentType detects content type from file bytes
func detectContentType(data []byte) string {
	if len(data) < 12 {
		return "application/octet-stream"
	}

	// Check for common image signatures
	switch {
	case len(data) >= 2 && data[0] == 0xFF && data[1] == 0xD8:
		return "image/jpeg"
	case len(data) >= 8 && string(data[:8]) == "\x89PNG\r\n\x1a\n":
		return "image/png"
	case len(data) >= 6 && (string(data[:6]) == "GIF87a" || string(data[:6]) == "GIF89a"):
		return "image/gif"
	case len(data) >= 12 && string(data[8:12]) == "WEBP":
		return "image/webp"
	default:
		return "application/octet-stream"
	}
}

// getExtensionFromMimeType returns file extension for a mime type
func getExtensionFromMimeType(mimeType string) string {
	switch mimeType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	default:
		return ".bin"
	}
}

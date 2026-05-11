package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/nimbus/backend/internal/repository"
)

const (
	MaxUploadSize    = 512 * 1024      // 512KB - reasonable for icons
	MaxAvatarSize    = 5 * 1024 * 1024 // 5MB for avatars
	UploadDir        = "uploads/service-icons"
	AvatarUploadDir  = "uploads/avatars"
	AllowedMimeTypes = "image/jpeg,image/png,image/gif,image/webp"
	// FaviconAllowedMimeTypes extends the upload set with .ico and SVG so server-fetched
	// favicons can be stored in their original (often crispest) form. SVG is safe to
	// serve via <img> because browsers disable scripts in that context.
	FaviconAllowedMimeTypes = AllowedMimeTypes + ",image/x-icon,image/vnd.microsoft.icon,image/svg+xml"
)

// errImageTooLarge / errImageInvalidType are returned by saveValidatedImage so
// callers can map them to 400 responses instead of 500.
var (
	errImageTooLarge    = errors.New("image exceeds maximum allowed size")
	errImageInvalidType = errors.New("image content does not match allowed types")
)

type UploadHandler struct {
	userRepo *repository.UserRepository
}

func NewUploadHandler(userRepo *repository.UserRepository) *UploadHandler {
	return &UploadHandler{
		userRepo: userRepo,
	}
}

// saveValidatedImage reads bytes from src, validates the magic-byte content
// type against allowedTypes, picks a unique filename + extension, and writes
// the data under dir. Returns the bare filename (no dir prefix).
// Caller is responsible for MkdirAll on dir before invocation.
func saveValidatedImage(src io.Reader, dir string, maxSize int64, allowedTypes string) (string, error) {
	// Cap the reader so an attacker can't blow up memory with a huge stream.
	limited := io.LimitReader(src, maxSize+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return "", fmt.Errorf("read image: %w", err)
	}
	if int64(len(data)) > maxSize {
		return "", errImageTooLarge
	}

	detected := detectContentType(data)
	if !isAllowedMimeType(detected, allowedTypes) {
		return "", errImageInvalidType
	}

	filename := generateUniqueFilename() + getExtensionFromMimeType(detected)
	filePath := filepath.Join(dir, filename)

	if err := os.WriteFile(filePath, data, 0o644); err != nil {
		return "", fmt.Errorf("write image: %w", err)
	}
	return filename, nil
}

// UploadServiceIcon handles service icon image uploads
func (h *UploadHandler) UploadServiceIcon(c *fiber.Ctx) error {
	if err := os.MkdirAll(UploadDir, 0o755); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create upload directory",
		})
	}

	file, err := c.FormFile("icon")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No file uploaded",
		})
	}

	if file.Size > MaxUploadSize {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("File size exceeds maximum allowed size of %d bytes", MaxUploadSize),
		})
	}

	// Cheap up-front check on the declared content type. The real validation
	// happens via magic-byte sniffing inside saveValidatedImage.
	if declared := file.Header.Get("Content-Type"); declared != "" && !isAllowedMimeType(declared, AllowedMimeTypes) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("Invalid file type. Allowed types: %s", AllowedMimeTypes),
		})
	}

	src, err := file.Open()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to open uploaded file",
		})
	}
	defer src.Close()

	filename, err := saveValidatedImage(src, UploadDir, MaxUploadSize, AllowedMimeTypes)
	if err != nil {
		return saveImageErrorResponse(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"icon_image_path": filename,
		"message":         "File uploaded successfully",
	})
}

// UploadAvatar handles user avatar uploads (local users only)
func (h *UploadHandler) UploadAvatar(c *fiber.Ctx) error {
	userID, err := RequireUserID(c)
	if err != nil {
		return err
	}

	user, err := h.userRepo.GetByID(userID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	if user.Provider != "local" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Avatar uploads are only allowed for local accounts. Your profile picture is synced from " + user.Provider,
		})
	}

	if err := os.MkdirAll(AvatarUploadDir, 0o755); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create upload directory",
		})
	}

	file, err := c.FormFile("avatar")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No file uploaded",
		})
	}

	if file.Size > MaxAvatarSize {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("File size exceeds maximum allowed size of %d MB", MaxAvatarSize/(1024*1024)),
		})
	}

	if declared := file.Header.Get("Content-Type"); declared != "" && !isAllowedMimeType(declared, AllowedMimeTypes) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("Invalid file type. Allowed types: %s", AllowedMimeTypes),
		})
	}

	src, err := file.Open()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to open uploaded file",
		})
	}
	defer src.Close()

	// Delete old avatar (if any) before writing the new one.
	if user.AvatarURL != nil && strings.HasPrefix(*user.AvatarURL, "/uploads/avatars/") {
		oldFilename := strings.TrimPrefix(*user.AvatarURL, "/uploads/avatars/")
		os.Remove(filepath.Join(AvatarUploadDir, oldFilename))
	}

	filename, err := saveValidatedImage(src, AvatarUploadDir, MaxAvatarSize, AllowedMimeTypes)
	if err != nil {
		return saveImageErrorResponse(c, err)
	}

	avatarURL := "/uploads/avatars/" + filename
	if err := h.userRepo.UpdateAvatar(userID, &avatarURL); err != nil {
		os.Remove(filepath.Join(AvatarUploadDir, filename))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update user avatar",
		})
	}

	updatedUser, err := h.userRepo.GetByID(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve updated user",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Avatar uploaded successfully",
		"user":    updatedUser,
	})
}

// saveImageErrorResponse maps saveValidatedImage errors to appropriate HTTP responses.
// Unexpected errors (disk full, EACCES, etc.) are logged so ops can debug.
func saveImageErrorResponse(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, errImageTooLarge):
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Image exceeds maximum allowed size",
		})
	case errors.Is(err, errImageInvalidType):
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "File content does not match allowed image types",
		})
	default:
		log.Printf("saveValidatedImage failed: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to save file",
		})
	}
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

// isAllowedMimeType checks if the mime type appears in the comma-separated allow list.
func isAllowedMimeType(mimeType, allowList string) bool {
	for _, a := range strings.Split(allowList, ",") {
		if strings.TrimSpace(a) == mimeType {
			return true
		}
	}
	return false
}

// detectContentType detects content type from file bytes via magic-byte sniffing.
// Binary formats are checked first; SVG is only considered if none match (since
// SVG is text and would otherwise false-positive on a few raster prefixes).
func detectContentType(data []byte) string {
	switch {
	case len(data) >= 2 && data[0] == 0xFF && data[1] == 0xD8:
		return "image/jpeg"
	case len(data) >= 8 && string(data[:8]) == "\x89PNG\r\n\x1a\n":
		return "image/png"
	case len(data) >= 6 && (string(data[:6]) == "GIF87a" || string(data[:6]) == "GIF89a"):
		return "image/gif"
	case len(data) >= 12 && string(data[8:12]) == "WEBP":
		return "image/webp"
	// ICO: 00 00 01 00 (reserved=0, type=1)
	case len(data) >= 4 && data[0] == 0x00 && data[1] == 0x00 && data[2] == 0x01 && data[3] == 0x00:
		return "image/x-icon"
	case looksLikeSVG(data):
		return "image/svg+xml"
	default:
		return "application/octet-stream"
	}
}

// looksLikeSVG does a loose text scan for the SVG root element. Real-world SVG
// files may start with an XML declaration, a doctype, comments, or whitespace
// before <svg>, so we scan the first chunk rather than just the first bytes.
func looksLikeSVG(data []byte) bool {
	const scan = 1024
	if len(data) > scan {
		data = data[:scan]
	}
	lower := strings.ToLower(string(data))
	// Skip BOM (U+FEFF) and leading whitespace
	lower = strings.TrimLeft(lower, " \t\r\n\ufeff")
	if !strings.HasPrefix(lower, "<") {
		return false
	}
	return strings.Contains(lower, "<svg")
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
	case "image/x-icon", "image/vnd.microsoft.icon":
		return ".ico"
	case "image/svg+xml":
		return ".svg"
	default:
		return ".bin"
	}
}

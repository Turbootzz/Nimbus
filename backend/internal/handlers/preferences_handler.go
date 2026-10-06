package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/url"
	"os"
	"regexp"
	"sync"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/repository"
	"github.com/nimbus/backend/internal/services"
	"github.com/nimbus/backend/internal/utils"
)

const (
	MaxWallpaperSize = 8 * 1024 * 1024
	// MaxRequestBodySize is the server's body limit: a wallpaper plus the
	// multipart overhead
	MaxRequestBodySize = MaxWallpaperSize + 2*1024*1024
	wallpaperURLPrefix = "/uploads/wallpapers/"
)

// WallpaperUploadDir is a var (not const) so tests can redirect uploads to t.TempDir().
var WallpaperUploadDir = "uploads/wallpapers"

// localWallpaper matches the theme_background an upload stores
var localWallpaper = regexp.MustCompile(`^/uploads/wallpapers/[0-9a-f]{32}\.(jpg|png|gif|webp)$`)

type PreferencesHandler struct {
	preferencesRepo *repository.PreferencesRepository
	validator       *validator.Validate
	poller          services.Poller
}

// SetPoller lets a status strip change start or stop polling its integrations
func (h *PreferencesHandler) SetPoller(p services.Poller) {
	h.poller = p
}

func NewPreferencesHandler(preferencesRepo *repository.PreferencesRepository) *PreferencesHandler {
	v := validator.New()

	// Register custom validator for HTTP(S) URLs only
	v.RegisterValidation("httpurl", func(fl validator.FieldLevel) bool {
		urlStr := fl.Field().String()
		if urlStr == "" {
			return true // Empty is valid (omitempty will handle required check)
		}

		parsedURL, err := url.Parse(urlStr)
		if err != nil {
			return false
		}

		// Only allow http and https schemes to prevent XSS
		return parsedURL.Scheme == "http" || parsedURL.Scheme == "https"
	})

	return &PreferencesHandler{
		preferencesRepo: preferencesRepo,
		validator:       v,
	}
}

// GetPreferences retrieves the current user's preferences
func (h *PreferencesHandler) GetPreferences(c *fiber.Ctx) error {
	userID, err := RequireUserID(c)
	if err != nil {
		return err
	}

	preferences, err := h.preferencesRepo.GetByUserID(c.Context(), userID)
	if err == sql.ErrNoRows {
		// Return default preferences if user hasn't set any yet
		// Same as the column defaults
		return c.JSON(models.PreferencesResponse{
			ThemeMode:             "auto",
			ThemeBackground:       nil,
			ThemeAccentColor:      nil,
			OpenInNewTab:          true,
			EnableCardResizing:    true,
			EnableServiceGrouping: true,
			CardScale:             "medium",
			ViewMode:              "grid",
			CardOpacity:           100,
			LayoutMode:            "classic",
			StatusStrip:           models.StatusStrip{Chips: []models.StatusChip{}},
			UpdatedAt:             time.Time{}, // Zero value for time
		})
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve preferences",
		})
	}

	return c.JSON(preferences.ToResponse())
}

// UpdatePreferences updates the current user's preferences
func (h *PreferencesHandler) UpdatePreferences(c *fiber.Ctx) error {
	userID, err := RequireUserID(c)
	if err != nil {
		return err
	}

	// Parse request body
	var req models.PreferencesUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Manual validation for NullableString fields. An uploaded wallpaper is
	// stored as its local path; a client may send back its own, never
	// another user's (the sweep would then keep or drop the wrong file).
	background := req.ThemeBackground.GetValue()
	if req.ThemeBackground.IsSet() && background != nil && localWallpaper.MatchString(*background) {
		current, err := h.preferencesRepo.GetByUserID(c.Context(), userID)
		if err != nil || current.ThemeBackground == nil || *current.ThemeBackground != *background {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":  "Validation failed",
				"fields": map[string]string{"theme_background": "Upload the image instead"},
			})
		}
	} else if req.ThemeBackground.IsSet() && background != nil {
		backgroundURL := *background

		// First validate URL format
		if err := h.validator.Var(backgroundURL, "httpurl"); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":  "Validation failed",
				"fields": map[string]string{"theme_background": "theme_background must be a valid HTTP or HTTPS URL"},
			})
		}

		// Validate URL format (allows private/local IPs for homelab use)
		if err := utils.ValidateIconURL(backgroundURL); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":  "Validation failed",
				"fields": map[string]string{"theme_background": fmt.Sprintf("Invalid background URL: %s", err.Error())},
			})
		}
	}
	if req.ThemeAccentColor.IsSet() && req.ThemeAccentColor.GetValue() != nil {
		if err := h.validator.Var(*req.ThemeAccentColor.GetValue(), "hexcolor"); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":  "Validation failed",
				"fields": map[string]string{"theme_accent_color": "theme_accent_color must be a valid hex color (e.g., #3B82F6)"},
			})
		}
	}

	// Validate request using struct tags
	if err := h.validator.Struct(req); err != nil {
		// Use comma-ok to safely type assert validation errors
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			// Parse validation errors and return field-specific messages
			errorMessages := make(map[string]string)

			for _, fieldError := range validationErrors {
				field := fieldError.Field()
				switch fieldError.Tag() {
				case "required":
					errorMessages[field] = fmt.Sprintf("%s is required", field)
				case "oneof":
					errorMessages[field] = fmt.Sprintf("%s must be one of: %s", field, fieldError.Param())
				case "hexcolor":
					errorMessages[field] = fmt.Sprintf("%s must be a valid hex color (e.g., #3B82F6)", field)
				case "httpurl":
					errorMessages[field] = fmt.Sprintf("%s must be a valid HTTP or HTTPS URL", field)
				case "min", "max":
					errorMessages[field] = fmt.Sprintf("%s is out of range", field)
				case "uuid":
					errorMessages[field] = fmt.Sprintf("%s must be an id", field)
				default:
					errorMessages[field] = fmt.Sprintf("%s is invalid", field)
				}
			}

			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":  "Validation failed",
				"fields": errorMessages,
			})
		}

		// If not ValidationErrors, treat as generic validation error
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("Validation error: %s", err.Error()),
		})
	}

	preferences, err := h.save(c.Context(), userID, &req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update preferences",
		})
	}
	if req.StatusStrip != nil && h.poller != nil {
		h.poller.Kick()
	}
	return c.JSON(preferences.ToResponse())
}

// save stores req and returns the updated preferences. When the background
// changes, wallpapers nothing points at anymore are deleted.
func (h *PreferencesHandler) save(ctx context.Context, userID string, req *models.PreferencesUpdateRequest) (*models.UserPreferences, error) {
	if err := h.preferencesRepo.Upsert(ctx, userID, req); err != nil {
		return nil, err
	}
	if req.ThemeBackground.IsSet() {
		PruneWallpapers(ctx, h.preferencesRepo)
	}
	return h.preferencesRepo.GetByUserID(ctx, userID)
}

// UploadWallpaper stores an uploaded image and makes it the user's background
func (h *PreferencesHandler) UploadWallpaper(c *fiber.Ctx) error {
	userID, err := RequireUserID(c)
	if err != nil {
		return err
	}

	file, err := c.FormFile("wallpaper")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No file uploaded",
		})
	}
	if file.Size > MaxWallpaperSize {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("File size exceeds maximum allowed size of %d MB", MaxWallpaperSize/(1024*1024)),
		})
	}
	if declared := file.Header.Get("Content-Type"); declared != "" && !isAllowedMimeType(declared, AllowedMimeTypes) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("Invalid file type. Allowed types: %s", AllowedMimeTypes),
		})
	}

	if err := os.MkdirAll(WallpaperUploadDir, 0o755); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create upload directory",
		})
	}
	wallpaperSweep.RLock()
	stored := sync.OnceFunc(wallpaperSweep.RUnlock)
	defer stored()
	src, err := file.Open()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to open uploaded file",
		})
	}
	defer src.Close()

	filename, err := saveValidatedImage(src, WallpaperUploadDir, MaxWallpaperSize, AllowedMimeTypes)
	if err != nil {
		return saveImageErrorResponse(c, err)
	}

	wallpaperURL := wallpaperURLPrefix + filename
	req := &models.PreferencesUpdateRequest{ThemeBackground: models.NullableString{Value: &wallpaperURL, Set: true}}
	if err := h.preferencesRepo.Upsert(c.Context(), userID, req); err != nil {
		removeLocalWallpaper(&wallpaperURL, "UploadWallpaper")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to save wallpaper",
		})
	}
	// Stored now, so this upload no longer holds the sweep back
	stored()
	PruneWallpapers(c.Context(), h.preferencesRepo)

	preferences, err := h.preferencesRepo.GetByUserID(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve updated preferences",
		})
	}
	return c.JSON(preferences.ToResponse())
}

// wallpaperSweep is read-locked by uploads from writing their file until it
// is stored, and locked by the sweep, so the sweep never takes a file that
// is about to be stored
var wallpaperSweep sync.RWMutex

// PruneWallpapers deletes uploaded wallpapers no preferences row points at:
// replaced or cleared ones, those of deleted users, and leftovers of
// concurrent changes. It skips a round while an upload is running. Errors
// are logged; a missed file goes next time.
func PruneWallpapers(ctx context.Context, repo *repository.PreferencesRepository) {
	if !wallpaperSweep.TryLock() {
		return
	}
	defer wallpaperSweep.Unlock()
	inUse, err := repo.WallpapersInUse(ctx)
	if err != nil {
		log.Printf("PruneWallpapers: %v", err)
		return
	}
	entries, err := os.ReadDir(WallpaperUploadDir)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("PruneWallpapers: %v", err)
		}
		return
	}
	for _, entry := range entries {
		fileURL := wallpaperURLPrefix + entry.Name()
		if entry.IsDir() || inUse[fileURL] {
			continue
		}
		removeLocalWallpaper(&fileURL, "PruneWallpapers")
	}
}

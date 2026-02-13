package handlers

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"log"
	"net/mail"
	"os"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/repository"
	"github.com/nimbus/backend/internal/services"
)

type PasswordHandler struct {
	userRepo          *repository.UserRepository
	authService       *services.AuthService
	emailService      *services.EmailService
	passwordResetRepo *repository.PasswordResetRepository
}

func NewPasswordHandler(
	userRepo *repository.UserRepository,
	authService *services.AuthService,
	emailService *services.EmailService,
	passwordResetRepo *repository.PasswordResetRepository,
) *PasswordHandler {
	return &PasswordHandler{
		userRepo:          userRepo,
		authService:       authService,
		emailService:      emailService,
		passwordResetRepo: passwordResetRepo,
	}
}

// Rate limiting for forgot-password
var (
	forgotPasswordRateLimit   = make(map[string][]time.Time)
	forgotPasswordRateLimitMu sync.Mutex
)

const (
	maxRequestsPerEmail = 3
	maxRequestsPerIP    = 10
	rateLimitWindow     = 15 * time.Minute
	tokenExpiry         = 1 * time.Hour
)

// ForgotPassword handles password reset requests
// Always returns 200 to prevent email enumeration
func (h *PasswordHandler) ForgotPassword(c *fiber.Ctx) error {
	var req models.ForgotPasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Email == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Email is required",
		})
	}

	// Validate email format
	if _, err := mail.ParseAddress(req.Email); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid email format",
		})
	}

	// Rate limit by email and IP
	ip := c.IP()
	if isRateLimited(req.Email, maxRequestsPerEmail) || isRateLimited("ip:"+ip, maxRequestsPerIP) {
		// Return same success message to prevent enumeration
		return c.JSON(fiber.Map{
			"message": "If an account with that email exists, a password reset link has been sent.",
		})
	}

	// Process in the background — always return 200 immediately
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Panic in processForgotPassword: %v", r)
			}
		}()
		h.processForgotPassword(req.Email)
	}()

	return c.JSON(fiber.Map{
		"message": "If an account with that email exists, a password reset link has been sent.",
	})
}

// processForgotPassword handles the actual token creation and email sending
func (h *PasswordHandler) processForgotPassword(email string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Look up user by email
	user, err := h.userRepo.GetByEmail(email)
	if err != nil {
		return // User not found — silently ignore
	}

	// Skip OAuth users (they don't have passwords)
	if user.Provider != "local" || user.Password == nil {
		return
	}

	// Generate random token (32 bytes)
	rawToken := make([]byte, 32)
	if _, err := rand.Read(rawToken); err != nil {
		log.Printf("Failed to generate reset token: %v", err)
		return
	}
	rawTokenHex := hex.EncodeToString(rawToken)

	// Hash token for storage
	hash := sha256.Sum256([]byte(rawTokenHex))
	tokenHash := hex.EncodeToString(hash[:])

	// Invalidate previous tokens for this user
	if err := h.passwordResetRepo.InvalidateForUser(ctx, user.ID); err != nil {
		log.Printf("Failed to invalidate old tokens: %v", err)
	}

	// Store the hashed token
	token := &models.PasswordResetToken{
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(tokenExpiry),
	}
	if err := h.passwordResetRepo.Create(ctx, token); err != nil {
		log.Printf("Failed to create reset token: %v", err)
		return
	}

	// Send password reset email
	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:3000"
	}

	if err := h.emailService.SendPasswordResetEmail(ctx, user.Email, rawTokenHex, user.Name, frontendURL); err != nil {
		log.Printf("Failed to send password reset email for user %s: %v", user.ID, err)
	}
}

// ResetPassword handles password reset using a token
func (h *PasswordHandler) ResetPassword(c *fiber.Ctx) error {
	var req models.ResetPasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Token == "" || req.NewPassword == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Token and new password are required",
		})
	}

	if len(req.NewPassword) < 8 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Password must be at least 8 characters",
		})
	}

	// Hash the provided token
	hash := sha256.Sum256([]byte(req.Token))
	tokenHash := hex.EncodeToString(hash[:])

	// Look up the token
	resetToken, err := h.passwordResetRepo.GetByTokenHash(c.Context(), tokenHash)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid or expired reset token",
		})
	}

	// Verify user exists and is a local auth user
	user, err := h.userRepo.GetByID(resetToken.UserID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid or expired reset token",
		})
	}

	if user.Provider != "local" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "This account uses OAuth login. Password reset is not available.",
		})
	}

	// Check new password is different from current
	if user.Password != nil && h.authService.ComparePassword(*user.Password, req.NewPassword) == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "New password must be different from your current password",
		})
	}

	// Hash new password
	hashedPassword, err := h.authService.HashPassword(req.NewPassword)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to process password",
		})
	}

	// Update password
	if err := h.userRepo.UpdatePassword(user.ID, hashedPassword); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update password",
		})
	}

	// Invalidate all tokens for this user (including the one just used)
	if err := h.passwordResetRepo.InvalidateForUser(c.Context(), user.ID); err != nil {
		log.Printf("Failed to invalidate tokens: %v", err)
	}

	return c.JSON(fiber.Map{
		"message": "Password has been reset successfully. Please log in with your new password.",
	})
}

// ChangePassword handles password change for authenticated users
func (h *PasswordHandler) ChangePassword(c *fiber.Ctx) error {
	userID, err := RequireUserID(c)
	if err != nil {
		return err
	}

	var req models.ChangePasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.CurrentPassword == "" || req.NewPassword == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Current password and new password are required",
		})
	}

	if len(req.NewPassword) < 8 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "New password must be at least 8 characters",
		})
	}

	if req.CurrentPassword == req.NewPassword {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "New password must be different from current password",
		})
	}

	// Load user
	user, err := h.userRepo.GetByID(userID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	// Only local auth users can change passwords
	if user.Provider != "local" || user.Password == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Password change is not available for OAuth accounts",
		})
	}

	// Verify current password
	if err := h.authService.ComparePassword(*user.Password, req.CurrentPassword); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Current password is incorrect",
		})
	}

	// Hash new password
	hashedPassword, err := h.authService.HashPassword(req.NewPassword)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to process password",
		})
	}

	// Update password
	if err := h.userRepo.UpdatePassword(userID, hashedPassword); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update password",
		})
	}

	// Invalidate any outstanding reset tokens
	if err := h.passwordResetRepo.InvalidateForUser(c.Context(), userID); err != nil {
		log.Printf("Failed to invalidate tokens after password change: %v", err)
	}

	return c.JSON(fiber.Map{
		"message": "Password changed successfully",
	})
}

// isRateLimited checks if a key has exceeded the rate limit
func isRateLimited(key string, maxRequests int) bool {
	forgotPasswordRateLimitMu.Lock()
	defer forgotPasswordRateLimitMu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rateLimitWindow)

	// Filter out old entries
	var recent []time.Time
	for _, t := range forgotPasswordRateLimit[key] {
		if t.After(cutoff) {
			recent = append(recent, t)
		}
	}

	if len(recent) >= maxRequests {
		forgotPasswordRateLimit[key] = recent
		return true
	}

	forgotPasswordRateLimit[key] = append(recent, now)
	return false
}

// CleanupForgotPasswordRateLimit removes expired rate limit entries
func CleanupForgotPasswordRateLimit() int {
	forgotPasswordRateLimitMu.Lock()
	defer forgotPasswordRateLimitMu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rateLimitWindow)
	removed := 0

	for key, times := range forgotPasswordRateLimit {
		var recent []time.Time
		for _, t := range times {
			if t.After(cutoff) {
				recent = append(recent, t)
			}
		}
		if len(recent) == 0 {
			delete(forgotPasswordRateLimit, key)
			removed++
		} else {
			forgotPasswordRateLimit[key] = recent
		}
	}

	return removed
}

package handlers

import "github.com/gofiber/fiber/v2"

// Standard HTTP error response helpers
// These reduce duplication and ensure consistent error response format across all handlers

// BadRequest returns a 400 Bad Request error with a custom message
func BadRequest(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
		"error": message,
	})
}

// Unauthorized returns a 401 Unauthorized error with a custom message
func Unauthorized(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
		"error": message,
	})
}

// Forbidden returns a 403 Forbidden error with a custom message
func Forbidden(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
		"error": message,
	})
}

// NotFound returns a 404 Not Found error with a custom message
func NotFound(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
		"error": message,
	})
}

// InternalError returns a 500 Internal Server Error with a custom message
func InternalError(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
		"error": message,
	})
}

// Success returns a 200 OK response with data
func Success(c *fiber.Ctx, data interface{}) error {
	return c.Status(fiber.StatusOK).JSON(data)
}

// Created returns a 201 Created response with data
func Created(c *fiber.Ctx, data interface{}) error {
	return c.Status(fiber.StatusCreated).JSON(data)
}

package handlers

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

func TestUploadServiceIcon_Success(t *testing.T) {
	// Setup
	handler := NewUploadHandler()
	app := fiber.New()
	app.Post("/upload", handler.UploadServiceIcon)

	// Create a test image (1x1 PNG)
	pngBytes := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, // PNG signature
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52, // IHDR chunk
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53,
		0xDE, 0x00, 0x00, 0x00, 0x0C, 0x49, 0x44, 0x41,
		0x54, 0x08, 0xD7, 0x63, 0xF8, 0xCF, 0xC0, 0x00,
		0x00, 0x03, 0x01, 0x01, 0x00, 0x18, 0xDD, 0x8D,
		0xB4, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4E,
		0x44, 0xAE, 0x42, 0x60, 0x82,
	}

	// Create multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Create form file with proper header
	h := make(map[string][]string)
	h["Content-Type"] = []string{"image/png"}
	part, err := writer.CreatePart(map[string][]string{
		"Content-Disposition": {`form-data; name="icon"; filename="test.png"`},
		"Content-Type":        {"image/png"},
	})
	assert.NoError(t, err)
	_, err = part.Write(pngBytes)
	assert.NoError(t, err)
	writer.Close()

	// Create request
	req := httptest.NewRequest("POST", "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Execute
	resp, err := app.Test(req)
	assert.NoError(t, err)

	// Assert
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	// Cleanup
	defer func() {
		os.RemoveAll(UploadDir)
	}()
}

func TestUploadServiceIcon_NoFile(t *testing.T) {
	// Setup
	handler := NewUploadHandler()
	app := fiber.New()
	app.Post("/upload", handler.UploadServiceIcon)

	// Create request without file
	req := httptest.NewRequest("POST", "/upload", nil)
	req.Header.Set("Content-Type", "multipart/form-data")

	// Execute
	resp, err := app.Test(req)
	assert.NoError(t, err)

	// Assert
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestUploadServiceIcon_FileTooLarge(t *testing.T) {
	// Setup
	handler := NewUploadHandler()
	app := fiber.New()
	app.Post("/upload", handler.UploadServiceIcon)

	// Create a large file (> 512KB)
	largeData := make([]byte, MaxUploadSize+1)

	// Create multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreatePart(map[string][]string{
		"Content-Disposition": {`form-data; name="icon"; filename="large.png"`},
		"Content-Type":        {"image/png"},
	})
	assert.NoError(t, err)
	_, err = part.Write(largeData)
	assert.NoError(t, err)
	writer.Close()

	// Create request
	req := httptest.NewRequest("POST", "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Execute
	resp, err := app.Test(req)
	assert.NoError(t, err)

	// Assert
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestUploadServiceIcon_InvalidFileType(t *testing.T) {
	// Setup
	handler := NewUploadHandler()
	app := fiber.New()
	app.Post("/upload", handler.UploadServiceIcon)

	// Create multipart form with text file
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreatePart(map[string][]string{
		"Content-Disposition": {`form-data; name="icon"; filename="test.txt"`},
		"Content-Type":        {"text/plain"},
	})
	assert.NoError(t, err)
	_, err = part.Write([]byte("This is not an image"))
	assert.NoError(t, err)
	writer.Close()

	// Create request
	req := httptest.NewRequest("POST", "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Execute
	resp, err := app.Test(req)
	assert.NoError(t, err)

	// Assert
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	// Cleanup
	defer func() {
		os.RemoveAll(UploadDir)
	}()
}

func TestDetectContentType(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected string
	}{
		{
			name:     "JPEG",
			data:     []byte{0xFF, 0xD8, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			expected: "image/jpeg",
		},
		{
			name:     "PNG",
			data:     []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x00},
			expected: "image/png",
		},
		{
			name:     "GIF87a",
			data:     []byte("GIF87a" + string([]byte{0, 0, 0, 0, 0, 0})),
			expected: "image/gif",
		},
		{
			name:     "GIF89a",
			data:     []byte("GIF89a" + string([]byte{0, 0, 0, 0, 0, 0})),
			expected: "image/gif",
		},
		{
			name:     "WEBP",
			data:     []byte{0, 0, 0, 0, 0, 0, 0, 0, 'W', 'E', 'B', 'P'},
			expected: "image/webp",
		},
		{
			name:     "Unknown",
			data:     []byte{0x00, 0x01, 0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			expected: "application/octet-stream",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectContentType(tt.data)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestServeServiceIcon_Success(t *testing.T) {
	// Setup
	handler := NewStaticHandler()
	app := fiber.New()
	app.Get("/uploads/service-icons/:filename", handler.ServeServiceIcon)

	// Create test directory and file
	os.MkdirAll(UploadDir, 0755)
	testFile := filepath.Join(UploadDir, "test.png")
	err := os.WriteFile(testFile, []byte("test image data"), 0644)
	assert.NoError(t, err)

	// Create request
	req := httptest.NewRequest("GET", "/uploads/service-icons/test.png", nil)

	// Execute
	resp, err := app.Test(req)
	assert.NoError(t, err)

	// Assert
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	assert.Equal(t, "image/png", resp.Header.Get("Content-Type"))

	// Read response body
	body, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Equal(t, "test image data", string(body))

	// Cleanup
	defer func() {
		os.RemoveAll(UploadDir)
	}()
}

func TestServeServiceIcon_NotFound(t *testing.T) {
	// Setup
	handler := NewStaticHandler()
	app := fiber.New()
	app.Get("/uploads/service-icons/:filename", handler.ServeServiceIcon)

	// Create request for non-existent file
	req := httptest.NewRequest("GET", "/uploads/service-icons/nonexistent.png", nil)

	// Execute
	resp, err := app.Test(req)
	assert.NoError(t, err)

	// Assert
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestServeServiceIcon_DirectoryTraversal(t *testing.T) {
	// Setup
	handler := NewStaticHandler()
	app := fiber.New()
	app.Get("/uploads/service-icons/:filename", handler.ServeServiceIcon)

	// Try directory traversal attack
	req := httptest.NewRequest("GET", "/uploads/service-icons/../../../etc/passwd", nil)

	// Execute
	resp, err := app.Test(req)
	assert.NoError(t, err)

	// Assert - should not find file because filepath.Base prevents traversal
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

package services

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/nimbus/backend/internal/models"
)

// A refused service URL tells which rule it broke
func TestNewServiceFromRequestURLMessage(t *testing.T) {
	for url, want := range map[string]string{
		"ftp://nas.lan":                    "Invalid URL: it must start with http:// or https:// and have a host",
		"http://169.254.169.254/":          "Invalid URL: cloud metadata endpoints are not allowed",
		"http://metadata.google.internal/": "Invalid URL: cloud metadata endpoints are not allowed",
	} {
		_, err := NewServiceFromRequest(&models.ServiceCreateRequest{Name: "x", URL: url})
		assert.EqualError(t, err, want, url)
	}
}

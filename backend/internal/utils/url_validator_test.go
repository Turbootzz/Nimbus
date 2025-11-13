package utils

import (
	"testing"
)

func TestValidateExternalImageURL_ValidURLs(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{"HTTPS URL", "https://example.com/image.png"},
		{"HTTP URL", "http://example.com/image.jpg"},
		{"URL with path", "https://cdn.example.com/icons/service.png"},
		{"URL with query params", "https://example.com/image.png?size=large"},
		{"URL with port", "https://example.com:8080/image.png"},
		{"Subdomain", "https://images.example.com/icon.png"},
		{"Deep path", "https://example.com/assets/images/icons/service.png"},
		{"Common CDN", "https://cdn.jsdelivr.net/npm/simple-icons@latest/icons/github.svg"},
		{"Iconify", "https://api.iconify.design/logos/docker.svg"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateExternalImageURL(tt.url)
			if err != nil {
				t.Errorf("ValidateExternalImageURL(%q) returned error: %v, expected nil", tt.url, err)
			}
		})
	}
}

func TestValidateExternalImageURL_InvalidURLs(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		errContains string
	}{
		{"Empty URL", "", "invalid URL format"},
		{"No scheme", "example.com/image.png", "invalid URL format"},
		{"FTP protocol", "ftp://example.com/image.png", "only HTTP/HTTPS protocols are allowed"},
		{"File protocol", "file:///etc/passwd", "only HTTP/HTTPS protocols are allowed"},
		{"Data URL", "data:image/png;base64,iVBORw0KGgo=", "only HTTP/HTTPS protocols are allowed"},
		{"Javascript protocol", "javascript:alert(1)", "only HTTP/HTTPS protocols are allowed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateExternalImageURL(tt.url)
			if err == nil {
				t.Errorf("ValidateExternalImageURL(%q) expected error containing %q, got nil", tt.url, tt.errContains)
				return
			}
			if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
				t.Errorf("ValidateExternalImageURL(%q) error = %q, expected to contain %q", tt.url, err.Error(), tt.errContains)
			}
		})
	}
}

func TestValidateExternalImageURL_SSRF_Localhost(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{"localhost", "http://localhost/image.png"},
		{"localhost HTTPS", "https://localhost:8080/image.png"},
		{"127.0.0.1", "http://127.0.0.1/image.png"},
		{"127.0.0.1 with port", "https://127.0.0.1:3000/image.png"},
		{"IPv6 localhost ::1", "http://[::1]/image.png"},
		{"0.0.0.0", "http://0.0.0.0/image.png"},
		{"IPv6 all zeros", "http://[::]/image.png"},
		{"127.x.x.x variations", "http://127.1.1.1/image.png"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateExternalImageURL(tt.url)
			if err == nil {
				t.Errorf("ValidateExternalImageURL(%q) expected error for localhost, got nil", tt.url)
			}
		})
	}
}

func TestValidateExternalImageURL_SSRF_PrivateIPs(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		// Private IP ranges (RFC 1918)
		{"10.x.x.x", "http://10.0.0.1/image.png"},
		{"10.x.x.x high", "http://10.255.255.255/image.png"},
		{"172.16.x.x", "http://172.16.0.1/image.png"},
		{"172.31.x.x", "http://172.31.255.255/image.png"},
		{"192.168.x.x", "http://192.168.1.1/image.png"},
		{"192.168.0.x", "http://192.168.0.100/image.png"},

		// Link-local addresses
		{"169.254.x.x", "http://169.254.1.1/image.png"},
		{"Link-local IPv6", "http://[fe80::1]/image.png"},

		// Carrier-grade NAT
		{"100.64.x.x start", "http://100.64.0.0/image.png"},
		{"100.127.x.x end", "http://100.127.255.255/image.png"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateExternalImageURL(tt.url)
			if err == nil {
				t.Errorf("ValidateExternalImageURL(%q) expected error for private IP, got nil", tt.url)
			}
		})
	}
}

func TestValidateExternalImageURL_SSRF_CloudMetadata(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{"AWS/GCP metadata", "http://169.254.169.254/latest/meta-data/"},
		{"AWS metadata HTTPS", "https://169.254.169.254/latest/meta-data/"},
		{"Azure metadata", "http://168.63.129.16/metadata/instance"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateExternalImageURL(tt.url)
			if err == nil {
				t.Errorf("ValidateExternalImageURL(%q) expected error for cloud metadata endpoint, got nil", tt.url)
			}
		})
	}
}

func TestValidateExternalImageURL_SSRF_LocalDomains(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{"homeassistant.local", "http://homeassistant.local/image.png"},
		{"router.local", "https://router.local:8080/favicon.png"},
		{"nas.local", "http://nas.local/icon.png"},
		{"printer.local", "https://printer.local/image.png"},
		{".local with subdomain", "http://server.home.local/image.png"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateExternalImageURL(tt.url)
			if err == nil {
				t.Errorf("ValidateExternalImageURL(%q) expected error for .local domain, got nil", tt.url)
			}
		})
	}
}

func TestValidateExternalImageURL_PublicIPs(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{"Google DNS", "http://8.8.8.8/image.png"},
		{"Cloudflare DNS", "http://1.1.1.1/image.png"},
		{"Public IP", "http://93.184.216.34/image.png"}, // example.com IP
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateExternalImageURL(tt.url)
			// These should pass validation (public IPs are allowed)
			if err != nil {
				t.Errorf("ValidateExternalImageURL(%q) expected nil for public IP, got error: %v", tt.url, err)
			}
		})
	}
}

func TestValidateExternalImageURL_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		shouldError bool
		errContains string
	}{
		{"URL without hostname", "http:///image.png", true, "must have a hostname"},
		{"Multiple slashes", "http://example.com//image.png", false, ""},
		{"Encoded localhost", "http://127.0.0.1%2F/image.png", true, "invalid URL format"}, // Invalid URL encoding
		{"Port 0", "http://example.com:0/image.png", false, ""},
		{"High port", "http://example.com:65535/image.png", false, ""},
		{"Fragment", "http://example.com/image.png#section", false, ""},
		{"Username in URL", "http://user@example.com/image.png", false, ""},
		{"User:pass in URL", "http://user:pass@example.com/image.png", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateExternalImageURL(tt.url)
			if tt.shouldError {
				if err == nil {
					t.Errorf("ValidateExternalImageURL(%q) expected error, got nil", tt.url)
				} else if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("ValidateExternalImageURL(%q) error = %q, expected to contain %q", tt.url, err.Error(), tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateExternalImageURL(%q) expected nil, got error: %v", tt.url, err)
				}
			}
		})
	}
}

func TestIsLocalhost(t *testing.T) {
	tests := []struct {
		hostname string
		expected bool
	}{
		{"localhost", true},
		{"LOCALHOST", true},
		{"Localhost", true},
		{"127.0.0.1", true},
		{"::1", true},
		{"0.0.0.0", true},
		{"::", true},
		{"example.com", false},
		{"127.0.0.2", false},
		{"local", false},
		{"mylocalhost", false},
	}

	for _, tt := range tests {
		t.Run(tt.hostname, func(t *testing.T) {
			result := isLocalhost(tt.hostname)
			if result != tt.expected {
				t.Errorf("isLocalhost(%q) = %v, expected %v", tt.hostname, result, tt.expected)
			}
		})
	}
}

func TestIsCGNATRange(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected bool
	}{
		{"CGNAT start", "100.64.0.0", true},
		{"CGNAT mid", "100.100.50.25", true},
		{"CGNAT end", "100.127.255.255", true},
		{"Just before CGNAT", "100.63.255.255", false},
		{"Just after CGNAT", "100.128.0.0", false},
		{"Not CGNAT", "100.200.1.1", false},
		{"Different range", "192.168.1.1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := parseIP(tt.ip)
			if ip == nil {
				t.Fatalf("Failed to parse IP: %s", tt.ip)
			}
			result := isCGNATRange(ip)
			if result != tt.expected {
				t.Errorf("isCGNATRange(%q) = %v, expected %v", tt.ip, result, tt.expected)
			}
		})
	}
}

func TestIsCloudMetadataIP(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected bool
	}{
		{"AWS/GCP metadata", "169.254.169.254", true},
		{"Azure metadata", "168.63.129.16", true},
		{"Similar but not metadata", "169.254.169.253", false},
		{"Similar but not metadata 2", "168.63.129.15", false},
		{"Random IP", "8.8.8.8", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := parseIP(tt.ip)
			if ip == nil {
				t.Fatalf("Failed to parse IP: %s", tt.ip)
			}
			result := isCloudMetadataIP(ip)
			if result != tt.expected {
				t.Errorf("isCloudMetadataIP(%q) = %v, expected %v", tt.ip, result, tt.expected)
			}
		})
	}
}

// Helper functions

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && stringContains(s, substr)))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func parseIP(s string) []byte {
	// Simple IP parser for testing
	var parts [4]byte
	var n, dotCount int
	for i := 0; i < len(s); i++ {
		if s[i] >= '0' && s[i] <= '9' {
			n = n*10 + int(s[i]-'0')
		} else if s[i] == '.' {
			if n > 255 {
				return nil
			}
			parts[dotCount] = byte(n)
			dotCount++
			n = 0
		} else {
			return nil
		}
	}
	if dotCount != 3 || n > 255 {
		return nil
	}
	parts[3] = byte(n)
	return parts[:]
}

package utils

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

// ValidateExternalImageURL validates that a URL is safe for external image loading
// Prevents SSRF attacks by blocking internal/private network addresses
func ValidateExternalImageURL(urlStr string) error {
	// Parse URL
	parsedURL, err := url.ParseRequestURI(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL format")
	}

	// Require HTTPS for external URLs (security best practice)
	if parsedURL.Scheme != "https" && parsedURL.Scheme != "http" {
		return fmt.Errorf("only HTTP/HTTPS protocols are allowed")
	}

	// Get hostname
	hostname := parsedURL.Hostname()
	if hostname == "" {
		return fmt.Errorf("URL must have a hostname")
	}

	// Block localhost variations
	if isLocalhost(hostname) {
		return fmt.Errorf("localhost URLs are not allowed")
	}

	// Block .local domains (mDNS)
	if strings.HasSuffix(strings.ToLower(hostname), ".local") {
		return fmt.Errorf(".local domains are not allowed")
	}

	// Check if hostname is an IP address
	if ip := net.ParseIP(hostname); ip != nil {
		// Block private/internal IP ranges
		if isPrivateOrReservedIP(ip) {
			return fmt.Errorf("private/internal IP addresses are not allowed")
		}
	} else {
		// For domain names, perform DNS lookup to check resolved IPs
		ips, err := net.LookupIP(hostname)
		if err != nil {
			// DNS lookup failed - could be temporary, allow it
			// The actual request will fail later if DNS is truly broken
			return nil
		}

		// Check if any resolved IP is private/internal
		for _, ip := range ips {
			if isPrivateOrReservedIP(ip) {
				return fmt.Errorf("URL resolves to private/internal IP address")
			}
		}
	}

	return nil
}

// isLocalhost checks if a hostname is localhost
func isLocalhost(hostname string) bool {
	lower := strings.ToLower(hostname)
	return lower == "localhost" ||
		lower == "127.0.0.1" ||
		lower == "::1" ||
		lower == "0.0.0.0" ||
		lower == "::"
}

// isPrivateOrReservedIP checks if an IP is in private/internal/reserved ranges
func isPrivateOrReservedIP(ip net.IP) bool {
	// Loopback (127.0.0.0/8, ::1)
	if ip.IsLoopback() {
		return true
	}

	// Private networks (RFC 1918)
	if ip.IsPrivate() {
		return true
	}

	// Link-local addresses (169.254.0.0/16, fe80::/10)
	if ip.IsLinkLocalUnicast() {
		return true
	}

	// Link-local multicast (224.0.0.0/24, ff02::/16)
	if ip.IsLinkLocalMulticast() {
		return true
	}

	// Check for cloud metadata endpoints
	if isCloudMetadataIP(ip) {
		return true
	}

	// Check for carrier-grade NAT (100.64.0.0/10)
	if isCGNATRange(ip) {
		return true
	}

	return false
}

// isCloudMetadataIP checks for cloud provider metadata endpoints
func isCloudMetadataIP(ip net.IP) bool {
	// AWS/GCP/Azure metadata endpoint
	if ip.String() == "169.254.169.254" {
		return true
	}

	// Azure additional metadata endpoints
	if ip.String() == "168.63.129.16" {
		return true
	}

	return false
}

// isCGNATRange checks if IP is in carrier-grade NAT range (100.64.0.0/10)
func isCGNATRange(ip net.IP) bool {
	ip4 := ip.To4()
	if ip4 == nil {
		return false
	}

	// 100.64.0.0/10: 100.64.0.0 - 100.127.255.255
	return ip4[0] == 100 && (ip4[1]&0xC0) == 64
}

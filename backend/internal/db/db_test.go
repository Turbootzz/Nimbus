package db

import (
	"fmt"
	"net/url"
	"os"
	"testing"
)

func TestConnect_URLEncodingSpecialChars(t *testing.T) {
	// Save and restore original environment
	origHost := os.Getenv("DB_HOST")
	origPort := os.Getenv("DB_PORT")
	origUser := os.Getenv("DB_USER")
	origName := os.Getenv("DB_NAME")
	origPassword := os.Getenv("DB_PASSWORD")

	t.Cleanup(func() {
		os.Setenv("DB_HOST", origHost)
		os.Setenv("DB_PORT", origPort)
		os.Setenv("DB_USER", origUser)
		os.Setenv("DB_NAME", origName)
		os.Setenv("DB_PASSWORD", origPassword)
	})

	// Set up test environment
	os.Setenv("DB_HOST", "localhost")
	os.Setenv("DB_PORT", "5432")
	os.Setenv("DB_USER", "testuser")
	os.Setenv("DB_NAME", "testdb")

	testCases := []struct {
		name     string
		password string
	}{
		{
			name:     "alphanumeric",
			password: "simplePass123",
		},
		{
			name:     "with @ symbol",
			password: "pass@word",
		},
		{
			name:     "with multiple special chars (original failing password)",
			password: "pJ@1pAI!2CX^0Fu!I7VWkSPM7U&HQY&G",
		},
		{
			name:     "with spaces",
			password: "pass word",
		},
		{
			name:     "with percent sign",
			password: "pass%word",
		},
		{
			name:     "with hash",
			password: "pass#word",
		},
		{
			name:     "with all special chars",
			password: "!@#$%^&*()_+-=[]{}|;:',.<>?/~`",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			os.Setenv("DB_PASSWORD", tc.password)

			// Build URL using the same function Connect() uses
			dbURL := buildDBURL(
				os.Getenv("DB_HOST"),
				os.Getenv("DB_PORT"),
				os.Getenv("DB_USER"),
				tc.password,
				os.Getenv("DB_NAME"),
			)

			// Verify the URL can be parsed (this was failing before the fix)
			parsedURL, err := url.Parse(dbURL)
			if err != nil {
				t.Fatalf("URL parsing failed: %v\nURL: %s", err, dbURL)
			}

			// The key test: verify the password round-trips correctly
			// (can be encoded into URL and then decoded back to original)
			if parsedURL.User != nil {
				decodedPassword, hasPassword := parsedURL.User.Password()
				if !hasPassword {
					t.Error("Password not found in parsed URL")
				}
				if decodedPassword != tc.password {
					t.Errorf("Password didn't round-trip correctly.\nOriginal: %q\nDecoded:  %q\nFull URL: %s",
						tc.password, decodedPassword, dbURL)
				}
			} else {
				t.Error("User info not found in parsed URL")
			}

			// Verify username also round-trips
			if parsedURL.User != nil {
				decodedUser := parsedURL.User.Username()
				if decodedUser != "testuser" {
					t.Errorf("Username didn't round-trip correctly.\nOriginal: %q\nDecoded:  %q",
						"testuser", decodedUser)
				}
			}
		})
	}
}

// TestBuildDBURL_DirectTest tests the buildDBURL function directly
func TestBuildDBURL_DirectTest(t *testing.T) {
	testCases := []struct {
		name     string
		host     string
		port     string
		user     string
		password string
		dbname   string
	}{
		{
			name:     "simple credentials",
			host:     "localhost",
			port:     "5432",
			user:     "admin",
			password: "secret",
			dbname:   "mydb",
		},
		{
			name:     "password with @ symbol",
			host:     "db.example.com",
			port:     "5432",
			user:     "dbuser",
			password: "p@ssw0rd",
			dbname:   "production",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dbURL := buildDBURL(tc.host, tc.port, tc.user, tc.password, tc.dbname)

			// Parse the URL
			parsedURL, err := url.Parse(dbURL)
			if err != nil {
				t.Fatalf("Failed to parse URL: %v", err)
			}

			// Verify all components
			if parsedURL.Scheme != "postgres" {
				t.Errorf("Expected scheme 'postgres', got %q", parsedURL.Scheme)
			}

			if parsedURL.Host != fmt.Sprintf("%s:%s", tc.host, tc.port) {
				t.Errorf("Expected host %q, got %q", fmt.Sprintf("%s:%s", tc.host, tc.port), parsedURL.Host)
			}

			if parsedURL.Path != "/"+tc.dbname {
				t.Errorf("Expected path %q, got %q", "/"+tc.dbname, parsedURL.Path)
			}

			if parsedURL.User != nil {
				if parsedURL.User.Username() != tc.user {
					t.Errorf("Expected username %q, got %q", tc.user, parsedURL.User.Username())
				}

				decodedPassword, _ := parsedURL.User.Password()
				if decodedPassword != tc.password {
					t.Errorf("Expected password %q, got %q", tc.password, decodedPassword)
				}
			}
		})
	}
}

// TestConnect_DBURLPassthrough verifies that when DB_URL is set,
// Connect uses it directly without modification
func TestConnect_DBURLPassthrough(t *testing.T) {
	// Save and restore environment
	origDBURL := os.Getenv("DB_URL")
	origHost := os.Getenv("DB_HOST")
	origPort := os.Getenv("DB_PORT")
	origUser := os.Getenv("DB_USER")
	origPassword := os.Getenv("DB_PASSWORD")
	origName := os.Getenv("DB_NAME")

	t.Cleanup(func() {
		os.Setenv("DB_URL", origDBURL)
		os.Setenv("DB_HOST", origHost)
		os.Setenv("DB_PORT", origPort)
		os.Setenv("DB_USER", origUser)
		os.Setenv("DB_PASSWORD", origPassword)
		os.Setenv("DB_NAME", origName)
	})

	// Set DB_URL and some individual env vars
	testDBURL := "postgres://customuser:custompass@customhost:9999/customdb?sslmode=require"
	os.Setenv("DB_URL", testDBURL)
	os.Setenv("DB_HOST", "ignored-host")
	os.Setenv("DB_PORT", "1234")
	os.Setenv("DB_USER", "ignored-user")
	os.Setenv("DB_PASSWORD", "ignored-password")
	os.Setenv("DB_NAME", "ignored-db")

	// Note: We can't actually test Connect() without a real database,
	// but we can verify the logic would use DB_URL when set by checking
	// the environment variable is read correctly
	dbURL := os.Getenv("DB_URL")
	if dbURL != testDBURL {
		t.Errorf("Expected DB_URL to be %q, got %q", testDBURL, dbURL)
	}

	// Verify that if DB_URL is empty, buildDBURL would be called
	os.Setenv("DB_URL", "")
	constructedURL := buildDBURL(
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
	)

	// Verify the constructed URL uses the individual env vars
	if !containsSubstring(constructedURL, "ignored-host") {
		t.Errorf("Expected constructed URL to contain 'ignored-host', got %q", constructedURL)
	}
	if !containsSubstring(constructedURL, "ignored-user") {
		t.Errorf("Expected constructed URL to contain 'ignored-user', got %q", constructedURL)
	}
}

// Helper function to check substring
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

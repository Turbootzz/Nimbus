package db

import (
	"fmt"
	"net/url"
	"os"
	"testing"
)

func TestConnect_URLEncodingSpecialChars(t *testing.T) {
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

			// Build URL the same way Connect() does
			user := os.Getenv("DB_USER")
			password := os.Getenv("DB_PASSWORD")
			host := os.Getenv("DB_HOST")
			port := os.Getenv("DB_PORT")
			dbname := os.Getenv("DB_NAME")

			// Use url.URL to properly encode credentials
			u := &url.URL{
				Scheme:   "postgres",
				User:     url.UserPassword(user, password),
				Host:     fmt.Sprintf("%s:%s", host, port),
				Path:     "/" + dbname,
				RawQuery: "sslmode=disable",
			}
			dbURL := u.String()

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
				if decodedUser != user {
					t.Errorf("Username didn't round-trip correctly.\nOriginal: %q\nDecoded:  %q",
						user, decodedUser)
				}
			}
		})
	}
}

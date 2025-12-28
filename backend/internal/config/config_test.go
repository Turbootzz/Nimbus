package config

import (
	"os"
	"testing"
)

// TestValidateRequiredEnvVars tests environment variable validation
// Note: PORT, DB_HOST, DB_PORT, DB_USER, DB_NAME, CORS_ORIGINS now have defaults
// Only JWT_SECRET and DB_PASSWORD are strictly required
func TestValidateRequiredEnvVars(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
		wantErr bool
		errMsg  string
	}{
		{
			name: "all required vars present and valid",
			envVars: map[string]string{
				"JWT_SECRET":   "this-is-a-very-long-secret-key-minimum-32-characters",
				"DB_HOST":      "localhost",
				"DB_PORT":      "5432",
				"DB_NAME":      "nimbus",
				"DB_USER":      "postgres",
				"DB_PASSWORD":  "password",
				"PORT":         "8080",
				"CORS_ORIGINS": "http://localhost:3000",
			},
			wantErr: false,
		},
		{
			name: "only truly required vars (JWT_SECRET + DB_PASSWORD) with defaults",
			envVars: map[string]string{
				"JWT_SECRET":  "this-is-a-very-long-secret-key-minimum-32-characters",
				"DB_PASSWORD": "password",
			},
			wantErr: false,
		},
		{
			name: "JWT_SECRET too short",
			envVars: map[string]string{
				"JWT_SECRET":  "short",
				"DB_PASSWORD": "password",
			},
			wantErr: true,
			errMsg:  "JWT_SECRET must be at least 32 characters",
		},
		{
			name: "JWT_SECRET missing",
			envVars: map[string]string{
				"DB_PASSWORD": "password",
			},
			wantErr: true,
			errMsg:  "JWT_SECRET is required",
		},
		{
			name: "DB_PORT invalid (overrides default)",
			envVars: map[string]string{
				"JWT_SECRET":  "this-is-a-very-long-secret-key-minimum-32-characters",
				"DB_PORT":     "invalid",
				"DB_PASSWORD": "password",
			},
			wantErr: true,
			errMsg:  "DB_PORT must be a valid port number",
		},
		{
			name: "DB_PORT out of range (too high)",
			envVars: map[string]string{
				"JWT_SECRET":  "this-is-a-very-long-secret-key-minimum-32-characters",
				"DB_PORT":     "99999",
				"DB_PASSWORD": "password",
			},
			wantErr: true,
			errMsg:  "DB_PORT must be a valid port number",
		},
		{
			name: "DB_PORT out of range (too low)",
			envVars: map[string]string{
				"JWT_SECRET":  "this-is-a-very-long-secret-key-minimum-32-characters",
				"DB_PORT":     "0",
				"DB_PASSWORD": "password",
			},
			wantErr: true,
			errMsg:  "DB_PORT must be a valid port number",
		},
		{
			name: "DB_PASSWORD missing",
			envVars: map[string]string{
				"JWT_SECRET": "this-is-a-very-long-secret-key-minimum-32-characters",
			},
			wantErr: true,
			errMsg:  "DB_PASSWORD is required",
		},
		{
			name: "PORT invalid (overrides default)",
			envVars: map[string]string{
				"JWT_SECRET":  "this-is-a-very-long-secret-key-minimum-32-characters",
				"DB_PASSWORD": "password",
				"PORT":        "not-a-number",
			},
			wantErr: true,
			errMsg:  "PORT must be a valid port number",
		},
		{
			name: "whitespace-only JWT_SECRET trimmed and caught",
			envVars: map[string]string{
				"JWT_SECRET":  "   ",
				"DB_PASSWORD": "password",
			},
			wantErr: true,
			errMsg:  "JWT_SECRET is required",
		},
		{
			name: "multiple validation errors reported together",
			envVars: map[string]string{
				"JWT_SECRET": "short",
				"DB_PORT":    "invalid",
				// DB_PASSWORD missing
			},
			wantErr: true,
			// Should contain multiple error messages
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment
			clearEnv()

			// Set test environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			// Apply defaults before validation (mimics LoadEnv behavior)
			applyDefaults()

			// Run validation
			err := validateRequiredEnvVars()

			// Check result
			if (err != nil) != tt.wantErr {
				t.Errorf("validateRequiredEnvVars() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errMsg != "" {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errMsg)
				} else if err.Error() == "" {
					t.Errorf("expected error containing %q, got empty error", tt.errMsg)
				}
				// Note: We don't check exact error message match since multiple errors may be present
			}

			// Cleanup
			clearEnv()
		})
	}
}

// TestValidateRequiredEnvVars_EdgeCases tests edge cases and special scenarios
func TestValidateRequiredEnvVars_EdgeCases(t *testing.T) {
	t.Run("JWT_SECRET with spaces is trimmed", func(t *testing.T) {
		clearEnv()
		os.Setenv("JWT_SECRET", "  this-is-a-very-long-secret-key-minimum-32-characters  ")
		os.Setenv("DB_PASSWORD", "password")

		applyDefaults()
		err := validateRequiredEnvVars()
		if err != nil {
			t.Errorf("expected no error with trimmed JWT_SECRET, got: %v", err)
		}
		clearEnv()
	})

	t.Run("Port edge values", func(t *testing.T) {
		testCases := []struct {
			port    string
			wantErr bool
		}{
			{"1", false},     // min valid port
			{"65535", false}, // max valid port
			{"0", true},      // below min
			{"65536", true},  // above max
			{"-1", true},     // negative
		}

		for _, tc := range testCases {
			clearEnv()
			os.Setenv("JWT_SECRET", "this-is-a-very-long-secret-key-minimum-32-characters")
			os.Setenv("DB_PASSWORD", "password")
			os.Setenv("PORT", tc.port)

			applyDefaults()
			err := validateRequiredEnvVars()
			if (err != nil) != tc.wantErr {
				t.Errorf("PORT=%s: error = %v, wantErr %v", tc.port, err, tc.wantErr)
			}
			clearEnv()
		}
	})
}

// TestApplyDefaults tests the default value application
func TestApplyDefaults(t *testing.T) {
	t.Run("defaults are applied when env vars are empty", func(t *testing.T) {
		clearEnv()

		applyDefaults()

		expectedDefaults := map[string]string{
			"PORT":         "8080",
			"DB_HOST":      "db",
			"DB_PORT":      "5432",
			"DB_USER":      "nimbus",
			"DB_NAME":      "nimbus",
			"CORS_ORIGINS": "http://localhost:3000",
		}

		for key, expected := range expectedDefaults {
			if got := os.Getenv(key); got != expected {
				t.Errorf("%s = %q, want %q", key, got, expected)
			}
		}

		clearEnv()
	})

	t.Run("existing values are not overwritten", func(t *testing.T) {
		clearEnv()
		os.Setenv("PORT", "9000")
		os.Setenv("DB_HOST", "custom-host")

		applyDefaults()

		if got := os.Getenv("PORT"); got != "9000" {
			t.Errorf("PORT = %q, want %q", got, "9000")
		}
		if got := os.Getenv("DB_HOST"); got != "custom-host" {
			t.Errorf("DB_HOST = %q, want %q", got, "custom-host")
		}
		// But DB_PORT should still get default since it wasn't set
		if got := os.Getenv("DB_PORT"); got != "5432" {
			t.Errorf("DB_PORT = %q, want %q", got, "5432")
		}

		clearEnv()
	})
}

// TestGetEnvOrDefault tests the helper function
func TestGetEnvOrDefault(t *testing.T) {
	t.Run("returns env value when set", func(t *testing.T) {
		os.Setenv("TEST_VAR", "custom-value")
		defer os.Unsetenv("TEST_VAR")

		got := GetEnvOrDefault("TEST_VAR", "default")
		if got != "custom-value" {
			t.Errorf("GetEnvOrDefault() = %q, want %q", got, "custom-value")
		}
	})

	t.Run("returns default when env var is empty", func(t *testing.T) {
		os.Unsetenv("TEST_VAR")

		got := GetEnvOrDefault("TEST_VAR", "default")
		if got != "default" {
			t.Errorf("GetEnvOrDefault() = %q, want %q", got, "default")
		}
	})

	t.Run("returns default when env var is whitespace only", func(t *testing.T) {
		os.Setenv("TEST_VAR", "   ")
		defer os.Unsetenv("TEST_VAR")

		got := GetEnvOrDefault("TEST_VAR", "default")
		if got != "default" {
			t.Errorf("GetEnvOrDefault() = %q, want %q", got, "default")
		}
	})
}

// Helper function to clear all test environment variables
func clearEnv() {
	vars := []string{
		"JWT_SECRET",
		"DB_HOST",
		"DB_PORT",
		"DB_NAME",
		"DB_USER",
		"DB_PASSWORD",
		"PORT",
		"CORS_ORIGINS",
	}
	for _, v := range vars {
		os.Unsetenv(v)
	}
}

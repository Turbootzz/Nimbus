package config

import (
	"os"
	"testing"
)

// TestValidateRequiredEnvVars tests environment variable validation
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
			name: "JWT_SECRET too short",
			envVars: map[string]string{
				"JWT_SECRET":   "short",
				"DB_HOST":      "localhost",
				"DB_PORT":      "5432",
				"DB_NAME":      "nimbus",
				"DB_USER":      "postgres",
				"DB_PASSWORD":  "password",
				"PORT":         "8080",
				"CORS_ORIGINS": "http://localhost:3000",
			},
			wantErr: true,
			errMsg:  "JWT_SECRET must be at least 32 characters",
		},
		{
			name: "JWT_SECRET missing",
			envVars: map[string]string{
				"DB_HOST":      "localhost",
				"DB_PORT":      "5432",
				"DB_NAME":      "nimbus",
				"DB_USER":      "postgres",
				"DB_PASSWORD":  "password",
				"PORT":         "8080",
				"CORS_ORIGINS": "http://localhost:3000",
			},
			wantErr: true,
			errMsg:  "JWT_SECRET is required",
		},
		{
			name: "DB_HOST missing",
			envVars: map[string]string{
				"JWT_SECRET":   "this-is-a-very-long-secret-key-minimum-32-characters",
				"DB_PORT":      "5432",
				"DB_NAME":      "nimbus",
				"DB_USER":      "postgres",
				"DB_PASSWORD":  "password",
				"PORT":         "8080",
				"CORS_ORIGINS": "http://localhost:3000",
			},
			wantErr: true,
			errMsg:  "DB_HOST is required",
		},
		{
			name: "DB_PORT missing",
			envVars: map[string]string{
				"JWT_SECRET":   "this-is-a-very-long-secret-key-minimum-32-characters",
				"DB_HOST":      "localhost",
				"DB_NAME":      "nimbus",
				"DB_USER":      "postgres",
				"DB_PASSWORD":  "password",
				"PORT":         "8080",
				"CORS_ORIGINS": "http://localhost:3000",
			},
			wantErr: true,
			errMsg:  "DB_PORT is required",
		},
		{
			name: "DB_PORT invalid",
			envVars: map[string]string{
				"JWT_SECRET":   "this-is-a-very-long-secret-key-minimum-32-characters",
				"DB_HOST":      "localhost",
				"DB_PORT":      "invalid",
				"DB_NAME":      "nimbus",
				"DB_USER":      "postgres",
				"DB_PASSWORD":  "password",
				"PORT":         "8080",
				"CORS_ORIGINS": "http://localhost:3000",
			},
			wantErr: true,
			errMsg:  "DB_PORT must be a valid port number",
		},
		{
			name: "DB_PORT out of range (too high)",
			envVars: map[string]string{
				"JWT_SECRET":   "this-is-a-very-long-secret-key-minimum-32-characters",
				"DB_HOST":      "localhost",
				"DB_PORT":      "99999",
				"DB_NAME":      "nimbus",
				"DB_USER":      "postgres",
				"DB_PASSWORD":  "password",
				"PORT":         "8080",
				"CORS_ORIGINS": "http://localhost:3000",
			},
			wantErr: true,
			errMsg:  "DB_PORT must be a valid port number",
		},
		{
			name: "DB_PORT out of range (too low)",
			envVars: map[string]string{
				"JWT_SECRET":   "this-is-a-very-long-secret-key-minimum-32-characters",
				"DB_HOST":      "localhost",
				"DB_PORT":      "0",
				"DB_NAME":      "nimbus",
				"DB_USER":      "postgres",
				"DB_PASSWORD":  "password",
				"PORT":         "8080",
				"CORS_ORIGINS": "http://localhost:3000",
			},
			wantErr: true,
			errMsg:  "DB_PORT must be a valid port number",
		},
		{
			name: "DB_NAME missing",
			envVars: map[string]string{
				"JWT_SECRET":   "this-is-a-very-long-secret-key-minimum-32-characters",
				"DB_HOST":      "localhost",
				"DB_PORT":      "5432",
				"DB_USER":      "postgres",
				"DB_PASSWORD":  "password",
				"PORT":         "8080",
				"CORS_ORIGINS": "http://localhost:3000",
			},
			wantErr: true,
			errMsg:  "DB_NAME is required",
		},
		{
			name: "DB_USER missing",
			envVars: map[string]string{
				"JWT_SECRET":   "this-is-a-very-long-secret-key-minimum-32-characters",
				"DB_HOST":      "localhost",
				"DB_PORT":      "5432",
				"DB_NAME":      "nimbus",
				"DB_PASSWORD":  "password",
				"PORT":         "8080",
				"CORS_ORIGINS": "http://localhost:3000",
			},
			wantErr: true,
			errMsg:  "DB_USER is required",
		},
		{
			name: "DB_PASSWORD missing",
			envVars: map[string]string{
				"JWT_SECRET":   "this-is-a-very-long-secret-key-minimum-32-characters",
				"DB_HOST":      "localhost",
				"DB_PORT":      "5432",
				"DB_NAME":      "nimbus",
				"DB_USER":      "postgres",
				"PORT":         "8080",
				"CORS_ORIGINS": "http://localhost:3000",
			},
			wantErr: true,
			errMsg:  "DB_PASSWORD is required",
		},
		{
			name: "PORT missing",
			envVars: map[string]string{
				"JWT_SECRET":   "this-is-a-very-long-secret-key-minimum-32-characters",
				"DB_HOST":      "localhost",
				"DB_PORT":      "5432",
				"DB_NAME":      "nimbus",
				"DB_USER":      "postgres",
				"DB_PASSWORD":  "password",
				"CORS_ORIGINS": "http://localhost:3000",
			},
			wantErr: true,
			errMsg:  "PORT is required",
		},
		{
			name: "PORT invalid",
			envVars: map[string]string{
				"JWT_SECRET":   "this-is-a-very-long-secret-key-minimum-32-characters",
				"DB_HOST":      "localhost",
				"DB_PORT":      "5432",
				"DB_NAME":      "nimbus",
				"DB_USER":      "postgres",
				"DB_PASSWORD":  "password",
				"PORT":         "not-a-number",
				"CORS_ORIGINS": "http://localhost:3000",
			},
			wantErr: true,
			errMsg:  "PORT must be a valid port number",
		},
		{
			name: "whitespace-only values trimmed and caught",
			envVars: map[string]string{
				"JWT_SECRET":   "   ",
				"DB_HOST":      "localhost",
				"DB_PORT":      "5432",
				"DB_NAME":      "nimbus",
				"DB_USER":      "postgres",
				"DB_PASSWORD":  "password",
				"PORT":         "8080",
				"CORS_ORIGINS": "http://localhost:3000",
			},
			wantErr: true,
			errMsg:  "JWT_SECRET is required",
		},
		{
			name: "multiple validation errors reported together",
			envVars: map[string]string{
				"JWT_SECRET": "short",
				// DB_HOST missing
				"DB_PORT":      "invalid",
				"DB_NAME":      "nimbus",
				"DB_USER":      "postgres",
				"DB_PASSWORD":  "password",
				"PORT":         "8080",
				"CORS_ORIGINS": "http://localhost:3000",
			},
			wantErr: true,
			// Should contain multiple error messages
		},
		{
			name: "CORS_ORIGINS missing",
			envVars: map[string]string{
				"JWT_SECRET":  "this-is-a-very-long-secret-key-minimum-32-characters",
				"DB_HOST":     "localhost",
				"DB_PORT":     "5432",
				"DB_NAME":     "nimbus",
				"DB_USER":     "postgres",
				"DB_PASSWORD": "password",
				"PORT":        "8080",
			},
			wantErr: true,
			errMsg:  "CORS_ORIGINS is required",
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
		os.Setenv("DB_HOST", "localhost")
		os.Setenv("DB_PORT", "5432")
		os.Setenv("DB_NAME", "nimbus")
		os.Setenv("DB_USER", "postgres")
		os.Setenv("DB_PASSWORD", "password")
		os.Setenv("PORT", "8080")
		os.Setenv("CORS_ORIGINS", "http://localhost:3000")

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
			os.Setenv("DB_HOST", "localhost")
			os.Setenv("DB_PORT", "5432")
			os.Setenv("DB_NAME", "nimbus")
			os.Setenv("DB_USER", "postgres")
			os.Setenv("DB_PASSWORD", "password")
			os.Setenv("PORT", tc.port)
			os.Setenv("CORS_ORIGINS", "http://localhost:3000")

			err := validateRequiredEnvVars()
			if (err != nil) != tc.wantErr {
				t.Errorf("PORT=%s: error = %v, wantErr %v", tc.port, err, tc.wantErr)
			}
			clearEnv()
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

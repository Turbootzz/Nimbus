package services

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestAuthService_HashPassword(t *testing.T) {
	authService := NewAuthService()

	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{
			name:     "Valid password",
			password: "SecurePassword123!",
			wantErr:  false,
		},
		{
			name:     "Short password",
			password: "123",
			wantErr:  false,
		},
		{
			name:     "Long password",
			password: strings.Repeat("a", 72), // bcrypt max length
			wantErr:  false,
		},
		{
			name:     "Empty password",
			password: "",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := authService.HashPassword(tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("HashPassword() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				// Verify hash is not empty
				if hash == "" {
					t.Error("HashPassword() returned empty hash")
				}
				// Verify hash is different from password
				if hash == tt.password {
					t.Error("HashPassword() returned password as hash")
				}
				// Verify hash starts with bcrypt prefix
				if !strings.HasPrefix(hash, "$2a$") && !strings.HasPrefix(hash, "$2b$") {
					t.Error("HashPassword() returned invalid bcrypt hash format")
				}
			}
		})
	}
}

func TestAuthService_HashPassword_Uniqueness(t *testing.T) {
	authService := NewAuthService()
	password := "TestPassword123"

	// Hash the same password twice
	hash1, err1 := authService.HashPassword(password)
	hash2, err2 := authService.HashPassword(password)

	if err1 != nil || err2 != nil {
		t.Fatalf("HashPassword() failed: err1=%v, err2=%v", err1, err2)
	}

	// Hashes should be different (bcrypt uses random salt)
	if hash1 == hash2 {
		t.Error("HashPassword() produced identical hashes for same password (should use random salt)")
	}
}

func TestAuthService_ComparePassword(t *testing.T) {
	authService := NewAuthService()

	// Create a hash for testing
	password := "CorrectPassword123"
	hash, err := authService.HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	tests := []struct {
		name           string
		hashedPassword string
		password       string
		wantErr        bool
	}{
		{
			name:           "Correct password",
			hashedPassword: hash,
			password:       password,
			wantErr:        false,
		},
		{
			name:           "Wrong password",
			hashedPassword: hash,
			password:       "WrongPassword",
			wantErr:        true,
		},
		{
			name:           "Empty password",
			hashedPassword: hash,
			password:       "",
			wantErr:        true,
		},
		{
			name:           "Case sensitive - wrong case",
			hashedPassword: hash,
			password:       "correctpassword123",
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := authService.ComparePassword(tt.hashedPassword, tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("ComparePassword() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAuthService_GenerateToken(t *testing.T) {
	// Set a test secret
	os.Setenv("JWT_SECRET", "test-secret-for-jwt")
	defer os.Unsetenv("JWT_SECRET")

	authService := NewAuthService()

	tests := []struct {
		name    string
		userID  string
		email   string
		role    string
		wantErr bool
	}{
		{
			name:    "Valid admin user",
			userID:  "user-123",
			email:   "admin@example.com",
			role:    "admin",
			wantErr: false,
		},
		{
			name:    "Valid regular user",
			userID:  "user-456",
			email:   "user@example.com",
			role:    "user",
			wantErr: false,
		},
		{
			name:    "Empty user ID",
			userID:  "",
			email:   "test@example.com",
			role:    "user",
			wantErr: false, // GenerateToken doesn't validate input
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := authService.GenerateToken(tt.userID, tt.email, tt.role)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				// Verify token is not empty
				if token == "" {
					t.Error("GenerateToken() returned empty token")
				}
				// Verify token has 3 parts (header.payload.signature)
				parts := strings.Split(token, ".")
				if len(parts) != 3 {
					t.Errorf("GenerateToken() returned invalid JWT format, got %d parts", len(parts))
				}
			}
		})
	}
}

func TestAuthService_ValidateToken(t *testing.T) {
	// Set a test secret
	os.Setenv("JWT_SECRET", "test-secret-for-jwt")
	defer os.Unsetenv("JWT_SECRET")

	authService := NewAuthService()

	// Generate a valid token for testing
	userID := "user-123"
	email := "test@example.com"
	role := "admin"
	validToken, err := authService.GenerateToken(userID, email, role)
	if err != nil {
		t.Fatalf("Failed to generate test token: %v", err)
	}

	// Create an expired token
	expiredClaims := jwt.MapClaims{
		"user_id": userID,
		"email":   email,
		"role":    role,
		"exp":     time.Now().Add(-time.Hour).Unix(), // Expired 1 hour ago
		"iat":     time.Now().Add(-2 * time.Hour).Unix(),
	}
	expiredTokenObj := jwt.NewWithClaims(jwt.SigningMethodHS256, expiredClaims)
	expiredToken, _ := expiredTokenObj.SignedString([]byte("test-secret-for-jwt"))

	// Create a token with wrong signature
	wrongSecretAuth := &AuthService{jwtSecret: []byte("wrong-secret")}
	wrongSignatureToken, _ := wrongSecretAuth.GenerateToken(userID, email, role)

	tests := []struct {
		name        string
		token       string
		wantErr     bool
		checkClaims bool
	}{
		{
			name:        "Valid token",
			token:       validToken,
			wantErr:     false,
			checkClaims: true,
		},
		{
			name:    "Expired token",
			token:   expiredToken,
			wantErr: true,
		},
		{
			name:    "Invalid signature",
			token:   wrongSignatureToken,
			wantErr: true,
		},
		{
			name:    "Malformed token",
			token:   "not.a.valid.jwt.token",
			wantErr: true,
		},
		{
			name:    "Empty token",
			token:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := authService.ValidateToken(tt.token)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.checkClaims && claims != nil {
				// Verify claims contain expected fields
				if (*claims)["user_id"] != userID {
					t.Errorf("ValidateToken() user_id = %v, want %v", (*claims)["user_id"], userID)
				}
				if (*claims)["email"] != email {
					t.Errorf("ValidateToken() email = %v, want %v", (*claims)["email"], email)
				}
				if (*claims)["role"] != role {
					t.Errorf("ValidateToken() role = %v, want %v", (*claims)["role"], role)
				}
			}
		})
	}
}

func TestAuthService_GetUserIDFromToken(t *testing.T) {
	authService := NewAuthService()

	tests := []struct {
		name    string
		claims  jwt.MapClaims
		want    string
		wantErr bool
	}{
		{
			name: "Valid claims with user_id",
			claims: jwt.MapClaims{
				"user_id": "user-123",
				"email":   "test@example.com",
			},
			want:    "user-123",
			wantErr: false,
		},
		{
			name: "Missing user_id",
			claims: jwt.MapClaims{
				"email": "test@example.com",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "user_id is not a string",
			claims: jwt.MapClaims{
				"user_id": 12345, // integer instead of string
			},
			want:    "",
			wantErr: true,
		},
		{
			name:    "Empty claims",
			claims:  jwt.MapClaims{},
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := authService.GetUserIDFromToken(&tt.claims)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetUserIDFromToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetUserIDFromToken() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAuthService_TokenExpiration(t *testing.T) {
	os.Setenv("JWT_SECRET", "test-secret-for-jwt")
	defer os.Unsetenv("JWT_SECRET")

	authService := NewAuthService()

	// Generate a token
	token, err := authService.GenerateToken("user-123", "test@example.com", "user")
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Validate and check expiration
	claims, err := authService.ValidateToken(token)
	if err != nil {
		t.Fatalf("Failed to validate token: %v", err)
	}

	// Check expiration is set to ~24 hours from now
	exp, ok := (*claims)["exp"].(float64)
	if !ok {
		t.Fatal("Token expiration not found or wrong type")
	}

	expirationTime := time.Unix(int64(exp), 0)
	expectedExpiration := time.Now().Add(24 * time.Hour)
	timeDiff := expirationTime.Sub(expectedExpiration)

	// Allow 5 seconds of variance
	if timeDiff > 5*time.Second || timeDiff < -5*time.Second {
		t.Errorf("Token expiration = %v, want ~%v (diff: %v)", expirationTime, expectedExpiration, timeDiff)
	}
}

func TestAuthService_DefaultSecret(t *testing.T) {
	// Ensure no JWT_SECRET is set
	os.Unsetenv("JWT_SECRET")

	authService := NewAuthService()

	// Should use default secret
	if string(authService.jwtSecret) != "default-secret-change-in-production" {
		t.Errorf("NewAuthService() with no env var, jwtSecret = %v, want default", string(authService.jwtSecret))
	}

	// Should still be able to generate and validate tokens
	token, err := authService.GenerateToken("user-123", "test@example.com", "user")
	if err != nil {
		t.Errorf("GenerateToken() with default secret failed: %v", err)
	}

	_, err = authService.ValidateToken(token)
	if err != nil {
		t.Errorf("ValidateToken() with default secret failed: %v", err)
	}
}

func TestAuthService_FullWorkflow(t *testing.T) {
	os.Setenv("JWT_SECRET", "test-secret-for-jwt")
	defer os.Unsetenv("JWT_SECRET")

	authService := NewAuthService()

	// Simulate user registration flow
	password := "SecurePassword123!"
	hashedPassword, err := authService.HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() failed: %v", err)
	}

	// Simulate user login - correct password
	err = authService.ComparePassword(hashedPassword, password)
	if err != nil {
		t.Errorf("ComparePassword() failed for correct password: %v", err)
	}

	// Generate token after successful login
	userID := "user-123"
	email := "user@example.com"
	role := "user"
	token, err := authService.GenerateToken(userID, email, role)
	if err != nil {
		t.Fatalf("GenerateToken() failed: %v", err)
	}

	// Validate token (simulating protected endpoint)
	claims, err := authService.ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken() failed: %v", err)
	}

	// Extract user ID from claims
	extractedUserID, err := authService.GetUserIDFromToken(claims)
	if err != nil {
		t.Fatalf("GetUserIDFromToken() failed: %v", err)
	}

	if extractedUserID != userID {
		t.Errorf("Extracted user ID = %v, want %v", extractedUserID, userID)
	}

	// Verify all claims
	if (*claims)["email"] != email {
		t.Errorf("Token email = %v, want %v", (*claims)["email"], email)
	}
	if (*claims)["role"] != role {
		t.Errorf("Token role = %v, want %v", (*claims)["role"], role)
	}

	// Simulate failed login - wrong password
	err = authService.ComparePassword(hashedPassword, "WrongPassword")
	if err == nil {
		t.Error("ComparePassword() should fail for wrong password")
	}
}

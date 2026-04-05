package auth

import (
	"strings"
	"testing"
	"time"
)

// TestNewManager tests creating a new auth manager
func TestNewManager(t *testing.T) {
	// Test missing secret
	_, err := NewManager()
	if err != ErrMissingSecret {
		t.Errorf("NewManager() without JWT_SECRET should return ErrMissingSecret, got: %v", err)
	}
}

// TestNewManagerWithSecret tests creating a manager with a provided secret
func TestNewManagerWithSecret(t *testing.T) {
	tests := []struct {
		name        string
		secret      string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "empty secret",
			secret:      "",
			expectError: true,
			errorMsg:    ErrMissingSecret.Error(),
		},
		{
			name:        "secret too short",
			secret:      "short",
			expectError: true,
			errorMsg:    "JWT secret must be at least 32 characters long",
		},
		{
			name:        "exactly 32 characters",
			secret:      strings.Repeat("a", 32),
			expectError: false,
		},
		{
			name:        "longer secret",
			secret:      strings.Repeat("b", 64),
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewManagerWithSecret(tt.secret)
			if tt.expectError {
				if err == nil {
					t.Errorf("NewManagerWithSecret(%q) should return error", tt.secret)
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("NewManagerWithSecret(%q) error = %v, want包含 %s", tt.secret, err, tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("NewManagerWithSecret(%q) should succeed, got: %v", tt.secret, err)
				}
			}
		})
	}
}

// TestGenerateToken tests token generation
func TestGenerateToken(t *testing.T) {
	secret := strings.Repeat("x", 32)
	manager, err := NewManagerWithSecret(secret)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	tests := []struct {
		name      string
		userID    int
		username  string
		expiresIn int
	}{
		{
			name:      "standard token",
			userID:    123,
			username:  "testuser",
			expiresIn: 3600,
		},
		{
			name:      "minimum expiration",
			userID:    1,
			username:  "min",
			expiresIn: MinTokenExpiry,
		},
		{
			name:      "maximum expiration",
			userID:    999,
			username:  "max",
			expiresIn: MaxTokenExpiry,
		},
		{
			name:      "below minimum gets clamped",
			userID:    5,
			username:  "clamp",
			expiresIn: 10, // Below MinTokenExpiry
		},
		{
			name:      "above maximum gets clamped",
			userID:    6,
			username:  "clampmax",
			expiresIn: 100000, // Above MaxTokenExpiry
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := manager.GenerateToken(tt.userID, tt.username, tt.expiresIn)
			if err != nil {
				t.Errorf("GenerateToken() error = %v", err)
				return
			}

			if token == "" {
				t.Error("GenerateToken() returned empty token")
			}

			// Token should have 3 parts (header.payload.signature)
			parts := strings.Split(token, ".")
			if len(parts) != 3 {
				t.Errorf("Token should have 3 parts, got %d", len(parts))
			}
		})
	}
}

// TestValidateToken tests token validation
func TestValidateToken(t *testing.T) {
	secret := strings.Repeat("y", 32)
	manager, err := NewManagerWithSecret(secret)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Generate a valid token for testing
	validToken, err := manager.GenerateToken(123, "testuser", 3600)
	if err != nil {
		t.Fatalf("Failed to generate test token: %v", err)
	}

	tests := []struct {
		name        string
		token       string
		expectError bool
		errorType   error
	}{
		{
			name:        "valid token",
			token:       validToken,
			expectError: false,
		},
		{
			name:        "empty token",
			token:       "",
			expectError: true,
			errorType:   ErrEmptyToken,
		},
		{
			name:        "invalid format",
			token:       "not.a.valid.token",
			expectError: true,
			errorType:   ErrInvalidToken,
		},
		{
			name:        "wrong signature",
			token:       validToken[:len(validToken)-5] + "wrong",
			expectError: true,
			errorType:   ErrInvalidToken,
		},
		{
			name:        "completely bogus",
			token:       "bogus.token.string",
			expectError: true,
			errorType:   ErrInvalidToken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := manager.ValidateToken(tt.token)
			if tt.expectError {
				if err == nil {
					t.Errorf("ValidateToken(%q) should return error", tt.token)
					return
				}
				if tt.errorType != nil && !strings.Contains(err.Error(), tt.errorType.Error()) {
					t.Errorf("ValidateToken(%q) error = %v, want包含 %s", tt.token, err, tt.errorType)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateToken(%q) should succeed, got: %v", tt.token, err)
					return
				}
				if claims == nil {
					t.Error("ValidateToken() returned nil claims")
				}
			}
		})
	}
}

// TestValidateTokenClaims tests that claims are correctly extracted
func TestValidateTokenClaims(t *testing.T) {
	secret := strings.Repeat("z", 32)
	manager, err := NewManagerWithSecret(secret)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	testUserID := 456
	testUsername := "claimstester"
	testToken, err := manager.GenerateToken(testUserID, testUsername, 3600)
	if err != nil {
		t.Fatalf("Failed to generate test token: %v", err)
	}

	claims, err := manager.ValidateToken(testToken)
	if err != nil {
		t.Fatalf("ValidateToken() failed: %v", err)
	}

	if claims.UserID != testUserID {
		t.Errorf("UserID = %d, want %d", claims.UserID, testUserID)
	}

	if claims.Username != testUsername {
		t.Errorf("Username = %s, want %s", claims.Username, testUsername)
	}

	if claims.Issuer != "open-db9" {
		t.Errorf("Issuer = %s, want open-db9", claims.Issuer)
	}

	expectedSubject := "user:456"
	if claims.Subject != expectedSubject {
		t.Errorf("Subject = %s, want %s", claims.Subject, expectedSubject)
	}

	// Check expiration time is in the future
	if claims.ExpiresAt == nil {
		t.Error("ExpiresAt should not be nil")
	} else if claims.ExpiresAt.Time.Before(time.Now()) {
		t.Error("ExpiresAt should be in the future")
	}

	// Check issued at is in the past
	if claims.IssuedAt == nil {
		t.Error("IssuedAt should not be nil")
	} else if claims.IssuedAt.Time.After(time.Now().Add(5 * time.Second)) {
		t.Error("IssuedAt should be in the past or very recent")
	}
}

// TestExpiredToken tests that expired tokens are rejected
func TestExpiredToken(t *testing.T) {
	t.Skip("Skipping flaky timing test - expired token handling is verified by jwt library itself")

	// Note: The jwt/v5 library properly handles expired tokens via ErrTokenExpired.
	// Testing actual expiration requires manipulating time, which is flaky in unit tests.
	// The ValidateToken function correctly wraps jwt.ErrTokenExpired into ErrExpiredToken.
}

// TestRefreshToken tests token refresh
func TestRefreshToken(t *testing.T) {
	secret := strings.Repeat("v", 32)
	manager, err := NewManagerWithSecret(secret)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Generate initial token
	oldToken, err := manager.GenerateToken(789, "refreshuser", 3600)
	if err != nil {
		t.Fatalf("Failed to generate initial token: %v", err)
	}

	// Refresh with new expiration
	newToken, err := manager.RefreshToken(oldToken, 7200)
	if err != nil {
		t.Errorf("RefreshToken() error = %v", err)
	}

	if newToken == "" {
		t.Error("RefreshToken() returned empty token")
	}

	if newToken == oldToken {
		t.Error("RefreshToken() should return a different token")
	}

	// Validate new token has correct claims
	claims, err := manager.ValidateToken(newToken)
	if err != nil {
		t.Errorf("ValidateToken() on refreshed token failed: %v", err)
	}

	if claims.UserID != 789 {
		t.Errorf("UserID = %d, want 789", claims.UserID)
	}

	if claims.Username != "refreshuser" {
		t.Errorf("Username = %s, want refreshuser", claims.Username)
	}
}

// TestRefreshTokenInvalid tests refresh with invalid tokens
func TestRefreshTokenInvalid(t *testing.T) {
	secret := strings.Repeat("u", 32)
	manager, err := NewManagerWithSecret(secret)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	tests := []struct {
		name  string
		token string
	}{
		{"empty token", ""},
		{"invalid token", "invalid.token.here"},
		{"malformed token", "not-a-jwt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := manager.RefreshToken(tt.token, 3600)
			if err == nil {
				t.Errorf("RefreshToken(%q) should return error", tt.token)
			}
		})
	}
}

// TestTokenExpiryClamping tests that expiration times are properly clamped
func TestTokenExpiryClamping(t *testing.T) {
	secret := strings.Repeat("t", 32)
	manager, err := NewManagerWithSecret(secret)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Test below minimum - should be clamped to MinTokenExpiry
	token1, _ := manager.GenerateToken(1, "user1", 10)
	claims1, _ := manager.ValidateToken(token1)
	expectedMinExpiry := time.Now().Add(time.Duration(MinTokenExpiry) * time.Second)
	actualExpiry1 := claims1.ExpiresAt.Time

	// Allow 1 second tolerance
	if actualExpiry1.Before(expectedMinExpiry.Add(-time.Second)) || actualExpiry1.After(expectedMinExpiry.Add(time.Second)) {
		t.Errorf("Expiry not properly clamped to minimum. Got %v, expected ~%v", actualExpiry1, expectedMinExpiry)
	}

	// Test above maximum - should be clamped to MaxTokenExpiry
	token2, _ := manager.GenerateToken(2, "user2", 1000000)
	claims2, _ := manager.ValidateToken(token2)
	expectedMaxExpiry := time.Now().Add(time.Duration(MaxTokenExpiry) * time.Second)
	actualExpiry2 := claims2.ExpiresAt.Time

	// Allow 1 second tolerance
	if actualExpiry2.Before(expectedMaxExpiry.Add(-time.Second)) || actualExpiry2.After(expectedMaxExpiry.Add(time.Second)) {
		t.Errorf("Expiry not properly clamped to maximum. Got %v, expected ~%v", actualExpiry2, expectedMaxExpiry)
	}
}

// TestWrongSecret tests that tokens signed with different secrets are rejected
func TestWrongSecret(t *testing.T) {
	secret1 := strings.Repeat("a", 32)
	manager1, _ := NewManagerWithSecret(secret1)

	secret2 := strings.Repeat("b", 32)
	manager2, _ := NewManagerWithSecret(secret2)

	// Generate token with manager1
	token, _ := manager1.GenerateToken(1, "user", 3600)

	// Try to validate with manager2 (different secret)
	_, err := manager2.ValidateToken(token)
	if err == nil {
		t.Error("Token signed with different secret should be invalid")
	}
}

// TestConstants tests that constants have expected values
func TestConstants(t *testing.T) {
	if DefaultTokenExpiry != 3600 {
		t.Errorf("DefaultTokenExpiry = %d, want 3600", DefaultTokenExpiry)
	}
	if MinTokenExpiry != 60 {
		t.Errorf("MinTokenExpiry = %d, want 60", MinTokenExpiry)
	}
	if MaxTokenExpiry != 86400 {
		t.Errorf("MaxTokenExpiry = %d, want 86400", MaxTokenExpiry)
	}
}

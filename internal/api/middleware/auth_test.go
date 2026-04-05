package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/open-db9/db9/internal/auth"
)

// TestAuthMiddleware_ValidToken tests that valid tokens pass through
func TestAuthMiddleware_ValidToken(t *testing.T) {
	// Create auth manager with test secret
	authManager, err := auth.NewManagerWithSecret("test-secret-key-32-characters-long!!")
	if err != nil {
		t.Fatalf("Failed to create auth manager: %v", err)
	}

	// Generate a valid token
	token, err := authManager.GenerateToken(123, "testuser", 3600)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Create a test handler that checks if user context is set
	testHandler := func(w http.ResponseWriter, r *http.Request) {
		userID, ok := GetUserID(r)
		if !ok {
			t.Error("User ID not found in context")
			return
		}

		username, ok := GetUsername(r)
		if !ok {
			t.Error("Username not found in context")
			return
		}

		if userID != 123 {
			t.Errorf("Expected user ID 123, got %d", userID)
		}

		if username != "testuser" {
			t.Errorf("Expected username 'testuser', got '%s'", username)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}

	// Create middleware with auth
	middleware := AuthMiddleware(authManager, testHandler)

	// Create request with valid token
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	// Create response recorder
	rr := httptest.NewRecorder()

	// Call middleware
	middleware.ServeHTTP(rr, req)

	// Check status code
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	if rr.Body.String() != "OK" {
		t.Errorf("Expected body 'OK', got '%s'", rr.Body.String())
	}
}

// TestAuthMiddleware_MissingToken tests that missing tokens return 401
func TestAuthMiddleware_MissingToken(t *testing.T) {
	authManager, err := auth.NewManagerWithSecret("test-secret-key-32-characters-long!!")
	if err != nil {
		t.Fatalf("Failed to create auth manager: %v", err)
	}

	testHandler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	middleware := AuthMiddleware(authManager, testHandler)

	// Create request without Authorization header
	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	middleware.ServeHTTP(rr, req)

	// Check status code
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rr.Code)
	}
}

// TestAuthMiddleware_InvalidToken tests that invalid tokens return 401
func TestAuthMiddleware_InvalidToken(t *testing.T) {
	authManager, err := auth.NewManagerWithSecret("test-secret-key-32-characters-long!!")
	if err != nil {
		t.Fatalf("Failed to create auth manager: %v", err)
	}

	testHandler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	middleware := AuthMiddleware(authManager, testHandler)

	tests := []struct {
		name        string
		authHeader  string
		expectError bool
	}{
		{
			name:        "invalid token format",
			authHeader:  "Bearer invalid.token.here",
			expectError: true,
		},
		{
			name:        "missing Bearer prefix",
			authHeader:  "valid-token-but-no-bearer",
			expectError: true,
		},
		{
			name:        "empty Bearer token",
			authHeader:  "Bearer ",
			expectError: true,
		},
		{
			name:        "malformed token",
			authHeader:  "Bearer not-a-jwt",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("Authorization", tt.authHeader)
			rr := httptest.NewRecorder()

			middleware.ServeHTTP(rr, req)

			if tt.expectError && rr.Code != http.StatusUnauthorized {
				t.Errorf("Expected status 401, got %d", rr.Code)
			}
		})
	}
}

// TestAuthMiddleware_WrongSecret tests that tokens signed with different secret are rejected
func TestAuthMiddleware_WrongSecret(t *testing.T) {
	// Create manager1 for signing
	manager1, err := auth.NewManagerWithSecret("test-secret-key-32-characters-long!!")
	if err != nil {
		t.Fatalf("Failed to create auth manager 1: %v", err)
	}

	// Create manager2 for validation (different secret)
	manager2, err := auth.NewManagerWithSecret("different-secret-key-32-characters!")
	if err != nil {
		t.Fatalf("Failed to create auth manager 2: %v", err)
	}

	// Generate token with manager1
	token, err := manager1.GenerateToken(123, "testuser", 3600)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	testHandler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	// Create middleware with manager2 (different secret)
	middleware := AuthMiddleware(manager2, testHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	middleware.ServeHTTP(rr, req)

	// Should be rejected
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401 for token signed with different secret, got %d", rr.Code)
	}
}

// TestGetUserID tests extracting user ID from context
func TestGetUserID(t *testing.T) {
	tests := []struct {
		name      string
		setupCtx  func(context.Context) context.Context
		expectOK  bool
		expectID  int
	}{
		{
			name: "valid user ID in context",
			setupCtx: func(ctx context.Context) context.Context {
				return context.WithValue(ctx, UserIDKey, 456)
			},
			expectOK: true,
			expectID: 456,
		},
		{
			name:     "no user ID in context",
			setupCtx: func(ctx context.Context) context.Context { return ctx },
			expectOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			ctx := tt.setupCtx(req.Context())
			req = req.WithContext(ctx)

			userID, ok := GetUserID(req)

			if ok != tt.expectOK {
				t.Errorf("Expected ok=%v, got %v", tt.expectOK, ok)
			}

			if tt.expectOK && userID != tt.expectID {
				t.Errorf("Expected user ID %d, got %d", tt.expectID, userID)
			}
		})
	}
}

// TestGetUsername tests extracting username from context
func TestGetUsername(t *testing.T) {
	tests := []struct {
		name           string
		setupCtx       func(context.Context) context.Context
		expectOK       bool
		expectUsername string
	}{
		{
			name: "valid username in context",
			setupCtx: func(ctx context.Context) context.Context {
				return context.WithValue(ctx, UsernameKey, "testuser")
			},
			expectOK:       true,
			expectUsername: "testuser",
		},
		{
			name:     "no username in context",
			setupCtx: func(ctx context.Context) context.Context { return ctx },
			expectOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			ctx := tt.setupCtx(req.Context())
			req = req.WithContext(ctx)

			username, ok := GetUsername(req)

			if ok != tt.expectOK {
				t.Errorf("Expected ok=%v, got %v", tt.expectOK, ok)
			}

			if tt.expectOK && username != tt.expectUsername {
				t.Errorf("Expected username '%s', got '%s'", tt.expectUsername, username)
			}
		})
	}
}

// TestOptionalAuth tests optional authentication middleware
func TestOptionalAuth(t *testing.T) {
	authManager, err := auth.NewManagerWithSecret("test-secret-key-32-characters-long!!")
	if err != nil {
		t.Fatalf("Failed to create auth manager: %v", err)
	}

	testHandler := func(w http.ResponseWriter, r *http.Request) {
		// Check if user context is set
		_, ok := GetUserID(r)
		if ok {
			w.Write([]byte("authenticated"))
		} else {
			w.Write([]byte("unauthenticated"))
		}
	}

	middleware := OptionalAuth(authManager, testHandler)

	tests := []struct {
		name           string
		authHeader     string
		expectBody     string
	}{
		{
			name:       "with valid token",
			authHeader: "Bearer " + func() string {
				token, _ := authManager.GenerateToken(123, "testuser", 3600)
				return token
			}(),
			expectBody: "authenticated",
		},
		{
			name:       "without token",
			authHeader: "",
			expectBody: "unauthenticated",
		},
		{
			name:       "with invalid token",
			authHeader: "Bearer invalid.token",
			expectBody: "unauthenticated",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			rr := httptest.NewRecorder()
			middleware.ServeHTTP(rr, req)

			if rr.Body.String() != tt.expectBody {
				t.Errorf("Expected body '%s', got '%s'", tt.expectBody, rr.Body.String())
			}
		})
	}
}

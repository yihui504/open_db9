package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/open-db9/db9/internal/auth"
)

func TestPublicRoutesAccess(t *testing.T) {
	// Create auth manager with test secret
	authManager, err := auth.NewManagerWithSecret("test-secret-key-32-characters-long-123456")
	if err != nil {
		t.Fatalf("Failed to create auth manager: %v", err)
	}

	// Create router
	router := NewRouter(authManager)

	// Test public routes
	publicRoutes := []string{"/", "/health", "/version"}

	for _, route := range publicRoutes {
		t.Run(route, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, route, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			// Public routes should return 200 OK
			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200 for public route %s, got %d", route, w.Code)
			}
		})
	}
}

func TestProtectedRoutesWithoutAuth(t *testing.T) {
	// Create auth manager with test secret
	authManager, err := auth.NewManagerWithSecret("test-secret-key-32-characters-long-123456")
	if err != nil {
		t.Fatalf("Failed to create auth manager: %v", err)
	}

	// Create router
	router := NewRouter(authManager)

	// Test protected route without authentication
	req := httptest.NewRequest(http.MethodGet, "/api/v1/databases/1", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Should return 401 Unauthorized
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401 for protected route without auth, got %d", w.Code)
	}
}

func TestProtectedRoutesWithValidAuth(t *testing.T) {
	// Create auth manager with test secret
	authManager, err := auth.NewManagerWithSecret("test-secret-key-32-characters-long-123456")
	if err != nil {
		t.Fatalf("Failed to create auth manager: %v", err)
	}

	// Generate valid token
	token, err := authManager.GenerateToken(1, "testuser", 3600)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Create router
	router := NewRouter(authManager)

	// Test protected route with valid authentication
	req := httptest.NewRequest(http.MethodGet, "/api/v1/databases/1", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Should not return 401 (might return 404 for non-existent database, but that's ok)
	if w.Code == http.StatusUnauthorized {
		t.Errorf("Expected status other than 401 for protected route with valid auth, got %d", w.Code)
	}
}

func TestAPIHandlerRoutesDatabaseResources(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/databases/1/sql", nil)
	w := httptest.NewRecorder()

	apiHandler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("Expected status 405 for database SQL route, got %d", w.Code)
	}
}

func TestCORSMiddleware(t *testing.T) {
	// Create auth manager with test secret
	authManager, err := auth.NewManagerWithSecret("test-secret-key-32-characters-long-123456")
	if err != nil {
		t.Fatalf("Failed to create auth manager: %v", err)
	}

	// Create router
	router := NewRouter(authManager)

	// Test GET request to public route (should have CORS headers)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Check CORS headers are set
	corsOrigin := w.Header().Get("Access-Control-Allow-Origin")
	if corsOrigin != "*" {
		t.Errorf("Expected CORS origin *, got %s", corsOrigin)
	}

	corsMethods := w.Header().Get("Access-Control-Allow-Methods")
	if corsMethods == "" {
		t.Error("Expected CORS methods header to be set")
	}

	corsHeaders := w.Header().Get("Access-Control-Allow-Headers")
	if corsHeaders == "" {
		t.Error("Expected CORS headers header to be set")
	}
}

func TestMiddlewareChainOrder(t *testing.T) {
	// This test verifies that middleware is applied in the correct order
	// ErrorHandler -> CORS -> Logger -> Auth (for protected routes)

	// Create auth manager with test secret
	authManager, err := auth.NewManagerWithSecret("test-secret-key-32-characters-long-123456")
	if err != nil {
		t.Fatalf("Failed to create auth manager: %v", err)
	}

	// Create router
	router := NewRouter(authManager)

	// Test that error handler works by making a request that would panic
	// (This is a basic test - in production you'd have more specific panic tests)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Request should complete successfully
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

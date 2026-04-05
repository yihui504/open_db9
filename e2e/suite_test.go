package e2e

import (
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
	"time"

	"github.com/open-db9/db9/internal/api/router"
	"github.com/open-db9/db9/internal/auth"
)

// TestAPISmokeTest runs a basic smoke test of the API
func TestAPISmokeTest(t *testing.T) {
	// Skip E2E tests on Windows due to testcontainers limitations
	if runtime.GOOS == "windows" {
		t.Skip("E2E tests require Docker with testcontainers-go, which is not fully supported on Windows")
	}

	// Setup test environment
	env := SetupTestEnvironment(t)
	defer TeardownTestEnvironment(t, env)

	// Create router
	authManager, err := auth.NewManagerWithSecret("test-secret-for-e2e-tests-only-32chars!!")
	if err != nil {
		t.Fatalf("Failed to create auth manager: %v", err)
	}
	mux := router.NewRouter(authManager)
	server := httptest.NewServer(mux)
	defer server.Close()

	client := &http.Client{Timeout: 10 * time.Second}

	t.Run("Health endpoint", func(t *testing.T) {
		resp, err := client.Get(server.URL + "/health")
		if err != nil {
			t.Fatalf("Failed to call health endpoint: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
	})

	t.Run("Version endpoint", func(t *testing.T) {
		resp, err := client.Get(server.URL + "/version")
		if err != nil {
			t.Fatalf("Failed to call version endpoint: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
	})
}

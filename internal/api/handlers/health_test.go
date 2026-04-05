package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestHealthHandler(t *testing.T) {
	// Create a test storage directory
	testStoragePath := filepath.Join(os.TempDir(), "db9_health_test")
	defer os.RemoveAll(testStoragePath)

	// Create test request
	req, err := http.NewRequest("GET", "/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create response recorder
	rr := httptest.NewRecorder()

	// Call the handler
	HealthHandler(rr, req)

	// Check status code - accept both OK and ServiceUnavailable (degraded)
	// since FS9 service may not be available in test environment
	if status := rr.Code; status != http.StatusOK && status != http.StatusServiceUnavailable {
		t.Errorf("handler returned wrong status code: got %v want %v or %v", status, http.StatusOK, http.StatusServiceUnavailable)
	}

	// Check content type
	if contentType := rr.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("handler returned wrong content type: got %v want %v", contentType, "application/json")
	}
}

func TestCheckStoragePathAccessibility(t *testing.T) {
	// Test with valid temporary directory
	validPath := filepath.Join(os.TempDir(), "db9_storage_test")
	defer os.RemoveAll(validPath)

	accessible, err := checkStoragePathAccessibility(validPath)
	if err != nil {
		t.Errorf("checkStoragePathAccessibility failed: %v", err)
	}
	if !accessible {
		t.Error("Expected storage path to be accessible")
	}

	// Test with invalid path (if we can create an invalid path scenario)
	// Note: On most systems, we can't create truly invalid paths, so we'll skip this test
}

func TestCheckBasicFileOperations(t *testing.T) {
	// Create test directory
	testDir := filepath.Join(os.TempDir(), "db9_ops_test")
	defer os.RemoveAll(testDir)

	// Test basic operations
	success, err := checkBasicFileOperations(testDir)
	if err != nil {
		t.Errorf("checkBasicFileOperations failed: %v", err)
	}
	if !success {
		t.Error("Expected basic file operations to succeed")
	}
}

func TestCheckFS9ServiceWithRetry(t *testing.T) {
	ctx := context.Background()
	config := HealthCheckConfig{
		MaxRetries:        2, // Use fewer retries for testing
		InitialRetryDelay: 100 * time.Millisecond,
		FS9ServiceURL:     "http://localhost:9999", // Use non-existent service
	}

	// This should fail since we're using a non-existent service
	reachable, err := checkFS9ServiceWithRetry(ctx, config)
	if reachable {
		t.Error("Expected FS9 service check to fail for non-existent service")
	}
	if err == nil {
		t.Error("Expected error from FS9 service check")
	}
}

func TestDetermineOverallStatus(t *testing.T) {
	tests := []struct {
		name           string
		fsHealth       FilesystemHealth
		expectedStatus HealthStatus
	}{
		{
			name: "All healthy",
			fsHealth: FilesystemHealth{
				StoragePathAccessible: true,
				FS9ServiceReachable:   true,
				BasicOperationsWork:   true,
			},
			expectedStatus: HealthStatusHealthy,
		},
		{
			name: "Storage path not accessible",
			fsHealth: FilesystemHealth{
				StoragePathAccessible: false,
				FS9ServiceReachable:   true,
				BasicOperationsWork:   true,
			},
			expectedStatus: HealthStatusUnhealthy,
		},
		{
			name: "FS9 service not reachable",
			fsHealth: FilesystemHealth{
				StoragePathAccessible: true,
				FS9ServiceReachable:   false,
				BasicOperationsWork:   true,
			},
			expectedStatus: HealthStatusDegraded,
		},
		{
			name: "Basic operations failed",
			fsHealth: FilesystemHealth{
				StoragePathAccessible: true,
				FS9ServiceReachable:   true,
				BasicOperationsWork:   false,
			},
			expectedStatus: HealthStatusDegraded,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := determineOverallStatus(tt.fsHealth)
			if status != tt.expectedStatus {
				t.Errorf("determineOverallStatus() = %v, want %v", status, tt.expectedStatus)
			}
		})
	}
}

func TestDefaultHealthCheckConfig(t *testing.T) {
	config := DefaultHealthCheckConfig()

	if config.MaxRetries != 3 {
		t.Errorf("Expected MaxRetries to be 3, got %d", config.MaxRetries)
	}

	if config.InitialRetryDelay != 1*time.Second {
		t.Errorf("Expected InitialRetryDelay to be 1s, got %v", config.InitialRetryDelay)
	}

	if config.StoragePath != "./data/storage" {
		t.Errorf("Expected StoragePath to be './data/storage', got %s", config.StoragePath)
	}
}
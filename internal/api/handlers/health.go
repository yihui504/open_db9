package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/open-db9/db9/pkg/logger"
)

// HealthStatus represents the overall health status
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusDegraded  HealthStatus = "degraded"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
)

// FilesystemHealth represents filesystem health check results
type FilesystemHealth struct {
	Status                HealthStatus `json:"status"`
	StoragePathAccessible bool         `json:"storage_path_accessible"`
	FS9ServiceReachable   bool         `json:"fs9_service_reachable"`
	BasicOperationsWork   bool         `json:"basic_operations_work"`
	Error                 string       `json:"error,omitempty"`
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status     HealthStatus     `json:"status"`
	Filesystem FilesystemHealth `json:"filesystem"`
	Timestamp  time.Time        `json:"timestamp"`
}

// HealthCheckConfig holds configuration for health checks
type HealthCheckConfig struct {
	MaxRetries        int
	InitialRetryDelay time.Duration
	StoragePath       string
	FS9ServiceURL     string
}

// DefaultHealthCheckConfig returns default health check configuration
func DefaultHealthCheckConfig() HealthCheckConfig {
	return HealthCheckConfig{
		MaxRetries:        3,
		InitialRetryDelay: 1 * time.Second,
		StoragePath:       "./data/storage",
		FS9ServiceURL:     FS9ServiceURL,
	}
}

// HealthHandler performs comprehensive health checks
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	config := DefaultHealthCheckConfig()

	// Perform health checks
	fsHealth := checkFilesystemHealth(ctx, config)

	// Determine overall status
	overallStatus := determineOverallStatus(fsHealth)

	response := HealthResponse{
		Status:     overallStatus,
		Filesystem: fsHealth,
		Timestamp:  time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")

	// Return appropriate HTTP status code
	statusCode := http.StatusOK
	if overallStatus == HealthStatusDegraded {
		statusCode = http.StatusServiceUnavailable
	} else if overallStatus == HealthStatusUnhealthy {
		statusCode = http.StatusServiceUnavailable
	}

	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(Response{
		Success: overallStatus == HealthStatusHealthy,
		Data:    response,
	})
}

// checkFilesystemHealth performs comprehensive filesystem health checks
func checkFilesystemHealth(ctx context.Context, config HealthCheckConfig) FilesystemHealth {
	health := FilesystemHealth{
		Status: HealthStatusHealthy,
	}

	// Check 1: Storage path accessibility
	storageAccessible, storageErr := checkStoragePathAccessibility(config.StoragePath)
	health.StoragePathAccessible = storageAccessible
	if !storageAccessible {
		health.Status = HealthStatusDegraded
		health.Error = fmt.Sprintf("Storage path not accessible: %v", storageErr)
		logger.Error("Storage path accessibility check failed: %v", storageErr)
	}

	// Check 2: FS9 service reachability (with retry)
	fs9Reachable, fs9Err := checkFS9ServiceWithRetry(ctx, config)
	health.FS9ServiceReachable = fs9Reachable
	if !fs9Reachable {
		health.Status = HealthStatusDegraded
		if health.Error == "" {
			health.Error = fmt.Sprintf("FS9 service not reachable: %v", fs9Err)
		} else {
			health.Error += fmt.Sprintf("; FS9 service: %v", fs9Err)
		}
		logger.Error("FS9 service reachability check failed: %v", fs9Err)
	}

	// Check 3: Basic file operations (only if storage is accessible)
	if storageAccessible {
		basicOpsWork, basicOpsErr := checkBasicFileOperations(config.StoragePath)
		health.BasicOperationsWork = basicOpsWork
		if !basicOpsWork {
			health.Status = HealthStatusDegraded
			if health.Error == "" {
				health.Error = fmt.Sprintf("Basic file operations failed: %v", basicOpsErr)
			} else {
				health.Error += fmt.Sprintf("; Basic operations: %v", basicOpsErr)
			}
			logger.Error("Basic file operations check failed: %v", basicOpsErr)
		}
	}

	return health
}

// checkFS9ServiceWithRetry checks FS9 service connectivity with exponential backoff retry
func checkFS9ServiceWithRetry(ctx context.Context, config HealthCheckConfig) (bool, error) {
	var lastErr error

	for attempt := 0; attempt < config.MaxRetries; attempt++ {
		// Create FS9 client for health check
		client := NewFS9Client(config.FS9ServiceURL)

		// Attempt health check with timeout
		checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		err := client.HealthCheck(checkCtx)
		cancel()

		if err == nil {
			// Success on this attempt
			if attempt > 0 {
				logger.Info("FS9 service health check succeeded on attempt %d", attempt+1)
			}
			return true, nil
		}

		lastErr = err
		logger.Error("FS9 service health check attempt %d failed: %v", attempt+1, err)

		// Don't wait after the last attempt
		if attempt < config.MaxRetries-1 {
			// Calculate exponential backoff delay: 1s, 2s, 4s
			delay := config.InitialRetryDelay * time.Duration(1<<uint(attempt))
			logger.Info("Retrying FS9 health check in %v...", delay)

			select {
			case <-time.After(delay):
				// Continue to next attempt
			case <-ctx.Done():
				return false, fmt.Errorf("health check cancelled: %w", ctx.Err())
			}
		}
	}

	return false, fmt.Errorf("FS9 service health check failed after %d attempts: %w", config.MaxRetries, lastErr)
}

// checkStoragePathAccessibility verifies that the storage path is accessible
func checkStoragePathAccessibility(storagePath string) (bool, error) {
	// Create storage directory if it doesn't exist
	if err := os.MkdirAll(storagePath, 0755); err != nil {
		return false, fmt.Errorf("failed to create storage directory: %w", err)
	}

	// Check if directory is writable
	testFile := filepath.Join(storagePath, ".health_check_test")
	file, err := os.Create(testFile)
	if err != nil {
		return false, fmt.Errorf("storage path not writable: %w", err)
	}
	file.Close()

	// Clean up test file
	os.Remove(testFile)

	return true, nil
}

// checkBasicFileOperations performs basic file operation tests
func checkBasicFileOperations(storagePath string) (bool, error) {
	testDir := filepath.Join(storagePath, ".health_test")
	testFile := filepath.Join(testDir, "test.txt")
	testContent := []byte("health check test")

	// Clean up any existing test directory
	os.RemoveAll(testDir)

	// Create directory
	if err := os.MkdirAll(testDir, 0755); err != nil {
		return false, fmt.Errorf("failed to create test directory: %w", err)
	}

	// Write file
	if err := os.WriteFile(testFile, testContent, 0644); err != nil {
		return false, fmt.Errorf("failed to write test file: %w", err)
	}

	// Read file
	readContent, err := os.ReadFile(testFile)
	if err != nil {
		return false, fmt.Errorf("failed to read test file: %w", err)
	}

	if string(readContent) != string(testContent) {
		return false, fmt.Errorf("file content mismatch")
	}

	// Delete file
	if err := os.Remove(testFile); err != nil {
		return false, fmt.Errorf("failed to delete test file: %w", err)
	}

	// Clean up test directory
	if err := os.RemoveAll(testDir); err != nil {
		return false, fmt.Errorf("failed to clean up test directory: %w", err)
	}

	return true, nil
}

// determineOverallStatus determines the overall health status based on component health
func determineOverallStatus(fsHealth FilesystemHealth) HealthStatus {
	// If all checks passed, return healthy
	if fsHealth.StoragePathAccessible && fsHealth.FS9ServiceReachable && fsHealth.BasicOperationsWork {
		return HealthStatusHealthy
	}

	// If critical checks failed, return unhealthy
	if !fsHealth.StoragePathAccessible {
		return HealthStatusUnhealthy
	}

	// If non-critical checks failed, return degraded
	return HealthStatusDegraded
}

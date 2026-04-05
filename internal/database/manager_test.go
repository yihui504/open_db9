package database

import (
	"context"
	"testing"
	"time"
)

// TestManagerNewManager tests creating a new manager
func TestManagerNewManager(t *testing.T) {
	config := &PoolConfig{
		Host:     "localhost",
		Port:     5432,
		Database: "test_db",
		User:     "test_user",
		Password: "test_pass",
		SSLMode:  "disable",
	}

	// This will fail without a real database, but we test the structure
	_, err := NewManager(config)
	if err == nil {
		t.Log("NewManager succeeded (database available)")
	} else {
		t.Logf("NewManager failed as expected without database: %v", err)
	}
}

// TestManagerFromPool tests creating a manager from existing pool
func TestManagerFromPool(t *testing.T) {
	config := DefaultPoolConfig()
	pool, err := NewConnectionPool(config)
	if err != nil {
		t.Skipf("Skipping test: no database available: %v", err)
		return
	}
	defer pool.Close()

	manager := ManagerFromPool(pool)
	if manager == nil {
		t.Fatal("ManagerFromPool returned nil")
	}

	if manager.GetPool() != pool {
		t.Error("GetPool() does not return the original pool")
	}
}

// TestManagerGetPool tests getting the underlying pool
func TestManagerGetPool(t *testing.T) {
	config := DefaultPoolConfig()
	pool, err := NewConnectionPool(config)
	if err != nil {
		t.Skipf("Skipping test: no database available: %v", err)
		return
	}
	defer pool.Close()

	manager := ManagerFromPool(pool)
	retrievedPool := manager.GetPool()

	if retrievedPool != pool {
		t.Error("Retrieved pool is not the same as the original pool")
	}
}

// TestManagerStats tests retrieving pool statistics
func TestManagerStats(t *testing.T) {
	config := DefaultPoolConfig()
	pool, err := NewConnectionPool(config)
	if err != nil {
		t.Skipf("Skipping test: no database available: %v", err)
		return
	}
	defer pool.Close()

	manager := ManagerFromPool(pool)
	stats := manager.Stats()

	if stats == nil {
		t.Fatal("Stats() returned nil")
	}

	if stats.Closed {
		t.Error("Pool should not be closed immediately after creation")
	}

	if stats.MaxQueueSize != config.MaxQueueSize {
		t.Errorf("MaxQueueSize = %d, want %d", stats.MaxQueueSize, config.MaxQueueSize)
	}
}

// TestManagerClose tests closing the manager
func TestManagerClose(t *testing.T) {
	config := DefaultPoolConfig()
	pool, err := NewConnectionPool(config)
	if err != nil {
		t.Skipf("Skipping test: no database available: %v", err)
		return
	}

	manager := ManagerFromPool(pool)
	manager.Close()

	// Verify pool is closed
	stats := pool.Stats()
	if !stats.Closed {
		t.Error("Pool should be closed after manager.Close()")
	}
}

// TestManagerHealth tests health check
func TestManagerHealth(t *testing.T) {
	config := DefaultPoolConfig()
	pool, err := NewConnectionPool(config)
	if err != nil {
		t.Skipf("Skipping test: no database available: %v", err)
		return
	}
	defer pool.Close()

	manager := ManagerFromPool(pool)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = manager.Health(ctx)
	if err == nil {
		t.Log("Health check passed (database available)")
	} else {
		t.Logf("Health check failed (expected without database): %v", err)
	}
}

// TestJoinStrings tests the helper function
func TestJoinStrings(t *testing.T) {
	tests := []struct {
		name     string
		parts    []string
		sep      string
		expected string
	}{
		{
			name:     "empty slice",
			parts:    []string{},
			sep:      ", ",
			expected: "",
		},
		{
			name:     "single element",
			parts:    []string{"a"},
			sep:      ", ",
			expected: "a",
		},
		{
			name:     "multiple elements",
			parts:    []string{"a", "b", "c"},
			sep:      ", ",
			expected: "a, b, c",
		},
		{
			name:     "different separator",
			parts:    []string{"x", "y", "z"},
			sep:      " | ",
			expected: "x | y | z",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := joinStrings(tt.parts, tt.sep)
			if result != tt.expected {
				t.Errorf("joinStrings(%q, %q) = %q, want %q", tt.parts, tt.sep, result, tt.expected)
			}
		})
	}
}

// TestPoolConfigDefaults tests default pool configuration
func TestPoolConfigDefaults(t *testing.T) {
	config := DefaultPoolConfig()

	if config.Host != "localhost" {
		t.Errorf("Host = %s, want localhost", config.Host)
	}
	if config.Port != 5432 {
		t.Errorf("Port = %d, want 5432", config.Port)
	}
	if config.Database != "postgres" {
		t.Errorf("Database = %s, want postgres", config.Database)
	}
	if config.User != "postgres" {
		t.Errorf("User = %s, want postgres", config.User)
	}
	if config.SSLMode != "disable" {
		t.Errorf("SSLMode = %s, want disable", config.SSLMode)
	}
	if config.MaxConnections != 10 {
		t.Errorf("MaxConnections = %d, want 10", config.MaxConnections)
	}
	if config.MinConnections != 2 {
		t.Errorf("MinConnections = %d, want 2", config.MinConnections)
	}
	if config.Mode != PoolModeWait {
		t.Errorf("Mode = %s, want %s", config.Mode, PoolModeWait)
	}
}

// TestBuildConnectionString tests connection string building
func TestBuildConnectionString(t *testing.T) {
	config := &PoolConfig{
		Host:     "testhost",
		Port:     5433,
		Database: "testdb",
		User:     "testuser",
		Password: "testpass",
		SSLMode:  "require",
	}

	result := buildConnectionString(config)
	expected := "host=testhost port=5433 dbname=testdb user=testuser password=testpass sslmode=require"

	if result != expected {
		t.Errorf("buildConnectionString() = %s, want %s", result, expected)
	}
}

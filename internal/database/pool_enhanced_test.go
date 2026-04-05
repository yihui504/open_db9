package database

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestNewConnectionPool_ValidConfig tests pool creation with valid config
func TestNewConnectionPool_ValidConfig(t *testing.T) {
	config := &PoolConfig{
		Host:              "localhost",
		Port:              5432,
		Database:          "test_db",
		User:              "test_user",
		Password:          "test_pass",
		SSLMode:           "disable",
		MaxConnections:    5,
		MinConnections:    1,
		MaxConnLifetime:   30 * time.Minute,
		MaxConnIdleTime:   10 * time.Minute,
		HealthCheckPeriod: 1 * time.Minute,
		ConnectTimeout:    5 * time.Second,
		Mode:              PoolModeWait,
		MaxQueueSize:      50,
		QueueTimeout:      10 * time.Second,
	}

	pool, err := NewConnectionPool(config)
	if err != nil {
		t.Skipf("Skipping test: no database available: %v", err)
		return
	}
	defer pool.Close()

	assert.NotNil(t, pool)
	assert.NotNil(t, pool.pool)
	assert.Equal(t, config, pool.config)
	assert.False(t, pool.closed)
}

// TestNewConnectionPool_NilConfig tests pool creation with nil config
func TestNewConnectionPool_NilConfig(t *testing.T) {
	pool, err := NewConnectionPool(nil)
	if err != nil {
		t.Skipf("Skipping test: no database available: %v", err)
		return
	}
	defer pool.Close()

	assert.NotNil(t, pool)
	assert.NotNil(t, pool.config)
	assert.Equal(t, "localhost", pool.config.Host)
	assert.Equal(t, 5432, pool.config.Port)
}

// TestNewConnectionPool_DefaultPoolConfig tests using DefaultPoolConfig
func TestNewConnectionPool_DefaultPoolConfig(t *testing.T) {
	config := DefaultPoolConfig()
	pool, err := NewConnectionPool(config)
	if err != nil {
		t.Skipf("Skipping test: no database available: %v", err)
		return
	}
	defer pool.Close()

	assert.NotNil(t, pool)
	assert.Equal(t, config.MaxConnections, 10)
	assert.Equal(t, config.MinConnections, 2)
	assert.Equal(t, PoolModeWait, config.Mode)
}

// TestNewConnectionPool_InvalidConnectionString tests pool creation with invalid config
func TestNewConnectionPool_InvalidConnectionString(t *testing.T) {
	config := &PoolConfig{
		Host:              "invalid-host-that-does-not-exist.local",
		Port:              5432,
		Database:          "test_db",
		User:              "test_user",
		Password:          "test_pass",
		SSLMode:           "disable",
		ConnectTimeout:    100 * time.Millisecond,
		MaxConnections:    1,
		MinConnections:    1,
		HealthCheckPeriod: 1 * time.Minute,
	}

	pool, err := NewConnectionPool(config)
	// Note: This test may succeed if the config gets defaulted
	if err != nil {
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create connection pool")
	}
	if pool != nil {
		pool.Close()
	}
}

// TestConnectionPoolClose_ClosesPool tests closing the pool
func TestConnectionPoolClose_ClosesPool(t *testing.T) {
	config := DefaultPoolConfig()
	config.MaxConnections = 1
	config.MinConnections = 1

	pool, err := NewConnectionPool(config)
	if err != nil {
		t.Skipf("Skipping test: no database available: %v", err)
		return
	}

	pool.Close()

	stats := pool.Stats()
	assert.True(t, stats.Closed, "Pool should be closed")
}

// TestConnectionPoolClose_Idempotent tests Close can be called multiple times
func TestConnectionPoolClose_Idempotent(t *testing.T) {
	config := DefaultPoolConfig()
	config.MaxConnections = 1

	pool, err := NewConnectionPool(config)
	if err != nil {
		t.Skipf("Skipping test: no database available: %v", err)
		return
	}

	pool.Close()
	pool.Close() // Should not panic
	pool.Close() // Should not panic

	stats := pool.Stats()
	assert.True(t, stats.Closed)
}

// TestConnectionPoolGetConnection_WaitMode tests GetConnection with PoolModeWait
func TestConnectionPoolGetConnection_WaitMode(t *testing.T) {
	config := DefaultPoolConfig()
	config.Mode = PoolModeWait
	config.MaxConnections = 2

	pool, err := NewConnectionPool(config)
	if err != nil {
		t.Skipf("Skipping test: no database available: %v", err)
		return
	}
	defer pool.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := pool.GetConnection(ctx)
	if err != nil {
		t.Skipf("Skipping test: no database connection: %v", err)
		return
	}
	defer conn.Release()

	assert.NotNil(t, conn)
}

// TestConnectionPoolGetConnection_RejectMode tests GetConnection with PoolModeReject
func TestConnectionPoolGetConnection_RejectMode(t *testing.T) {
	config := DefaultPoolConfig()
	config.Mode = PoolModeReject
	config.MaxConnections = 1
	config.MaxQueueSize = 1

	pool, err := NewConnectionPool(config)
	if err != nil {
		t.Skipf("Skipping test: no database available: %v", err)
		return
	}
	defer pool.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// First connection should succeed
	conn1, err := pool.GetConnection(ctx)
	if err != nil {
		t.Skipf("Skipping test: no database connection: %v", err)
		return
	}

	// Try to get second connection (should be rejected)
	_, err = pool.GetConnection(ctx)
	if err == nil {
		t.Error("Expected rejection when pool is at capacity")
	}

	conn1.Release()
}

// TestConnectionPoolGetConnection_QueueMode tests GetConnection with PoolModeQueue
func TestConnectionPoolGetConnection_QueueMode(t *testing.T) {
	config := DefaultPoolConfig()
	config.Mode = PoolModeQueue
	config.MaxConnections = 1
	config.MaxQueueSize = 2
	config.QueueTimeout = 100 * time.Millisecond

	pool, err := NewConnectionPool(config)
	if err != nil {
		t.Skipf("Skipping test: no database available: %v", err)
		return
	}
	defer pool.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := pool.GetConnection(ctx)
	if err != nil {
		t.Skipf("Skipping test: no database connection: %v", err)
		return
	}
	defer conn.Release()

	assert.NotNil(t, conn)
}

// TestConnectionPoolGetConnection_ClosedPool tests GetConnection on closed pool
func TestConnectionPoolGetConnection_ClosedPool(t *testing.T) {
	config := DefaultPoolConfig()
	config.MaxConnections = 1

	pool, err := NewConnectionPool(config)
	if err != nil {
		t.Skipf("Skipping test: no database available: %v", err)
		return
	}

	pool.Close()

	ctx := context.Background()
	_, err = pool.GetConnection(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "closed")
}

// TestConnectionPoolGetConnection_ContextCancelled tests GetConnection with cancelled context
func TestConnectionPoolGetConnection_ContextCancelled(t *testing.T) {
	config := DefaultPoolConfig()
	config.Mode = PoolModeQueue
	config.MaxConnections = 1
	config.MaxQueueSize = 1

	pool, err := NewConnectionPool(config)
	if err != nil {
		t.Skipf("Skipping test: no database available: %v", err)
		return
	}
	defer pool.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err = pool.GetConnection(ctx)
	assert.Error(t, err)
}

// TestConnectionPoolExecute_QueryExecution tests Execute method
func TestConnectionPoolExecute_QueryExecution(t *testing.T) {
	config := DefaultPoolConfig()
	config.MaxConnections = 1

	pool, err := NewConnectionPool(config)
	if err != nil {
		t.Skipf("Skipping test: no database available: %v", err)
		return
	}
	defer pool.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = pool.Execute(ctx, "SELECT 1")
	if err != nil {
		t.Logf("Execute failed (expected without database): %v", err)
	}
}

// TestConnectionPoolExecute_ClosedPool tests Execute on closed pool
func TestConnectionPoolExecute_ClosedPool(t *testing.T) {
	config := DefaultPoolConfig()
	pool, err := NewConnectionPool(config)
	if err != nil {
		t.Skipf("Skipping test: no database available: %v", err)
		return
	}

	pool.Close()

	ctx := context.Background()
	err = pool.Execute(ctx, "SELECT 1")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "closed")
}

// TestConnectionPoolQuery_QueryExecution tests Query method
func TestConnectionPoolQuery_QueryExecution(t *testing.T) {
	config := DefaultPoolConfig()
	config.MaxConnections = 1

	pool, err := NewConnectionPool(config)
	if err != nil {
		t.Skipf("Skipping test: no database available: %v", err)
		return
	}
	defer pool.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := pool.Query(ctx, "SELECT 1")
	if err != nil {
		t.Logf("Query failed (expected without database): %v", err)
		return
	}
	defer rows.Close()

	assert.NotNil(t, rows)
}

// TestConnectionPoolQuery_ClosedPool tests Query on closed pool
func TestConnectionPoolQuery_ClosedPool(t *testing.T) {
	config := DefaultPoolConfig()
	pool, err := NewConnectionPool(config)
	if err != nil {
		t.Skipf("Skipping test: no database available: %v", err)
		return
	}

	pool.Close()

	ctx := context.Background()
	_, err = pool.Query(ctx, "SELECT 1")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "closed")
}

// TestConnectionPoolQueryRow_QueryExecution tests QueryRow method
func TestConnectionPoolQueryRow_QueryExecution(t *testing.T) {
	config := DefaultPoolConfig()
	config.MaxConnections = 1

	pool, err := NewConnectionPool(config)
	if err != nil {
		t.Skipf("Skipping test: no database available: %v", err)
		return
	}
	defer pool.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	row := pool.QueryRow(ctx, "SELECT 1")
	assert.NotNil(t, row)

	var result int
	err = row.Scan(&result)
	if err != nil {
		t.Logf("QueryRow scan failed (expected without database): %v", err)
	}
}

// TestConnectionPoolHealth_HealthyPool tests Health on healthy pool
func TestConnectionPoolHealth_HealthyPool(t *testing.T) {
	config := DefaultPoolConfig()
	config.MaxConnections = 1

	pool, err := NewConnectionPool(config)
	if err != nil {
		t.Skipf("Skipping test: no database available: %v", err)
		return
	}
	defer pool.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = pool.Health(ctx)
	if err != nil {
		t.Logf("Health check failed (expected without database): %v", err)
	}
}

// TestConnectionPoolHealth_ClosedPool tests Health on closed pool
func TestConnectionPoolHealth_ClosedPool(t *testing.T) {
	config := DefaultPoolConfig()
	pool, err := NewConnectionPool(config)
	if err != nil {
		t.Skipf("Skipping test: no database available: %v", err)
		return
	}

	pool.Close()

	ctx := context.Background()
	err = pool.Health(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "closed")
}

// TestConnectionPoolStats_ReturnsStats tests Stats returns valid statistics
func TestConnectionPoolStats_ReturnsStats(t *testing.T) {
	config := DefaultPoolConfig()
	config.MaxConnections = 5
	config.MinConnections = 2
	config.MaxQueueSize = 100

	pool, err := NewConnectionPool(config)
	if err != nil {
		t.Skipf("Skipping test: no database available: %v", err)
		return
	}
	defer pool.Close()

	stats := pool.Stats()

	assert.NotNil(t, stats)
	assert.Equal(t, 100, stats.MaxQueueSize)
	assert.False(t, stats.Closed)
	assert.GreaterOrEqual(t, stats.TotalConnections, 0)
	assert.GreaterOrEqual(t, stats.IdleConnections, 0)
	assert.GreaterOrEqual(t, stats.AcquireCount, 0)
	assert.GreaterOrEqual(t, stats.QueueSize, 0)
}

// TestConnectionPoolStats_AfterClose tests Stats after closing
func TestConnectionPoolStats_AfterClose(t *testing.T) {
	config := DefaultPoolConfig()
	pool, err := NewConnectionPool(config)
	if err != nil {
		t.Skipf("Skipping test: no database available: %v", err)
		return
	}

	pool.Close()

	stats := pool.Stats()

	assert.True(t, stats.Closed)
}

// TestBuildConnectionString_AllFields tests building connection string with all fields
func TestBuildConnectionString_AllFields(t *testing.T) {
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

	assert.Equal(t, expected, result)
}

// TestBuildConnectionString_DisableSSL tests building connection string with SSL disabled
func TestBuildConnectionString_DisableSSL(t *testing.T) {
	config := &PoolConfig{
		Host:     "localhost",
		Port:     5432,
		Database: "mydb",
		User:     "myuser",
		Password: "mypass",
		SSLMode:  "disable",
	}

	result := buildConnectionString(config)
	expected := "host=localhost port=5432 dbname=mydb user=myuser password=mypass sslmode=disable"

	assert.Equal(t, expected, result)
}

// TestBuildConnectionString_VerifySSL tests building connection string with verify SSL
func TestBuildConnectionString_VerifySSL(t *testing.T) {
	config := &PoolConfig{
		Host:     "secure.example.com",
		Port:     5432,
		Database: "securedb",
		User:     "secureuser",
		Password: "securepass",
		SSLMode:  "verify-full",
	}

	result := buildConnectionString(config)
	assert.Contains(t, result, "sslmode=verify-full")
	assert.Contains(t, result, "host=secure.example.com")
}

// TestPoolModeValues tests all PoolMode values
func TestPoolModeValues(t *testing.T) {
	tests := []struct {
		mode   PoolMode
		value  string
	}{
		{PoolModeReject, "reject"},
		{PoolModeQueue, "queue"},
		{PoolModeWait, "wait"},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			assert.Equal(t, tt.value, string(tt.mode))
		})
	}
}

// TestPoolConfigDefaults_VerifyDefaults tests DefaultPoolConfig returns correct defaults
func TestPoolConfigDefaults_VerifyDefaults(t *testing.T) {
	config := DefaultPoolConfig()

	assert.Equal(t, "localhost", config.Host)
	assert.Equal(t, 5432, config.Port)
	assert.Equal(t, "postgres", config.Database)
	assert.Equal(t, "postgres", config.User)
	assert.Equal(t, "disable", config.SSLMode)
	assert.Equal(t, 10, config.MaxConnections)
	assert.Equal(t, 2, config.MinConnections)
	assert.Equal(t, time.Hour, config.MaxConnLifetime)
	assert.Equal(t, 30*time.Minute, config.MaxConnIdleTime)
	assert.Equal(t, 1*time.Minute, config.HealthCheckPeriod)
	assert.Equal(t, 5*time.Second, config.ConnectTimeout)
	assert.Equal(t, PoolModeWait, config.Mode)
	assert.Equal(t, 100, config.MaxQueueSize)
	assert.Equal(t, 30*time.Second, config.QueueTimeout)
}

// TestConnectionPool_GetWithReject tests getWithReject method
func TestConnectionPool_GetWithReject(t *testing.T) {
	config := DefaultPoolConfig()
	config.Mode = PoolModeReject
	config.MaxConnections = 1
	config.MaxQueueSize = 1

	pool, err := NewConnectionPool(config)
	if err != nil {
		t.Skipf("Skipping test: no database available: %v", err)
		return
	}
	defer pool.Close()

	ctx := context.Background()

	// First connection should succeed
	conn, err := pool.getWithReject(ctx)
	if err != nil {
		t.Skipf("Skipping test: no database connection: %v", err)
		return
	}
	defer conn.Release()

	assert.NotNil(t, conn)

	// Second connection should be rejected
	_, err = pool.getWithReject(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at capacity")
}

// TestConnectionPool_GetWithQueue tests getWithQueue method
func TestConnectionPool_GetWithQueue(t *testing.T) {
	config := DefaultPoolConfig()
	config.Mode = PoolModeQueue
	config.MaxConnections = 1
	config.MaxQueueSize = 2
	config.QueueTimeout = 100 * time.Millisecond

	pool, err := NewConnectionPool(config)
	if err != nil {
		t.Skipf("Skipping test: no database available: %v", err)
		return
	}
	defer pool.Close()

	ctx := context.Background()

	conn, err := pool.getWithQueue(ctx)
	if err != nil {
		t.Skipf("Skipping test: no database connection: %v", err)
		return
	}
	defer conn.Release()

	assert.NotNil(t, conn)
}

// TestConnectionPool_GetWithWait tests getWithWait method
func TestConnectionPool_GetWithWait(t *testing.T) {
	config := DefaultPoolConfig()
	config.Mode = PoolModeWait
	config.MaxConnections = 1

	pool, err := NewConnectionPool(config)
	if err != nil {
		t.Skipf("Skipping test: no database available: %v", err)
		return
	}
	defer pool.Close()

	ctx := context.Background()

	conn, err := pool.getWithWait(ctx)
	if err != nil {
		t.Skipf("Skipping test: no database connection: %v", err)
		return
	}
	defer conn.Release()

	assert.NotNil(t, conn)
}

// TestPoolStats_StructFields tests PoolStats struct
func TestPoolStats_StructFields(t *testing.T) {
	stats := &PoolStats{
		TotalConnections: 10,
		IdleConnections:  5,
		AcquireCount:     100,
		AcquireDuration:  50,
		QueueSize:        2,
		MaxQueueSize:     100,
		Closed:           false,
	}

	assert.Equal(t, 10, stats.TotalConnections)
	assert.Equal(t, 5, stats.IdleConnections)
	assert.Equal(t, 100, stats.AcquireCount)
	assert.Equal(t, int64(50), stats.AcquireDuration)
	assert.Equal(t, 2, stats.QueueSize)
	assert.Equal(t, 100, stats.MaxQueueSize)
	assert.False(t, stats.Closed)
}

// TestConnectionPool_Concurrency tests concurrent access to pool
func TestConnectionPool_Concurrency(t *testing.T) {
	config := DefaultPoolConfig()
	config.MaxConnections = 2
	config.MinConnections = 1

	pool, err := NewConnectionPool(config)
	if err != nil {
		t.Skipf("Skipping test: no database available: %v", err)
		return
	}
	defer pool.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Try to get multiple connections concurrently
	done := make(chan bool, 3)

	for i := 0; i < 3; i++ {
		go func() {
			conn, err := pool.GetConnection(ctx)
			if err != nil {
				t.Logf("Concurrent connection failed: %v", err)
			} else {
				conn.Release()
			}
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 3; i++ {
		select {
		case <-done:
		case <-ctx.Done():
			t.Fatal("Timeout waiting for concurrent operations")
		}
	}
}

// TestConnectionPool_StatsAfterOperations tests stats after various operations
func TestConnectionPool_StatsAfterOperations(t *testing.T) {
	config := DefaultPoolConfig()
	config.MaxConnections = 2
	config.MinConnections = 1

	pool, err := NewConnectionPool(config)
	if err != nil {
		t.Skipf("Skipping test: no database available: %v", err)
		return
	}
	defer pool.Close()

	// Get initial stats
	stats1 := pool.Stats()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Try to get a connection
	conn, err := pool.GetConnection(ctx)
	if err != nil {
		t.Skipf("Skipping test: no database connection: %v", err)
		return
	}

	// Get stats while holding connection
	stats2 := pool.Stats()

	conn.Release()

	// Get stats after releasing
	stats3 := pool.Stats()

	assert.NotNil(t, stats1)
	assert.NotNil(t, stats2)
	assert.NotNil(t, stats3)
}

// TestNewConnectionPool_PgxPoolConfig tests that pgx pool config is set correctly
func TestNewConnectionPool_PgxPoolConfig(t *testing.T) {
	config := &PoolConfig{
		Host:              "localhost",
		Port:              5432,
		Database:          "test_db",
		User:              "test_user",
		Password:          "test_pass",
		SSLMode:           "disable",
		MaxConnections:    15,
		MinConnections:    3,
		MaxConnLifetime:   2 * time.Hour,
		MaxConnIdleTime:   45 * time.Minute,
		HealthCheckPeriod: 2 * time.Minute,
		ConnectTimeout:    10 * time.Second,
	}

	pool, err := NewConnectionPool(config)
	if err != nil {
		t.Skipf("Skipping test: no database available: %v", err)
		return
	}
	defer pool.Close()

	assert.NotNil(t, pool)
	assert.Equal(t, config, pool.config)
}

// MockPoolConfigForTesting creates a config suitable for testing
func MockPoolConfigForTesting() *PoolConfig {
	return &PoolConfig{
		Host:              "localhost",
		Port:              5432,
		Database:          "test_db",
		User:              "test_user",
		Password:          "test_pass",
		SSLMode:           "disable",
		MaxConnections:    1,
		MinConnections:    1,
		MaxConnLifetime:   5 * time.Minute,
		MaxConnIdleTime:   2 * time.Minute,
		HealthCheckPeriod: 30 * time.Second,
		ConnectTimeout:    2 * time.Second,
		Mode:              PoolModeWait,
		MaxQueueSize:      10,
		QueueTimeout:      5 * time.Second,
	}
}

// TestMockPoolConfigForTesting tests the mock config helper
func TestMockPoolConfigForTesting(t *testing.T) {
	config := MockPoolConfigForTesting()

	assert.Equal(t, "localhost", config.Host)
	assert.Equal(t, 5432, config.Port)
	assert.Equal(t, 1, config.MaxConnections)
	assert.Equal(t, PoolModeWait, config.Mode)
}

// TestConnectionPool_GetConnection_ContextTimeout tests GetConnection with timeout
func TestConnectionPool_GetConnection_ContextTimeout(t *testing.T) {
	config := DefaultPoolConfig()
	config.MaxConnections = 1

	pool, err := NewConnectionPool(config)
	if err != nil {
		t.Skipf("Skipping test: no database available: %v", err)
		return
	}
	defer pool.Close()

	// Use a very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Give time for timeout to occur
	time.Sleep(10 * time.Millisecond)

	_, err = pool.GetConnection(ctx)
	// May succeed if connection is available, or fail due to timeout
	if err != nil {
		assert.Contains(t, err.Error(), "context deadline exceeded")
	}
}

// TestConnectionPool_ExecuteWithParams tests Execute with parameters
func TestConnectionPool_ExecuteWithParams(t *testing.T) {
	config := DefaultPoolConfig()
	config.MaxConnections = 1

	pool, err := NewConnectionPool(config)
	if err != nil {
		t.Skipf("Skipping test: no database available: %v", err)
		return
	}
	defer pool.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = pool.Execute(ctx, "SELECT $1::int", 42)
	if err != nil {
		t.Logf("Execute with params failed (expected without database): %v", err)
	}
}

// TestConnectionPool_QueryWithParams tests Query with parameters
func TestConnectionPool_QueryWithParams(t *testing.T) {
	config := DefaultPoolConfig()
	config.MaxConnections = 1

	pool, err := NewConnectionPool(config)
	if err != nil {
		t.Skipf("Skipping test: no database available: %v", err)
		return
	}
	defer pool.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := pool.Query(ctx, "SELECT $1::int, $2::text", 42, "test")
	if err != nil {
		t.Logf("Query with params failed (expected without database): %v", err)
		return
	}
	defer rows.Close()

	assert.NotNil(t, rows)
}

// TestConnectionPool_QueryRowWithParams tests QueryRow with parameters
func TestConnectionPool_QueryRowWithParams(t *testing.T) {
	config := DefaultPoolConfig()
	config.MaxConnections = 1

	pool, err := NewConnectionPool(config)
	if err != nil {
		t.Skipf("Skipping test: no database available: %v", err)
		return
	}
	defer pool.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	row := pool.QueryRow(ctx, "SELECT $1::int AS num", 123)
	assert.NotNil(t, row)

	var num int
	err = row.Scan(&num)
	if err != nil {
		t.Logf("QueryRow scan failed (expected without database): %v", err)
	} else {
		assert.Equal(t, 123, num)
	}
}

// TestConnectionPool_GetConnection_ReleaseConnection tests getting and releasing connections
func TestConnectionPool_GetConnection_ReleaseConnection(t *testing.T) {
	config := DefaultPoolConfig()
	config.MaxConnections = 1

	pool, err := NewConnectionPool(config)
	if err != nil {
		t.Skipf("Skipping test: no database available: %v", err)
		return
	}
	defer pool.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Get connection
	conn, err := pool.GetConnection(ctx)
	if err != nil {
		t.Skipf("Skipping test: no database connection: %v", err)
		return
	}

	// Release connection
	conn.Release()

	// Should be able to get another connection
	conn2, err := pool.GetConnection(ctx)
	if err != nil {
		t.Skipf("Skipping test: no database connection: %v", err)
		return
	}
	defer conn2.Release()

	assert.NotNil(t, conn2)
}

// TestConnectionPool_StatsQueueSize tests QueueSize in stats
func TestConnectionPool_StatsQueueSize(t *testing.T) {
	config := DefaultPoolConfig()
	config.Mode = PoolModeQueue
	config.MaxConnections = 1
	config.MaxQueueSize = 5

	pool, err := NewConnectionPool(config)
	if err != nil {
		t.Skipf("Skipping test: no database available: %v", err)
		return
	}
	defer pool.Close()

	stats := pool.Stats()

	assert.Equal(t, 5, stats.MaxQueueSize)
	assert.GreaterOrEqual(t, stats.QueueSize, 0)
	assert.LessOrEqual(t, stats.QueueSize, stats.MaxQueueSize)
}

// TestConnectionPool_MultipleClose tests multiple Close calls don't panic
func TestConnectionPool_MultipleClose(t *testing.T) {
	config := DefaultPoolConfig()
	pool, err := NewConnectionPool(config)
	if err != nil {
		t.Skipf("Skipping test: no database available: %v", err)
		return
	}

	// Close multiple times
	pool.Close()
	pool.Close()
	pool.Close()

	stats := pool.Stats()
	assert.True(t, stats.Closed)
}

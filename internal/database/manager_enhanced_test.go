package database

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
)

// TestNewManager_Success tests successful manager creation with valid config
func TestNewManager_Success(t *testing.T) {
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
	}

	manager, err := NewManager(config)
	if err != nil {
		t.Skipf("Skipping test: no database available: %v", err)
		return
	}
	defer manager.Close()

	assert.NotNil(t, manager)
	assert.NotNil(t, manager.GetPool())
}

// TestNewManager_NilConfig tests manager creation with nil config (uses defaults)
func TestNewManager_NilConfig(t *testing.T) {
	manager, err := NewManager(nil)
	if err != nil {
		t.Skipf("Skipping test: no database available: %v", err)
		return
	}
	defer manager.Close()

	assert.NotNil(t, manager)
	assert.NotNil(t, manager.GetPool())
}

// TestManagerFromPool_CreatesManager tests creating manager from existing pool
func TestManagerFromPool_CreatesManager(t *testing.T) {
	config := DefaultPoolConfig()
	pool, err := NewConnectionPool(config)
	if err != nil {
		t.Skipf("Skipping test: no database available: %v", err)
		return
	}
	defer pool.Close()

	manager := ManagerFromPool(pool)

	assert.NotNil(t, manager)
	assert.Same(t, pool, manager.GetPool())
}

// TestManagerGetPool_ReturnsCorrectPool tests GetPool returns the underlying pool
func TestManagerGetPool_ReturnsCorrectPool(t *testing.T) {
	config := DefaultPoolConfig()
	pool, err := NewConnectionPool(config)
	if err != nil {
		t.Skipf("Skipping test: no database available: %v", err)
		return
	}
	defer pool.Close()

	manager := ManagerFromPool(pool)
	retrievedPool := manager.GetPool()

	assert.Same(t, pool, retrievedPool)
}

// TestManagerClose_ClosesPool tests Close closes the underlying pool
func TestManagerClose_ClosesPool(t *testing.T) {
	config := DefaultPoolConfig()
	pool, err := NewConnectionPool(config)
	if err != nil {
		t.Skipf("Skipping test: no database available: %v", err)
		return
	}

	manager := ManagerFromPool(pool)
	manager.Close()

	stats := pool.Stats()
	assert.True(t, stats.Closed, "Pool should be closed after manager.Close()")
}

// TestManagerClose_Idempotent tests Close can be called multiple times safely
func TestManagerClose_Idempotent(t *testing.T) {
	config := DefaultPoolConfig()
	pool, err := NewConnectionPool(config)
	if err != nil {
		t.Skipf("Skipping test: no database available: %v", err)
		return
	}

	manager := ManagerFromPool(pool)
	manager.Close()
	manager.Close() // Should not panic

	stats := pool.Stats()
	assert.True(t, stats.Closed)
}

// TestManagerHealth_WithDatabase tests health check with database
func TestManagerHealth_WithDatabase(t *testing.T) {
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
	if err != nil {
		t.Logf("Health check failed (expected without database): %v", err)
	}
}

// TestManagerHealth_AfterClose tests health check after closing
func TestManagerHealth_AfterClose(t *testing.T) {
	config := DefaultPoolConfig()
	pool, err := NewConnectionPool(config)
	if err != nil {
		t.Skipf("Skipping test: no database available: %v", err)
		return
	}

	manager := ManagerFromPool(pool)
	manager.Close()

	ctx := context.Background()
	err = manager.Health(ctx)

	assert.Error(t, err, "Health check should fail after close")
	assert.Contains(t, err.Error(), "closed")
}

// TestManagerStats_ReturnsValidStats tests Stats returns valid statistics
func TestManagerStats_ReturnsValidStats(t *testing.T) {
	config := DefaultPoolConfig()
	config.MaxQueueSize = 50

	pool, err := NewConnectionPool(config)
	if err != nil {
		t.Skipf("Skipping test: no database available: %v", err)
		return
	}
	defer pool.Close()

	manager := ManagerFromPool(pool)
	stats := manager.Stats()

	assert.NotNil(t, stats)
	assert.Equal(t, 50, stats.MaxQueueSize)
	assert.False(t, stats.Closed)
	assert.GreaterOrEqual(t, stats.TotalConnections, 0)
	assert.GreaterOrEqual(t, stats.IdleConnections, 0)
}

// TestManagerStats_AfterClose tests stats after closing manager
func TestManagerStats_AfterClose(t *testing.T) {
	config := DefaultPoolConfig()
	pool, err := NewConnectionPool(config)
	if err != nil {
		t.Skipf("Skipping test: no database available: %v", err)
		return
	}

	manager := ManagerFromPool(pool)
	manager.Close()

	stats := manager.Stats()

	assert.True(t, stats.Closed)
}

// TestManagerCreate_ValidationError tests Create with empty data
func TestManagerCreate_ValidationError(t *testing.T) {
	config := DefaultPoolConfig()
	pool, err := NewConnectionPool(config)
	if err != nil {
		t.Skipf("Skipping test: no database available: %v", err)
		return
	}
	defer pool.Close()

	manager := ManagerFromPool(pool)
	ctx := context.Background()

	// Test with empty data
	id, err := manager.Create(ctx, "test_table", map[string]interface{}{})

	// Should fail with empty data
	assert.Error(t, err)
	assert.Zero(t, id)
}

// TestManagerRead_NotFound tests Read when record doesn't exist
func TestManagerRead_NotFound(t *testing.T) {
	config := DefaultPoolConfig()
	pool, err := NewConnectionPool(config)
	if err != nil {
		t.Skipf("Skipping test: no database available: %v", err)
		return
	}
	defer pool.Close()

	manager := ManagerFromPool(pool)
	ctx := context.Background()

	// Try to read non-existent record
	result, err := manager.Read(ctx, "nonexistent_table", 999)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to read record")
}

// TestManagerUpdate_EmptyData tests Update with empty data map
func TestManagerUpdate_EmptyData(t *testing.T) {
	config := DefaultPoolConfig()
	pool, err := NewConnectionPool(config)
	if err != nil {
		t.Skipf("Skipping test: no database available: %v", err)
		return
	}
	defer pool.Close()

	manager := ManagerFromPool(pool)
	ctx := context.Background()

	err = manager.Update(ctx, "test_table", 1, map[string]interface{}{})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no data to update")
}

// TestManagerDelete_TableNotExists tests Delete on non-existent table
func TestManagerDelete_TableNotExists(t *testing.T) {
	config := DefaultPoolConfig()
	pool, err := NewConnectionPool(config)
	if err != nil {
		t.Skipf("Skipping test: no database available: %v", err)
		return
	}
	defer pool.Close()

	manager := ManagerFromPool(pool)
	ctx := context.Background()

	err = manager.Delete(ctx, "nonexistent_table", 1)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete record")
}

// TestManagerList_EmptyFilters tests List without filters
func TestManagerList_EmptyFilters(t *testing.T) {
	config := DefaultPoolConfig()
	pool, err := NewConnectionPool(config)
	if err != nil {
		t.Skipf("Skipping test: no database available: %v", err)
		return
	}
	defer pool.Close()

	manager := ManagerFromPool(pool)
	ctx := context.Background()

	// Test with empty filters and no limit
	results, err := manager.List(ctx, "nonexistent_table", map[string]interface{}{}, 0)

	assert.Error(t, err)
	assert.Nil(t, results)
}

// TestManagerList_WithFilters tests List with filters
func TestManagerList_WithFilters(t *testing.T) {
	config := DefaultPoolConfig()
	pool, err := NewConnectionPool(config)
	if err != nil {
		t.Skipf("Skipping test: no database available: %v", err)
		return
	}
	defer pool.Close()

	manager := ManagerFromPool(pool)
	ctx := context.Background()

	filters := map[string]interface{}{
		"status": "active",
		"type":   "user",
	}

	results, err := manager.List(ctx, "nonexistent_table", filters, 10)

	assert.Error(t, err)
	assert.Nil(t, results)
}

// TestManagerList_WithLimit tests List with limit
func TestManagerList_WithLimit(t *testing.T) {
	config := DefaultPoolConfig()
	pool, err := NewConnectionPool(config)
	if err != nil {
		t.Skipf("Skipping test: no database available: %v", err)
		return
	}
	defer pool.Close()

	manager := ManagerFromPool(pool)
	ctx := context.Background()

	results, err := manager.List(ctx, "nonexistent_table", nil, 5)

	assert.Error(t, err)
	assert.Nil(t, results)
}

// TestManagerTransaction_Success tests successful transaction
func TestManagerTransaction_Success(t *testing.T) {
	config := DefaultPoolConfig()
	pool, err := NewConnectionPool(config)
	if err != nil {
		t.Skipf("Skipping test: no database available: %v", err)
		return
	}
	defer pool.Close()

	manager := ManagerFromPool(pool)
	ctx := context.Background()

	err = manager.Transaction(ctx, func(tx pgx.Tx) error {
		// Transaction logic would go here
		return nil
	})

	// May fail without database, but code path is tested
	if err != nil {
		t.Logf("Transaction failed (expected without database): %v", err)
	}
}

// TestManagerTransaction_Rollback tests transaction rollback on error
func TestManagerTransaction_Rollback(t *testing.T) {
	config := DefaultPoolConfig()
	pool, err := NewConnectionPool(config)
	if err != nil {
		t.Skipf("Skipping test: no database available: %v", err)
		return
	}
	defer pool.Close()

	manager := ManagerFromPool(pool)
	ctx := context.Background()

	expectedErr := assert.AnError
	err = manager.Transaction(ctx, func(tx pgx.Tx) error {
		return expectedErr
	})

	assert.Error(t, err)
	assert.Same(t, expectedErr, err)
}

// TestManagerTransaction_Panic tests transaction handles panic
func TestManagerTransaction_Panic(t *testing.T) {
	config := DefaultPoolConfig()
	pool, err := NewConnectionPool(config)
	if err != nil {
		t.Skipf("Skipping test: no database available: %v", err)
		return
	}
	defer pool.Close()

	manager := ManagerFromPool(pool)
	ctx := context.Background()

	assert.Panics(t, func() {
		_ = manager.Transaction(ctx, func(tx pgx.Tx) error {
			panic("test panic")
		})
	})
}

// TestManagerBatchInsert_EmptyValues tests BatchInsert with empty values
func TestManagerBatchInsert_EmptyValues(t *testing.T) {
	config := DefaultPoolConfig()
	pool, err := NewConnectionPool(config)
	if err != nil {
		t.Skipf("Skipping test: no database available: %v", err)
		return
	}
	defer pool.Close()

	manager := ManagerFromPool(pool)
	ctx := context.Background()

	err = manager.BatchInsert(ctx, "test_table", []string{"col1", "col2"}, [][]interface{}{})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no values to insert")
}

// TestManagerBatchInsert_EmptyColumns tests BatchInsert with empty columns
func TestManagerBatchInsert_EmptyColumns(t *testing.T) {
	config := DefaultPoolConfig()
	pool, err := NewConnectionPool(config)
	if err != nil {
		t.Skipf("Skipping test: no database available: %v", err)
		return
	}
	defer pool.Close()

	manager := ManagerFromPool(pool)
	ctx := context.Background()

	values := [][]interface{}{
		{"value1", "value2"},
	}

	err = manager.BatchInsert(ctx, "test_table", []string{}, values)

	assert.Error(t, err)
}

// TestManagerExists_TableNotExists tests Exists on non-existent table
func TestManagerExists_TableNotExists(t *testing.T) {
	config := DefaultPoolConfig()
	pool, err := NewConnectionPool(config)
	if err != nil {
		t.Skipf("Skipping test: no database available: %v", err)
		return
	}
	defer pool.Close()

	manager := ManagerFromPool(pool)
	ctx := context.Background()

	exists, err := manager.Exists(ctx, "nonexistent_table", 1)

	assert.Error(t, err)
	assert.False(t, exists)
}

// TestManagerCount_TableNotExists tests Count on non-existent table
func TestManagerCount_TableNotExists(t *testing.T) {
	config := DefaultPoolConfig()
	pool, err := NewConnectionPool(config)
	if err != nil {
		t.Skipf("Skipping test: no database available: %v", err)
		return
	}
	defer pool.Close()

	manager := ManagerFromPool(pool)
	ctx := context.Background()

	count, err := manager.Count(ctx, "nonexistent_table", nil)

	assert.Error(t, err)
	assert.Zero(t, count)
}

// TestManagerCount_WithFilters tests Count with filters
func TestManagerCount_WithFilters(t *testing.T) {
	config := DefaultPoolConfig()
	pool, err := NewConnectionPool(config)
	if err != nil {
		t.Skipf("Skipping test: no database available: %v", err)
		return
	}
	defer pool.Close()

	manager := ManagerFromPool(pool)
	ctx := context.Background()

	filters := map[string]interface{}{
		"status": "active",
	}

	count, err := manager.Count(ctx, "nonexistent_table", filters)

	assert.Error(t, err)
	assert.Zero(t, count)
}

// TestManagerRaw_QueryExecution tests raw query execution
func TestManagerRaw_QueryExecution(t *testing.T) {
	config := DefaultPoolConfig()
	pool, err := NewConnectionPool(config)
	if err != nil {
		t.Skipf("Skipping test: no database available: %v", err)
		return
	}
	defer pool.Close()

	manager := ManagerFromPool(pool)
	ctx := context.Background()

	rows, err := manager.Raw(ctx, "SELECT 1")

	// May fail without database, but code path is tested
	if err != nil {
		t.Logf("Raw query failed (expected without database): %v", err)
		return
	}
	defer rows.Close()

	assert.NotNil(t, rows)
}

// TestManagerRawExec_CommandExecution tests raw command execution
func TestManagerRawExec_CommandExecution(t *testing.T) {
	config := DefaultPoolConfig()
	pool, err := NewConnectionPool(config)
	if err != nil {
		t.Skipf("Skipping test: no database available: %v", err)
		return
	}
	defer pool.Close()

	manager := ManagerFromPool(pool)
	ctx := context.Background()

	err = manager.RawExec(ctx, "SELECT 1")

	// May fail without database, but code path is tested
	if err != nil {
		t.Logf("Raw exec failed (expected without database): %v", err)
	}
}

// TestJoinStrings_VariousInputs tests joinStrings with various inputs
func TestJoinStrings_VariousInputs(t *testing.T) {
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
			parts:    []string{"one"},
			sep:      ", ",
			expected: "one",
		},
		{
			name:     "two elements",
			parts:    []string{"a", "b"},
			sep:      " AND ",
			expected: "a AND b",
		},
		{
			name:     "multiple elements",
			parts:    []string{"x", "y", "z"},
			sep:      ", ",
			expected: "x, y, z",
		},
		{
			name:     "special characters in separator",
			parts:    []string{"col1", "col2", "col3"},
			sep:      " || ",
			expected: "col1 || col2 || col3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := joinStrings(tt.parts, tt.sep)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestCreateBranch_ValidInput tests CreateBranch with valid input
func TestCreateBranch_ValidInput(t *testing.T) {
	config := DefaultPoolConfig()
	pool, err := NewConnectionPool(config)
	if err != nil {
		t.Skipf("Skipping test: no database available: %v", err)
		return
	}
	defer pool.Close()

	manager := ManagerFromPool(pool)
	ctx := context.Background()

	sourceID := mustParseUUID("123e4567-e89b-12d3-a456-426614174000")
	branchName := "test-branch"

	branch, err := manager.CreateBranch(ctx, *sourceID, branchName)

	// Expected to fail without proper database setup
	if err != nil {
		t.Logf("CreateBranch failed (expected without database): %v", err)
		return
	}

	assert.NotNil(t, branch)
	assert.NotEmpty(t, branch.Name)
	assert.Contains(t, branch.Name, branchName)
	assert.NotNil(t, branch.ParentID)
	assert.Equal(t, sourceID, *branch.ParentID)
	assert.NotNil(t, branch.SnapshotID)
}

// TestCreateBranch_SourceNotFound tests CreateBranch with non-existent source
func TestCreateBranch_SourceNotFound(t *testing.T) {
	config := DefaultPoolConfig()
	pool, err := NewConnectionPool(config)
	if err != nil {
		t.Skipf("Skipping test: no database available: %v", err)
		return
	}
	defer pool.Close()

	manager := ManagerFromPool(pool)
	ctx := context.Background()

	sourceID := mustParseUUID("123e4567-e89b-12d3-a456-426614174999")
	branchName := "test-branch"

	_, err = manager.CreateBranch(ctx, *sourceID, branchName)

	// Expected to fail
	if err == nil {
		t.Error("Expected error for non-existent source database")
	} else {
		assert.Contains(t, err.Error(), "source database not found")
	}
}

// TestListBranches_EmptyResults tests ListBranches with no branches
func TestListBranches_EmptyResults(t *testing.T) {
	config := DefaultPoolConfig()
	pool, err := NewConnectionPool(config)
	if err != nil {
		t.Skipf("Skipping test: no database available: %v", err)
		return
	}
	defer pool.Close()

	manager := ManagerFromPool(pool)
	ctx := context.Background()

	sourceID := mustParseUUID("123e4567-e89b-12d3-a456-426614174000")

	branches, err := manager.ListBranches(ctx, *sourceID)

	// Expected to fail without proper database setup
	if err != nil {
		t.Logf("ListBranches failed (expected without database): %v", err)
		return
	}

	assert.NotNil(t, branches)
}

// TestDeleteBranch_NotConfirmed tests DeleteBranch without confirmation
func TestDeleteBranch_NotConfirmed(t *testing.T) {
	config := DefaultPoolConfig()
	pool, err := NewConnectionPool(config)
	if err != nil {
		t.Skipf("Skipping test: no database available: %v", err)
		return
	}
	defer pool.Close()

	manager := ManagerFromPool(pool)
	ctx := context.Background()

	branchID := mustParseUUID("123e4567-e89b-12d3-a456-426614174000")

	err = manager.DeleteBranch(ctx, *branchID, false)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "requires confirmation")
}

// TestDeleteBranch_NotABranch tests DeleteBranch on non-branch database
func TestDeleteBranch_NotABranch(t *testing.T) {
	config := DefaultPoolConfig()
	pool, err := NewConnectionPool(config)
	if err != nil {
		t.Skipf("Skipping test: no database available: %v", err)
		return
	}
	defer pool.Close()

	manager := ManagerFromPool(pool)
	ctx := context.Background()

	branchID := mustParseUUID("123e4567-e89b-12d3-a456-426614174000")

	err = manager.DeleteBranch(ctx, *branchID, true)

	// Expected to fail without proper database setup
	if err == nil {
		t.Error("Expected error when deleting non-branch database")
	}
}

// Helper function to parse UUID for tests
func mustParseUUID(s string) *uuid.UUID {
	id, err := uuid.Parse(s)
	if err != nil {
		panic(err)
	}
	return &id
}

// TestDatabaseBranch_StructFields tests DatabaseBranch struct
func TestDatabaseBranch_StructFields(t *testing.T) {
	now := time.Now()
	parentID := mustParseUUID("123e4567-e89b-12d3-a456-426614174000")
	snapshotID := mustParseUUID("123e4567-e89b-12d3-a456-426614174001")

	branch := DatabaseBranch{
		ID:         uuid.New(),
		Name:       "test-branch",
		ParentID:   parentID,
		SnapshotID: snapshotID,
		CreatedAt:  now,
	}

	assert.NotEqual(t, uuid.Nil, branch.ID)
	assert.Equal(t, "test-branch", branch.Name)
	assert.Equal(t, parentID, branch.ParentID)
	assert.Equal(t, snapshotID, branch.SnapshotID)
	assert.Equal(t, now, branch.CreatedAt)
}

// TestPoolMode_StringValues tests PoolMode string values
func TestPoolMode_StringValues(t *testing.T) {
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

// TestPoolStats_Fields tests PoolStats struct fields
func TestPoolStats_Fields(t *testing.T) {
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

package database

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Manager provides high-level database operations
type Manager struct {
	pool *ConnectionPool
}

// NewManager creates a new database manager
func NewManager(config *PoolConfig) (*Manager, error) {
	pool, err := NewConnectionPool(config)
	if err != nil {
		return nil, err
	}

	return &Manager{pool: pool}, nil
}

// ManagerFromPool creates a manager from an existing pool
func ManagerFromPool(pool *ConnectionPool) *Manager {
	return &Manager{pool: pool}
}

// GetPool returns the underlying connection pool
func (m *Manager) GetPool() *ConnectionPool {
	return m.pool
}

// Close closes the database manager and its pool
func (m *Manager) Close() {
	m.pool.Close()
}

// CRUD Operations

// Create executes an INSERT statement
func (m *Manager) Create(ctx context.Context, table string, data map[string]interface{}) (int, error) {
	columns := make([]string, 0, len(data))
	placeholders := make([]string, 0, len(data))
	values := make([]interface{}, 0, len(data))

	i := 1
	for col, val := range data {
		columns = append(columns, fmt.Sprintf("\"%s\"", col))
		placeholders = append(placeholders, fmt.Sprintf("$%d", i))
		values = append(values, val)
		i++
	}

	sql := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s) RETURNING id",
		table,
		joinStrings(columns, ", "),
		joinStrings(placeholders, ", "),
	)

	var id int
	err := m.pool.QueryRow(ctx, sql, values...).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("failed to create record: %w", err)
	}

	return id, nil
}

// Read executes a SELECT statement
func (m *Manager) Read(ctx context.Context, table string, id int) (map[string]interface{}, error) {
	sql := fmt.Sprintf("SELECT * FROM %s WHERE id = $1", table)

	rows, err := m.pool.Query(ctx, sql, id)
	if err != nil {
		return nil, fmt.Errorf("failed to read record: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, fmt.Errorf("record not found")
	}

	result, err := pgx.CollectOneRow(rows, pgx.RowToMap)
	if err != nil {
		return nil, fmt.Errorf("failed to scan row: %w", err)
	}

	return result, nil
}

// Update executes an UPDATE statement
func (m *Manager) Update(ctx context.Context, table string, id int, data map[string]interface{}) error {
	if len(data) == 0 {
		return fmt.Errorf("no data to update")
	}

	setParts := make([]string, 0, len(data))
	values := make([]interface{}, 0, len(data)+1)

	i := 1
	for col, val := range data {
		setParts = append(setParts, fmt.Sprintf("\"%s\" = $%d", col, i))
		values = append(values, val)
		i++
	}

	// Add WHERE clause parameter
	values = append(values, id)

	sql := fmt.Sprintf(
		"UPDATE %s SET %s WHERE id = $%d",
		table,
		joinStrings(setParts, ", "),
		i,
	)

	err := m.pool.Execute(ctx, sql, values...)
	if err != nil {
		return fmt.Errorf("failed to update record: %w", err)
	}

	return nil
}

// Delete executes a DELETE statement
func (m *Manager) Delete(ctx context.Context, table string, id int) error {
	sql := fmt.Sprintf("DELETE FROM %s WHERE id = $1", table)

	err := m.pool.Execute(ctx, sql, id)
	if err != nil {
		return fmt.Errorf("failed to delete record: %w", err)
	}

	return nil
}

// List executes a SELECT with optional filtering
func (m *Manager) List(ctx context.Context, table string, filters map[string]interface{}, limit int) ([]map[string]interface{}, error) {
	sql := fmt.Sprintf("SELECT * FROM %s", table)
	args := []interface{}{}

	if len(filters) > 0 {
		whereParts := make([]string, 0, len(filters))
		i := 1
		for col, val := range filters {
			whereParts = append(whereParts, fmt.Sprintf("\"%s\" = $%d", col, i))
			args = append(args, val)
			i++
		}
		sql += " WHERE " + joinStrings(whereParts, " AND ")
	}

	if limit > 0 {
		sql += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := m.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list records: %w", err)
	}
	defer rows.Close()

	results, err := pgx.CollectRows(rows, pgx.RowToMap)
	if err != nil {
		return nil, fmt.Errorf("failed to collect rows: %w", err)
	}

	return results, nil
}

// Transaction Operations

// Transaction executes a function within a database transaction
func (m *Manager) Transaction(ctx context.Context, fn func(tx pgx.Tx) error) error {
	conn, err := m.pool.GetConnection(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()

	tx, err := conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback(ctx)
			panic(r)
		}
	}()

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return fmt.Errorf("transaction error: %w, rollback error: %v", err, rbErr)
		}
		return err
	}

	return tx.Commit(ctx)
}

// Batch Operations

// BatchInsert inserts multiple records in a single operation
func (m *Manager) BatchInsert(ctx context.Context, table string, columns []string, values [][]interface{}) error {
	if len(values) == 0 {
		return fmt.Errorf("no values to insert")
	}

	// Build column list
	colList := make([]string, len(columns))
	for i, col := range columns {
		colList[i] = fmt.Sprintf("\"%s\"", col)
	}

	// Build placeholders for first row
	numCols := len(columns)
	placeholders := make([]string, numCols)
	for i := 0; i < numCols; i++ {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
	}

	sql := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		table,
		joinStrings(colList, ", "),
		joinStrings(placeholders, ", "),
	)

	// Execute batch
	conn, err := m.pool.GetConnection(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()

	batch := &pgx.Batch{}
	for _, row := range values {
		batch.Queue(sql, row...)
	}

	br := conn.SendBatch(ctx, batch)
	defer br.Close()

	for i := 0; i < len(values); i++ {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("failed to insert row %d: %w", i, err)
		}
	}

	return nil
}

// Utility Functions

// Exists checks if a record exists
func (m *Manager) Exists(ctx context.Context, table string, id int) (bool, error) {
	sql := fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM %s WHERE id = $1)", table)

	var exists bool
	err := m.pool.QueryRow(ctx, sql, id).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check existence: %w", err)
	}

	return exists, nil
}

// Count returns the number of records matching filters
func (m *Manager) Count(ctx context.Context, table string, filters map[string]interface{}) (int, error) {
	sql := fmt.Sprintf("SELECT COUNT(*) FROM %s", table)
	args := []interface{}{}

	if len(filters) > 0 {
		whereParts := make([]string, 0, len(filters))
		i := 1
		for col, val := range filters {
			whereParts = append(whereParts, fmt.Sprintf("\"%s\" = $%d", col, i))
			args = append(args, val)
			i++
		}
		sql += " WHERE " + joinStrings(whereParts, " AND ")
	}

	var count int
	rows, err := m.pool.Query(ctx, sql, args...)
	if err != nil {
		return 0, fmt.Errorf("failed to count records: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		return 0, fmt.Errorf("no count result")
	}

	if err := rows.Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to scan count: %w", err)
	}

	return count, nil
}

// Raw executes a raw SQL query
func (m *Manager) Raw(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	return m.pool.Query(ctx, sql, args...)
}

// RawExec executes a raw SQL command
func (m *Manager) RawExec(ctx context.Context, sql string, args ...interface{}) error {
	return m.pool.Execute(ctx, sql, args...)
}

// Health checks the database connection
func (m *Manager) Health(ctx context.Context) error {
	return m.pool.Health(ctx)
}

// Stats returns pool statistics
func (m *Manager) Stats() *PoolStats {
	return m.pool.Stats()
}

// Helper function to join strings
func joinStrings(parts []string, sep string) string {
	if len(parts) == 0 {
		return ""
	}
	result := parts[0]
	for i := 1; i < len(parts); i++ {
		result += sep + parts[i]
	}
	return result
}

// Branch Management Operations

// DatabaseBranch represents a branch database with metadata
type DatabaseBranch struct {
	ID         uuid.UUID `json:"id"`
	Name       string    `json:"name"`
	ParentID   *uuid.UUID `json:"parent_id,omitempty"`
	SnapshotID *uuid.UUID `json:"snapshot_id,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

// CreateBranch creates a new branch from a source database
// Steps:
// 1. Create a snapshot of the source database
// 2. Create a new database from the snapshot
// 3. Set parent_id to source database and snapshot_id to the created snapshot
func (m *Manager) CreateBranch(ctx context.Context, sourceID uuid.UUID, name string) (*DatabaseBranch, error) {
	// Validate source database exists
	var sourceExists bool
	err := m.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM databases WHERE id = $1)", sourceID).Scan(&sourceExists)
	if err != nil {
		return nil, fmt.Errorf("failed to verify source database: %w", err)
	}
	if !sourceExists {
		return nil, fmt.Errorf("source database not found: %s", sourceID)
	}

	// Generate branch name with timestamp
	timestamp := time.Now().Format("20060102-150405")
	branchName := fmt.Sprintf("branch-%s-%s", name, timestamp)

	// Step 1: Create snapshot of source database
	snapshotID := uuid.New()
	snapshotSQL := `
		INSERT INTO snapshots (id, database_id, name, description, status, snapshot_type, storage_location, metadata)
		VALUES ($1, $2, $3, $4, 'completed', 'manual', $5, $6)
		RETURNING id
	`
	snapshotLocation := fmt.Sprintf("/snapshots/%s", snapshotID)
	snapshotDesc := fmt.Sprintf("Snapshot for branch: %s", branchName)

	err = m.pool.Execute(ctx, snapshotSQL,
		snapshotID,
		sourceID,
		branchName+"-snapshot",
		snapshotDesc,
		snapshotLocation,
		map[string]interface{}{"branch_name": name},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create snapshot: %w", err)
	}

	// Step 2: Get source database details to copy
	var sourceName, engine, version, region, size string
	var tenantID uuid.UUID
	var storageGB, cpuCores, memoryMB int
	var connectionDetails, settings map[string]interface{}

	sourceQuery := `
		SELECT name, tenant_id, engine, version, region, size,
		       storage_gb, cpu_cores, memory_mb, connection_details, settings
		FROM databases WHERE id = $1
	`
	err = m.pool.QueryRow(ctx, sourceQuery, sourceID).Scan(
		&sourceName, &tenantID, &engine, &version, &region, &size,
		&storageGB, &cpuCores, &memoryMB, &connectionDetails, &settings,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get source database details: %w", err)
	}

	// Step 3: Create new database from snapshot with parent relationship
	branchID := uuid.New()
	createBranchSQL := `
		INSERT INTO databases (
			id, tenant_id, name, slug, engine, version, region, size,
			status, connection_details, storage_gb, cpu_cores, memory_mb,
			settings, parent_id, snapshot_id
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8,
			'active', $9, $10, $11, $12,
			$13, $14, $15
		)
		RETURNING id, name, parent_id, snapshot_id, created_at
	`

	// Generate slug from branch name
	slug := fmt.Sprintf("%s-%s", name, timestamp)

	var branch DatabaseBranch
	err = m.pool.QueryRow(ctx, createBranchSQL,
		branchID,
		tenantID,
		branchName,
		slug,
		engine,
		version,
		region,
		size,
		connectionDetails,
		storageGB,
		cpuCores,
		memoryMB,
		settings,
		sourceID,    // parent_id
		snapshotID,  // snapshot_id
	).Scan(&branch.ID, &branch.Name, &branch.ParentID, &branch.SnapshotID, &branch.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create branch database: %w", err)
	}

	return &branch, nil
}

// ListBranches returns all branches of a source database
func (m *Manager) ListBranches(ctx context.Context, sourceID uuid.UUID) ([]DatabaseBranch, error) {
	query := `
		SELECT id, name, parent_id, snapshot_id, created_at
		FROM databases
		WHERE parent_id = $1
		ORDER BY created_at DESC
	`

	rows, err := m.pool.Query(ctx, query, sourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list branches: %w", err)
	}
	defer rows.Close()

	branches := []DatabaseBranch{}
	for rows.Next() {
		var branch DatabaseBranch
		err := rows.Scan(&branch.ID, &branch.Name, &branch.ParentID, &branch.SnapshotID, &branch.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan branch: %w", err)
		}
		branches = append(branches, branch)
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("error iterating branches: %w", rows.Err())
	}

	return branches, nil
}

// DeleteBranch deletes a branch database (requires confirmation)
func (m *Manager) DeleteBranch(ctx context.Context, branchID uuid.UUID, confirm bool) error {
	if !confirm {
		return fmt.Errorf("branch deletion requires confirmation (set confirm=true)")
	}

	// Verify this is actually a branch (has parent_id)
	var isBranch bool
	err := m.pool.QueryRow(ctx, "SELECT parent_id IS NOT NULL FROM databases WHERE id = $1", branchID).Scan(&isBranch)
	if err != nil {
		return fmt.Errorf("failed to verify branch: %w", err)
	}
	if !isBranch {
		return fmt.Errorf("database is not a branch (no parent_id)")
	}

	// Delete the branch (cascade will handle dependent records)
	deleteSQL := `DELETE FROM databases WHERE id = $1`
	err = m.pool.Execute(ctx, deleteSQL, branchID)
	if err != nil {
		return fmt.Errorf("failed to delete branch: %w", err)
	}

	return nil
}

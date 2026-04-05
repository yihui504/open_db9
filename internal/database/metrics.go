package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// MetricsCollector collects database metrics from pg_stat_statements and other sources
type MetricsCollector struct {
	pool *ConnectionPool
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(pool *ConnectionPool) *MetricsCollector {
	return &MetricsCollector{pool: pool}
}

// QueryStats represents statistics from pg_stat_statements
type QueryStats struct {
	QueryID           int64
	Query             string
	Calls             int64
	TotalExecTime     float64
	MeanExecTime      float64
	MaxExecTime       float64
	StdDevExecTime    float64
	Rows              int64
	SharedBlksHit     int64
	SharedBlksRead    int64
	SharedBlksDirtied int64
	SharedBlksWritten int64
}

// DatabaseStats represents overall database statistics
type DatabaseStats struct {
	DatabaseName          string
	DatabaseSize          int64
	ConnectionsCount      int
	ActiveConnections     int
	IdleConnections       int
	TransactionsCommitted int64
	TransactionsRolledBack int64
	CacheHitRatio         float64
	SampledAt             time.Time
}

// CollectQueryStats collects query statistics from pg_stat_statements
func (mc *MetricsCollector) CollectQueryStats(ctx context.Context, limit int) ([]QueryStats, error) {
	// Get a connection from the pool
	pconn, err := mc.pool.GetConnection(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire connection: %w", err)
	}
	defer pconn.Release()

	// Check if pg_stat_statements is enabled
	var extVersion string
	err = pconn.QueryRow(ctx, "SELECT extversion FROM pg_extension WHERE extname = 'pg_stat_statements'").Scan(&extVersion)
	if err != nil {
		return nil, fmt.Errorf("pg_stat_statements extension not enabled: %w", err)
	}

	// Query pg_stat_statements for top queries by execution time
	query := `
		SELECT
			queryid,
			query,
			calls,
			total_exec_time,
			mean_exec_time,
			max_exec_time,
			stddev_exec_time,
			rows,
			shared_blks_hit,
			shared_blks_read,
			shared_blks_dirtied,
			shared_blks_written
		FROM pg_stat_statements
		ORDER BY total_exec_time DESC
		LIMIT $1
	`

	rows, err := pconn.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query pg_stat_statements: %w", err)
	}
	defer rows.Close()

	var stats []QueryStats
	for rows.Next() {
		var s QueryStats
		err := rows.Scan(
			&s.QueryID,
			&s.Query,
			&s.Calls,
			&s.TotalExecTime,
			&s.MeanExecTime,
			&s.MaxExecTime,
			&s.StdDevExecTime,
			&s.Rows,
			&s.SharedBlksHit,
			&s.SharedBlksRead,
			&s.SharedBlksDirtied,
			&s.SharedBlksWritten,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan query stats: %w", err)
		}
		stats = append(stats, s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating query stats: %w", err)
	}

	return stats, nil
}

// CollectDatabaseStats collects general database statistics
func (mc *MetricsCollector) CollectDatabaseStats(ctx context.Context) (*DatabaseStats, error) {
	// Get a connection from the pool
	pconn, err := mc.pool.GetConnection(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire connection: %w", err)
	}
	defer pconn.Release()

	stats := &DatabaseStats{
		SampledAt: time.Now(),
	}

	// Get database name
	err = pconn.QueryRow(ctx, "SELECT current_database()").Scan(&stats.DatabaseName)
	if err != nil {
		return nil, fmt.Errorf("failed to get database name: %w", err)
	}

	// Get database size
	err = pconn.QueryRow(ctx, "SELECT pg_database_size(current_database())").Scan(&stats.DatabaseSize)
	if err != nil {
		return nil, fmt.Errorf("failed to get database size: %w", err)
	}

	// Get connection counts from pg_stat_activity
	connQuery := `
		SELECT
			COUNT(*) as total_conn,
			COUNT(*) FILTER (WHERE state = 'active') as active_conn,
			COUNT(*) FILTER (WHERE state = 'idle') as idle_conn
		FROM pg_stat_activity
		WHERE datname = current_database()
	`
	err = pconn.QueryRow(ctx, connQuery).Scan(
		&stats.ConnectionsCount,
		&stats.ActiveConnections,
		&stats.IdleConnections,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection counts: %w", err)
	}

	// Get transaction statistics from pg_stat_database
	dbQuery := `
		SELECT
			xact_commit,
			xact_rollback,
			blks_hit,
			blks_read
		FROM pg_stat_database
		WHERE datname = current_database()
	`
	var blksHit, blksRead int64
	err = pconn.QueryRow(ctx, dbQuery).Scan(
		&stats.TransactionsCommitted,
		&stats.TransactionsRolledBack,
		&blksHit,
		&blksRead,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get database stats: %w", err)
	}

	// Calculate cache hit ratio
	if blksHit+blksRead > 0 {
		stats.CacheHitRatio = float64(blksHit) / float64(blksHit+blksRead) * 100
	}

	return stats, nil
}

// GetSlowQueries retrieves queries that exceed the threshold
func (mc *MetricsCollector) GetSlowQueries(ctx context.Context, thresholdMs int64, limit int) ([]QueryStats, error) {
	// Get a connection from the pool
	pconn, err := mc.pool.GetConnection(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire connection: %w", err)
	}
	defer pconn.Release()

	// Query pg_stat_statements for slow queries
	query := `
		SELECT
			queryid,
			query,
			calls,
			total_exec_time,
			mean_exec_time,
			max_exec_time,
			rows
		FROM pg_stat_statements
		WHERE mean_exec_time > $1
		ORDER BY mean_exec_time DESC
		LIMIT $2
	`

	rows, err := pconn.Query(ctx, query, float64(thresholdMs), limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query pg_stat_statements: %w", err)
	}
	defer rows.Close()

	var slowQueries []QueryStats
	for rows.Next() {
		var s QueryStats
		err := rows.Scan(&s.QueryID, &s.Query, &s.Calls, &s.TotalExecTime, &s.MeanExecTime, &s.MaxExecTime, &s.Rows)
		if err != nil {
			continue
		}
		slowQueries = append(slowQueries, s)
	}

	return slowQueries, nil
}

// ResetStats resets pg_stat_statements statistics
func (mc *MetricsCollector) ResetStats(ctx context.Context) error {
	// Get a connection from the pool
	pconn, err := mc.pool.GetConnection(ctx)
	if err != nil {
		return fmt.Errorf("failed to acquire connection: %w", err)
	}
	defer pconn.Release()

	_, err = pconn.Exec(ctx, "SELECT pg_stat_statements_reset()")
	if err != nil {
		return fmt.Errorf("failed to reset pg_stat_statements: %w", err)
	}

	return nil
}

// CollectQueryStatsDirect collects query stats using a direct connection
func CollectQueryStatsDirect(ctx context.Context, conn *pgx.Conn, limit int, sortBy string) ([]QueryStats, error) {
	query := fmt.Sprintf(`
		SELECT
			queryid,
			query,
			calls,
			total_exec_time,
			mean_exec_time,
			max_exec_time,
			rows
		FROM pg_stat_statements
		ORDER BY %s DESC
		LIMIT $1
	`, sortBy)

	rows, err := conn.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query pg_stat_statements: %w", err)
	}
	defer rows.Close()

	var stats []QueryStats
	for rows.Next() {
		var s QueryStats
		err := rows.Scan(&s.QueryID, &s.Query, &s.Calls, &s.TotalExecTime, &s.MeanExecTime, &s.MaxExecTime, &s.Rows)
		if err != nil {
			continue
		}
		stats = append(stats, s)
	}

	return stats, nil
}

// CollectSlowQueriesDirect collects slow queries using a direct connection
func CollectSlowQueriesDirect(ctx context.Context, conn *pgx.Conn, thresholdMs int64, limit int) ([]QueryStats, error) {
	query := `
		SELECT
			queryid,
			query,
			calls,
			total_exec_time,
			mean_exec_time,
			max_exec_time,
			rows
		FROM pg_stat_statements
		WHERE mean_exec_time > $1
		ORDER BY mean_exec_time DESC
		LIMIT $2
	`

	rows, err := conn.Query(ctx, query, float64(thresholdMs), limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query pg_stat_statements: %w", err)
	}
	defer rows.Close()

	var stats []QueryStats
	for rows.Next() {
		var s QueryStats
		err := rows.Scan(&s.QueryID, &s.Query, &s.Calls, &s.TotalExecTime, &s.MeanExecTime, &s.MaxExecTime, &s.Rows)
		if err != nil {
			continue
		}
		stats = append(stats, s)
	}

	return stats, nil
}

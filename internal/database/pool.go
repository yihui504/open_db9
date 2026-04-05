package database

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PoolMode defines how connection pool handles overflow
type PoolMode string

const (
	PoolModeReject PoolMode = "reject" // Immediately reject new connections
	PoolModeQueue  PoolMode = "queue"  // Queue connection requests
	PoolModeWait   PoolMode = "wait"   // Wait indefinitely for a connection
)

// PoolConfig holds connection pool configuration
type PoolConfig struct {
	// Connection settings
	Host     string
	Port     int
	Database string
	User     string
	Password string
	SSLMode  string

	// Pool settings
	MaxConnections    int
	MinConnections    int
	MaxConnLifetime   time.Duration
	MaxConnIdleTime   time.Duration
	HealthCheckPeriod time.Duration
	ConnectTimeout    time.Duration

	// Overflow behavior
	Mode         PoolMode
	MaxQueueSize int
	QueueTimeout time.Duration
}

// DefaultPoolConfig returns a pool configuration with sensible defaults
func DefaultPoolConfig() *PoolConfig {
	return &PoolConfig{
		Host:              "localhost",
		Port:              5432,
		Database:          "postgres",
		User:              "postgres",
		SSLMode:           "disable",
		MaxConnections:    10,
		MinConnections:    2,
		MaxConnLifetime:   time.Hour,
		MaxConnIdleTime:   30 * time.Minute,
		HealthCheckPeriod: 1 * time.Minute,
		ConnectTimeout:    5 * time.Second,
		Mode:              PoolModeWait,
		MaxQueueSize:      100,
		QueueTimeout:      30 * time.Second,
	}
}

// ConnectionPool manages database connections with overflow handling
type ConnectionPool struct {
	config *PoolConfig
	pool   *pgxpool.Pool
	mu     sync.RWMutex
	queue  chan struct{}
	closed bool
}

// NewConnectionPool creates a new connection pool
func NewConnectionPool(config *PoolConfig) (*ConnectionPool, error) {
	if config == nil {
		config = DefaultPoolConfig()
	}

	cp := &ConnectionPool{
		config: config,
		queue:  make(chan struct{}, config.MaxQueueSize),
	}

	// Build connection string
	connString := buildConnectionString(config)

	// Configure pgx pool
	poolConfig, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse connection string: %w", err)
	}

	// Apply pool settings
	poolConfig.MaxConns = int32(config.MaxConnections)
	poolConfig.MinConns = int32(config.MinConnections)
	poolConfig.MaxConnLifetime = config.MaxConnLifetime
	poolConfig.MaxConnIdleTime = config.MaxConnIdleTime
	poolConfig.HealthCheckPeriod = config.HealthCheckPeriod
	poolConfig.ConnConfig.ConnectTimeout = config.ConnectTimeout

	// Create pool
	ctx, cancel := context.WithTimeout(context.Background(), config.ConnectTimeout)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	cp.pool = pool

	return cp, nil
}

// buildConnectionString creates a PostgreSQL connection string
func buildConnectionString(config *PoolConfig) string {
	return fmt.Sprintf(
		"host=%s port=%d dbname=%s user=%s password=%s sslmode=%s",
		config.Host,
		config.Port,
		config.Database,
		config.User,
		config.Password,
		config.SSLMode,
	)
}

// GetConnection acquires a connection from the pool
func (cp *ConnectionPool) GetConnection(ctx context.Context) (*pgxpool.Conn, error) {
	cp.mu.RLock()
	if cp.closed {
		cp.mu.RUnlock()
		return nil, fmt.Errorf("connection pool is closed")
	}
	cp.mu.RUnlock()

	// Handle overflow based on mode
	switch cp.config.Mode {
	case PoolModeReject:
		return cp.getWithReject(ctx)
	case PoolModeQueue:
		return cp.getWithQueue(ctx)
	case PoolModeWait:
		return cp.getWithWait(ctx)
	default:
		return cp.getWithWait(ctx)
	}
}

// getWithReject immediately rejects if pool is full
func (cp *ConnectionPool) getWithReject(ctx context.Context) (*pgxpool.Conn, error) {
	select {
	case cp.queue <- struct{}{}:
		defer func() { <-cp.queue }()
		return cp.pool.Acquire(ctx)
	default:
		return nil, fmt.Errorf("connection pool is at capacity")
	}
}

// getWithQueue queues connection requests
func (cp *ConnectionPool) getWithQueue(ctx context.Context) (*pgxpool.Conn, error) {
	// Add to queue
	select {
	case cp.queue <- struct{}{}:
		defer func() { <-cp.queue }()
	case <-ctx.Done():
		return nil, fmt.Errorf("connection request cancelled")
	}

	// Try to acquire with timeout
	acquireCtx, cancel := context.WithTimeout(ctx, cp.config.QueueTimeout)
	defer cancel()

	return cp.pool.Acquire(acquireCtx)
}

// getWithWait waits indefinitely for a connection
func (cp *ConnectionPool) getWithWait(ctx context.Context) (*pgxpool.Conn, error) {
	return cp.pool.Acquire(ctx)
}

// Execute executes a SQL query within a connection
func (cp *ConnectionPool) Execute(ctx context.Context, sql string, args ...interface{}) error {
	conn, err := cp.GetConnection(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()

	_, err = conn.Exec(ctx, sql, args...)
	return err
}

// Query executes a query and returns the results
func (cp *ConnectionPool) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	conn, err := cp.GetConnection(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	return conn.Query(ctx, sql, args...)
}

// QueryRow executes a query that returns at most one row
func (cp *ConnectionPool) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	return cp.pool.QueryRow(ctx, sql, args...)
}

// Close closes the connection pool
func (cp *ConnectionPool) Close() {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	if !cp.closed {
		cp.pool.Close()
		cp.closed = true
	}
}

// Stats returns pool statistics
func (cp *ConnectionPool) Stats() *PoolStats {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	stat := cp.pool.Stat()
	return &PoolStats{
		TotalConnections: int(stat.TotalConns()),
		IdleConnections:  int(stat.IdleConns()),
		AcquireCount:     int(stat.AcquireCount()),
		AcquireDuration:  stat.AcquireDuration().Milliseconds(),
		QueueSize:        len(cp.queue),
		MaxQueueSize:     cap(cp.queue),
		Closed:           cp.closed,
	}
}

// PoolStats holds connection pool statistics
type PoolStats struct {
	TotalConnections int
	IdleConnections  int
	AcquireCount     int
	AcquireDuration  int64
	QueueSize        int
	MaxQueueSize     int
	Closed           bool
}

// Health checks the health of the connection pool
func (cp *ConnectionPool) Health(ctx context.Context) error {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	if cp.closed {
		return fmt.Errorf("connection pool is closed")
	}

	// Ping the database
	return cp.pool.Ping(ctx)
}

package database

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/open-db9/db9/pkg/logger"
)

// SnapshotStatus represents the current state of a snapshot
type SnapshotStatus string

const (
	SnapshotStatusCreating SnapshotStatus = "creating"
	SnapshotStatusReady    SnapshotStatus = "ready"
	SnapshotStatusFailed   SnapshotStatus = "failed"
	SnapshotStatusExpired  SnapshotStatus = "expired"
)

// Snapshot represents a database snapshot
type Snapshot struct {
	ID           string          `json:"id"`
	DatabaseID   string          `json:"database_id"`
	Name         string          `json:"name"`
	SizeBytes    int64           `json:"size_bytes"`
	StoragePath  string          `json:"storage_path"`
	Status       SnapshotStatus  `json:"status"`
	CreatedAt    time.Time       `json:"created_at"`
	ExpiresAt    *time.Time      `json:"expires_at,omitempty"`
	Description  string          `json:"description,omitempty"`
	Error        string          `json:"error,omitempty"`
	DatabaseName string          `json:"database_name"`
	mu           sync.RWMutex    `json:"-"`
}

// SnapshotConfig defines snapshot management configuration
type SnapshotConfig struct {
	MaxSnapshotsPerDB  int           `json:"max_snapshots_per_db"`   // Default: 10
	DefaultRetention   time.Duration `json:"default_retention"`       // Default: 30 days
	StoragePath        string        `json:"storage_path"`            // Default: /data/db9-snapshots
	CompressionEnabled bool          `json:"compression_enabled"`     // Default: true (gzip)
	PgDumpPath         string        `json:"pg_dump_path"`            // Default: pg_dump
	PsqlPath           string        `json:"psql_path"`              // Default: psql
}

// SnapshotManager defines the interface for snapshot operations
type SnapshotManager interface {
	// CreateSnapshot initiates an async snapshot creation
	// Returns immediately with snapshot ID and status "creating"
	CreateSnapshot(ctx context.Context, databaseID string, name string) (*Snapshot, error)

	// ListSnapshots returns all snapshots for a database
	ListSnapshots(ctx context.Context, databaseID string) ([]Snapshot, error)

	// GetSnapshot retrieves a single snapshot by ID
	GetSnapshot(ctx context.Context, snapshotID string) (*Snapshot, error)

	// DeleteSnapshot removes a snapshot
	// confirm must be true to prevent accidental deletion
	DeleteSnapshot(ctx context.Context, snapshotID string, confirm bool) error

	// RestoreFromSnapshot restores a database from a snapshot
	// This is an async operation
	RestoreFromSnapshot(ctx context.Context, databaseID, snapshotID string) error

	// GetSnapshotStatus returns the current status of a snapshot
	GetSnapshotStatus(ctx context.Context, snapshotID string) (SnapshotStatus, error)

	// CreateSnapshotSync creates a snapshot synchronously
	CreateSnapshotSync(ctx context.Context, databaseID string, name string) (*Snapshot, error)
}

// DefaultSnapshotConfig returns the default snapshot configuration
func DefaultSnapshotConfig() *SnapshotConfig {
	return &SnapshotConfig{
		MaxSnapshotsPerDB:  10,
		DefaultRetention:   30 * 24 * time.Hour, // 30 days
		StoragePath:        "/data/db9-snapshots",
		CompressionEnabled: true,
		PgDumpPath:         "pg_dump",
		PsqlPath:           "psql",
	}
}

// PostgreSQLSnapshotManager implements SnapshotManager for PostgreSQL
type PostgreSQLSnapshotManager struct {
	pool      *ConnectionPool
	config    *SnapshotConfig
	snapshots map[string]*Snapshot
	mu        sync.RWMutex
}

// NewPostgreSQLSnapshotManager creates a new PostgreSQL snapshot manager
func NewPostgreSQLSnapshotManager(pool *ConnectionPool, config *SnapshotConfig) (*PostgreSQLSnapshotManager, error) {
	if config == nil {
		config = DefaultSnapshotConfig()
	}

	// Ensure snapshot directory exists with restricted permissions (0750 = owner/group full, others none)
	if err := os.MkdirAll(config.StoragePath, 0750); err != nil {
		return nil, fmt.Errorf("failed to create snapshot directory: %w", err)
	}

	sm := &PostgreSQLSnapshotManager{
		pool:      pool,
		config:    config,
		snapshots: make(map[string]*Snapshot),
	}

	// Load existing snapshots from disk
	if err := sm.loadExistingSnapshots(); err != nil {
		logger.Debug("Failed to load existing snapshots: %v", err)
	}

	return sm, nil
}

// loadExistingSnapshots loads snapshots from the filesystem
func (sm *PostgreSQLSnapshotManager) loadExistingSnapshots() error {
	entries, err := os.ReadDir(sm.config.StoragePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		databaseID := entry.Name()
		dbDir := filepath.Join(sm.config.StoragePath, databaseID)

		files, err := os.ReadDir(dbDir)
		if err != nil {
			continue
		}

		for _, file := range files {
			var snapshotID string
			if sm.config.CompressionEnabled {
				if strings.HasSuffix(file.Name(), ".sql.gz") {
					snapshotID = strings.TrimSuffix(file.Name(), ".sql.gz")
				}
			} else {
				if strings.HasSuffix(file.Name(), ".sql") {
					snapshotID = strings.TrimSuffix(file.Name(), ".sql")
				}
			}

			if snapshotID != "" {
				filePath := filepath.Join(dbDir, file.Name())

				info, err := file.Info()
				if err != nil {
					continue
				}

				expiresAt := info.ModTime().Add(sm.config.DefaultRetention)

				snapshot := &Snapshot{
					ID:          snapshotID,
					DatabaseID:  databaseID,
					Name:        snapshotID,
					Status:      SnapshotStatusReady,
					SizeBytes:   info.Size(),
					StoragePath: filePath,
					CreatedAt:   info.ModTime(),
					ExpiresAt:   &expiresAt,
				}

				sm.mu.Lock()
				sm.snapshots[snapshotID] = snapshot
				sm.mu.Unlock()
			}
		}
	}

	return nil
}

// CreateSnapshot initiates an async snapshot creation
func (sm *PostgreSQLSnapshotManager) CreateSnapshot(ctx context.Context, databaseID string, name string) (*Snapshot, error) {
	return sm.createSnapshot(ctx, databaseID, name, false)
}

// CreateSnapshotSync creates a snapshot synchronously
func (sm *PostgreSQLSnapshotManager) CreateSnapshotSync(ctx context.Context, databaseID string, name string) (*Snapshot, error) {
	return sm.createSnapshot(ctx, databaseID, name, true)
}

// validateDatabaseID validates database ID to prevent path traversal attacks
func validateDatabaseID(databaseID string) error {
	if databaseID == "" || len(databaseID) > 64 {
		return fmt.Errorf("database ID must be 1-64 characters")
	}
	// Check for path traversal attempts
	if strings.ContainsAny(databaseID, "\\/:") {
		return fmt.Errorf("invalid database ID: contains path separators")
	}
	// Check for absolute paths
	if filepath.IsAbs(databaseID) {
		return fmt.Errorf("database ID cannot be absolute path")
	}
	return nil
}

// createSnapshot creates a snapshot (sync or async)
func (sm *PostgreSQLSnapshotManager) createSnapshot(ctx context.Context, databaseID string, name string, sync bool) (*Snapshot, error) {
	// Validate database ID to prevent path traversal attacks
	if err := validateDatabaseID(databaseID); err != nil {
		return nil, err
	}

	snapshotID := uuid.New().String()
	if name == "" {
		name = fmt.Sprintf("snapshot-%s", time.Now().Format("20060102-150405"))
	}

	// Get database name from pool config
	dbName := sm.pool.config.Database
	if dbName == "" || dbName == "postgres" {
		dbName = databaseID
	}

	// Create snapshot directory with restricted permissions (0750 = owner/group full, others none)
	snapshotDir := filepath.Join(sm.config.StoragePath, databaseID)
	if err := os.MkdirAll(snapshotDir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create snapshot directory: %w", err)
	}

	// Determine file path
	var snapshotPath string
	if sm.config.CompressionEnabled {
		snapshotPath = filepath.Join(snapshotDir, snapshotID+".sql.gz")
	} else {
		snapshotPath = filepath.Join(snapshotDir, snapshotID+".sql")
	}

	// Calculate expiration
	expiresAt := time.Now().Add(sm.config.DefaultRetention)

	snapshot := &Snapshot{
		ID:           snapshotID,
		DatabaseID:   databaseID,
		Name:         name,
		Status:       SnapshotStatusCreating,
		StoragePath:  snapshotPath,
		CreatedAt:    time.Now(),
		ExpiresAt:    &expiresAt,
		DatabaseName: dbName,
	}

	sm.mu.Lock()
	sm.snapshots[snapshotID] = snapshot
	sm.mu.Unlock()

	if sync {
		// Synchronous creation
		if err := sm.createSnapshotFile(ctx, snapshot); err != nil {
			snapshot.mu.Lock()
			snapshot.Status = SnapshotStatusFailed
			snapshot.Error = err.Error()
			snapshot.mu.Unlock()
			return nil, fmt.Errorf("failed to create snapshot: %w", err)
		}

		snapshot.mu.Lock()
		snapshot.Status = SnapshotStatusReady
		snapshot.mu.Unlock()

		// Apply retention policy
		if err := sm.applyRetentionPolicy(ctx, databaseID); err != nil {
			logger.Debug("Failed to apply retention policy: %v", err)
		}

		return snapshot, nil
	}

	// Asynchronous creation
	go func() {
		// Create a new context with timeout for async operation
		asyncCtx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()

		if err := sm.createSnapshotFile(asyncCtx, snapshot); err != nil {
			snapshot.mu.Lock()
			snapshot.Status = SnapshotStatusFailed
			snapshot.Error = err.Error()
			snapshot.mu.Unlock()
			logger.Error("Async snapshot creation failed: %v", err)
			return
		}

		snapshot.mu.Lock()
		snapshot.Status = SnapshotStatusReady
		snapshot.mu.Unlock()

		// Apply retention policy
		if err := sm.applyRetentionPolicy(asyncCtx, databaseID); err != nil {
			logger.Debug("Failed to apply retention policy: %v", err)
		}
	}()

	return snapshot, nil
}

// createSnapshotFile executes pg_dump and creates the snapshot file
func (sm *PostgreSQLSnapshotManager) createSnapshotFile(ctx context.Context, snapshot *Snapshot) error {
	// Check available disk space (estimate 2x database size)
	requiredSpace := int64(100 * 1024 * 1024) // 100MB minimum
	if err := checkDiskSpace(sm.config.StoragePath, requiredSpace); err != nil {
		return fmt.Errorf("insufficient disk space: %w", err)
	}

	// Build pg_dump command with separated parameters to prevent command injection
	args := []string{
		"--no-owner",
		"--no-acl",
		"--host", sm.pool.config.Host,
		"--port", fmt.Sprintf("%d", sm.pool.config.Port),
		"--username", sm.pool.config.User,
		"--dbname", snapshot.DatabaseName,
	}

	cmd := exec.CommandContext(ctx, sm.config.PgDumpPath, args...)
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("PGPASSWORD=%s", sm.pool.config.Password),
	)

	// Create output file with restricted permissions (0600 = owner read/write only)
	file, err := os.OpenFile(snapshot.StoragePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to create snapshot file: %w", err)
	}
	defer file.Close()

	var writer io.Writer = file
	var gzWriter *gzip.Writer

	if sm.config.CompressionEnabled {
		gzWriter = gzip.NewWriter(file)
		defer gzWriter.Close()
		writer = gzWriter
	}

	// Pipe pg_dump output to file
	cmd.Stdout = writer
	cmd.Stderr = os.Stderr

	logger.Info("Creating snapshot %s for database %s", snapshot.ID, snapshot.DatabaseID)

	if err := cmd.Run(); err != nil {
		os.Remove(snapshot.StoragePath)
		return fmt.Errorf("pg_dump failed: %w", err)
	}

	// Ensure gzip writer is flushed
	if gzWriter != nil {
		if err := gzWriter.Close(); err != nil {
			return fmt.Errorf("failed to close gzip writer: %w", err)
		}
	}

	// Get file size
	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	snapshot.SizeBytes = info.Size()
	logger.Info("Snapshot %s created successfully (%d bytes)", snapshot.ID, snapshot.SizeBytes)

	return nil
}

// RestoreFromSnapshot restores a database from a snapshot
func (sm *PostgreSQLSnapshotManager) RestoreFromSnapshot(ctx context.Context, databaseID, snapshotID string) error {
	sm.mu.RLock()
	snapshot, exists := sm.snapshots[snapshotID]
	sm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("snapshot not found: %s", snapshotID)
	}

	if snapshot.Status != SnapshotStatusReady {
		return fmt.Errorf("snapshot is not ready (status: %s)", snapshot.Status)
	}

	// Verify snapshot belongs to the database
	if snapshot.DatabaseID != databaseID {
		return fmt.Errorf("snapshot %s does not belong to database %s", snapshotID, databaseID)
	}

	// Open the snapshot file
	file, err := os.Open(snapshot.StoragePath)
	if err != nil {
		return fmt.Errorf("failed to open snapshot file: %w", err)
	}
	defer file.Close()

	var reader io.Reader = file
	if sm.config.CompressionEnabled {
		gzReader, err := gzip.NewReader(file)
		if err != nil {
			return fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer gzReader.Close()
		reader = gzReader
	}

	// Build psql command with separated parameters to prevent command injection
	args := []string{
		"--host", sm.pool.config.Host,
		"--port", fmt.Sprintf("%d", sm.pool.config.Port),
		"--username", sm.pool.config.User,
		"--dbname", snapshot.DatabaseName,
	}

	cmd := exec.CommandContext(ctx, sm.config.PsqlPath, args...)
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("PGPASSWORD=%s", sm.pool.config.Password),
	)

	cmd.Stdin = reader
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	logger.Info("Restoring database %s from snapshot %s", databaseID, snapshotID)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("psql restore failed: %w", err)
	}

	logger.Info("Database %s restored successfully from snapshot %s", databaseID, snapshotID)

	return nil
}

// ListSnapshots returns all snapshots for a database
func (sm *PostgreSQLSnapshotManager) ListSnapshots(ctx context.Context, databaseID string) ([]Snapshot, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	var result []Snapshot
	for _, snapshot := range sm.snapshots {
		if snapshot.DatabaseID == databaseID {
			snapshot.mu.RLock()
			// Create a copy without the mutex
			result = append(result, Snapshot{
				ID:           snapshot.ID,
				DatabaseID:   snapshot.DatabaseID,
				Name:         snapshot.Name,
				SizeBytes:    snapshot.SizeBytes,
				StoragePath:  snapshot.StoragePath,
				Status:       snapshot.Status,
				CreatedAt:    snapshot.CreatedAt,
				ExpiresAt:    snapshot.ExpiresAt,
				Description:  snapshot.Description,
				Error:        snapshot.Error,
				DatabaseName: snapshot.DatabaseName,
			})
			snapshot.mu.RUnlock()
		}
	}

	return result, nil
}

// GetSnapshot retrieves a single snapshot by ID
func (sm *PostgreSQLSnapshotManager) GetSnapshot(ctx context.Context, snapshotID string) (*Snapshot, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	snapshot, exists := sm.snapshots[snapshotID]
	if !exists {
		return nil, fmt.Errorf("snapshot not found: %s", snapshotID)
	}

	snapshot.mu.RLock()
	defer snapshot.mu.RUnlock()

	// Return a copy without the mutex to avoid external modifications
	return &Snapshot{
		ID:           snapshot.ID,
		DatabaseID:   snapshot.DatabaseID,
		Name:         snapshot.Name,
		SizeBytes:    snapshot.SizeBytes,
		StoragePath:  snapshot.StoragePath,
		Status:       snapshot.Status,
		CreatedAt:    snapshot.CreatedAt,
		ExpiresAt:    snapshot.ExpiresAt,
		Description:  snapshot.Description,
		Error:        snapshot.Error,
		DatabaseName: snapshot.DatabaseName,
	}, nil
}

// DeleteSnapshot removes a snapshot
func (sm *PostgreSQLSnapshotManager) DeleteSnapshot(ctx context.Context, snapshotID string, confirm bool) error {
	if !confirm {
		return fmt.Errorf("confirmation required to delete snapshot")
	}

	sm.mu.Lock()
	snapshot, exists := sm.snapshots[snapshotID]
	if !exists {
		sm.mu.Unlock()
		return fmt.Errorf("snapshot not found: %s", snapshotID)
	}
	delete(sm.snapshots, snapshotID)
	sm.mu.Unlock()

	// Delete the file
	if err := os.Remove(snapshot.StoragePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete snapshot file: %w", err)
	}

	// Remove directory if empty
	dir := filepath.Dir(snapshot.StoragePath)
	entries, err := os.ReadDir(dir)
	if err == nil && len(entries) == 0 {
		os.Remove(dir)
	}

	logger.Info("Snapshot %s deleted", snapshotID)

	return nil
}

// GetSnapshotStatus returns the current status of a snapshot
func (sm *PostgreSQLSnapshotManager) GetSnapshotStatus(ctx context.Context, snapshotID string) (SnapshotStatus, error) {
	snapshot, err := sm.GetSnapshot(ctx, snapshotID)
	if err != nil {
		return "", err
	}
	return snapshot.Status, nil
}

// applyRetentionPolicy removes old snapshots exceeding the retention limits
func (sm *PostgreSQLSnapshotManager) applyRetentionPolicy(ctx context.Context, databaseID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	var dbSnapshots []*Snapshot
	now := time.Now()

	// Filter snapshots for this database
	for _, snapshot := range sm.snapshots {
		if snapshot.DatabaseID == databaseID {
			dbSnapshots = append(dbSnapshots, snapshot)
		}
	}

	// Remove expired snapshots
	for _, snapshot := range dbSnapshots {
		if snapshot.ExpiresAt != nil && snapshot.ExpiresAt.Before(now) {
			logger.Info("Deleting expired snapshot %s", snapshot.ID)
			delete(sm.snapshots, snapshot.ID)
			os.Remove(snapshot.StoragePath)
		}
	}

	// Rebuild list after removing expired
	dbSnapshots = nil
	for _, snapshot := range sm.snapshots {
		if snapshot.DatabaseID == databaseID {
			dbSnapshots = append(dbSnapshots, snapshot)
		}
	}

	// If we still have too many snapshots, remove the oldest ones
	if len(dbSnapshots) > sm.config.MaxSnapshotsPerDB {
		sort.Slice(dbSnapshots, func(i, j int) bool {
			return dbSnapshots[i].CreatedAt.Before(dbSnapshots[j].CreatedAt)
		})

		// Remove oldest snapshots
		toRemove := len(dbSnapshots) - sm.config.MaxSnapshotsPerDB
		for i := 0; i < toRemove; i++ {
			snapshot := dbSnapshots[i]
			logger.Info("Deleting old snapshot %s (max limit reached)", snapshot.ID)
			delete(sm.snapshots, snapshot.ID)
			os.Remove(snapshot.StoragePath)
		}
	}

	return nil
}

// CleanupExpiredSnapshots removes all expired snapshots across all databases
func (sm *PostgreSQLSnapshotManager) CleanupExpiredSnapshots(ctx context.Context) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := time.Now()
	var expiredSnapshots []string

	for id, snapshot := range sm.snapshots {
		if snapshot.ExpiresAt != nil && snapshot.ExpiresAt.Before(now) {
			expiredSnapshots = append(expiredSnapshots, id)
		}
	}

	for _, id := range expiredSnapshots {
		snapshot := sm.snapshots[id]
		logger.Info("Cleaning up expired snapshot %s", id)
		delete(sm.snapshots, id)
		os.Remove(snapshot.StoragePath)
	}

	return nil
}

package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/open-db9/db9/internal/database"
)

// MockSnapshotManagerExtended extends the MockSnapshotManager with CreateSnapshotSync
type MockSnapshotManagerExtended struct {
	snapshots map[string]*database.Snapshot
}

// NewMockSnapshotManagerExtended creates a new extended mock snapshot manager
func NewMockSnapshotManagerExtended() database.SnapshotManager {
	return &MockSnapshotManagerExtended{
		snapshots: make(map[string]*database.Snapshot),
	}
}

// CreateSnapshot creates a new snapshot asynchronously
func (m *MockSnapshotManagerExtended) CreateSnapshot(ctx context.Context, databaseID string, name string) (*database.Snapshot, error) {
	id := fmt.Sprintf("snap-%s", uuid.New().String()[:8])
	snapshot := &database.Snapshot{
		ID:          id,
		DatabaseID:  databaseID,
		Name:        name,
		SizeBytes:   0,
		StoragePath: fmt.Sprintf("/data/db9-snapshots/%s/%s.sql", databaseID, id),
		Status:      database.SnapshotStatusCreating,
		CreatedAt:   time.Now(),
	}
	m.snapshots[id] = snapshot

	// Simulate async completion
	go func() {
		time.Sleep(2 * time.Second)
		snapshot.Status = database.SnapshotStatusReady
		snapshot.SizeBytes = 1024 * 1024 * 100 // 100MB
	}()

	return snapshot, nil
}

// CreateSnapshotSync creates a snapshot synchronously
func (m *MockSnapshotManagerExtended) CreateSnapshotSync(ctx context.Context, databaseID string, name string) (*database.Snapshot, error) {
	id := fmt.Sprintf("snap-%s", uuid.New().String()[:8])
	snapshot := &database.Snapshot{
		ID:          id,
		DatabaseID:  databaseID,
		Name:        name,
		SizeBytes:   1024 * 1024 * 100, // 100MB
		StoragePath: fmt.Sprintf("/data/db9-snapshots/%s/%s.sql", databaseID, id),
		Status:      database.SnapshotStatusReady,
		CreatedAt:   time.Now(),
	}
	m.snapshots[id] = snapshot
	return snapshot, nil
}

// ListSnapshots returns all snapshots for a database
func (m *MockSnapshotManagerExtended) ListSnapshots(ctx context.Context, databaseID string) ([]database.Snapshot, error) {
	result := make([]database.Snapshot, 0)
	for _, snap := range m.snapshots {
		if snap.DatabaseID == databaseID {
			result = append(result, *snap)
		}
	}
	return result, nil
}

// GetSnapshot retrieves a single snapshot
func (m *MockSnapshotManagerExtended) GetSnapshot(ctx context.Context, snapshotID string) (*database.Snapshot, error) {
	snap, exists := m.snapshots[snapshotID]
	if !exists {
		return nil, fmt.Errorf("snapshot not found: %s", snapshotID)
	}
	return snap, nil
}

// DeleteSnapshot deletes a snapshot
func (m *MockSnapshotManagerExtended) DeleteSnapshot(ctx context.Context, snapshotID string, confirm bool) error {
	if !confirm {
		return fmt.Errorf("confirmation required")
	}
	if _, exists := m.snapshots[snapshotID]; !exists {
		return fmt.Errorf("snapshot not found: %s", snapshotID)
	}
	delete(m.snapshots, snapshotID)
	return nil
}

// RestoreFromSnapshot restores a database from snapshot
func (m *MockSnapshotManagerExtended) RestoreFromSnapshot(ctx context.Context, databaseID, snapshotID string) error {
	if _, exists := m.snapshots[snapshotID]; !exists {
		return fmt.Errorf("snapshot not found: %s", snapshotID)
	}
	return nil
}

// GetSnapshotStatus returns the status of a snapshot
func (m *MockSnapshotManagerExtended) GetSnapshotStatus(ctx context.Context, snapshotID string) (database.SnapshotStatus, error) {
	snap, exists := m.snapshots[snapshotID]
	if !exists {
		return "", fmt.Errorf("snapshot not found: %s", snapshotID)
	}
	return snap.Status, nil
}

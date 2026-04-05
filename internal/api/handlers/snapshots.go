package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/open-db9/db9/internal/database"
)

// CreateSnapshotRequest represents the request body for creating a snapshot
type CreateSnapshotRequest struct {
	Name string `json:"name"`
}

// SnapshotManagerRegistry manages snapshot managers by database ID
var (
	snapshotManagerRegistry = make(map[int]database.SnapshotManager)
	snapshotRegistryMu      sync.RWMutex
)

// RegisterSnapshotManager registers a snapshot manager for a database
func RegisterSnapshotManager(dbID int, manager database.SnapshotManager) {
	snapshotRegistryMu.Lock()
	defer snapshotRegistryMu.Unlock()
	snapshotManagerRegistry[dbID] = manager
}

// UnregisterSnapshotManager removes a snapshot manager from the registry
func UnregisterSnapshotManager(dbID int) {
	snapshotRegistryMu.Lock()
	defer snapshotRegistryMu.Unlock()
	delete(snapshotManagerRegistry, dbID)
}

// GetSnapshotManager retrieves a snapshot manager by database ID
func GetSnapshotManager(dbID int) (database.SnapshotManager, error) {
	snapshotRegistryMu.RLock()
	defer snapshotRegistryMu.RUnlock()

	manager, exists := snapshotManagerRegistry[dbID]
	if !exists {
		return nil, fmt.Errorf("snapshot manager for database %d not found", dbID)
	}
	return manager, nil
}

// CreateSnapshotHandler handles POST /api/v1/databases/:id/snapshots
func CreateSnapshotHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract database ID from URL
	// Expected path: /api/v1/databases/:id/snapshots
	dbIDStr, err := extractDatabaseID(r, "/api/v1/databases/")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate database ID
	dbID, err := strconv.Atoi(dbIDStr)
	if err != nil || dbID <= 0 || dbID > 1000000 {
		http.Error(w, "Invalid database ID", http.StatusBadRequest)
		return
	}

	// Validate Content-Type
	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "Content-Type must be application/json", http.StatusUnsupportedMediaType)
		return
	}

	// Parse request body
	var req CreateSnapshotRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Validate snapshot name
	if req.Name == "" {
		http.Error(w, "Snapshot name is required", http.StatusBadRequest)
		return
	}

	// Get snapshot manager
	manager, err := GetSnapshotManager(dbID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Create snapshot (async operation)
	ctx := r.Context()
	databaseID := strconv.Itoa(dbID)
	snapshot, err := manager.CreateSnapshot(ctx, databaseID, req.Name)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create snapshot: %v", err), http.StatusInternalServerError)
		return
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted) // 202 Accepted for async operation
	json.NewEncoder(w).Encode(Response{
		Success: true,
		Data:    snapshot,
	})
}

// ListSnapshotsHandler handles GET /api/v1/databases/:id/snapshots
func ListSnapshotsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract database ID from URL
	dbIDStr, err := extractDatabaseID(r, "/api/v1/databases/")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	dbID, err := strconv.Atoi(dbIDStr)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid database ID: %s", dbIDStr), http.StatusBadRequest)
		return
	}

	// Get snapshot manager
	manager, err := GetSnapshotManager(dbID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// List snapshots
	ctx := r.Context()
	databaseID := strconv.Itoa(dbID)
	snapshots, err := manager.ListSnapshots(ctx, databaseID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list snapshots: %v", err), http.StatusInternalServerError)
		return
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(Response{
		Success: true,
		Data: map[string]interface{}{
			"snapshots": snapshots,
			"count":     len(snapshots),
		},
	})
}

// GetSnapshotHandler handles GET /api/v1/databases/:id/snapshots/:sid
func GetSnapshotHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract database ID and snapshot ID from URL
	// Expected path: /api/v1/databases/:id/snapshots/:sid
	dbID, snapshotID, err := extractDatabaseAndSnapshotID(r.URL.Path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Get snapshot manager
	manager, err := GetSnapshotManager(dbID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Get snapshot
	ctx := r.Context()
	snapshot, err := manager.GetSnapshot(ctx, snapshotID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
		} else {
			http.Error(w, fmt.Sprintf("Failed to get snapshot: %v", err), http.StatusInternalServerError)
		}
		return
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(Response{
		Success: true,
		Data:    snapshot,
	})
}

// DeleteSnapshotHandler handles DELETE /api/v1/databases/:id/snapshots/:sid
func DeleteSnapshotHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract database ID and snapshot ID from URL
	dbID, snapshotID, err := extractDatabaseAndSnapshotID(r.URL.Path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Check confirmation query parameter
	confirm := r.URL.Query().Get("confirm")
	if confirm != "true" {
		http.Error(w, "Confirmation required. Add ?confirm=true to confirm deletion", http.StatusBadRequest)
		return
	}

	// Get snapshot manager
	manager, err := GetSnapshotManager(dbID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Delete snapshot
	ctx := r.Context()
	if err := manager.DeleteSnapshot(ctx, snapshotID, true); err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
		} else {
			http.Error(w, fmt.Sprintf("Failed to delete snapshot: %v", err), http.StatusInternalServerError)
		}
		return
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(Response{
		Success: true,
		Message: "Snapshot deleted successfully",
	})
}

// RestoreSnapshotHandler handles POST /api/v1/databases/:id/snapshots/:sid/restore
func RestoreSnapshotHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract database ID and snapshot ID from URL
	// Expected path: /api/v1/databases/:id/snapshots/:sid/restore
	dbID, snapshotID, err := extractDatabaseAndSnapshotIDFromRestorePath(r.URL.Path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Get snapshot manager
	manager, err := GetSnapshotManager(dbID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Restore from snapshot (async operation)
	ctx := r.Context()
	databaseID := strconv.Itoa(dbID)
	if err := manager.RestoreFromSnapshot(ctx, databaseID, snapshotID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
		} else {
			http.Error(w, fmt.Sprintf("Failed to restore snapshot: %v", err), http.StatusInternalServerError)
		}
		return
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted) // 202 Accepted for async operation
	json.NewEncoder(w).Encode(Response{
		Success: true,
		Message: "Snapshot restoration initiated",
		Data: map[string]interface{}{
			"snapshot_id": snapshotID,
			"status":      "restoring",
		},
	})
}

// Helper functions

// extractDatabaseAndSnapshotID extracts database ID and snapshot ID from URL path
// Expected format: /api/v1/databases/:id/snapshots/:sid
func extractDatabaseAndSnapshotID(path string) (int, string, error) {
	prefix := "/api/v1/databases/"
	if !strings.HasPrefix(path, prefix) {
		return 0, "", fmt.Errorf("invalid URL format")
	}

	rest := strings.TrimPrefix(path, prefix)
	parts := strings.Split(rest, "/")

	if len(parts) < 4 || parts[1] != "snapshots" {
		return 0, "", fmt.Errorf("invalid URL format. Expected: /api/v1/databases/:id/snapshots/:sid")
	}

	dbID, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, "", fmt.Errorf("invalid database ID: %s", parts[0])
	}

	snapshotID := parts[2]
	if snapshotID == "" {
		return 0, "", fmt.Errorf("snapshot ID is required")
	}

	return dbID, snapshotID, nil
}

// extractDatabaseAndSnapshotIDFromRestorePath extracts database ID and snapshot ID from restore path
// Expected format: /api/v1/databases/:id/snapshots/:sid/restore
func extractDatabaseAndSnapshotIDFromRestorePath(path string) (int, string, error) {
	prefix := "/api/v1/databases/"
	if !strings.HasPrefix(path, prefix) {
		return 0, "", fmt.Errorf("invalid URL format")
	}

	rest := strings.TrimPrefix(path, prefix)
	parts := strings.Split(rest, "/")

	if len(parts) < 5 || parts[1] != "snapshots" || parts[3] != "restore" {
		return 0, "", fmt.Errorf("invalid URL format. Expected: /api/v1/databases/:id/snapshots/:sid/restore")
	}

	dbID, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, "", fmt.Errorf("invalid database ID: %s", parts[0])
	}

	snapshotID := parts[2]
	if snapshotID == "" {
		return 0, "", fmt.Errorf("snapshot ID is required")
	}

	return dbID, snapshotID, nil
}

// MockSnapshotManager is a simple in-memory implementation for testing
type MockSnapshotManager struct {
	snapshots map[string]*database.Snapshot
}

// NewMockSnapshotManager creates a new mock snapshot manager
func NewMockSnapshotManager() *MockSnapshotManager {
	return &MockSnapshotManager{
		snapshots: make(map[string]*database.Snapshot),
	}
}

// CreateSnapshot creates a new snapshot
func (m *MockSnapshotManager) CreateSnapshot(ctx context.Context, databaseID string, name string) (*database.Snapshot, error) {
	id := fmt.Sprintf("snap-%s", uuid.New().String()[:8])
	snapshot := &database.Snapshot{
		ID:         id,
		DatabaseID: databaseID,
		Name:       name,
		SizeBytes:  0,
		StoragePath: fmt.Sprintf("/data/db9-snapshots/%s/%s.sql", databaseID, id),
		Status:     database.SnapshotStatusCreating,
		CreatedAt:  time.Now(),
	}
	m.snapshots[id] = snapshot

	// Simulate async completion - in real implementation, this would be a background job
	go func() {
		time.Sleep(2 * time.Second)
		snapshot.Status = database.SnapshotStatusReady
		snapshot.SizeBytes = 1024 * 1024 * 100 // 100MB
	}()

	return snapshot, nil
}

// ListSnapshots returns all snapshots for a database
func (m *MockSnapshotManager) ListSnapshots(ctx context.Context, databaseID string) ([]database.Snapshot, error) {
	result := make([]database.Snapshot, 0)
	for _, snap := range m.snapshots {
		if snap.DatabaseID == databaseID {
			result = append(result, *snap)
		}
	}
	return result, nil
}

// GetSnapshot retrieves a single snapshot
func (m *MockSnapshotManager) GetSnapshot(ctx context.Context, snapshotID string) (*database.Snapshot, error) {
	snap, exists := m.snapshots[snapshotID]
	if !exists {
		return nil, fmt.Errorf("snapshot not found: %s", snapshotID)
	}
	return snap, nil
}

// DeleteSnapshot deletes a snapshot
func (m *MockSnapshotManager) DeleteSnapshot(ctx context.Context, snapshotID string, confirm bool) error {
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
func (m *MockSnapshotManager) RestoreFromSnapshot(ctx context.Context, databaseID, snapshotID string) error {
	if _, exists := m.snapshots[snapshotID]; !exists {
		return fmt.Errorf("snapshot not found: %s", snapshotID)
	}
	// In real implementation, this would trigger pg_restore
	return nil
}

// GetSnapshotStatus returns the status of a snapshot
func (m *MockSnapshotManager) GetSnapshotStatus(ctx context.Context, snapshotID string) (database.SnapshotStatus, error) {
	snap, exists := m.snapshots[snapshotID]
	if !exists {
		return "", fmt.Errorf("snapshot not found: %s", snapshotID)
	}
	return snap.Status, nil
}

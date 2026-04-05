package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

// Branch operations use the existing dbRegistry from sql.go
// No separate registry needed - branches are just database operations

// CreateBranchRequest represents the request body for creating a branch
type CreateBranchRequest struct {
	Name string `json:"name"`
}

// BranchResponse represents the response for branch operations
type BranchResponse struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	ParentID   *string `json:"parent_id,omitempty"`
	SnapshotID *string `json:"snapshot_id,omitempty"`
	CreatedAt  string  `json:"created_at"`
	Status     string  `json:"status"`
}

// CreateBranchHandler handles POST /api/v1/databases/:id/branches
func CreateBranchHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract database ID from URL path
	// Expected path: /api/v1/databases/:id/branches
	dbIDStr, err := extractDatabaseIDFromBranchPath(r.URL.Path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate database ID
	dbID, err := strconv.Atoi(dbIDStr)
	if err != nil || dbID <= 0 {
		http.Error(w, "Invalid database ID", http.StatusBadRequest)
		return
	}

	// Validate Content-Type
	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "Content-Type must be application/json", http.StatusUnsupportedMediaType)
		return
	}

	// Parse request body
	var req CreateBranchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Validate branch name
	if req.Name == "" {
		http.Error(w, "Branch name is required", http.StatusBadRequest)
		return
	}

	// Get branch manager from registry
	manager, err := GetDatabase(dbID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Create branch
	ctx := r.Context()

	// Manager.CreateBranch expects uuid.UUID
	// Convert dbID (int) to a UUID for lookup
	// Note: This is a simplified conversion - in production, you would
	// look up the actual database UUID from the ID
	sourceUUID := uuid.NewMD5(uuid.Nil, []byte(strconv.Itoa(dbID)))

	branch, err := manager.CreateBranch(ctx, sourceUUID, req.Name)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
		} else {
			http.Error(w, fmt.Sprintf("Failed to create branch: %v", err), http.StatusInternalServerError)
		}
		return
	}

	// Convert DatabaseBranch to BranchResponse
	var parentIDStr, snapshotIDStr *string
	if branch.ParentID != nil {
		s := branch.ParentID.String()
		parentIDStr = &s
	}
	if branch.SnapshotID != nil {
		s := branch.SnapshotID.String()
		snapshotIDStr = &s
	}

	response := BranchResponse{
		ID:         branch.ID.String(),
		Name:       branch.Name,
		ParentID:   parentIDStr,
		SnapshotID: snapshotIDStr,
		CreatedAt:  branch.CreatedAt.Format("2006-01-02T15:04:05Z"),
		Status:     "active",
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(Response{
		Success: true,
		Data:    response,
	})
}

// ListBranchesHandler handles GET /api/v1/databases/:id/branches
func ListBranchesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract database ID
	dbIDStr, err := extractDatabaseIDFromBranchPath(r.URL.Path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	dbID, err := strconv.Atoi(dbIDStr)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid database ID: %s", dbIDStr), http.StatusBadRequest)
		return
	}

	// Get branch manager
	manager, err := GetDatabase(dbID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// List branches - expects uuid.UUID
	ctx := r.Context()
	sourceUUID := uuid.NewMD5(uuid.Nil, []byte(strconv.Itoa(dbID)))

	branches, err := manager.ListBranches(ctx, sourceUUID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list branches: %v", err), http.StatusInternalServerError)
		return
	}

	// Convert to response format
	response := make([]BranchResponse, len(branches))
	for i, b := range branches {
		var parentIDStr, snapshotIDStr *string
		if b.ParentID != nil {
			s := b.ParentID.String()
			parentIDStr = &s
		}
		if b.SnapshotID != nil {
			s := b.SnapshotID.String()
			snapshotIDStr = &s
		}

		response[i] = BranchResponse{
			ID:         b.ID.String(),
			Name:       b.Name,
			ParentID:   parentIDStr,
			SnapshotID: snapshotIDStr,
			CreatedAt:  b.CreatedAt.Format("2006-01-02T15:04:05Z"),
			Status:     "active",
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(Response{
		Success: true,
		Data: map[string]interface{}{
			"branches": response,
			"count":    len(response),
		},
	})
}

// DeleteBranchHandler handles DELETE /api/v1/databases/:id/branches/:bid
func DeleteBranchHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract database ID and branch ID from URL
	// Expected format: /api/v1/databases/:id/branches/:bid
	dbID, branchIDStr, err := extractDatabaseAndBranchID(r.URL.Path)
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

	// Parse branch ID as UUID
	branchUUID, err := uuid.Parse(branchIDStr)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid branch ID format: %v", err), http.StatusBadRequest)
		return
	}

	// Get branch manager from registry using database ID
	manager, err := GetDatabase(dbID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Delete branch using the manager
	ctx := r.Context()
	if err := manager.DeleteBranch(ctx, branchUUID, true); err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
		} else {
			http.Error(w, fmt.Sprintf("Failed to delete branch: %v", err), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(Response{
		Success: true,
		Message: "Branch deleted successfully",
	})
}

// Helper functions

// extractDatabaseAndBranchID extracts database ID and branch ID from /api/v1/databases/:id/branches/:bid
func extractDatabaseAndBranchID(path string) (int, string, error) {
	prefix := "/api/v1/databases/"
	if !strings.HasPrefix(path, prefix) {
		return 0, "", fmt.Errorf("invalid URL format")
	}

	rest := strings.TrimPrefix(path, prefix)
	parts := strings.Split(rest, "/")

	if len(parts) < 4 || parts[1] != "branches" {
		return 0, "", fmt.Errorf("invalid URL format. Expected: /api/v1/databases/:id/branches/:bid")
	}

	dbID, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, "", fmt.Errorf("invalid database ID: %s", parts[0])
	}

	branchID := parts[2]
	if branchID == "" {
		return 0, "", fmt.Errorf("branch ID is required")
	}

	return dbID, branchID, nil
}

// extractDatabaseIDFromBranchPath extracts database ID from /api/v1/databases/:id/branches
func extractDatabaseIDFromBranchPath(path string) (string, error) {
	prefix := "/api/v1/databases/"
	if !strings.HasPrefix(path, prefix) {
		return "", fmt.Errorf("invalid URL format")
	}

	rest := strings.TrimPrefix(path, prefix)
	parts := strings.Split(rest, "/")

	if len(parts) < 2 || parts[1] != "branches" {
		return "", fmt.Errorf("invalid URL format. Expected: /api/v1/databases/:id/branches")
	}

	return parts[0], nil
}

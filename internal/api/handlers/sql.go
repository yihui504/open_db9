package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/open-db9/db9/internal/database"
)

const (
	MinQueryTimeout     = 1 * time.Second
	MaxQueryTimeout     = 5 * time.Minute
	DefaultQueryTimeout = 30 * time.Second
)

// SQLRequest represents a SQL execution request
type SQLRequest struct {
	Query     string `json:"query"`
	TimeoutMs *int   `json:"timeout_ms,omitempty"`
}

// SQLResponse represents a SQL execution response
type SQLResponse struct {
	Columns  []interface{}            `json:"columns"`
	Rows     []map[string]interface{} `json:"rows"`
	RowCount int                      `json:"row_count"`
}

// dbRegistry manages database connections by ID
// In production, this should be replaced with proper dependency injection
var (
	dbRegistry = make(map[int]*database.Manager)
	registryMu sync.RWMutex
)

// RegisterDatabase registers a database manager for use by API handlers
func RegisterDatabase(id int, manager *database.Manager) {
	registryMu.Lock()
	defer registryMu.Unlock()
	dbRegistry[id] = manager
}

// UnregisterDatabase removes a database manager from the registry
func UnregisterDatabase(id int) {
	registryMu.Lock()
	defer registryMu.Unlock()
	delete(dbRegistry, id)
}

// GetDatabase retrieves a database manager by ID
func GetDatabase(id int) (*database.Manager, error) {
	registryMu.RLock()
	defer registryMu.RUnlock()

	manager, exists := dbRegistry[id]
	if !exists {
		return nil, fmt.Errorf("database with ID %d not found", id)
	}
	return manager, nil
}

// ExecuteSQLHandler handles SQL execution requests
func ExecuteSQLHandler(w http.ResponseWriter, r *http.Request) {
	// Only POST method is allowed
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Limit request body size to 1MB
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	// Extract database ID from URL path
	// Expected path: /api/v1/databases/:id/sql
	path := strings.TrimSuffix(r.URL.Path, "/")

	parts := strings.Split(path, "/")
	// Expected parts: ["", "api", "v1", "databases", "<id>", "sql"]
	if len(parts) < 6 || parts[5] != "sql" {
		http.Error(w, "Invalid URL format. Expected: /api/v1/databases/:id/sql", http.StatusBadRequest)
		return
	}

	dbIDStr := parts[4]

	// Parse database ID
	dbID, err := strconv.Atoi(dbIDStr)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid database ID: %s", dbIDStr), http.StatusBadRequest)
		return
	}

	// Validate database exists
	manager, err := GetDatabase(dbID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Validate Content-Type
	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "Content-Type must be application/json", http.StatusUnsupportedMediaType)
		return
	}

	// Parse request body
	var req SQLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Validate query is not empty
	if req.Query == "" {
		http.Error(w, "Query cannot be empty", http.StatusBadRequest)
		return
	}

	// Validate SQL query
	if err := validateSQLQuery(req.Query); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Set timeout (default 30 seconds)
	timeout := DefaultQueryTimeout
	if req.TimeoutMs != nil {
		timeout = time.Duration(*req.TimeoutMs) * time.Millisecond
		if timeout < MinQueryTimeout {
			timeout = MinQueryTimeout
		}
		if timeout > MaxQueryTimeout {
			timeout = MaxQueryTimeout
		}
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()

	// Execute query
	rows, err := manager.Raw(ctx, req.Query)
	if err != nil {
		// Check for context timeout
		if errors.Is(err, context.DeadlineExceeded) {
			http.Error(w, "Query execution timeout", http.StatusRequestTimeout)
			return
		}
		http.Error(w, fmt.Sprintf("Query execution failed: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Get column names
	fieldDescriptions := rows.FieldDescriptions()
	columns := make([]interface{}, len(fieldDescriptions))
	for i, fd := range fieldDescriptions {
		columns[i] = string(fd.Name)
	}

	// Collect all rows
	resultMaps, err := pgx.CollectRows(rows, pgx.RowToMap)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to collect results: %v", err), http.StatusInternalServerError)
		return
	}

	// Build response
	response := SQLResponse{
		Columns:  columns,
		Rows:     resultMaps,
		RowCount: len(resultMaps),
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(Response{
		Success: true,
		Data:    response,
	}); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
		return
	}
}

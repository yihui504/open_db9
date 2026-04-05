package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/open-db9/db9/internal/database"
)

// GetDatabaseStatsHandler returns database statistics
// GET /api/v1/databases/:id/metrics/stats
func GetDatabaseStatsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		ct := r.Header.Get("Content-Type")
		if ct != "" && !strings.Contains(ct, "application/json") {
			http.Error(w, "Content-Type must be application/json", http.StatusUnsupportedMediaType)
			return
		}
	}

	// Extract database ID from URL path
	path := strings.TrimSuffix(r.URL.Path, "/")
	parts := strings.Split(path, "/")
	// 路径格式: /api/v1/databases/{id}/metrics/stats
	// parts: ["", "api", "v1", "databases", "{id}", "metrics", "stats"]
	if len(parts) < 7 {
		http.Error(w, "Invalid URL format", http.StatusBadRequest)
		return
	}
	dbIDStr := parts[4]

	dbID, err := strconv.Atoi(dbIDStr)
	if err != nil || dbID <= 0 || dbID > 1000000 {
		http.Error(w, "Invalid database ID", http.StatusBadRequest)
		return
	}

	// Get database manager from registry
	manager, err := GetDatabase(dbID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Database not found: %v", err), http.StatusNotFound)
		return
	}

	// Use the manager's pool directly for querying (same pattern as schema.go)
	pool := manager.GetPool()

	// Collect statistics
	stats, err := collectDatabaseStatsV2(r.Context(), pool, dbIDStr)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to collect database stats: %v", err), http.StatusInternalServerError)
		return
	}

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// GetSlowQueriesHandler returns slow queries from pg_stat_statements
// GET /api/v1/databases/:id/metrics/slow?threshold=1000&limit=20
func GetSlowQueriesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		ct := r.Header.Get("Content-Type")
		if ct != "" && !strings.Contains(ct, "application/json") {
			http.Error(w, "Content-Type must be application/json", http.StatusUnsupportedMediaType)
			return
		}
	}

	// Extract database ID from URL path
	path := strings.TrimSuffix(r.URL.Path, "/")
	parts := strings.Split(path, "/")
	// 路径格式: /api/v1/databases/{id}/metrics/slow
	// parts: ["", "api", "v1", "databases", "{id}", "metrics", "slow"]
	if len(parts) < 7 {
		http.Error(w, "Invalid URL format", http.StatusBadRequest)
		return
	}
	dbIDStr := parts[4]

	dbID, err := strconv.Atoi(dbIDStr)
	if err != nil || dbID <= 0 || dbID > 1000000 {
		http.Error(w, "Invalid database ID", http.StatusBadRequest)
		return
	}

	// Get threshold from query params (default 1000ms, must be > 0)
	threshold := int64(1000)
	if thresholdStr := r.URL.Query().Get("threshold"); thresholdStr != "" {
		threshold, err = strconv.ParseInt(thresholdStr, 10, 64)
		if err != nil || threshold <= 0 {
			http.Error(w, "Invalid threshold (must be > 0)", http.StatusBadRequest)
			return
		}
	}

	// Get limit from query params (default 20, range 1-100)
	limit := 20
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		limit, err = strconv.Atoi(limitStr)
		if err != nil || limit < 1 || limit > 100 {
			http.Error(w, "Invalid limit (must be 1-100)", http.StatusBadRequest)
			return
		}
	}

	// Get database manager from registry
	manager, err := GetDatabase(dbID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Database not found: %v", err), http.StatusNotFound)
		return
	}

	// Use the manager's pool directly for querying (same pattern as schema.go)
	pool := manager.GetPool()

	// Check if pg_stat_statements is enabled
	var extVersion string
	err = pool.QueryRow(r.Context(),
		"SELECT extversion FROM pg_extension WHERE extname = 'pg_stat_statements'").Scan(&extVersion)
	if err != nil {
		// pg_stat_statements not available - return 503 with clear message
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "pg_stat_statements extension is not available",
			"message": "Please enable pg_stat_statements extension to use slow queries feature. Run: CREATE EXTENSION IF NOT EXISTS pg_stat_statements;",
		})
		return
	}

	// Query pg_stat_statements for slow queries
	slowQueries, err := getSlowQueriesFromConn(r.Context(), pool, threshold, limit)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to query slow queries: %v", err), http.StatusInternalServerError)
		return
	}

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"slow_queries": slowQueries,
		"threshold_ms": threshold,
		"count":        len(slowQueries),
	})
}

// GetQueryStatsHandler returns query statistics from pg_stat_statements
// GET /api/v1/databases/:id/metrics/queries?limit=10&sort=total_time
func GetQueryStatsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		ct := r.Header.Get("Content-Type")
		if ct != "" && !strings.Contains(ct, "application/json") {
			http.Error(w, "Content-Type must be application/json", http.StatusUnsupportedMediaType)
			return
		}
	}

	// Extract database ID from URL path
	path := strings.TrimSuffix(r.URL.Path, "/")
	parts := strings.Split(path, "/")
	// 路径格式: /api/v1/databases/{id}/metrics/queries
	// parts: ["", "api", "v1", "databases", "{id}", "metrics", "queries"]
	if len(parts) < 7 {
		http.Error(w, "Invalid URL format", http.StatusBadRequest)
		return
	}
	dbIDStr := parts[4]

	dbID, err := strconv.Atoi(dbIDStr)
	if err != nil || dbID <= 0 || dbID > 1000000 {
		http.Error(w, "Invalid database ID", http.StatusBadRequest)
		return
	}

	// Get limit from query params (default 10, range 1-100)
	limit := 10
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		limit, err = strconv.Atoi(limitStr)
		if err != nil || limit < 1 || limit > 100 {
			http.Error(w, "Invalid limit (must be 1-100)", http.StatusBadRequest)
			return
		}
	}

	// Get sort parameter (default: total_time)
	// Allowed values: total_time | calls | rows | mean_time
	sortBy := "total_time"
	if sortByStr := r.URL.Query().Get("sort"); sortByStr != "" {
		allowedSorts := map[string]bool{
			"total_time": true,
			"calls":      true,
			"rows":       true,
			"mean_time":  true,
		}
		if !allowedSorts[sortByStr] {
			http.Error(w, "Invalid sort parameter. Allowed values: total_time, calls, rows, mean_time", http.StatusBadRequest)
			return
		}
		sortBy = sortByStr
	}

	// Get database manager from registry
	manager, err := GetDatabase(dbID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Database not found: %v", err), http.StatusNotFound)
		return
	}

	// Use the manager's pool directly for querying (same pattern as schema.go)
	pool := manager.GetPool()

	// Check if pg_stat_statements is enabled
	var extVersion string
	err = pool.QueryRow(r.Context(),
		"SELECT extversion FROM pg_extension WHERE extname = 'pg_stat_statements'").Scan(&extVersion)
	if err != nil {
		// pg_stat_statements not available - return 503 with clear message
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "pg_stat_statements extension is not available",
			"message": "Please enable pg_stat_statements extension to use query stats feature. Run: CREATE EXTENSION IF NOT EXISTS pg_stat_statements;",
		})
		return
	}

	// Map sort parameter to actual column name
	sortColumn := mapSortParamToColumn(sortBy)

	// Query pg_stat_statements
	queryStats, err := getQueryStatsFromConn(r.Context(), pool, sortColumn, limit)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to query query stats: %v", err), http.StatusInternalServerError)
		return
	}

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"queries": queryStats,
		"sort":    sortBy,
		"limit":   limit,
		"count":   len(queryStats),
	})
}

// collectDatabaseStatsV2 collects comprehensive database statistics (version 2)
func collectDatabaseStatsV2(ctx context.Context, pool *database.ConnectionPool, dbID string) (map[string]interface{}, error) {
	stats := make(map[string]interface{})
	stats["database_id"] = dbID

	statsMap := make(map[string]interface{})

	// Get total connections
	var totalConnections int
	err := pool.QueryRow(ctx,
		"SELECT count(*) FROM pg_stat_activity WHERE datname = current_database()").Scan(&totalConnections)
	if err != nil {
		return nil, fmt.Errorf("failed to get total connections: %w", err)
	}
	statsMap["total_connections"] = totalConnections

	// Get active connections
	var activeConnections int
	err = pool.QueryRow(ctx,
		"SELECT count(*) FROM pg_stat_activity WHERE datname = current_database() AND state = 'active'").Scan(&activeConnections)
	if err != nil {
		return nil, fmt.Errorf("failed to get active connections: %w", err)
	}
	statsMap["active_connections"] = activeConnections

	// Get total tables
	var totalTables int
	err = pool.QueryRow(ctx,
		`SELECT count(*) FROM information_schema.tables
		 WHERE table_schema = 'public' AND table_type = 'BASE TABLE'`).Scan(&totalTables)
	if err != nil {
		return nil, fmt.Errorf("failed to get table count: %w", err)
	}
	statsMap["total_tables"] = totalTables

	// Get total indexes
	var totalIndexes int
	err = pool.QueryRow(ctx,
		"SELECT count(*) FROM pg_indexes WHERE schemaname = 'public'").Scan(&totalIndexes)
	if err != nil {
		return nil, fmt.Errorf("failed to get index count: %w", err)
	}
	statsMap["total_indexes"] = totalIndexes

	// Get database size in bytes
	var totalSizeBytes int64
	err = pool.QueryRow(ctx,
		"SELECT pg_database_size(current_database())").Scan(&totalSizeBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to get database size: %w", err)
	}
	statsMap["total_size_bytes"] = totalSizeBytes

	// Get uptime in seconds
	var uptimeSeconds int64
	err = pool.QueryRow(ctx,
		"SELECT extract(epoch from (now() - pg_postmaster_start_time()))::bigint").Scan(&uptimeSeconds)
	if err != nil {
		return nil, fmt.Errorf("failed to get uptime: %w", err)
	}
	statsMap["uptime_seconds"] = uptimeSeconds

	// Get cache hit ratio
	var cacheHitRatio float64
	err = pool.QueryRow(ctx,
		`SELECT round(sum(blks_hit)::numeric / NULLIF(sum(blks_hit)+sum(blks_read), 0), 4)
		 FROM pg_stat_database WHERE datname = current_database()`).Scan(&cacheHitRatio)
	if err != nil {
		return nil, fmt.Errorf("failed to get cache hit ratio: %w", err)
	}
	statsMap["cache_hit_ratio"] = cacheHitRatio

	stats["stats"] = statsMap

	return stats, nil
}

// getSlowQueriesFromConn retrieves slow queries exceeding the threshold
func getSlowQueriesFromConn(ctx context.Context, pool *database.ConnectionPool, thresholdMs int64, limit int) ([]map[string]interface{}, error) {
	query := `
		SELECT
			queryid as query_id,
			query as query_text,
			calls,
			total_exec_time as total_time_ms,
			mean_exec_time as mean_time_ms,
			max_exec_time as max_time_ms,
			min_exec_time as min_time_ms,
			rows
		FROM pg_stat_statements
		WHERE mean_exec_time > $1
		ORDER BY mean_exec_time DESC
		LIMIT $2
	`

	rows, err := pool.Query(ctx, query, float64(thresholdMs), limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query pg_stat_statements: %w", err)
	}
	defer rows.Close()

	var slowQueries []map[string]interface{}
	for rows.Next() {
		var queryID int64
		var queryText string
		var calls int64
		var totalTimeMs, meanTimeMs, maxTimeMs, minTimeMs float64
		var rowCount int64

		err := rows.Scan(&queryID, &queryText, &calls, &totalTimeMs, &meanTimeMs, &maxTimeMs, &minTimeMs, &rowCount)
		if err != nil {
			continue // Skip malformed rows
		}

		slowQueries = append(slowQueries, map[string]interface{}{
			"query_id":      queryID,
			"query_text":    queryText,
			"calls":         calls,
			"total_time_ms": totalTimeMs,
			"mean_time_ms":  meanTimeMs,
			"max_time_ms":   maxTimeMs,
			"min_time_ms":   minTimeMs,
			"rows":          rowCount,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating slow query results: %w", err)
	}

	return slowQueries, nil
}

// getQueryStatsFromConn retrieves query statistics sorted by specified column
func getQueryStatsFromConn(ctx context.Context, pool *database.ConnectionPool, sortColumn string, limit int) ([]map[string]interface{}, error) {
	query := fmt.Sprintf(`
		SELECT
			queryid::text as query_hash,
			left(query, 200) as query_text,
			calls,
			total_exec_time as total_time_ms,
			mean_exec_time as avg_time_ms,
			rows
		FROM pg_stat_statements
		ORDER BY %s DESC
		LIMIT $1
	`, sortColumn)

	rows, err := pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query pg_stat_statements: %w", err)
	}
	defer rows.Close()

	var queryStats []map[string]interface{}
	for rows.Next() {
		var queryHash string
		var queryText string
		var calls int64
		var totalTimeMs, avgTimeMs float64
		var rowCount int64

		err := rows.Scan(&queryHash, &queryText, &calls, &totalTimeMs, &avgTimeMs, &rowCount)
		if err != nil {
			continue // Skip malformed rows
		}

		queryStats = append(queryStats, map[string]interface{}{
			"query_hash":    queryHash,
			"query_text":    queryText,
			"calls":         calls,
			"total_time_ms": totalTimeMs,
			"avg_time_ms":   avgTimeMs,
			"rows":          rowCount,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating query stats results: %w", err)
	}

	return queryStats, nil
}

// mapSortParamToColumn maps API sort parameter to actual PostgreSQL column name
func mapSortParamToColumn(sortParam string) string {
	switch sortParam {
	case "total_time":
		return "total_exec_time"
	case "calls":
		return "calls"
	case "rows":
		return "rows"
	case "mean_time":
		return "mean_exec_time"
	default:
		return "total_exec_time"
	}
}

// Note: MetricsHandler struct and its methods are kept for backward compatibility
// but the preferred approach is to use the package-level handler functions above.

// MetricsHandler handles metrics collection endpoints (deprecated - use package-level functions)
type MetricsHandler struct{}

// NewMetricsHandler creates a new metrics handler (deprecated - use package-level functions)
func NewMetricsHandler() *MetricsHandler {
	return &MetricsHandler{}
}

// GetDatabaseStats returns database statistics (deprecated - use GetDatabaseStatsHandler)
func (h *MetricsHandler) GetDatabaseStats(w http.ResponseWriter, r *http.Request) {
	GetDatabaseStatsHandler(w, r)
}

// GetSlowQueries returns slow queries (deprecated - use GetSlowQueriesHandler)
func (h *MetricsHandler) GetSlowQueries(w http.ResponseWriter, r *http.Request) {
	GetSlowQueriesHandler(w, r)
}

// GetQueryStats returns query statistics (deprecated - use GetQueryStatsHandler)
func (h *MetricsHandler) GetQueryStats(w http.ResponseWriter, r *http.Request) {
	GetQueryStatsHandler(w, r)
}

// ResetStats resets pg_stat_statements statistics (deprecated - use ResetStatsHandler if needed)
func (h *MetricsHandler) ResetStats(w http.ResponseWriter, r *http.Request) {
	// This can be implemented later if needed
	http.Error(w, "Not implemented", http.StatusNotImplemented)
}

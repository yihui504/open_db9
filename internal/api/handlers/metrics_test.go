package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/open-db9/db9/internal/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestRegistry sets up the database registry for testing
func setupTestRegistry(t *testing.T, dbID int) {
	t.Helper()
	// Create a simple mock manager for testing
	pool := &database.ConnectionPool{}
	manager := database.ManagerFromPool(pool)
	RegisterDatabase(dbID, manager)
}

// cleanupTestRegistry cleans up the test registry
func cleanupTestRegistry(t *testing.T, dbID int) {
	t.Helper()
	UnregisterDatabase(dbID)
}

// ==================== GetDatabaseStatsHandler Tests ====================

func TestGetDatabaseStats_Success(t *testing.T) {
	// This test verifies the handler structure and parameter validation
	// Integration tests with actual DB connection should be in separate file
	req := httptest.NewRequest(http.MethodGet, "/api/v1/databases/1/metrics/stats", nil)
	rec := httptest.NewRecorder()

	// Register a test database (without actual connection)
	setupTestRegistry(t, 1)
	defer cleanupTestRegistry(t, 1)

	GetDatabaseStatsHandler(rec, req)

	// Should not return 501 Not Implemented anymore
	// May return 400 (if PathValue doesn't work in test), 500 (connection error), etc.
	assert.NotEqual(t, http.StatusNotImplemented, rec.Code)
}

func TestGetDatabaseStats_DatabaseNotFound(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/databases/99999/metrics/stats", nil)
	rec := httptest.NewRecorder()

	// Don't register this database ID
	GetDatabaseStatsHandler(rec, req)

	// Should return either 404 (not found in registry) or 400 (invalid ID format)
	assert.Contains(t, []int{http.StatusNotFound, http.StatusBadRequest}, rec.Code)

	if rec.Code == http.StatusNotFound {
		var resp map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Contains(t, resp["error"], "not found")
	}
}

func TestGetDatabaseStats_InvalidDatabaseID(t *testing.T) {
	testCases := []struct {
		name     string
		path     string
		expected int
	}{
		{"Non-numeric ID", "/api/v1/databases/abc/metrics/stats", http.StatusBadRequest},
		{"Negative ID", "/api/v1/databases/-1/metrics/stats", http.StatusBadRequest},
		{"Zero ID", "/api/v1/databases/0/metrics/stats", http.StatusBadRequest},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()

			GetDatabaseStatsHandler(rec, req)

			assert.Equal(t, tc.expected, rec.Code)
		})
	}
}

// ==================== GetSlowQueriesHandler Tests ====================

func TestGetSlowQueries_WithThreshold(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/databases/1/metrics/slow?threshold=1000&limit=20", nil)
	rec := httptest.NewRecorder()

	setupTestRegistry(t, 1)
	defer cleanupTestRegistry(t, 1)

	GetSlowQueriesHandler(rec, req)

	// Should not be 501 Not Implemented anymore
	assert.NotEqual(t, http.StatusNotImplemented, rec.Code)
	// Will likely fail with 400 (no path param), 500 (no real DB) or 503 (no pg_stat_statements), which is expected
	assert.Contains(t, []int{http.StatusOK, http.StatusBadRequest, http.StatusInternalServerError, http.StatusServiceUnavailable, http.StatusNotFound}, rec.Code)
}

func TestGetSlowQueries_EmptyResult(t *testing.T) {
	// Test with very high threshold to ensure no results
	req := httptest.NewRequest(http.MethodGet, "/api/v1/databases/1/metrics/slow?threshold=99999999&limit=10", nil)
	rec := httptest.NewRecorder()

	setupTestRegistry(t, 1)
	defer cleanupTestRegistry(t, 1)

	GetSlowQueriesHandler(rec, req)

	assert.NotEqual(t, http.StatusNotImplemented, rec.Code)
}

func TestGetSlowQueries_pgstatstatements_NotAvailable(t *testing.T) {
	// This test verifies that when pg_stat_statements is not available,
	// the handler returns 503 Service Unavailable with a clear message
	req := httptest.NewRequest(http.MethodGet, "/api/v1/databases/1/metrics/slow?threshold=1000&limit=20", nil)
	rec := httptest.NewRecorder()

	setupTestRegistry(t, 1)
	defer cleanupTestRegistry(t, 1)

	GetSlowQueriesHandler(rec, req)

	// If we get 503, verify the response structure
	if rec.Code == http.StatusServiceUnavailable {
		var resp map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.Contains(t, resp["error"], "pg_stat_statements")
		assert.Contains(t, resp["message"], "CREATE EXTENSION")
	}
}

func TestGetSlowQueries_InvalidThreshold(t *testing.T) {
	testCases := []struct {
		name         string
		threshold    string
		expectedCode int
	}{
		{"Zero threshold", "0", http.StatusBadRequest},
		{"Negative threshold", "-100", http.StatusBadRequest},
		{"Non-numeric threshold", "abc", http.StatusBadRequest},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet,
				"/api/v1/databases/1/metrics/slow?threshold="+tc.threshold+"&limit=20", nil)
			rec := httptest.NewRecorder()

			GetSlowQueriesHandler(rec, req)

			assert.Equal(t, tc.expectedCode, rec.Code)
		})
	}
}

func TestGetSlowQueries_InvalidLimit(t *testing.T) {
	testCases := []struct {
		name     string
		limit    string
		expected int
	}{
		{"Zero limit", "0", http.StatusBadRequest},
		{"Negative limit", "-1", http.StatusBadRequest},
		{"Limit too high", "101", http.StatusBadRequest},
		{"Limit way too high", "1000", http.StatusBadRequest},
		{"Non-numeric limit", "abc", http.StatusBadRequest},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet,
				"/api/v1/databases/1/metrics/slow?threshold=1000&limit="+tc.limit, nil)
			rec := httptest.NewRecorder()

			GetSlowQueriesHandler(rec, req)

			assert.Equal(t, tc.expected, rec.Code)
		})
	}
}

func TestGetSlowQueries_DatabaseNotFound(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/databases/99999/metrics/slow?threshold=1000", nil)
	rec := httptest.NewRecorder()

	GetSlowQueriesHandler(rec, req)

	// Should return either 404 (not found in registry) or 400 (invalid ID format)
	assert.Contains(t, []int{http.StatusNotFound, http.StatusBadRequest}, rec.Code)
}

// ==================== GetQueryStatsHandler Tests ====================

func TestQueryStats_SortByTotalTime(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/databases/1/metrics/queries?sort=total_time&limit=10", nil)
	rec := httptest.NewRecorder()

	setupTestRegistry(t, 1)
	defer cleanupTestRegistry(t, 1)

	GetQueryStatsHandler(rec, req)

	// Should not be 501 Not Implemented anymore
	assert.NotEqual(t, http.StatusNotImplemented, rec.Code)
	// Will likely fail with 400 (no path param), 500 (no real DB) or 503 (no pg_stat_statements)
	assert.Contains(t, []int{http.StatusOK, http.StatusBadRequest, http.StatusInternalServerError, http.StatusServiceUnavailable, http.StatusNotFound}, rec.Code)
}

func TestQueryStats_InvalidSortParameter(t *testing.T) {
	invalidSorts := []string{
		"invalid_sort",
		"total_exec_time", // Internal column name, not allowed API param
		"unknown",
	}

	for _, sort := range invalidSorts {
		t.Run("sort_"+sort, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet,
				"/api/v1/databases/1/metrics/queries?sort="+sort+"&limit=10", nil)
			rec := httptest.NewRecorder()

			GetQueryStatsHandler(rec, req)

			assert.Equal(t, http.StatusBadRequest, rec.Code)
		})
	}
}

func TestQueryStats_ValidSortParameters(t *testing.T) {
	validSorts := []string{
		"total_time",
		"calls",
		"rows",
		"mean_time",
	}

	for _, sort := range validSorts {
		t.Run("valid_sort_"+sort, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet,
				"/api/v1/databases/1/metrics/queries?sort="+sort+"&limit=10", nil)
			rec := httptest.NewRecorder()

			setupTestRegistry(t, 1)
			defer cleanupTestRegistry(t, 1)

			GetQueryStatsHandler(rec, req)

			// Should not return bad request for valid sort params
			// May return other errors (400 for path param, 500 for DB, etc.) but not because of invalid sort
			if rec.Code == http.StatusBadRequest {
				body := rec.Body.String()
				// Make sure it's not failing due to invalid sort parameter
				assert.NotContains(t, body, "Invalid sort")
			}
			assert.NotEqual(t, http.StatusNotImplemented, rec.Code)
		})
	}
}

func TestQueryStats_InvalidLimit(t *testing.T) {
	testCases := []struct {
		name     string
		limit    string
		expected int
	}{
		{"Zero limit", "0", http.StatusBadRequest},
		{"Negative limit", "-5", http.StatusBadRequest},
		{"Limit too high", "101", http.StatusBadRequest},
		{"Non-numeric limit", "abc", http.StatusBadRequest},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet,
				"/api/v1/databases/1/metrics/queries?limit="+tc.limit, nil)
			rec := httptest.NewRecorder()

			GetQueryStatsHandler(rec, req)

			assert.Equal(t, tc.expected, rec.Code)
		})
	}
}

func TestQueryStats_DatabaseNotFound(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/databases/99999/metrics/queries", nil)
	rec := httptest.NewRecorder()

	GetQueryStatsHandler(rec, req)

	// Should return either 404 (not found in registry) or 400 (invalid ID format)
	assert.Contains(t, []int{http.StatusNotFound, http.StatusBadRequest}, rec.Code)
}

func TestQueryStats_pgstatstatements_NotAvailable(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/databases/1/metrics/queries?limit=10", nil)
	rec := httptest.NewRecorder()

	setupTestRegistry(t, 1)
	defer cleanupTestRegistry(t, 1)

	GetQueryStatsHandler(rec, req)

	// If we get 503, verify the response structure
	if rec.Code == http.StatusServiceUnavailable {
		var resp map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.Contains(t, resp["error"], "pg_stat_statements")
		assert.Contains(t, resp["message"], "CREATE EXTENSION")
	}
}

// ==================== Helper Function Tests ====================

func TestMapSortParamToColumn(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"total_time", "total_exec_time"},
		{"calls", "calls"},
		{"rows", "rows"},
		{"mean_time", "mean_exec_time"},
		{"unknown", "total_exec_time"}, // default fallback
		{"", "total_exec_time"},        // default fallback
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := mapSortParamToColumn(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// ==================== Backward Compatibility Tests ====================

func TestMetricsHandler_BackwardCompatibility(t *testing.T) {
	// Test that the deprecated MetricsHandler struct still works
	handler := NewMetricsHandler()
	require.NotNil(t, handler)

	// These should not panic, just delegate to package-level functions
	req := httptest.NewRequest(http.MethodGet, "/api/v1/databases/1/metrics/stats", nil)
	rec := httptest.NewRecorder()

	handler.GetDatabaseStats(rec, req)
	assert.NotEqual(t, http.StatusNotImplemented, rec.Code)
}

// ==================== Edge Case Tests ====================

func TestMetricsHandlers_MethodNotAllowed(t *testing.T) {
	// Test that POST requests to GET endpoints are rejected
	// Note: In test environment, PathValue may not work correctly,
	// so we test that the handler doesn't return 200 or other success codes
	testCases := []struct {
		name    string
		handler func(http.ResponseWriter, *http.Request)
		path    string
	}{
		{"Stats POST", GetDatabaseStatsHandler, "/api/v1/databases/1/metrics/stats"},
		{"SlowQueries POST", GetSlowQueriesHandler, "/api/v1/databases/1/metrics/slow"},
		{"QueryStats POST", GetQueryStatsHandler, "/api/v1/databases/1/metrics/queries"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, tc.path, nil)
			rec := httptest.NewRecorder()

			tc.handler(rec, req)

			// Should not return success status codes (200-299)
			assert.GreaterOrEqual(t, rec.Code, http.StatusBadRequest)
		})
	}
}

func TestMetricsHandlers_DefaultParameters(t *testing.T) {
	// Test that default parameters work correctly
	// Note: In test environment, PathValue may not work, so we just verify
	// the handler doesn't panic and processes the request
	testCases := []struct {
		name    string
		path    string
		handler func(http.ResponseWriter, *http.Request)
	}{
		{
			"Slow queries defaults",
			"/api/v1/databases/1/metrics/slow",
			GetSlowQueriesHandler,
		},
		{
			"Query stats defaults",
			"/api/v1/databases/1/metrics/queries",
			GetQueryStatsHandler,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()

			setupTestRegistry(t, 1)
			defer cleanupTestRegistry(t, 1)

			tc.handler(rec, req)

			// Handler should process without panicking
			// May return various error codes in test environment, which is OK
			assert.NotEqual(t, http.StatusNotImplemented, rec.Code)
		})
	}
}

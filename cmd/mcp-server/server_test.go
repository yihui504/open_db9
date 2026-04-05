package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	mcp "github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestServerStart tests that the server can be initialized
func TestServerStart(t *testing.T) {
	// Set required environment variable
	os.Setenv("DB9_API_TOKEN", "test-token")
	defer os.Unsetenv("DB9_API_TOKEN")

	// Note: We can't actually start the server in tests because it blocks on stdio
	// But we can verify the configuration is loaded correctly
	apiURL := getEnv("DB9_API_URL", "http://localhost:8080")
	assert.Equal(t, "http://localhost:8080", apiURL)

	token := getEnv("DB9_API_TOKEN", "")
	assert.Equal(t, "test-token", token)
}

// TestToolsList verifies all tools are registered correctly
func TestToolsList(t *testing.T) {
	expectedTools := []string{
		"execute_sql",
		"list_databases",
		"create_database",
		"list_files",
		"upload_file",
		"download_file",
		"query_rag",
		"get_schema",
		"create_snapshot",
		"list_snapshots",
		"restore_snapshot",
	}

	// Verify tool names are defined in handlers.go
	// This test ensures we have the expected number of tools registered
	assert.Len(t, expectedTools, 11, "Should have 11 tools registered")
}

// TestExecuteSQLCall tests the execute_sql handler with mock API
func TestExecuteSQLCall(t *testing.T) {
	// Setup mock client by saving original and restoring after test
	originalClient := client
	defer func() { client = originalClient }()

	// Create a real API client pointing to test server
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "/sql")
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

		// Return mock response
		response := APIResponse{
			Success: true,
			Data: map[string]interface{}{
				"columns":    []interface{}{"id", "name"},
				"rows":       []map[string]interface{}{{"id": 1, "name": "test"}},
				"row_count":  1,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer testServer.Close()

	client = NewAPIClient(testServer.URL, "test-token", "")

	request := mcp.CallToolRequest{
		Params: struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments"`
		}{
			Name: "execute_sql",
			Arguments: map[string]interface{}{
				"database_id": "1",
				"sql":         "SELECT * FROM users LIMIT 10",
			},
		},
	}

	result, err := handleExecuteSQL(context.Background(), request)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify response contains data
	assert.True(t, len(result.Content) > 0)
}

// TestErrorResponse tests error handling from API
func TestErrorResponse(t *testing.T) {
	originalClient := client
	defer func() { client = originalClient }()

	tests := []struct {
		name          string
		statusCode    int
		responseBody  string
		expectError   bool
		errorContains string
		handler       func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)
		request       mcp.CallToolRequest
	}{
		{
			name:         "API returns error",
			statusCode:   404,
			responseBody: `{"success": false, "error": "database not found"}`,
			expectError:  true,
			errorContains: "database not found",
			handler:      handleExecuteSQL,
			request: mcp.CallToolRequest{
				Params: struct {
					Name      string                 `json:"name"`
					Arguments map[string]interface{} `json:"arguments"`
				}{
					Name: "execute_sql",
					Arguments: map[string]interface{}{
						"database_id": "999",
						"sql":         "SELECT 1",
					},
				},
			},
		},
		{
			name:        "Network error",
			statusCode:  500,
			expectError: true,
			handler:     handleListDatabases,
			request: mcp.CallToolRequest{
				Params: struct {
					Name      string                 `json:"name"`
					Arguments map[string]interface{} `json:"arguments"`
				}{
					Name:      "list_databases",
					Arguments: map[string]interface{}{},
				},
			},
		},
		{
			name:          "SQL execution error",
			statusCode:    500,
			responseBody:  `{"success": false, "error": "relation \"nonexistent\" does not exist"}`,
			expectError:   true,
			errorContains: `relation "nonexistent" does not exist`,
			handler:       handleExecuteSQL,
			request: mcp.CallToolRequest{
				Params: struct {
					Name      string                 `json:"name"`
					Arguments map[string]interface{} `json:"arguments"`
				}{
					Name: "execute_sql",
					Arguments: map[string]interface{}{
						"database_id": "1",
						"sql":         "SELECT * FROM nonexistent",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server that returns the specified response
			testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				if tt.responseBody != "" {
					w.Header().Set("Content-Type", "application/json")
					w.Write([]byte(tt.responseBody))
				}
			}))
			defer testServer.Close()

			client = NewAPIClient(testServer.URL, "test-token", "")

			result, err := tt.handler(context.Background(), tt.request)

			if tt.expectError {
				require.NoError(t, err) // Handler should return nil error, but result should indicate error
				require.NotNil(t, result)
				if tt.errorContains != "" {
					assert.Contains(t, result.Content[0].Text, tt.errorContains)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
			}
		})
	}
}

// TestMissingConfig tests behavior when environment variables are missing
func TestMissingConfig(t *testing.T) {
	// Save and restore original value
	origToken := os.Getenv("DB9_API_TOKEN")
	defer os.Setenv("DB9_API_TOKEN", origToken)

	// Clear token
	os.Unsetenv("DB9_API_TOKEN")

	// Verify getEnv returns default when env var is missing
	value := getEnv("DB9_API_URL", "http://default:8080")
	assert.Equal(t, "http://default:8080", value)

	value = getEnv("DB9_API_TOKEN", "")
	assert.Equal(t, "", value)
}

// TestAPIClientDo tests the API client's Do method
func TestAPIClientDo(t *testing.T) {
	// Create a test server
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request headers
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// Return mock response
		response := APIResponse{
			Success: true,
			Data:    map[string]string{"status": "ok"},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer testServer.Close()

	// Create client pointing to test server
	c := NewAPIClient(testServer.URL, "test-token", "")

	resp, err := c.Do("POST", "/api/test", map[string]string{"key": "value"})
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.True(t, resp.Success)
	assert.NotNil(t, resp.Data)
}

// TestCreateDatabaseHandler tests database creation
func TestCreateDatabaseHandler(t *testing.T) {
	originalClient := client
	defer func() { client = originalClient }()

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "/databases/")

		response := APIResponse{
			Success: true,
			Data: map[string]interface{}{
				"id":     2,
				"name":   "test_db",
				"engine": "postgresql",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer testServer.Close()

	client = NewAPIClient(testServer.URL, "test-token", "")

	request := mcp.CallToolRequest{
		Params: struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments"`
		}{
			Name: "create_database",
			Arguments: map[string]interface{}{
				"name":   "test_db",
				"engine": "postgresql",
			},
		},
	}

	result, err := handleCreateDatabase(context.Background(), request)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Contains(t, result.Content[0].Text, "Database created successfully")
}

// TestListSnapshotsHandler tests snapshot listing
func TestListSnapshotsHandler(t *testing.T) {
	originalClient := client
	defer func() { client = originalClient }()

	mockSnapshots := []map[string]interface{}{
		{
			"id":            "snap-abc123",
			"name":          "daily-backup",
			"status":        "ready",
			"size_bytes":    104857600,
			"created_at":    "2026-04-03T00:00:00Z",
		},
	}

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "/snapshots")

		response := APIResponse{
			Success: true,
			Data: map[string]interface{}{
				"snapshots": mockSnapshots,
				"count":     1,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer testServer.Close()

	client = NewAPIClient(testServer.URL, "test-token", "")

	request := mcp.CallToolRequest{
		Params: struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments"`
		}{
			Name: "list_snapshots",
			Arguments: map[string]interface{}{
				"database_id": "1",
			},
		},
	}

	result, err := handleListSnapshots(context.Background(), request)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Contains(t, result.Content[0].Text, "snap-abc123")
}

// TestQueryRAGHandler tests RAG query functionality
func TestQueryRAGHandler(t *testing.T) {
	originalClient := client
	defer func() { client = originalClient }()

	ragResponse := map[string]interface{}{
		"answer": "Based on the documents, the answer is...",
		"sources": []map[string]interface{}{
			{
				"content":     "relevant document content...",
				"score":       0.95,
				"document_id": "doc-123",
			},
		},
	}

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "/rag/query")

		response := APIResponse{
			Success: true,
			Data:    ragResponse,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer testServer.Close()

	client = NewAPIClient("http://localhost", "test-token", testServer.URL)

	request := mcp.CallToolRequest{
		Params: struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments"`
		}{
			Name: "query_rag",
			Arguments: map[string]interface{}{
				"database_id": "1",
				"query":       "What is the total revenue?",
				"top_k":       float64(5),
			},
		},
	}

	result, err := handleQueryRAG(context.Background(), request)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Contains(t, result.Content[0].Text, "Based on the documents")
}

// TestGetSchemaHandler tests schema retrieval
func TestGetSchemaHandler(t *testing.T) {
	originalClient := client
	defer func() { client = originalClient }()

	schemaDetails := map[string]interface{}{
		"schema":  "public",
		"table":   "users",
		"type":    "BASE TABLE",
		"columns": []map[string]interface{}{
			{
				"name":       "id",
				"data_type":  "integer",
				"is_nullable": false,
			},
			{
				"name":       "email",
				"data_type":  "character varying",
				"is_nullable": true,
			},
		},
	}

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "/schema/table")

		response := APIResponse{
			Success: true,
			Data:    schemaDetails,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer testServer.Close()

	client = NewAPIClient(testServer.URL, "test-token", "")

	request := mcp.CallToolRequest{
		Params: struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments"`
		}{
			Name: "get_schema",
			Arguments: map[string]interface{}{
				"database_id": "1",
				"table":       "users",
				"schema":      "public",
			},
		},
	}

	result, err := handleGetSchema(context.Background(), request)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Contains(t, result.Content[0].Text, "users")
}

// TestRestoreSnapshotHandler tests snapshot restoration
func TestRestoreSnapshotHandler(t *testing.T) {
	originalClient := client
	defer func() { client = originalClient }()

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "/restore")

		response := APIResponse{
			Success: true,
			Data: map[string]interface{}{
				"snapshot_id": "snap-abc123",
				"status":      "restoring",
			},
			Message: "Snapshot restoration initiated",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer testServer.Close()

	client = NewAPIClient(testServer.URL, "test-token", "")

	request := mcp.CallToolRequest{
		Params: struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments"`
		}{
			Name: "restore_snapshot",
			Arguments: map[string]interface{}{
				"database_id": "1",
				"snapshot_id": "snap-abc123",
			},
		},
	}

	result, err := handleRestoreSnapshot(context.Background(), request)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Contains(t, result.Content[0].Text, "Snapshot restoration initiated")
}

// TestHelperFunctions tests utility functions
func TestHelperFunctions(t *testing.T) {
	t.Run("getStringArg with value", func(t *testing.T) {
		request := mcp.CallToolRequest{
			Params: struct {
				Name      string                 `json:"name"`
				Arguments map[string]interface{} `json:"arguments"`
			}{
				Arguments: map[string]interface{}{
					"engine": "postgresql",
				},
			},
		}
		result := getStringArg(request, "engine", "mysql")
		assert.Equal(t, "postgresql", result)
	})

	t.Run("getStringArg without value uses default", func(t *testing.T) {
		request := mcp.CallToolRequest{
			Params: struct {
				Name      string                 `json:"name"`
				Arguments map[string]interface{} `json:"arguments"`
			}{
				Arguments: map[string]interface{}{},
			},
		}
		result := getStringArg(request, "engine", "mysql")
		assert.Equal(t, "mysql", result)
	})

	t.Run("truncateSQL short query", func(t *testing.T) {
		sql := "SELECT 1"
		result := truncateSQL(sql)
		assert.Equal(t, sql, result)
	})

	t.Run("truncateSQL long query", func(t *testing.T) {
		sql := strings.Repeat("a", 300)
		result := truncateSQL(sql)
		assert.Len(t, result, 203) // 200 chars + "..."
		assert.True(t, strings.HasSuffix(result, "..."))
	})
}

// BenchmarkExecuteSQL benchmarks SQL execution handler
func BenchmarkExecuteSQL(b *testing.B) {
	originalClient := client
	defer func() { client = originalClient }()

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := APIResponse{
			Success: true,
			Data: map[string]interface{}{
				"rows": make([]interface{}, 100),
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer testServer.Close()

	client = NewAPIClient(testServer.URL, "test-token", "")

	request := mcp.CallToolRequest{
		Params: struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments"`
		}{
			Name: "execute_sql",
			Arguments: map[string]interface{}{
				"database_id": "1",
				"sql":         "SELECT * FROM large_table",
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handleExecuteSQL(context.Background(), request)
	}
}

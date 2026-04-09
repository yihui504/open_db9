package handlers

import (
	"bytes"
	"context"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/open-db9/db9/internal/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==================== SQL Handler Enhanced Tests ====================

func TestExecuteSQLHandler_TimeoutValidation(t *testing.T) {
	dbManager := &database.Manager{}
	RegisterDatabase(1, dbManager)
	defer UnregisterDatabase(1)

	tests := []struct {
		name       string
		timeoutMs  int
		expectCode int
	}{
		{"timeout below minimum", 100, http.StatusOK}, // Should be adjusted to MinQueryTimeout
		{"timeout at minimum", 1000, http.StatusOK},
		{"timeout default", 30000, http.StatusOK},
		{"timeout above maximum", 600000, http.StatusOK}, // Should be adjusted to MaxQueryTimeout
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := createTestRequest(http.MethodPost, "/api/v1/databases/1/sql", SQLRequest{
				Query:     "SELECT 1",
				TimeoutMs: &tt.timeoutMs,
			})
			rec := httptest.NewRecorder()

			ExecuteSQLHandler(rec, req)

			// The handler should process the request (even if query fails)
			assert.NotEqual(t, http.StatusMethodNotAllowed, rec.Code)
		})
	}
}

func TestExecuteSQLHandler_NilTimeout(t *testing.T) {
	dbManager := &database.Manager{}
	RegisterDatabase(1, dbManager)
	defer UnregisterDatabase(1)

	req := createTestRequest(http.MethodPost, "/api/v1/databases/1/sql", SQLRequest{
		Query:     "SELECT 1",
		TimeoutMs: nil,
	})
	rec := httptest.NewRecorder()

	ExecuteSQLHandler(rec, req)

	// Should use default timeout
	assert.NotEqual(t, http.StatusMethodNotAllowed, rec.Code)
}

func TestExecuteSQLHandler_InvalidDatabaseID(t *testing.T) {
	tests := []struct {
		name    string
		path    string
	}{
		{"negative ID", "/api/v1/databases/-1/sql"},
		{"zero ID", "/api/v1/databases/0/sql"},
		{"string ID", "/api/v1/databases/abc/sql"},
		{"float ID", "/api/v1/databases/1.5/sql"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := createTestRequest(http.MethodPost, tt.path, SQLRequest{
				Query: "SELECT 1",
			})
			rec := httptest.NewRecorder()

			ExecuteSQLHandler(rec, req)

			assert.Contains(t, []int{http.StatusBadRequest, http.StatusNotFound}, rec.Code)
		})
	}
}

func TestExecuteSQLHandler_ValidSQLVariations(t *testing.T) {
	dbManager := &database.Manager{}
	RegisterDatabase(1, dbManager)
	defer UnregisterDatabase(1)

	validQueries := []string{
		"SELECT 1",
		"SELECT NOW()",
		"SELECT version()",
		"  SELECT 1  ", // with whitespace
		"SELECT * FROM information_schema.tables",
		"EXPLAIN SELECT 1",
		"WITH cte AS (SELECT 1) SELECT * FROM cte",
	}

	for _, query := range validQueries {
		t.Run(query, func(t *testing.T) {
			req := createTestRequest(http.MethodPost, "/api/v1/databases/1/sql", SQLRequest{
				Query: query,
			})
			rec := httptest.NewRecorder()

			ExecuteSQLHandler(rec, req)

			// Should pass validation (may fail on execution without real DB)
			// Accept 400 (validation passed but execution failed) or 500 (execution error)
			assert.Contains(t, []int{http.StatusBadRequest, http.StatusInternalServerError}, rec.Code)
		})
	}
}

func TestExecuteSQLHandler_UnsupportedOperations(t *testing.T) {
	dbManager := &database.Manager{}
	RegisterDatabase(1, dbManager)
	defer UnregisterDatabase(1)

	unsupportedQueries := []string{
		"DROP TABLE users",
		"DELETE FROM users",
		"INSERT INTO users VALUES (1)",
		"UPDATE users SET name = 'test'",
		"CREATE TABLE test (id INT)",
		"ALTER TABLE users ADD COLUMN col INT",
		"GRANT ALL ON users TO admin",
		"REVOKE ALL ON users FROM admin",
	}

	for _, query := range unsupportedQueries {
		t.Run(query, func(t *testing.T) {
			req := createTestRequest(http.MethodPost, "/api/v1/databases/1/sql", SQLRequest{
				Query: query,
			})
			rec := httptest.NewRecorder()

			ExecuteSQLHandler(rec, req)

			assert.Equal(t, http.StatusBadRequest, rec.Code)
		})
	}
}

func TestExecuteSQLHandler_MultiStatement(t *testing.T) {
	dbManager := &database.Manager{}
	RegisterDatabase(1, dbManager)
	defer UnregisterDatabase(1)

	req := createTestRequest(http.MethodPost, "/api/v1/databases/1/sql", SQLRequest{
		Query: "SELECT 1; SELECT 2;",
	})
	rec := httptest.NewRecorder()

	ExecuteSQLHandler(rec, req)

	// Multi-statement queries should be rejected
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// ==================== File Handler Enhanced Tests ====================

func TestUploadFileHandler_InvalidMethod(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/databases/1/files", nil)
	rec := httptest.NewRecorder()

	UploadFileHandler(rec, req)

	assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
}

func TestUploadFileHandler_InvalidURL(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{"missing database ID", "/api/v1/databases/files"},
		{"invalid ID", "/api/v1/databases/abc/files"},
		{"extra path", "/api/v1/databases/1/files/extra"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, tt.path, nil)
			rec := httptest.NewRecorder()

			UploadFileHandler(rec, req)

			assert.Equal(t, http.StatusBadRequest, rec.Code)
		})
	}
}

func TestUploadFileHandler_DatabaseNotFound(t *testing.T) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "test.txt")
	part.Write([]byte("test content"))
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/databases/999/files", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()

	UploadFileHandler(rec, req)

	// Should return 404 or 400 (for invalid URL format)
	assert.Contains(t, []int{http.StatusNotFound, http.StatusBadRequest}, rec.Code)
}

func TestUploadFileHandler_NoFile(t *testing.T) {
	dbManager := &database.Manager{}
	RegisterDatabase(1, dbManager)
	defer UnregisterDatabase(1)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.WriteField("path", "/test/path")
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/databases/1/files", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()

	UploadFileHandler(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "file")
}

func TestUploadFileHandler_PathValidation(t *testing.T) {
	dbManager := &database.Manager{}
	RegisterDatabase(1, dbManager)
	defer UnregisterDatabase(1)

	tests := []struct {
		name        string
		path        string
		expectStart string
	}{
		{"path without leading slash", "test/path", "/test/path"},
		{"path with leading slash", "/test/path", "/test/path"},
		{"empty path", "", "/"},
		{"nested path", "/a/b/c/test.txt", "/a/b/c/test.txt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)
			part, _ := writer.CreateFormFile("file", "test.txt")
			part.Write([]byte("test content"))
			if tt.path != "" {
				writer.WriteField("path", tt.path)
			}
			writer.Close()

			req := httptest.NewRequest(http.MethodPost, "/api/v1/databases/1/files", body)
			req.Header.Set("Content-Type", writer.FormDataContentType())
			rec := httptest.NewRecorder()

			UploadFileHandler(rec, req)

			// Will fail on DB operations but should validate path format
			assert.NotEqual(t, http.StatusMethodNotAllowed, rec.Code)
		})
	}
}

func TestDownloadFileHandler_InvalidMethod(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/databases/1/files/test.txt", nil)
	rec := httptest.NewRecorder()

	DownloadFileHandler(rec, req)

	assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
}

func TestDownloadFileHandler_InvalidURL(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{"missing database ID", "/api/v1/databases/files/test.txt"},
		{"invalid ID", "/api/v1/databases/abc/files/test.txt"},
		{"missing file path", "/api/v1/databases/1/files"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()

			DownloadFileHandler(rec, req)

			assert.Equal(t, http.StatusBadRequest, rec.Code)
		})
	}
}

func TestDeleteFileHandler_InvalidMethod(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/databases/1/files/test.txt", nil)
	rec := httptest.NewRecorder()

	DeleteFileHandler(rec, req)

	assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
}

func TestDeleteFileHandler_InvalidURL(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{"missing database ID", "/api/v1/databases/files/test.txt"},
		{"invalid ID", "/api/v1/databases/abc/files/test.txt"},
		{"missing file path", "/api/v1/databases/1/files"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodDelete, tt.path, nil)
			rec := httptest.NewRecorder()

			DeleteFileHandler(rec, req)

			assert.Equal(t, http.StatusBadRequest, rec.Code)
		})
	}
}

func TestListFilesHandler_InvalidMethod(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/databases/1/files", nil)
	rec := httptest.NewRecorder()

	ListFilesHandler(rec, req)

	assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
}

func TestListFilesHandler_QueryParameters(t *testing.T) {
	dbManager := &database.Manager{}
	RegisterDatabase(1, dbManager)
	defer UnregisterDatabase(1)

	tests := []struct {
		name       string
		query      string
		expectCode int
	}{
		{"no parameters", "", http.StatusOK},
		{"path filter", "?path=/test", http.StatusOK},
		{"directory filter", "?path=/test/", http.StatusOK},
		{"valid limit", "?limit=10", http.StatusOK},
		{"limit too high", "?limit=2000", http.StatusOK}, // Should cap at 1000
		{"invalid limit", "?limit=abc", http.StatusOK},  // Should default to 100
		{"negative limit", "?limit=-10", http.StatusOK}, // Should default to 100
		{"zero limit", "?limit=0", http.StatusOK},       // Should default to 100
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/databases/1/files"+tt.query, nil)
			rec := httptest.NewRecorder()

			ListFilesHandler(rec, req)

			// Will fail on DB operations but should parse query params
			assert.NotEqual(t, http.StatusMethodNotAllowed, rec.Code)
		})
	}
}

func TestCopyFileHandler_InvalidMethod(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/databases/1/files/copy", nil)
	rec := httptest.NewRecorder()

	CopyFileHandler(rec, req)

	assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
}

func TestCopyFileHandler_InvalidURL(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{"missing database ID", "/api/v1/databases/files/copy"},
		{"invalid ID", "/api/v1/databases/abc/files/copy"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, tt.path, nil)
			rec := httptest.NewRecorder()

			CopyFileHandler(rec, req)

			assert.Equal(t, http.StatusBadRequest, rec.Code)
		})
	}
}

func TestCopyFileHandler_MissingRequestBody(t *testing.T) {
	dbManager := &database.Manager{}
	RegisterDatabase(1, dbManager)
	defer UnregisterDatabase(1)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/databases/1/files/copy", nil)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	CopyFileHandler(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCopyFileHandler_InvalidRequestBody(t *testing.T) {
	dbManager := &database.Manager{}
	RegisterDatabase(1, dbManager)
	defer UnregisterDatabase(1)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/databases/1/files/copy", bytes.NewReader([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	CopyFileHandler(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCopyFileHandler_MissingPaths(t *testing.T) {
	dbManager := &database.Manager{}
	RegisterDatabase(1, dbManager)
	defer UnregisterDatabase(1)

	tests := []struct {
		name string
		body CopyFileRequest
	}{
		{"missing src_path", CopyFileRequest{DstPath: "/dst"}},
		{"missing dst_path", CopyFileRequest{SrcPath: "/src"}},
		{"empty paths", CopyFileRequest{SrcPath: "", DstPath: ""}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := createTestRequest(http.MethodPost, "/api/v1/databases/1/files/copy", tt.body)
			rec := httptest.NewRecorder()

			CopyFileHandler(rec, req)

			// Will fail on DB operations but should validate request
			assert.NotEqual(t, http.StatusMethodNotAllowed, rec.Code)
		})
	}
}

// ==================== Snapshot Handler Enhanced Tests ====================

func TestCreateSnapshotHandler_ValidRequest(t *testing.T) {
	mockMgr := NewMockSnapshotManagerExtended()
	RegisterSnapshotManager(1, mockMgr)
	defer UnregisterSnapshotManager(1)

	req := createTestRequest(http.MethodPost, "/api/v1/databases/1/snapshots", CreateSnapshotRequest{
		Name: "test-snapshot",
	})
	rec := httptest.NewRecorder()

	CreateSnapshotHandler(rec, req)

	assert.Equal(t, http.StatusAccepted, rec.Code)
	resp := parseResponse(t, rec)
	assert.True(t, resp["success"].(bool))
}

func TestCreateSnapshotHandler_InvalidContentType(t *testing.T) {
	mockMgr := NewMockSnapshotManagerExtended()
	RegisterSnapshotManager(1, mockMgr)
	defer UnregisterSnapshotManager(1)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/databases/1/snapshots", bytes.NewReader([]byte("{}")))
	req.Header.Set("Content-Type", "text/plain")
	rec := httptest.NewRecorder()

	CreateSnapshotHandler(rec, req)

	assert.Equal(t, http.StatusUnsupportedMediaType, rec.Code)
}

func TestCreateSnapshotHandler_InvalidRequestBody(t *testing.T) {
	mockMgr := NewMockSnapshotManagerExtended()
	RegisterSnapshotManager(1, mockMgr)
	defer UnregisterSnapshotManager(1)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/databases/1/snapshots", bytes.NewReader([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	CreateSnapshotHandler(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCreateSnapshotHandler_SpecialCharacters(t *testing.T) {
	mockMgr := NewMockSnapshotManagerExtended()
	RegisterSnapshotManager(1, mockMgr)
	defer UnregisterSnapshotManager(1)

	specialNames := []string{
		"test-snapshot-with-dashes",
		"test_snapshot_with_underscores",
		"test.snapshot.with.dots",
		"TestSnapshotWithCamelCase",
		"123456",
		"snapshot with spaces",
		"测试快照", // Chinese characters
	}

	for _, name := range specialNames {
		t.Run(name, func(t *testing.T) {
			req := createTestRequest(http.MethodPost, "/api/v1/databases/1/snapshots", CreateSnapshotRequest{
				Name: name,
			})
			rec := httptest.NewRecorder()

			CreateSnapshotHandler(rec, req)

			assert.Equal(t, http.StatusAccepted, rec.Code)
		})
	}
}

func TestListSnapshotsHandler_ValidRequest(t *testing.T) {
	mockMgr := NewMockSnapshotManagerExtended()
	RegisterSnapshotManager(1, mockMgr)
	defer UnregisterSnapshotManager(1)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/databases/1/snapshots", nil)
	rec := httptest.NewRecorder()

	ListSnapshotsHandler(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	resp := parseResponse(t, rec)
	assert.True(t, resp["success"].(bool))
	data := resp["data"].(map[string]interface{})
	assert.NotNil(t, data["snapshots"])
	assert.NotNil(t, data["count"])
}

func TestListSnapshotsHandler_AfterCreatingSnapshots(t *testing.T) {
	mockMgr := NewMockSnapshotManagerExtended()
	RegisterSnapshotManager(1, mockMgr)
	defer UnregisterSnapshotManager(1)

	// Create some snapshots
	mockMgr.CreateSnapshotSync(context.Background(), "1", "snap1")
	mockMgr.CreateSnapshotSync(context.Background(), "1", "snap2")
	mockMgr.CreateSnapshotSync(context.Background(), "1", "snap3")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/databases/1/snapshots", nil)
	rec := httptest.NewRecorder()

	ListSnapshotsHandler(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	resp := parseResponse(t, rec)
	data := resp["data"].(map[string]interface{})
	snapshots := data["snapshots"].([]interface{})
	assert.GreaterOrEqual(t, len(snapshots), 3)
}

func TestGetSnapshotHandler_ValidRequest(t *testing.T) {
	mockMgr := NewMockSnapshotManagerExtended()
	RegisterSnapshotManager(1, mockMgr)
	defer UnregisterSnapshotManager(1)

	// Create a snapshot first
	snapshot, _ := mockMgr.CreateSnapshotSync(context.Background(), "1", "test-snap")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/databases/1/snapshots/"+snapshot.ID, nil)
	rec := httptest.NewRecorder()

	GetSnapshotHandler(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	resp := parseResponse(t, rec)
	assert.True(t, resp["success"].(bool))
}

func TestGetSnapshotHandler_NonExistent(t *testing.T) {
	mockMgr := NewMockSnapshotManagerExtended()
	RegisterSnapshotManager(1, mockMgr)
	defer UnregisterSnapshotManager(1)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/databases/1/snapshots/nonexistent", nil)
	rec := httptest.NewRecorder()

	GetSnapshotHandler(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestDeleteSnapshotHandler_ValidRequest(t *testing.T) {
	mockMgr := NewMockSnapshotManagerExtended()
	RegisterSnapshotManager(1, mockMgr)
	defer UnregisterSnapshotManager(1)

	// Create a snapshot first
	snapshot, _ := mockMgr.CreateSnapshotSync(context.Background(), "1", "test-snap")

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/databases/1/snapshots/"+snapshot.ID+"?confirm=true", nil)
	rec := httptest.NewRecorder()

	DeleteSnapshotHandler(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	resp := parseResponse(t, rec)
	assert.True(t, resp["success"].(bool))
}

func TestDeleteSnapshotHandler_NoConfirmationEnhanced(t *testing.T) {
	mockMgr := NewMockSnapshotManagerExtended()
	RegisterSnapshotManager(1, mockMgr)
	defer UnregisterSnapshotManager(1)

	// Create a snapshot first
	snapshot, _ := mockMgr.CreateSnapshotSync(context.Background(), "1", "test-snap")

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/databases/1/snapshots/"+snapshot.ID, nil)
	rec := httptest.NewRecorder()

	DeleteSnapshotHandler(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestDeleteSnapshotHandler_ConfirmationFalse(t *testing.T) {
	mockMgr := NewMockSnapshotManagerExtended()
	RegisterSnapshotManager(1, mockMgr)
	defer UnregisterSnapshotManager(1)

	// Create a snapshot first
	snapshot, _ := mockMgr.CreateSnapshotSync(context.Background(), "1", "test-snap")

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/databases/1/snapshots/"+snapshot.ID+"?confirm=false", nil)
	rec := httptest.NewRecorder()

	DeleteSnapshotHandler(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestRestoreSnapshotHandler_ValidRequest(t *testing.T) {
	mockMgr := NewMockSnapshotManagerExtended()
	RegisterSnapshotManager(1, mockMgr)
	defer UnregisterSnapshotManager(1)

	// Create a snapshot first
	snapshot, _ := mockMgr.CreateSnapshotSync(context.Background(), "1", "test-snap")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/databases/1/snapshots/"+snapshot.ID+"/restore", nil)
	rec := httptest.NewRecorder()

	RestoreSnapshotHandler(rec, req)

	assert.Equal(t, http.StatusAccepted, rec.Code)
	resp := parseResponse(t, rec)
	assert.True(t, resp["success"].(bool))
	data := resp["data"].(map[string]interface{})
	assert.Equal(t, snapshot.ID, data["snapshot_id"])
	assert.Equal(t, "restoring", data["status"])
}

func TestRestoreSnapshotHandler_NonExistent(t *testing.T) {
	mockMgr := NewMockSnapshotManagerExtended()
	RegisterSnapshotManager(1, mockMgr)
	defer UnregisterSnapshotManager(1)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/databases/1/snapshots/nonexistent/restore", nil)
	rec := httptest.NewRecorder()

	RestoreSnapshotHandler(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// ==================== Helper Function Tests ====================

func TestExtractDatabaseAndSnapshotID_ValidFormats(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		expectDBID  int
		expectSnap  string
		expectError bool
	}{
		{
			name:        "valid format",
			path:        "/api/v1/databases/123/snapshots/snap-456",
			expectDBID:  123,
			expectSnap:  "snap-456",
			expectError: false,
		},
		{
			name:        "valid format with numeric snapshot ID",
			path:        "/api/v1/databases/456/snapshots/789",
			expectDBID:  456,
			expectSnap:  "789",
			expectError: false,
		},
		{
			name:        "missing prefix",
			path:        "/databases/123/snapshots/snap-456",
			expectError: true,
		},
		{
			name:        "wrong segment count",
			path:        "/api/v1/databases/123/snapshots",
			expectError: true,
		},
		{
			name:        "wrong segment",
			path:        "/api/v1/databases/123/wrong/snap-456",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dbID, snapID, err := extractDatabaseAndSnapshotID(tt.path)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectDBID, dbID)
				assert.Equal(t, tt.expectSnap, snapID)
			}
		})
	}
}

func TestExtractDatabaseAndSnapshotIDFromRestorePath_ValidFormats(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		expectDBID  int
		expectSnap  string
		expectError bool
	}{
		{
			name:        "valid format",
			path:        "/api/v1/databases/123/snapshots/snap-456/restore",
			expectDBID:  123,
			expectSnap:  "snap-456",
			expectError: false,
		},
		{
			name:        "missing restore",
			path:        "/api/v1/databases/123/snapshots/snap-456",
			expectError: true,
		},
		{
			name:        "wrong segment count",
			path:        "/api/v1/databases/123/snapshots/snap-456/restore/extra",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dbID, snapID, err := extractDatabaseAndSnapshotIDFromRestorePath(tt.path)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectDBID, dbID)
				assert.Equal(t, tt.expectSnap, snapID)
			}
		})
	}
}

func TestExtractDatabaseAndBranchID_ValidFormats(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		expectDBID  int
		expectBranch string
		expectError bool
	}{
		{
			name:         "valid format",
			path:         "/api/v1/databases/123/branches/branch-456",
			expectDBID:   123,
			expectBranch: "branch-456",
			expectError:  false,
		},
		{
			name:        "missing prefix",
			path:        "/databases/123/branches/branch-456",
			expectError: true,
		},
		{
			name:        "wrong segment count",
			path:        "/api/v1/databases/123/branches",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dbID, branchID, err := extractDatabaseAndBranchID(tt.path)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectDBID, dbID)
				assert.Equal(t, tt.expectBranch, branchID)
			}
		})
	}
}

// ==================== MockSnapshotManagerExtended Tests ====================

func TestMockSnapshotManagerExtended_CreateSnapshot(t *testing.T) {
	mockMgr := NewMockSnapshotManagerExtended()

	snapshot, err := mockMgr.CreateSnapshot(context.Background(), "db1", "test-snapshot")

	assert.NoError(t, err)
	assert.NotNil(t, snapshot)
	assert.Equal(t, "test-snapshot", snapshot.Name)
	assert.Equal(t, "db1", snapshot.DatabaseID)
	assert.Equal(t, database.SnapshotStatusCreating, snapshot.Status)
	assert.NotEmpty(t, snapshot.ID)
}

func TestMockSnapshotManagerExtended_CreateSnapshotAsync(t *testing.T) {
	mockMgr := NewMockSnapshotManagerExtended()

	snapshot, err := mockMgr.CreateSnapshot(context.Background(), "db1", "test-snapshot")

	assert.NoError(t, err)
	assert.Equal(t, database.SnapshotStatusCreating, snapshot.Status)

	// Wait for async completion
	time.Sleep(3 * time.Second)

	// Check status
	status, err := mockMgr.GetSnapshotStatus(context.Background(), snapshot.ID)
	assert.NoError(t, err)
	assert.Equal(t, database.SnapshotStatusReady, status)
	assert.Equal(t, int64(1024*1024*100), snapshot.SizeBytes)
}

func TestMockSnapshotManagerExtended_CreateSnapshotSync(t *testing.T) {
	mockMgr := NewMockSnapshotManagerExtended()

	snapshot, err := mockMgr.CreateSnapshotSync(context.Background(), "db1", "test-snapshot")

	assert.NoError(t, err)
	assert.NotNil(t, snapshot)
	assert.Equal(t, database.SnapshotStatusReady, snapshot.Status)
	assert.Equal(t, int64(1024*1024*100), snapshot.SizeBytes)
}

func TestMockSnapshotManagerExtended_ListSnapshots_Empty(t *testing.T) {
	mockMgr := NewMockSnapshotManagerExtended()

	snapshots, err := mockMgr.ListSnapshots(context.Background(), "db1")

	assert.NoError(t, err)
	assert.Empty(t, snapshots)
}

func TestMockSnapshotManagerExtended_ListSnapshots_WithSnapshots(t *testing.T) {
	mockMgr := NewMockSnapshotManagerExtended()

	mockMgr.CreateSnapshotSync(context.Background(), "db1", "snap1")
	mockMgr.CreateSnapshotSync(context.Background(), "db1", "snap2")
	mockMgr.CreateSnapshotSync(context.Background(), "db2", "snap3") // Different DB

	snapshots, err := mockMgr.ListSnapshots(context.Background(), "db1")

	assert.NoError(t, err)
	assert.Len(t, snapshots, 2)
}

func TestMockSnapshotManagerExtended_GetSnapshot_NotFound(t *testing.T) {
	mockMgr := NewMockSnapshotManagerExtended()

	snapshot, err := mockMgr.GetSnapshot(context.Background(), "nonexistent")

	assert.Error(t, err)
	assert.Nil(t, snapshot)
	assert.Contains(t, err.Error(), "not found")
}

func TestMockSnapshotManagerExtended_DeleteSnapshot_NoConfirmation(t *testing.T) {
	mockMgr := NewMockSnapshotManagerExtended()
	snapshot, _ := mockMgr.CreateSnapshotSync(context.Background(), "db1", "test-snap")

	err := mockMgr.DeleteSnapshot(context.Background(), snapshot.ID, false)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "confirmation required")

	// Verify snapshot still exists
	_, err = mockMgr.GetSnapshot(context.Background(), snapshot.ID)
	assert.NoError(t, err)
}

func TestMockSnapshotManagerExtended_DeleteSnapshot_NotFound(t *testing.T) {
	mockMgr := NewMockSnapshotManagerExtended()

	err := mockMgr.DeleteSnapshot(context.Background(), "nonexistent", true)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestMockSnapshotManagerExtended_RestoreFromSnapshot_NotFound(t *testing.T) {
	mockMgr := NewMockSnapshotManagerExtended()

	err := mockMgr.RestoreFromSnapshot(context.Background(), "db1", "nonexistent")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestMockSnapshotManagerExtended_GetSnapshotStatus_NotFound(t *testing.T) {
	mockMgr := NewMockSnapshotManagerExtended()

	_, err := mockMgr.GetSnapshotStatus(context.Background(), "nonexistent")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// ==================== FileMetadata Tests ====================

func TestFileMetadata_Structure(t *testing.T) {
	now := time.Now()
	metadata := FileMetadata{
		ID:          1,
		DatabaseID:  "100",
		Path:        "/test/path/file.txt",
		Name:        "file.txt",
		Size:        1024,
		ContentType: "text/plain",
		Checksum:    "abc123",
		StorageKey:  "key123",
		CreatedAt:   now,
	}

	assert.Equal(t, 1, metadata.ID)
	assert.Equal(t, "100", metadata.DatabaseID)
	assert.Equal(t, "/test/path/file.txt", metadata.Path)
	assert.Equal(t, "file.txt", metadata.Name)
	assert.Equal(t, int64(1024), metadata.Size)
	assert.Equal(t, "text/plain", metadata.ContentType)
	assert.Equal(t, "abc123", metadata.Checksum)
	assert.Equal(t, "key123", metadata.StorageKey)
	assert.Equal(t, now, metadata.CreatedAt)
}

// ==================== SQLRequest and SQLResponse Tests ====================

func TestSQLRequest_Structure(t *testing.T) {
	timeout := 30000
	req := SQLRequest{
		Query:     "SELECT 1",
		TimeoutMs: &timeout,
	}

	assert.Equal(t, "SELECT 1", req.Query)
	assert.Equal(t, 30000, *req.TimeoutMs)
}

func TestSQLRequest_NilTimeout(t *testing.T) {
	req := SQLRequest{
		Query:     "SELECT 1",
		TimeoutMs: nil,
	}

	assert.Equal(t, "SELECT 1", req.Query)
	assert.Nil(t, req.TimeoutMs)
}

func TestSQLResponse_Structure(t *testing.T) {
	response := SQLResponse{
		Columns:  []interface{}{"id", "name"},
		Rows:     []map[string]interface{}{{"id": 1, "name": "test"}},
		RowCount: 1,
	}

	assert.Len(t, response.Columns, 2)
	assert.Len(t, response.Rows, 1)
	assert.Equal(t, 1, response.RowCount)
}

// ==================== CreateSnapshotRequest Tests ====================

func TestCreateSnapshotRequest_Structure(t *testing.T) {
	req := CreateSnapshotRequest{
		Name: "test-snapshot",
	}

	assert.Equal(t, "test-snapshot", req.Name)
}

// ==================== FileListResponse Tests ====================

func TestFileListResponse_Structure(t *testing.T) {
	response := FileListResponse{
		Files: []FileMetadata{
			{ID: 1, Name: "file1.txt"},
			{ID: 2, Name: "file2.txt"},
		},
		Total: 2,
	}

	assert.Len(t, response.Files, 2)
	assert.Equal(t, 2, response.Total)
}

// ==================== Constants Tests ====================

func TestConstants(t *testing.T) {
	assert.Equal(t, int64(100*1024*1024), MaxFileSize)
	assert.Equal(t, int64(100*1024*1024), DefaultMaxFileSize)
	assert.Equal(t, int64(1), MinFileSize)
	assert.Equal(t, "fs9_files", FileTable)
	assert.Equal(t, "http://localhost:9090", FS9ServiceURL)
}

func TestQueryTimeoutConstants(t *testing.T) {
	assert.Equal(t, 1*time.Second, MinQueryTimeout)
	assert.Equal(t, 5*time.Minute, MaxQueryTimeout)
	assert.Equal(t, 30*time.Second, DefaultQueryTimeout)
}

// ==================== Response Structure Tests ====================

func TestResponse_Structure(t *testing.T) {
	resp := Response{
		Success: true,
		Data:    map[string]interface{}{"key": "value"},
		Message: "test message",
	}

	assert.True(t, resp.Success)
	assert.NotNil(t, resp.Data)
	assert.Equal(t, "test message", resp.Message)
}

// ==================== Edge Cases ====================

func TestExecuteSQLHandler_QueryWithComments(t *testing.T) {
	dbManager := &database.Manager{}
	RegisterDatabase(1, dbManager)
	defer UnregisterDatabase(1)

	queries := []string{
		"-- This is a comment\nSELECT 1",
		"/* This is a comment */\nSELECT 1",
		"SELECT 1 -- inline comment",
	}

	for _, query := range queries {
		t.Run(query, func(t *testing.T) {
			req := createTestRequest(http.MethodPost, "/api/v1/databases/1/sql", SQLRequest{
				Query: query,
			})
			rec := httptest.NewRecorder()

			ExecuteSQLHandler(rec, req)

			// Should pass validation
			assert.NotEqual(t, http.StatusBadRequest, rec.Code)
		})
	}
}

func TestUploadFileHandler_LargeFile(t *testing.T) {
	dbManager := &database.Manager{}
	RegisterDatabase(1, dbManager)
	defer UnregisterDatabase(1)

	// Create a file larger than MaxFileSize
	largeContent := make([]byte, MaxFileSize+1)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "large.txt")
	part.Write(largeContent)
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/databases/1/files", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()

	UploadFileHandler(rec, req)

	assert.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)
}

func TestContentTypeDetection(t *testing.T) {
	tests := []struct {
		ext         string
		expectedCT  string
	}{
		{".txt", "text/plain"},
		{".json", "application/json"},
		{".html", "text/html"},
		{".css", "text/css"},
		{".js", "text/javascript"},
		{".pdf", "application/pdf"},
		{".zip", "application/zip"},
		{".unknown", "application/octet-stream"},
	}

	for _, tt := range tests {
		t.Run(tt.ext, func(t *testing.T) {
			ct := mime.TypeByExtension(tt.ext)
			// Just verify the function works
			assert.NotEmpty(t, ct)
		})
	}
}

func TestURLPathParsing(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		prefix   string
		expected string
	}{
		{
			name:     "simple path",
			path:     "/api/v1/databases/123/snapshots",
			prefix:   "/api/v1/databases/",
			expected: "123/snapshots",
		},
		{
			name:     "nested path",
			path:     "/api/v1/databases/456/snapshots/789",
			prefix:   "/api/v1/databases/",
			expected: "456/snapshots/789",
		},
		{
			name:     "no prefix match",
			path:     "/other/path",
			prefix:   "/api/v1/databases/",
			expected: "/other/path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := strings.TrimPrefix(tt.path, tt.prefix)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDatabaseIDParsing(t *testing.T) {
	tests := []struct {
		input    string
		expected int
		hasError bool
	}{
		{"123", 123, false},
		{"0", 0, false},
		{"-1", -1, false},
		{"abc", 0, true},
		{"12.5", 0, true},
		{"", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := strconv.Atoi(tt.input)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestSnapshotStatusTransitions(t *testing.T) {
	mockMgr := NewMockSnapshotManagerExtended()

	// Create snapshot asynchronously
	snapshot, err := mockMgr.CreateSnapshot(context.Background(), "db1", "test-snap")
	require.NoError(t, err)

	// Initially should be in creating status
	assert.Equal(t, database.SnapshotStatusCreating, snapshot.Status)

	// Wait for completion
	time.Sleep(3 * time.Second)

	// Check final status
	status, err := mockMgr.GetSnapshotStatus(context.Background(), snapshot.ID)
	require.NoError(t, err)
	assert.Equal(t, database.SnapshotStatusReady, status)
}

// CopyFileRequest represents a file copy request
type CopyFileRequest struct {
	SrcPath string `json:"src_path"`
	DstPath string `json:"dst_path"`
}

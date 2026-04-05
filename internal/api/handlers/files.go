package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/open-db9/db9/internal/database"
)

const (
	// MaxFileSize is the maximum file size allowed (100MB)
	MaxFileSize = 100 * 1024 * 1024
	// DefaultMaxFileSize is the default maximum file size (100MB)
	DefaultMaxFileSize = 100 * 1024 * 1024
	// MinFileSize is the minimum non-zero file size (1 byte)
	MinFileSize = 1
	// FileTable is the table name for file metadata
	FileTable = "fs9_files"
)

var (
	FS9ServiceURL = "http://localhost:9090"
	// fs9Client is the global fs9-service client
	fs9Client *FS9Client
)

// init initializes the fs9 client
func init() {
	if envURL := os.Getenv("DB9_FS9_URL"); envURL != "" {
		FS9ServiceURL = envURL
	}
	fs9Client = NewFS9Client(FS9ServiceURL)
}

// FileMetadata represents file metadata response
type FileMetadata struct {
	ID          int       `json:"id"`
	DatabaseID  string    `json:"database_id"`
	Path        string    `json:"path"`
	Name        string    `json:"name"`
	Size        int64     `json:"size"`
	ContentType string    `json:"content_type"`
	Checksum    string    `json:"checksum,omitempty"`
	StorageKey  string    `json:"storage_key,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// FileListResponse represents a list of files
type FileListResponse struct {
	Files []FileMetadata `json:"files"`
	Total int            `json:"total"`
}

// UploadFileHandler handles file uploads
func UploadFileHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := strings.TrimSuffix(r.URL.Path, "/")
	parts := strings.Split(path, "/")
	if len(parts) < 6 || parts[5] != "files" {
		http.Error(w, "Invalid URL format. Expected: /api/v1/databases/:id/files", http.StatusBadRequest)
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

	// Parse multipart form (max 100MB + some overhead for form fields)
	if err := r.ParseMultipartForm(MaxFileSize + 10*1024*1024); err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse multipart form: %v", err), http.StatusBadRequest)
		return
	}

	// Get file from form
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get file from form: %v", err), http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Check file size
	if header.Size > MaxFileSize {
		http.Error(w, fmt.Sprintf("File size exceeds maximum allowed size of %d bytes", MaxFileSize), http.StatusRequestEntityTooLarge)
		return
	}

	// Get path from form (optional)
	filePath := r.FormValue("path")
	if filePath == "" {
		filePath = "/" + header.Filename
	}

	// Ensure path starts with /
	if !strings.HasPrefix(filePath, "/") {
		filePath = "/" + filePath
	}

	// Detect content type
	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = mime.TypeByExtension(filepath.Ext(header.Filename))
		if contentType == "" {
			contentType = "application/octet-stream"
		}
	}

	// Read file content and calculate checksum
	hash := sha256.New()
	var size int64

	// Create a temporary file to store content
	tempFile, err := os.CreateTemp("", "upload-*.tmp")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create temp file: %v", err), http.StatusInternalServerError)
		return
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath)
	defer tempFile.Close()

	// Copy file to temp and calculate checksum
	multiWriter := io.MultiWriter(tempFile, hash)
	size, err = io.Copy(multiWriter, file)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read file: %v", err), http.StatusInternalServerError)
		return
	}

	checksum := hex.EncodeToString(hash.Sum(nil))

	// Get database connection
	ctx := r.Context()

	// Seek back to beginning of temp file for upload
	if _, err := tempFile.Seek(0, 0); err != nil {
		http.Error(w, fmt.Sprintf("Failed to seek temp file: %v", err), http.StatusInternalServerError)
		return
	}

	// Generate storage key: dbID/filePath
	storageKey := fmt.Sprintf("%s%s", dbIDStr, filePath)

	// Upload file to fs9-service
	uploadResp, err := fs9Client.UploadFile(ctx, storageKey, tempFile, contentType)
	if err != nil {
		// Log error but continue - file metadata will still be saved
		// This allows graceful degradation if fs9-service is unavailable
		fmt.Printf("Warning: failed to upload to fs9-service: %v\n", err)
		storageKey = ""
	} else {
		storageKey = uploadResp.Key
	}

	// Check if fs9_files table exists, create if not
	if err := ensureFileTable(ctx, manager); err != nil {
		http.Error(w, fmt.Sprintf("Failed to ensure file table: %v", err), http.StatusInternalServerError)
		return
	}

	// Check if file already exists
	var existingID int
	checkQuery := fmt.Sprintf("SELECT id FROM %s WHERE database_id = $1 AND path = $2", FileTable)
	err = manager.GetPool().QueryRow(ctx, checkQuery, dbIDStr, filePath).Scan(&existingID)
	if err == nil {
		// File exists, update it
		updateQuery := fmt.Sprintf(`
			UPDATE %s
			SET name = $1, size = $2, content_type = $3, checksum = $4, storage_key = $5, updated_at = $6
			WHERE id = $6
		`, FileTable)
		err = manager.GetPool().Execute(ctx, updateQuery, header.Filename, size, contentType, checksum, storageKey, time.Now(), existingID)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to update file metadata: %v", err), http.StatusInternalServerError)
			return
		}
	} else if err != pgx.ErrNoRows {
		http.Error(w, fmt.Sprintf("Failed to check existing file: %v", err), http.StatusInternalServerError)
		return
	} else {
		// Insert new file metadata
		insertQuery := fmt.Sprintf(`
			INSERT INTO %s (database_id, path, name, size, content_type, checksum, storage_key, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			RETURNING id
		`, FileTable)
		err = manager.GetPool().QueryRow(ctx, insertQuery, dbIDStr, filePath, header.Filename, size, contentType, checksum, storageKey, time.Now(), time.Now()).Scan(&existingID)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to save file metadata: %v", err), http.StatusInternalServerError)
			return
		}
	}

	// Build response
	metadata := FileMetadata{
		ID:          existingID,
		DatabaseID:  dbIDStr,
		Path:        filePath,
		Name:        header.Filename,
		Size:        size,
		ContentType: contentType,
		Checksum:    checksum,
		StorageKey:  storageKey,
		CreatedAt:   time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(Response{
		Success: true,
		Data:    metadata,
	})
}

// DownloadFileHandler handles file downloads
func DownloadFileHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract database ID and file path from URL path
	// Expected path: /api/v1/databases/:id/files/*
	prefix := "/api/v1/databases/"
	if !strings.HasPrefix(r.URL.Path, prefix) {
		http.Error(w, "Invalid URL format", http.StatusBadRequest)
		return
	}

	rest := strings.TrimPrefix(r.URL.Path, prefix)
	parts := strings.SplitN(rest, "/", 2)
	if len(parts) < 2 {
		http.Error(w, "Invalid URL format. Expected: /api/v1/databases/:id/files/:path", http.StatusBadRequest)
		return
	}

	dbIDStr := parts[0]
	filePath := "/" + parts[1]

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

	ctx := r.Context()

	// Query file metadata
	query := fmt.Sprintf("SELECT id, database_id, path, name, size, content_type, checksum, storage_key, created_at FROM %s WHERE database_id = $1 AND path = $2", FileTable)
	rows, err := manager.GetPool().Query(ctx, query, dbIDStr, filePath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to query file metadata: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	if !rows.Next() {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	var metadata FileMetadata
	err = rows.Scan(&metadata.ID, &metadata.DatabaseID, &metadata.Path, &metadata.Name, &metadata.Size, &metadata.ContentType, &metadata.Checksum, &metadata.StorageKey, &metadata.CreatedAt)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to scan file metadata: %v", err), http.StatusInternalServerError)
		return
	}

	// Retrieve file content from fs9-service
	if metadata.StorageKey == "" {
		http.Error(w, "File content not available", http.StatusServiceUnavailable)
		return
	}

	reader, contentType, err := fs9Client.DownloadFile(ctx, metadata.StorageKey)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to download file content: %v", err), http.StatusInternalServerError)
		return
	}
	defer reader.Close()

	// Set headers for file download
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", strconv.FormatInt(metadata.Size, 10))
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", metadata.Name))

	// Stream file content
	if _, err := io.Copy(w, reader); err != nil {
		fmt.Printf("Warning: failed to stream file content: %v\n", err)
	}
}

// DeleteFileHandler handles file deletion
func DeleteFileHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract database ID and file path from URL path
	prefix := "/api/v1/databases/"
	if !strings.HasPrefix(r.URL.Path, prefix) {
		http.Error(w, "Invalid URL format", http.StatusBadRequest)
		return
	}

	rest := strings.TrimPrefix(r.URL.Path, prefix)
	parts := strings.SplitN(rest, "/", 2)
	if len(parts) < 2 {
		http.Error(w, "Invalid URL format. Expected: /api/v1/databases/:id/files/:path", http.StatusBadRequest)
		return
	}

	dbIDStr := parts[0]
	filePath := "/" + parts[1]

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

	ctx := r.Context()

	// Query storage_key before deleting metadata
	var storageKey string
	selectQuery := fmt.Sprintf("SELECT storage_key FROM %s WHERE database_id = $1 AND path = $2", FileTable)
	err = manager.GetPool().QueryRow(ctx, selectQuery, dbIDStr, filePath).Scan(&storageKey)
	if err != nil && err != pgx.ErrNoRows {
		http.Error(w, fmt.Sprintf("Failed to query file metadata: %v", err), http.StatusInternalServerError)
		return
	}

	// Delete file content from fs9-service if storage_key exists
	if storageKey != "" {
		if err := fs9Client.DeleteFile(ctx, storageKey); err != nil {
			// Log error but continue with metadata deletion
			fmt.Printf("Warning: failed to delete file from storage: %v\n", err)
		}
	}

	// Delete file metadata
	deleteQuery := fmt.Sprintf("DELETE FROM %s WHERE database_id = $1 AND path = $2", FileTable)
	err = manager.GetPool().Execute(ctx, deleteQuery, dbIDStr, filePath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete file: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(Response{
		Success: true,
		Message: "File deleted successfully",
	})
}

// ListFilesHandler handles listing files
func ListFilesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := strings.TrimSuffix(r.URL.Path, "/")
	parts := strings.Split(path, "/")
	if len(parts) < 6 || parts[5] != "files" {
		http.Error(w, "Invalid URL format. Expected: /api/v1/databases/:id/files", http.StatusBadRequest)
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

	ctx := r.Context()

	// Parse query parameters
	pathFilter := r.URL.Query().Get("path")
	limitStr := r.URL.Query().Get("limit")
	limit := 100 // default limit
	if limitStr != "" {
		limit, err = strconv.Atoi(limitStr)
		if err != nil || limit <= 0 || limit > 1000 {
			limit = 100
		}
	}

	// Build query
	query := fmt.Sprintf("SELECT id, database_id, path, name, size, content_type, checksum, created_at FROM %s WHERE database_id = $1", FileTable)
	args := []interface{}{dbIDStr}
	argIdx := 2

	if pathFilter != "" {
		// Support prefix matching for directory listing
		if strings.HasSuffix(pathFilter, "/") {
			query += fmt.Sprintf(" AND path LIKE $%d", argIdx)
			args = append(args, pathFilter+"%")
			argIdx++
		} else {
			query += fmt.Sprintf(" AND path = $%d", argIdx)
			args = append(args, pathFilter)
			argIdx++
		}
	}

	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d", argIdx)
	args = append(args, limit)

	// Execute query
	rows, err := manager.GetPool().Query(ctx, query, args...)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list files: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Collect results
	files := []FileMetadata{}
	for rows.Next() {
		var metadata FileMetadata
		err = rows.Scan(&metadata.ID, &metadata.DatabaseID, &metadata.Path, &metadata.Name, &metadata.Size, &metadata.ContentType, &metadata.Checksum, &metadata.StorageKey, &metadata.CreatedAt)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to scan file metadata: %v", err), http.StatusInternalServerError)
			return
		}
		files = append(files, metadata)
	}

	// Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE database_id = $1", FileTable)
	countArgs := []interface{}{dbIDStr}
	if pathFilter != "" {
		if strings.HasSuffix(pathFilter, "/") {
			countQuery += " AND path LIKE $2"
			countArgs = append(countArgs, pathFilter+"%")
		} else {
			countQuery += " AND path = $2"
			countArgs = append(countArgs, pathFilter)
		}
	}

	var total int
	err = manager.GetPool().QueryRow(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to count files: %v", err), http.StatusInternalServerError)
		return
	}

	response := FileListResponse{
		Files: files,
		Total: total,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(Response{
		Success: true,
		Data:    response,
	})
}

// CopyFileHandler handles file copying within database storage
func CopyFileHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := strings.TrimSuffix(r.URL.Path, "/")
	parts := strings.Split(path, "/")
	if len(parts) < 7 || parts[6] != "copy" {
		http.Error(w, "Invalid URL format. Expected: /api/v1/databases/:id/files/copy", http.StatusBadRequest)
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

	// Parse request body for source and destination paths
	var req struct {
		SrcPath string `json:"src_path"`
		DstPath string `json:"dst_path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse request body: %v", err), http.StatusBadRequest)
		return
	}

	// Ensure paths start with /
	if !strings.HasPrefix(req.SrcPath, "/") {
		req.SrcPath = "/" + req.SrcPath
	}
	if !strings.HasPrefix(req.DstPath, "/") {
		req.DstPath = "/" + req.DstPath
	}

	ctx := r.Context()

	// Query source file metadata
	query := fmt.Sprintf("SELECT id, database_id, path, name, size, content_type, checksum, storage_key, created_at FROM %s WHERE database_id = $1 AND path = $2", FileTable)
	var srcMetadata FileMetadata
	err = manager.GetPool().QueryRow(ctx, query, dbIDStr, req.SrcPath).Scan(
		&srcMetadata.ID, &srcMetadata.DatabaseID, &srcMetadata.Path, &srcMetadata.Name,
		&srcMetadata.Size, &srcMetadata.ContentType, &srcMetadata.Checksum, &srcMetadata.StorageKey, &srcMetadata.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		http.Error(w, "Source file not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to query source file: %v", err), http.StatusInternalServerError)
		return
	}

	// Check if destination already exists
	var existingID int
	checkQuery := fmt.Sprintf("SELECT id FROM %s WHERE database_id = $1 AND path = $2", FileTable)
	err = manager.GetPool().QueryRow(ctx, checkQuery, dbIDStr, req.DstPath).Scan(&existingID)

	// Generate new storage key for destination
	dstStorageKey := fmt.Sprintf("%s%s", dbIDStr, req.DstPath)

	if err == pgx.ErrNoRows {
		// Insert new file metadata for destination
		insertQuery := fmt.Sprintf(`
			INSERT INTO %s (database_id, path, name, size, content_type, checksum, storage_key, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			RETURNING id
		`, FileTable)
		err = manager.GetPool().QueryRow(ctx, insertQuery, dbIDStr, req.DstPath, filepath.Base(req.DstPath), srcMetadata.Size, srcMetadata.ContentType, srcMetadata.Checksum, dstStorageKey, time.Now(), time.Now()).Scan(&existingID)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to create file copy metadata: %v", err), http.StatusInternalServerError)
			return
		}
	} else if err != nil {
		http.Error(w, fmt.Sprintf("Failed to check destination file: %v", err), http.StatusInternalServerError)
		return
	} else {
		// Update existing destination file metadata
		updateQuery := fmt.Sprintf(`
			UPDATE %s
			SET name = $1, size = $2, content_type = $3, checksum = $4, storage_key = $5, updated_at = $6
			WHERE id = $7
		`, FileTable)
		err = manager.GetPool().Execute(ctx, updateQuery, filepath.Base(req.DstPath), srcMetadata.Size, srcMetadata.ContentType, srcMetadata.Checksum, dstStorageKey, time.Now(), existingID)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to update file copy metadata: %v", err), http.StatusInternalServerError)
			return
		}
	}

	// Copy file content in fs9-service if storage_key exists
	if srcMetadata.StorageKey != "" {
		// For fs9-service, we need to copy the file content
		// Download from source and upload to destination
		reader, contentType, err := fs9Client.DownloadFile(ctx, srcMetadata.StorageKey)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to download source file content: %v", err), http.StatusInternalServerError)
			return
		}
		defer reader.Close()

		// Upload to destination
		uploadResp, err := fs9Client.UploadFile(ctx, dstStorageKey, reader, contentType)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to upload file copy: %v", err), http.StatusInternalServerError)
			return
		}
		_ = uploadResp // Suppress unused warning
	}

	// Build response
	dstMetadata := FileMetadata{
		ID:          existingID,
		DatabaseID:  dbIDStr,
		Path:        req.DstPath,
		Name:        filepath.Base(req.DstPath),
		Size:        srcMetadata.Size,
		ContentType: srcMetadata.ContentType,
		Checksum:    srcMetadata.Checksum,
		StorageKey:  dstStorageKey,
		CreatedAt:   time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(Response{
		Success: true,
		Data:    dstMetadata,
		Message: "File copied successfully",
	})
}

// ensureFileTable creates the fs9_files table if it doesn't exist
func ensureFileTable(ctx context.Context, manager *database.Manager) error {
	createTableQuery := `
		CREATE TABLE IF NOT EXISTS fs9_files (
			id SERIAL PRIMARY KEY,
			database_id UUID NOT NULL,
			path VARCHAR(1024) NOT NULL,
			name VARCHAR(512) NOT NULL,
			size BIGINT NOT NULL,
			content_type VARCHAR(256),
			checksum VARCHAR(128),
			storage_key VARCHAR(512),
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
			UNIQUE(database_id, path)
		);

		CREATE INDEX IF NOT EXISTS idx_fs9_files_database_id ON fs9_files(database_id);
		CREATE INDEX IF NOT EXISTS idx_fs9_files_path ON fs9_files(path);
		CREATE INDEX IF NOT EXISTS idx_fs9_files_storage_key ON fs9_files(storage_key);
	`

	return manager.RawExec(ctx, createTableQuery)
}

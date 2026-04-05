package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/open-db9/db9/internal/filesystem"
	"github.com/open-db9/db9/pkg/logger"
)

const defaultMaxFileSize = 100 * 1024 * 1024 // 100MB

func main() {
	// Load configuration
	port := getEnv("FS9_SERVICE_PORT", "9090")
	storagePath := getEnv("STORAGE_PATH", "/data/fs9")
	maxFileSize := getEnvAsInt64("MAX_FILE_SIZE", defaultMaxFileSize)

	// Initialize storage
	storage, err := filesystem.NewDiskStorage(storagePath)
	if err != nil {
		logger.Fatal("Failed to initialize storage: %v", err)
	}

	// Create server
	server := &Server{
		storage:      storage,
		maxFileSize:  maxFileSize,
		storagePath:  storagePath,
	}

	// Setup routes
	mux := http.NewServeMux()
	mux.HandleFunc("/health", server.handleHealth)
	mux.HandleFunc("/files/upload", server.handleUpload)
	mux.HandleFunc("/files/", server.handleFiles)

	// Start server
	addr := ":" + port
	logger.Info("FS9 Service starting on port %s", port)
	logger.Info("Storage path: %s", storagePath)
	logger.Info("Max file size: %d bytes", maxFileSize)

	if err := http.ListenAndServe(addr, loggingMiddleware(mux)); err != nil {
		logger.Fatal("Server failed to start: %v", err)
	}
}

// Server holds the service dependencies
type Server struct {
	storage     filesystem.Storage
	maxFileSize int64
	storagePath string
}

// handleHealth handles health check requests
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":    "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"service":   "fs9",
	})
}

// handleUpload handles file upload requests
func (s *Server) handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse multipart form (max memory: 32MB)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		logger.Error("Failed to parse multipart form: %v", err)
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		logger.Error("Failed to get file from form: %v", err)
		http.Error(w, "No file provided", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Check file size
	if header.Size > s.maxFileSize {
		logger.Error("File size %d exceeds maximum %d", header.Size, s.maxFileSize)
		http.Error(w, fmt.Sprintf("File size exceeds maximum of %d bytes", s.maxFileSize), http.StatusRequestEntityTooLarge)
		return
	}

	// Get key from form or use filename
	key := r.FormValue("key")
	if key == "" {
		key = header.Filename
	}

	// Sanitize key
	key = filepath.Clean(filepath.Join("/", key))
	key = strings.TrimPrefix(key, "/")

	logger.Info("Uploading file: key=%s, size=%d, content_type=%s", key, header.Size, header.Header.Get("Content-Type"))

	// Create metadata
	metadata := filesystem.Metadata{
		ContentType: header.Header.Get("Content-Type"),
		Size:        header.Size,
		ModTime:     time.Now(),
		Custom:      make(map[string]string),
	}

	// Store file
	ctx := r.Context()
	if err := s.storage.Put(ctx, key, file, metadata); err != nil {
		logger.Error("Failed to store file: %v", err)
		http.Error(w, "Failed to store file", http.StatusInternalServerError)
		return
	}

	logger.Info("File uploaded successfully: %s", key)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"key":     key,
		"size":    header.Size,
		"success": true,
	})
}

// handleFiles handles file operations (get, delete, list)
func (s *Server) handleFiles(w http.ResponseWriter, r *http.Request) {
	// Extract key from path
	path := strings.TrimPrefix(r.URL.Path, "/files/")

	// Handle list operation
	if path == "" || path == "/" {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		s.handleList(w, r)
		return
	}

	// Handle specific file operations
	switch r.Method {
	case http.MethodGet:
		s.handleDownload(w, r, path)
	case http.MethodDelete:
		s.handleDelete(w, r, path)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleDownload handles file download requests
func (s *Server) handleDownload(w http.ResponseWriter, r *http.Request, key string) {
	ctx := r.Context()

	// Check if file exists
	exists, err := s.storage.Exists(ctx, key)
	if err != nil {
		logger.Error("Failed to check file existence: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if !exists {
		logger.Error("File not found: %s", key)
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	// Get file size
	size, err := s.storage.Size(ctx, key)
	if err != nil {
		logger.Error("Failed to get file size: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Get file
	reader, err := s.storage.Get(ctx, key)
	if err != nil {
		logger.Error("Failed to get file: %v", err)
		http.Error(w, "Failed to retrieve file", http.StatusInternalServerError)
		return
	}
	defer reader.Close()

	// Detect content type
	contentType := detectContentType(key)

	logger.Info("Downloading file: key=%s, size=%d, content_type=%s", key, size, contentType)

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filepath.Base(key)))

	if _, err := io.Copy(w, reader); err != nil {
		logger.Error("Failed to write file to response: %v", err)
	}
}

// handleDelete handles file deletion requests
func (s *Server) handleDelete(w http.ResponseWriter, r *http.Request, key string) {
	ctx := r.Context()

	logger.Info("Deleting file: %s", key)

	if err := s.storage.Delete(ctx, key); err != nil {
		logger.Error("Failed to delete file: %v", err)
		http.Error(w, "Failed to delete file", http.StatusInternalServerError)
		return
	}

	logger.Info("File deleted successfully: %s", key)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"key":     key,
		"success": true,
	})
}

// handleList handles file listing requests
func (s *Server) handleList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get prefix from query parameter
	prefix := r.URL.Query().Get("prefix")

	logger.Info("Listing files with prefix: %s", prefix)

	entries, err := s.storage.List(ctx, prefix)
	if err != nil {
		logger.Error("Failed to list files: %v", err)
		http.Error(w, "Failed to list files", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"count":  len(entries),
		"files":  entries,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// loggingMiddleware wraps the handler with logging
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		logger.Info("%s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
		logger.Info("Completed %s %s in %v", r.Method, r.URL.Path, time.Since(start))
	})
}

// getEnv retrieves an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsInt64 retrieves an environment variable as int64 or returns a default value
func getEnvAsInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intVal
		}
	}
	return defaultValue
}

// detectContentType detects content type based on file extension
func detectContentType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".html", ".htm":
		return "text/html"
	case ".css":
		return "text/css"
	case ".js":
		return "application/javascript"
	case ".json":
		return "application/json"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".svg":
		return "image/svg+xml"
	case ".pdf":
		return "application/pdf"
	case ".txt":
		return "text/plain"
	case ".xml":
		return "application/xml"
	case ".zip":
		return "application/zip"
	case ".tar":
		return "application/x-tar"
	case ".gz":
		return "application/gzip"
	default:
		return "application/octet-stream"
	}
}

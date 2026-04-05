package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/open-db9/db9/pkg/logger"
	"github.com/spf13/cobra"
)

// FSConfig holds filesystem command configuration
type FSConfig struct {
	BaseURL    string
	MaxRetries int
	Timeout    time.Duration
}

// FileMetadata represents file metadata from API
type FileMetadata struct {
	ID          int       `json:"id"`
	DatabaseID  int       `json:"database_id"`
	Path        string    `json:"path"`
	Name        string    `json:"name"`
	Size        int64     `json:"size"`
	ContentType string    `json:"content_type"`
	Checksum    string    `json:"checksum,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// FileListResponse represents API response for file listing
type FileListResponse struct {
	Files []FileMetadata `json:"files"`
	Total int            `json:"total"`
}

// APIResponse represents standard API response
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	Message string      `json:"message,omitempty"`
}

// Cmd represents the filesystem command
var Cmd = &cobra.Command{
	Use:   "fs",
	Short: "Filesystem management commands",
	Long: `Manage files in database storage.

Provides commands for:
  - Uploading files to database storage
  - Downloading files from database storage
  - Listing files in database storage
  - Deleting files from database storage
  - Copying files within database storage`,
}

var fsConfig FSConfig

func init() {
	// Initialize default config
	fsConfig = FSConfig{
		BaseURL:    "http://localhost:8080",
		MaxRetries: 3,
		Timeout:    30 * time.Second,
	}

	// Add subcommands
	Cmd.AddCommand(uploadCmd)
	Cmd.AddCommand(downloadCmd)
	Cmd.AddCommand(lsCmd)
	Cmd.AddCommand(rmCmd)
	Cmd.AddCommand(cpCmd)

	// Persistent flags
	Cmd.PersistentFlags().StringVar(&fsConfig.BaseURL, "api-url", fsConfig.BaseURL, "API base URL")
	Cmd.PersistentFlags().IntVar(&fsConfig.MaxRetries, "retries", fsConfig.MaxRetries, "Maximum number of retries")
	Cmd.PersistentFlags().DurationVar(&fsConfig.Timeout, "timeout", fsConfig.Timeout, "Request timeout")
}

// uploadCmd represents the upload command
var uploadCmd = &cobra.Command{
	Use:   "upload <database-id> <local-file> <remote-path>",
	Short: "Upload a file to database storage",
	Long: `Upload a local file to the database storage.

The remote-path specifies where the file will be stored in the database storage.
If the remote-path doesn't start with '/', it will be prepended automatically.

Example:
  db9 fs upload 1 ./myfile.txt /documents/myfile.txt
  db9 fs upload 1 ./data.csv /imports/data.csv`,
	Args: cobra.ExactArgs(3),
	RunE: runUpload,
}

// downloadCmd represents the download command
var downloadCmd = &cobra.Command{
	Use:   "download <database-id> <remote-path> <local-file>",
	Short: "Download a file from database storage",
	Long: `Download a file from the database storage to local filesystem.

The remote-path specifies the file to download from the database storage.
The local-file specifies where to save the downloaded file.

Example:
  db9 fs download 1 /documents/myfile.txt ./myfile.txt
  db9 fs download 1 /exports/result.csv ./result.csv`,
	Args: cobra.ExactArgs(3),
	RunE: runDownload,
}

// lsCmd represents the list command
var lsCmd = &cobra.Command{
	Use:   "ls <database-id> [path]",
	Short: "List files in database storage",
	Long: `List files in the database storage.

If path is provided, lists files in that directory.
If path ends with '/', performs prefix matching for directory listing.
If no path is provided, lists all files for the database.

Example:
  db9 fs ls 1
  db9 fs ls 1 /documents/
  db9 fs ls 1 /documents/report.pdf`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runList,
}

// rmCmd represents the remove command
var rmCmd = &cobra.Command{
	Use:   "rm <database-id> <path>",
	Short: "Delete a file from database storage",
	Long: `Delete a file from the database storage.

Example:
  db9 fs rm 1 /documents/oldfile.txt`,
	Args: cobra.ExactArgs(2),
	RunE: runRemove,
}

// cpCmd represents the copy command
var cpCmd = &cobra.Command{
	Use:   "cp <database-id> <src> <dst>",
	Short: "Copy a file within database storage",
	Long: `Copy a file to a new location within the database storage.

Example:
  db9 fs cp 1 /documents/file.txt /documents/file_backup.txt`,
	Args: cobra.ExactArgs(3),
	RunE: runCopy,
}

// runUpload executes the upload command
func runUpload(cmd *cobra.Command, args []string) error {
	dbID, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid database ID: %w", err)
	}

	localPath := args[1]
	remotePath := args[2]

	// Ensure remote path starts with /
	if !strings.HasPrefix(remotePath, "/") {
		remotePath = "/" + remotePath
	}

	// Check if local file exists
	fileInfo, err := os.Stat(localPath)
	if err != nil {
		return fmt.Errorf("local file not found: %w", err)
	}

	// Open local file
	file, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open local file: %w", err)
	}
	defer file.Close()

	// Detect content type
	contentType := mime.TypeByExtension(filepath.Ext(localPath))
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	logger.Info("Uploading file...")
	logger.Info("  Database ID: %d", dbID)
	logger.Info("  Local: %s", localPath)
	logger.Info("  Remote: %s", remotePath)
	logger.Info("  Size: %s", formatBytes(fileInfo.Size()))

	// Create multipart form
	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	// Create form file
	part, err := writer.CreateFormFile("file", filepath.Base(localPath))
	if err != nil {
		return fmt.Errorf("failed to create form file: %w", err)
	}

	// Copy file content with progress
	progressWriter := &progressWriter{
		Writer:  part,
		Total:   fileInfo.Size(),
		Name:    "Uploading",
		Verbose: logger.IsVerbose(),
	}

	if _, err := io.Copy(progressWriter, file); err != nil {
		return fmt.Errorf("failed to copy file content: %w", err)
	}

	// Add path field
	if err := writer.WriteField("path", remotePath); err != nil {
		return fmt.Errorf("failed to write path field: %w", err)
	}

	// Close multipart writer
	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close multipart writer: %w", err)
	}

	// Make request with retry
	url := fmt.Sprintf("%s/api/v1/databases/%d/files", fsConfig.BaseURL, dbID)
	var response APIResponse
	if err := retryRequest("POST", url, writer.FormDataContentType(), requestBody, &response); err != nil {
		return err
	}

	if !response.Success {
		return fmt.Errorf("upload failed: %s", response.Message)
	}

	// Parse response data
	dataBytes, err := json.Marshal(response.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal response data: %w", err)
	}

	var metadata FileMetadata
	if err := json.Unmarshal(dataBytes, &metadata); err != nil {
		return fmt.Errorf("failed to unmarshal file metadata: %w", err)
	}

	logger.Info("Upload completed successfully!")
	logger.Info("  File ID: %d", metadata.ID)
	logger.Info("  Checksum: %s", metadata.Checksum)

	return nil
}

// runDownload executes the download command
func runDownload(cmd *cobra.Command, args []string) error {
	dbID, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid database ID: %w", err)
	}

	remotePath := args[1]
	localPath := args[2]

	// Ensure remote path starts with /
	if !strings.HasPrefix(remotePath, "/") {
		remotePath = "/" + remotePath
	}

	logger.Info("Downloading file...")
	logger.Info("  Database ID: %d", dbID)
	logger.Info("  Remote: %s", remotePath)
	logger.Info("  Local: %s", localPath)

	// Download binary content with retry
	url := fmt.Sprintf("%s/api/v1/databases/%d/files%s", fsConfig.BaseURL, dbID, remotePath)
	size, contentType, err := downloadBinaryContent(url, localPath)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	logger.Info("  Size: %s", formatBytes(size))
	logger.Info("  Type: %s", contentType)
	logger.Info("Download completed successfully!")

	return nil
}

// runList executes the list command
func runList(cmd *cobra.Command, args []string) error {
	dbID, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid database ID: %w", err)
	}

	path := "/"
	if len(args) > 1 {
		path = args[1]
		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}
	}

	logger.Info("Listing files...")
	logger.Info("  Database ID: %d", dbID)
	if path != "/" {
		logger.Info("  Path: %s", path)
	}

	// Build URL with query parameters
	url := fmt.Sprintf("%s/api/v1/databases/%d/files?path=%s", fsConfig.BaseURL, dbID, url.QueryEscape(path))

	var response APIResponse
	if err := retryRequest("GET", url, "", bytes.Buffer{}, &response); err != nil {
		return err
	}

	if !response.Success {
		return fmt.Errorf("list failed: %s", response.Message)
	}

	// Parse response data
	dataBytes, err := json.Marshal(response.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal response data: %w", err)
	}

	var listResponse FileListResponse
	if err := json.Unmarshal(dataBytes, &listResponse); err != nil {
		return fmt.Errorf("failed to unmarshal list response: %w", err)
	}

	logger.Info("Found %d file(s):\n", listResponse.Total)

	if len(listResponse.Files) == 0 {
		logger.Info("  No files found")
		return nil
	}

	// Print file list
	for _, file := range listResponse.Files {
		fmt.Printf("  %s\n", file.Path)
		fmt.Printf("    Name: %s\n", file.Name)
		fmt.Printf("    Size: %s\n", formatBytes(file.Size))
		fmt.Printf("    Type: %s\n", file.ContentType)
		if file.Checksum != "" {
			fmt.Printf("    Checksum: %s\n", file.Checksum)
		}
		fmt.Printf("    Created: %s\n", file.CreatedAt.Format(time.RFC3339))
		fmt.Println()
	}

	return nil
}

// runRemove executes the remove command
func runRemove(cmd *cobra.Command, args []string) error {
	dbID, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid database ID: %w", err)
	}

	path := args[1]
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	logger.Info("Deleting file...")
	logger.Info("  Database ID: %d", dbID)
	logger.Info("  Path: %s", path)

	// Confirm deletion
	fmt.Print("Are you sure you want to delete this file? (yes/no): ")
	var confirm string
	fmt.Scanln(&confirm)
	if strings.ToLower(confirm) != "yes" {
		logger.Info("Deletion cancelled")
		return nil
	}

	// Make request with retry
	url := fmt.Sprintf("%s/api/v1/databases/%d/files%s", fsConfig.BaseURL, dbID, path)
	var response APIResponse
	if err := retryRequest("DELETE", url, "", bytes.Buffer{}, &response); err != nil {
		return err
	}

	if !response.Success {
		return fmt.Errorf("delete failed: %s", response.Message)
	}

	logger.Info("File deleted successfully")

	return nil
}

// runCopy executes the copy command
func runCopy(cmd *cobra.Command, args []string) error {
	dbID, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid database ID: %w", err)
	}

	srcPath := args[1]
	dstPath := args[2]

	// Ensure paths start with /
	if !strings.HasPrefix(srcPath, "/") {
		srcPath = "/" + srcPath
	}
	if !strings.HasPrefix(dstPath, "/") {
		dstPath = "/" + dstPath
	}

	logger.Info("Copying file...")
	logger.Info("  Database ID: %d", dbID)
	logger.Info("  Source: %s", srcPath)
	logger.Info("  Destination: %s", dstPath)

	// Build request body
	reqBody := map[string]string{
		"src_path": srcPath,
		"dst_path": dstPath,
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Make copy request
	url := fmt.Sprintf("%s/api/v1/databases/%d/files/copy", fsConfig.BaseURL, dbID)
	var response APIResponse
	if err := retryRequest("POST", url, "application/json", *bytes.NewBuffer(bodyBytes), &response); err != nil {
		return err
	}

	if !response.Success {
		return fmt.Errorf("copy failed: %s", response.Message)
	}

	// Parse response data
	dataBytes, err := json.Marshal(response.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal response data: %w", err)
	}

	var metadata FileMetadata
	if err := json.Unmarshal(dataBytes, &metadata); err != nil {
		return fmt.Errorf("failed to unmarshal file metadata: %w", err)
	}

	logger.Info("Copy completed successfully!")
	logger.Info("  File ID: %d", metadata.ID)
	logger.Info("  Size: %s", formatBytes(metadata.Size))

	return nil
}

// retryRequest performs an HTTP request with retry logic
func retryRequest(method, url, contentType string, body bytes.Buffer, result interface{}) error {
	var lastErr error

	for attempt := 0; attempt < fsConfig.MaxRetries; attempt++ {
		if attempt > 0 {
			logger.Info("Retry attempt %d/%d...", attempt, fsConfig.MaxRetries)
			time.Sleep(time.Second * time.Duration(attempt))
		}

		// Create request
		var req *http.Request
		var err error

		if body.Len() > 0 {
			req, err = http.NewRequest(method, url, &body)
			if err != nil {
				return fmt.Errorf("failed to create request: %w", err)
			}
			req.Header.Set("Content-Type", contentType)
		} else {
			req, err = http.NewRequest(method, url, nil)
			if err != nil {
				return fmt.Errorf("failed to create request: %w", err)
			}
		}

		// Set timeout
		client := &http.Client{
			Timeout: fsConfig.Timeout,
		}

		// Send request
		resp, err := client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			if logger.IsVerbose() {
				logger.Debug("Request error: %v", err)
			}
			continue
		}

		defer resp.Body.Close()

		// Read response
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = fmt.Errorf("failed to read response: %w", err)
			continue
		}

		// Check status code
		if resp.StatusCode >= 400 {
			lastErr = fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
			if logger.IsVerbose() {
				logger.Debug("API error response: %s", string(respBody))
			}
			continue
		}

		// Parse response
		if err := json.Unmarshal(respBody, result); err != nil {
			lastErr = fmt.Errorf("failed to parse response: %w", err)
			if logger.IsVerbose() {
				logger.Debug("Parse error: %v, Response: %s", err, string(respBody))
			}
			continue
		}

		// Success
		return nil
	}

	return fmt.Errorf("max retries exceeded: %w", lastErr)
}

// progressWriter tracks upload/download progress
type progressWriter struct {
	Writer  io.Writer
	Total   int64
	Written int64
	Name    string
	Verbose bool
	lastUpdate time.Time
}

func (pw *progressWriter) Write(p []byte) (int, error) {
	n, err := pw.Writer.Write(p)
	if err != nil {
		return n, err
	}

	pw.Written += int64(n)

	// Update progress at most once per second
	if pw.Verbose && time.Since(pw.lastUpdate) > time.Second {
		percent := float64(pw.Written) / float64(pw.Total) * 100
		logger.Info("%s: %s / %s (%.1f%%)", pw.Name, formatBytes(pw.Written), formatBytes(pw.Total), percent)
		pw.lastUpdate = time.Now()
	}

	return n, nil
}

// formatBytes formats a byte size into human-readable format
func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}

// downloadBinaryContent downloads binary file content with retry logic
func downloadBinaryContent(url, localPath string) (int64, string, error) {
	var lastErr error

	for attempt := 0; attempt < fsConfig.MaxRetries; attempt++ {
		if attempt > 0 {
			logger.Info("Retry attempt %d/%d...", attempt, fsConfig.MaxRetries)
			time.Sleep(time.Second * time.Duration(attempt))
		}

		// Create request
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return 0, "", fmt.Errorf("failed to create request: %w", err)
		}

		// Set timeout
		client := &http.Client{
			Timeout: fsConfig.Timeout,
		}

		// Send request
		resp, err := client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			if logger.IsVerbose() {
				logger.Debug("Request error: %v", err)
			}
			continue
		}

		// Check status code
		if resp.StatusCode >= 400 {
			lastErr = fmt.Errorf("API error (status %d)", resp.StatusCode)
			resp.Body.Close()
			if logger.IsVerbose() {
				logger.Debug("API error status: %d", resp.StatusCode)
			}
			continue
		}

		// Get content length from headers
		var size int64
		if contentLength := resp.Header.Get("Content-Length"); contentLength != "" {
			if sizeInt, err := strconv.ParseInt(contentLength, 10, 64); err == nil {
				size = sizeInt
			}
		}

		contentType := resp.Header.Get("Content-Type")
		if contentType == "" {
			contentType = "application/octet-stream"
		}

		// Create output file
		outFile, err := os.Create(localPath)
		if err != nil {
			resp.Body.Close()
			return 0, "", fmt.Errorf("failed to create output file: %w", err)
		}

		// Copy with progress tracking
		progressWriter := &progressWriter{
			Writer:  outFile,
			Total:   size,
			Name:    "Downloading",
			Verbose: logger.IsVerbose(),
		}

		written, err := io.Copy(progressWriter, resp.Body)
		resp.Body.Close()
		outFile.Close()

		if err != nil {
			lastErr = fmt.Errorf("failed to download content: %w", err)
			os.Remove(localPath)
			continue
		}

		return written, contentType, nil
	}

	return 0, "", fmt.Errorf("max retries exceeded: %w", lastErr)
}

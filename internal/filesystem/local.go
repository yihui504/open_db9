package filesystem

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// LocalStorage implements Storage interface using local filesystem
type LocalStorage struct {
	baseDir       string
	maxFileSize   int64
	createDirPerm os.FileMode
	filePerm      os.FileMode
}

// LocalStorageConfig holds configuration for LocalStorage
type LocalStorageConfig struct {
	BaseDir     string   // Base directory for storage
	MaxFileSize int64    // Maximum file size in bytes (0 = no limit)
	CreateDirPerm os.FileMode // Permission for created directories
	FilePerm      os.FileMode // Permission for created files
}

// NewLocalStorage creates a new LocalStorage instance
func NewLocalStorage(config LocalStorageConfig) (*LocalStorage, error) {
	if config.BaseDir == "" {
		return nil, fmt.Errorf("base directory cannot be empty")
	}

	// Set default permissions if not specified
	if config.CreateDirPerm == 0 {
		config.CreateDirPerm = 0755
	}
	if config.FilePerm == 0 {
		config.FilePerm = 0644
	}

	// Ensure base directory exists
	if err := os.MkdirAll(config.BaseDir, config.CreateDirPerm); err != nil {
		return nil, fmt.Errorf("failed to create base directory: %w", err)
	}

	return &LocalStorage{
		baseDir:       config.BaseDir,
		maxFileSize:   config.MaxFileSize,
		createDirPerm: config.CreateDirPerm,
		filePerm:      config.FilePerm,
	}, nil
}

// Put stores a file with the given key and metadata
func (ls *LocalStorage) Put(ctx context.Context, key string, data io.Reader, metadata Metadata) error {
	// Validate key
	if err := ls.validateKey(key); err != nil {
		return err
	}

	// Build full file path
	filePath := ls.filePath(key)

	// Create directory structure if needed
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, ls.createDirPerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Create temporary file for atomic write
	tempPath := filePath + ".tmp"
	file, err := os.OpenFile(tempPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, ls.filePerm)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}

	// Ensure file is closed on error
	defer func() {
		file.Close()
		if err != nil {
			os.Remove(tempPath)
		}
	}()

	// Track bytes written for size limit
	var written int64
	buf := make([]byte, 32*1024) // 32KB buffer

	for {
		n, err := data.Read(buf)
		if n > 0 {
			// Check file size limit
			if ls.maxFileSize > 0 && (written+int64(n)) > ls.maxFileSize {
				return fmt.Errorf("file size exceeds maximum allowed size of %d bytes", ls.maxFileSize)
			}

			if _, writeErr := file.Write(buf[:n]); writeErr != nil {
				return fmt.Errorf("failed to write data: %w", writeErr)
			}
			written += int64(n)
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read data: %w", err)
		}

		// Check for context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}

	// Sync to disk
	if err := file.Sync(); err != nil {
		return fmt.Errorf("failed to sync file: %w", err)
	}

	// Close file
	if err := file.Close(); err != nil {
		return fmt.Errorf("failed to close file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempPath, filePath); err != nil {
		return fmt.Errorf("failed to move file: %w", err)
	}

	return nil
}

// Get retrieves a file by key
func (ls *LocalStorage) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	if err := ls.validateKey(key); err != nil {
		return nil, err
	}

	filePath := ls.filePath(key)

	// Check for context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %s", key)
		}
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	return file, nil
}

// Delete removes a file by key
func (ls *LocalStorage) Delete(ctx context.Context, key string) error {
	if err := ls.validateKey(key); err != nil {
		return err
	}

	filePath := ls.filePath(key)

	// Check for context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	err := os.Remove(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file not found: %s", key)
		}
		return fmt.Errorf("failed to delete file: %w", err)
	}

	// Clean up empty directories
	if err := ls.cleanEmptyDirs(filepath.Dir(filePath)); err != nil {
		// Log error but don't fail the delete operation
		fmt.Printf("warning: failed to clean empty directories: %v\n", err)
	}

	return nil
}

// List returns all files with the given prefix
func (ls *LocalStorage) List(ctx context.Context, prefix string) ([]FileEntry, error) {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	dirPath := ls.baseDir
	if prefix != "" {
		dirPath = ls.filePath(prefix)
	}

	var entries []FileEntry

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Check for context cancellation during walk
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Convert path back to key
		key, err := ls.keyFromPath(path)
		if err != nil {
			return err
		}

		// Filter by prefix if specified
		if prefix != "" && !strings.HasPrefix(key, prefix) {
			return nil
		}

		entries = append(entries, FileEntry{
			Key:         key,
			Size:        info.Size(),
			ModTime:     info.ModTime(),
			ContentType: "", // Content type not stored in filesystem
		})

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	return entries, nil
}

// Exists checks if a file exists
func (ls *LocalStorage) Exists(ctx context.Context, key string) (bool, error) {
	if err := ls.validateKey(key); err != nil {
		return false, err
	}

	filePath := ls.filePath(key)

	// Check for context cancellation
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	default:
	}

	_, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check file existence: %w", err)
	}

	return true, nil
}

// Size returns the size of a file
func (ls *LocalStorage) Size(ctx context.Context, key string) (int64, error) {
	if err := ls.validateKey(key); err != nil {
		return 0, err
	}

	filePath := ls.filePath(key)

	// Check for context cancellation
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	default:
	}

	info, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, fmt.Errorf("file not found: %s", key)
		}
		return 0, fmt.Errorf("failed to get file info: %w", err)
	}

	return info.Size(), nil
}

// validateKey validates the storage key
func (ls *LocalStorage) validateKey(key string) error {
	if key == "" {
		return fmt.Errorf("key cannot be empty")
	}

	// Prevent path traversal attacks
	if strings.Contains(key, "..") {
		return fmt.Errorf("key cannot contain path traversal sequences")
	}

	// Ensure key uses forward slashes
	key = filepath.ToSlash(key)

	// Check for absolute paths
	if strings.HasPrefix(key, "/") {
		return fmt.Errorf("key cannot be an absolute path")
	}

	return nil
}

// filePath converts a storage key to a full filesystem path
func (ls *LocalStorage) filePath(key string) string {
	// Normalize key to use forward slashes
	key = filepath.ToSlash(key)
	return filepath.Join(ls.baseDir, filepath.FromSlash(key))
}

// keyFromPath converts a filesystem path back to a storage key
func (ls *LocalStorage) keyFromPath(path string) (string, error) {
	absBase, err := filepath.Abs(ls.baseDir)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute base path: %w", err)
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute file path: %w", err)
	}

	relPath, err := filepath.Rel(absBase, absPath)
	if err != nil {
		return "", fmt.Errorf("failed to get relative path: %w", err)
	}

	// Convert back to forward slashes for storage keys
	return filepath.ToSlash(relPath), nil
}

// cleanEmptyDirs removes empty directories starting from the given path
func (ls *LocalStorage) cleanEmptyDirs(dir string) error {
	// Don't delete the base directory
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return err
	}

	absBase, err := filepath.Abs(ls.baseDir)
	if err != nil {
		return err
	}

	if absDir == absBase {
		return nil
	}

	// Check if directory is empty
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	if len(entries) == 0 {
		// Remove empty directory
		if err := os.Remove(dir); err != nil {
			return err
		}

		// Recursively clean parent directories
		parent := filepath.Dir(dir)
		return ls.cleanEmptyDirs(parent)
	}

	return nil
}

package filesystem

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// DiskStorage implements Storage interface using local filesystem
type DiskStorage struct {
	basePath    string
	maxFileSize int64
}

// NewDiskStorage creates a new DiskStorage instance
func NewDiskStorage(basePath string) (*DiskStorage, error) {
	// Create base directory if it doesn't exist
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base directory: %w", err)
	}
	return &DiskStorage{
		basePath:    basePath,
		maxFileSize: 100 * 1024 * 1024, // 100MB
	}, nil
}

// Put stores a file with the given key and metadata
func (d *DiskStorage) Put(ctx context.Context, key string, data io.Reader, metadata Metadata) error {
	// Validate key
	if err := d.validateKey(key); err != nil {
		return err
	}

	fullPath := filepath.Join(d.basePath, key)

	// Create directory structure if needed
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Create file
	file, err := os.Create(fullPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Copy data with size limit
	limitedReader := io.LimitReader(data, d.maxFileSize)
	written, err := io.Copy(file, limitedReader)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	// Check if file was too large
	if written == d.maxFileSize {
		// Remove the partial file
		os.Remove(fullPath)
		return fmt.Errorf("file size exceeds maximum allowed size of %d bytes", d.maxFileSize)
	}

	return nil
}

// Get retrieves a file by key
func (d *DiskStorage) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	// Validate key
	if err := d.validateKey(key); err != nil {
		return nil, err
	}

	fullPath := filepath.Join(d.basePath, key)
	file, err := os.Open(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %s", key)
		}
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	return file, nil
}

// Delete removes a file by key
func (d *DiskStorage) Delete(ctx context.Context, key string) error {
	// Validate key
	if err := d.validateKey(key); err != nil {
		return err
	}

	fullPath := filepath.Join(d.basePath, key)
	if err := os.Remove(fullPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file not found: %s", key)
		}
		return fmt.Errorf("failed to delete file: %w", err)
	}
	return nil
}

// List returns all files with the given prefix
func (d *DiskStorage) List(ctx context.Context, prefix string) ([]FileEntry, error) {
	// Validate prefix
	if err := d.validateKey(prefix); err != nil {
		return nil, err
	}

	searchPath := filepath.Join(d.basePath, prefix)
	var entries []FileEntry

	err := filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		// Get relative path from base path
		relPath, err := filepath.Rel(d.basePath, path)
		if err != nil {
			return err
		}

		// Use forward slashes for keys
		key := filepath.ToSlash(relPath)

		entries = append(entries, FileEntry{
			Key:         key,
			Size:        info.Size(),
			ModTime:     info.ModTime(),
			ContentType: detectContentType(path),
		})

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	return entries, nil
}

// Exists checks if a file exists
func (d *DiskStorage) Exists(ctx context.Context, key string) (bool, error) {
	// Validate key
	if err := d.validateKey(key); err != nil {
		return false, err
	}

	fullPath := filepath.Join(d.basePath, key)
	_, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check file existence: %w", err)
	}
	return true, nil
}

// Size returns the size of a file
func (d *DiskStorage) Size(ctx context.Context, key string) (int64, error) {
	// Validate key
	if err := d.validateKey(key); err != nil {
		return 0, err
	}

	fullPath := filepath.Join(d.basePath, key)
	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, fmt.Errorf("file not found: %s", key)
		}
		return 0, fmt.Errorf("failed to get file info: %w", err)
	}
	return info.Size(), nil
}

func detectContentType(path string) string {
	ext := filepath.Ext(path)
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
	default:
		return "application/octet-stream"
	}
}

// validateKey validates the storage key to prevent path traversal attacks
func (d *DiskStorage) validateKey(key string) error {
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

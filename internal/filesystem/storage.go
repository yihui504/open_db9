package filesystem

import (
	"context"
	"io"
	"time"
)

// Metadata holds file metadata information
type Metadata struct {
	ContentType string
	Size        int64
	ModTime     time.Time
	Custom      map[string]string
}

// FileEntry represents a file entry in storage
type FileEntry struct {
	Key         string
	Size        int64
	ModTime     time.Time
	ContentType string
}

// Storage defines the interface for file storage operations
type Storage interface {
	// Put stores a file with the given key and metadata
	Put(ctx context.Context, key string, data io.Reader, metadata Metadata) error

	// Get retrieves a file by key
	Get(ctx context.Context, key string) (io.ReadCloser, error)

	// Delete removes a file by key
	Delete(ctx context.Context, key string) error

	// List returns all files with the given prefix
	List(ctx context.Context, prefix string) ([]FileEntry, error)

	// Exists checks if a file exists
	Exists(ctx context.Context, key string) (bool, error)

	// Size returns the size of a file
	Size(ctx context.Context, key string) (int64, error)
}

package models

import (
	"encoding/json"
	"time"
)

// Note: User-related types have been moved to user.go

// AnonymousRegisterRequest represents the request body for anonymous registration
type AnonymousRegisterRequest struct {
	SessionID string `json:"session_id"`
}

// AnonymousRegisterResponse represents the response for anonymous registration
type AnonymousRegisterResponse struct {
	Token        json.RawMessage `json:"token"`
	TenantID     string          `json:"tenant_id"`
	SessionID    string          `json:"session_id"`
	Capabilities json.RawMessage `json:"capabilities"`
	ExpiresAt    time.Time       `json:"expires_at"`
}

// ClaimAccountRequest represents the request body for claiming an anonymous account
type ClaimAccountRequest struct {
	SessionID string `json:"session_id"`
}

// ClaimAccountResponse represents the response for claiming an anonymous account
type ClaimAccountResponse struct {
	Token            json.RawMessage `json:"token"`
	Message          string          `json:"message"`
	PreviousTenantID string          `json:"previous_tenant_id"`
}

// Database represents a database connection configuration
type Database struct {
	ID         int       `json:"id"`
	Name       string    `json:"name"`
	Type       string    `json:"type"`
	Host       string    `json:"host"`
	Port       int       `json:"port"`
	User       string    `json:"user"`
	Database   string    `json:"database"`
	ParentID   *int      `json:"parent_id,omitempty"`   // ID of parent database if this is a branch
	SnapshotID *int      `json:"snapshot_id,omitempty"` // ID of snapshot used to create this branch
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// Query represents a database query
type Query struct {
	ID          int       `json:"id"`
	DatabaseID  int       `json:"database_id"`
	Name        string    `json:"name"`
	Query       string    `json:"query"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Result represents a query result
type Result struct {
	ID       int       `json:"id"`
	QueryID  int       `json:"query_id"`
	Rows     int       `json:"rows"`
	Time     float64   `json:"time"`
	Success  bool      `json:"success"`
	Error    string    `json:"error,omitempty"`
	Data     []byte    `json:"data,omitempty"`
	CreateAt time.Time `json:"created_at"`
}

// File represents a file in the system
type File struct {
	ID          int       `json:"id"`
	DatabaseID  int       `json:"database_id"`
	Path        string    `json:"path"`
	Name        string    `json:"name"`
	Size        int64     `json:"size"`
	ContentType string    `json:"content_type"`
	Checksum    string    `json:"checksum,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Branch represents a database branch
type Branch struct {
	ID         int       `json:"id"`
	BranchID   string    `json:"branch_id"`   // Unique identifier for the branch
	BranchName string    `json:"branch_name"` // Human-readable name
	ParentID   string    `json:"parent_database_id"`
	SnapshotID string    `json:"snapshot_id"`
	CreatedAt  time.Time `json:"created_at"`
	Status     string    `json:"status"` // active, deleted, etc.
}

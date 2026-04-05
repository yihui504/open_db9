package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/open-db9/db9/internal/database"
)

// ExtensionListResponse represents the response for listing extensions
type ExtensionListResponse struct {
	Extensions []database.Extension `json:"extensions"`
}

// ExtensionInstallRequest represents the request to install an extension
type ExtensionInstallRequest struct {
	Name string `json:"name"`
}

// ExtensionVerifyRequest represents the request to verify an extension
type ExtensionVerifyRequest struct {
	Name string `json:"name"`
}

// ExtensionVerifyResponse represents the response for verifying an extension
type ExtensionVerifyResponse struct {
	Installed bool   `json:"installed"`
	Version   string `json:"version,omitempty"`
	Message   string `json:"message,omitempty"`
}

// extractDatabaseID extracts database ID from URL path
// Pattern should be like "/api/v1/databases/%s/..."
func extractDatabaseID(r *http.Request, prefix string) (string, error) {
	path := r.URL.Path
	if !strings.HasPrefix(path, prefix) {
		return "", fmt.Errorf("invalid URL path")
	}

	rest := strings.TrimPrefix(path, prefix)
	parts := strings.Split(rest, "/")
	if len(parts) == 0 || parts[0] == "" {
		return "", fmt.Errorf("missing database ID")
	}

	return parts[0], nil
}

// ListExtensionsHandler handles GET /api/v1/databases/:id/extensions
func ListExtensionsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract database ID from URL path
	dbIDStr, err := extractDatabaseID(r, "/api/v1/databases/")
	if err != nil {
		http.Error(w, "Invalid URL format. Expected: /api/v1/databases/:id/extensions", http.StatusBadRequest)
		return
	}

	// Parse database ID
	dbID, err := strconv.Atoi(dbIDStr)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid database ID: %s", dbIDStr), http.StatusBadRequest)
		return
	}

	// Get database manager
	manager, err := GetDatabase(dbID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Get extensions manager
	extManager := database.NewExtensionsManager(manager)

	// List extensions
	ctx := r.Context()
	extensions, err := extManager.ListExtensions(ctx, dbIDStr)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list extensions: %v", err), http.StatusInternalServerError)
		return
	}

	// Build response
	response := Response{
		Success: true,
		Data: ExtensionListResponse{
			Extensions: extensions,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
		return
	}
}

// InstallExtensionHandler handles POST /api/v1/databases/:id/extensions/install
func InstallExtensionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract database ID from URL path
	dbIDStr, err := extractDatabaseID(r, "/api/v1/databases/")
	if err != nil {
		http.Error(w, "Invalid URL format. Expected: /api/v1/databases/:id/extensions/install", http.StatusBadRequest)
		return
	}

	// Parse database ID
	dbID, err := strconv.Atoi(dbIDStr)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid database ID: %s", dbIDStr), http.StatusBadRequest)
		return
	}

	// Get database manager
	manager, err := GetDatabase(dbID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Validate Content-Type
	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "Content-Type must be application/json", http.StatusUnsupportedMediaType)
		return
	}

	// Parse request body
	var req ExtensionInstallRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Validate extension name
	if req.Name == "" {
		http.Error(w, "Extension name is required", http.StatusBadRequest)
		return
	}

	// Get extensions manager
	extManager := database.NewExtensionsManager(manager)

	// Install extension
	ctx := r.Context()
	if err := extManager.InstallExtension(ctx, dbIDStr, req.Name); err != nil {
		// Check if it's a timeout error
		if errors.Is(err, context.DeadlineExceeded) {
			http.Error(w, "Extension installation timeout", http.StatusRequestTimeout)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to install extension: %v", err), http.StatusInternalServerError)
		return
	}

	// Build response
	response := Response{
		Success: true,
		Data: map[string]string{
			"message": fmt.Sprintf("Extension '%s' installed successfully", req.Name),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
		return
	}
}

// VerifyExtensionHandler handles POST /api/v1/databases/:id/extensions/verify
func VerifyExtensionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract database ID from URL path
	dbIDStr, err := extractDatabaseID(r, "/api/v1/databases/")
	if err != nil {
		http.Error(w, "Invalid URL format. Expected: /api/v1/databases/:id/extensions/verify", http.StatusBadRequest)
		return
	}

	// Parse database ID
	dbID, err := strconv.Atoi(dbIDStr)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid database ID: %s", dbIDStr), http.StatusBadRequest)
		return
	}

	// Get database manager
	manager, err := GetDatabase(dbID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Validate Content-Type
	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "Content-Type must be application/json", http.StatusUnsupportedMediaType)
		return
	}

	// Parse request body
	var req ExtensionVerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Validate extension name
	if req.Name == "" {
		http.Error(w, "Extension name is required", http.StatusBadRequest)
		return
	}

	// Get extensions manager
	extManager := database.NewExtensionsManager(manager)

	// Verify extension
	ctx := r.Context()
	installed, err := extManager.VerifyExtension(ctx, dbIDStr, req.Name)
	if err != nil {
		// Check if it's a timeout error
		if errors.Is(err, context.DeadlineExceeded) {
			http.Error(w, "Extension verification timeout", http.StatusRequestTimeout)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to verify extension: %v", err), http.StatusInternalServerError)
		return
	}

	// Get version info if installed
	version := ""
	if installed {
		if info, err := extManager.GetInfo(ctx, req.Name); err == nil {
			version = info.Version
		}
	}

	// Build message
	message := fmt.Sprintf("Extension '%s' is %s", req.Name, map[bool]string{true: "installed", false: "not installed"}[installed])

	// Build response
	response := Response{
		Success: true,
		Data: ExtensionVerifyResponse{
			Installed: installed,
			Version:   version,
			Message:   message,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
		return
	}
}

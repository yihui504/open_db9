package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	httpext "github.com/open-db9/db9/internal/extensions/http"
)

// HttpExtensionHandler handles HTTP extension endpoints
type HttpExtensionHandler struct {
	client *httpext.Client
}

// NewHttpExtensionHandler creates a new HTTP extension handler
func NewHttpExtensionHandler(client *httpext.Client) *HttpExtensionHandler {
	return &HttpExtensionHandler{
		client: client,
	}
}

// GetRequest handles POST /api/v1/databases/:id/http/get
func (h *HttpExtensionHandler) GetRequest(w http.ResponseWriter, r *http.Request) {
	// Validate database ID
	path := strings.TrimSuffix(r.URL.Path, "/")
	parts := strings.Split(path, "/")
	if len(parts) < 7 {
		http.Error(w, "Invalid URL format", http.StatusBadRequest)
		return
	}
	dbIDStr := parts[4]
	if dbIDStr == "" {
		http.Error(w, "Database ID is required", http.StatusBadRequest)
		return
	}

	dbID, err := strconv.Atoi(dbIDStr)
	if err != nil || dbID <= 0 || dbID > 1000000 {
		http.Error(w, "Invalid database ID", http.StatusBadRequest)
		return
	}

	// Parse request body
	var requestBody struct {
		URL     string            `json:"url"`
		Headers map[string]string `json:"headers"`
	}

	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		http.Error(w, "Invalid request body: must be valid JSON", http.StatusBadRequest)
		return
	}

	// Validate URL
	if requestBody.URL == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
		return
	}

	// Initialize headers if nil
	if requestBody.Headers == nil {
		requestBody.Headers = make(map[string]string)
	}

	// Execute GET request
	response, err := h.client.DoGet(r.Context(), requestBody.URL, requestBody.Headers)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// PostRequest handles POST /api/v1/databases/:id/http/post
func (h *HttpExtensionHandler) PostRequest(w http.ResponseWriter, r *http.Request) {
	// Validate database ID
	path := strings.TrimSuffix(r.URL.Path, "/")
	parts := strings.Split(path, "/")
	if len(parts) < 7 {
		http.Error(w, "Invalid URL format", http.StatusBadRequest)
		return
	}
	dbIDStr := parts[4]
	if dbIDStr == "" {
		http.Error(w, "Database ID is required", http.StatusBadRequest)
		return
	}

	dbID, err := strconv.Atoi(dbIDStr)
	if err != nil || dbID <= 0 || dbID > 1000000 {
		http.Error(w, "Invalid database ID", http.StatusBadRequest)
		return
	}

	// Parse request body
	var requestBody struct {
		URL         string            `json:"url"`
		Body        string            `json:"body"`
		ContentType string            `json:"content_type"`
		Headers     map[string]string `json:"headers"`
	}

	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		http.Error(w, "Invalid request body: must be valid JSON", http.StatusBadRequest)
		return
	}

	// Validate URL
	if requestBody.URL == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
		return
	}

	// Initialize headers if nil
	if requestBody.Headers == nil {
		requestBody.Headers = make(map[string]string)
	}

	// Execute POST request
	response, err := h.client.DoPost(
		r.Context(),
		requestBody.URL,
		requestBody.Body,
		requestBody.ContentType,
		requestBody.Headers,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

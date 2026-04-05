package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	mcpserver "github.com/mark3labs/mcp-go/server"
)

// Version information (can be set at build time)
var (
	Version   = "0.1.0"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

func main() {
	// Load configuration from environment variables
	apiURL := getEnv("DB9_API_URL", "http://localhost:8080")
	apiToken := getEnv("DB9_API_TOKEN", "")
	ragURL := getEnv("DB9_RAG_URL", "http://localhost:8001")

	// Validate required configuration
	if apiToken == "" {
		log.Fatal("Error: DB9_API_TOKEN environment variable is required")
	}

	// Print version and configuration information
	log.Printf("=== Open-DB9 MCP Server ===")
	log.Printf("Version: %s", Version)
	log.Printf("Build Time: %s", BuildTime)
	log.Printf("Git Commit: %s", GitCommit)
	log.Printf("")
	log.Printf("Configuration:")
	log.Printf("  API URL:  %s", apiURL)
	log.Printf("  RAG URL:  %s", ragURL)
	log.Printf("  Token:    %s***", apiToken[:min(4, len(apiToken))])
	log.Printf("")

	// Initialize API client
	client = NewAPIClient(apiURL, apiToken, ragURL)

	// Create MCP server
	s := mcpserver.NewMCPServer(
		"open-db9",
		Version,
		mcpserver.WithToolCapabilities(true),
	)

	// Register all tools
	registerTools(s)

	// Handle interrupt signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start the server using stdio transport
	log.Println("MCP Server starting on stdio...")
	if err := mcpserver.ServeStdio(s); err != nil && err != io.EOF {
		log.Printf("Server error: %v", err)
		os.Exit(1)
	}

	log.Println("MCP Server shutdown complete")
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// APIResponse represents a standard API response from DB9 API Server
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// APIClient is a client for communicating with the DB9 API Server
type APIClient struct {
	baseURL    string
	token      string
	ragURL     string
	httpClient *http.Client
}

var client *APIClient

// NewAPIClient creates a new API client
func NewAPIClient(baseURL, token, ragURL string) *APIClient {
	return &APIClient{
		baseURL: baseURL,
		token:   token,
		ragURL:  ragURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Do performs an HTTP request to the API server
func (c *APIClient) Do(method, path string, body interface{}) (*APIResponse, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = strings.NewReader(string(jsonBody))
	}

	url := c.baseURL + path
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var apiResp APIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &apiResp, nil
}

// DoRAG performs an HTTP request to the RAG server
func (c *APIClient) DoRAG(method, path string, body interface{}) (*APIResponse, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = strings.NewReader(string(jsonBody))
	}

	url := c.ragURL + path
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var apiResp APIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &apiResp, nil
}

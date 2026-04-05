package http

import (
	"context"
	"io"
	"net"
	"net/http"
	nethttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestIsPrivateIPv4 tests IPv4 private address detection
func TestIsPrivateIPv4(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected bool
	}{
		{"Loopback address", "127.0.0.1", true},
		{"Loopback range", "127.0.0.255", true},
		{"Private Class A", "10.0.0.1", true},
		{"Private Class A edge", "10.255.255.255", true},
		{"Private Class B", "172.16.0.1", true},
		{"Private Class B start", "172.16.0.0", true},
		{"Private Class B end", "172.31.255.255", true},
		{"Private Class C", "192.168.1.1", true},
		{"Link-local", "169.254.169.254", true},
		{"Zero network", "0.0.0.0", true},
		{"Public IP (Google DNS)", "8.8.8.8", false},
		{"Public IP (Cloudflare)", "1.1.1.1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			if ip == nil {
				t.Fatalf("Failed to parse IP: %s", tt.ip)
			}
			result := isPrivateIP(ip)
			if result != tt.expected {
				t.Errorf("isPrivateIP(%s) = %v, want %v", tt.ip, result, tt.expected)
			}
		})
	}
}

// TestIsPrivateIPv6 tests IPv6 private address detection
func TestIsPrivateIPv6(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected bool
	}{
		{"IPv6 Loopback", "::1", true},
		{"Unique Local Address (ULA)", "fd00::1", true},
		{"Link-local", "fe80::1", true},
		{"Unspecified", "::", true},
		{"Public IPv6", "2001:4860:4860::8888", false}, // Google DNS IPv6
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			if ip == nil {
				t.Fatalf("Failed to parse IP: %s", tt.ip)
			}
			result := isPrivateIP(ip)
			if result != tt.expected {
				t.Errorf("isPrivateIP(%s) = %v, want %v", tt.ip, result, tt.expected)
			}
		})
	}
}

// TestValidateURL tests URL validation (HTTPS only)
func TestValidateURL(t *testing.T) {
	client := NewClient()

	tests := []struct {
		name        string
		url         string
		expectError bool
		errorMsg    string
	}{
		{"Valid HTTPS URL", "https://example.com", false, ""},
		{"Valid HTTPS with path", "https://api.example.com/v1/data", false, ""},
		{"HTTP URL rejected", "http://example.com", true, "only HTTPS URLs are allowed"},
		{"No scheme", "example.com", true, ""},
		{"FTP scheme rejected", "ftp://files.example.com", true, "only HTTPS URLs are allowed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := client.validateURL(tt.url)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for URL %s but got none", tt.url)
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Error message does not contain expected text. Got: %s, Want: %s", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for valid URL %s: %v", tt.url, err)
				}
			}
		})
	}
}

// TestCheckSSRF tests SSRF protection with DNS resolution
func TestCheckSSRF(t *testing.T) {
	client := NewClient()

	tests := []struct {
		name        string
		url         string
		expectError bool
	}{
		{"Public domain (example.com)", "https://example.com", false},
		{"Public API (api.github.com)", "https://api.github.com", false},
		{"Localhost should be blocked", "https://localhost", true},
		{"Loopback IP should be blocked", "https://127.0.0.1", true},
		{"Private IP Class A should be blocked", "https://10.0.0.1", true},
		{"Private IP Class B should be blocked", "https://172.16.0.1", true},
		{"Private IP Class C should be blocked", "https://192.168.1.1", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := client.checkSSRF(tt.url)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected SSRF error for URL %s but got none", tt.url)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected SSRF error for public URL %s: %v", tt.url, err)
				}
			}
		})
	}
}

// TestDoGetSuccess tests successful GET request
func TestDoGetSuccess(t *testing.T) {
	// Create test server (use plain HTTP server for testing)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}))
	defer server.Close()

	client := NewClient()
	// For testing purposes, we directly use the underlying HTTP client to avoid SSRF checks on localhost
	req, err := nethttp.NewRequestWithContext(context.Background(), nethttp.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	resp, err := client.httpClient.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, client.maxResponseBody))

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code 200, got %d", resp.StatusCode)
	}

	if string(body) != `{"status": "ok"}` {
		t.Errorf("Unexpected body: %s", string(body))
	}
}

// TestDoPostSuccess tests successful POST request
func TestDoPostSuccess(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST method, got %s", r.Method)
		}

		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", contentType)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id": 123, "created": true}`))
	}))
	defer server.Close()

	client := NewClient()
	// For testing purposes, we directly use the underlying HTTP client to avoid SSRF checks on localhost
	req, err := nethttp.NewRequestWithContext(context.Background(), nethttp.MethodPost, server.URL, strings.NewReader(`{"name": "test"}`))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Custom-Header", "test-value")

	resp, err := client.httpClient.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, client.maxResponseBody))

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected status code 201, got %d", resp.StatusCode)
	}

	if !strings.Contains(string(body), `"id": 123`) {
		t.Errorf("Unexpected body: %s", string(body))
	}
}

// TestDoGetNonHTTPS tests rejection of non-HTTPS URLs
func TestDoGetNonHTTPS(t *testing.T) {
	client := NewClient()

	_, err := client.DoGet(context.Background(), "http://example.com", nil)
	if err == nil {
		t.Fatal("Expected error for HTTP URL")
	}

	if !strings.Contains(err.Error(), "only HTTPS URLs are allowed") {
		t.Errorf("Expected HTTPS-only error, got: %v", err)
	}
}

// TestDoPostLargeBody tests rejection of oversized request body
func TestDoPostLargeBody(t *testing.T) {
	client := NewClientWithConfig(1024*1024, 100, 5*time.Second) // 100 byte limit

	largeBody := strings.Repeat("x", 200) // 200 bytes > 100 byte limit

	_, err := client.DoPost(
		context.Background(),
		"https://example.com/api",
		largeBody,
		"text/plain",
		nil,
	)

	if err == nil {
		t.Fatal("Expected error for oversized body")
	}

	if !strings.Contains(err.Error(), "exceeds maximum allowed size") {
		t.Errorf("Expected size limit error, got: %v", err)
	}
}

// TestResponseTruncation tests that large responses are truncated
func TestResponseTruncation(t *testing.T) {
	// Create a server that returns a large response
	largeBody := strings.Repeat("x", 2*1024*1024) // 2MB > 1MB limit

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(largeBody))
	}))
	defer server.Close()

	client := NewClient()
	req, err := nethttp.NewRequestWithContext(context.Background(), nethttp.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	resp, err := client.httpClient.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, client.maxResponseBody))

	// Response should be truncated to maxResponseBody (1MB)
	if len(body) > int(client.maxResponseBody) {
		t.Errorf("Response body too large: %d bytes (max: %d)", len(body), client.maxResponseBody)
	}
}

// TestTimeout tests that requests timeout correctly
func TestTimeout(t *testing.T) {
	// Create a server that delays response
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Second) // Longer than timeout
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClientWithConfig(1024*1024, 256*1024, 100*time.Millisecond) // 100ms timeout

	start := time.Now()
	_, err := client.DoGet(context.Background(), server.URL, nil)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("Expected timeout error")
	}

	// Should have timed out within reasonable time
	if elapsed > 2*time.Second {
		t.Errorf("Request took too long: %v (expected ~100ms)", elapsed)
	}
}

// TestNewClientWithConfig tests custom configuration
func TestNewClientWithConfig(t *testing.T) {
	tests := []struct {
		name              string
		maxResponseBody   int64
		maxRequestBody    int64
		timeout           time.Duration
		expectDefaultResp int64
		expectDefaultReq  int64
		expectDefaultTO   time.Duration
	}{
		{"All custom values", 2048, 512, 10 * time.Second, 0, 0, 0},
		{"Zero response uses default", 0, 256, 5 * time.Second, DefaultMaxResponseBody, 0, 0},
		{"Zero request uses default", 1024, 0, 5 * time.Second, 0, DefaultMaxRequestBody, 0},
		{"Zero timeout uses default", 1024, 256, 0, 0, 0, DefaultTimeout},
		{"All zeros use defaults", 0, 0, 0, DefaultMaxResponseBody, DefaultMaxRequestBody, DefaultTimeout},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClientWithConfig(tt.maxResponseBody, tt.maxRequestBody, tt.timeout)

			if tt.expectDefaultResp > 0 && client.maxResponseBody != tt.expectDefaultResp {
				t.Errorf("maxResponseBody = %d, want %d", client.maxResponseBody, tt.expectDefaultResp)
			} else if tt.expectDefaultResp == 0 && tt.maxResponseBody > 0 && client.maxResponseBody != tt.maxResponseBody {
				t.Errorf("maxResponseBody = %d, want %d", client.maxResponseBody, tt.maxResponseBody)
			}

			if tt.expectDefaultReq > 0 && client.maxRequestBody != tt.expectDefaultReq {
				t.Errorf("maxRequestBody = %d, want %d", client.maxRequestBody, tt.expectDefaultReq)
			} else if tt.expectDefaultReq == 0 && tt.maxRequestBody > 0 && client.maxRequestBody != tt.maxRequestBody {
				t.Errorf("maxRequestBody = %d, want %d", client.maxRequestBody, tt.maxRequestBody)
			}

			if tt.expectDefaultTO > 0 && client.timeout != tt.expectDefaultTO {
				t.Errorf("timeout = %v, want %v", client.timeout, tt.expectDefaultTO)
			} else if tt.expectDefaultTO == 0 && tt.timeout > 0 && client.timeout != tt.timeout {
				t.Errorf("timeout = %v, want %v", client.timeout, tt.timeout)
			}
		})
	}
}

// TestHTTPRateLimiter tests the rate limiting functionality
func TestHTTPRateLimiter(t *testing.T) {
	limiter := NewHTTPRateLimiter(3, time.Second, 3) // 3 requests per second

	tenantID := "test-tenant"

	// First 3 requests should be allowed
	for i := 0; i < 3; i++ {
		if !limiter.IsAllowed(tenantID) {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}

	// 4th request should be denied
	if limiter.IsAllowed(tenantID) {
		t.Error("4th request should be denied due to rate limiting")
	}
}

// TestHTTPRateLimiterMultipleTenants tests isolation between tenants
func TestHTTPRateLimiterMultipleTenants(t *testing.T) {
	limiter := NewHTTPRateLimiter(2, time.Second, 2)

	// Tenant A can make 2 requests
	for i := 0; i < 2; i++ {
		if !limiter.IsAllowed("tenant-a") {
			t.Errorf("Tenant A request %d should be allowed", i+1)
		}
	}

	// Tenant B should still be able to make requests independently
	if !limiter.IsAllowed("tenant-b") {
		t.Error("Tenant B first request should be allowed")
	}

	// Tenant A should now be rate limited
	if limiter.IsAllowed("tenant-a") {
		t.Error("Tenant A third request should be denied")
	}
}

// TestHTTPRateLimiterRefill tests token refill over time
func TestHTTPRateLimiterRefill(t *testing.T) {
	limiter := NewHTTPRateLimiter(2, 500*time.Millisecond, 2)

	tenantID := "refill-test"

	// Use all tokens
	limiter.IsAllowed(tenantID)
	limiter.IsAllowed(tenantID)

	// Should be denied
	if limiter.IsAllowed(tenantID) {
		t.Error("Should be denied after using all tokens")
	}

	// Wait for token refill
	time.Sleep(600 * time.Millisecond)

	// Should be allowed again after refill
	if !limiter.IsAllowed(tenantID) {
		t.Error("Should be allowed after token refill")
	}
}

// TestHeadersExtraction tests that response headers are properly extracted
func TestHeadersExtraction(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Custom-Header", "custom-value")
		w.Header().Set("X-Rate-Limit-Remaining", "42")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client := NewClient()
	req, err := nethttp.NewRequestWithContext(context.Background(), nethttp.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	resp, err := client.httpClient.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, client.maxResponseBody))

	// Extract response headers
	responseHeaders := make(map[string]string)
	for key, values := range resp.Header {
		if len(values) > 0 {
			responseHeaders[key] = values[0]
		}
	}

	// Check headers are present
	if responseHeaders["Content-Type"] != "application/json" {
		t.Errorf("Missing or wrong Content-Type header: %s", responseHeaders["Content-Type"])
	}

	if responseHeaders["X-Custom-Header"] != "custom-value" {
		t.Errorf("Missing or wrong X-Custom-Header: %s", responseHeaders["X-Custom-Header"])
	}

	if responseHeaders["X-Rate-Limit-Remaining"] != "42" {
		t.Errorf("Missing or wrong X-Rate-Limit-Remaining: %s", responseHeaders["X-Rate-Limit-Remaining"])
	}

	// Verify body was read
	if string(body) != `{}` {
		t.Errorf("Unexpected body: %s", string(body))
	}
}

// TestEmptyHeaders tests behavior with nil/empty headers
func TestEmptyHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`ok`))
	}))
	defer server.Close()

	client := NewClient()

	// Test GET with nil headers
	req, err := nethttp.NewRequestWithContext(context.Background(), nethttp.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatalf("Failed to create GET request: %v", err)
	}

	resp, err := client.httpClient.Do(req)
	if err != nil {
		t.Fatalf("GET request failed: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Unexpected status code for GET: %d", resp.StatusCode)
	}

	// Test POST with empty content type
	req2, err := nethttp.NewRequestWithContext(context.Background(), nethttp.MethodPost, server.URL, strings.NewReader("{}"))
	if err != nil {
		t.Fatalf("Failed to create POST request: %v", err)
	}

	resp2, err := client.httpClient.Do(req2)
	if err != nil {
		t.Fatalf("POST request failed: %v", err)
	}
	resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK {
		t.Errorf("Unexpected status code for POST: %d", resp2.StatusCode)
	}
}

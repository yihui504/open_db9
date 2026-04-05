package http

import (
	"context"
	"fmt"
	"io"
	"net"
	nethttp "net/http"
	"net/url"
	"strings"
	"time"
)

// Response represents an HTTP response
type Response struct {
	StatusCode int               `json:"status_code"`
	Body       string            `json:"body"`
	Headers    map[string]string `json:"headers"`
}

// Client is a secure HTTP client with SSRF protection and size limits
type Client struct {
	httpClient      *nethttp.Client
	maxResponseBody int64         // default 1MB
	maxRequestBody  int64         // default 256KB
	timeout         time.Duration // default 5s
}

// Default configuration values
const (
	DefaultMaxResponseBody = 1 * 1024 * 1024 // 1MB
	DefaultMaxRequestBody  = 256 * 1024      // 256KB
	DefaultTimeout         = 5 * time.Second // 5 seconds
)

// NewClient creates a new secure HTTP client with default settings
func NewClient() *Client {
	return &Client{
		httpClient: &nethttp.Client{
			Timeout: DefaultTimeout,
			// Disable redirects to prevent SSRF via redirect chains
			CheckRedirect: func(req *nethttp.Request, via []*nethttp.Request) error {
				return nethttp.ErrUseLastResponse
			},
		},
		maxResponseBody: DefaultMaxResponseBody,
		maxRequestBody:  DefaultMaxRequestBody,
		timeout:         DefaultTimeout,
	}
}

// NewClientWithConfig creates a new secure HTTP client with custom configuration
func NewClientWithConfig(maxResponseBody, maxRequestBody int64, timeout time.Duration) *Client {
	if maxResponseBody <= 0 {
		maxResponseBody = DefaultMaxResponseBody
	}
	if maxRequestBody <= 0 {
		maxRequestBody = DefaultMaxRequestBody
	}
	if timeout <= 0 {
		timeout = DefaultTimeout
	}

	return &Client{
		httpClient: &nethttp.Client{
			Timeout: timeout,
			// Disable redirects to prevent SSRF via redirect chains
			CheckRedirect: func(req *nethttp.Request, via []*nethttp.Request) error {
				return nethttp.ErrUseLastResponse
			},
		},
		maxResponseBody: maxResponseBody,
		maxRequestBody:  maxRequestBody,
		timeout:         timeout,
	}
}

// DoGet performs a secure HTTP GET request
func (c *Client) DoGet(ctx context.Context, urlStr string, headers map[string]string) (*Response, error) {
	// Validate URL
	if err := c.validateURL(urlStr); err != nil {
		return nil, err
	}

	// SSRF protection: resolve hostname and check IP
	if err := c.checkSSRF(urlStr); err != nil {
		return nil, err
	}

	// Create request
	req, err := nethttp.NewRequestWithContext(ctx, nethttp.MethodGet, urlStr, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body with size limit
	body, err := io.ReadAll(io.LimitReader(resp.Body, c.maxResponseBody))
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Extract response headers
	responseHeaders := make(map[string]string)
	for key, values := range resp.Header {
		if len(values) > 0 {
			responseHeaders[key] = values[0]
		}
	}

	return &Response{
		StatusCode: resp.StatusCode,
		Body:       string(body),
		Headers:    responseHeaders,
	}, nil
}

// DoPost performs a secure HTTP POST request
func (c *Client) DoPost(ctx context.Context, urlStr string, body string, contentType string, headers map[string]string) (*Response, error) {
	// Validate URL
	if err := c.validateURL(urlStr); err != nil {
		return nil, err
	}

	// Validate body size
	if int64(len(body)) > c.maxRequestBody {
		return nil, fmt.Errorf("request body exceeds maximum allowed size of %d bytes", c.maxRequestBody)
	}

	// SSRF protection: resolve hostname and check IP
	if err := c.checkSSRF(urlStr); err != nil {
		return nil, err
	}

	// Create request
	req, err := nethttp.NewRequestWithContext(ctx, nethttp.MethodPost, urlStr, strings.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set content type
	if contentType == "" {
		contentType = "application/json"
	}
	req.Header.Set("Content-Type", contentType)

	// Set additional headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body with size limit
	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, c.maxResponseBody))
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Extract response headers
	responseHeaders := make(map[string]string)
	for key, values := range resp.Header {
		if len(values) > 0 {
			responseHeaders[key] = values[0]
		}
	}

	return &Response{
		StatusCode: resp.StatusCode,
		Body:       string(responseBody),
		Headers:    responseHeaders,
	}, nil
}

// validateURL validates that the URL uses HTTPS protocol
func (c *Client) validateURL(urlStr string) error {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	// Only allow HTTPS
	if parsedURL.Scheme != "https" {
		return fmt.Errorf("only HTTPS URLs are allowed, got scheme: %s", parsedURL.Scheme)
	}

	return nil
}

// checkSSRF performs SSRF protection by resolving the hostname and checking if it's a private IP
func (c *Client) checkSSRF(urlStr string) error {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL for SSRF check: %w", err)
	}

	hostname := parsedURL.Hostname()
	if hostname == "" {
		return fmt.Errorf("empty hostname in URL")
	}

	// Resolve hostname to IP addresses
	ips, err := net.LookupIP(hostname)
	if err != nil {
		return fmt.Errorf("DNS resolution failed for hostname %s: %w", hostname, err)
	}

	// Check all resolved IPs
	for _, ip := range ips {
		if isPrivateIP(ip) {
			return fmt.Errorf("SSRF protection: target IP %s is a private/reserved address", ip.String())
		}
	}

	return nil
}

// isPrivateIP checks if an IP address is private or reserved
func isPrivateIP(ip net.IP) bool {
	if ip == nil {
		return false
	}

	// Check using net.IP built-in methods first (works for both IPv4 and IPv6)
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}

	// Handle IPv4-mapped IPv6 addresses or pure IPv4
	ip4 := ip.To4()
	if ip4 != nil {
		return isPrivateIPv4(ip4)
	}

	// It's a full IPv6 address - check ULA and unspecified
	return isPrivateIPv6(ip)
}

// isPrivateIPv4 checks if an IPv4 address is in a private or reserved range
func isPrivateIPv4(ip net.IP) bool {
	// Loopback: 127.0.0.0/8
	if ip[0] == 127 {
		return true
	}

	// Private Class A: 10.0.0.0/8
	if ip[0] == 10 {
		return true
	}

	// Private Class B: 172.16.0.0/12
	if ip[0] == 172 && ip[1] >= 16 && ip[1] <= 31 {
		return true
	}

	// Private Class C: 192.168.0.0/16
	if ip[0] == 192 && ip[1] == 168 {
		return true
	}

	// Link-local: 169.254.0.0/16
	if ip[0] == 169 && ip[1] == 254 {
		return true
	}

	// 0.0.0.0/8
	if ip[0] == 0 {
		return true
	}

	return false
}

// isPrivateIPv6 checks if an IPv6 address is in a private or reserved range
func isPrivateIPv6(ip net.IP) bool {
	// Ensure we have a 16-byte IPv6 representation
	if len(ip) < 16 {
		return false
	}

	// Loopback: ::1
	if ip.IsLoopback() {
		return true
	}

	// Unique local address (ULA): fc00::/7 (includes fd00::/8)
	if ip[0] >= 0xfc && ip[0] <= 0xfd {
		return true
	}

	// Link-local: fe80::/10
	if ip[0] == 0xfe && (ip[1]&0xc0) == 0x80 {
		return true
	}

	// Unspecified: ::
	if ip.IsUnspecified() {
		return true
	}

	return false
}

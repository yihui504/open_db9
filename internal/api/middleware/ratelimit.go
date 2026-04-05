package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

// RateLimiter manages rate limiting for API requests
type RateLimiter struct {
	mu           sync.RWMutex
	requests     map[string][]time.Time // clientID -> request timestamps
	maxRequests  int                    // requests per window
	window       time.Duration         // time window
	cleanupInterval time.Duration
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	MaxRequests       int           // requests per window per token
	Window           time.Duration // time window
	CleanupInterval  time.Duration // how often to clean old timestamps
	Default          *RateLimiter  // default rate limiter
}

// Default rate limit: 100 requests per minute
const (
	DefaultMaxRequests      = 100
	DefaultWindow           = time.Minute
	DefaultCleanupInterval  = 5 * time.Minute
)

// DefaultRateLimitConfig returns default rate limiting configuration
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		MaxRequests:      DefaultMaxRequests,
		Window:           DefaultWindow,
		CleanupInterval:  DefaultCleanupInterval,
	}
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(maxRequests int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		requests:     make(map[string][]time.Time),
		maxRequests:  maxRequests,
		window:       window,
		cleanupInterval: DefaultCleanupInterval,
	}

	// Start cleanup goroutine
	go rl.cleanupLoop()

	return rl
}

// cleanupLoop periodically removes old request timestamps
func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(rl.cleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		rl.cleanup(time.Now())
	}
}

// cleanup removes timestamps older than the window
func (rl *RateLimiter) cleanup(now time.Time) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	cutoff := now.Add(-rl.window)

	for clientID, timestamps := range rl.requests {
		var valid []time.Time
		for _, ts := range timestamps {
			if ts.After(cutoff) {
				valid = append(valid, ts)
			}
		}

		if len(valid) == 0 {
			delete(rl.requests, clientID)
		} else {
			rl.requests[clientID] = valid
		}
	}
}

// IsAllowed checks if a request from the given clientID is allowed
func (rl *RateLimiter) IsAllowed(clientID string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	// Clean old timestamps for this client
	timestamps := rl.requests[clientID]
	var valid []time.Time
	for _, ts := range timestamps {
		if ts.After(cutoff) {
			valid = append(valid, ts)
		}
	}
	rl.requests[clientID] = valid

	// Check if under limit
	if len(valid) >= rl.maxRequests {
		return false
	}

	// Add current request
	rl.requests[clientID] = append(valid, now)
	return true
}

// RateLimitMiddleware creates a rate limiting middleware
// It extracts the client identifier from the JWT token or IP address
func RateLimitMiddleware(config RateLimitConfig) func(http.Handler) http.Handler {
	limiter := config.Default
	if limiter == nil {
		limiter = NewRateLimiter(config.MaxRequests, config.Window)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract client ID from JWT token or IP
			clientID := extractClientID(r)

			// Check rate limit
			if !limiter.IsAllowed(clientID) {
				respondWithError(w, http.StatusTooManyRequests, "Rate limit exceeded. Please try again later.")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// extractClientID extracts a unique client identifier from the request
// Priority: JWT user_id > JWT username > IP address
func extractClientID(r *http.Request) string {
	// Try to get user_id from context (set by auth middleware)
	if userID, ok := r.Context().Value(UserIDKey).(int); ok {
		return fmt.Sprintf("user:%d", userID)
	}

	// Try to get username from context
	if username, ok := r.Context().Value(UsernameKey).(string); ok {
		return fmt.Sprintf("user:%s", username)
	}

	// Fall back to IP address
	return fmt.Sprintf("ip:%s", r.RemoteAddr)
}

// RateLimitInfo returns current rate limit statistics for a client
func (rl *RateLimiter) RateLimitInfo(clientID string) (current int, remaining int, reset time.Time) {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	timestamps := rl.requests[clientID]
	var valid []time.Time
	for _, ts := range timestamps {
		if ts.After(cutoff) {
			valid = append(valid, ts)
		}
	}

	currentCount := len(valid)
	remainingCount := rl.maxRequests - currentCount

	// Calculate reset time (oldest valid timestamp + window)
	var resetTime time.Time
	if len(valid) > 0 {
		resetTime = valid[0].Add(rl.window)
	} else {
		resetTime = now.Add(rl.window)
	}

	return currentCount, remainingCount, resetTime
}

// GetRateLimitHeaders returns HTTP headers for rate limit information
func GetRateLimitHeaders(current, remaining int, reset time.Time) map[string]string {
	return map[string]string{
		"X-RateLimit-Limit":     fmt.Sprintf("%d", DefaultMaxRequests),
		"X-RateLimit-Remaining": fmt.Sprintf("%d", remaining),
		"X-RateLimit-Reset":     fmt.Sprintf("%d", reset.Unix()),
	}
}

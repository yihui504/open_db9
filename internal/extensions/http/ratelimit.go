package http

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

// HTTPRateLimiter implements token bucket rate limiting for HTTP extension requests
type HTTPRateLimiter struct {
	mu              sync.RWMutex
	buckets         map[string]*tokenBucket // tenantID -> token bucket
	maxRequests     int                     // requests per window per tenant
	refillRate      time.Duration           // how often to add a new token
	capacity        int                     // max tokens in bucket
	cleanupInterval time.Duration           // how often to clean up old buckets
}

// tokenBucket represents a token bucket for rate limiting
type tokenBucket struct {
	tokens     int
	lastRefill time.Time
	mu         sync.Mutex
}

// Default configuration for HTTP extension rate limiting
const (
	DefaultHTTPMaxRequests     = 10               // 10 requests per minute per tenant
	DefaultHTTPRefillRate      = 6 * time.Second  // refill every 6 seconds (10/min)
	DefaultHTTPCapacity        = 10               // bucket capacity
	DefaultHTTPCleanupInterval = 10 * time.Minute // cleanup interval
)

// NewHTTPRateLimiter creates a new HTTP extension rate limiter
func NewHTTPRateLimiter(maxRequests int, refillRate time.Duration, capacity int) *HTTPRateLimiter {
	if maxRequests <= 0 {
		maxRequests = DefaultHTTPMaxRequests
	}
	if refillRate <= 0 {
		refillRate = DefaultHTTPRefillRate
	}
	if capacity <= 0 {
		capacity = DefaultHTTPCapacity
	}

	rl := &HTTPRateLimiter{
		buckets:         make(map[string]*tokenBucket),
		maxRequests:     maxRequests,
		refillRate:      refillRate,
		capacity:        capacity,
		cleanupInterval: DefaultHTTPCleanupInterval,
	}

	// Start cleanup goroutine
	go rl.cleanupLoop()

	return rl
}

// cleanupLoop periodically removes inactive buckets
func (rl *HTTPRateLimiter) cleanupLoop() {
	ticker := time.NewTicker(rl.cleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		rl.cleanup()
	}
}

// cleanup removes buckets that haven't been used recently
func (rl *HTTPRateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.cleanupInterval)

	for id, bucket := range rl.buckets {
		bucket.mu.Lock()
		if bucket.lastRefill.Before(cutoff) {
			delete(rl.buckets, id)
		}
		bucket.mu.Unlock()
	}
}

// IsAllowed checks if a request from the given tenant is allowed under the rate limit
func (rl *HTTPRateLimiter) IsAllowed(tenantID string) bool {
	rl.mu.RLock()
	bucket, exists := rl.buckets[tenantID]
	rl.mu.RUnlock()

	if !exists {
		rl.mu.Lock()
		// Double-check after acquiring write lock
		if bucket, exists = rl.buckets[tenantID]; !exists {
			bucket = &tokenBucket{
				tokens:     rl.capacity - 1, // Use one token for this request
				lastRefill: time.Now(),
			}
			rl.buckets[tenantID] = bucket
			rl.mu.Unlock()
			return true
		}
		rl.mu.Unlock()
	}

	bucket.mu.Lock()
	defer bucket.mu.Unlock()

	// Refill tokens based on elapsed time
	now := time.Now()
	elapsed := now.Sub(bucket.lastRefill)
	tokensToAdd := int(elapsed / rl.refillRate)

	if tokensToAdd > 0 {
		bucket.tokens += tokensToAdd
		if bucket.tokens > rl.capacity {
			bucket.tokens = rl.capacity
		}
		bucket.lastRefill = now
	}

	// Check if we have tokens available
	if bucket.tokens > 0 {
		bucket.tokens--
		return true
	}

	return false
}

// GetBucketInfo returns current rate limit information for a tenant
func (rl *HTTPRateLimiter) GetBucketInfo(tenantID string) (currentTokens int, nextRefill time.Time) {
	rl.mu.RLock()
	bucket, exists := rl.buckets[tenantID]
	rl.mu.RUnlock()

	if !exists {
		return rl.capacity - 1, time.Now().Add(rl.refillRate)
	}

	bucket.mu.Lock()
	defer bucket.mu.Unlock()

	// Refill tokens based on elapsed time
	now := time.Now()
	elapsed := now.Sub(bucket.lastRefill)
	tokensToAdd := int(elapsed / rl.refillRate)

	if tokensToAdd > 0 {
		bucket.tokens += tokensToAdd
		if bucket.tokens > rl.capacity {
			bucket.tokens = rl.capacity
		}
		bucket.lastRefill = now
	}

	return bucket.tokens, bucket.lastRefill.Add(rl.refillRate)
}

// HTTPRateLimitMiddleware creates middleware for HTTP extension rate limiting
func HTTPRateLimitMiddleware(limiter *HTTPRateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract tenant ID from context or use database ID from path
			tenantID := extractTenantID(r)

			// Check rate limit
			if !limiter.IsAllowed(tenantID) {
				currentTokens, nextRefill := limiter.GetBucketInfo(tenantID)

				w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", limiter.maxRequests))
				w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", currentTokens))
				w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", nextRefill.Unix()))
				http.Error(w, "HTTP extension rate limit exceeded. Please try again later.", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// extractTenantID extracts tenant identifier from request context or path
func extractTenantID(r *http.Request) string {
	// Try to get user_id from context (set by auth middleware)
	if userID, ok := r.Context().Value("user_id").(int); ok {
		return fmt.Sprintf("user:%d", userID)
	}

	// Try to get username from context
	if username, ok := r.Context().Value("username").(string); ok && username != "" {
		return fmt.Sprintf("user:%s", username)
	}

	// Fall back to database ID from path
	dbID := r.PathValue("id")
	if dbID != "" {
		return fmt.Sprintf("db:%s", dbID)
	}

	// Last resort: IP address
	return fmt.Sprintf("ip:%s", r.RemoteAddr)
}

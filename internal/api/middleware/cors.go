package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/open-db9/db9/internal/config"
)

// CORSConfig holds CORS configuration
type CORSConfig struct {
	// AllowedOrigins is a list of origins allowed to make requests
	// Use "*" to allow all origins, or specific origins like "https://example.com"
	AllowedOrigins []string

	// AllowedMethods is the list of HTTP methods allowed
	AllowedMethods []string

	// AllowedHeaders is the list of headers allowed in requests
	AllowedHeaders []string

	// ExposedHeaders is the list of headers exposed to the browser
	ExposedHeaders []string

	// AllowCredentials indicates whether the request can include credentials
	AllowCredentials bool

	// MaxAge is the maximum age (in seconds) for preflight requests
	MaxAge int
}

// DefaultCORSConfig returns default CORS configuration
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type", "Authorization", "X-Requested-With"},
		ExposedHeaders: []string{"X-Total-Count", "X-Page-Limit"},
		AllowCredentials: false,
		MaxAge:           86400, // 24 hours
	}
}

// RestrictiveCORSConfig returns a restrictive CORS configuration for production
func RestrictiveCORSConfig(allowedOrigins []string) CORSConfig {
	return CORSConfig{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:    []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowedHeaders:    []string{"Content-Type", "Authorization", "X-Requested-With"},
		ExposedHeaders:    []string{"X-Total-Count", "X-Page-Limit"},
		AllowCredentials:  true,
		MaxAge:            86400,
	}
}

// CORSMiddleware creates a CORS middleware from config.CORSConfig
func CORSMiddleware(cfg config.CORSConfig) func(http.Handler) http.Handler {
	corsConfig := CORSConfig{
		AllowedOrigins:   cfg.AllowedOrigins,
		AllowedMethods:   cfg.AllowedMethods,
		AllowedHeaders:   cfg.AllowedHeaders,
		ExposedHeaders:   cfg.ExposedHeaders,
		AllowCredentials: cfg.AllowCredentials,
		MaxAge:           cfg.MaxAge,
	}
	return CORSWithConfig(corsConfig)
}

// CORSWithConfig creates a CORS middleware with custom configuration
func CORSWithConfig(config CORSConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Check if origin is allowed
			if !isOriginAllowed(origin, config.AllowedOrigins) {
				// Origin not allowed, don't set CORS headers
				next.ServeHTTP(w, r)
				return
			}

			// Set Access-Control-Allow-Origin
			if len(config.AllowedOrigins) == 1 && config.AllowedOrigins[0] == "*" {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			} else {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			}

			// Set Access-Control-Allow-Credentials
			if config.AllowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			// Set Access-Control-Expose-Headers
			if len(config.ExposedHeaders) > 0 {
				w.Header().Set("Access-Control-Expose-Headers", strings.Join(config.ExposedHeaders, ", "))
			}

			// Handle preflight requests
			if r.Method == http.MethodOptions {
				// Set Access-Control-Allow-Methods
				w.Header().Set("Access-Control-Allow-Methods", strings.Join(config.AllowedMethods, ", "))

				// Set Access-Control-Allow-Headers
				w.Header().Set("Access-Control-Allow-Headers", strings.Join(config.AllowedHeaders, ", "))

				// Set Access-Control-Max-Age
				w.Header().Set("Access-Control-Max-Age", fmt.Sprintf("%d", config.MaxAge))

				// Respond with 204 No Content
				w.WriteHeader(http.StatusNoContent)
				return
			}

			// Set Access-Control-Allow-Headers for simple requests
			if len(config.AllowedHeaders) > 0 {
				w.Header().Set("Access-Control-Allow-Headers", strings.Join(config.AllowedHeaders, ", "))
			}

			next.ServeHTTP(w, r)
		})
	}
}

// isOriginAllowed checks if the given origin is allowed
func isOriginAllowed(origin string, allowedOrigins []string) bool {
	if origin == "" {
		return true // Same-origin requests don't need CORS
	}

	for _, allowed := range allowedOrigins {
		if allowed == "*" {
			return true
		}
		if allowed == origin {
			return true
		}
	}
	return false
}

// CORSMiddlewareForOrigins creates a CORS middleware with production-safe defaults
// It uses allowed origins provided as string arguments
func CORSMiddlewareForOrigins(allowedOrigins ...string) func(http.Handler) http.Handler {
	config := DefaultCORSConfig()

	if len(allowedOrigins) > 0 {
		config.AllowedOrigins = allowedOrigins
		config.AllowCredentials = true // Enable credentials when origins are restricted
	}

	return CORSWithConfig(config)
}

// PreflightRequestHandler handles OPTIONS preflight requests
// This is useful for explicit preflight handling if needed
func PreflightRequestHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Max-Age", "86400")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// CORSPreflightHandler creates a simple preflight handler
func CORSPreflightHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Max-Age", "3600")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	})
}

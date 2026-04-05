package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/open-db9/db9/internal/auth"
)

// Context keys for storing user information in request context
type contextKey string

const (
	// UserIDKey is the context key for user ID
	UserIDKey contextKey = "user_id"
	// UsernameKey is the context key for username
	UsernameKey contextKey = "username"
	// RoleKey is the context key for user role (for future use)
	RoleKey contextKey = "role"
)

// AuthResponse represents an authentication error response
type AuthResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// AuthMiddleware creates a JWT authentication middleware
// It validates JWT tokens from the Authorization header and sets user context
func AuthMiddleware(authManager *auth.Manager, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			respondAuthError(w, "Missing authorization header", http.StatusUnauthorized)
			return
		}

		// Check if the header has the Bearer prefix
		if !strings.HasPrefix(authHeader, "Bearer ") {
			respondAuthError(w, "Invalid authorization header format. Expected: Bearer <token>", http.StatusUnauthorized)
			return
		}

		// Extract the token
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		tokenString = strings.TrimSpace(tokenString)

		if tokenString == "" {
			respondAuthError(w, "Empty token", http.StatusUnauthorized)
			return
		}

		// Validate token
		claims, err := authManager.ValidateToken(tokenString)
		if err != nil {
			// Check for specific error types
			if err == auth.ErrExpiredToken {
				respondAuthError(w, "Token has expired", http.StatusUnauthorized)
				return
			}
			if err == auth.ErrEmptyToken {
				respondAuthError(w, "Empty token", http.StatusUnauthorized)
				return
			}
			// Generic invalid token error
			respondAuthError(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		// Create new context with user information
		ctx := r.Context()
		ctx = context.WithValue(ctx, UserIDKey, claims.UserID)
		ctx = context.WithValue(ctx, UsernameKey, claims.Username)
		ctx = context.WithValue(ctx, RoleKey, claims.Subject) // Using Subject as role placeholder

		// Create new request with updated context
		r = r.WithContext(ctx)

		// Call the next handler
		next(w, r)
	}
}

// RequireAuth wraps an http.Handler with JWT authentication
// This is an alternative signature for handlers that use http.Handler instead of http.HandlerFunc
func RequireAuth(authManager *auth.Manager, handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			respondAuthError(w, "Missing authorization header", http.StatusUnauthorized)
			return
		}

		// Check if the header has the Bearer prefix
		if !strings.HasPrefix(authHeader, "Bearer ") {
			respondAuthError(w, "Invalid authorization header format. Expected: Bearer <token>", http.StatusUnauthorized)
			return
		}

		// Extract the token
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		tokenString = strings.TrimSpace(tokenString)

		if tokenString == "" {
			respondAuthError(w, "Empty token", http.StatusUnauthorized)
			return
		}

		// Validate token
		claims, err := authManager.ValidateToken(tokenString)
		if err != nil {
			// Check for specific error types
			if err == auth.ErrExpiredToken {
				respondAuthError(w, "Token has expired", http.StatusUnauthorized)
				return
			}
			if err == auth.ErrEmptyToken {
				respondAuthError(w, "Empty token", http.StatusUnauthorized)
				return
			}
			// Generic invalid token error
			respondAuthError(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		// Create new context with user information
		ctx := r.Context()
		ctx = context.WithValue(ctx, UserIDKey, claims.UserID)
		ctx = context.WithValue(ctx, UsernameKey, claims.Username)
		ctx = context.WithValue(ctx, RoleKey, claims.Subject) // Using Subject as role placeholder

		// Create new request with updated context
		r = r.WithContext(ctx)

		// Call the next handler
		handler.ServeHTTP(w, r)
	})
}

// OptionalAuth provides optional authentication - doesn't fail if token is missing
// but still validates and sets context if token is present
func OptionalAuth(authManager *auth.Manager, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Try to extract and validate token, but don't fail if missing
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			tokenString = strings.TrimSpace(tokenString)

			if tokenString != "" {
				// Validate token
				claims, err := authManager.ValidateToken(tokenString)
				if err == nil {
					// Token is valid, set context
					ctx = context.WithValue(ctx, UserIDKey, claims.UserID)
					ctx = context.WithValue(ctx, UsernameKey, claims.Username)
					ctx = context.WithValue(ctx, RoleKey, claims.Subject)
					r = r.WithContext(ctx)
				}
				// If token is invalid, we just continue without setting context
			}
		}

		// Call the next handler
		next(w, r)
	}
}

// GetUserID retrieves the user ID from the request context
func GetUserID(r *http.Request) (int, bool) {
	userID, ok := r.Context().Value(UserIDKey).(int)
	return userID, ok
}

// GetUsername retrieves the username from the request context
func GetUsername(r *http.Request) (string, bool) {
	username, ok := r.Context().Value(UsernameKey).(string)
	return username, ok
}

// GetRole retrieves the role from the request context
func GetRole(r *http.Request) (string, bool) {
	role, ok := r.Context().Value(RoleKey).(string)
	return role, ok
}

// RequireRole checks if the user has the required role
// Returns 403 Forbidden if the user doesn't have the required role
func RequireRole(requiredRole string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			role, ok := GetRole(r)
			if !ok {
				respondAuthError(w, "User role not found in context", http.StatusForbidden)
				return
			}

			// Simple role check - can be extended for more complex role-based access
			if role != requiredRole {
				respondAuthError(w, "Insufficient permissions", http.StatusForbidden)
				return
			}

			next(w, r)
		}
	}
}

// respondAuthError sends a standardized authentication error response
func respondAuthError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(AuthResponse{
		Success: false,
		Error:   message,
	})
}

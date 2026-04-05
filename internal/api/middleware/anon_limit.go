package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
)

// AnonymousContextKey is the context key for anonymous user information
const AnonymousContextKey contextKey = "anonymous_info"

// AnonymousInfo stores information about an anonymous user
type AnonymousInfo struct {
	TenantID     string          `json:"tenant_id"`
	SessionID    string          `json:"session_id"`
	Capabilities json.RawMessage `json:"capabilities"`
	MaxDatabases int             `json:"max_databases"`
}

// AnonymousLimitMiddleware checks if an anonymous user has exceeded their database creation limit
// It should be applied to database creation endpoints
func AnonymousLimitMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Check if user is anonymous by examining the subject/role in context
		role, ok := GetRole(r)
		if !ok || !strings.HasPrefix(role, "anonymous:") {
			// Not an anonymous user, proceed normally
			next(w, r)
			return
		}

		// Extract tenant ID from role (format: "anonymous:<tenant_id>")
		tenantID := strings.TrimPrefix(role, "anonymous:")
		if tenantID == "" {
			respondAuthError(w, "Invalid anonymous tenant ID", http.StatusForbidden)
			return
		}

		// Get database connection
		db, err := getDatabaseFromContext(ctx)
		if err != nil {
			respondAuthError(w, "Database not available", http.StatusInternalServerError)
			return
		}

		// Query anonymous account capabilities
		var capabilities []byte
		err = db.QueryRow(ctx,
			"SELECT capabilities FROM anonymous_accounts WHERE tenant_id = $1",
			tenantID).Scan(&capabilities)

		if err == pgx.ErrNoRows {
			// Anonymous account not found or expired, deny access
			respondAuthError(w, "Anonymous account not found or expired", http.StatusForbidden)
			return
		}
		if err != nil {
			respondAuthError(w, "Failed to query anonymous account", http.StatusInternalServerError)
			return
		}

		// Parse capabilities to get max_databases limit
		var caps map[string]interface{}
		if err := json.Unmarshal(capabilities, &caps); err != nil {
			respondAuthError(w, "Failed to parse capabilities", http.StatusInternalServerError)
			return
		}

		maxDatabases := 5 // default limit
		if max, ok := caps["max_databases"].(float64); ok {
			maxDatabases = int(max)
		}

		// Count current databases for this tenant
		var dbCount int
		err = db.QueryRow(ctx,
			"SELECT COUNT(*) FROM databases WHERE tenant_id = $1",
			tenantID).Scan(&dbCount)

		if err != nil {
			respondAuthError(w, "Failed to count databases", http.StatusInternalServerError)
			return
		}

		// Create anonymous info and store in context
		anonInfo := &AnonymousInfo{
			TenantID:     tenantID,
			Capabilities: capabilities,
			MaxDatabases: maxDatabases,
		}
		ctx = context.WithValue(ctx, AnonymousContextKey, anonInfo)
		r = r.WithContext(ctx)

		// Check if under limit (allowing this creation)
		if r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/databases") {
			if dbCount >= maxDatabases {
				respondAuthError(w,
					fmt.Sprintf("Anonymous account limit reached. Maximum %d databases allowed.", maxDatabases),
					http.StatusForbidden)
				return
			}
		}

		// Proceed to next handler
		next(w, r)
	}
}

// GetAnonymousInfo retrieves anonymous user info from request context
func GetAnonymousInfo(r *http.Request) (*AnonymousInfo, bool) {
	info, ok := r.Context().Value(AnonymousContextKey).(*AnonymousInfo)
	return info, ok
}

// IsAnonymousUser checks if the current user is an anonymous user
func IsAnonymousUser(r *http.Request) bool {
	role, ok := GetRole(r)
	if !ok {
		return false
	}
	return strings.HasPrefix(role, "anonymous:")
}

// Helper function to get database connection (similar to handlers.GetDefaultDatabase)
// This is a simplified version - in production, you'd use the proper registry
func getDatabaseFromContext(ctx context.Context) (*databaseConnection, error) {
	// This would typically get the DB from a global registry or dependency injection
	// For now, we'll use a placeholder that should be replaced with actual implementation
	// In the actual implementation, this would call handlers.GetDefaultDatabase()
	return nil, fmt.Errorf("database connection not available in middleware context")
}

// Placeholder for database connection type
// In production, this would be *database.ConnectionPool
type databaseConnection struct {
	// Database connection fields
}

func (db *databaseConnection) QueryRow(ctx context.Context, sql string, args ...interface{}) rowScanner {
	// Placeholder implementation
	return &placeholderRow{}
}

func (db *databaseConnection) Query(ctx context.Context, sql string, args ...interface{}) (rowsScanner, error) {
	// Placeholder implementation
	return nil, nil
}

// Interfaces for database operations
type rowScanner interface {
	Scan(dest ...interface{}) error
}

type rowsScanner interface {
	Next() bool
	Scan(dest ...interface{}) error
	Close() error
	Err() error
}

// placeholderRow implements rowScanner for testing
type placeholderRow struct{}

func (r *placeholderRow) Scan(dest ...interface{}) error {
	return fmt.Errorf("placeholder row scanner")
}

package router

import (
	"net/http"
	"strings"
	"time"

	"github.com/open-db9/db9/internal/api/handlers"
	"github.com/open-db9/db9/internal/api/middleware"
	"github.com/open-db9/db9/internal/auth"
	"github.com/open-db9/db9/internal/config"
	httpext "github.com/open-db9/db9/internal/extensions/http"
)

// Global HTTP extension handler instance
var httpExtensionHandler *handlers.HttpExtensionHandler

func init() {
	// Initialize HTTP extension client and handler
	httpClient := httpext.NewClient()
	httpExtensionHandler = handlers.NewHttpExtensionHandler(httpClient)
}

// NewRouter creates and configures a new HTTP router with middleware chain
func NewRouter(authManager *auth.Manager) http.Handler {
	mux := http.NewServeMux()

	// Public routes (no authentication required)
	mux.HandleFunc("/", handlers.RootHandler)
	mux.HandleFunc("/health", handlers.HealthHandler)
	mux.HandleFunc("/version", handlers.VersionHandler)
	mux.Handle("/metrics", handlers.PrometheusHandler())

	// Anonymous authentication routes (public)
	mux.HandleFunc("/api/v1/auth/anonymous", func(w http.ResponseWriter, r *http.Request) {
		handlers.AnonymousRegisterHandler(w, r, authManager)
	})

	// Protected routes (authentication required)
	// Unified API handler for all /api/v1/databases/* routes
	// This avoids http.ServeMux route conflicts by using a single handler
	// that routes based on the second path segment
	mux.HandleFunc("/api/v1/databases/", apiHandler)

	// User management routes
	mux.HandleFunc("/api/v1/users/", usersHandler)

	// Auth routes (protected)
	mux.HandleFunc("/api/v1/auth/claim", func(w http.ResponseWriter, r *http.Request) {
		handlers.ClaimAccountHandler(w, r, authManager)
	})

	// Load configuration via singleton pattern
	cfg := config.Load()

	// Build middleware chain
	// Order is important: ErrorHandler (outermost) -> RateLimiter -> CORS -> RequestID -> Logger -> Auth (for protected routes)
	var handler http.Handler = mux

	// Apply middleware in reverse order (last applied is first executed)
	middlewares := []func(http.Handler) http.Handler{
		middleware.Logger,
		middleware.RequestID,
		middleware.CORSWithConfig(middleware.CORSConfig{
			AllowedOrigins:   cfg.CORS.AllowedOrigins,
			AllowedMethods:   cfg.CORS.AllowedMethods,
			AllowedHeaders:   cfg.CORS.AllowedHeaders,
			ExposedHeaders:   cfg.CORS.ExposedHeaders,
			AllowCredentials: cfg.CORS.AllowCredentials,
			MaxAge:           cfg.CORS.MaxAge,
		}),
		middleware.ErrorHandler,
	}

	// Add rate limiter if enabled
	if cfg.RateLimit.Enabled {
		rateLimitConfig := middleware.RateLimitConfig{
			MaxRequests:     cfg.RateLimit.RequestsPerMinute,
			Window:          time.Minute,
			CleanupInterval: 5 * time.Minute,
		}
		middlewares = append(middlewares, middleware.RateLimitMiddleware(rateLimitConfig))
	}

	handler = middleware.Chain(handler, middlewares...)

	// Wrap protected routes with auth middleware
	return createProtectedRoutes(handler, authManager)
}

// createProtectedRoutes wraps protected routes with authentication middleware
func createProtectedRoutes(handler http.Handler, authManager *auth.Manager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if the path is public
		if isPublicPath(r.URL.Path) {
			handler.ServeHTTP(w, r)
			return
		}

		// Apply auth middleware to protected routes using RequireAuth
		protectedHandler := middleware.RequireAuth(authManager, handler)
		protectedHandler.ServeHTTP(w, r)
	})
}

// isPublicPath checks if a given path is public (doesn't require authentication)
func isPublicPath(path string) bool {
	publicPaths := []string{
		"/",
		"/health",
		"/version",
		"/metrics",
		"/api/v1/auth/anonymous",
	}

	for _, publicPath := range publicPaths {
		if path == publicPath {
			return true
		}
	}

	return false
}

// apiHandler is a unified handler that routes to specific handlers
// based on the second path segment after /api/v1/databases/
func apiHandler(w http.ResponseWriter, r *http.Request) {
	prefix := "/api/v1/databases/"
	if !strings.HasPrefix(r.URL.Path, prefix) {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	rest := strings.Trim(strings.TrimPrefix(r.URL.Path, prefix), "/")

	if rest == "" {
		handleDatabasesCollection(w, r)
		return
	}

	parts := strings.Split(rest, "/")

	if len(parts) < 1 || parts[0] == "" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	if len(parts) == 1 {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	switch parts[1] {
	case "sql":
		handleSQL(w, r)
	case "files":
		handleFiles(w, r)
	case "snapshots":
		handleSnapshots(w, r)
	case "branches":
		handleBranches(w, r)
	case "metrics":
		handleMetrics(w, r)
	case "schema":
		handleSchema(w, r)
	case "rag":
		handleRAG(w, r)
	case "http":
		handleHTTPExtension(w, r)
	default:
		http.Error(w, "Not found", http.StatusNotFound)
	}
}

// handleDatabasesCollection handles requests to /api/v1/databases/
// Supports: GET (list databases), POST (create database)
func handleDatabasesCollection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		handlers.ListDatabasesHandler(w, r)
	case http.MethodPost:
		handlers.CreateDatabaseHandler(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleSQL routes SQL execution requests
func handleSQL(w http.ResponseWriter, r *http.Request) {
	// Existing sql.go logic
	handlers.ExecuteSQLHandler(w, r)
}

// handleSnapshots routes snapshot management requests
func handleSnapshots(w http.ResponseWriter, r *http.Request) {
	prefix := "/api/v1/databases/"
	rest := strings.TrimPrefix(r.URL.Path, prefix)
	parts := strings.Split(rest, "/")

	// Expected formats:
	// /api/v1/databases/:id/snapshots - create/list
	// /api/v1/databases/:id/snapshots/:sid - get/delete
	// /api/v1/databases/:id/snapshots/:sid/restore - restore
	if len(parts) < 2 || parts[1] != "snapshots" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	// Route based on method and path
	if len(parts) == 2 {
		// /api/v1/databases/:id/snapshots
		switch r.Method {
		case http.MethodPost:
			handlers.CreateSnapshotHandler(w, r)
		case http.MethodGet:
			handlers.ListSnapshotsHandler(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	} else if len(parts) == 3 {
		// /api/v1/databases/:id/snapshots/:sid
		switch r.Method {
		case http.MethodGet:
			handlers.GetSnapshotHandler(w, r)
		case http.MethodDelete:
			handlers.DeleteSnapshotHandler(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	} else if len(parts) == 4 && parts[3] == "restore" {
		// /api/v1/databases/:id/snapshots/:sid/restore
		switch r.Method {
		case http.MethodPost:
			handlers.RestoreSnapshotHandler(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	} else {
		http.Error(w, "Not found", http.StatusNotFound)
	}
}

// handleBranches routes branch management requests
func handleBranches(w http.ResponseWriter, r *http.Request) {
	prefix := "/api/v1/databases/"
	rest := strings.TrimPrefix(r.URL.Path, prefix)
	parts := strings.Split(rest, "/")

	// Expected formats:
	// /api/v1/databases/:id/branches - create/list
	// /api/v1/databases/:id/branches/:bid - delete
	if len(parts) < 2 || parts[1] != "branches" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	if len(parts) == 2 {
		// /api/v1/databases/:id/branches
		switch r.Method {
		case http.MethodPost:
			handlers.CreateBranchHandler(w, r)
		case http.MethodGet:
			handlers.ListBranchesHandler(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	} else if len(parts) == 3 {
		// /api/v1/databases/:id/branches/:bid
		switch r.Method {
		case http.MethodDelete:
			handlers.DeleteBranchHandler(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	} else {
		http.Error(w, "Not found", http.StatusNotFound)
	}
}

// handleFiles routes file management requests
func handleFiles(w http.ResponseWriter, r *http.Request) {
	prefix := "/api/v1/databases/"
	rest := strings.TrimPrefix(r.URL.Path, prefix)
	parts := strings.SplitN(rest, "/", 3)

	// Expected format: /api/v1/databases/:id/files[/:path]
	if len(parts) < 2 || parts[1] != "files" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	// Route based on method and path
	if len(parts) == 2 {
		// /api/v1/databases/:id/files
		switch r.Method {
		case http.MethodPost:
			handlers.UploadFileHandler(w, r)
		case http.MethodGet:
			handlers.ListFilesHandler(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	} else if len(parts) >= 3 {
		// Check for copy endpoint: /api/v1/databases/:id/files/copy
		if parts[2] == "copy" {
			if r.Method == http.MethodPost {
				handlers.CopyFileHandler(w, r)
			} else {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
			return
		}

		// /api/v1/databases/:id/files/:path
		switch r.Method {
		case http.MethodGet:
			handlers.DownloadFileHandler(w, r)
		case http.MethodDelete:
			handlers.DeleteFileHandler(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

// handleMetrics routes metrics and observability requests
func handleMetrics(w http.ResponseWriter, r *http.Request) {
	prefix := "/api/v1/databases/"
	rest := strings.TrimPrefix(r.URL.Path, prefix)
	parts := strings.Split(rest, "/")

	// Expected formats:
	// /api/v1/databases/:id/metrics/stats - database statistics
	// /api/v1/databases/:id/metrics/queries - query statistics
	// /api/v1/databases/:id/metrics/slow - slow queries
	// /api/v1/databases/:id/metrics/reset - reset statistics
	if len(parts) < 2 || parts[1] != "metrics" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	if len(parts) == 3 {
		// /api/v1/databases/:id/metrics/:action
		switch parts[2] {
		case "stats":
			switch r.Method {
			case http.MethodGet:
				handlers.GetDatabaseStatsHandler(w, r)
			default:
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		case "queries":
			switch r.Method {
			case http.MethodGet:
				handlers.GetQueryStatsHandler(w, r)
			default:
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		case "slow":
			switch r.Method {
			case http.MethodGet:
				handlers.GetSlowQueriesHandler(w, r)
			default:
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		case "reset":
			switch r.Method {
			case http.MethodPost:
				http.Error(w, "Not implemented yet", http.StatusNotImplemented)
			default:
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		default:
			http.Error(w, "Not found", http.StatusNotFound)
		}
	} else {
		http.Error(w, "Not found", http.StatusNotFound)
	}
}

// handleSchema routes schema introspection requests
func handleSchema(w http.ResponseWriter, r *http.Request) {
	prefix := "/api/v1/databases/"
	rest := strings.TrimPrefix(r.URL.Path, prefix)
	parts := strings.Split(rest, "/")

	// Expected formats:
	// /api/v1/databases/:id/schema - list schemas
	// /api/v1/databases/:id/schema/tables - list tables
	// /api/v1/databases/:id/schema/table - table details
	if len(parts) < 2 || parts[1] != "schema" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	if len(parts) == 2 {
		// /api/v1/databases/:id/schema
		switch r.Method {
		case http.MethodGet:
			handlers.ListSchemasHandler(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	} else if len(parts) == 3 {
		// /api/v1/databases/:id/schema/:action
		switch parts[2] {
		case "tables":
			switch r.Method {
			case http.MethodGet:
				handlers.ListTablesHandler(w, r)
			default:
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		case "table":
			switch r.Method {
			case http.MethodGet:
				handlers.GetTableDetailsHandler(w, r)
			default:
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		default:
			http.Error(w, "Not found", http.StatusNotFound)
		}
	} else {
		http.Error(w, "Not found", http.StatusNotFound)
	}
}

// usersHandler routes user management requests
func usersHandler(w http.ResponseWriter, r *http.Request) {
	prefix := "/api/v1/users/"
	if !strings.HasPrefix(r.URL.Path, prefix) {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	rest := strings.TrimPrefix(r.URL.Path, prefix)

	// /api/v1/users - list or create users
	if rest == "" {
		switch r.Method {
		case http.MethodGet:
			handlers.ListUsersHandler(w, r)
		case http.MethodPost:
			handlers.CreateUserHandler(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	// /api/v1/users/:id - get, update, or delete specific user
	parts := strings.Split(rest, "/")
	if len(parts) == 1 && parts[0] != "" {
		switch r.Method {
		case http.MethodGet:
			handlers.GetUserHandler(w, r)
		case http.MethodPut, http.MethodPatch:
			handlers.UpdateUserHandler(w, r)
		case http.MethodDelete:
			handlers.DeleteUserHandler(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}
}

// handleRAG routes RAG system requests
func handleRAG(w http.ResponseWriter, r *http.Request) {
	prefix := "/api/v1/databases/"
	rest := strings.TrimPrefix(r.URL.Path, prefix)
	parts := strings.Split(rest, "/")

	// Expected formats:
	// /api/v1/databases/:id/rag/documents - create/list documents
	// /api/v1/databases/:id/rag/documents/:doc_id - get/delete/update document
	// /api/v1/databases/:id/rag/query - query RAG system
	if len(parts) < 2 || parts[1] != "rag" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	if len(parts) == 2 {
		// /api/v1/databases/:id/rag
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	if len(parts) == 3 {
		// /api/v1/databases/:id/rag/:action
		switch parts[2] {
		case "documents":
			// Documents endpoint
			switch r.Method {
			case http.MethodPost:
				handlers.CreateDocumentHandler(w, r)
			case http.MethodGet:
				handlers.ListDocumentsHandler(w, r)
			default:
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		case "query":
			// Query endpoint
			switch r.Method {
			case http.MethodPost:
				// This would be handled by the Python FastAPI service
				http.Error(w, "Query endpoint handled by RAG Python service", http.StatusNotImplemented)
			default:
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		default:
			http.Error(w, "Not found", http.StatusNotFound)
		}
		return
	}

	if len(parts) == 4 && parts[2] == "documents" {
		// /api/v1/databases/:id/rag/documents/:doc_id
		switch r.Method {
		case http.MethodGet:
			handlers.GetDocumentHandler(w, r)
		case http.MethodDelete:
			handlers.DeleteDocumentHandler(w, r)
		case http.MethodPatch:
			handlers.UpdateDocumentChunkCountHandler(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	http.Error(w, "Not found", http.StatusNotFound)
}

// handleHTTPExtension routes HTTP extension requests
func handleHTTPExtension(w http.ResponseWriter, r *http.Request) {
	prefix := "/api/v1/databases/"
	rest := strings.TrimPrefix(r.URL.Path, prefix)
	parts := strings.Split(rest, "/")

	// Expected formats:
	// /api/v1/databases/:id/http/get - HTTP GET request
	// /api/v1/databases/:id/http/post - HTTP POST request
	if len(parts) < 3 || parts[1] != "http" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	if len(parts) == 3 {
		// /api/v1/databases/:id/http - invalid, need action
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	if len(parts) == 4 {
		// /api/v1/databases/:id/http/:action
		switch parts[3] {
		case "get":
			if r.Method != http.MethodPost {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}
			httpExtensionHandler.GetRequest(w, r)
		case "post":
			if r.Method != http.MethodPost {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}
			httpExtensionHandler.PostRequest(w, r)
		default:
			http.Error(w, "Not found", http.StatusNotFound)
		}
		return
	}

	http.Error(w, "Not found", http.StatusNotFound)
}

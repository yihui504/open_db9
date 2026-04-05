package database

import (
	"context"
	"fmt"
	"log"
	"strings"
)

// Extension holds detailed metadata about a database extension
type Extension struct {
	Name         string   `json:"name"`
	Version      string   `json:"version"`
	Description  string   `json:"description"`
	SQLInstall   string   `json:"sql_install"`
	SQLVerify    string   `json:"sql_verify"`
	Dependencies []string `json:"dependencies"`
	Limitations  []string `json:"limitations"`
}

// ExtensionManager defines the interface for extension management
type ExtensionManager interface {
	InstallExtension(ctx context.Context, databaseID, extName string) error
	ListExtensions(ctx context.Context, databaseID string) ([]Extension, error)
	EnsureExtensions(ctx context.Context, databaseID string) error
	VerifyExtension(ctx context.Context, databaseID, extName string) (bool, error)
}

// ExtensionInfo holds information about a database extension
type ExtensionInfo struct {
	Name           string
	Version        string
	Schema         string
	Installed      bool
	DefaultVersion string
	Comment        string
}

// ExtensionsManager handles database extension operations
type ExtensionsManager struct {
	manager *Manager
}

// NewExtensionsManager creates a new extensions manager
func NewExtensionsManager(manager *Manager) *ExtensionsManager {
	return &ExtensionsManager{manager: manager}
}

// List lists all available and installed extensions
func (em *ExtensionsManager) List(ctx context.Context) ([]ExtensionInfo, error) {
	sql := `
		SELECT
			ext.name as name,
			ext.default_version as default_version,
			n.nspname as schema,
			COALESCE(ext.version, ext.default_version) as version,
			ext.installed_version IS NOT NULL as installed,
			ext.comment as comment
		FROM pg_available_extensions ext
		LEFT JOIN pg_namespace n ON n.oid = ext.extnamespace
		ORDER BY ext.name
	`

	rows, err := em.manager.Raw(ctx, sql)
	if err != nil {
		return nil, fmt.Errorf("failed to list extensions: %w", err)
	}
	defer rows.Close()

	var extensions []ExtensionInfo
	for rows.Next() {
		var ext ExtensionInfo
		if err := rows.Scan(
			&ext.Name,
			&ext.DefaultVersion,
			&ext.Schema,
			&ext.Version,
			&ext.Installed,
			&ext.Comment,
		); err != nil {
			return nil, fmt.Errorf("failed to scan extension: %w", err)
		}
		extensions = append(extensions, ext)
	}

	return extensions, nil
}

// Installed returns only installed extensions
func (em *ExtensionsManager) Installed(ctx context.Context) ([]ExtensionInfo, error) {
	sql := `
		SELECT
			ext.name as name,
			ext.version as version,
			n.nspname as schema,
			ext.default_version as default_version,
			ext.comment as comment
		FROM pg_extension ext
		LEFT JOIN pg_namespace n ON n.oid = ext.extnamespace
		ORDER BY ext.name
	`

	rows, err := em.manager.Raw(ctx, sql)
	if err != nil {
		return nil, fmt.Errorf("failed to list installed extensions: %w", err)
	}
	defer rows.Close()

	var extensions []ExtensionInfo
	for rows.Next() {
		var ext ExtensionInfo
		if err := rows.Scan(
			&ext.Name,
			&ext.Version,
			&ext.Schema,
			&ext.DefaultVersion,
			&ext.Comment,
		); err != nil {
			return nil, fmt.Errorf("failed to scan extension: %w", err)
		}
		ext.Installed = true
		extensions = append(extensions, ext)
	}

	return extensions, nil
}

// Install installs a database extension
func (em *ExtensionsManager) Install(ctx context.Context, name string, schema string, cascade bool) error {
	sql := fmt.Sprintf("CREATE EXTENSION IF NOT EXISTS %s", quoteIdentifier(name))

	args := []interface{}{}

	if schema != "" {
		sql += " SCHEMA " + quoteIdentifier(schema)
	}

	if cascade {
		sql += " CASCADE"
	}

	err := em.manager.RawExec(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("failed to install extension %s: %w", name, err)
	}

	return nil
}

// Uninstall removes a database extension
func (em *ExtensionsManager) Uninstall(ctx context.Context, name string, cascade bool) error {
	sql := fmt.Sprintf("DROP EXTENSION IF EXISTS %s", quoteIdentifier(name))

	if cascade {
		sql += " CASCADE"
	}

	err := em.manager.RawExec(ctx, sql)
	if err != nil {
		return fmt.Errorf("failed to uninstall extension %s: %w", name, err)
	}

	return nil
}

// Update updates an extension to a specific version or latest
func (em *ExtensionsManager) Update(ctx context.Context, name string, version string) error {
	sql := fmt.Sprintf("ALTER EXTENSION %s UPDATE", quoteIdentifier(name))

	if version != "" {
		sql += " TO " + quoteIdentifier(version)
	}

	err := em.manager.RawExec(ctx, sql)
	if err != nil {
		return fmt.Errorf("failed to update extension %s: %w", name, err)
	}

	return nil
}

// GetInfo retrieves detailed information about a specific extension
func (em *ExtensionsManager) GetInfo(ctx context.Context, name string) (*ExtensionInfo, error) {
	sql := `
		SELECT
			ext.name as name,
			ext.default_version as default_version,
			n.nspname as schema,
			COALESCE(ext.version, ext.default_version) as version,
			ext.installed_version IS NOT NULL as installed,
			ext.comment as comment
		FROM pg_available_extensions ext
		LEFT JOIN pg_namespace n ON n.oid = ext.extnamespace
		WHERE ext.name = $1
	`

	rows, err := em.manager.Raw(ctx, sql, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get extension info: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, fmt.Errorf("extension %s not found", name)
	}

	var ext ExtensionInfo
	if err := rows.Scan(
		&ext.Name,
		&ext.DefaultVersion,
		&ext.Schema,
		&ext.Version,
		&ext.Installed,
		&ext.Comment,
	); err != nil {
		return nil, fmt.Errorf("failed to scan extension: %w", err)
	}

	return &ext, nil
}

// Common extension installation helpers

// InstallVectorExtensions installs common vector database extensions
func (em *ExtensionsManager) InstallVectorExtensions(ctx context.Context) error {
	extensions := []struct {
		name   string
		schema string
	}{
		{"vector", "public"},
		{"pg_trgm", "public"},
		{"btree_gin", "public"},
		{"btree_gist", "public"},
	}

	for _, ext := range extensions {
		if err := em.Install(ctx, ext.name, ext.schema, false); err != nil {
			// Log but continue with other extensions
			fmt.Printf("Warning: failed to install %s: %v\n", ext.name, err)
		}
	}

	return nil
}

// InstallPostGIS installs PostGIS extension for geospatial data
func (em *ExtensionsManager) InstallPostGIS(ctx context.Context, schema string) error {
	// PostGIS requires topology to be installed first
	if err := em.Install(ctx, "postgis_topology", schema, true); err != nil {
		return fmt.Errorf("failed to install postgis_topology: %w", err)
	}

	if err := em.Install(ctx, "postgis", schema, false); err != nil {
		return fmt.Errorf("failed to install postgis: %w", err)
	}

	return nil
}

// InstallFullTextSearch installs extensions for full-text search
func (em *ExtensionsManager) InstallFullTextSearch(ctx context.Context) error {
	extensions := []string{"pg_trgm", "unaccent", "zhparser"}

	for _, ext := range extensions {
		if err := em.Install(ctx, ext, "public", false); err != nil {
			// Some extensions might not be available, log and continue
			fmt.Printf("Warning: failed to install %s: %v\n", ext, err)
		}
	}

	return nil
}

// InstallTimescaleDB installs TimescaleDB extension for time-series data
func (em *ExtensionsManager) InstallTimescaleDB(ctx context.Context) error {
	return em.Install(ctx, "timescaledb", "public", false)
}

// ValidateExtension checks if an extension is properly installed
func (em *ExtensionsManager) ValidateExtension(ctx context.Context, name string) error {
	info, err := em.GetInfo(ctx, name)
	if err != nil {
		return err
	}

	if !info.Installed {
		return fmt.Errorf("extension %s is not installed", name)
	}

	// Try to use the extension
	sql := fmt.Sprintf("SELECT %s.version()", quoteIdentifier(name))
	rows, err := em.manager.Raw(ctx, sql)
	if err != nil {
		return fmt.Errorf("extension %s is installed but not functional: %w", name, err)
	}
	defer rows.Close()

	return nil
}

// ExtensionVersion returns the version of an installed extension
func (em *ExtensionsManager) ExtensionVersion(ctx context.Context, name string) (string, error) {
	info, err := em.GetInfo(ctx, name)
	if err != nil {
		return "", err
	}

	if !info.Installed {
		return "", fmt.Errorf("extension %s is not installed", name)
	}

	return info.Version, nil
}

// Helper function to safely quote identifiers
func quoteIdentifier(name string) string {
	// Escape existing quotes by doubling them, then wrap in quotes
	// This prevents SQL injection by properly escaping identifier names
	return fmt.Sprintf("\"%s\"", strings.ReplaceAll(name, "\"", "\"\""))
}

// Built-in extension definitions

// builtinExtensions contains the registry of supported extensions
var builtinExtensions = map[string]Extension{
	"vector": {
		Name:        "vector",
		Version:     "0.5.0",
		Description: "Vector data type and ivfflat and HNSW indexes for vector similarity search",
		SQLInstall:  "CREATE EXTENSION IF NOT EXISTS vector",
		SQLVerify:   "SELECT EXISTS(SELECT 1 FROM pg_extension WHERE extname = 'vector')",
		Dependencies: []string{},
		Limitations: []string{
			"Requires PostgreSQL 12+",
			"Vector dimensions must be consistent within a column",
			"HNSW index requires additional memory",
		},
	},
	"pg_cron": {
		Name:        "pg_cron",
		Version:     "1.6",
		Description: "Job scheduler for PostgreSQL that runs periodic jobs",
		SQLInstall:  "CREATE EXTENSION IF NOT EXISTS pg_cron",
		SQLVerify:   "SELECT EXISTS(SELECT 1 FROM pg_extension WHERE extname = 'pg_cron')",
		Dependencies: []string{},
		Limitations: []string{
			"Requires superuser privileges",
			"Jobs run with the permissions of the job creator",
			"Cron schedule uses UTC timezone by default",
		},
	},
	"pg_net": {
		Name:        "pg_net",
		Version:     "0.8",
		Description: "HTTP client for PostgreSQL allowing async HTTP requests",
		SQLInstall:  "CREATE EXTENSION IF NOT EXISTS pg_net",
		SQLVerify:   "SELECT EXISTS(SELECT 1 FROM pg_extension WHERE extname = 'pg_net')",
		Dependencies: []string{},
		Limitations: []string{
			"Requires PostgreSQL 14+",
			"Network requests are asynchronous",
			"Timeouts and retries must be configured",
		},
	},
}

// ExtensionManager implementation

// InstallExtension installs a specific extension by name
func (em *ExtensionsManager) InstallExtension(ctx context.Context, databaseID, extName string) error {
	extDef, exists := builtinExtensions[extName]
	if !exists {
		return fmt.Errorf("extension %s is not a recognized built-in extension", extName)
	}

	// Log installation attempt
	log.Printf("[ExtensionManager] Installing extension '%s' for database '%s'", extName, databaseID)

	// Check if extension is already installed
	verified, err := em.VerifyExtension(ctx, databaseID, extName)
	if err != nil {
		log.Printf("[ExtensionManager] Warning: could not verify extension status: %v", err)
	}
	if verified {
		log.Printf("[ExtensionManager] Extension '%s' is already installed", extName)
		return nil
	}

	// Install dependencies first
	for _, dep := range extDef.Dependencies {
		log.Printf("[ExtensionManager] Installing dependency '%s' for '%s'", dep, extName)
		if err := em.InstallExtension(ctx, databaseID, dep); err != nil {
			return fmt.Errorf("failed to install dependency %s: %w", dep, err)
		}
	}

	// Execute installation SQL
	err = em.manager.RawExec(ctx, extDef.SQLInstall)
	if err != nil {
		log.Printf("[ExtensionManager] Failed to install extension '%s': %v", extName, err)
		return fmt.Errorf("failed to install extension %s: %w", extName, err)
	}

	log.Printf("[ExtensionManager] Successfully installed extension '%s'", extName)
	return nil
}

// ListExtensions returns all available built-in extensions
func (em *ExtensionsManager) ListExtensions(ctx context.Context, databaseID string) ([]Extension, error) {
	extensions := make([]Extension, 0, len(builtinExtensions))

	for name, extDef := range builtinExtensions {
		// Check if extension is installed
		verified, err := em.VerifyExtension(ctx, databaseID, name)
		if err != nil {
			log.Printf("[ExtensionManager] Warning: could not verify extension '%s': %v", name, err)
		}

		// Update version if installed
		if verified {
			if info, err := em.GetInfo(ctx, name); err == nil {
				extDef.Version = info.Version
			}
		}

		extensions = append(extensions, extDef)
	}

	log.Printf("[ExtensionManager] Listed %d extensions for database '%s'", len(extensions), databaseID)
	return extensions, nil
}

// EnsureExtensions ensures all built-in extensions are installed
func (em *ExtensionsManager) EnsureExtensions(ctx context.Context, databaseID string) error {
	log.Printf("[ExtensionManager] Ensuring all built-in extensions for database '%s'", databaseID)

	var installErrors []string
	successCount := 0

	for extName := range builtinExtensions {
		if err := em.InstallExtension(ctx, databaseID, extName); err != nil {
			log.Printf("[ExtensionManager] Failed to ensure extension '%s': %v", extName, err)
			installErrors = append(installErrors, fmt.Sprintf("%s: %v", extName, err))
		} else {
			successCount++
		}
	}

	if len(installErrors) > 0 {
		log.Printf("[ExtensionManager] Completed with %d successes, %d errors", successCount, len(installErrors))
		return fmt.Errorf("failed to install some extensions: %s", strings.Join(installErrors, "; "))
	}

	log.Printf("[ExtensionManager] Successfully ensured all %d built-in extensions", successCount)
	return nil
}

// VerifyExtension checks if a specific extension is installed and functional
func (em *ExtensionsManager) VerifyExtension(ctx context.Context, databaseID, extName string) (bool, error) {
	extDef, exists := builtinExtensions[extName]
	if !exists {
		return false, fmt.Errorf("extension %s is not a recognized built-in extension", extName)
	}

	// Execute verification query
	rows, err := em.manager.Raw(ctx, extDef.SQLVerify)
	if err != nil {
		log.Printf("[ExtensionManager] Verification query failed for '%s': %v", extName, err)
		return false, fmt.Errorf("verification query failed: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		return false, fmt.Errorf("no result from verification query")
	}

	var installed bool
	if err := rows.Scan(&installed); err != nil {
		log.Printf("[ExtensionManager] Failed to scan verification result for '%s': %v", extName, err)
		return false, fmt.Errorf("failed to scan verification result: %w", err)
	}

	log.Printf("[ExtensionManager] Extension '%s' verification: %v", extName, installed)
	return installed, nil
}

// GetBuiltinExtension returns the definition of a built-in extension
func GetBuiltinExtension(name string) (Extension, bool) {
	ext, exists := builtinExtensions[name]
	return ext, exists
}

// ListBuiltinExtensions returns all built-in extension names
func ListBuiltinExtensions() []string {
	names := make([]string, 0, len(builtinExtensions))
	for name := range builtinExtensions {
		names = append(names, name)
	}
	return names
}

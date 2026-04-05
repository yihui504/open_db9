package database

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// MockRows is a mock implementation of pgx.Rows
type MockRows struct {
	values     [][]interface{}
	rowIndex   int
	shouldNext bool
	scanFunc   func(dest ...interface{}) error
	closeFunc  func()
}

func (m *MockRows) Close() {
	if m.closeFunc != nil {
		m.closeFunc()
	}
}

func (m *MockRows) Err() error {
	return nil
}

func (m *MockRows) CommandTag() pgconn.CommandTag {
	return pgconn.CommandTag{}
}

func (m *MockRows) Fields() []pgconn.FieldDescription {
	return nil
}

func (m *MockRows) FieldDescriptions() []pgconn.FieldDescription {
	return nil
}

func (m *MockRows) Next() bool {
	if m.rowIndex >= len(m.values) {
		return false
	}
	result := m.shouldNext && m.rowIndex < len(m.values)
	m.rowIndex++
	return result
}

func (m *MockRows) Scan(dest ...interface{}) error {
	if m.scanFunc != nil {
		return m.scanFunc(dest...)
	}
	if m.rowIndex-1 >= 0 && m.rowIndex-1 < len(m.values) {
		for i, val := range m.values[m.rowIndex-1] {
			if i < len(dest) {
				switch ptr := dest[i].(type) {
				case *string:
					if str, ok := val.(string); ok {
						*ptr = str
					}
				case *bool:
					if b, ok := val.(bool); ok {
						*ptr = b
					}
				case *interface{}:
					*ptr = val
				}
			}
		}
	}
	return nil
}

func (m *MockRows) Values() ([]interface{}, error) {
	if m.rowIndex-1 >= 0 && m.rowIndex-1 < len(m.values) {
		return m.values[m.rowIndex-1], nil
	}
	return nil, errors.New("no row available")
}

func (m *MockRows) RawValues() [][]byte {
	return nil
}

func (m *MockRows) Conn() *pgx.Conn {
	return nil
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	if len(s) < len(substr) {
		return false
	}
	if s == substr {
		return true
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestGetBuiltinExtension tests retrieving built-in extension definitions
func TestGetBuiltinExtension(t *testing.T) {
	tests := []struct {
		name    string
		extName string
		want    Extension
		exists  bool
	}{
		{
			name:    "get vector extension",
			extName: "vector",
			want: Extension{
				Name:        "vector",
				Version:     "0.5.0",
				Description: "Vector data type and ivfflat and HNSW indexes for vector similarity search",
				SQLInstall:  "CREATE EXTENSION IF NOT EXISTS vector",
				SQLVerify:   "SELECT EXISTS(SELECT 1 FROM pg_extension WHERE extname = 'vector')",
			},
			exists: true,
		},
		{
			name:    "get pg_cron extension",
			extName: "pg_cron",
			want: Extension{
				Name:        "pg_cron",
				Version:     "1.6",
				Description: "Job scheduler for PostgreSQL that runs periodic jobs",
				SQLInstall:  "CREATE EXTENSION IF NOT EXISTS pg_cron",
				SQLVerify:   "SELECT EXISTS(SELECT 1 FROM pg_extension WHERE extname = 'pg_cron')",
			},
			exists: true,
		},
		{
			name:    "get pg_net extension",
			extName: "pg_net",
			want: Extension{
				Name:        "pg_net",
				Version:     "0.8",
				Description: "HTTP client for PostgreSQL allowing async HTTP requests",
				SQLInstall:  "CREATE EXTENSION IF NOT EXISTS pg_net",
				SQLVerify:   "SELECT EXISTS(SELECT 1 FROM pg_extension WHERE extname = 'pg_net')",
			},
			exists: true,
		},
		{
			name:    "get non-existent extension",
			extName: "nonexistent",
			want:    Extension{},
			exists:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, exists := GetBuiltinExtension(tt.extName)

			if exists != tt.exists {
				t.Errorf("GetBuiltinExtension() exists = %v, want %v", exists, tt.exists)
			}

			if tt.exists {
				if got.Name != tt.want.Name {
					t.Errorf("GetBuiltinExtension() Name = %v, want %v", got.Name, tt.want.Name)
				}
				if got.Version != tt.want.Version {
					t.Errorf("GetBuiltinExtension() Version = %v, want %v", got.Version, tt.want.Version)
				}
				if got.Description != tt.want.Description {
					t.Errorf("GetBuiltinExtension() Description = %v, want %v", got.Description, tt.want.Description)
				}
				if got.SQLInstall != tt.want.SQLInstall {
					t.Errorf("GetBuiltinExtension() SQLInstall = %v, want %v", got.SQLInstall, tt.want.SQLInstall)
				}
				if got.SQLVerify != tt.want.SQLVerify {
					t.Errorf("GetBuiltinExtension() SQLVerify = %v, want %v", got.SQLVerify, tt.want.SQLVerify)
				}
			}
		})
	}
}

// TestListBuiltinExtensions tests listing all built-in extensions
func TestListBuiltinExtensions(t *testing.T) {
	extensions := ListBuiltinExtensions()

	if len(extensions) == 0 {
		t.Error("ListBuiltinExtensions() returned empty list")
	}

	expectedExtensions := []string{"vector", "pg_cron", "pg_net"}
	for _, expected := range expectedExtensions {
		found := false
		for _, ext := range extensions {
			if ext == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("ListBuiltinExtensions() missing expected extension %q", expected)
		}
	}

	// Check for duplicates
	seen := make(map[string]bool)
	for _, ext := range extensions {
		if seen[ext] {
			t.Errorf("ListBuiltinExtensions() returned duplicate extension %q", ext)
		}
		seen[ext] = true
	}
}

// TestExtensionManager_UnknownExtension tests error handling for unknown extensions
func TestExtensionManager_UnknownExtension(t *testing.T) {
	ctx := context.Background()

	// Create a minimal manager - we're just testing error handling
	config := DefaultPoolConfig()
	config.MaxConnections = 1

	pool, err := NewConnectionPool(config)
	if err != nil {
		t.Skipf("Skipping test: no database available: %v", err)
		return
	}
	defer pool.Close()

	manager := ManagerFromPool(pool)
	em := NewExtensionsManager(manager)

	// Test InstallExtension with unknown extension
	err = em.InstallExtension(ctx, "testdb", "unknown_extension")
	if err == nil {
		t.Error("InstallExtension() expected error for unknown extension, got nil")
	} else if !contains(err.Error(), "not a recognized built-in extension") {
		t.Errorf("InstallExtension() error = %v, want error containing 'not a recognized built-in extension'", err)
	}

	// Test VerifyExtension with unknown extension
	_, err = em.VerifyExtension(ctx, "testdb", "unknown_extension")
	if err == nil {
		t.Error("VerifyExtension() expected error for unknown extension, got nil")
	} else if !contains(err.Error(), "not a recognized built-in extension") {
		t.Errorf("VerifyExtension() error = %v, want error containing 'not a recognized built-in extension'", err)
	}
}

// TestExtensionManager_ListBuiltin tests listing built-in extensions (without DB)
func TestExtensionManager_ListBuiltin(t *testing.T) {
	ctx := context.Background()

	// This test will fail without a database, but we can test the structure
	config := DefaultPoolConfig()
	config.MaxConnections = 1

	pool, err := NewConnectionPool(config)
	if err != nil {
		t.Skipf("Skipping test: no database available: %v", err)
		return
	}
	defer pool.Close()

	manager := ManagerFromPool(pool)
	em := NewExtensionsManager(manager)

	// Test ListExtensions - this will fail without actual extensions but tests the code path
	_, err = em.ListExtensions(ctx, "testdb")
	if err != nil {
		t.Logf("ListExtensions() failed as expected without database: %v", err)
	}
}

// TestQuoteIdentifier tests the quoteIdentifier helper function
func TestQuoteIdentifier(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple identifier",
			input:    "vector",
			expected: `"vector"`,
		},
		{
			name:     "identifier with space",
			input:    "my extension",
			expected: `"my extension"`,
		},
		{
			name:     "identifier with quote",
			input:    `my"extension`,
			expected: `"my""extension"`,
		},
		{
			name:     "identifier with multiple quotes",
			input:    `my""extension`,
			expected: `"my""""extension"`,
		},
		{
			name:     "empty identifier",
			input:    "",
			expected: `""`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := quoteIdentifier(tt.input)
			if result != tt.expected {
				t.Errorf("quoteIdentifier(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestExtensionDependencies tests that extension definitions have proper dependency info
func TestExtensionDependencies(t *testing.T) {
	// Test that vector extension has no dependencies
	vector, exists := GetBuiltinExtension("vector")
	if !exists {
		t.Fatal("vector extension should exist")
	}
	if len(vector.Dependencies) != 0 {
		t.Errorf("vector extension should have no dependencies, got %v", vector.Dependencies)
	}

	// Test that pg_cron extension has no dependencies
	pgCron, exists := GetBuiltinExtension("pg_cron")
	if !exists {
		t.Fatal("pg_cron extension should exist")
	}
	if len(pgCron.Dependencies) != 0 {
		t.Errorf("pg_cron extension should have no dependencies, got %v", pgCron.Dependencies)
	}

	// Test that pg_net extension has no dependencies
	pgNet, exists := GetBuiltinExtension("pg_net")
	if !exists {
		t.Fatal("pg_net extension should exist")
	}
	if len(pgNet.Dependencies) != 0 {
		t.Errorf("pg_net extension should have no dependencies, got %v", pgNet.Dependencies)
	}
}

// TestExtensionSQLTests tests that extension definitions have proper SQL
func TestExtensionSQLTests(t *testing.T) {
	tests := []struct {
		name          string
		extName       string
		hasInstall    bool
		hasVerify     bool
	}{
		{
			name:       "vector extension SQL",
			extName:    "vector",
			hasInstall: true,
			hasVerify:  true,
		},
		{
			name:       "pg_cron extension SQL",
			extName:    "pg_cron",
			hasInstall: true,
			hasVerify:  true,
		},
		{
			name:       "pg_net extension SQL",
			extName:    "pg_net",
			hasInstall: true,
			hasVerify:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ext, exists := GetBuiltinExtension(tt.extName)
			if !exists {
				t.Fatalf("%s should exist", tt.extName)
			}

			if tt.hasInstall && ext.SQLInstall == "" {
				t.Errorf("%s should have SQLInstall", tt.extName)
			}

			if tt.hasVerify && ext.SQLVerify == "" {
				t.Errorf("%s should have SQLVerify", tt.extName)
			}

			// Verify SQLInstall contains CREATE EXTENSION
			if tt.hasInstall && !contains(ext.SQLInstall, "CREATE EXTENSION") {
				t.Errorf("%s SQLInstall should contain 'CREATE EXTENSION', got %s", tt.extName, ext.SQLInstall)
			}

			// Verify SQLVerify contains SELECT
			if tt.hasVerify && !contains(ext.SQLVerify, "SELECT") {
				t.Errorf("%s SQLVerify should contain 'SELECT', got %s", tt.extName, ext.SQLVerify)
			}
		})
	}
}

// TestExtensionLimitations tests that extension definitions have proper limitations
func TestExtensionLimitations(t *testing.T) {
	// Test that extensions have limitations documented
	tests := []struct {
		name               string
		extName            string
		minLimitations     int
	}{
		{
			name:           "vector limitations",
			extName:        "vector",
			minLimitations: 1,
		},
		{
			name:           "pg_cron limitations",
			extName:        "pg_cron",
			minLimitations: 1,
		},
		{
			name:           "pg_net limitations",
			extName:        "pg_net",
			minLimitations: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ext, exists := GetBuiltinExtension(tt.extName)
			if !exists {
				t.Fatalf("%s should exist", tt.extName)
			}

			if len(ext.Limitations) < tt.minLimitations {
				t.Errorf("%s should have at least %d limitations, got %d", tt.extName, tt.minLimitations, len(ext.Limitations))
			}
		})
	}
}

// TestExtensionValidation tests extension validation logic
func TestExtensionValidation(t *testing.T) {
	tests := []struct {
		name       string
		extName    string
		shouldPass bool
	}{
		{
			name:       "valid vector extension",
			extName:    "vector",
			shouldPass: true,
		},
		{
			name:       "valid pg_cron extension",
			extName:    "pg_cron",
			shouldPass: true,
		},
		{
			name:       "valid pg_net extension",
			extName:    "pg_net",
			shouldPass: true,
		},
		{
			name:       "invalid extension",
			extName:    "invalid",
			shouldPass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, exists := GetBuiltinExtension(tt.extName)
			if exists != tt.shouldPass {
				t.Errorf("GetBuiltinExtension(%s) exists = %v, want %v", tt.extName, exists, tt.shouldPass)
			}
		})
	}
}

// TestExtensionConsistency tests that extension definitions are consistent
func TestExtensionConsistency(t *testing.T) {
	extensions := ListBuiltinExtensions()

	for _, extName := range extensions {
		t.Run(extName+" consistency", func(t *testing.T) {
			ext, exists := GetBuiltinExtension(extName)
			if !exists {
				t.Errorf("%s should exist in builtinExtensions", extName)
			}

			// Check that name matches
			if ext.Name != extName {
				t.Errorf("%s: Name = %s, want %s", extName, ext.Name, extName)
			}

			// Check that version is not empty
			if ext.Version == "" {
				t.Errorf("%s: Version should not be empty", extName)
			}

			// Check that description is not empty
			if ext.Description == "" {
				t.Errorf("%s: Description should not be empty", extName)
			}

			// Check that SQLInstall is not empty
			if ext.SQLInstall == "" {
				t.Errorf("%s: SQLInstall should not be empty", extName)
			}

			// Check that SQLVerify is not empty
			if ext.SQLVerify == "" {
				t.Errorf("%s: SQLVerify should not be empty", extName)
			}

			// Check that Dependencies is not nil
			if ext.Dependencies == nil {
				t.Errorf("%s: Dependencies should not be nil", extName)
			}

			// Check that Limitations is not nil
			if ext.Limitations == nil {
				t.Errorf("%s: Limitations should not be nil", extName)
			}
		})
	}
}

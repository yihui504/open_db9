package python

import (
	"context"
	"testing"
)

func TestParseTableName(t *testing.T) {
	tests := []struct {
		name    string
		schema  string
		want    string
		wantErr bool
	}{
		{
			name:    "simple table name",
			schema:  "users",
			want:    "users",
			wantErr: false,
		},
		{
			name:    "schema qualified table",
			schema:  "public.users",
			want:    "users",
			wantErr: false,
		},
		{
			name:    "schema qualified with custom schema",
			schema:  "my_schema.products",
			want:    "products",
			wantErr: false,
		},
		{
			name:    "empty schema",
			schema:  "",
			want:    "",
			wantErr: true,
		},
		{
			name:    "too many dots",
			schema:  "schema.table.extra",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseTableName(tt.schema)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseTableName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseTableName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestToPascalCase(t *testing.T) {
	tests := []struct {
		name string
		input string
		want string
	}{
		{
			name:  "simple word",
			input: "user",
			want:  "User",
		},
		{
			name:  "snake case single underscore",
			input: "user_profile",
			want:  "UserProfile",
		},
		{
			name:  "snake case multiple underscores",
			input: "order_item_details",
			want:  "OrderItemDetails",
		},
		{
			name:  "with leading underscore",
			input: "_private_field",
			want:  "PrivateField",
		},
		{
			name:  "multiple consecutive underscores",
			input: "test__multiple__underscores",
			want:  "TestMultipleUnderscores",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := toPascalCase(tt.input); got != tt.want {
				t.Errorf("toPascalCase() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMapPostgresTypeToPython(t *testing.T) {
	tests := []struct {
		name     string
		pgType   string
		expected string
	}{
		// String types
		{"varchar", "varchar", "str"},
		{"VARCHAR", "VARCHAR", "str"},
		{"varchar with length", "VARCHAR(255)", "str"},
		{"text", "text", "str"},
		{"char", "CHAR(10)", "str"},

		// Integer types
		{"integer", "INTEGER", "int"},
		{"int", "INT", "int"},
		{"bigint", "BIGINT", "int"},
		{"smallint", "SMALLINT", "int"},
		{"serial", "SERIAL", "int"},

		// Boolean
		{"boolean", "BOOLEAN", "bool"},
		{"bool", "BOOL", "bool"},

		// Date/Time
		{"timestamp", "TIMESTAMP", "datetime"},
		{"timestamptz", "TIMESTAMPTZ", "datetime"},
		{"date", "DATE", "datetime"},

		// JSON
		{"json", "JSON", "dict[str, Any]"},
		{"jsonb", "JSONB", "dict[str, Any]"},

		// UUID
		{"uuid", "UUID", "str"},

		// Decimal
		{"numeric", "NUMERIC", "float"},
		{"real", "REAL", "float"},

		// Unknown types
		{"unknown", "custom_type", "Any"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mapPostgresTypeToPython(tt.pgType); got != tt.expected {
				t.Errorf("mapPostgresTypeToPython(%q) = %v, want %v", tt.pgType, got, tt.expected)
			}
		})
	}
}

func TestPythonGenerator_Generate(t *testing.T) {
	ctx := context.Background()
	g := NewPythonGenerator()

	tests := []struct {
		name       string
		databaseID string
		schema     string
		wantErr    bool
	}{
		{
			name:       "simple table",
			databaseID: "1",
			schema:     "users",
			wantErr:    false,
		},
		{
			name:       "schema qualified",
			databaseID: "1",
			schema:     "public.users",
			wantErr:    false,
		},
		{
			name:       "empty schema",
			databaseID: "1",
			schema:     "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := g.Generate(ctx, tt.databaseID, tt.schema)
			if (err != nil) != tt.wantErr {
				t.Errorf("Generate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got == "" {
				t.Error("Generate() returned empty string")
			}
		})
	}
}

func TestPythonGenerator_Validate(t *testing.T) {
	ctx := context.Background()
	g := NewPythonGenerator()

	tests := []struct {
		name       string
		databaseID string
		schema     string
		wantErr    bool
		wantErrors int
	}{
		{
			name:       "valid schema",
			databaseID: "1",
			schema:     "users",
			wantErr:    false,
			wantErrors: 0,
		},
		{
			name:       "schema qualified",
			databaseID: "1",
			schema:     "public.users",
			wantErr:    false,
			wantErrors: 0,
		},
		{
			name:       "empty schema",
			databaseID: "1",
			schema:     "",
			wantErr:    false,
			wantErrors: 1,
		},
		{
			name:       "invalid characters",
			databaseID: "1",
			schema:     "users-table",
			wantErr:    false,
			wantErrors: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors, err := g.Validate(ctx, tt.databaseID, tt.schema)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(errors) != tt.wantErrors {
				t.Errorf("Validate() got %d errors, want %d", len(errors), tt.wantErrors)
			}
		})
	}
}

func TestPythonGenerator_GenerateFromSchema(t *testing.T) {
	ctx := context.Background()
	g := NewPythonGenerator()

	columns := []ColumnInfo{
		{Name: "id", Type: "INTEGER", Nullable: false},
		{Name: "username", Type: "VARCHAR", Nullable: false},
		{Name: "email", Type: "VARCHAR(255)", Nullable: true},
		{Name: "is_active", Type: "BOOLEAN", Nullable: false},
		{Name: "created_at", Type: "TIMESTAMP", Nullable: true},
		{Name: "metadata", Type: "JSONB", Nullable: true},
	}

	code, err := g.GenerateFromSchema(ctx, "users", columns)
	if err != nil {
		t.Fatalf("GenerateFromSchema() error = %v", err)
	}

	// Check that the generated code contains expected elements
	expectedSubstrings := []string{
		"class Users:",
		"id: int",
		"username: str",
		"email: Optional[str]",
		"is_active: bool",
		"created_at: Optional[datetime]",
		"metadata: Optional[dict[str, Any]]",
		"@dataclass",
		"from dataclasses import dataclass",
	}

	for _, expected := range expectedSubstrings {
		if !contains(code, expected) {
			t.Errorf("GenerateFromSchema() output missing %q", expected)
		}
	}
}

func TestMapPostgresTypeToPython_Arrays(t *testing.T) {
	tests := []struct {
		pgType   string
		expected string
	}{
		{"_INT4", "list"},
		{"VARCHAR[]", "list"},
		{"INTEGER ARRAY", "list"},
	}

	for _, tt := range tests {
		t.Run(tt.pgType, func(t *testing.T) {
			if got := mapPostgresTypeToPython(tt.pgType); got != tt.expected {
				t.Errorf("mapPostgresTypeToPython(%q) = %v, want %v", tt.pgType, got, tt.expected)
			}
		})
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || contains(s[1:], substr)))
}

package python

import (
	"context"
	"fmt"
	"strings"
)

// ValidationError represents a validation error for schema generation
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// PythonGenerator generates Python dataclass types from database schemas
type PythonGenerator struct{}

// NewPythonGenerator creates a new Python generator instance
func NewPythonGenerator() *PythonGenerator {
	return &PythonGenerator{}
}

// Generate creates Python dataclass code from a database schema
func (g *PythonGenerator) Generate(ctx context.Context, databaseID, schema string) (string, error) {
	// Parse the schema (expected format: table_name or schema.table_name)
	tableName, err := parseTableName(schema)
	if err != nil {
		return "", fmt.Errorf("failed to parse table name: %w", err)
	}

	// In a real implementation, we would query the database schema here
	// For now, we'll generate a template based on the table name
	className := toPascalCase(tableName)

	// Generate the Python dataclass code
	code := g.generateDataclass(className, tableName)

	return code, nil
}

// Validate checks if the schema is valid for Python generation
func (g *PythonGenerator) Validate(ctx context.Context, databaseID, schema string) ([]ValidationError, error) {
	var errors []ValidationError

	// Parse the schema
	tableName, err := parseTableName(schema)
	if err != nil {
		errors = append(errors, ValidationError{
			Field:   "schema",
			Message: err.Error(),
		})
		return errors, nil
	}

	// Validate table name format
	if tableName == "" {
		errors = append(errors, ValidationError{
			Field:   "table_name",
			Message: "table name cannot be empty",
		})
	}

	// Check for invalid characters
	if strings.ContainsAny(tableName, " -;") {
		errors = append(errors, ValidationError{
			Field:   "table_name",
			Message: "table name contains invalid characters",
		})
	}

	return errors, nil
}

// generateDataclass creates Python dataclass code with type annotations
func (g *PythonGenerator) generateDataclass(className, tableName string) string {
	var sb strings.Builder

	// Write imports
	sb.WriteString("# Auto-generated Python dataclass for ")
	sb.WriteString(tableName)
	sb.WriteString("\n")
	sb.WriteString("from dataclasses import dataclass\n")
	sb.WriteString("from datetime import datetime\n")
	sb.WriteString("from typing import Any, Optional\n\n")

	// Write dataclass decorator
	sb.WriteString("@dataclass\n")
	sb.WriteString("class ")
	sb.WriteString(className)
	sb.WriteString(":\n")
	sb.WriteString("    \"\"\"")
	sb.WriteString(className)
	sb.WriteString(" represents the ")
	sb.WriteString(tableName)
	sb.WriteString(" table\"\"\"\n\n")

	// Write common fields (these would come from actual schema introspection)
	sb.WriteString("    id: int\n")
	sb.WriteString("    created_at: Optional[datetime] = None\n")
	sb.WriteString("    updated_at: Optional[datetime] = None\n")

	return sb.String()
}

// ColumnInfo represents database column information
type ColumnInfo struct {
	Name     string
	Type     string
	Nullable bool
}

// GenerateFromSchema generates Python dataclass from actual column schema
func (g *PythonGenerator) GenerateFromSchema(ctx context.Context, tableName string, columns []ColumnInfo) (string, error) {
	className := toPascalCase(tableName)
	var sb strings.Builder

	// Write imports
	sb.WriteString(fmt.Sprintf("# Auto-generated Python dataclass for %s\n", tableName))
	sb.WriteString("from dataclasses import dataclass\n")
	sb.WriteString("from datetime import datetime\n")
	sb.WriteString("from typing import Any, Optional\n\n")

	// Write dataclass decorator
	sb.WriteString("@dataclass\n")
	sb.WriteString(fmt.Sprintf("class %s:\n", className))
	sb.WriteString(fmt.Sprintf("    \"\"\"%s represents the %s table\"\"\"\n\n", className, tableName))

	// Write fields
	for _, col := range columns {
		pythonType := mapPostgresTypeToPython(col.Type)
		fieldDef := fmt.Sprintf("    %s: ", col.Name)

		if col.Nullable {
			fieldDef += fmt.Sprintf("Optional[%s]", pythonType)
		} else {
			fieldDef += pythonType
		}

		if col.Nullable {
			fieldDef += " = None"
		}

		sb.WriteString(fieldDef)
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

// mapPostgresTypeToPython maps PostgreSQL types to Python types
func mapPostgresTypeToPython(pgType string) string {
	upperType := strings.ToUpper(pgType)

	typeMappings := map[string]string{
		// String types
		"VARCHAR": "str",
		"CHAR":    "str",
		"TEXT":    "str",
		"NAME":    "str",

		// Integer types
		"INT":                "int",
		"INT2":               "int",
		"INT4":               "int",
		"INT8":               "int",
		"INTEGER":            "int",
		"SMALLINT":           "int",
		"BIGINT":             "int",
		"SERIAL":             "int",
		"BIGSERIAL":          "int",
		"SMALLSERIAL":        "int",

		// Boolean
		"BOOLEAN": "bool",
		"BOOL":    "bool",

		// Decimal/Numeric
		"DECIMAL": "float",
		"NUMERIC": "float",
		"REAL":    "float",
		"FLOAT4":  "float",
		"FLOAT8":  "float",
		"DOUBLE":  "float",

		// Date/Time
		"DATE":                  "datetime",
		"TIME":                  "datetime",
		"TIMESTAMP":             "datetime",
		"TIMESTAMPTZ":           "datetime",
		"DATETIME":              "datetime",

		// JSON
		"JSON":   "dict[str, Any]",
		"JSONB":  "dict[str, Any]",

		// UUID
		"UUID": "str",

		// Binary
		"BYTEA": "bytes",

		// Array
		"ARRAY": "list",
	}

	// Check for array types
	if strings.HasPrefix(upperType, "_") || strings.Contains(upperType, "[]") || strings.Contains(upperType, " ARRAY") {
		return "list"
	}

	// Look up exact match
	if pythonType, ok := typeMappings[upperType]; ok {
		return pythonType
	}

	// Handle type with parameters (e.g., VARCHAR(255))
	baseType := upperType
	if idx := strings.Index(baseType, "("); idx != -1 {
		baseType = baseType[:idx]
	}

	if pythonType, ok := typeMappings[baseType]; ok {
		return pythonType
	}

	// Default to Any for unknown types
	return "Any"
}

// parseTableName extracts table name from schema string
func parseTableName(schema string) (string, error) {
	schema = strings.TrimSpace(schema)
	if schema == "" {
		return "", fmt.Errorf("schema cannot be empty")
	}

	// Handle schema.table format
	parts := strings.Split(schema, ".")
	if len(parts) > 2 {
		return "", fmt.Errorf("invalid schema format: %s", schema)
	}

	// Return the last part (table name)
	tableName := parts[len(parts)-1]
	return tableName, nil
}

// toPascalCase converts snake_case to PascalCase
func toPascalCase(s string) string {
	parts := strings.Split(s, "_")
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
	}
	return strings.Join(parts, "")
}

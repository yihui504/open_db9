package typescript

import (
	"context"
	"fmt"
	"strings"
)

// ValidationError represents a schema validation error
type ValidationError struct {
	Table   string `json:"table"`
	Column  string `json:"column,omitempty"`
	Message string `json:"message"`
}

// TypeScriptGenerator generates TypeScript types from database schema
type TypeScriptGenerator struct {
	// db holds the database manager for querying schema information
	// This is intentionally left as an interface to allow dependency injection
	db SchemaReader
}

// SchemaReader defines the interface for reading database schema information
type SchemaReader interface {
	GetTables(ctx context.Context, schema string) ([]string, error)
	GetColumns(ctx context.Context, schema, table string) ([]Column, error)
}

// Column represents a database column definition
type Column struct {
	Name     string
	Type     string
	Nullable bool
}

// NewTypeScriptGenerator creates a new TypeScript generator
func NewTypeScriptGenerator(db SchemaReader) *TypeScriptGenerator {
	return &TypeScriptGenerator{
		db: db,
	}
}

// Generate generates TypeScript type definitions from database schema
func (g *TypeScriptGenerator) Generate(ctx context.Context, databaseID, schema string) (string, error) {
	tables, err := g.db.GetTables(ctx, schema)
	if err != nil {
		return "", fmt.Errorf("failed to get tables: %w", err)
	}

	if len(tables) == 0 {
		return "", fmt.Errorf("no tables found in schema %s", schema)
	}

	var builder strings.Builder

	// Add file header comment
	builder.WriteString("// Auto-generated TypeScript types from database schema\n")
	builder.WriteString(fmt.Sprintf("// Database: %s, Schema: %s\n", databaseID, schema))
	builder.WriteString("\n")

	// Generate interface for each table
	for _, table := range tables {
		interfaceDef, err := g.generateTableInterface(ctx, schema, table)
		if err != nil {
			return "", fmt.Errorf("failed to generate interface for table %s: %w", table, err)
		}
		builder.WriteString(interfaceDef)
		builder.WriteString("\n")
	}

	return builder.String(), nil
}

// generateTableInterface generates a TypeScript interface for a single table
func (g *TypeScriptGenerator) generateTableInterface(ctx context.Context, schema, table string) (string, error) {
	columns, err := g.db.GetColumns(ctx, schema, table)
	if err != nil {
		return "", fmt.Errorf("failed to get columns: %w", err)
	}

	var builder strings.Builder

	// Interface declaration with proper naming convention
	interfaceName := toPascalCase(table)
	builder.WriteString(fmt.Sprintf("// %s represents the %s table\n", interfaceName, table))
	builder.WriteString(fmt.Sprintf("export interface %s {\n", interfaceName))

	// Generate fields for each column
	for i, col := range columns {
		tsType := mapPostgresTypeToTypeScript(col.Type)
		optional := ""
		if col.Nullable {
			optional = " | null"
		}

		// Add field with comment
		builder.WriteString(fmt.Sprintf("  /** %s column (%s) */\n", col.Name, col.Type))
		builder.WriteString(fmt.Sprintf("  %s%s:%s%s;\n", toCamelCase(col.Name), optional, tsType, optional))

		// Add comma between fields (except last)
		if i < len(columns)-1 {
			builder.WriteString("\n")
		}
	}

	builder.WriteString("}\n")

	return builder.String(), nil
}

// Validate validates the database schema and returns any errors
func (g *TypeScriptGenerator) Validate(ctx context.Context, databaseID, schema string) ([]ValidationError, error) {
	var errors []ValidationError

	tables, err := g.db.GetTables(ctx, schema)
	if err != nil {
		return nil, fmt.Errorf("failed to get tables: %w", err)
	}

	if len(tables) == 0 {
		errors = append(errors, ValidationError{
			Table:   schema,
			Message: fmt.Sprintf("no tables found in schema %s", schema),
		})
		return errors, nil
	}

	// Validate each table
	for _, table := range tables {
		columns, err := g.db.GetColumns(ctx, schema, table)
		if err != nil {
			errors = append(errors, ValidationError{
				Table:   table,
				Message: fmt.Sprintf("failed to read columns: %v", err),
			})
			continue
		}

		if len(columns) == 0 {
			errors = append(errors, ValidationError{
				Table:   table,
				Message: "table has no columns",
			})
		}

		// Validate column types
		for _, col := range columns {
			tsType := mapPostgresTypeToTypeScript(col.Type)
			if tsType == "unknown" {
				errors = append(errors, ValidationError{
					Table:   table,
					Column:  col.Name,
					Message: fmt.Sprintf("unsupported PostgreSQL type: %s", col.Type),
				})
			}
		}
	}

	return errors, nil
}

// mapPostgresTypeToTypeScript maps PostgreSQL data types to TypeScript types
func mapPostgresTypeToTypeScript(pgType string) string {
	// Normalize type name (lowercase, remove schema prefix if present)
	typeName := strings.ToLower(pgType)
	if idx := strings.Index(typeName, "."); idx != -1 {
		typeName = typeName[idx+1:]
	}

	// Remove array suffix if present
	typeName = strings.TrimSuffix(typeName, "[]")

	// Map PostgreSQL types to TypeScript types
	switch typeName {
	case "varchar", "text", "char", "character", "bpchar", "name", "citext", "uuid", "inet", "cidr", "macaddr":
		return "string"
	case "int", "int2", "int4", "integer", "smallint", "serial", "serial2", "serial4", "oid":
		return "number"
	case "int8", "bigint", "bigserial":
		return "bigint"
	case "numeric", "decimal":
		return "number"
	case "float4", "real":
		return "number"
	case "float8", "double precision":
		return "number"
	case "bool", "boolean":
		return "boolean"
	case "timestamp", "timestamptz", "date", "time", "timetz":
		return "Date"
	case "json", "jsonb":
		return "any"
	case "bytea":
		return "Buffer"
	case "array":
		return "any[]"
	case "hstore":
		return "Record<string, string>"
	case "point", "line", "lseg", "box", "path", "polygon", "circle":
		return "any" // Geometric types - would need custom mapping
	case "macaddr8":
		return "string"
	case "bit", "varbit":
		return "string"
	case "tsvector", "tsquery":
		return "string"
	case "xml":
		return "string"
	case "money":
		return "string" // PostgreSQL money is represented as string
	default:
		return "unknown"
	}
}

// toPascalCase converts a string to PascalCase
func toPascalCase(s string) string {
	// Split by underscore, dash, space, or camelCase boundaries
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == '_' || r == '-' || r == ' ' || r == '.'
	})

	var result string
	for _, part := range parts {
		if len(part) > 0 {
			result += strings.ToUpper(string(part[0])) + strings.ToLower(part[1:])
		}
	}

	// Handle camelCase input
	if len(result) > 0 && len(s) > 0 {
		// If original was already camelCase or PascalCase, preserve it better
		hasLower := false
		for i := 1; i < len(s); i++ {
			if s[i] >= 'a' && s[i] <= 'z' {
				hasLower = true
				break
			}
		}
		if hasLower && len(s) > 0 && s[0] >= 'a' && s[0] <= 'z' {
			return strings.ToUpper(string(s[0])) + s[1:]
		}
	}

	return result
}

// toCamelCase converts a string to camelCase
func toCamelCase(s string) string {
	pascal := toPascalCase(s)
	if len(pascal) == 0 {
		return ""
	}
	return strings.ToLower(string(pascal[0])) + pascal[1:]
}

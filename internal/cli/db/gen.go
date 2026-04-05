package db

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/jackc/pgx/v5"
	"github.com/spf13/cobra"
)

var (
	genLang      string
	genSchema    string
	genOutput    string
	genValidate  bool
)

// genCmd represents the gen command
var genCmd = &cobra.Command{
	Use:   "gen <database-id>",
	Short: "Generate type definitions from database schema",
	Long: `Generate type definitions for TypeScript or Python from database schema.

This command analyzes the database schema and generates type-safe definitions
that match your table structures, columns, and data types.

Supported languages:
  - typescript: Generate TypeScript interfaces and types
  - python: Generate Python dataclasses and type hints

Examples:
  db9 gen types mydb --lang typescript
  db9 gen types mydb --lang python -o types.py
  db9 gen types mydb --lang typescript --schema my_schema
  db9 gen types mydb --validate`,
	Args: cobra.ExactArgs(1),
	RunE: runGen,
}

// genTypesCmd represents the gen types command
var genTypesCmd = &cobra.Command{
	Use:   "types <database-id>",
	Short: "Generate type definitions from database schema",
	Long: `Generate TypeScript or Python type definitions from database schema.

The generated types will match your table structures including:
  - Table names as type names
  - Column names as fields
  - Proper data type mapping
  - Nullable field handling

Options:
  --lang       Target language (typescript, python)
  --schema     Schema to introspect (default: public)
  -o, --output Output file (default: stdout)
  --validate   Validate types without generating code`,
	Args: cobra.ExactArgs(1),
	RunE: runGenTypes,
}

func init() {
	// Add gen command to db command
	Cmd.AddCommand(genCmd)

	// Add types subcommand
	genCmd.AddCommand(genTypesCmd)

	// Command flags
	genTypesCmd.Flags().StringVar(&genLang, "lang", "",
		"Target language (typescript, python)")

	genTypesCmd.Flags().StringVar(&genSchema, "schema", "public",
		"Schema to introspect (default: public)")

	genTypesCmd.Flags().StringVarP(&genOutput, "output", "o", "",
		"Output file (default: stdout)")

	genTypesCmd.Flags().BoolVar(&genValidate, "validate", false,
		"Validate types without generating code")

	// Mark required flags
	genTypesCmd.MarkFlagRequired("lang")
}

// runGen executes the gen command
func runGen(cmd *cobra.Command, args []string) error {
	return cmd.Help()
}

// runGenTypes executes the gen types command
func runGenTypes(cmd *cobra.Command, args []string) error {
	databaseID := args[0]

	// Validate language
	if genLang != "typescript" && genLang != "python" {
		return fmt.Errorf("invalid language '%s'. Must be 'typescript' or 'python'", genLang)
	}

	// Get database connection
	ctx := context.Background()
	conn, err := getDBConnection(ctx, databaseID)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)

	// Fetch schema information
	tables, err := fetchSchemaTables(ctx, conn, genSchema)
	if err != nil {
		return fmt.Errorf("failed to fetch schema: %w", err)
	}

	if len(tables) == 0 {
		return fmt.Errorf("no tables found in schema '%s'", genSchema)
	}

	// Validate or generate
	if genValidate {
		return validateTypes(tables)
	}

	// Generate type definitions
	var output string
	switch genLang {
	case "typescript":
		output, err = generateTypeScript(tables)
	case "python":
		output, err = generatePython(tables)
	default:
		return fmt.Errorf("unsupported language: %s", genLang)
	}

	if err != nil {
		return err
	}

	// Write output
	if genOutput != "" {
		if err := os.WriteFile(genOutput, []byte(output), 0644); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Generated types written to %s\n", genOutput)
	} else {
		fmt.Println(output)
	}

	return nil
}

// Table represents a database table structure
type Table struct {
	Name    string
	Columns []Column
}

// Column represents a database column
type Column struct {
	Name     string
	Type     string
	Nullable bool
}

// fetchSchemaTables fetches all tables and their columns from the schema
func fetchSchemaTables(ctx context.Context, conn *pgx.Conn, schema string) ([]Table, error) {
	// Query to get all tables in the schema
	tableQuery := `
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = $1
		AND table_type = 'BASE TABLE'
		ORDER BY table_name
	`

	rows, err := conn.Query(ctx, tableQuery, schema)
	if err != nil {
		return nil, fmt.Errorf("failed to query tables: %w", err)
	}
	defer rows.Close()

	var tables []Table
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return nil, fmt.Errorf("failed to scan table name: %w", err)
		}

		// Get columns for this table
		columns, err := fetchTableColumns(ctx, conn, schema, tableName)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch columns for %s: %w", tableName, err)
		}

		tables = append(tables, Table{
			Name:    tableName,
			Columns: columns,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating tables: %w", err)
	}

	return tables, nil
}

// fetchTableColumns fetches all columns for a specific table
func fetchTableColumns(ctx context.Context, conn *pgx.Conn, schema, tableName string) ([]Column, error) {
	columnQuery := `
		SELECT
			column_name,
			data_type,
			is_nullable
		FROM information_schema.columns
		WHERE table_schema = $1
		AND table_name = $2
		ORDER BY ordinal_position
	`

	rows, err := conn.Query(ctx, columnQuery, schema, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to query columns: %w", err)
	}
	defer rows.Close()

	var columns []Column
	for rows.Next() {
		var columnName, dataType, isNullable string
		if err := rows.Scan(&columnName, &dataType, &isNullable); err != nil {
			return nil, fmt.Errorf("failed to scan column: %w", err)
		}

		columns = append(columns, Column{
			Name:     columnName,
			Type:     dataType,
			Nullable: isNullable == "YES",
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating columns: %w", err)
	}

	return columns, nil
}

// generateTypeScript generates TypeScript type definitions
func generateTypeScript(tables []Table) (string, error) {
	var builder strings.Builder

	builder.WriteString("// Auto-generated TypeScript types from database schema\n")
	builder.WriteString("// Generated by open-db9\n")
	builder.WriteString("\n")

	for _, table := range tables {
		typeName := toPascalCase(table.Name)
		builder.WriteString(fmt.Sprintf("export interface %s {\n", typeName))

		for _, col := range table.Columns {
			tsType := mapPostgreSQLToTypeScript(col.Type, col.Nullable)
			builder.WriteString(fmt.Sprintf("  %s: %s;\n", col.Name, tsType))
		}

		builder.WriteString("}\n\n")
	}

	// Generate input types for creating records
	for _, table := range tables {
		typeName := toPascalCase(table.Name)
		inputName := fmt.Sprintf("%sInput", typeName)
		builder.WriteString(fmt.Sprintf("export interface %s {\n", inputName))

		for _, col := range table.Columns {
			// For input types, all fields are optional unless they're required (not nullable and no default)
			// For simplicity, we'll make all optional for input
			tsType := mapPostgreSQLToTypeScript(col.Type, true)
			builder.WriteString(fmt.Sprintf("  %s?: %s;\n", col.Name, tsType))
		}

		builder.WriteString("}\n\n")
	}

	return builder.String(), nil
}

// generatePython generates Python dataclass definitions
func generatePython(tables []Table) (string, error) {
	var builder strings.Builder

	builder.WriteString("# Auto-generated Python types from database schema\n")
	builder.WriteString("# Generated by open-db9\n")
	builder.WriteString("\n")
	builder.WriteString("from dataclasses import dataclass\n")
	builder.WriteString("from typing import Optional\n")
	builder.WriteString("\n")

	for _, table := range tables {
		className := toPascalCase(table.Name)
		builder.WriteString(fmt.Sprintf("@dataclass\nclass %s:\n", className))

		for _, col := range table.Columns {
			pyType := mapPostgreSQLToPython(col.Type, col.Nullable)
			builder.WriteString(fmt.Sprintf("    %s: %s\n", col.Name, pyType))
		}

		builder.WriteString("\n")
	}

	return builder.String(), nil
}

// validateTypes validates the schema and displays information
func validateTypes(tables []Table) error {
	fmt.Println("Schema Validation Report")
	fmt.Println("======================")

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "TABLE\tCOLUMNS\tDESCRIPTION")
	fmt.Fprintln(w, "-----\t-------\t-----------")

	for _, table := range tables {
		desc := fmt.Sprintf("%d column(s)", len(table.Columns))
		fmt.Fprintf(w, "%s\t%d\t%s\n", table.Name, len(table.Columns), desc)
	}
	w.Flush()

	fmt.Println()
	fmt.Println("Column Details:")
	fmt.Println("---------------")

	for _, table := range tables {
		fmt.Printf("\n%s.%s:\n", genSchema, table.Name)
		for _, col := range table.Columns {
			nullable := "NULL"
			if !col.Nullable {
				nullable = "NOT NULL"
			}
			fmt.Printf("  %-30s %-20s %s\n", col.Name, col.Type, nullable)
		}
	}

	fmt.Printf("\nValidation complete: %d table(s) validated\n", len(tables))
	return nil
}

// mapPostgreSQLToTypeScript maps PostgreSQL types to TypeScript types
func mapPostgreSQLToTypeScript(pgType string, nullable bool) string {
	tsType := "any"

	switch strings.ToLower(pgType) {
	case "integer", "int", "int4", "smallint", "int2", "bigint", "int8":
		tsType = "number"
	case "numeric", "decimal":
		tsType = "number"
	case "real", "float4", "double precision", "float8":
		tsType = "number"
	case "boolean", "bool":
		tsType = "boolean"
	case "varchar", "char", "text", "character varying", "character":
		tsType = "string"
	case "date", "time", "timestamp", "timestamptz", "timetz":
		tsType = "Date"
	case "json", "jsonb":
		tsType = "any"
	case "uuid":
		tsType = "string"
	case "bytea":
		tsType = "Buffer"
	case "array":
		tsType = "any[]"
	default:
		// Default to string for unknown types
		tsType = "string"
	}

	if nullable {
		return tsType + " | null"
	}
	return tsType
}

// mapPostgreSQLToPython maps PostgreSQL types to Python types
func mapPostgreSQLToPython(pgType string, nullable bool) string {
	pyType := "Any"

	switch strings.ToLower(pgType) {
	case "integer", "int", "int4", "smallint", "int2", "bigint", "int8":
		pyType = "int"
	case "numeric", "decimal":
		pyType = "Decimal"
	case "real", "float4", "double precision", "float8":
		pyType = "float"
	case "boolean", "bool":
		pyType = "bool"
	case "varchar", "char", "text", "character varying", "character":
		pyType = "str"
	case "date", "time", "timestamp", "timestamptz", "timetz":
		pyType = "datetime"
	case "json", "jsonb":
		pyType = "dict"
	case "uuid":
		pyType = "str"
	case "bytea":
		pyType = "bytes"
	case "array":
		pyType = "list"
	default:
		pyType = "str"
	}

	if nullable {
		return fmt.Sprintf("Optional[%s]", pyType)
	}
	return pyType
}

// toPascalCase converts a string to PascalCase
func toPascalCase(s string) string {
	parts := strings.Split(s, "_")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
		}
	}
	return strings.Join(parts, "")
}

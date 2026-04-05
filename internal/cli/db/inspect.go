package db

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/jackc/pgx/v5"
	"github.com/spf13/cobra"
)

var (
	inspectSchema  string
	inspectTableName   string
	inspectJSON    bool
	inspectVerbose bool
)

// inspectCmd represents the inspect command
var inspectCmd = &cobra.Command{
	Use:   "inspect <database-id>",
	Short: "Introspect database schema",
	Long: `Introspect and analyze database schema structure.

Provides commands for:
  - Listing all schemas
  - Listing tables in a schema
  - Showing detailed table structure (columns, indexes, constraints, triggers)
  - Analyzing relationships and dependencies`,
}

// inspectSchemasCmd represents the inspect schemas command
var inspectSchemasCmd = &cobra.Command{
	Use:   "schemas <database-id>",
	Short: "List all schemas in the database",
	Long: `List all non-system schemas in the database.

Excludes system schemas like pg_catalog, information_schema, and pg_toast.`,
	Args: cobra.ExactArgs(1),
	RunE: runInspectSchemas,
}

// inspectTablesCmd represents the inspect tables command
var inspectTablesCmd = &cobra.Command{
	Use:   "tables <database-id>",
	Short: "List tables in a schema",
	Long: `List all tables and views in the specified schema.

By default, lists tables in the 'public' schema. Use --schema to specify
a different schema.`,
	Args: cobra.ExactArgs(1),
	RunE: runInspectTables,
}

// inspectTableCmd represents the inspect table command
var inspectTableCmd = &cobra.Command{
	Use:   "table <database-id>",
	Short: "Show detailed table information",
	Long: `Show detailed information about a specific table.

Displays:
  - Table metadata (type, size, row count)
  - Column definitions (types, nullability, defaults)
  - Indexes (type, columns, size)
  - Constraints (primary key, foreign keys, unique, check)
  - Triggers (name, timing, events)

Use --verbose to show additional details like index definitions.`,
	Args: cobra.ExactArgs(1),
	RunE: runInspectTable,
}

func init() {
	// Add inspect command to db command
	Cmd.AddCommand(inspectCmd)

	// Add subcommands
	inspectCmd.AddCommand(inspectSchemasCmd)
	inspectCmd.AddCommand(inspectTablesCmd)
	inspectCmd.AddCommand(inspectTableCmd)

	// Tables command flags
	inspectTablesCmd.Flags().StringVarP(&inspectSchema, "schema", "s", "public",
		"Schema to inspect")
	inspectTablesCmd.Flags().BoolVar(&inspectJSON, "json", false,
		"Output in JSON format")

	// Table command flags
	inspectTableCmd.Flags().StringVarP(&inspectSchema, "schema", "s", "public",
		"Schema containing the table")
	inspectTableCmd.Flags().StringVarP(&inspectTableName, "table", "t", "",
		"Table name to inspect (required)")
	inspectTableCmd.MarkFlagRequired("table")
	inspectTableCmd.Flags().BoolVar(&inspectJSON, "json", false,
		"Output in JSON format")
	inspectTableCmd.Flags().BoolVarP(&inspectVerbose, "verbose", "v", false,
		"Show verbose output including index definitions")
}

// runInspectSchemas executes the inspect schemas command
func runInspectSchemas(cmd *cobra.Command, args []string) error {
	databaseID := args[0]

	ctx := context.Background()
	conn, err := getDBConnection(ctx, databaseID)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)

	schemas, err := listSchemas(ctx, conn)
	if err != nil {
		return fmt.Errorf("failed to list schemas: %w", err)
	}

	if inspectJSON {
		return outputSchemasJSON(schemas)
	}

	printSchemas(schemas)
	return nil
}

// runInspectTables executes the inspect tables command
func runInspectTables(cmd *cobra.Command, args []string) error {
	databaseID := args[0]

	ctx := context.Background()
	conn, err := getDBConnection(ctx, databaseID)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)

	tables, err := listTables(ctx, conn, inspectSchema)
	if err != nil {
		return fmt.Errorf("failed to list tables: %w", err)
	}

	if inspectJSON {
		return outputTablesJSON(tables)
	}

	printTables(tables, inspectSchema)
	return nil
}

// runInspectTable executes the inspect table command
func runInspectTable(cmd *cobra.Command, args []string) error {
	databaseID := args[0]

	ctx := context.Background()
	conn, err := getDBConnection(ctx, databaseID)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)

	details, err := inspectTable(ctx, conn, inspectSchema, inspectTableName)
	if err != nil {
		return fmt.Errorf("failed to inspect table: %w", err)
	}

	if inspectJSON {
		return outputTableDetailsJSON(details)
	}

	printTableDetails(details, inspectVerbose)
	return nil
}

// TableDetails contains detailed table information
type TableDetails struct {
	Schema             string
	Name               string
	Type               string
	RowCount           int64
	Size               int64
	Columns            []ColumnDetails
	Indexes            []IndexDetails
	Constraints        []ConstraintDetails
	Triggers           []TriggerDetails
}

// ColumnDetails contains column information
type ColumnDetails struct {
	Name               string
	Position           int
	DataType           string
	MaxLength          *int
	Nullable           bool
	DefaultValue       *string
}

// IndexDetails contains index information
type IndexDetails struct {
	Name       string
	Type       string
	Unique     bool
	Primary    bool
	Size       int64
	Definition string
	Columns    []string
}

// ConstraintDetails contains constraint information
type ConstraintDetails struct {
	Name              string
	Type              string
	Column            string
	ReferencesTable   *string
	ReferencesColumn  *string
	CheckClause       *string
}

// TriggerDetails contains trigger information
type TriggerDetails struct {
	Name       string
	Enabled    bool
	Definition string
}

// listSchemas lists all non-system schemas
func listSchemas(ctx context.Context, conn *pgx.Conn) ([]string, error) {
	query := `
		SELECT schema_name
		FROM information_schema.schemata
		WHERE schema_name NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
		ORDER BY schema_name
	`

	rows, err := conn.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var schemas []string
	for rows.Next() {
		var schema string
		if err := rows.Scan(&schema); err != nil {
			continue
		}
		schemas = append(schemas, schema)
	}

	return schemas, nil
}

// listTables lists all tables in a schema
func listTables(ctx context.Context, conn *pgx.Conn, schema string) ([]map[string]interface{}, error) {
	query := `
		SELECT
			table_name,
			table_type
		FROM information_schema.tables
		WHERE table_schema = $1
		ORDER BY table_name
	`

	rows, err := conn.Query(ctx, query, schema)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []map[string]interface{}
	for rows.Next() {
		var name, tableType string
		if err := rows.Scan(&name, &tableType); err != nil {
			continue
		}
		tables = append(tables, map[string]interface{}{
			"name": name,
			"type": tableType,
		})
	}

	return tables, nil
}

// inspectTable retrieves detailed table information
func inspectTable(ctx context.Context, conn *pgx.Conn, schema, table string) (*TableDetails, error) {
	details := &TableDetails{
		Schema: schema,
		Name:   table,
	}

	// Get table type
	err := conn.QueryRow(ctx, `
		SELECT table_type
		FROM information_schema.tables
		WHERE table_schema = $1 AND table_name = $2
	`, schema, table).Scan(&details.Type)
	if err != nil {
		return nil, err
	}

	// Get row count and size
	err = conn.QueryRow(ctx, `
		SELECT
			COALESCE(c.reltuples::bigint, 0) as row_count,
			COALESCE(pg_total_relation_size(quote_ident($1) || '.' || quote_ident($2)), 0) as table_size
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = $1 AND c.relname = $2
	`, schema, table).Scan(&details.RowCount, &details.Size)
	if err != nil {
		return nil, err
	}

	// Get columns
	details.Columns, err = getColumnsDetails(ctx, conn, schema, table)
	if err != nil {
		return nil, err
	}

	// Get indexes
	details.Indexes, err = getIndexesDetails(ctx, conn, schema, table)
	if err != nil {
		return nil, err
	}

	// Get constraints
	details.Constraints, err = getConstraintsDetails(ctx, conn, schema, table)
	if err != nil {
		return nil, err
	}

	// Get triggers
	details.Triggers, err = getTriggersDetails(ctx, conn, schema, table)
	if err != nil {
		return nil, err
	}

	return details, nil
}

// getColumnsDetails retrieves column details
func getColumnsDetails(ctx context.Context, conn *pgx.Conn, schema, table string) ([]ColumnDetails, error) {
	query := `
		SELECT
			column_name,
			ordinal_position,
			data_type,
			character_maximum_length,
			is_nullable,
			column_default
		FROM information_schema.columns
		WHERE table_schema = $1 AND table_name = $2
		ORDER BY ordinal_position
	`

	rows, err := conn.Query(ctx, query, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []ColumnDetails
	for rows.Next() {
		var col ColumnDetails
		var maxLen *int
		var defaultVal *string
		var isNullable string

		err := rows.Scan(&col.Name, &col.Position, &col.DataType, &maxLen, &isNullable, &defaultVal)
		if err != nil {
			continue
		}

		col.MaxLength = maxLen
		col.Nullable = isNullable == "YES"
		col.DefaultValue = defaultVal

		columns = append(columns, col)
	}

	return columns, nil
}

// getIndexesDetails retrieves index details
func getIndexesDetails(ctx context.Context, conn *pgx.Conn, schema, table string) ([]IndexDetails, error) {
	query := `
		SELECT
			i.indexrelid::regclass as index_name,
			am.amname as index_type,
			i.indisunique as is_unique,
			i.indisprimary as is_primary,
			pg_relation_size(i.indexrelid) as index_size,
			pg_get_indexdef(i.indexrelid) as index_def,
			array_agg(a.attname ORDER BY array_position(ix.indkey, a.attnum)) as index_columns
		FROM pg_index ix
		JOIN pg_class t ON t.oid = ix.indrelid
		JOIN pg_class idx ON idx.oid = ix.indexrelid
		JOIN pg_namespace n ON n.oid = t.relnamespace
		JOIN pg_am am ON am.oid = idx.relam
		JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = ANY(ix.indkey)
		WHERE n.nspname = $1 AND t.relname = $2
		GROUP BY i.indexrelid, am.amname, i.indisunique, i.indisprimary, i.indexrelid
		ORDER BY i.indisprimary DESC, i.indisunique DESC, 2
	`

	rows, err := conn.Query(ctx, query, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var indexes []IndexDetails
	for rows.Next() {
		var idx IndexDetails
		var columns []string

		err := rows.Scan(&idx.Name, &idx.Type, &idx.Unique, &idx.Primary, &idx.Size, &idx.Definition, &columns)
		if err != nil {
			continue
		}

		idx.Columns = columns
		indexes = append(indexes, idx)
	}

	return indexes, nil
}

// getConstraintsDetails retrieves constraint details
func getConstraintsDetails(ctx context.Context, conn *pgx.Conn, schema, table string) ([]ConstraintDetails, error) {
	query := `
		SELECT
			con.conname as constraint_name,
			con.contype as constraint_type,
			a.attname as column_name,
			cf.relname as foreign_table,
			af.attname as foreign_column
		FROM pg_constraint con
		JOIN pg_class rel ON rel.oid = con.conrelid
		JOIN pg_namespace nsp ON nsp.oid = rel.relnamespace
		LEFT JOIN pg_attribute a ON a.attrelid = rel.oid AND a.attnum = ANY(con.conkey)
		LEFT JOIN pg_class cf ON cf.oid = con.confrelid
		LEFT JOIN pg_attribute af ON af.attrelid = cf.oid AND af.attnum = ANY(con.confkey)
		WHERE nsp.nspname = $1 AND rel.relname = $2
		ORDER BY con.conname
	`

	rows, err := conn.Query(ctx, query, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var constraints []ConstraintDetails
	for rows.Next() {
		var c ConstraintDetails
		var constraintType string
		var foreignTable, foreignColumn *string

		err := rows.Scan(&c.Name, &constraintType, &c.Column, &foreignTable, &foreignColumn)
		if err != nil {
			continue
		}

		c.Type = mapConstraintType(constraintType)
		c.ReferencesTable = foreignTable
		c.ReferencesColumn = foreignColumn

		constraints = append(constraints, c)
	}

	return constraints, nil
}

// getTriggersDetails retrieves trigger details
func getTriggersDetails(ctx context.Context, conn *pgx.Conn, schema, table string) ([]TriggerDetails, error) {
	query := `
		SELECT
			t.tgname as trigger_name,
			t.tgenabled != 'D' as is_enabled,
			pg_get_triggerdef(t.oid) as trigger_def
		FROM pg_trigger t
		JOIN pg_class rel ON rel.oid = t.tgrelid
		JOIN pg_namespace nsp ON nsp.oid = rel.relnamespace
		WHERE nsp.nspname = $1 AND rel.relname = $2
		AND NOT t.tgisinternal
		ORDER BY t.tgname
	`

	rows, err := conn.Query(ctx, query, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var triggers []TriggerDetails
	for rows.Next() {
		var t TriggerDetails

		err := rows.Scan(&t.Name, &t.Enabled, &t.Definition)
		if err != nil {
			continue
		}

		triggers = append(triggers, t)
	}

	return triggers, nil
}

// printSchemas prints schema list
func printSchemas(schemas []string) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	defer w.Flush()

	fmt.Fprintln(w, "Schema Name")
	fmt.Fprintln(w, "------------")

	for _, schema := range schemas {
		fmt.Fprintln(w, schema)
	}

	fmt.Printf("\n%d schema(s)\n", len(schemas))
}

// printTables prints table list
func printTables(tables []map[string]interface{}, schema string) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	defer w.Flush()

	fmt.Fprintln(w, "Table Name\tType")
	fmt.Fprintln(w, "----------\t----")

	for _, t := range tables {
		fmt.Fprintf(w, "%s\t%s\n", t["name"], t["type"])
	}

	fmt.Printf("\n%d table(s) in schema '%s'\n", len(tables), schema)
}

// printTableDetails prints detailed table information
func printTableDetails(details *TableDetails, verbose bool) {
	// Header
	fmt.Printf("\n%s.%s\n", details.Schema, details.Name)
	fmt.Printf("Type: %s | Rows: %d | Size: %s\n\n",
		details.Type, details.RowCount, formatBytes(details.Size))

	// Columns
	fmt.Println("Columns:")
	fmt.Println("--------")
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "Name\tType\tNullable\tDefault")
	fmt.Fprintln(w, "----\t----\t--------\t-------")

	for _, col := range details.Columns {
		dataType := col.DataType
		if col.MaxLength != nil {
			dataType = fmt.Sprintf("%s(%d)", dataType, *col.MaxLength)
		}

		nullable := "NULL"
		if !col.Nullable {
			nullable = "NOT NULL"
		}

		defaultVal := ""
		if col.DefaultValue != nil {
			defaultVal = *col.DefaultValue
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", col.Name, dataType, nullable, defaultVal)
	}
	w.Flush()

	// Indexes
	if len(details.Indexes) > 0 {
		fmt.Println("\nIndexes:")
		fmt.Println("--------")
		w = tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "Name\tType\tUnique\tPrimary\tSize\tColumns")
		fmt.Fprintln(w, "----\t----\t------\t-------\t----\t-------")

		for _, idx := range details.Indexes {
			unique := ""
			if idx.Unique {
				unique = "Y"
			}
			primary := ""
			if idx.Primary {
				primary = "Y"
			}
			columns := strings.Join(idx.Columns, ", ")
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
				idx.Name, idx.Type, unique, primary, formatBytes(idx.Size), columns)
		}
		w.Flush()

		if verbose {
			fmt.Println("\nIndex Definitions:")
			fmt.Println("-------------------")
			for _, idx := range details.Indexes {
				fmt.Printf("%s:\n  %s\n", idx.Name, idx.Definition)
			}
		}
	}

	// Constraints
	if len(details.Constraints) > 0 {
		fmt.Println("\nConstraints:")
		fmt.Println("------------")
		w = tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "Name\tType\tColumn\tReferences")
		fmt.Fprintln(w, "----\t----\t------\t----------")

		for _, c := range details.Constraints {
			refs := ""
			if c.ReferencesTable != nil {
				refs = fmt.Sprintf("%s.%s", *c.ReferencesTable,
					func() string {
						if c.ReferencesColumn != nil {
							return *c.ReferencesColumn
						}
						return ""
					}())
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", c.Name, c.Type, c.Column, refs)
		}
		w.Flush()
	}

	// Triggers
	if len(details.Triggers) > 0 {
		fmt.Println("\nTriggers:")
		fmt.Println("---------")
		for _, t := range details.Triggers {
			enabled := "disabled"
			if t.Enabled {
				enabled = "enabled"
			}
			fmt.Printf("%s (%s)\n", t.Name, enabled)
			if verbose {
				fmt.Printf("  %s\n", t.Definition)
			}
		}
	}
}

// outputSchemasJSON outputs schemas in JSON format
func outputSchemasJSON(schemas []string) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(map[string]interface{}{
		"schemas": schemas,
		"count":   len(schemas),
	})
}

// outputTablesJSON outputs tables in JSON format
func outputTablesJSON(tables []map[string]interface{}) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(map[string]interface{}{
		"tables": tables,
		"count":  len(tables),
	})
}

// outputTableDetailsJSON outputs table details in JSON format
func outputTableDetailsJSON(details *TableDetails) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(details)
}

// mapConstraintType maps PostgreSQL constraint type character to name
func mapConstraintType(ct string) string {
	switch ct {
	case "c":
		return "CHECK"
	case "f":
		return "FOREIGN KEY"
	case "p":
		return "PRIMARY KEY"
	case "u":
		return "UNIQUE"
	case "x":
		return "EXCLUSION"
	default:
		return strings.ToUpper(ct)
	}
}

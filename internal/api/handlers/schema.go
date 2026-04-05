package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/open-db9/db9/internal/database"
	"github.com/open-db9/db9/internal/validate"
)

// ListSchemasHandler returns all schemas in the database
func ListSchemasHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		ct := r.Header.Get("Content-Type")
		if ct != "" && !strings.Contains(ct, "application/json") {
			http.Error(w, "Content-Type must be application/json", http.StatusUnsupportedMediaType)
			return
		}
	}

	// Extract database ID from URL path
	prefix := "/api/v1/databases/"
	rest := strings.TrimPrefix(r.URL.Path, prefix)
	parts := strings.SplitN(rest, "/", 2)

	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "Database ID is required", http.StatusBadRequest)
		return
	}

	dbIDStr := parts[0]
	dbID, err := strconv.Atoi(dbIDStr)
	if err != nil || dbID <= 0 || dbID > 1000000 {
		http.Error(w, "Invalid database ID", http.StatusBadRequest)
		return
	}

	// Get database manager from registry
	manager, err := GetDatabase(dbID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Database not found: %v", err), http.StatusNotFound)
		return
	}

	// Use the manager's pool directly for querying
	pool := manager.GetPool()

	// Query schemas
	query := `
		SELECT schema_name
		FROM information_schema.schemata
		WHERE schema_name NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
		ORDER BY schema_name
	`

	rows, err := pool.Query(r.Context(), query)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to query schemas: %v", err), http.StatusInternalServerError)
		return
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"schemas": schemas,
		"count":   len(schemas),
	})
}

// ListTablesHandler returns all tables in a schema
func ListTablesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		ct := r.Header.Get("Content-Type")
		if ct != "" && !strings.Contains(ct, "application/json") {
			http.Error(w, "Content-Type must be application/json", http.StatusUnsupportedMediaType)
			return
		}
	}

	// Extract database ID from URL path
	prefix := "/api/v1/databases/"
	rest := strings.TrimPrefix(r.URL.Path, prefix)
	parts := strings.SplitN(rest, "/", 2)

	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "Database ID is required", http.StatusBadRequest)
		return
	}

	dbIDStr := parts[0]
	dbID, err := strconv.Atoi(dbIDStr)
	if err != nil || dbID <= 0 || dbID > 1000000 {
		http.Error(w, "Invalid database ID", http.StatusBadRequest)
		return
	}

	// Get schema from query params (default: public)
	schema := r.URL.Query().Get("schema")
	if schema == "" {
		schema = "public"
	}

	// Validate schema name
	if err := validate.IsValidSchemaName(schema); err != nil {
		// Allow system schemas for listing but return a warning header
		if validate.IsSystemSchema(schema) {
			w.Header().Set("X-Warning", "Accessing system schema: use with caution")
		} else {
			http.Error(w, fmt.Sprintf("Invalid schema name: %v", err), http.StatusBadRequest)
			return
		}
	}

	// Get database manager from registry
	manager, err := GetDatabase(dbID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Database not found: %v", err), http.StatusNotFound)
		return
	}

	// Use the manager's pool directly for querying
	pool := manager.GetPool()

	// Query tables
	query := `
		SELECT
			table_name,
			table_type
		FROM information_schema.tables
		WHERE table_schema = $1
		ORDER BY table_name
	`

	rows, err := pool.Query(r.Context(), query, schema)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to query tables: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var tables []map[string]interface{}
	for rows.Next() {
		var tableName, tableType string
		if err := rows.Scan(&tableName, &tableType); err != nil {
			continue
		}
		tables = append(tables, map[string]interface{}{
			"name": tableName,
			"type": tableType,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"schema": schema,
		"tables": tables,
		"count":  len(tables),
	})
}

// GetTableDetailsHandler returns detailed information about a table
func GetTableDetailsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		ct := r.Header.Get("Content-Type")
		if ct != "" && !strings.Contains(ct, "application/json") {
			http.Error(w, "Content-Type must be application/json", http.StatusUnsupportedMediaType)
			return
		}
	}

	// Extract database ID from URL path
	prefix := "/api/v1/databases/"
	rest := strings.TrimPrefix(r.URL.Path, prefix)
	parts := strings.SplitN(rest, "/", 2)

	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "Database ID is required", http.StatusBadRequest)
		return
	}

	dbIDStr := parts[0]
	dbID, err := strconv.Atoi(dbIDStr)
	if err != nil || dbID <= 0 || dbID > 1000000 {
		http.Error(w, "Invalid database ID", http.StatusBadRequest)
		return
	}

	// Get schema and table from query params
	schema := r.URL.Query().Get("schema")
	if schema == "" {
		schema = "public"
	}
	table := r.URL.Query().Get("table")
	if table == "" {
		http.Error(w, "Table name is required", http.StatusBadRequest)
		return
	}

	// Validate schema name
	if err := validate.IsValidSchemaName(schema); err != nil {
		if validate.IsSystemSchema(schema) {
			w.Header().Set("X-Warning", "Accessing system schema: use with caution")
		} else {
			http.Error(w, fmt.Sprintf("Invalid schema name: %v", err), http.StatusBadRequest)
			return
		}
	}

	// Validate table name
	if err := validate.IsValidTableName(table); err != nil {
		http.Error(w, fmt.Sprintf("Invalid table name: %v", err), http.StatusBadRequest)
		return
	}

	// Get database manager from registry
	manager, err := GetDatabase(dbID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Database not found: %v", err), http.StatusNotFound)
		return
	}

	// Use the manager's pool directly for querying
	pool := manager.GetPool()

	// Get table details
	details, err := getTableDetails(r.Context(), pool, schema, table)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get table details: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(details)
}

// getTableDetails retrieves complete table information
func getTableDetails(ctx context.Context, pool *database.ConnectionPool, schema, table string) (map[string]interface{}, error) {
	details := make(map[string]interface{})
	details["schema"] = schema
	details["table"] = table

	// Get table type and row count
	var tableType string
	var rowCount int64
	err := pool.QueryRow(ctx, `
		SELECT t.table_type, COALESCE(c.reltuples::bigint, 0)
		FROM information_schema.tables t
		LEFT JOIN pg_class c ON c.relname = t.table_name
		WHERE t.table_schema = $1 AND t.table_name = $2
	`, schema, table).Scan(&tableType, &rowCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get table info: %w", err)
	}
	details["type"] = tableType
	details["row_count_estimate"] = rowCount

	// Get table size
	var tableSize int64
	err = pool.QueryRow(ctx, `
		SELECT COALESCE(pg_total_relation_size(quote_ident($1) || '.' || quote_ident($2)), 0)
	`, schema, table).Scan(&tableSize)
	if err == nil {
		details["size_bytes"] = tableSize
	}

	// Get columns
	columns, err := getColumns(ctx, pool, schema, table)
	if err == nil {
		details["columns"] = columns
	}

	// Get indexes
	indexes, err := getIndexes(ctx, pool, schema, table)
	if err == nil {
		details["indexes"] = indexes
	}

	// Get constraints
	constraints, err := getConstraints(ctx, pool, schema, table)
	if err == nil {
		details["constraints"] = constraints
	}

	// Get triggers
	triggers, err := getTriggers(ctx, pool, schema, table)
	if err == nil {
		details["triggers"] = triggers
	}

	return details, nil
}

// getColumns retrieves column information
func getColumns(ctx context.Context, pool *database.ConnectionPool, schema, table string) ([]map[string]interface{}, error) {
	query := `
		SELECT
			column_name,
			ordinal_position,
			data_type,
			character_maximum_length,
			numeric_precision,
			numeric_scale,
			is_nullable,
			column_default
		FROM information_schema.columns
		WHERE table_schema = $1 AND table_name = $2
		ORDER BY ordinal_position
	`

	rows, err := pool.Query(ctx, query, schema, table)
	if err != nil {
		return nil, fmt.Errorf("failed to query columns: %w", err)
	}
	defer rows.Close()

	var columns []map[string]interface{}
	for rows.Next() {
		var col map[string]interface{}
		var name, dataType, isNullable string
		var position int
		var charMaxLen, numericPrecision, numericScale *int
		var columnDefault *string

		err := rows.Scan(&name, &position, &dataType, &charMaxLen, &numericPrecision,
			&numericScale, &isNullable, &columnDefault)
		if err != nil {
			continue
		}

		col = map[string]interface{}{
			"name":             name,
			"ordinal_position": position,
			"data_type":        dataType,
			"is_nullable":      isNullable == "YES",
		}

		if charMaxLen != nil {
			col["character_maximum_length"] = *charMaxLen
		}
		if numericPrecision != nil {
			col["numeric_precision"] = *numericPrecision
		}
		if numericScale != nil {
			col["numeric_scale"] = *numericScale
		}
		if columnDefault != nil {
			col["column_default"] = *columnDefault
		}

		columns = append(columns, col)
	}

	return columns, nil
}

// getIndexes retrieves index information
func getIndexes(ctx context.Context, pool *database.ConnectionPool, schema, table string) ([]map[string]interface{}, error) {
	query := `
		SELECT
			i.indexrelid::regclass as index_name,
			am.amname as index_type,
			i.indisunique as is_unique,
			i.indisprimary as is_primary,
			pg_relation_size(i.indexrelid) as index_size,
			pg_get_indexdef(i.indexrelid) as index_def
		FROM pg_index i
		JOIN pg_class t ON t.oid = i.indrelid
		JOIN pg_class idx ON idx.oid = i.indexrelid
		JOIN pg_namespace n ON n.oid = t.relnamespace
		JOIN pg_am am ON am.oid = idx.relam
		WHERE n.nspname = $1 AND t.relname = $2
		ORDER BY i.indisprimary DESC, i.indisunique DESC, 2, 3
	`

	rows, err := pool.Query(ctx, query, schema, table)
	if err != nil {
		return nil, fmt.Errorf("failed to query indexes: %w", err)
	}
	defer rows.Close()

	var indexes []map[string]interface{}
	for rows.Next() {
		var idx map[string]interface{}
		var name, indexType, indexDef string
		var isUnique, isPrimary bool
		var indexSize int64

		err := rows.Scan(&name, &indexType, &isUnique, &isPrimary, &indexSize, &indexDef)
		if err != nil {
			continue
		}

		idx = map[string]interface{}{
			"name":       name,
			"type":       indexType,
			"is_unique":  isUnique,
			"is_primary": isPrimary,
			"size_bytes": indexSize,
			"definition": indexDef,
		}

		indexes = append(indexes, idx)
	}

	return indexes, nil
}

// getConstraints retrieves constraint information
func getConstraints(ctx context.Context, pool *database.ConnectionPool, schema, table string) ([]map[string]interface{}, error) {
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

	rows, err := pool.Query(ctx, query, schema, table)
	if err != nil {
		return nil, fmt.Errorf("failed to query constraints: %w", err)
	}
	defer rows.Close()

	var constraints []map[string]interface{}
	for rows.Next() {
		var c map[string]interface{}
		var name, constraintType, columnName string
		var foreignTable, foreignColumn *string

		err := rows.Scan(&name, &constraintType, &columnName, &foreignTable, &foreignColumn)
		if err != nil {
			continue
		}

		c = map[string]interface{}{
			"name":            name,
			"column":          columnName,
			"constraint_type": mapConstraintType(constraintType),
		}

		if foreignTable != nil {
			c["references_table"] = *foreignTable
		}
		if foreignColumn != nil {
			c["references_column"] = *foreignColumn
		}

		constraints = append(constraints, c)
	}

	return constraints, nil
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

// getTriggers retrieves trigger information
func getTriggers(ctx context.Context, pool *database.ConnectionPool, schema, table string) ([]map[string]interface{}, error) {
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

	rows, err := pool.Query(ctx, query, schema, table)
	if err != nil {
		return nil, fmt.Errorf("failed to query triggers: %w", err)
	}
	defer rows.Close()

	var triggers []map[string]interface{}
	for rows.Next() {
		var t map[string]interface{}
		var name, triggerDef string
		var isEnabled bool

		err := rows.Scan(&name, &isEnabled, &triggerDef)
		if err != nil {
			continue
		}

		t = map[string]interface{}{
			"name":       name,
			"is_enabled": isEnabled,
			"definition": triggerDef,
		}

		triggers = append(triggers, t)
	}

	return triggers, nil
}

// Note: SchemaHandler struct and NewSchemaHandler are kept for backward compatibility
// but the preferred approach is to use the package-level handler functions above.

// SchemaHandler handles schema introspection endpoints (deprecated - use package-level functions)
type SchemaHandler struct {
	db *database.Manager
}

// NewSchemaHandler creates a new schema handler (deprecated - use package-level functions)
func NewSchemaHandler(db *database.Manager) *SchemaHandler {
	return &SchemaHandler{db: db}
}

// ListSchemas returns all schemas in the database (deprecated - use ListSchemasHandler)
func (h *SchemaHandler) ListSchemas(w http.ResponseWriter, r *http.Request) {
	ListSchemasHandler(w, r)
}

// ListTables returns all tables in a schema (deprecated - use ListTablesHandler)
func (h *SchemaHandler) ListTables(w http.ResponseWriter, r *http.Request) {
	ListTablesHandler(w, r)
}

// GetTableDetails returns detailed information about a table (deprecated - use GetTableDetailsHandler)
func (h *SchemaHandler) GetTableDetails(w http.ResponseWriter, r *http.Request) {
	GetTableDetailsHandler(w, r)
}

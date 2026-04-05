package database

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// SchemaIntrospector provides database schema introspection capabilities
type SchemaIntrospector struct {
	pool *ConnectionPool
}

// NewSchemaIntrospector creates a new schema introspector
func NewSchemaIntrospector(pool *ConnectionPool) *SchemaIntrospector {
	return &SchemaIntrospector{pool: pool}
}

// TableInfo contains detailed information about a table
type TableInfo struct {
	SchemaName   string
	TableName    string
	TableType    string // BASE TABLE, VIEW, FOREIGN TABLE, etc.
	RowCount     int64
	TableSize    int64
	Columns      []ColumnInfo
	Indexes      []IndexInfo
	Constraints  []ConstraintInfo
	Triggers     []TriggerInfo
}

// ColumnInfo contains information about a column
type ColumnInfo struct {
	ColumnName        string
	OrdinalPosition   int
	DataType          string
	CharacterMaxLength int
	NumericPrecision  int
	NumericScale      int
	IsNullable        bool
	ColumnDefault     string
	IsIdentity        string
	IsGenerated       string
	IsUpdatable       bool
}

// IndexInfo contains information about an index
type IndexInfo struct {
	IndexName      string
	IndexType      string // btree, hash, gist, gin, etc.
	IsUnique       bool
	IsPrimary      bool
	IndexColumns   []IndexColumnInfo
	IndexSize      int64
	IsPartial      bool
	IndexDef       string
}

// IndexColumnInfo contains column information within an index
type IndexColumnInfo struct {
	ColumnName string
	Position   int
	Order      string // ASC, DESC, NULLS FIRST, NULLS LAST
}

// ConstraintInfo contains information about a constraint
type ConstraintInfo struct {
	ConstraintName  string
	ConstraintType  string // PRIMARY KEY, FOREIGN KEY, UNIQUE, CHECK, EXCLUDE
	ColumnName      string
	ReferenceTable  string
	ReferenceColumn string
	CheckClause     string
}

// TriggerInfo contains information about a trigger
type TriggerInfo struct {
	TriggerName     string
	TriggerManually bool // Is trigger manually created?
	TriggerEnabled  bool
	TriggerActionTime string // BEFORE, AFTER, INSTEAD OF
	TriggerEvent    string // INSERT, UPDATE, DELETE, TRUNCATE
	TriggerFunction string
}

// IntrospectSchema returns complete schema information for all tables in a schema
func (si *SchemaIntrospector) IntrospectSchema(ctx context.Context, schemaName string) ([]TableInfo, error) {
	// Get a connection from the pool
	pconn, err := si.pool.GetConnection(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire connection: %w", err)
	}
	defer pconn.Release()

	// Get all tables in the schema
	tableQuery := `
		SELECT
			table_schema,
			table_name,
			table_type
		FROM information_schema.tables
		WHERE table_schema = $1
		ORDER BY table_name
	`

	rows, err := pconn.Query(ctx, tableQuery, schemaName)
	if err != nil {
		return nil, fmt.Errorf("failed to query tables: %w", err)
	}
	defer rows.Close()

	var tables []TableInfo
	for rows.Next() {
		var ti TableInfo
		if err := rows.Scan(&ti.SchemaName, &ti.TableName, &ti.TableType); err != nil {
			return nil, fmt.Errorf("failed to scan table: %w", err)
		}

		// Skip fetching detailed info for views (handle separately if needed)
		if ti.TableType == "BASE TABLE" || ti.TableType == "VIEW" {
			// Get additional details using the same connection
			ti.Columns, err = getColumnsConn(ctx, pconn, schemaName, ti.TableName)
			if err != nil {
				return nil, fmt.Errorf("failed to get columns for %s: %w", ti.TableName, err)
			}

			ti.Indexes, err = getIndexesConn(ctx, pconn, schemaName, ti.TableName)
			if err != nil {
				return nil, fmt.Errorf("failed to get indexes for %s: %w", ti.TableName, err)
			}

			ti.Constraints, err = getConstraintsConn(ctx, pconn, schemaName, ti.TableName)
			if err != nil {
				return nil, fmt.Errorf("failed to get constraints for %s: %w", ti.TableName, err)
			}

			ti.Triggers, err = getTriggersConn(ctx, pconn, schemaName, ti.TableName)
			if err != nil {
				return nil, fmt.Errorf("failed to get triggers for %s: %w", ti.TableName, err)
			}

			ti.RowCount, ti.TableSize, err = getTableStatsConn(ctx, pconn, schemaName, ti.TableName)
			if err != nil {
				return nil, fmt.Errorf("failed to get table stats for %s: %w", ti.TableName, err)
			}
		}

		tables = append(tables, ti)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating tables: %w", err)
	}

	return tables, nil
}

// IntrospectTable returns detailed information about a specific table
func (si *SchemaIntrospector) IntrospectTable(ctx context.Context, schemaName, tableName string) (*TableInfo, error) {
	// Get a connection from the pool
	pconn, err := si.pool.GetConnection(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire connection: %w", err)
	}
	defer pconn.Release()

	ti := &TableInfo{
		SchemaName: schemaName,
		TableName:  tableName,
	}

	// Get table type
	err = pconn.QueryRow(ctx, `
		SELECT table_type
		FROM information_schema.tables
		WHERE table_schema = $1 AND table_name = $2
	`, schemaName, tableName).Scan(&ti.TableType)
	if err != nil {
		return nil, fmt.Errorf("failed to get table type: %w", err)
	}

	ti.Columns, err = getColumnsConn(ctx, pconn, schemaName, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	ti.Indexes, err = getIndexesConn(ctx, pconn, schemaName, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to get indexes: %w", err)
	}

	ti.Constraints, err = getConstraintsConn(ctx, pconn, schemaName, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to get constraints: %w", err)
	}

	ti.Triggers, err = getTriggersConn(ctx, pconn, schemaName, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to get triggers: %w", err)
	}

	ti.RowCount, ti.TableSize, err = getTableStatsConn(ctx, pconn, schemaName, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to get table stats: %w", err)
	}

	return ti, nil
}

// getColumnsConn retrieves column information for a table using a pooled connection
func getColumnsConn(ctx context.Context, pconn *pgxpool.Conn, schemaName, tableName string) ([]ColumnInfo, error) {
	query := `
		SELECT
			column_name,
			ordinal_position,
			data_type,
			character_maximum_length,
			numeric_precision,
			numeric_scale,
			is_nullable,
			column_default,
			is_identity,
			is_generated,
			is_updatable
		FROM information_schema.columns
		WHERE table_schema = $1 AND table_name = $2
		ORDER BY ordinal_position
	`

	rows, err := pconn.Query(ctx, query, schemaName, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to query columns: %w", err)
	}
	defer rows.Close()

	var columns []ColumnInfo
	for rows.Next() {
		var ci ColumnInfo
		var charMaxLen, numericPrecision, numericScale *int
		var columnDefault *string
		var isIdentity, isGenerated *string

		err := rows.Scan(
			&ci.ColumnName,
			&ci.OrdinalPosition,
			&ci.DataType,
			&charMaxLen,
			&numericPrecision,
			&numericScale,
			&ci.IsNullable,
			&columnDefault,
			&isIdentity,
			&isGenerated,
			&ci.IsUpdatable,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan column: %w", err)
		}

		if charMaxLen != nil {
			ci.CharacterMaxLength = *charMaxLen
		}
		if numericPrecision != nil {
			ci.NumericPrecision = *numericPrecision
		}
		if numericScale != nil {
			ci.NumericScale = *numericScale
		}
		if columnDefault != nil {
			ci.ColumnDefault = *columnDefault
		}
		if isIdentity != nil {
			ci.IsIdentity = *isIdentity
		}
		if isGenerated != nil {
			ci.IsGenerated = *isGenerated
		}

		columns = append(columns, ci)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating columns: %w", err)
	}

	return columns, nil
}

// getIndexesConn retrieves index information for a table using a pooled connection
func getIndexesConn(ctx context.Context, pconn *pgxpool.Conn, schemaName, tableName string) ([]IndexInfo, error) {
	query := `
		SELECT
			i.indexrelid::regclass as index_name,
			am.amname as index_type,
			i.indisunique as is_unique,
			i.indisprimary as is_primary,
			pg_relation_size(i.indexrelid) as index_size,
			pg_get_indexdef(i.indexrelid) as index_def,
			c.indispartial as is_partial
		FROM pg_index i
		JOIN pg_class t ON t.oid = i.indrelid
		JOIN pg_class idx ON idx.oid = i.indexrelid
		JOIN pg_namespace n ON n.oid = t.relnamespace
		JOIN pg_am am ON am.oid = idx.relam
		LEFT JOIN pg_constraint c ON c.conindid = i.indexrelid
		WHERE n.nspname = $1 AND t.relname = $2
		ORDER BY i.indisprimary DESC, i.indisunique DESC, 2, 3
	`

	rows, err := pconn.Query(ctx, query, schemaName, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to query indexes: %w", err)
	}
	defer rows.Close()

	var indexes []IndexInfo
	for rows.Next() {
		var ii IndexInfo
		err := rows.Scan(
			&ii.IndexName,
			&ii.IndexType,
			&ii.IsUnique,
			&ii.IsPrimary,
			&ii.IndexSize,
			&ii.IndexDef,
			&ii.IsPartial,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan index: %w", err)
		}

		// Get index columns
		ii.IndexColumns, err = getIndexColumnsConn(ctx, pconn, ii.IndexName)
		if err != nil {
			return nil, fmt.Errorf("failed to get index columns: %w", err)
		}

		indexes = append(indexes, ii)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating indexes: %w", err)
	}

	return indexes, nil
}

// getIndexColumnsConn retrieves columns that make up an index using a pooled connection
func getIndexColumnsConn(ctx context.Context, pconn *pgxpool.Conn, indexName string) ([]IndexColumnInfo, error) {
	query := `
		SELECT
			a.attname as column_name,
			array_position(ix.indkey, a.attnum) as position,
			CASE
				WHEN ix.indoption[array_position(ix.indkey, a.attnum)] & 1 = 1 THEN 'DESC'
				ELSE 'ASC'
			END as order_direction
		FROM pg_index ix
		JOIN pg_class t ON t.oid = ix.indrelid
		JOIN pg_class i ON i.oid = ix.indexrelid
		JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = ANY(ix.indkey)
		WHERE i.relname = $1
		ORDER BY array_position(ix.indkey, a.attnum)
	`

	rows, err := pconn.Query(ctx, query, indexName)
	if err != nil {
		return nil, fmt.Errorf("failed to query index columns: %w", err)
	}
	defer rows.Close()

	var columns []IndexColumnInfo
	for rows.Next() {
		var ici IndexColumnInfo
		err := rows.Scan(&ici.ColumnName, &ici.Position, &ici.Order)
		if err != nil {
			return nil, fmt.Errorf("failed to scan index column: %w", err)
		}
		columns = append(columns, ici)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating index columns: %w", err)
	}

	return columns, nil
}

// getConstraintsConn retrieves constraint information for a table using a pooled connection
func getConstraintsConn(ctx context.Context, pconn *pgxpool.Conn, schemaName, tableName string) ([]ConstraintInfo, error) {
	query := `
		SELECT
			con.conname as constraint_name,
			con.contype as constraint_type,
			a.attname as column_name,
			cf.relname as foreign_table,
			af.attname as foreign_column,
			pg_get_constraintdef(con.oid) as constraint_def
		FROM pg_constraint con
		JOIN pg_class rel ON rel.oid = con.conrelid
		JOIN pg_namespace nsp ON nsp.oid = rel.relnamespace
		LEFT JOIN pg_attribute a ON a.attrelid = rel.oid AND a.attnum = ANY(con.conkey)
		LEFT JOIN pg_class cf ON cf.oid = con.confrelid
		LEFT JOIN pg_attribute af ON af.attrelid = cf.oid AND af.attnum = ANY(con.confkey)
		WHERE nsp.nspname = $1 AND rel.relname = $2
		ORDER BY con.conname
	`

	rows, err := pconn.Query(ctx, query, schemaName, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to query constraints: %w", err)
	}
	defer rows.Close()

	var constraints []ConstraintInfo
	for rows.Next() {
		var ci ConstraintInfo
		var constraintType string
		var foreignTable, foreignColumn, constraintDef *string

		err := rows.Scan(
			&ci.ConstraintName,
			&constraintType,
			&ci.ColumnName,
			&foreignTable,
			&foreignColumn,
			&constraintDef,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan constraint: %w", err)
		}

		ci.ConstraintType = mapConstraintType(constraintType)
		if foreignTable != nil {
			ci.ReferenceTable = *foreignTable
		}
		if foreignColumn != nil {
			ci.ReferenceColumn = *foreignColumn
		}
		if constraintDef != nil {
			ci.CheckClause = *constraintDef
		}

		constraints = append(constraints, ci)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating constraints: %w", err)
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

// getTriggersConn retrieves trigger information for a table using a pooled connection
func getTriggersConn(ctx context.Context, pconn *pgxpool.Conn, schemaName, tableName string) ([]TriggerInfo, error) {
	query := `
		SELECT
			t.tgname as trigger_name,
			t.tgisinternal as is_internal,
			t.tgenabled != 'D' as is_enabled,
			pg_get_triggerdef(t.oid) as trigger_def
		FROM pg_trigger t
		JOIN pg_class rel ON rel.oid = t.tgrelid
		JOIN pg_namespace nsp ON nsp.oid = rel.relnamespace
		WHERE nsp.nspname = $1 AND rel.relname = $2
		AND NOT t.tgisinternal
		ORDER BY t.tgname
	`

	rows, err := pconn.Query(ctx, query, schemaName, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to query triggers: %w", err)
	}
	defer rows.Close()

	var triggers []TriggerInfo
	for rows.Next() {
		var ti TriggerInfo
		var triggerDef string
		var isInternal bool

		err := rows.Scan(&ti.TriggerName, &isInternal, &ti.TriggerEnabled, &triggerDef)
		if err != nil {
			return nil, fmt.Errorf("failed to scan trigger: %w", err)
		}

		ti.TriggerManually = !isInternal
		ti.TriggerFunction = extractTriggerFunction(triggerDef)
		ti.TriggerActionTime, ti.TriggerEvent = extractTriggerAction(triggerDef)

		triggers = append(triggers, ti)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating triggers: %w", err)
	}

	return triggers, nil
}

// extractTriggerFunction extracts the function name from trigger definition
func extractTriggerFunction(def string) string {
	// Simple extraction - parse "EXECUTE FUNCTION func_name()"
	if idx := strings.Index(def, "EXECUTE FUNCTION "); idx >= 0 {
		rest := def[idx+16:]
		if endIdx := strings.Index(rest, "("); endIdx >= 0 {
			return rest[:endIdx]
		}
	}
	return ""
}

// extractTriggerAction extracts action time and event from trigger definition
func extractTriggerAction(def string) (actionTime, event string) {
	defUpper := strings.ToUpper(def)

	// Extract action time
	if strings.Contains(defUpper, "BEFORE") {
		actionTime = "BEFORE"
	} else if strings.Contains(defUpper, "AFTER") {
		actionTime = "AFTER"
	} else if strings.Contains(defUpper, "INSTEAD OF") {
		actionTime = "INSTEAD OF"
	}

	// Extract event
	events := []string{"INSERT", "UPDATE", "DELETE", "TRUNCATE"}
	for _, e := range events {
		if strings.Contains(defUpper, e) {
			if event == "" {
				event = e
			} else {
				event += " OR " + e
			}
		}
	}

	return actionTime, event
}

// getTableStatsConn retrieves row count and table size using a pooled connection
func getTableStatsConn(ctx context.Context, pconn *pgxpool.Conn, schemaName, tableName string) (rowCount, tableSize int64, err error) {
	// Get row count from pg_class (more accurate than COUNT(*))
	err = pconn.QueryRow(ctx, `
		SELECT COALESCE(reltuples::bigint, 0) as row_count
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = $1 AND c.relname = $2
	`, schemaName, tableName).Scan(&rowCount)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get row count: %w", err)
	}

	// Get table size
	err = pconn.QueryRow(ctx, `
		SELECT COALESCE(pg_total_relation_size(quote_ident($1) || '.' || quote_ident($2)), 0) as table_size
	`, schemaName, tableName).Scan(&tableSize)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get table size: %w", err)
	}

	return rowCount, tableSize, nil
}

// ListSchemas returns all schemas in the database
func (si *SchemaIntrospector) ListSchemas(ctx context.Context) ([]string, error) {
	// Get a connection from the pool
	pconn, err := si.pool.GetConnection(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire connection: %w", err)
	}
	defer pconn.Release()

	query := `
		SELECT schema_name
		FROM information_schema.schemata
		WHERE schema_name NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
		ORDER BY schema_name
	`

	rows, err := pconn.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query schemas: %w", err)
	}
	defer rows.Close()

	var schemas []string
	for rows.Next() {
		var schema string
		if err := rows.Scan(&schema); err != nil {
			return nil, fmt.Errorf("failed to scan schema: %w", err)
		}
		schemas = append(schemas, schema)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating schemas: %w", err)
	}

	return schemas, nil
}

// ListTables returns all tables in a schema
func (si *SchemaIntrospector) ListTables(ctx context.Context, schemaName string) ([]string, error) {
	// Get a connection from the pool
	pconn, err := si.pool.GetConnection(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire connection: %w", err)
	}
	defer pconn.Release()

	query := `
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = $1 AND table_type = 'BASE TABLE'
		ORDER BY table_name
	`

	rows, err := pconn.Query(ctx, query, schemaName)
	if err != nil {
		return nil, fmt.Errorf("failed to query tables: %w", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			return nil, fmt.Errorf("failed to scan table: %w", err)
		}
		tables = append(tables, table)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating tables: %w", err)
	}

	return tables, nil
}

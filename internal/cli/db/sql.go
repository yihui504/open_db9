package db

import (
	"bufio"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/spf13/cobra"
)

var (
	sqlQuery    string
	sqlFile     string
	sqlTimeout  int
	sqlDSN      string
	sqlFormat   string
)

// sqlCmd represents the sql command
var sqlCmd = &cobra.Command{
	Use:   "sql <database-id>",
	Short: "Execute SQL queries against a database",
	Long: `Execute SQL queries against a database.

Supports:
  - Direct query execution with -q
  - File-based queries with -f
  - Interactive mode (default when no -q or -f)
  - Multiple output formats: table, json, csv`,
	Args: cobra.ExactArgs(1),
	RunE: runSQL,
}

func init() {
	// Add sql command to db command
	Cmd.AddCommand(sqlCmd)

	// Command flags
	sqlCmd.Flags().StringVarP(&sqlQuery, "query", "q", "", "SQL query to execute")
	sqlCmd.Flags().StringVarP(&sqlFile, "file", "f", "", "File containing SQL query")
	sqlCmd.Flags().IntVar(&sqlTimeout, "timeout", 30000, "Query timeout in milliseconds")
	sqlCmd.Flags().StringVarP(&sqlDSN, "dsn", "D", "", "Custom DSN connection string")
	sqlCmd.Flags().StringVar(&sqlFormat, "format", "table", "Output format (table, json, csv)")
}

func runSQL(cmd *cobra.Command, args []string) error {
	databaseID := args[0]

	// Validate format
	if sqlFormat != "table" && sqlFormat != "json" && sqlFormat != "csv" {
		return fmt.Errorf("invalid format '%s'. Must be one of: table, json, csv", sqlFormat)
	}

	// Check if query or file is provided
	if sqlQuery == "" && sqlFile == "" {
		// Interactive mode
		return runInteractiveMode(databaseID)
	}

	// Get SQL to execute
	sqlToExecute, err := getSQLToExecute()
	if err != nil {
		return err
	}

	// Execute query
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(sqlTimeout)*time.Millisecond)
	defer cancel()

	return executeSQL(ctx, databaseID, sqlToExecute)
}

// getSQLToExecute reads SQL from query flag or file
func getSQLToExecute() (string, error) {
	if sqlQuery != "" {
		return sqlQuery, nil
	}

	if sqlFile != "" {
		content, err := os.ReadFile(sqlFile)
		if err != nil {
			return "", fmt.Errorf("failed to read SQL file: %w", err)
		}
		return string(content), nil
	}

	return "", fmt.Errorf("no SQL query provided")
}

// runInteractiveMode runs the interactive SQL shell
func runInteractiveMode(databaseID string) error {
	fmt.Printf("Interactive SQL mode for database '%s'\n", databaseID)
	fmt.Println("Enter SQL queries (one per line). Type 'exit' or 'quit' to exit.")

	reader := bufio.NewReader(os.Stdin)
	lineNum := 0

	for {
		lineNum++
		prompt := fmt.Sprintf("[%d] sql> ", lineNum)
		fmt.Print(prompt)

		line, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}

		// Trim whitespace and newlines
		line = strings.TrimSpace(line)

		// Check for exit commands
		if strings.ToLower(line) == "exit" || strings.ToLower(line) == "quit" {
			fmt.Println("Goodbye!")
			return nil
		}

		// Skip empty lines
		if line == "" {
			continue
		}

		// Execute the query
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(sqlTimeout)*time.Millisecond)

		if err := executeSQL(ctx, databaseID, line); err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				fmt.Fprintf(os.Stderr, "Query timeout after %dms\n", sqlTimeout)
			} else if ctx.Err() != nil {
				fmt.Fprintf(os.Stderr, "Context error: %v\n", ctx.Err())
			} else {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			}
		}

		cancel()
		fmt.Println()
	}
}

// executeSQL executes a SQL query and formats the output
func executeSQL(ctx context.Context, databaseID, sql string) error {
	// Get database connection
	conn, err := getDatabaseConnection(ctx, databaseID)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer conn.Close(ctx)

	// Execute query
	rows, err := conn.Query(ctx, sql)
	if err != nil {
		return fmt.Errorf("query execution failed: %w", err)
	}
	defer rows.Close()

	// Get column names
	fieldDescriptions := rows.FieldDescriptions()
	columns := make([]string, len(fieldDescriptions))
	for i, fd := range fieldDescriptions {
		columns[i] = string(fd.Name)
	}

	// Collect all rows
	var results []map[string]interface{}
	for rows.Next() {
		row, err := pgx.RowToMap(rows)
		if err != nil {
			return fmt.Errorf("failed to scan row: %w", err)
		}
		results = append(results, row)
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating rows: %w", err)
	}

	// Format output based on format flag
	switch sqlFormat {
	case "json":
		return outputJSON(results)
	case "csv":
		return outputCSV(columns, results)
	default:
		return outputTable(columns, results)
	}
}

// getDatabaseConnection creates a database connection
func getDatabaseConnection(ctx context.Context, databaseID string) (*pgx.Conn, error) {
	var connString string

	// Use custom DSN if provided
	if sqlDSN != "" {
		connString = sqlDSN
	} else {
		// Build connection string from database ID
		// For now, use environment variables or defaults
		host := getEnv("DB_HOST", "localhost")
		port := getEnv("DB_PORT", "5432")
		user := getEnv("DB_USER", "postgres")
		password := getEnv("DB_PASSWORD", "")
		database := databaseID

		connString = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			host, port, user, password, database)
	}

	// Connect to database
	conn, err := pgx.Connect(ctx, connString)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

// outputTable outputs results in table format
func outputTable(columns []string, results []map[string]interface{}) error {
	if len(results) == 0 {
		fmt.Println("(no results)")
		return nil
	}

	// Calculate column widths
	colWidths := make([]int, len(columns))
	for i, col := range columns {
		colWidths[i] = len(col)
	}

	for _, row := range results {
		for i, col := range columns {
			val := formatValue(row[col])
			if len(val) > colWidths[i] {
				colWidths[i] = len(val)
			}
		}
	}

	// Print header separator
	printSeparator(colWidths)

	// Print header
	printRow(columns, colWidths)

	// Print header separator
	printSeparator(colWidths)

	// Print data rows
	for _, row := range results {
		values := make([]string, len(columns))
		for i, col := range columns {
			values[i] = formatValue(row[col])
		}
		printRow(values, colWidths)
	}

	// Print footer separator
	printSeparator(colWidths)

	// Print row count
	fmt.Printf("\n%d row(s) returned\n", len(results))

	return nil
}

// printSeparator prints a separator line
func printSeparator(widths []int) {
	for i, w := range widths {
		if i > 0 {
			fmt.Print("+")
		}
		fmt.Print(strings.Repeat("-", w+2))
	}
	fmt.Println()
}

// printRow prints a row of data
func printRow(values []string, widths []int) {
	for i, val := range values {
		if i > 0 {
			fmt.Print("|")
		}
		fmt.Printf(" %-*s ", widths[i], val)
	}
	fmt.Println()
}

// formatValue formats a value for display
func formatValue(v interface{}) string {
	if v == nil {
		return "NULL"
	}
	return fmt.Sprintf("%v", v)
}

// outputJSON outputs results in JSON format
func outputJSON(results []map[string]interface{}) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(results); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}
	fmt.Fprintf(os.Stderr, "\n%d row(s) returned\n", len(results))
	return nil
}

// outputCSV outputs results in CSV format
func outputCSV(columns []string, results []map[string]interface{}) error {
	writer := csv.NewWriter(os.Stdout)
	defer writer.Flush()

	// Write header
	if err := writer.Write(columns); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write data rows
	for _, row := range results {
		values := make([]string, len(columns))
		for i, col := range columns {
			values[i] = formatValue(row[col])
		}
		if err := writer.Write(values); err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	fmt.Fprintf(os.Stderr, "%d row(s) returned\n", len(results))
	return nil
}

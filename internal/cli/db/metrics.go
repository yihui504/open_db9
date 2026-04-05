package db

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/jackc/pgx/v5"
	"github.com/spf13/cobra"
)

var (
	metricsThreshold int
	metricsLimit     int
	metricsSort      string
	metricsJSON      bool
)

// metricsCmd represents the metrics command
var metricsCmd = &cobra.Command{
	Use:   "metrics <database-id>",
	Short: "Query database metrics and statistics",
	Long: `Query database metrics and performance statistics.

Provides commands for:
  - Viewing database statistics (size, connections, cache hit ratio)
  - Analyzing query performance from pg_stat_statements
  - Identifying slow queries
  - Resetting statistics`,
}

// metricsStatsCmd represents the metrics stats command
var metricsStatsCmd = &cobra.Command{
	Use:   "stats <database-id>",
	Short: "Show database statistics",
	Long: `Show general database statistics including:
  - Database size
  - Connection counts (total, active, idle)
  - Transaction counts (committed, rolled back)
  - Cache hit ratio
  - pg_stat_statements status`,
	Args: cobra.ExactArgs(1),
	RunE: runMetricsStats,
}

// metricsQueriesCmd represents the metrics queries command
var metricsQueriesCmd = &cobra.Command{
	Use:   "queries <database-id>",
	Short: "Show query statistics from pg_stat_statements",
	Long: `Show query performance statistics from pg_stat_statements.

Displays top queries by execution time, including:
  - Query ID and text
  - Number of calls
  - Execution time (total, mean, max)
  - Rows affected
  - Block I/O statistics`,
	Args: cobra.ExactArgs(1),
	RunE: runMetricsQueries,
}

// metricsSlowCmd represents the metrics slow command
var metricsSlowCmd = &cobra.Command{
	Use:   "slow <database-id>",
	Short: "Show slow queries",
	Long: `Show queries that exceed the execution time threshold.

Queries are retrieved from pg_stat_statements and filtered
by mean execution time. Useful for identifying performance bottlenecks.`,
	Args: cobra.ExactArgs(1),
	RunE: runMetricsSlow,
}

// metricsResetCmd represents the metrics reset command
var metricsResetCmd = &cobra.Command{
	Use:   "reset <database-id>",
	Short: "Reset pg_stat_statements statistics",
	Long: `Reset all statistics in pg_stat_statements.

WARNING: This operation cannot be undone. All accumulated
query statistics will be lost.`,
	Args: cobra.ExactArgs(1),
	RunE: runMetricsReset,
}

func init() {
	// Add metrics command to db command
	Cmd.AddCommand(metricsCmd)

	// Add subcommands
	metricsCmd.AddCommand(metricsStatsCmd)
	metricsCmd.AddCommand(metricsQueriesCmd)
	metricsCmd.AddCommand(metricsSlowCmd)
	metricsCmd.AddCommand(metricsResetCmd)

	// Queries command flags
	metricsQueriesCmd.Flags().IntVarP(&metricsLimit, "limit", "l", 100,
		"Maximum number of queries to show (1-1000)")
	metricsQueriesCmd.Flags().StringVar(&metricsSort, "sort", "total_exec_time",
		"Sort by field (total_exec_time, mean_exec_time, max_exec_time, calls, rows)")
	metricsQueriesCmd.Flags().BoolVar(&metricsJSON, "json", false,
		"Output in JSON format")

	// Slow command flags
	metricsSlowCmd.Flags().IntVarP(&metricsThreshold, "threshold", "t", 1000,
		"Execution time threshold in milliseconds")
	metricsSlowCmd.Flags().IntVarP(&metricsLimit, "limit", "l", 50,
		"Maximum number of queries to show (1-500)")
	metricsSlowCmd.Flags().BoolVar(&metricsJSON, "json", false,
		"Output in JSON format")
}

// runMetricsStats executes the metrics stats command
func runMetricsStats(cmd *cobra.Command, args []string) error {
	databaseID := args[0]

	ctx := context.Background()
	conn, err := getDBConnection(ctx, databaseID)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)

	stats, err := collectDatabaseStats(ctx, conn)
	if err != nil {
		return fmt.Errorf("failed to collect stats: %w", err)
	}

	printDatabaseStats(stats)
	return nil
}

// runMetricsQueries executes the metrics queries command
func runMetricsQueries(cmd *cobra.Command, args []string) error {
	databaseID := args[0]

	// Validate parameters
	if metricsLimit < 1 || metricsLimit > 1000 {
		return fmt.Errorf("invalid limit: must be between 1 and 1000")
	}

	allowedSorts := map[string]bool{
		"total_exec_time": true,
		"mean_exec_time":  true,
		"max_exec_time":   true,
		"calls":           true,
		"rows":            true,
	}
	if !allowedSorts[metricsSort] {
		return fmt.Errorf("invalid sort field: must be one of total_exec_time, mean_exec_time, max_exec_time, calls, rows")
	}

	ctx := context.Background()
	conn, err := getDBConnection(ctx, databaseID)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)

	// Check pg_stat_statements
	var extVersion string
	err = conn.QueryRow(ctx, "SELECT extversion FROM pg_extension WHERE extname = 'pg_stat_statements'").Scan(&extVersion)
	if err != nil {
		return fmt.Errorf("pg_stat_statements extension not enabled. Run: CREATE EXTENSION pg_stat_statements;")
	}

	stats, err := collectQueryStats(ctx, conn, metricsLimit, metricsSort)
	if err != nil {
		return fmt.Errorf("failed to collect query stats: %w", err)
	}

	if metricsJSON {
		return outputQueryStatsJSON(stats)
	}

	printQueryStats(stats, metricsSort)
	return nil
}

// runMetricsSlow executes the metrics slow command
func runMetricsSlow(cmd *cobra.Command, args []string) error {
	databaseID := args[0]

	// Validate parameters
	if metricsThreshold < 0 {
		return fmt.Errorf("invalid threshold: must be >= 0")
	}
	if metricsLimit < 1 || metricsLimit > 500 {
		return fmt.Errorf("invalid limit: must be between 1 and 500")
	}

	ctx := context.Background()
	conn, err := getDBConnection(ctx, databaseID)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)

	// Check pg_stat_statements
	var extVersion string
	err = conn.QueryRow(ctx, "SELECT extversion FROM pg_extension WHERE extname = 'pg_stat_statements'").Scan(&extVersion)
	if err != nil {
		return fmt.Errorf("pg_stat_statements extension not enabled. Run: CREATE EXTENSION pg_stat_statements;")
	}

	slowQueries, err := collectSlowQueries(ctx, conn, int64(metricsThreshold), metricsLimit)
	if err != nil {
		return fmt.Errorf("failed to collect slow queries: %w", err)
	}

	if metricsJSON {
		return outputSlowQueriesJSON(slowQueries, metricsThreshold)
	}

	printSlowQueries(slowQueries, metricsThreshold)
	return nil
}

// runMetricsReset executes the metrics reset command
func runMetricsReset(cmd *cobra.Command, args []string) error {
	databaseID := args[0]

	// Confirm reset
	fmt.Printf("Are you sure you want to reset pg_stat_statements for database '%s'? [y/N]: ", databaseID)
	var response string
	fmt.Scanln(&response)
	if response != "y" && response != "yes" {
		fmt.Println("Reset cancelled.")
		return nil
	}

	ctx := context.Background()
	conn, err := getDBConnection(ctx, databaseID)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)

	// Reset pg_stat_statements
	_, err = conn.Exec(ctx, "SELECT pg_stat_statements_reset()")
	if err != nil {
		return fmt.Errorf("failed to reset pg_stat_statements: %w", err)
	}

	fmt.Println("pg_stat_statements statistics reset successfully.")
	return nil
}

// DatabaseStats represents database statistics
type DatabaseStats struct {
	DatabaseName         string
	DatabaseSize         int64
	TotalConnections     int
	ActiveConnections    int
	IdleConnections      int
	TransactionsCommitted int64
	TransactionsRolledBack int64
	CacheHitRatio        float64
	PgStatStatementsEnabled bool
	PgStatStatementsVersion string
}

// QueryStat represents query statistics
type QueryStat struct {
	QueryID           int64
	Query             string
	Calls             int64
	TotalExecTime     float64
	MeanExecTime      float64
	MaxExecTime       float64
	Rows              int64
	SharedBlksHit     int64
	SharedBlksRead    int64
}

// collectDatabaseStats collects general database statistics
func collectDatabaseStats(ctx context.Context, conn *pgx.Conn) (*DatabaseStats, error) {
	stats := &DatabaseStats{}

	// Get database name
	err := conn.QueryRow(ctx, "SELECT current_database()").Scan(&stats.DatabaseName)
	if err != nil {
		return nil, err
	}

	// Get database size
	err = conn.QueryRow(ctx, "SELECT pg_database_size(current_database())").Scan(&stats.DatabaseSize)
	if err != nil {
		return nil, err
	}

	// Get connection counts
	err = conn.QueryRow(ctx, `
		SELECT
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE state = 'active') as active,
			COUNT(*) FILTER (WHERE state = 'idle') as idle
		FROM pg_stat_activity
		WHERE datname = current_database()
	`).Scan(&stats.TotalConnections, &stats.ActiveConnections, &stats.IdleConnections)
	if err != nil {
		return nil, err
	}

	// Get transaction statistics
	var blksHit, blksRead int64
	err = conn.QueryRow(ctx, `
		SELECT xact_commit, xact_rollback, blks_hit, blks_read
		FROM pg_stat_database
		WHERE datname = current_database()
	`).Scan(&stats.TransactionsCommitted, &stats.TransactionsRolledBack, &blksHit, &blksRead)
	if err != nil {
		return nil, err
	}

	// Calculate cache hit ratio
	if blksHit+blksRead > 0 {
		stats.CacheHitRatio = float64(blksHit) / float64(blksHit+blksRead) * 100
	}

	// Check pg_stat_statements
	err = conn.QueryRow(ctx, "SELECT extversion FROM pg_extension WHERE extname = 'pg_stat_statements'").Scan(&stats.PgStatStatementsVersion)
	if err == nil {
		stats.PgStatStatementsEnabled = true
	}

	return stats, nil
}

// collectQueryStats collects query statistics from pg_stat_statements
func collectQueryStats(ctx context.Context, conn *pgx.Conn, limit int, sortBy string) ([]QueryStat, error) {
	query := fmt.Sprintf(`
		SELECT
			queryid,
			query,
			calls,
			total_exec_time,
			mean_exec_time,
			max_exec_time,
			rows
		FROM pg_stat_statements
		ORDER BY %s DESC
		LIMIT $1
	`, sortBy)

	rows, err := conn.Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []QueryStat
	for rows.Next() {
		var s QueryStat
		err := rows.Scan(&s.QueryID, &s.Query, &s.Calls, &s.TotalExecTime, &s.MeanExecTime, &s.MaxExecTime, &s.Rows)
		if err != nil {
			continue
		}
		stats = append(stats, s)
	}

	return stats, nil
}

// collectSlowQueries collects slow queries from pg_stat_statements
func collectSlowQueries(ctx context.Context, conn *pgx.Conn, thresholdMs int64, limit int) ([]QueryStat, error) {
	query := `
		SELECT
			queryid,
			query,
			calls,
			total_exec_time,
			mean_exec_time,
			max_exec_time,
			rows
		FROM pg_stat_statements
		WHERE mean_exec_time > $1
		ORDER BY mean_exec_time DESC
		LIMIT $2
	`

	rows, err := conn.Query(ctx, query, float64(thresholdMs), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []QueryStat
	for rows.Next() {
		var s QueryStat
		err := rows.Scan(&s.QueryID, &s.Query, &s.Calls, &s.TotalExecTime, &s.MeanExecTime, &s.MaxExecTime, &s.Rows)
		if err != nil {
			continue
		}
		stats = append(stats, s)
	}

	return stats, nil
}

// printDatabaseStats prints database statistics
func printDatabaseStats(stats *DatabaseStats) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	defer w.Flush()

	fmt.Fprintln(w, "Metric\tValue")
	fmt.Fprintln(w, "------\t-----")

	fmt.Fprintf(w, "Database\t%s\n", stats.DatabaseName)
	fmt.Fprintf(w, "Size\t%s\n", formatBytes(stats.DatabaseSize))
	fmt.Fprintf(w, "Connections\t%d (active: %d, idle: %d)\n",
		stats.TotalConnections, stats.ActiveConnections, stats.IdleConnections)
	fmt.Fprintf(w, "Transactions\t%d committed, %d rolled back\n",
		stats.TransactionsCommitted, stats.TransactionsRolledBack)
	fmt.Fprintf(w, "Cache Hit Ratio\t%.2f%%\n", stats.CacheHitRatio)

	if stats.PgStatStatementsEnabled {
		fmt.Fprintf(w, "pg_stat_statments\tenabled (version %s)\n", stats.PgStatStatementsVersion)
	} else {
		fmt.Fprintf(w, "pg_stat_statments\tnot enabled\n")
	}
}

// printQueryStats prints query statistics
func printQueryStats(stats []QueryStat, sortBy string) {
	if len(stats) == 0 {
		fmt.Println("No query statistics found.")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	defer w.Flush()

	fmt.Fprintf(w, "Query ID\tCalls\tTotal Time\tMean Time\tMax Time\tRows\tQuery\n")
	fmt.Fprintf(w, "---------\t-----\t----------\t---------\t--------\t----\t-----\n")

	for _, s := range stats {
		query := truncateQuery(s.Query, 50)
		fmt.Fprintf(w, "%d\t%d\t%.2f ms\t%.2f ms\t%.2f ms\t%d\t%s\n",
			s.QueryID, s.Calls, s.TotalExecTime, s.MeanExecTime, s.MaxExecTime, s.Rows, query)
	}

	fmt.Printf("\n%d query(s) shown (sorted by %s)\n", len(stats), sortBy)
}

// printSlowQueries prints slow queries
func printSlowQueries(stats []QueryStat, threshold int) {
	if len(stats) == 0 {
		fmt.Printf("No slow queries found (threshold: %d ms).\n", threshold)
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	defer w.Flush()

	fmt.Fprintf(w, "Query ID\tCalls\tMean Time\tMax Time\tRows\tQuery\n")
	fmt.Fprintf(w, "---------\t-----\t---------\t--------\t----\t-----\n")

	for _, s := range stats {
		query := truncateQuery(s.Query, 50)
		fmt.Fprintf(w, "%d\t%d\t%.2f ms\t%.2f ms\t%d\t%s\n",
			s.QueryID, s.Calls, s.MeanExecTime, s.MaxExecTime, s.Rows, query)
	}

	fmt.Printf("\n%d slow query(s) found (threshold: %d ms)\n", len(stats), threshold)
}

// outputQueryStatsJSON outputs query statistics in JSON format
func outputQueryStatsJSON(stats []QueryStat) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(stats)
}

// outputSlowQueriesJSON outputs slow queries in JSON format
func outputSlowQueriesJSON(stats []QueryStat, threshold int) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(map[string]interface{}{
		"threshold_ms": threshold,
		"count":        len(stats),
		"slow_queries": stats,
	})
}

// formatBytes formats a byte size into human-readable format
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// truncateQuery truncates a query string
func truncateQuery(query string, maxLen int) string {
	if len(query) <= maxLen {
		return query
	}
	// Remove newlines and extra spaces
	query = compactQuery(query)
	if len(query) <= maxLen {
		return query
	}
	return query[:maxLen-3] + "..."
}

// compactQuery removes newlines and extra spaces from query
func compactQuery(query string) string {
	result := ""
	for _, ch := range query {
		if ch == '\n' || ch == '\t' {
			if len(result) > 0 && result[len(result)-1] != ' ' {
				result += " "
			}
		} else {
			result += string(ch)
		}
	}
	return result
}

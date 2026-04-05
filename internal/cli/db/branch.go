package db

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/open-db9/db9/internal/database"
	"github.com/spf13/cobra"
)

var (
	branchName  string
	branchID    string
	branchYes   bool
	branchJSON  bool
)

// dbManager is the database manager for CLI commands
// This will be initialized by main.go
var dbManager *database.Manager

// SetDatabaseManager sets the database manager for CLI commands
// This is called from cmd/db9/main.go during initialization
func SetDatabaseManager(manager *database.Manager) {
	dbManager = manager
}

// branchCmd represents the branch command
var branchCmd = &cobra.Command{
	Use:   "branch",
	Short: "Database branch management",
	Long: `Manage database branches for isolated testing and development.

Branch Semantics:
  - Isolation: Each branch is a completely isolated database
  - No Merge: Branches cannot be merged back to parent
  - Snapshot: Branches are created from snapshots of the parent database
  - Independent: Branches operate independently with their own data

Use cases:
  - Parallel testing without interference
  - Isolated development environments
  - Safe experimentation with production data
  - Reproducible bug investigation`,
}

// branchCreateCmd represents the branch create command
var branchCreateCmd = &cobra.Command{
	Use:   "create <database-id>",
	Short: "Create a new branch from a database",
	Long: `Create a new branch from an existing database.

The branch creation process:
  1. Creates a snapshot of the parent database
  2. Creates a new database from the snapshot
  3. Returns the branch ID for reference

Example:
  db9 db branch create mydb --name "feature-test"`,
	Args: cobra.ExactArgs(1),
	RunE: runBranchCreate,
}

// branchListCmd represents the branch list command
var branchListCmd = &cobra.Command{
	Use:   "list <database-id>",
	Short: "List all branches of a database",
	Long: `List all branches created from a specific database.

Displays:
  - Branch ID and name
  - Parent database ID
  - Snapshot ID
  - Creation time
  - Current status

Example:
  db9 db branch list mydb`,
	Args: cobra.ExactArgs(1),
	RunE: runBranchList,
}

// branchDeleteCmd represents the branch delete command
var branchDeleteCmd = &cobra.Command{
	Use:   "delete <database-id> --branch <branch-id>",
	Short: "Delete a branch",
	Long: `Delete a branch and its associated database.

Warning: This action cannot be undone. All data in the branch will be permanently deleted.

Use --yes flag to skip confirmation prompt.

Example:
  db9 db branch delete mydb --branch branch_123 --yes`,
	Args: cobra.ExactArgs(1),
	RunE: runBranchDelete,
}

func init() {
	// Add branch command to db command
	Cmd.AddCommand(branchCmd)

	// Add subcommands to branch command
	branchCmd.AddCommand(branchCreateCmd)
	branchCmd.AddCommand(branchListCmd)
	branchCmd.AddCommand(branchDeleteCmd)

	// Create command flags
	branchCreateCmd.Flags().StringVarP(&branchName, "name", "n", "", "Branch name (required)")
	branchCreateCmd.MarkFlagRequired("name")
	branchCreateCmd.Flags().BoolVar(&branchJSON, "json", false, "Output in JSON format")

	// List command flags
	branchListCmd.Flags().BoolVar(&branchJSON, "json", false, "Output in JSON format")

	// Delete command flags
	branchDeleteCmd.Flags().StringVar(&branchID, "branch", "", "Branch ID (required)")
	branchDeleteCmd.MarkFlagRequired("branch")
	branchDeleteCmd.Flags().BoolVarP(&branchYes, "yes", "y", false, "Skip confirmation prompt")
}

// runBranchCreate executes the branch create command
func runBranchCreate(cmd *cobra.Command, args []string) error {
	databaseID := args[0]

	if dbManager == nil {
		return fmt.Errorf("database manager not initialized. Please check your configuration")
	}

	ctx := context.Background()

	// Parse database ID as integer, then convert to UUID using MD5 (same as API)
	dbIDInt, err := strconv.Atoi(databaseID)
	if err != nil || dbIDInt <= 0 {
		return fmt.Errorf("invalid database ID: must be a positive integer")
	}

	// Convert int ID to UUID using MD5 hash (consistent with API handler)
	sourceUUID := uuid.NewMD5(uuid.Nil, []byte(strconv.Itoa(dbIDInt)))

	// Create branch using Manager
	fmt.Printf("Creating branch '%s' from database '%s'...\n", branchName, databaseID)

	branch, err := dbManager.CreateBranch(ctx, sourceUUID, branchName)
	if err != nil {
		return fmt.Errorf("failed to create branch: %w", err)
	}

	// Build branch info for output
	branchInfo := map[string]interface{}{
		"branch_id":       branch.ID.String(),
		"branch_name":     branch.Name,
		"parent_database": databaseID,
		"snapshot_id":     "",
		"created_at":      branch.CreatedAt.Format(time.RFC3339),
		"status":          "active",
	}

	if branch.SnapshotID != nil {
		branchInfo["snapshot_id"] = branch.SnapshotID.String()
	}

	if branchJSON {
		return outputBranchJSON(branchInfo)
	}

	// Display summary
	fmt.Println("\nBranch created successfully:")
	fmt.Printf("  Branch ID:     %s\n", branch.ID.String())
	fmt.Printf("  Branch Name:   %s\n", branch.Name)
	fmt.Printf("  Parent DB:     %s\n", databaseID)
	if branch.SnapshotID != nil {
		fmt.Printf("  Snapshot ID:   %s\n", branch.SnapshotID.String())
	}
	fmt.Printf("  Created At:    %s\n", branchInfo["created_at"])
	fmt.Printf("  Status:        %s\n", branchInfo["status"])

	return nil
}

// runBranchList executes the branch list command
func runBranchList(cmd *cobra.Command, args []string) error {
	databaseID := args[0]

	if dbManager == nil {
		return fmt.Errorf("database manager not initialized. Please check your configuration")
	}

	ctx := context.Background()

	// Parse database ID as integer, then convert to UUID using MD5 (same as API)
	dbIDInt, err := strconv.Atoi(databaseID)
	if err != nil || dbIDInt <= 0 {
		return fmt.Errorf("invalid database ID: must be a positive integer")
	}

	// Convert int ID to UUID using MD5 hash (consistent with API handler)
	sourceUUID := uuid.NewMD5(uuid.Nil, []byte(strconv.Itoa(dbIDInt)))

	// List branches using Manager
	branches, err := dbManager.ListBranches(ctx, sourceUUID)
	if err != nil {
		return fmt.Errorf("failed to list branches: %w", err)
	}

	if len(branches) == 0 {
		fmt.Printf("No branches found for database '%s'\n", databaseID)
		return nil
	}

	// Convert to map format for JSON output
	branchMaps := make([]map[string]interface{}, len(branches))
	for i, branch := range branches {
		branchMaps[i] = map[string]interface{}{
			"branch_id":       branch.ID.String(),
			"branch_name":     branch.Name,
			"parent_database": databaseID,
			"snapshot_id":     "",
			"created_at":      branch.CreatedAt.Format(time.RFC3339),
			"status":          "active",
		}
		if branch.SnapshotID != nil {
			branchMaps[i]["snapshot_id"] = branch.SnapshotID.String()
		}
	}

	if branchJSON {
		return outputBranchJSON(branchMaps)
	}

	// Display branches in table format
	fmt.Printf("Branches for database '%s':\n\n", databaseID)
	printDatabaseBranchTable(branches)

	return nil
}

// runBranchDelete executes the branch delete command
func runBranchDelete(cmd *cobra.Command, args []string) error {
	databaseID := args[0]

	if dbManager == nil {
		return fmt.Errorf("database manager not initialized. Please check your configuration")
	}

	// Confirm deletion unless --yes flag is provided
	if !branchYes {
		if !confirmBranchDeletion(databaseID, branchID) {
			fmt.Println("Deletion cancelled")
			return nil
		}
	}

	ctx := context.Background()

	// Parse branch ID as UUID
	branchUUID, err := uuid.Parse(branchID)
	if err != nil {
		return fmt.Errorf("invalid branch ID format: %w", err)
	}

	// Delete branch using Manager
	fmt.Printf("Deleting branch '%s'...\n", branchID)
	if err := dbManager.DeleteBranch(ctx, branchUUID, true); err != nil {
		return fmt.Errorf("failed to delete branch: %w", err)
	}

	fmt.Printf("✓ Branch '%s' deleted successfully\n", branchID)
	return nil
}

// printDatabaseBranchTable displays database branches in a formatted table
func printDatabaseBranchTable(branches []database.DatabaseBranch) {
	// Calculate column widths
	colWidths := map[string]int{
		"Branch ID":     40,
		"Name":          30,
		"Snapshot ID":   40,
		"Created At":    25,
		"Status":        10,
	}

	// Print header
	printBranchSeparator(colWidths)
	printHeader([]string{"Branch ID", "Name", "Snapshot ID", "Created At", "Status"}, colWidths)
	printBranchSeparator(colWidths)

	// Print data rows
	for _, branch := range branches {
		printDatabaseBranchRow(branch, colWidths)
	}

	// Print footer
	printBranchSeparator(colWidths)

	fmt.Printf("\n%d branch(es)\n", len(branches))
}

// printDatabaseBranchRow prints a single database branch row
func printDatabaseBranchRow(branch database.DatabaseBranch, widths map[string]int) {
	cols := []string{"Branch ID", "Name", "Snapshot ID", "Created At", "Status"}
	for _, col := range cols {
		var val string

		switch col {
		case "Branch ID":
			val = branch.ID.String()
		case "Name":
			val = branch.Name
		case "Snapshot ID":
			if branch.SnapshotID != nil {
				val = branch.SnapshotID.String()
			}
		case "Created At":
			val = branch.CreatedAt.Format("2006-01-02 15:04:05")
		case "Status":
			val = "active"
		}

		fmt.Printf(" %-*s ", widths[col], val)
		fmt.Print("|")
	}
	fmt.Println()
}

// confirmBranchDeletion prompts the user to confirm branch deletion
func confirmBranchDeletion(databaseID, branchID string) bool {
	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("\n⚠️  WARNING: You are about to delete branch '%s'\n", branchID)
	fmt.Printf("   Parent database: %s\n", databaseID)
	fmt.Println("   This action cannot be undone!")
	fmt.Print("\nAre you sure you want to continue? (yes/no): ")

	input, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	input = strings.TrimSpace(strings.ToLower(input))
	return input == "yes" || input == "y"
}

// printBranchSeparator prints a separator line for branch table
func printBranchSeparator(widths map[string]int) {
	fmt.Print("+")
	for _, col := range []string{"Branch ID", "Name", "Snapshot ID", "Created At", "Status"} {
		fmt.Print(strings.Repeat("-", widths[col]+2))
		fmt.Print("+")
	}
	fmt.Println()
}

// printHeader prints the table header
func printHeader(cols []string, widths map[string]int) {
	for _, col := range cols {
		fmt.Printf(" %-*s ", widths[col], col)
		fmt.Print("|")
	}
	fmt.Println()
}

// outputBranchJSON outputs data in JSON format
func outputBranchJSON(data interface{}) error {
	fmt.Println("{")
	switch v := data.(type) {
	case map[string]interface{}:
		first := true
		for key, val := range v {
			if !first {
				fmt.Println(",")
			}
			first = false
			fmt.Printf("  \"%s\": %v", key, formatJSONValue(val))
		}
	case []map[string]interface{}:
		for i, item := range v {
			if i > 0 {
				fmt.Println(",")
			}
			fmt.Printf("  {")
			first := true
			for key, val := range item {
				if !first {
					fmt.Print(", ")
				}
				first = false
				fmt.Printf("\"%s\": %v", key, formatJSONValue(val))
			}
			fmt.Print("  }")
		}
	}
	fmt.Println("\n}")
	return nil
}

// formatJSONValue formats a value for JSON output
func formatJSONValue(val interface{}) string {
	switch v := val.(type) {
	case string:
		return fmt.Sprintf("\"%s\"", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}


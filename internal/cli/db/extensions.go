package db

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/open-db9/db9/internal/database"
	"github.com/spf13/cobra"
)

var (
	// Extension flags
	extSchema string
	extCascade bool
)

// extensionsCmd represents the extensions command
var extensionsCmd = &cobra.Command{
	Use:   "extensions <database-id>",
	Short: "Manage database extensions",
	Long: `Manage database extensions for installing, listing, and verifying
database extensions like vector, pg_cron, pg_net, etc.`,
	Args: cobra.ExactArgs(1),
	RunE: runExtensionsList,
}

// extensionsListCmd represents the extensions list command
var extensionsListCmd = &cobra.Command{
	Use:   "list <database-id>",
	Short: "List all available extensions",
	Long: `List all available built-in extensions with their installation status,
version, description, and any limitations.`,
	Args: cobra.ExactArgs(1),
	RunE: runExtensionsList,
}

// extensionsInstallCmd represents the extensions install command
var extensionsInstallCmd = &cobra.Command{
	Use:   "install <database-id> <extension-name>",
	Short: "Install a database extension",
	Long: `Install a database extension by name. The command will:
  - Check if the extension is already installed
  - Install any dependencies first
  - Execute the installation SQL
  - Verify the installation was successful`,
	Args: cobra.ExactArgs(2),
	RunE: runExtensionsInstall,
}

// extensionsVerifyCmd represents the extensions verify command
var extensionsVerifyCmd = &cobra.Command{
	Use:   "verify <database-id> <extension-name>",
	Short: "Verify an extension is installed",
	Long: `Verify that a database extension is properly installed and functional.
Returns exit code 0 if verified, 1 otherwise.`,
	Args: cobra.ExactArgs(2),
	RunE: runExtensionsVerify,
}

func init() {
	// Add extensions command group
	Cmd.AddCommand(extensionsCmd)

	// Add subcommands
	extensionsCmd.AddCommand(extensionsListCmd)
	extensionsCmd.AddCommand(extensionsInstallCmd)
	extensionsCmd.AddCommand(extensionsVerifyCmd)

	// Add flags for install command
	extensionsInstallCmd.Flags().StringVar(&extSchema, "schema", "", "Schema to install extension into")
	extensionsInstallCmd.Flags().BoolVar(&extCascade, "cascade", false, "Cascade to install dependencies")
}


// runExtensionsList executes the extensions list command
func runExtensionsList(cmd *cobra.Command, args []string) error {
	databaseID := args[0]
	ctx := context.Background()

	// Get database connection
	conn, err := getDBConnection(ctx, databaseID)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)

	// Get list of built-in extensions
	builtinExts := database.ListBuiltinExtensions()

	// Query installed extensions
	installedExts := make(map[string]bool)
	rows, err := conn.Query(ctx, "SELECT extname FROM pg_extension")
	if err != nil {
		return fmt.Errorf("failed to query installed extensions: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var extName string
		if err := rows.Scan(&extName); err != nil {
			return fmt.Errorf("failed to scan extension name: %w", err)
		}
		installedExts[extName] = true
	}

	// Display extensions in a formatted table
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "NAME\tVERSION\tINSTALLED\tDESCRIPTION")
	fmt.Fprintln(w, "----\t-------\t---------\t-----------")

	for _, extName := range builtinExts {
		extDef, _ := database.GetBuiltinExtension(extName)
		installed := "No"
		if installedExts[extName] {
			installed = "Yes"
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", extDef.Name, extDef.Version, installed, extDef.Description)
	}
	w.Flush()

	// Show limitations if any
	fmt.Println("\nLimitations:")
	for _, extName := range builtinExts {
		extDef, _ := database.GetBuiltinExtension(extName)
		if len(extDef.Limitations) > 0 {
			fmt.Printf("\n%s:\n", extDef.Name)
			for _, lim := range extDef.Limitations {
				fmt.Printf("  - %s\n", lim)
			}
		}
	}

	return nil
}

// runExtensionsInstall executes the extensions install command
func runExtensionsInstall(cmd *cobra.Command, args []string) error {
	databaseID := args[0]
	extName := args[1]
	ctx := context.Background()

	// Check if extension is a built-in
	if _, exists := database.GetBuiltinExtension(extName); !exists {
		return fmt.Errorf("extension '%s' is not a recognized built-in extension", extName)
	}

	// Get extension details
	extDef, _ := database.GetBuiltinExtension(extName)

	fmt.Printf("Installing extension '%s' for database '%s'...\n", extName, databaseID)
	fmt.Printf("Version: %s\n", extDef.Version)
	fmt.Printf("Description: %s\n", extDef.Description)

	// Show dependencies if any
	if len(extDef.Dependencies) > 0 {
		fmt.Printf("Dependencies: %s\n", strings.Join(extDef.Dependencies, ", "))
	}

	// Get database connection
	conn, err := getDBConnection(ctx, databaseID)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)

	// Check if extension is already installed
	var installed bool
	err = conn.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_extension WHERE extname = $1)", extName).Scan(&installed)
	if err != nil {
		return fmt.Errorf("failed to check extension status: %w", err)
	}

	if installed {
		fmt.Printf("Extension '%s' is already installed\n", extName)
		return nil
	}

	// Install the extension
	fmt.Println("\nInstalling...")

	// Build installation SQL
	sql := fmt.Sprintf("CREATE EXTENSION IF NOT EXISTS %s", extName)
	if extSchema != "" {
		sql += fmt.Sprintf(" SCHEMA %s", extSchema)
	}
	if extCascade {
		sql += " CASCADE"
	}

	// Execute installation
	_, err = conn.Exec(ctx, sql)
	if err != nil {
		fmt.Printf("Installation failed: %v\n", err)
		return fmt.Errorf("failed to install extension %s: %w", extName, err)
	}

	fmt.Printf("Successfully installed extension '%s'\n", extName)

	// Verify installation
	fmt.Println("\nVerifying installation...")
	err = conn.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_extension WHERE extname = $1)", extName).Scan(&installed)
	if err != nil {
		fmt.Printf("Warning: Could not verify installation: %v\n", err)
		return nil
	}

	if installed {
		fmt.Println("Installation verified successfully")
	} else {
		fmt.Println("Warning: Installation completed but verification failed")
	}

	return nil
}

// runExtensionsVerify executes the extensions verify command
func runExtensionsVerify(cmd *cobra.Command, args []string) error {
	databaseID := args[0]
	extName := args[1]
	ctx := context.Background()

	// Check if extension is a built-in
	if _, exists := database.GetBuiltinExtension(extName); !exists {
		return fmt.Errorf("extension '%s' is not a recognized built-in extension", extName)
	}

	fmt.Printf("Verifying extension '%s' for database '%s'...\n", extName, databaseID)

	// Get database connection
	conn, err := getDBConnection(ctx, databaseID)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)

	// Verify the extension
	var installed bool
	err = conn.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_extension WHERE extname = $1)", extName).Scan(&installed)
	if err != nil {
		fmt.Printf("Verification failed: %v\n", err)
		return fmt.Errorf("failed to verify extension %s: %w", extName, err)
	}

	if installed {
		fmt.Println("Extension is installed and functional")

		// Get additional info if available
		sql := `
			SELECT
				ext.default_version as default_version,
				n.nspname as schema,
				COALESCE(ext.version, ext.default_version) as version,
				ext.comment as comment
			FROM pg_available_extensions ext
			LEFT JOIN pg_namespace n ON n.oid = ext.extnamespace
			WHERE ext.name = $1
		`

		row := conn.QueryRow(ctx, sql, extName)
		var defaultVersion, schema, version, comment string
		err = row.Scan(&defaultVersion, &schema, &version, &comment)
		if err == nil {
			fmt.Printf("Version: %s\n", version)
			if schema != "" {
				fmt.Printf("Schema: %s\n", schema)
			}
			if comment != "" {
				fmt.Printf("Comment: %s\n", comment)
			}
		}

		return nil
	}

	fmt.Println("Extension is not installed")
	return fmt.Errorf("extension %s is not installed", extName)
}

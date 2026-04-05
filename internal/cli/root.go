package cli

import (
	"fmt"
	"os"

	"github.com/open-db9/db9/internal/cli/cmd"
	"github.com/open-db9/db9/internal/cli/commands"
	"github.com/open-db9/db9/internal/cli/db"
	"github.com/spf13/cobra"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "open-db9",
	Short: "OpenDB9 - Database testing and quality assurance platform",
	Long: `OpenDB9 is a comprehensive database testing and quality assurance platform
designed for vector databases and traditional relational databases.

It provides automated testing, contract validation, differential testing,
and semantic consistency checks across multiple database systems.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error executing command: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Add subcommands
	rootCmd.AddCommand(db.Cmd)
	rootCmd.AddCommand(commands.Cmd)
	rootCmd.AddCommand(cmd.OnboardCmd)

	// Persistent flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "",
		"config file (default is $HOME/.open-db9.yaml)")

	rootCmd.PersistentFlags().StringP("output", "o", "",
		"output format (json, yaml, table)")

	rootCmd.PersistentFlags().CountVarP(&verbose, "verbose", "v",
		"verbose output (can be used multiple times)")

	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false,
		"enable debug mode")
}

var verbose int
var debug bool

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag
		Config.SetConfigFile(cfgFile)
	} else {
		// Find home directory
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".open-db9" (without extension)
		Config.AddConfigPath(home)
		Config.SetConfigType("yaml")
		Config.SetConfigName(".open-db9")
	}

	// Read in environment variables that match
	Config.SetEnvPrefix("DB9")
	Config.AutomaticEnv()

	// If a config file is found, read it in
	if err := Config.ReadInConfig(); err == nil && verbose > 0 {
		fmt.Fprintf(os.Stderr, "Using config file: %s\n", Config.ConfigFileUsed())
	}
}

package commands

import (
	"fmt"
	"os"
)

// Command represents a CLI command
type Command struct {
	Name        string
	Description string
	Handler     func(args []string) error
}

// CommandRegistry holds all available commands
var CommandRegistry []Command

func init() {
	CommandRegistry = []Command{
		{
			Name:        "version",
			Description: "Print version information",
			Handler:     VersionCommand,
		},
		{
			Name:        "help",
			Description: "Show help information",
			Handler:     HelpCommand,
		},
	}
}

// VersionCommand prints version information
func VersionCommand(args []string) error {
	fmt.Println("DB9 CLI v0.1.0")
	return nil
}

// HelpCommand shows help information
func HelpCommand(args []string) error {
	fmt.Println("DB9 - A modern database CLI tool")
	fmt.Println("\nAvailable commands:")
	for _, cmd := range CommandRegistry {
		fmt.Printf("  %-12s %s\n", cmd.Name, cmd.Description)
	}
	fmt.Println("\nUse 'db9 help <command>' for more information.")
	return nil
}

// Execute runs a command by name
func Execute(name string, args []string) error {
	for _, cmd := range CommandRegistry {
		if cmd.Name == name {
			return cmd.Handler(args)
		}
	}
	return fmt.Errorf("unknown command: %s", name)
}

// PrintUsage prints the usage information
func PrintUsage() {
	fmt.Println("Usage: db9 <command> [arguments]")
	fmt.Println("\nUse 'db9 help' for more information.")
}

// ExitWithError prints an error and exits
func ExitWithError(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
	os.Exit(1)
}

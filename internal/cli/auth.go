package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/open-db9/db9/internal/secrets"
	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authentication and credential management",
	Long: `Manage authentication credentials for database connections and external services.

Supports multiple authentication methods including API keys, OAuth tokens,
and database connection credentials.`,
}

var loginCmd = &cobra.Command{
	Use:   "login [service]",
	Short: "Login to a service or database",
	Long: `Authenticate with a service or database using credentials.

Supported services:
  - postgres: PostgreSQL database
  - mysql: MySQL/MariaDB database
  - milvus: Milvus vector database
  - qdrant: Qdrant vector database
  - weaviate: Weaviate vector database
  - github: GitHub API (for test result synchronization)`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		service := args[0]
		username, _ := cmd.Flags().GetString("username")
		password, _ := cmd.Flags().GetString("password")
		token, _ := cmd.Flags().GetString("token")
		host, _ := cmd.Flags().GetString("host")
		port, _ := cmd.Flags().GetString("port")

		if verbose > 0 {
			fmt.Printf("Logging in to service: %s\n", service)
		}

		// Store credentials securely
		cred := Credentials{
			Service:  service,
			Username: username,
			Password: password,
			Token:    token,
			Host:     host,
			Port:     port,
		}

		if err := StoreCredentials(cred); err != nil {
			return fmt.Errorf("failed to store credentials: %w", err)
		}

		fmt.Printf("Successfully logged in to %s\n", service)
		return nil
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show authentication status",
	Long:  `Display current authentication status and stored credentials.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		creds, err := ListCredentials()
		if err != nil {
			return fmt.Errorf("failed to list credentials: %w", err)
		}

		if len(creds) == 0 {
			fmt.Println("No stored credentials found.")
			return nil
		}

		fmt.Println("Stored credentials:")
		for _, cred := range creds {
			fmt.Printf("  - %s: %s@%s:%s\n", cred.Service, cred.Username, cred.Host, cred.Port)
		}
		return nil
	},
}

var logoutCmd = &cobra.Command{
	Use:   "logout [service]",
	Short: "Logout and remove credentials",
	Long:  `Remove stored credentials for a service. If no service specified, removes all credentials.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			// Remove all credentials
			if err := ClearAllCredentials(); err != nil {
				return fmt.Errorf("failed to clear credentials: %w", err)
			}
			fmt.Println("All credentials removed.")
			return nil
		}

		service := args[0]
		if err := RemoveCredentials(service); err != nil {
			return fmt.Errorf("failed to remove credentials: %w", err)
		}

		fmt.Printf("Credentials for %s removed.\n", service)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(authCmd)
	authCmd.AddCommand(loginCmd)
	authCmd.AddCommand(statusCmd)
	authCmd.AddCommand(logoutCmd)

	// Login command flags
	loginCmd.Flags().StringP("username", "u", "", "Username")
	loginCmd.Flags().StringP("password", "p", "", "Password")
	loginCmd.Flags().StringP("token", "t", "", "API token")
	loginCmd.Flags().StringP("host", "H", "localhost", "Host")
	loginCmd.Flags().StringP("port", "P", "", "Port")
}

// Credentials represents stored authentication credentials
type Credentials struct {
	Service  string
	Username string
	Password string
	Token    string
	Host     string
	Port     string
}

// secretsManager holds the global secrets manager instance
var secretsManager *secrets.Manager

// initSecretsManager initializes the secrets manager with a master key
func initSecretsManager() error {
	if secretsManager != nil {
		return nil
	}

	// Get master key from environment or prompt
	masterKey := os.Getenv("DB9_MASTER_KEY")
	if masterKey == "" {
		// For CLI usage, derive a key from user home directory
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get user home directory: %w", err)
		}
		// Use a machine-specific key derivation
		masterKey = "db9-" + homeDir
	}

	// Secrets file path
	configDir, err := os.UserConfigDir()
	if err != nil {
		return fmt.Errorf("failed to get config directory: %w", err)
	}
	secretsPath := filepath.Join(configDir, "db9", "credentials.sec")

	// Create secrets manager
	mgr, err := secrets.NewManager(secretsPath, masterKey)
	if err != nil {
		return fmt.Errorf("failed to create secrets manager: %w", err)
	}

	secretsManager = mgr
	return nil
}

// StoreCredentials securely stores credentials using AES-GCM encryption
func StoreCredentials(cred Credentials) error {
	if err := initSecretsManager(); err != nil {
		return err
	}

	// Store each credential field as encrypted secrets
	prefix := fmt.Sprintf("credentials.%s", cred.Service)

	if cred.Username != "" {
		if err := secretsManager.SetEncrypted(prefix+".username", cred.Username); err != nil {
			return fmt.Errorf("failed to store username: %w", err)
		}
	}
	if cred.Password != "" {
		if err := secretsManager.SetEncrypted(prefix+".password", cred.Password); err != nil {
			return fmt.Errorf("failed to store password: %w", err)
		}
	}
	if cred.Token != "" {
		if err := secretsManager.SetEncrypted(prefix+".token", cred.Token); err != nil {
			return fmt.Errorf("failed to store token: %w", err)
		}
	}
	if cred.Host != "" {
		if err := secretsManager.Set(prefix+".host", cred.Host, false); err != nil {
			return fmt.Errorf("failed to store host: %w", err)
		}
	}
	if cred.Port != "" {
		if err := secretsManager.Set(prefix+".port", cred.Port, false); err != nil {
			return fmt.Errorf("failed to store port: %w", err)
		}
	}

	// Save to disk
	return secretsManager.Save()
}

// ListCredentials returns all stored credentials (without passwords)
func ListCredentials() ([]Credentials, error) {
	if err := initSecretsManager(); err != nil {
		return nil, err
	}

	var creds []Credentials
	services := []string{"postgres", "mysql", "milvus", "qdrant", "weaviate", "github"}

	for _, service := range services {
		prefix := fmt.Sprintf("credentials.%s", service)

		// Check if any credential exists for this service
		keys := secretsManager.List()
		hasCredential := false
		for _, key := range keys {
			if key == prefix+".username" || key == prefix+".token" {
				hasCredential = true
				break
			}
		}

		if hasCredential {
			username, _ := secretsManager.Get(prefix + ".username")
			host, _ := secretsManager.Get(prefix + ".host")
			port, _ := secretsManager.Get(prefix + ".port")

			cred := Credentials{
				Service:  service,
				Username: username,
				Host:     host,
				Port:     port,
			}
			creds = append(creds, cred)
		}
	}

	return creds, nil
}

// RemoveCredentials removes credentials for a specific service
func RemoveCredentials(service string) error {
	if err := initSecretsManager(); err != nil {
		return err
	}

	prefix := fmt.Sprintf("credentials.%s", service)
	keys := []string{
		prefix + ".username",
		prefix + ".password",
		prefix + ".token",
		prefix + ".host",
		prefix + ".port",
	}

	for _, key := range keys {
		if err := secretsManager.Delete(key); err != nil && err != secrets.ErrSecretNotFound {
			return fmt.Errorf("failed to delete %s: %w", key, err)
		}
	}

	return secretsManager.Save()
}

// ClearAllCredentials removes all stored credentials
func ClearAllCredentials() error {
	if err := initSecretsManager(); err != nil {
		return err
	}

	keys := secretsManager.List()
	for _, key := range keys {
		if strings.HasPrefix(key, "credentials.") {
			if err := secretsManager.Delete(key); err != nil && err != secrets.ErrSecretNotFound {
				return fmt.Errorf("failed to delete %s: %w", key, err)
			}
		}
	}

	return secretsManager.Save()
}

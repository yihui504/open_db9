package cli

import (
	"github.com/spf13/viper"
)

// Config is the global configuration instance
var Config = viper.New()

// AppConfig holds the application configuration
type AppConfig struct {
	// Database settings
	Database struct {
		DefaultProvider string `mapstructure:"default_provider"`
		Timeout         int    `mapstructure:"timeout"`
		MaxConnections  int    `mapstructure:"max_connections"`
	} `mapstructure:"database"`

	// Output settings
	Output struct {
		Format string `mapstructure:"format"`
		Dir    string `mapstructure:"dir"`
	} `mapstructure:"output"`

	// Logging settings
	Logging struct {
		Level  string `mapstructure:"level"`
		File   string `mapstructure:"file"`
		Format string `mapstructure:"format"`
	} `mapstructure:"logging"`
}

// GetConfig returns the current application configuration
func GetConfig() *AppConfig {
	config := &AppConfig{}
	if err := Config.Unmarshal(config); err != nil {
		// Return defaults if unmarshal fails
		return getDefaultConfig()
	}
	return config
}

// getDefaultConfig returns default configuration values
func getDefaultConfig() *AppConfig {
	config := &AppConfig{}
	config.Database.DefaultProvider = "postgres"
	config.Database.Timeout = 30
	config.Database.MaxConnections = 10
	config.Output.Format = "table"
	config.Output.Dir = "./results"
	config.Logging.Level = "info"
	config.Logging.Format = "text"
	return config
}

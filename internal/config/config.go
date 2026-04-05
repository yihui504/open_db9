package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath" // Used for filepath.Clean
	"strconv"
	"strings"
	"sync"

	"gopkg.in/yaml.v3" // Used for yaml.Unmarshal
)

var (
	instance *Config
	once     sync.Once
)

// Config holds application configuration
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Auth     AuthConfig
	CORS     CORSConfig
	RateLimit RateLimitConfig
}

// Get returns the singleton config instance
func Get() *Config {
	return instance
}

// ServerConfig holds server-specific configuration
type ServerConfig struct {
	Port    string
	Host    string
	Timeout int
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Driver   string
	Host     string
	Port     int
	User     string
	Password string
	Database string
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	Secret    string
	ExpiresIn int
}

// CORSConfig holds CORS configuration
type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	ExposedHeaders   []string
	AllowCredentials bool
	MaxAge           int
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	Enabled          bool
	RequestsPerMinute int
	Burst            int
}

// ConfigFile represents the YAML configuration file structure
type ConfigFile struct {
	Server   ServerConfigFile   `yaml:"server"`
	Database DatabaseConfigFile `yaml:"database"`
	Auth     AuthConfigFile     `yaml:"auth"`
	CORS     CORSConfigFile     `yaml:"cors"`
	RateLimit RateLimitConfigFile `yaml:"rate_limit"`
}

// ServerConfigFile holds server-specific configuration from YAML
type ServerConfigFile struct {
	Port    *string `yaml:"port"`
	Host    *string `yaml:"host"`
	Timeout *int    `yaml:"timeout"`
}

// DatabaseConfigFile holds database configuration from YAML
type DatabaseConfigFile struct {
	Driver   *string `yaml:"driver"`
	Host     *string `yaml:"host"`
	Port     *int    `yaml:"port"`
	User     *string `yaml:"user"`
	Password *string `yaml:"password"`
	Database *string `yaml:"database"`
}

// AuthConfigFile holds authentication configuration from YAML
type AuthConfigFile struct {
	Secret    *string `yaml:"secret"`
	ExpiresIn *int    `yaml:"expires_in"`
}

// CORSConfigFile holds CORS configuration from YAML
type CORSConfigFile struct {
	AllowedOrigins   *string `yaml:"allowed_origins"`
	AllowedMethods   *string `yaml:"allowed_methods"`
	AllowedHeaders   *string `yaml:"allowed_headers"`
	ExposedHeaders   *string `yaml:"exposed_headers"`
	AllowCredentials *bool   `yaml:"allow_credentials"`
	MaxAge           *int    `yaml:"max_age"`
}

// RateLimitConfigFile holds rate limiting configuration from YAML
type RateLimitConfigFile struct {
	Enabled           *bool `yaml:"enabled"`
	RequestsPerMinute *int  `yaml:"requests_per_minute"`
	Burst             *int  `yaml:"burst"`
}

// LoadFromFile loads configuration from a YAML file
func LoadFromFile(path string) (*Config, error) {
	// Read the YAML file
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	var yamlConfig ConfigFile
	if err := yaml.Unmarshal(data, &yamlConfig); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Convert to Config struct
	config := &Config{
		Server: ServerConfig{
			Port:    getEnv("PORT", toString(yamlConfig.Server.Port, "8080")),
			Host:    getEnv("HOST", toString(yamlConfig.Server.Host, "localhost")),
			Timeout: getEnvAsInt("TIMEOUT", toInt(yamlConfig.Server.Timeout, 30)),
		},
		Database: DatabaseConfig{
			Driver:   getEnv("DB_DRIVER", toString(yamlConfig.Database.Driver, "postgres")),
			Host:     getEnv("DB_HOST", toString(yamlConfig.Database.Host, "localhost")),
			Port:     getEnvAsInt("DB_PORT", toInt(yamlConfig.Database.Port, 5432)),
			User:     getEnv("DB_USER", toString(yamlConfig.Database.User, "db9")),
			Password: getEnv("DB_PASSWORD", toString(yamlConfig.Database.Password, "")),
			Database: getEnv("DB_NAME", toString(yamlConfig.Database.Database, "db9")),
		},
		Auth: AuthConfig{
			Secret:    getEnv("AUTH_SECRET", toString(yamlConfig.Auth.Secret, "")),
			ExpiresIn: getEnvAsInt("AUTH_EXPIRES_IN", toInt(yamlConfig.Auth.ExpiresIn, 3600)),
		},
		CORS: CORSConfig{
			AllowedOrigins:   parseStringList(getEnv("CORS_ALLOWED_ORIGINS", toString(yamlConfig.CORS.AllowedOrigins, "http://localhost:3000,http://localhost:8080"))),
			AllowedMethods:   parseStringList(getEnv("CORS_ALLOWED_METHODS", toString(yamlConfig.CORS.AllowedMethods, "GET,POST,PUT,DELETE,OPTIONS"))),
			AllowedHeaders:   parseStringList(getEnv("CORS_ALLOWED_HEADERS", toString(yamlConfig.CORS.AllowedHeaders, "Content-Type,Authorization"))),
			ExposedHeaders:   parseStringList(getEnv("CORS_EXPOSED_HEADERS", toString(yamlConfig.CORS.ExposedHeaders, ""))),
			AllowCredentials: getEnvAsBool("CORS_ALLOW_CREDENTIALS", toBool(yamlConfig.CORS.AllowCredentials, false)),
			MaxAge:           getEnvAsInt("CORS_MAX_AGE", toInt(yamlConfig.CORS.MaxAge, 86400)),
		},
		RateLimit: RateLimitConfig{
			Enabled:           getEnvAsBool("RATE_LIMIT_ENABLED", toBool(yamlConfig.RateLimit.Enabled, false)),
			RequestsPerMinute: getEnvAsInt("RATE_LIMIT_RPM", toInt(yamlConfig.RateLimit.RequestsPerMinute, 100)),
			Burst:             getEnvAsInt("RATE_LIMIT_BURST", toInt(yamlConfig.RateLimit.Burst, 20)),
		},
	}

	return config, nil
}

// Load loads configuration from environment variables or YAML file and sets the singleton
// Environment variables override YAML values
func Load() *Config {
	once.Do(func() {
		var err error

		// Check if CONFIG_FILE environment variable is set
		configFile := os.Getenv("CONFIG_FILE")
		if configFile != "" {
			instance, err = LoadFromFile(configFile)
			if err != nil {
				// If loading from file fails, fall back to environment variables
				fmt.Printf("Warning: failed to load config file '%s': %v. Using environment variables.\n", configFile, err)
			} else {
				// Successfully loaded from file
				return
			}
		}

		// Load from environment variables with defaults
		instance = &Config{
			Server: ServerConfig{
				Port:    getEnv("PORT", "8080"),
				Host:    getEnv("HOST", "localhost"),
				Timeout: getEnvAsInt("TIMEOUT", 30),
			},
			Database: DatabaseConfig{
				Driver:   getEnv("DB_DRIVER", "postgres"),
				Host:     getEnv("DB_HOST", "localhost"),
				Port:     getEnvAsInt("DB_PORT", 5432),
				User:     getEnv("DB_USER", "db9"),
				Password: getEnv("DB_PASSWORD", ""),
				Database: getEnv("DB_NAME", "db9"),
			},
			Auth: AuthConfig{
				Secret:    getEnv("AUTH_SECRET", ""),
				ExpiresIn: getEnvAsInt("AUTH_EXPIRES_IN", 3600),
			},
			CORS: CORSConfig{
				AllowedOrigins:   parseStringList(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:3000,http://localhost:8080")),
				AllowedMethods:   parseStringList(getEnv("CORS_ALLOWED_METHODS", "GET,POST,PUT,DELETE,OPTIONS")),
				AllowedHeaders:   parseStringList(getEnv("CORS_ALLOWED_HEADERS", "Content-Type,Authorization")),
				ExposedHeaders:   parseStringList(getEnv("CORS_EXPOSED_HEADERS", "")),
				AllowCredentials: getEnvAsBool("CORS_ALLOW_CREDENTIALS", false),
				MaxAge:           getEnvAsInt("CORS_MAX_AGE", 86400),
			},
			RateLimit: RateLimitConfig{
				Enabled:           getEnvAsBool("RATE_LIMIT_ENABLED", false),
				RequestsPerMinute: getEnvAsInt("RATE_LIMIT_RPM", 100),
				Burst:             getEnvAsInt("RATE_LIMIT_BURST", 20),
			},
		}
	})
	return instance
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}

func parseStringList(value string) []string {
	if value == "" {
		return []string{}
	}
	return strings.Split(value, ",")
}

func toString(ptr *string, defaultVal string) string {
	if ptr != nil {
		return *ptr
	}
	return defaultVal
}

func toInt(ptr *int, defaultVal int) int {
	if ptr != nil {
		return *ptr
	}
	return defaultVal
}

func toBool(ptr *bool, defaultVal bool) bool {
	if ptr != nil {
		return *ptr
	}
	return defaultVal
}

// Validate validates the configuration and returns an error if invalid
func (c *Config) Validate() error {
	// Validate AUTH_SECRET
	if c.Auth.Secret == "" {
		return errors.New("AUTH_SECRET environment variable must be set")
	}
	if len(c.Auth.Secret) < 32 {
		return fmt.Errorf("AUTH_SECRET must be at least 32 characters long, got %d", len(c.Auth.Secret))
	}
	// Check for weak patterns
	weakPatterns := []string{"change-me", "secret", "password", "123456"}
	lowerSecret := strings.ToLower(c.Auth.Secret)
	for _, pattern := range weakPatterns {
		if strings.Contains(lowerSecret, pattern) {
			return fmt.Errorf("AUTH_SECRET contains weak pattern '%s'", pattern)
		}
	}
	return nil
}

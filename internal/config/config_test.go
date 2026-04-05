package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	// Save original env values
	origPort := os.Getenv("PORT")
	origHost := os.Getenv("HOST")
	origTimeout := os.Getenv("TIMEOUT")
	origDBDriver := os.Getenv("DB_DRIVER")
	origDBHost := os.Getenv("DB_HOST")
	origDBPort := os.Getenv("DB_PORT")
	origDBUser := os.Getenv("DB_USER")
	origDBPassword := os.Getenv("DB_PASSWORD")
	origDBName := os.Getenv("DB_NAME")
	origAuthSecret := os.Getenv("AUTH_SECRET")
	origAuthExpiresIn := os.Getenv("AUTH_EXPIRES_IN")

	// Restore after test
	defer func() {
		os.Setenv("PORT", origPort)
		os.Setenv("HOST", origHost)
		os.Setenv("TIMEOUT", origTimeout)
		os.Setenv("DB_DRIVER", origDBDriver)
		os.Setenv("DB_HOST", origDBHost)
		os.Setenv("DB_PORT", origDBPort)
		os.Setenv("DB_USER", origDBUser)
		os.Setenv("DB_PASSWORD", origDBPassword)
		os.Setenv("DB_NAME", origDBName)
		os.Setenv("AUTH_SECRET", origAuthSecret)
		os.Setenv("AUTH_EXPIRES_IN", origAuthExpiresIn)
	}()

	// Clear env to test defaults
	os.Unsetenv("PORT")
	os.Unsetenv("HOST")
	os.Unsetenv("TIMEOUT")
	os.Unsetenv("DB_DRIVER")
	os.Unsetenv("DB_HOST")
	os.Unsetenv("DB_PORT")
	os.Unsetenv("DB_USER")
	os.Unsetenv("DB_PASSWORD")
	os.Unsetenv("DB_NAME")
	os.Unsetenv("AUTH_SECRET")
	os.Unsetenv("AUTH_EXPIRES_IN")

	cfg := Load()

	// Test default values
	if cfg.Server.Port != "8080" {
		t.Errorf("expected default Port 8080, got %s", cfg.Server.Port)
	}
	if cfg.Server.Host != "localhost" {
		t.Errorf("expected default Host localhost, got %s", cfg.Server.Host)
	}
	if cfg.Server.Timeout != 30 {
		t.Errorf("expected default Timeout 30, got %d", cfg.Server.Timeout)
	}
	if cfg.Database.Driver != "postgres" {
		t.Errorf("expected default Driver postgres, got %s", cfg.Database.Driver)
	}
	if cfg.Database.Host != "localhost" {
		t.Errorf("expected default DB Host localhost, got %s", cfg.Database.Host)
	}
	if cfg.Database.Port != 5432 {
		t.Errorf("expected default DB Port 5432, got %d", cfg.Database.Port)
	}
	if cfg.Database.User != "db9" {
		t.Errorf("expected default DB User db9, got %s", cfg.Database.User)
	}
	if cfg.Database.Database != "db9" {
		t.Errorf("expected default DB Name db9, got %s", cfg.Database.Database)
	}
	if cfg.Auth.Secret != "" {
		t.Errorf("expected default Auth Secret empty, got %s", cfg.Auth.Secret)
	}
	if cfg.Auth.ExpiresIn != 3600 {
		t.Errorf("expected default Auth ExpiresIn 3600, got %d", cfg.Auth.ExpiresIn)
	}
}

func TestLoadWithEnv(t *testing.T) {
	// Skip this test if config is already loaded (singleton pattern)
	if Get() != nil {
		t.Skip("Config already loaded (singleton), skipping env override test")
		return
	}

	// Save original env values
	origPort := os.Getenv("PORT")
	origTimeout := os.Getenv("TIMEOUT")
	origDBPort := os.Getenv("DB_PORT")

	// Restore after test
	defer func() {
		os.Setenv("PORT", origPort)
		os.Setenv("TIMEOUT", origTimeout)
		os.Setenv("DB_PORT", origDBPort)
	}()

	// Set custom env values
	os.Setenv("PORT", "9000")
	os.Setenv("TIMEOUT", "60")
	os.Setenv("DB_PORT", "3306")

	cfg := Load()

	// Test env values are loaded
	if cfg.Server.Port != "9000" {
		t.Errorf("expected Port 9000 from env, got %s", cfg.Server.Port)
	}
	if cfg.Server.Timeout != 60 {
		t.Errorf("expected Timeout 60 from env, got %d", cfg.Server.Timeout)
	}
	if cfg.Database.Port != 3306 {
		t.Errorf("expected DB Port 3306 from env, got %d", cfg.Database.Port)
	}
}

func TestGetEnv(t *testing.T) {
	// Save and restore
	origTestVar := os.Getenv("TEST_CONFIG_VAR")
	defer func() {
		if origTestVar != "" {
			os.Setenv("TEST_CONFIG_VAR", origTestVar)
		} else {
			os.Unsetenv("TEST_CONFIG_VAR")
		}
	}()

	// Test with env set
	os.Setenv("TEST_CONFIG_VAR", "test_value")
	if result := getEnv("TEST_CONFIG_VAR", "default"); result != "test_value" {
		t.Errorf("expected test_value, got %s", result)
	}

	// Test with env unset
	os.Unsetenv("TEST_CONFIG_VAR")
	if result := getEnv("TEST_CONFIG_VAR", "default"); result != "default" {
		t.Errorf("expected default, got %s", result)
	}
}

func TestGetEnvAsInt(t *testing.T) {
	// Save and restore
	origTestVar := os.Getenv("TEST_CONFIG_INT")
	defer func() {
		if origTestVar != "" {
			os.Setenv("TEST_CONFIG_INT", origTestVar)
		} else {
			os.Unsetenv("TEST_CONFIG_INT")
		}
	}()

	// Test with valid int
	os.Setenv("TEST_CONFIG_INT", "12345")
	if result := getEnvAsInt("TEST_CONFIG_INT", 100); result != 12345 {
		t.Errorf("expected 12345, got %d", result)
	}

	// Test with invalid int
	os.Setenv("TEST_CONFIG_INT", "not_a_number")
	if result := getEnvAsInt("TEST_CONFIG_INT", 100); result != 100 {
		t.Errorf("expected default 100 for invalid int, got %d", result)
	}

	// Test with env unset
	os.Unsetenv("TEST_CONFIG_INT")
	if result := getEnvAsInt("TEST_CONFIG_INT", 100); result != 100 {
		t.Errorf("expected default 100 when unset, got %d", result)
	}

	// Test with negative int
	os.Setenv("TEST_CONFIG_INT", "-100")
	if result := getEnvAsInt("TEST_CONFIG_INT", 0); result != -100 {
		t.Errorf("expected -100, got %d", result)
	}

	// Test with zero
	os.Setenv("TEST_CONFIG_INT", "0")
	if result := getEnvAsInt("TEST_CONFIG_INT", 100); result != 0 {
		t.Errorf("expected 0, got %d", result)
	}
}

func TestParseStringList(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: []string{},
		},
		{
			name:     "single item",
			input:    "item1",
			expected: []string{"item1"},
		},
		{
			name:     "comma separated",
			input:    "item1,item2,item3",
			expected: []string{"item1", "item2", "item3"},
		},
		{
			name:     "with spaces",
			input:    "item1, item2, item3",
			expected: []string{"item1", " item2", " item3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseStringList(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("expected length %d, got %d", len(tt.expected), len(result))
				return
			}
			for i, item := range result {
				if item != tt.expected[i] {
					t.Errorf("item %d: expected %s, got %s", i, tt.expected[i], item)
				}
			}
		})
	}
}

func TestGetEnvAsBool(t *testing.T) {
	// Save and restore
	origTestVar := os.Getenv("TEST_CONFIG_BOOL")
	defer func() {
		if origTestVar != "" {
			os.Setenv("TEST_CONFIG_BOOL", origTestVar)
		} else {
			os.Unsetenv("TEST_CONFIG_BOOL")
		}
	}()

	tests := []struct {
		name      string
		value     string
		defaultVal bool
		expected  bool
	}{
		{
			name:      "true lowercase",
			value:     "true",
			defaultVal: false,
			expected:  true,
		},
		{
			name:      "true uppercase",
			value:     "TRUE",
			defaultVal: false,
			expected:  true,
		},
		{
			name:      "false lowercase",
			value:     "false",
			defaultVal: true,
			expected:  false,
		},
		{
			name:      "false uppercase",
			value:     "FALSE",
			defaultVal: true,
			expected:  false,
		},
		{
			name:      "1 as true",
			value:     "1",
			defaultVal: false,
			expected:  true,
		},
		{
			name:      "0 as false",
			value:     "0",
			defaultVal: true,
			expected:  false,
		},
		{
			name:      "invalid uses default",
			value:     "invalid",
			defaultVal: true,
			expected:  true,
		},
		{
			name:      "empty uses default",
			value:     "",
			defaultVal: false,
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != "" {
				os.Setenv("TEST_CONFIG_BOOL", tt.value)
			} else {
				os.Unsetenv("TEST_CONFIG_BOOL")
			}
			result := getEnvAsBool("TEST_CONFIG_BOOL", tt.defaultVal)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: Config{
				Auth: AuthConfig{
					Secret:    "this-is-a-very-strong-key-without-weak-patterns-32",
					ExpiresIn: 3600,
				},
			},
			wantErr: false,
		},
		{
			name: "empty auth secret",
			config: Config{
				Auth: AuthConfig{
					Secret: "",
				},
			},
			wantErr: true,
		},
		{
			name: "weak auth secret - too short",
			config: Config{
				Auth: AuthConfig{
					Secret: "weak",
				},
			},
			wantErr: true,
		},
		{
			name: "weak auth secret - contains weak pattern",
			config: Config{
				Auth: AuthConfig{
					Secret: "change-me-this-is-not-secure-enough-but-long-enough",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

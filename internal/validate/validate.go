package validate

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	// Valid PostgreSQL identifier regex (simplified)
	// Allows: letters, digits, underscores, must start with letter or underscore
	// Max length: 63 characters (PostgreSQL limit)
	validIdentifierRE = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]{0,62}$`)

	// System schemas that should be handled carefully
	systemSchemas = map[string]bool{
		"pg_catalog":      true,
		"information_schema": true,
		"pg_toast":        true,
		"pg_temp_":        true,
		"pg_toast_temp_":  true,
	}
)

// IsValidIdentifier checks if a string is a valid PostgreSQL identifier
func IsValidIdentifier(name string) bool {
	if name == "" {
		return false
	}

	// Check if it's quoted (double quotes) - if so, more flexible but still need validation
	if strings.HasPrefix(name, "\"") && strings.HasSuffix(name, "\"") {
		unquoted := name[1 : len(name)-1]
		// Check for escaped quotes
		if strings.Contains(unquoted, "\"") && !strings.Contains(unquoted, `""`) {
			return false
		}
		return len(unquoted) <= 63
	}

	return validIdentifierRE.MatchString(name)
}

// IsValidSchemaName checks if a schema name is valid
func IsValidSchemaName(name string) error {
	if !IsValidIdentifier(name) {
		return fmt.Errorf("invalid schema name: '%s'", name)
	}

	// Warn about system schemas (but allow access with caution)
	if IsSystemSchema(name) {
		return fmt.Errorf("system schema access: '%s' requires special handling", name)
	}

	return nil
}

// IsValidTableName checks if a table name is valid
func IsValidTableName(name string) error {
	if !IsValidIdentifier(name) {
		return fmt.Errorf("invalid table name: '%s'", name)
	}
	return nil
}

// IsSystemSchema checks if a schema is a system schema
func IsSystemSchema(name string) bool {
	// Direct match
	if systemSchemas[name] {
		return true
	}

	// Prefix match for temporary schemas
	if strings.HasPrefix(name, "pg_temp_") || strings.HasPrefix(name, "pg_toast_temp_") {
		return true
	}

	return false
}

// SanitizeQueryText removes or redacts sensitive information from query text
func SanitizeQueryText(query string, maxLength int) string {
	// Redact common sensitive patterns first
	query = redactPasswords(query)
	query = redactTokens(query)
	query = redactAPIKeys(query)

	if len(query) > maxLength {
		// Truncate and add ellipsis
		if maxLength > 3 {
			return query[:maxLength-3] + "..."
		}
		return query[:maxLength]
	}

	return query
}

// redactPasswords removes potential password values from queries
func redactPasswords(query string) string {
	// Common password patterns in queries
	patterns := []string{
		`password\s*=\s*'[^']*'`,
		`password\s*=\s*\$[^$]*\$`,
		`pwd\s*=\s*'[^']*'`,
		`PASS:\s*\S+`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(`(?i)`+pattern)
		query = re.ReplaceAllString(query, "password='***'")
	}

	return query
}

// redactTokens removes potential token values
func redactTokens(query string) string {
	// Token patterns
	patterns := []string{
		`token\s*=\s*'[^']*'`,
		`bearer\s+[A-Za-z0-9\-._~+/]+=*`,
		`authorization\s*:\s*Bearer\s+\S+`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(`(?i)`+pattern)
		query = re.ReplaceAllString(query, "token='***'")
	}

	return query
}

// redactAPIKeys removes potential API key values
func redactAPIKeys(query string) string {
	// API key patterns
	patterns := []string{
		`api[_-]?key\s*=\s*'[^']*'`,
		`apikey\s*=\s*'[^']*'`,
		`secret\s*=\s*'[^']*'`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(`(?i)`+pattern)
		query = re.ReplaceAllString(query, "***='***'")
	}

	return query
}

// ValidateLimit checks if a limit value is within acceptable range
func ValidateLimit(limit int64, minVal, maxVal int64) error {
	if limit < minVal {
		return fmt.Errorf("limit must be at least %d", minVal)
	}
	if limit > maxVal {
		return fmt.Errorf("limit must not exceed %d", maxVal)
	}
	return nil
}

// ValidateThreshold checks if a threshold value is valid
func ValidateThreshold(threshold int64, minVal int64) error {
	if threshold < minVal {
		return fmt.Errorf("threshold must be at least %d", minVal)
	}
	return nil
}

// SafeSchemaName returns a safe version of a schema name for error messages
func SafeSchemaName(name string) string {
	if len(name) > 30 {
		return name[:27] + "..."
	}
	return name
}

// SafeTableName returns a safe version of a table name for error messages
func SafeTableName(name string) string {
	if len(name) > 30 {
		return name[:27] + "..."
	}
	return name
}

// ValidateDatabaseID validates a database ID
func ValidateDatabaseID(dbID string) error {
	if dbID == "" {
		return fmt.Errorf("database ID cannot be empty")
	}
	if len(dbID) > 64 {
		return fmt.Errorf("database ID too long (max 64 characters)")
	}
	// Check for path traversal attempts
	if strings.ContainsAny(dbID, "\\/:") {
		return fmt.Errorf("database ID contains invalid characters")
	}
	return nil
}

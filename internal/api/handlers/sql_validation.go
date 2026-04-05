package handlers

import (
	"fmt"
	"strings"
)

var dangerousKeywords = []string{
	"DROP", "DELETE", "TRUNCATE", "ALTER", "CREATE", "INSERT",
	"UPDATE", "GRANT", "REVOKE", "EXECUTE", "SCRIPT",
}

// validateSQLQuery performs basic SQL injection prevention
func validateSQLQuery(query string) error {
	query = strings.TrimSpace(query)
	if query == "" {
		return fmt.Errorf("empty query")
	}

	// Check query length (prevent DOS)
	if len(query) > 100000 { // 100KB max
		return fmt.Errorf("query too large")
	}

	upperQuery := strings.ToUpper(query)

	// Block multi-statement queries
	if strings.Count(upperQuery, ";") > 1 {
		return fmt.Errorf("multi-statement queries not allowed")
	}

	// Check for dangerous keywords in non-SELECT queries
	for _, keyword := range dangerousKeywords {
		if strings.HasPrefix(upperQuery, keyword) {
			return fmt.Errorf("%s statements not allowed via API", keyword)
		}
	}

	// Only allow SELECT, SHOW, DESCRIBE, EXPLAIN at start
	allowedPrefixes := []string{"SELECT", "SHOW", "DESCRIBE", "DESC", "EXPLAIN", "WITH"}
	hasAllowedPrefix := false
	for _, prefix := range allowedPrefixes {
		if strings.HasPrefix(upperQuery, prefix) {
			hasAllowedPrefix = true
			break
		}
	}

	if !hasAllowedPrefix {
		return fmt.Errorf("only SELECT, SHOW, DESCRIBE, EXPLAIN queries allowed")
	}

	return nil
}

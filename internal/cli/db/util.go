package db

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
)

// getEnv retrieves environment variables with defaults
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// getDBConnection creates a direct pgx connection to a database
func getDBConnection(ctx context.Context, databaseID string) (*pgx.Conn, error) {
	// Build connection string from environment variables
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "postgres")
	password := getEnv("DB_PASSWORD", "")
	database := databaseID // Use databaseID as database name

	connString := fmt.Sprintf("host=%s port=%s user=%s password=%s database=%s sslmode=disable",
		host, port, user, password, database)

	return pgx.Connect(ctx, connString)
}

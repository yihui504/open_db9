package e2e

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// TestEnvironment represents the E2E test environment
type TestEnvironment struct {
	PostgreSQLContainer testcontainers.Container
	ConnString          string
	Context             context.Context
	Cancel              context.CancelFunc
}

// SetupTestEnvironment creates a new test environment with PostgreSQL
func SetupTestEnvironment(t *testing.T) *TestEnvironment {
	ctx := context.Background()
	cancelCtx, cancel := context.WithCancel(ctx)

	// Start PostgreSQL container
	pgContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "postgres:15-alpine",
			ExposedPorts: []string{"5432/tcp"},
			Env: map[string]string{
				"POSTGRES_USER":     "testuser",
				"POSTGRES_PASSWORD": "testpass",
				"POSTGRES_DB":       "testdb",
			},
			WaitingFor: wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		t.Fatalf("Failed to start PostgreSQL container: %v", err)
	}

	// Get connection string
	host, err := pgContainer.Host(ctx)
	if err != nil {
		t.Fatalf("Failed to get container host: %v", err)
	}

	port, err := pgContainer.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatalf("Failed to get container port: %v", err)
	}

	connString := fmt.Sprintf("host=%s port=%d user=testuser password=testpass dbname=testdb sslmode=disable", host, port.Int())

	// Test connection
	conn, err := pgx.Connect(ctx, connString)
	if err != nil {
		t.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer conn.Close(ctx)

	return &TestEnvironment{
		PostgreSQLContainer: pgContainer,
		ConnString:          connString,
		Context:             cancelCtx,
		Cancel:              cancel,
	}
}

// TeardownTestEnvironment cleans up the test environment
func TeardownTestEnvironment(t *testing.T, env *TestEnvironment) {
	env.Cancel()

	if env.PostgreSQLContainer != nil {
		if err := env.PostgreSQLContainer.Terminate(env.Context); err != nil {
			t.Logf("Failed to terminate PostgreSQL container: %v", err)
		}
	}
}

// CreateTestDatabase creates a new test database
func CreateTestDatabase(t *testing.T, env *TestEnvironment, dbName string) {
	conn, err := pgx.Connect(env.Context, env.ConnString)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close(env.Context)

	_, err = conn.Exec(env.Context, fmt.Sprintf("CREATE DATABASE %s", pgx.Identifier{dbName}))
	if err != nil {
		t.Fatalf("Failed to create database %s: %v", dbName, err)
	}
}

// DropTestDatabase drops a test database
func DropTestDatabase(t *testing.T, env *TestEnvironment, dbName string) {
	conn, err := pgx.Connect(env.Context, env.ConnString)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close(env.Context)

	_, err = conn.Exec(env.Context, fmt.Sprintf("DROP DATABASE IF EXISTS %s", pgx.Identifier{dbName}))
	if err != nil {
		t.Logf("Failed to drop database %s: %v", dbName, err)
	}
}

// RunSQL executes SQL and returns the result
func RunSQL(t *testing.T, env *TestEnvironment, databaseName, sql string) ([][]interface{}, error) {
	connString := strings.Replace(env.ConnString, "testdb", databaseName, 1)

	conn, err := pgx.Connect(env.Context, connString)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}
	defer conn.Close(env.Context)

	rows, err := conn.Query(env.Context, sql)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var results [][]interface{}

	fieldDescriptions := rows.FieldDescriptions()
	for rows.Next() {
		row := make([]interface{}, len(fieldDescriptions))
		values, err := rows.Values()
		if err != nil {
			return nil, fmt.Errorf("failed to get values: %w", err)
		}
		for i, v := range values {
			row[i] = v
		}
		results = append(results, row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return results, nil
}

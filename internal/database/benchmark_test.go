package database

import (
	"context"
	"testing"
)

// BenchmarkConnectionPoolAcquire benchmarks connection acquisition
func BenchmarkConnectionPoolAcquire(b *testing.B) {
	ctx := context.Background()
	config := DefaultPoolConfig()
	config.MaxConnections = 50

	pool, err := NewConnectionPool(config)
	if err != nil {
		b.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		conn, err := pool.GetConnection(ctx)
		if err != nil {
			b.Fatalf("Failed to get connection: %v", err)
		}
		conn.Release()
	}
}

// BenchmarkConnectionPoolQuery benchmarks simple queries through the pool
func BenchmarkConnectionPoolQuery(b *testing.B) {
	ctx := context.Background()
	config := DefaultPoolConfig()
	config.MaxConnections = 50

	pool, err := NewConnectionPool(config)
	if err != nil {
		b.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Close()

	// Setup: create a test table
	pool.Execute(ctx, "DROP TABLE IF EXISTS bench_test")
	pool.Execute(ctx, "CREATE TABLE bench_test (id SERIAL, value TEXT)")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := pool.Execute(ctx, "INSERT INTO bench_test (value) VALUES ('test')")
		if err != nil {
			b.Fatalf("Failed to execute: %v", err)
		}
	}

	// Cleanup
	pool.Execute(ctx, "DROP TABLE bench_test")
}

// BenchmarkSnapshotCreation benchmarks snapshot creation operations
func BenchmarkSnapshotCreation(b *testing.B) {
	// This benchmark tests the snapshot creation performance
	// In a real scenario, this would involve pg_dump operations

	b.Run("SerialSnapshots", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			// Simulate snapshot creation
			simulateSnapshotCreation()
		}
	})
}

func simulateSnapshotCreation() {
	// Simulated work for snapshot creation
	data := make([]byte, 1024*1024) // 1MB of data
	for i := range data {
		data[i] = byte(i % 256)
	}
}

// BenchmarkMetricsCollection benchmarks metrics collection operations
func BenchmarkMetricsCollection(b *testing.B) {
	ctx := context.Background()
	config := DefaultPoolConfig()

	pool, err := NewConnectionPool(config)
	if err != nil {
		b.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Close()

	// Setup: create pg_stat_statements extension
	pool.Execute(ctx, "CREATE EXTENSION IF NOT EXISTS pg_stat_statements")

	collector := NewMetricsCollector(pool)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := collector.CollectDatabaseStats(ctx)
		if err != nil {
			// Skip if extension not available
			b.SkipNow()
		}
	}
}

// BenchmarkSchemaIntrospection benchmarks schema introspection
func BenchmarkSchemaIntrospection(b *testing.B) {
	ctx := context.Background()
	config := DefaultPoolConfig()

	pool, err := NewConnectionPool(config)
	if err != nil {
		b.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Close()

	// Setup: create test tables
	pool.Execute(ctx, "DROP TABLE IF EXISTS bench_users, bench_posts")
	pool.Execute(ctx, "CREATE TABLE bench_users (id SERIAL, name TEXT, email TEXT)")
	pool.Execute(ctx, "CREATE TABLE bench_posts (id SERIAL, title TEXT, content TEXT, user_id INTEGER)")
	pool.Execute(ctx, "CREATE INDEX idx_bench_users_email ON bench_users(email)")
	pool.Execute(ctx, "CREATE INDEX idx_bench_posts_user_id ON bench_posts(user_id)")

	introspector := NewSchemaIntrospector(pool)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := introspector.IntrospectSchema(ctx, "public")
		if err != nil {
			b.Fatalf("Failed to introspect schema: %v", err)
		}
	}

	// Cleanup
	pool.Execute(ctx, "DROP TABLE IF EXISTS bench_users, bench_posts")
}

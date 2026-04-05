-- Observability and Monitoring Schema
-- Migration: 004_observability
-- Description: Creates tables for metrics collection, slow query tracking, and connection pool monitoring

-- Metrics Samples table: Time-series metrics data
CREATE TABLE metrics_samples (
    id BIGSERIAL PRIMARY KEY,
    database_id UUID NOT NULL REFERENCES databases(id) ON DELETE CASCADE,
    metric_name VARCHAR(100) NOT NULL, -- cpu_usage, memory_usage, disk_io, connections, etc.
    metric_type VARCHAR(50) NOT NULL, -- gauge, counter, histogram
    metric_value DOUBLE PRECISION NOT NULL,
    metric_unit VARCHAR(50), -- percent, bytes, count, seconds, etc.
    labels JSONB NOT NULL DEFAULT '{}', -- Additional labels for aggregation
    sampled_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Slow Queries table: Track and analyze slow database queries
CREATE TABLE slow_queries (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    database_id UUID NOT NULL REFERENCES databases(id) ON DELETE CASCADE,
    query_text TEXT NOT NULL,
    query_fingerprint VARCHAR(255) NOT NULL, -- Normalized query pattern
    query_hash VARCHAR(64) NOT NULL, -- Hash of normalized query for grouping
    execution_duration_ms BIGINT NOT NULL,
    rows_affected INTEGER,
    rows_examined INTEGER,
    query_plan JSONB, -- EXPLAIN ANALYZE output
    user_name VARCHAR(255),
    client_ip VARCHAR(45),
    application_name VARCHAR(255),
    error_message TEXT,
    metadata JSONB NOT NULL DEFAULT '{}',
    executed_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Connection Pools table: Monitor connection pool status and health
CREATE TABLE connection_pools (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    database_id UUID NOT NULL REFERENCES databases(id) ON DELETE CASCADE,
    pool_name VARCHAR(255) NOT NULL,
    pool_type VARCHAR(50) NOT NULL, -- application, proxy, internal
    total_connections INTEGER NOT NULL DEFAULT 0,
    active_connections INTEGER NOT NULL DEFAULT 0,
    idle_connections INTEGER NOT NULL DEFAULT 0,
    waiting_clients INTEGER NOT NULL DEFAULT 0,
    max_connections INTEGER NOT NULL,
    min_connections INTEGER NOT NULL DEFAULT 0,
    connection_timeout_ms INTEGER,
    idle_timeout_ms INTEGER,
    max_lifetime_ms INTEGER,
    health_status VARCHAR(20) NOT NULL DEFAULT 'healthy', -- healthy, degraded, unhealthy
    last_health_check_at TIMESTAMP WITH TIME ZONE,
    metadata JSONB NOT NULL DEFAULT '{}',
    sampled_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create indexes for metrics_samples
CREATE INDEX idx_metrics_samples_database_id ON metrics_samples(database_id);
CREATE INDEX idx_metrics_samples_metric_name ON metrics_samples(metric_name);
CREATE INDEX idx_metrics_samples_metric_type ON metrics_samples(metric_type);
CREATE INDEX idx_metrics_samples_sampled_at ON metrics_samples(sampled_at DESC);
CREATE INDEX idx_metrics_samples_database_metric_time ON metrics_samples(database_id, metric_name, sampled_at DESC);

-- Create indexes for slow_queries
CREATE INDEX idx_slow_queries_database_id ON slow_queries(database_id);
CREATE INDEX idx_slow_queries_query_fingerprint ON slow_queries(query_fingerprint);
CREATE INDEX idx_slow_queries_query_hash ON slow_queries(query_hash);
CREATE INDEX idx_slow_queries_execution_duration_ms ON slow_queries(execution_duration_ms DESC);
CREATE INDEX idx_slow_queries_executed_at ON slow_queries(executed_at DESC);
CREATE INDEX idx_slow_queries_user_name ON slow_queries(user_name);
CREATE INDEX idx_slow_queries_database_duration_time ON slow_queries(database_id, execution_duration_ms DESC, executed_at DESC);

-- Create indexes for connection_pools
CREATE INDEX idx_connection_pools_database_id ON connection_pools(database_id);
CREATE INDEX idx_connection_pools_pool_name ON connection_pools(pool_name);
CREATE INDEX idx_connection_pools_pool_type ON connection_pools(pool_type);
CREATE INDEX idx_connection_pools_health_status ON connection_pools(health_status);
CREATE INDEX idx_connection_pools_sampled_at ON connection_pools(sampled_at DESC);

-- Create triggers for updated_at
CREATE TRIGGER update_connection_pools_updated_at BEFORE UPDATE ON connection_pools
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Create partitioning function for metrics_samples (optional, for large-scale deployments)
-- This creates a monthly partitioning scheme for metrics data
CREATE OR REPLACE FUNCTION create_metrics_partition(month_date DATE)
RETURNS VOID AS $$
DECLARE
    partition_name TEXT;
    start_date TEXT;
    end_date TEXT;
BEGIN
    partition_name := 'metrics_samples_' || TO_CHAR(month_date, 'YYYY_MM');
    start_date := TO_CHAR(month_date, 'YYYY-MM') || '-01';
    end_date := TO_CHAR(month_date + INTERVAL '1 month', 'YYYY-MM') || '-01';

    EXECUTE format(
        'CREATE TABLE IF NOT EXISTS %I PARTITION OF metrics_samples FOR VALUES FROM (%L) TO (%L)',
        partition_name, start_date, end_date
    );
END;
$$ LANGUAGE plpgsql;

-- Note: Partitioning requires the parent table to be created as a partitioned table.
-- Uncomment and modify the metrics_samples table definition above to enable partitioning:
-- CREATE TABLE metrics_samples (...) PARTITION BY RANGE (sampled_at);

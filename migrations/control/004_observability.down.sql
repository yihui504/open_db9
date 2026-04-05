-- Rollback Observability and Monitoring Schema
-- Migration: 004_observability (down)

-- Drop partitioning function
DROP FUNCTION IF EXISTS create_metrics_partition(DATE);

-- Drop triggers
DROP TRIGGER IF EXISTS update_connection_pools_updated_at ON connection_pools;

-- Drop indexes for connection_pools
DROP INDEX IF EXISTS idx_connection_pools_sampled_at;
DROP INDEX IF EXISTS idx_connection_pools_health_status;
DROP INDEX IF EXISTS idx_connection_pools_pool_type;
DROP INDEX IF EXISTS idx_connection_pools_pool_name;
DROP INDEX IF EXISTS idx_connection_pools_database_id;

-- Drop indexes for slow_queries
DROP INDEX IF EXISTS idx_slow_queries_database_duration_time;
DROP INDEX IF EXISTS idx_slow_queries_user_name;
DROP INDEX IF EXISTS idx_slow_queries_executed_at;
DROP INDEX IF EXISTS idx_slow_queries_execution_duration_ms;
DROP INDEX IF EXISTS idx_slow_queries_query_hash;
DROP INDEX IF EXISTS idx_slow_queries_query_fingerprint;
DROP INDEX IF EXISTS idx_slow_queries_database_id;

-- Drop indexes for metrics_samples
DROP INDEX IF EXISTS idx_metrics_samples_database_metric_time;
DROP INDEX IF EXISTS idx_metrics_samples_sampled_at;
DROP INDEX IF EXISTS idx_metrics_samples_metric_type;
DROP INDEX IF EXISTS idx_metrics_samples_metric_name;
DROP INDEX IF EXISTS idx_metrics_samples_database_id;

-- Drop tables (in correct order due to foreign keys)
DROP TABLE IF EXISTS connection_pools;
DROP TABLE IF EXISTS slow_queries;
DROP TABLE IF EXISTS metrics_samples;

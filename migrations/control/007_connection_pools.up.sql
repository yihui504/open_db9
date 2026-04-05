-- 007_connection_pools.up.sql
-- Create connection_pools table to store connection pool configurations
-- This table tracks pool settings for each database instance

CREATE TABLE IF NOT EXISTS connection_pools (
    id SERIAL PRIMARY KEY,
    database_id INTEGER NOT NULL REFERENCES databases(id) ON DELETE CASCADE,

    -- Pool configuration
    max_connections INTEGER NOT NULL DEFAULT 25,
    min_connections INTEGER NOT NULL DEFAULT 5,
    max_conn_lifetime_seconds INTEGER,
    max_conn_idle_time_seconds INTEGER,
    health_check_period_seconds INTEGER,
    connect_timeout_seconds INTEGER,

    -- Overflow behavior
    mode VARCHAR(10) NOT NULL DEFAULT 'wait' CHECK (mode IN ('reject', 'queue', 'wait')),
    max_queue_size INTEGER DEFAULT 100,
    queue_timeout_seconds INTEGER DEFAULT 30,

    -- Current statistics
    current_connections INTEGER DEFAULT 0,
    idle_connections INTEGER DEFAULT 0,
    acquire_count BIGINT DEFAULT 0,

    -- Metadata
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    -- Constraints
    CONSTRAINT connection_pools_unique_database UNIQUE (database_id)
);

-- Index for faster lookups by database
CREATE INDEX IF NOT EXISTS idx_connection_pools_database_id ON connection_pools(database_id);

-- Index for monitoring active pools
CREATE INDEX IF NOT EXISTS idx_connection_pools_created_at ON connection_pools(created_at);

-- Trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_connection_pools_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER connection_pools_updated_at
    BEFORE UPDATE ON connection_pools
    FOR EACH ROW
    EXECUTE FUNCTION update_connection_pools_updated_at();

-- Comments
COMMENT ON TABLE connection_pools IS 'Stores connection pool configuration for each database instance';
COMMENT ON COLUMN connection_pools.database_id IS 'Reference to the database this pool belongs to';
COMMENT ON COLUMN connection_pools.max_connections IS 'Maximum number of connections in the pool';
COMMENT ON COLUMN connection_pools.min_connections IS 'Minimum number of connections to maintain';
COMMENT ON COLUMN connection_pools.mode IS 'Overflow behavior: reject, queue, or wait';
COMMENT ON COLUMN connection_pools.current_connections IS 'Current number of active connections';
COMMENT ON COLUMN connection_pools.idle_connections IS 'Current number of idle connections';

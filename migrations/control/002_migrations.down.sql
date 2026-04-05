-- Rollback Migrations Tracking Table
-- Migration: 002_migrations (down)

-- Drop indexes
DROP INDEX IF EXISTS idx_migrations_batch;
DROP INDEX IF EXISTS idx_migrations_name;

-- Drop table
DROP TABLE IF EXISTS migrations;

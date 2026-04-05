-- Rollback Branch Support from Databases Table
-- Migration: 006_add_branch_support

-- Drop indexes
DROP INDEX IF EXISTS idx_databases_snapshot_id;
DROP INDEX IF EXISTS idx_databases_parent_id;

-- Drop columns
ALTER TABLE databases
DROP COLUMN IF EXISTS snapshot_id;

ALTER TABLE databases
DROP COLUMN IF EXISTS parent_id;

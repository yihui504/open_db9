-- Add Branch Support to Databases Table
-- Migration: 006_add_branch_support
-- Description: Adds parent_id and snapshot_id fields to support database branching

-- Add parent_id column to track branch relationships
ALTER TABLE databases
ADD COLUMN IF NOT EXISTS parent_id UUID REFERENCES databases(id) ON DELETE SET NULL;

-- Add snapshot_id column to track the snapshot used to create the branch
ALTER TABLE databases
ADD COLUMN IF NOT EXISTS snapshot_id UUID REFERENCES snapshots(id) ON DELETE SET NULL;

-- Create index for efficient branch queries
CREATE INDEX IF NOT EXISTS idx_databases_parent_id ON databases(parent_id);

-- Create index for snapshot-based queries
CREATE INDEX IF NOT EXISTS idx_databases_snapshot_id ON databases(snapshot_id);

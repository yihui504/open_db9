-- Rollback: FS9 Metadata Tables
-- Description: Drop fs9 file and directory metadata tables

-- Drop trigger and function
DROP TRIGGER IF EXISTS trigger_update_fs9_files_updated_at ON fs9_files;
DROP FUNCTION IF EXISTS update_fs9_files_updated_at();

-- Drop tables (indexes will be dropped automatically)
DROP TABLE IF EXISTS fs9_files;
DROP TABLE IF EXISTS fs9_directories;

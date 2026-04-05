-- Rollback FS9 Metadata Schema
-- Migration: 003_fs9_metadata (down)

-- Drop triggers
DROP TRIGGER IF EXISTS update_fs9_directories_updated_at ON fs9_directories;
DROP TRIGGER IF EXISTS update_fs9_files_updated_at ON fs9_files;

-- Drop trigger function
DROP FUNCTION IF EXISTS update_accessed_at_column();

-- Drop indexes for FS9 directories
DROP INDEX IF EXISTS idx_fs9_directories_is_root;
DROP INDEX IF EXISTS idx_fs9_directories_name;
DROP INDEX IF EXISTS idx_fs9_directories_path;
DROP INDEX IF EXISTS idx_fs9_directories_parent_directory_id;
DROP INDEX IF EXISTS idx_fs9_directories_database_id;

-- Drop indexes for FS9 files
DROP INDEX IF EXISTS idx_fs9_files_checksum;
DROP INDEX IF EXISTS idx_fs9_files_created_at;
DROP INDEX IF EXISTS idx_fs9_files_storage_backend;
DROP INDEX IF EXISTS idx_fs9_files_is_directory;
DROP INDEX IF EXISTS idx_fs9_files_mime_type;
DROP INDEX IF EXISTS idx_fs9_files_extension;
DROP INDEX IF EXISTS idx_fs9_files_name;
DROP INDEX IF EXISTS idx_fs9_files_path;
DROP INDEX IF EXISTS idx_fs9_files_parent_directory_id;
DROP INDEX IF EXISTS idx_fs9_files_database_id;

-- Drop tables (in correct order due to foreign keys)
DROP TABLE IF EXISTS fs9_files;
DROP TABLE IF EXISTS fs9_directories;

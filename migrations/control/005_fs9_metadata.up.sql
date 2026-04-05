-- Migration: FS9 Metadata Tables
-- Description: Create tables for fs9 file and directory metadata tracking

-- Create fs9_files table
CREATE TABLE fs9_files (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    database_id TEXT NOT NULL REFERENCES databases(id) ON DELETE CASCADE,
    path TEXT NOT NULL,
    filename TEXT NOT NULL,
    size BIGINT,
    storage_key TEXT,
    content_type TEXT,
    checksum TEXT,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(database_id, path)
);

-- Create fs9_directories table
CREATE TABLE fs9_directories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    database_id TEXT NOT NULL REFERENCES databases(id) ON DELETE CASCADE,
    path TEXT NOT NULL,
    parent_id UUID REFERENCES fs9_directories(id),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(database_id, path)
);

-- Create indexes for fs9_files
CREATE INDEX idx_fs9_files_database_id ON fs9_files(database_id);
CREATE INDEX idx_fs9_files_path ON fs9_files(path);
CREATE INDEX idx_fs9_files_filename ON fs9_files(filename);
CREATE INDEX idx_fs9_files_storage_key ON fs9_files(storage_key);
CREATE INDEX idx_fs9_files_content_type ON fs9_files(content_type);
CREATE INDEX idx_fs9_files_created_at ON fs9_files(created_at);
CREATE INDEX idx_fs9_files_metadata ON fs9_files USING GIN(metadata);

-- Create indexes for fs9_directories
CREATE INDEX idx_fs9_directories_database_id ON fs9_directories(database_id);
CREATE INDEX idx_fs9_directories_path ON fs9_directories(path);
CREATE INDEX idx_fs9_directories_parent_id ON fs9_directories(parent_id);
CREATE INDEX idx_fs9_directories_created_at ON fs9_directories(created_at);

-- Create trigger for updated_at on fs9_files
CREATE OR REPLACE FUNCTION update_fs9_files_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_fs9_files_updated_at
    BEFORE UPDATE ON fs9_files
    FOR EACH ROW
    EXECUTE FUNCTION update_fs9_files_updated_at();

-- Add comment for documentation
COMMENT ON TABLE fs9_files IS 'Metadata table for fs9 file storage';
COMMENT ON TABLE fs9_directories IS 'Metadata table for fs9 directory structure';
COMMENT ON COLUMN fs9_files.metadata IS 'Additional file metadata stored as JSONB';
COMMENT ON COLUMN fs9_files.checksum IS 'File checksum for integrity verification';
COMMENT ON COLUMN fs9_files.storage_key IS 'Key used in storage backend to retrieve file content';

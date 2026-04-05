-- FS9 Metadata Schema
-- Migration: 003_fs9_metadata
-- Description: Creates tables for FS9 filesystem metadata tracking

-- FS9 Directories table: Track directory structure in FS9 filesystem
CREATE TABLE fs9_directories (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    database_id UUID NOT NULL REFERENCES databases(id) ON DELETE CASCADE,
    path VARCHAR(1024) NOT NULL,
    name VARCHAR(255) NOT NULL,
    parent_directory_id UUID REFERENCES fs9_directories(id) ON DELETE CASCADE,
    metadata JSONB NOT NULL DEFAULT '{}',
    permissions JSONB NOT NULL DEFAULT '{}',
    is_root BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    created_by UUID REFERENCES anonymous_accounts(id) ON DELETE SET NULL,

    CONSTRAINT fs9_directories_database_path_unique UNIQUE (database_id, path),
    CONSTRAINT fs9_directories_no_self_parent CHECK (id != parent_directory_id)
);

-- FS9 Files table: Track individual files in FS9 filesystem
CREATE TABLE fs9_files (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    database_id UUID NOT NULL REFERENCES databases(id) ON DELETE CASCADE,
    path VARCHAR(1024) NOT NULL,
    name VARCHAR(255) NOT NULL,
    extension VARCHAR(50),
    size_bytes BIGINT NOT NULL DEFAULT 0,
    mime_type VARCHAR(100),
    checksum VARCHAR(64), -- SHA-256 hash
    metadata JSONB NOT NULL DEFAULT '{}',
    is_directory BOOLEAN NOT NULL DEFAULT false,
    parent_directory_id UUID REFERENCES fs9_directories(id) ON DELETE SET NULL,
    storage_backend VARCHAR(50) NOT NULL, -- local, s3, azure, gcs
    storage_path VARCHAR(1024) NOT NULL,
    version INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    accessed_at TIMESTAMP WITH TIME ZONE,
    created_by UUID REFERENCES anonymous_accounts(id) ON DELETE SET NULL,
    modified_by UUID REFERENCES anonymous_accounts(id) ON DELETE SET NULL,

    CONSTRAINT fs9_files_database_path_unique UNIQUE (database_id, path)
);

-- Create indexes for FS9 files
CREATE INDEX idx_fs9_files_database_id ON fs9_files(database_id);
CREATE INDEX idx_fs9_files_parent_directory_id ON fs9_files(parent_directory_id);
CREATE INDEX idx_fs9_files_path ON fs9_files(path);
CREATE INDEX idx_fs9_files_name ON fs9_files(name);
CREATE INDEX idx_fs9_files_extension ON fs9_files(extension);
CREATE INDEX idx_fs9_files_mime_type ON fs9_files(mime_type);
CREATE INDEX idx_fs9_files_is_directory ON fs9_files(is_directory);
CREATE INDEX idx_fs9_files_storage_backend ON fs9_files(storage_backend);
CREATE INDEX idx_fs9_files_created_at ON fs9_files(created_at DESC);
CREATE INDEX idx_fs9_files_checksum ON fs9_files(checksum);

-- Create indexes for FS9 directories
CREATE INDEX idx_fs9_directories_database_id ON fs9_directories(database_id);
CREATE INDEX idx_fs9_directories_parent_directory_id ON fs9_directories(parent_directory_id);
CREATE INDEX idx_fs9_directories_path ON fs9_directories(path);
CREATE INDEX idx_fs9_directories_name ON fs9_directories(name);
CREATE INDEX idx_fs9_directories_is_root ON fs9_directories(is_root);

-- Create triggers for updated_at
CREATE TRIGGER update_fs9_files_updated_at BEFORE UPDATE ON fs9_files
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_fs9_directories_updated_at BEFORE UPDATE ON fs9_directories
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Create function to update accessed_at timestamp
CREATE OR REPLACE FUNCTION update_accessed_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.accessed_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create trigger for accessed_at (will be called on SELECT operations via application logic)
-- Note: This trigger function is available but not automatically attached, as accessed_at
-- should be updated explicitly by the application layer when files are read

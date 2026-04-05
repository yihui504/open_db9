-- Control Plane Schema Initialization
-- Migration: 001_init_schema
-- Description: Creates core control plane tables for multi-tenant database management

-- Enable UUID extension if not already enabled
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Tenants table: Multi-tenant isolation
CREATE TABLE tenants (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) NOT NULL UNIQUE,
    plan VARCHAR(50) NOT NULL DEFAULT 'free', -- free, pro, enterprise
    settings JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Databases table: Managed database instances
CREATE TABLE databases (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) NOT NULL,
    engine VARCHAR(50) NOT NULL, -- postgresql, mysql, mongodb, etc.
    version VARCHAR(20) NOT NULL,
    region VARCHAR(50) NOT NULL,
    size VARCHAR(20) NOT NULL, -- small, medium, large, xlarge
    status VARCHAR(20) NOT NULL DEFAULT 'provisioning', -- provisioning, active, suspended, terminated
    connection_details JSONB NOT NULL DEFAULT '{}',
    storage_gb INTEGER NOT NULL DEFAULT 10,
    cpu_cores INTEGER NOT NULL DEFAULT 1,
    memory_mb INTEGER NOT NULL DEFAULT 1024,
    settings JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT databases_tenant_slug_unique UNIQUE (tenant_id, slug)
);

-- Snapshots table: Database backups and snapshots
CREATE TABLE snapshots (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    database_id UUID NOT NULL REFERENCES databases(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'pending', -- pending, completed, failed
    size_bytes BIGINT,
    snapshot_type VARCHAR(20) NOT NULL DEFAULT 'manual', -- manual, automated, scheduled
    storage_location VARCHAR(255) NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE
);

-- API Tokens table: Authentication for control plane API
CREATE TABLE api_tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    token_hash VARCHAR(255) NOT NULL UNIQUE,
    scopes JSONB NOT NULL DEFAULT '[]', -- Array of permitted scopes
    last_used_at TIMESTAMP WITH TIME ZONE,
    expires_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    revoked_at TIMESTAMP WITH TIME ZONE,
    is_active BOOLEAN NOT NULL DEFAULT true
);

-- Anonymous Accounts table: Temporary access for testing/trials
CREATE TABLE anonymous_accounts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    session_id VARCHAR(255) NOT NULL UNIQUE,
    tenant_id UUID REFERENCES tenants(id) ON DELETE SET NULL,
    capabilities JSONB NOT NULL DEFAULT '{}',
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    last_activity_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create indexes for performance
CREATE INDEX idx_databases_tenant_id ON databases(tenant_id);
CREATE INDEX idx_databases_status ON databases(status);
CREATE INDEX idx_databases_engine ON databases(engine);
CREATE INDEX idx_snapshots_database_id ON snapshots(database_id);
CREATE INDEX idx_snapshots_status ON snapshots(status);
CREATE INDEX idx_snapshots_created_at ON snapshots(created_at DESC);
CREATE INDEX idx_api_tokens_tenant_id ON api_tokens(tenant_id);
CREATE INDEX idx_api_tokens_token_hash ON api_tokens(token_hash);
CREATE INDEX idx_api_tokens_is_active ON api_tokens(is_active);
CREATE INDEX idx_anonymous_accounts_session_id ON anonymous_accounts(session_id);
CREATE INDEX idx_anonymous_accounts_expires_at ON anonymous_accounts(expires_at);
CREATE INDEX idx_anonymous_accounts_updated_at ON anonymous_accounts(updated_at);
CREATE INDEX idx_tenants_slug ON tenants(slug);

-- Create updated_at trigger function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create triggers for updated_at
CREATE TRIGGER update_tenants_updated_at BEFORE UPDATE ON tenants
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_databases_updated_at BEFORE UPDATE ON databases
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_anonymous_accounts_updated_at BEFORE UPDATE ON anonymous_accounts
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

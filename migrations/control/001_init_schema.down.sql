-- Rollback Control Plane Schema Initialization
-- Migration: 001_init_schema (down)

-- Drop triggers
DROP TRIGGER IF EXISTS update_databases_updated_at ON databases;
DROP TRIGGER IF EXISTS update_tenants_updated_at ON tenants;

-- Drop trigger function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop indexes
DROP INDEX IF EXISTS idx_tenants_slug;
DROP INDEX IF EXISTS idx_anonymous_accounts_expires_at;
DROP INDEX IF EXISTS idx_anonymous_accounts_session_id;
DROP INDEX IF EXISTS idx_api_tokens_is_active;
DROP INDEX IF EXISTS idx_api_tokens_token_hash;
DROP INDEX IF EXISTS idx_api_tokens_tenant_id;
DROP INDEX IF EXISTS idx_snapshots_created_at;
DROP INDEX IF EXISTS idx_snapshots_status;
DROP INDEX IF EXISTS idx_snapshots_database_id;
DROP INDEX IF EXISTS idx_databases_engine;
DROP INDEX IF EXISTS idx_databases_status;
DROP INDEX IF EXISTS idx_databases_tenant_id;

-- Drop tables (in correct order due to foreign keys)
DROP TABLE IF EXISTS anonymous_accounts;
DROP TABLE IF EXISTS api_tokens;
DROP TABLE IF EXISTS snapshots;
DROP TABLE IF EXISTS databases;
DROP TABLE IF EXISTS tenants;

-- Drop UUID extension (optional, only if no other tables use it)
-- DROP EXTENSION IF EXISTS "uuid-ossp";

-- Migrations Tracking Table
-- Migration: 002_migrations
-- Description: Creates schema migrations tracking table

-- Migrations table: Track applied schema migrations
CREATE TABLE migrations (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    batch INTEGER NOT NULL,
    applied_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create index for faster migration lookups
CREATE INDEX idx_migrations_name ON migrations(name);
CREATE INDEX idx_migrations_batch ON migrations(batch);

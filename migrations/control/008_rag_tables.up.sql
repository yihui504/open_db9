-- Migration 008: RAG System Tables
-- Creates schema for Retrieval-Augmented Generation system
-- with pgvector support and tenant-level isolation via RLS

-- Enable pgvector extension
CREATE EXTENSION IF NOT EXISTS vector;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Documents table (metadata only, content in FS9)
-- Note: database_id references databases(id) which is UUID
CREATE TABLE IF NOT EXISTS rag_documents (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    database_id UUID NOT NULL REFERENCES databases(id) ON DELETE CASCADE,
    fs9_key VARCHAR(512),                    -- FS9 storage key
    title VARCHAR(512) NOT NULL,
    filename VARCHAR(512) NOT NULL,
    content_type VARCHAR(128),
    total_chunks INTEGER DEFAULT 0,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(tenant_id, database_id, filename)
);

-- Chunks table (text chunks with embeddings)
CREATE TABLE IF NOT EXISTS rag_chunks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    document_id UUID NOT NULL REFERENCES rag_documents(id) ON DELETE CASCADE,
    chunk_index INTEGER NOT NULL,
    content TEXT NOT NULL,
    embedding vector(1024),                  -- ZhipuAI embedding-2 dimension
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(document_id, chunk_index)
);

-- Query history for analytics
CREATE TABLE IF NOT EXISTS rag_queries (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    database_id UUID NOT NULL REFERENCES databases(id) ON DELETE CASCADE,
    query_text TEXT NOT NULL,
    retrieved_chunks UUID[],
    response_text TEXT,
    latency_ms INTEGER,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for efficient retrieval
CREATE INDEX IF NOT EXISTS idx_rag_documents_tenant ON rag_documents(tenant_id);
CREATE INDEX IF NOT EXISTS idx_rag_documents_db ON rag_documents(database_id);
CREATE INDEX IF NOT EXISTS idx_rag_chunks_tenant ON rag_chunks(tenant_id);
CREATE INDEX IF NOT EXISTS idx_rag_chunks_document ON rag_chunks(document_id);
CREATE INDEX IF NOT EXISTS idx_rag_chunks_embedding ON rag_chunks USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);
CREATE INDEX IF NOT EXISTS idx_rag_queries_tenant ON rag_queries(tenant_id);
CREATE INDEX IF NOT EXISTS idx_rag_queries_db ON rag_queries(database_id, created_at DESC);

-- GIN index for metadata queries
CREATE INDEX IF NOT EXISTS idx_rag_documents_metadata ON rag_documents USING gin(metadata);
CREATE INDEX IF NOT EXISTS idx_rag_chunks_metadata ON rag_chunks USING gin(metadata);

-- Enable Row-Level Security for tenant isolation
ALTER TABLE rag_documents ENABLE ROW LEVEL SECURITY;
ALTER TABLE rag_chunks ENABLE ROW LEVEL SECURITY;
ALTER TABLE rag_queries ENABLE ROW LEVEL SECURITY;

-- RLS Policies: Users can only access their tenant's data
CREATE POLICY rag_documents_tenant_isolation ON rag_documents
    FOR ALL USING (tenant_id = current_setting('app.current_tenant_id')::UUID);

CREATE POLICY rag_chunks_tenant_isolation ON rag_chunks
    FOR ALL USING (tenant_id = current_setting('app.current_tenant_id')::UUID);

CREATE POLICY rag_queries_tenant_isolation ON rag_queries
    FOR ALL USING (tenant_id = current_setting('app.current_tenant_id')::UUID);

-- Function to set tenant context for RLS
CREATE OR REPLACE FUNCTION set_rag_tenant_id(tenant_uuid UUID)
RETURNS VOID AS $$
BEGIN
    PERFORM set_config('app.current_tenant_id', tenant_uuid::TEXT, FALSE);
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

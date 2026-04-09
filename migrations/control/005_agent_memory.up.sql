-- Agent Memory Tables for AI Agent Memory Layer (better than memory.md)
-- This migration adds tables for structured memory storage with vector search support

-- Main memories table: stores all agent memories with metadata
CREATE TABLE IF NOT EXISTS agent_memories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id VARCHAR(255) NOT NULL DEFAULT 'default',
    session_id VARCHAR(255),
    memory_type VARCHAR(50) NOT NULL DEFAULT 'fact',
    content TEXT NOT NULL,
    summary TEXT,
    tags TEXT[] DEFAULT '{}',
    importance_score FLOAT DEFAULT 0.5,
    access_count INTEGER DEFAULT 0,
    last_accessed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Vector embeddings table for semantic similarity search
CREATE TABLE IF NOT EXISTS memory_embeddings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    memory_id UUID NOT NULL REFERENCES agent_memories(id) ON DELETE CASCADE,
    embedding VECTOR(1536),
    embedding_model VARCHAR(100) DEFAULT 'text-embedding-ada-002',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_memories_agent_id ON agent_memories(agent_id);
CREATE INDEX idx_memories_session_id ON agent_memories(session_id);
CREATE INDEX idx_memories_type ON agent_memories(memory_type);
CREATE INDEX idx_memories_importance ON agent_memories(importance_score DESC);
CREATE INDEX idx_memories_created_at ON agent_memories(created_at DESC);
CREATE INDEX idx_memories_last_accessed ON agent_memories(last_accessed_at);
CREATE INDEX idx_memories_tags ON agent_memories USING GIN(tags);
CREATE INDEX idx_embeddings_vector ON memory_embeddings USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);

CREATE OR REPLACE FUNCTION update_memory_updated_at() 
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_memories_updated_at 
    BEFORE UPDATE ON agent_memories
    FOR EACH ROW EXECUTE FUNCTION update_memory_updated_at();

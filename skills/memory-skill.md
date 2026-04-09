# DB9 Memory Skill - Persistent Agent Memory Layer

> **Replace `memory.md` with structured, searchable, SQL-accessible memory storage.**

## Why Memory Skill?

| Feature | memory.md | DB9 Memory |
|---------|-----------|------------|
| Storage | Single text file | PostgreSQL database |
| Search | Keyword (Ctrl+F) | Semantic similarity |
| Filtering | Manual | By type, tags, date, agent |
| Size limit | File bloat issues | Unlimited |
| Sharing | Copy-paste | Multi-agent, multi-session |
| Analytics | None | SQL queries, metrics |
| Versioning | Git only | Snapshots + branches |

## Quick Start (3 Steps)

### 1. Store a Memory

```bash
# Via REST API
curl -X POST http://localhost:8080/api/v1/databases/1/memories \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "content": "User prefers TypeScript for frontend projects",
    "memory_type": "preference",
    "tags": ["tech", "frontend", "typescript"],
    "importance_score": 0.8
  }'
```

Response:
```json
{"success": true, "data": {"id": "uuid-...", "agent_id": "default", "memory_type": "preference"}}
```

### 2. Recall Memories (Semantic Search)

```bash
curl -X POST http://localhost:8080/api/v1/databases/1/memories/recall \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "query": "what does the user prefer for web development?",
    "top_k": 5,
    "filter": {
      "agent_id": "my-agent",
      "tags": ["tech"]
    }
  }'
```

### 3. List & Manage

```bash
# List all memories for an agent
curl "http://localhost:8080/api/v1/databases/1/memories?agent_id=my-agent" \
  -H "Authorization: Bearer YOUR_TOKEN"

# Filter by type
curl "http://localhost:8080/api/v1/databases/1/memories?memory_type=preference" \
  -H "Authorization: Bearer YOUR_TOKEN"

# Delete a memory
curl -X DELETE "http://localhost:8080/api/v1/databases/1/memories/<id>" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

## Memory Types

Use these types to categorize memories:

| Type | When to Use | Example |
|------|-------------|---------|
| `fact` | Factual information | "Project uses Go 1.25" |
| `preference` | User preferences | "User likes dark mode UI" |
| `context` | Project context | "This is a SaaS product" |
| `decision` | Decisions made | "Chose pgvector over pinecone" |
| `error` | Mistakes to avoid | "Don't use fmt.Sscanf for URLs" |
| `observation` | Noteworthy events | "Build takes ~45s" |
| `plan` | Future plans | "Add Redis caching next sprint" |

## Tag Strategy

Tags enable fast filtering. Use consistent tag categories:

- **Domain**: `frontend`, `backend`, `database`, `infra`
- **Tech**: `go`, `postgres`, `docker`, `react`
- **Priority**: `critical`, `important`, `nice-to-have`
- **Agent**: Agent identifier for multi-agent setups

```json
{"tags": ["backend", "postgres", "critical", "workbuddy"]}
```

## Importance Score (0.0 - 1.0)

Guide automatic summarization and cleanup:

| Score | Meaning | Action |
|-------|---------|--------|
| 0.9-1.0 | Critical | Never auto-delete |
| 0.7-0.8 | Important | Keep long-term |
| 0.4-0.6 | Normal | Default, may summarize |
| 0.1-0.3 | Low priority | Auto-summarize after N accesses |
| 0.0 | Temporary | Short-lived context |

## MCP Tools Reference

If MCP Server is configured, use these tools directly:

### `memory_store`
```json
{
  "content": "Memory text content",
  "type": "fact|preference|context|decision|error|observation|plan",
  "tags": ["tag1", "tag2"],
  "importance_score": 0.7,
  "session_id": "optional-session-id",
  "agent_id": "my-agent"
}
```

### `memory_recall`
```json
{
  "query": "natural language query",
  "top_k": 10,
  "filter": {
    "agent_id": "optional",
    "memory_type": "optional",
    "tags": ["optional"]
  }
}
```

### `memory_list`
```json
{
  "agent_id": "optional",
  "memory_type": "optional",
  "tags": "optional"
}
```

### `memory_delete`
```json
{
  "id": "memory-uuid"
}
```

## API Endpoints Reference

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/databases/:id/memories` | Store a new memory |
| GET | `/api/v1/databases/:id/memories` | List memories (with filters) |
| POST | `/api/v1/databases/:id/memories/recall` | Semantic recall search |
| DELETE | `/api/v1/databases/:id/memories/:id` | Delete a memory |

**Query Parameters (GET /memories):**
- `agent_id` — Filter by agent
- `memory_type` — Filter by type
- `tags` — Filter by tags (comma-separated)

## WorkBuddy Integration

For WorkBuddy / OpenClaw agents:

1. **Install skill**: `db9 onboard --agent workbuddy --scope both`
2. **Configure MCP**: Use generated `mcp-config.json`
3. **Start using**: Call `memory_store` when you learn something important

Example workflow:
```
User: "I want to use Rust for this new service"
WorkBuddy -> memory_store(content="New service will use Rust", type="decision", tags=["rust", "service"])

... later session ...

WorkBuddy -> memory_recall(query="what language for the new service")
-> Returns: "New service will use Rust" (type: decision)
```

## Best Practices

1. **Store early, store often** — Don't wait for "important" moments
2. **Use specific types** — Makes retrieval more accurate
3. **Tag consistently** — Create a tag taxonomy for your project
4. **Set importance wisely** — Critical facts get 0.9+, observations get 0.3-0.5
5. **Recall before acting** — Check existing knowledge before asking user
6. **Summarize periodically** — Convert low-importance memories to summaries
7. **Use sessions** — Group related work in sessions for better organization

## SQL Direct Access

Since data is in PostgreSQL, you can run analytics:

```sql
-- What topics does the user care about most?
SELECT unnest(tags) as tag, count(*) as cnt
FROM agent_memories
GROUP BY tag ORDER BY cnt DESC LIMIT 20;

-- How many memories per type?
SELECT memory_type, count(*), avg(importance_score) as avg_importance
FROM agent_memories GROUP BY memory_type;

-- Recent important decisions
SELECT content, created_at FROM agent_memories
WHERE memory_type = 'decision' AND importance_score > 0.7
ORDER BY created_at DESC LIMIT 10;
```

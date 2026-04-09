# Open-DB9 Quick Start Guide - Agent Memory Layer

> **Get up and running in 5 minutes with persistent, searchable agent memory.**

## Prerequisites

- Docker (20.10+)
- Docker Compose (2.0+)
- curl (for API testing)

## One-Line Start

```bash
curl -fsSL https://raw.githubusercontent.com/yihui504/open_db9/main/scripts/db9-quickstart.sh | bash
```

Or if you have the repo cloned:

```bash
bash scripts/db9-quickstart.sh
```

## Manual Step-by-Step

### 1. Clone & Start Services

```bash
git clone https://github.com/yihui504/open_db9.git
cd open_db9/deployments/docker
docker compose up -d --build
```

Wait ~30 seconds for services to start.

### 2. Verify Health

```bash
curl http://localhost:8080/health
# Expected: {"status":"healthy","service":"db9-api","version":"..."}
```

### 3. Get Authentication Token

```bash
TOKEN=$(python3 -c "
import jwt, time
secret = 'db9-local-jwt-hmac-key-2026-alpha-xkcd'
payload = {'sub':'demo-user','user_id':1,'username':'demo','exp':int(time.time())+3600,'iat':int(time.time())}
print(jwt.encode(payload, secret, algorithm='HS256'))
")
echo "Token: $TOKEN"
```

### 4. Create Memory Database

```bash
DB_ID=$(curl -s -X POST http://localhost:8080/api/v1/databases/ \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"name":"agent-memory"}' | python3 -c "import sys,json; print(json.load(sys.stdin).get('id','1'))")
echo "Database ID: $DB_ID"
```

### 5. Initialize Memory Tables

```bash
curl -s -X POST "http://localhost:8080/api/v1/databases/$DB_ID/sql" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"query":"CREATE TABLE IF NOT EXISTS agent_memories (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), agent_id VARCHAR(255) DEFAULT '\"'\"'\"default'\"'\"'\"', memory_type VARCHAR(50) DEFAULT '\"'\"'\"fact'\"'\"'\"', content TEXT NOT NULL, tags TEXT[] DEFAULT '\"'\"'\"{}'\"'\"'\"', importance_score FLOAT DEFAULT 0.5, created_at TIMESTAMPTZ DEFAULT NOW());"}'
```

### 6. Store Your First Memory

```bash
curl -s -X POST "http://localhost:8080/api/v1/databases/$DB_ID/memories" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"content":"Hello from my AI agent! This is my first persistent memory.","type":"observation","tags":["first","test"]}'
```

Expected response:
```json
{"success":true,"data":{"id":"<uuid>","agent_id":"default","memory_type":"observation"}}
```

### 7. Recall Your Memory

```bash
curl -s -X POST "http://localhost:8080/api/v1/databases/$DB_ID/memories/recall" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"query":"my first memory","top_k":5}'
```

You should get back your stored memory!

## Install for Your Agent

```bash
# For WorkBuddy / OpenClaw
db9 onboard --agent workbuddy --scope both

# For Claude Code
db9 onboard --agent claude

# For Cursor IDE
db9 onboard --agent cursor --scope project
```

## Generate MCP Configuration

```bash
# The quickstart script generates this automatically
# Or create manually:

cat > mcp-config.json << 'EOF'
{
  "mcpServers": {
    "open-db9": {
      "command": "./build/mcp-server",
      "args": [],
      "env": {
        "DB9_API_URL": "http://localhost:8080",
        "DB9_API_TOKEN": "<YOUR_TOKEN>",
        "DB9_DATABASE_ID": "<YOUR_DB_ID>"
      }
    }
  }
}
EOF
```

Copy this config to your Agent's settings directory.

## What's Next?

1. Read [memory-skill.md](./memory-skill.md) for full documentation
2. Try the demo: `bash demo/run-demo.sh`
3. Explore SQL access to your memories
4. Configure RAG for document-based memory

## Troubleshooting

| Problem | Solution |
|---------|----------|
| Docker not found | Install Docker Desktop |
| Port 8080 in use | Edit `docker-compose.yml` port mapping |
| Invalid token | Regenerate with the python command above |
| Table not found | Run step 5 to initialize tables |
| Connection refused | Check `docker compose ps` |

## Architecture Overview

```
Your Agent (Claude/WorkBuddy/Cursor/etc.)
         |
         | MCP Protocol or REST API
         v
   +------+------+
   | DB9 Server  | :8080
   +------+------+
         |
         | pgx connection pool
         v
   +------+------+
   | PostgreSQL  | :5432
   | + pgvector  |
   +-------------+
     - agent_memories table
     - memory_embeddings table
     - Full SQL access
```

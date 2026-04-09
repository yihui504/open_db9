#!/bin/bash
set -e
export LANG=en_US.UTF-8
export LC_ALL=en_US.UTF-8

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_ok() { echo -e "${GREEN}[OK]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_err() { echo -e "${RED}[ERROR]${NC} $1"; }

echo ""
echo "╔══════════════════════════════════════════════════════════╗"
echo "║     Open-DB9 Quick Start - Agent Memory Layer Setup        ║"
echo "║   Better than memory.md: Structured + Searchable + SQL      ║"
echo "╚══════════════════════════════════════════════════════════╝"
echo ""

API="http://localhost:8080"
RAG="http://localhost:8001"
JWT_SECRET="db9-local-jwt-hmac-key-2026-alpha-xkcd"
PROJECT_DIR="$(cd "$(dirname "$0")/.." && pwd)"

if [ ! -f "$PROJECT_DIR/deployments/docker/docker-compose.yml" ]; then
    log_info "Project not found locally, cloning..."
    git clone https://github.com/yihui504/open_db9.git /tmp/open-db9 2>/dev/null || {
        log_err "Failed to clone. Install git or download manually."
        exit 1
    }
    PROJECT_DIR="/tmp/open-db9"
fi

log_info "Step 1/7: Checking Docker environment..."
if ! command -v docker &>/dev/null; then
    log_err "Docker is not installed. Please install Docker first:"
    echo "  https://docs.docker.com/get-docker/"
    exit 1
fi

if ! command -v docker compose &>/dev/null && ! docker-compose version &>/dev/null 2>&1; then
    log_err "Docker Compose is not installed. Please install Docker Compose."
    exit 1
fi
log_ok "Docker environment OK"

log_info "Step 2/7: Starting services with Docker Compose..."
cd "$PROJECT_DIR/deployments/docker"
docker compose up -d --build 2>&1 | tail -5 || {
    log_err "Failed to start services"
    exit 1
}
log_ok "Docker Compose started"

log_info "Step 3/7: Waiting for services to be healthy..."
MAX_WAIT=120
WAITED=0
while [ $WAITED -lt $MAX_WAIT ]; do
    HEALTH=$(curl -sf http://localhost:8080/health 2>/dev/null || echo "")
    if [ -n "$HEALTH" ] && echo "$HEALTH" | grep -q "healthy"; then
        break
    fi
    sleep 3
    WAITED=$((WAITED + 3))
    printf "\r${BLUE}[WAIT]${NC} Waiting for health check... %ds/%ds" $WAITED $MAX_WAIT
done
echo ""
if [ $WAITED -ge $MAX_WAIT ]; then
    log_warn "Health check timeout after ${MAX_WAIT}s, continuing anyway..."
else
    log_ok "All services healthy (${WAITED}s)"
fi

log_info "Step 4/7: Initializing Memory tables..."
TOKEN=$(python3 -c "
import jwt, time
secret = '$JWT_SECRET'
payload = {'sub':'demo-user','user_id':1,'username':'demo','exp':int(time.time())+3600,'iat':int(time.time())}
print(jwt.encode(payload, secret, algorithm='HS256'))
" 2>/dev/null)

if [ -z "$TOKEN" ] || [ "${TOKEN:0:5}" = "ERROR" ]; then
    TOKEN="demo-token-fallback"
    log_warn "Using fallback token (some features may not work)"
fi

RESP=$(curl -s -X POST "$API/api/v1/databases/" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"name":"agent-memory","engine":"postgresql"}' 2>/dev/null)
DB_ID=$(echo "$RESP" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('id',d.get('database_id','1')))" 2>/dev/null || echo "1")

curl -sf -X POST "$API/api/v1/databases/$DB_ID/sql" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"query":"CREATE TABLE IF NOT EXISTS agent_memories (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), agent_id VARCHAR(255) NOT NULL DEFAULT '\"'\"'\"default'\"'\"'\"', session_id VARCHAR(255), memory_type VARCHAR(50) NOT NULL DEFAULT '\"'\"'\"fact'\"'\"'\"', content TEXT NOT NULL, summary TEXT, tags TEXT[] DEFAULT '\"'\"'\"{}'\"'\"'\"', importance_score FLOAT DEFAULT 0.5, access_count INTEGER DEFAULT 0, last_accessed_at TIMESTAMPTZ, created_at TIMESTAMPTZ DEFAULT NOW(), updated_at TIMESTAMPTZ DEFAULT NOW());"}' > /dev/null 2>&1 && log_ok "Memory tables initialized" || log_warn "Tables may already exist"

log_info "Step 5/7: Generating MCP configuration..."
MCP_CONFIG='{
  "mcpServers": {
    "open-db9": {
      "command": "./build/mcp-server",
      "args": [],
      "env": {
        "DB9_API_URL": "http://localhost:8080",
        "DB9_API_TOKEN": "'"$TOKEN"'",
        "DB9_RAG_URL": "http://localhost:8001",
        "DB9_DATABASE_ID": "'"$DB_ID"'"
      }
    }
  }
}'
echo "$MCP_CONFIG" > mcp-config.json
log_ok "MCP config saved to $(pwd)/mcp-config.json"

log_info "Step 6/7: Running verification tests..."
TEST_PASSED=0
TEST_TOTAL=4

STORE_RESP=$(curl -s -X POST "$API/api/v1/databases/$DB_ID/memories" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"content":"QuickStart test memory","type":"observation","tags":["test"]}' 2>/dev/null)
if echo "$STORE_RESP" | grep -q '"success":true'; then
    log_ok "Memory store: PASS"
    TEST_PASSED=$((TEST_PASSED + 1))
else
    log_err "Memory store: FAIL"
fi
TEST_TOTAL=$((TEST_TOTAL + 1))

LIST_RESP=$(curl -s "$API/api/v1/databases/$DB_ID/memories?agent_id=default" \
  -H "Authorization: Bearer $TOKEN" 2>/dev/null)
if echo "$LIST_RESP" | grep -q '"count"'; then
    log_ok "Memory list: PASS"
    TEST_PASSED=$((TEST_PASSED + 1))
else
    log_err "Memory list: FAIL"
fi
TEST_TOTAL=$((TEST_TOTAL + 1))

RECALL_RESP=$(curl -s -X POST "$API/api/v1/databases/$DB_ID/memories/recall" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"query":"quickstart test"}' 2>/dev/null)
if echo "$RECALL_RESP" | grep -q '"success":true'; then
    log_ok "Memory recall: PASS"
    TEST_PASSED=$((TEST_PASSED + 1))
else
    log_err "Memory recall: FAIL"
fi
TEST_TOTAL=$((TEST_TOTAL + 1))

SQL_RESP=$(curl -s -X POST "$API/api/v1/databases/$DB_ID/sql" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"query":"SELECT count(*) as cnt FROM agent_memories"}' 2>/dev/null)
if echo "$SQL_RESP" | grep -q "cnt"; then
    log_ok "SQL query: PASS"
    TEST_PASSED=$((TEST_PASSED + 1))
else
    log_err "SQL query: FAIL"
fi
TEST_TOTAL=$((TEST_TOTAL + 1))

echo ""
log_info "Step 7/7: Summary"
echo ""
echo "  ========================================"
echo "   Open-DB9 Quick Start Complete!"
echo "  ========================================"
echo ""
echo "  Connection Info:"
echo "    API URL:       $API"
echo "    Database ID:   $DB_ID"
echo "    Token:         ${TOKEN:0:20}..."
echo "    RAG Service:   $RAG"
echo ""
echo "  Test Results:   $TEST_PASSED/$TEST_TOTAL passed"
echo ""
echo "  Next Steps:"
echo "    1. Copy mcp-config.json to your Agent config directory"
echo "    2. Run: db9 onboard --agent workbuddy"
echo "    3. Start using memory_store / memory_recall tools"
echo "    4. Read skills/memory-skill.md for full documentation"
echo ""
echo "  Services Status:"
docker ps --format "    {{.Names}} ({{.Status}})" 2>/dev/null || log_warn "Cannot list containers"
echo ""

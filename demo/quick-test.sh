#!/bin/bash
API="http://localhost:8080"
JWT_SECRET="${JWT_SECRET:-}"
if [ -z "$JWT_SECRET" ]; then
    echo "ERROR: JWT_SECRET environment variable is not set. Please set it before running."
    echo "  Example: export JWT_SECRET=<your-secret-key>"
    exit 1
fi

TOKEN=$(python3 -c "
import jwt, time
secret = '$JWT_SECRET'
payload = {
    'sub': 'demo-user',
    'user_id': 1,
    'username': 'demo',
    'exp': int(time.time()) + 3600,
    'iat': int(time.time())
}
print(jwt.encode(payload, secret, algorithm='HS256'))
")

echo "=== Token: ${TOKEN:0:60}... ==="
echo ""

echo "=== Test 1: Health Check ==="
curl -s "$API/health" | python3 -m json.tool 2>/dev/null || curl -s "$API/health"
echo ""

echo "=== Test 2: Create Database ==="
RESP=$(curl -s -X POST "$API/api/v1/databases/" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"name":"test-db-$(date +%s)","engine":"postgresql"}')
echo "$RESP" | python3 -m json.tool 2>/dev/null || echo "$RESP"
DB_ID=$(echo "$RESP" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('id', d.get('database_id', 'unknown')))" 2>/dev/null)
echo ">>> DB_ID: $DB_ID"
echo ""

echo "=== Test 3: Execute SQL (DB ID=1) ==="
curl -s -X POST "$API/api/v1/databases/1/sql" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"query":"SELECT 1 AS test, current_timestamp AS now;"}' | python3 -m json.tool 2>/dev/null || curl -s -X POST "$API/api/v1/databases/1/sql" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"query":"SELECT 1 AS test, current_timestamp AS now;"}'
echo ""

echo "=== Test 4: Schema Tables (DB ID=1) ==="
curl -s "$API/api/v1/databases/1/schema/tables?schema=public" \
  -H "Authorization: Bearer $TOKEN" | python3 -m json.tool 2>/dev/null || curl -s "$API/api/v1/databases/1/schema/tables?schema=public" \
  -H "Authorization: Bearer $TOKEN"
echo ""

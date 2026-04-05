#!/bin/bash
API="http://localhost:8080"
JWT_SECRET="db9-local-jwt-hmac-key-2026-alpha-xkcd"

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

echo "Token: ${TOKEN:0:50}..."

echo ""
echo "=== Testing SQL endpoint ==="
RESPONSE=$(curl -s -w "\nHTTP_CODE:%{http_code}" -X POST "$API/api/v1/databases/1/sql" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"query":"SELECT 1 AS test"}')

echo "$RESPONSE"

echo ""
echo "=== Testing with DB ID as string ==="
RESPONSE2=$(curl -s -w "\nHTTP_CODE:%{http_code}" -X POST "$API/api/v1/databases/1/sql" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"query":"SELECT current_timestamp"}')

echo "$RESPONSE2"

#!/bin/bash
API="http://localhost:8080"
JWT_SECRET="db9-local-jwt-hmac-key-2026-alpha-xkcd"
TOKEN=$(python3 -c "import jwt,time; print(jwt.encode({'sub':'demo-user','user_id':1,'username':'demo','exp':int(time.time())+3600,'iat':int(time.time())},'$JWT_SECRET',algorithm='HS256'))")

echo "=== Test SQL ==="
curl -s -X POST "$API/api/v1/databases/1/sql" -H "Content-Type: application/json" -H "Authorization: Bearer $TOKEN" -d '{"query":"SELECT 1 AS test"}'
echo ""

echo "=== Test Metrics ==="
curl -s "$API/api/v1/databases/1/metrics/stats" -H "Authorization: Bearer $TOKEN"
echo ""

echo "=== Test Anonymous ==="
curl -s -X POST "$API/api/v1/auth/anonymous" -H "Content-Type: application/json" -d '{"session_id":"test-session-123"}'
echo ""

echo "=== Test File Upload ==="
echo "test-data" > /tmp/test-fix.txt
curl -s -X POST "$API/api/v1/databases/1/files" -H "Authorization: Bearer $TOKEN" -F "file=@/tmp/test-fix.txt" -F "path=/test.txt"
echo ""

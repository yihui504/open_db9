# Open-DB9 API 文档

## 基础信息

**Base URL**: `http://localhost:8080`
**API Version**: `v1`
**前缀**: `/api/v1`

---

## 🔐 认证

### 获取 Token

所有 API 端点（除公共端点外）都需要 JWT Bearer Token 认证。

#### 登录

```http
POST /api/v1/users/login
Content-Type: application/json

{
  "username": "admin",
  "password": "your-password"
}
```

**响应**:
```json
{
  "success": true,
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "user": {
      "id": 1,
      "username": "admin",
      "email": "admin@example.com",
      "role": "admin"
    }
  }
}
```

#### 使用 Token

```http
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

---

## 📊 响应格式

### 成功响应

```json
{
  "success": true,
  "data": { ... }
}
```

### 成功响应（带消息）

```json
{
  "success": true,
  "message": "操作成功",
  "data": { ... }
}
```

### 错误响应

```json
{
  "success": false,
  "error": "错误描述信息"
}
```

---

## 📋 HTTP 状态码

| 代码 | 描述 |
|------|------|
| 200 | OK - 请求成功 |
| 201 | Created - 资源创建成功 |
| 204 | No Content - 成功但无返回内容 |
| 400 | Bad Request - 无效输入 |
| 401 | Unauthorized - 未认证或认证失败 |
| 403 | Forbidden - 无权限访问 |
| 404 | Not Found - 资源不存在 |
| 405 | Method Not Allowed - 不支持的方法 |
| 413 | Payload Too Large - 请求体过大 |
| 429 | Too Many Requests - 超出速率限制 |
| 500 | Internal Server Error - 服务器错误 |
| 503 | Service Unavailable - 服务不可用 |

---

## 🚀 速率限制

- **限制**: 每分钟 100 请求（每 Token）
- **窗口**: 60 秒滚动窗口
- **响应头**:
  ```
  X-RateLimit-Limit: 100
  X-RateLimit-Remaining: 95
  X-RateLimit-Reset: 1617200000
  ```

超出限制时:
```json
{
  "success": false,
  "error": "Rate limit exceeded. Please try again later."
}
```

---

## 🛣️ 公共端点

### 根端点

#### GET /

返回 API 信息。

**响应**:
```json
{
  "success": true,
  "data": {
    "message": "DB9 API Server",
    "version": "1.0.0"
  }
}
```

---

### 健康检查

#### GET /health

检查服务健康状态。

**响应**:
```json
{
  "success": true,
  "data": {
    "status": "healthy",
    "filesystem": {
      "status": "healthy",
      "storage_path_accessible": true,
      "fs9_service_reachable": true,
      "basic_operations_work": true
    },
    "timestamp": "2026-03-31T14:00:00Z"
  }
}
```

**健康状态**:
- `healthy` - 所有系统正常
- `degraded` - 部分功能降级（如 FS9 服务不可达）
- `unhealthy` - 系统不健康（如存储路径不可访问）

---

### 版本信息

#### GET /version

获取 API 版本信息。

**响应**:
```json
{
  "success": true,
  "data": {
    "version": "1.0.0",
    "go_version": "go1.25.0"
  }
}
```

---

## 👥 用户管理端点

### 创建用户

#### POST /api/v1/users

创建新用户（需要管理员权限）。

**请求头**:
```
Content-Type: application/json
Authorization: Bearer <token>
```

**请求体**:
```json
{
  "username": "newuser",
  "email": "newuser@example.com",
  "password": "secure-password",
  "role": "user"
}
```

| 字段 | 类型 | 必需 | 描述 |
|------|------|------|------|
| username | string | ✅ | 用户名（唯一） |
| email | string | ✅ | 邮箱（唯一） |
| password | string | ✅ | 密码（将被哈希） |
| role | string | ❌ | 角色（默认: "user"） |

**响应** (201):
```json
{
  "success": true,
  "data": {
    "id": 2,
    "username": "newuser",
    "email": "newuser@example.com",
    "role": "user",
    "created_at": "2026-03-31T14:00:00Z",
    "updated_at": "2026-03-31T14:00:00Z"
  }
}
```

---

### 获取用户

#### GET /api/v1/users/:id

获取用户信息。

**响应**:
```json
{
  "success": true,
  "data": {
    "id": 1,
    "username": "admin",
    "email": "admin@example.com",
    "role": "admin",
    "created_at": "2026-03-31T10:00:00Z",
    "updated_at": "2026-03-31T10:00:00Z"
  }
}
```

---

### 列出用户

#### GET /api/v1/users

列出所有用户（支持分页）。

**查询参数**:
- `page` (integer) - 页码（默认: 1）
- `limit` (integer) - 每页数量（默认: 100，最大: 1000）

**请求**:
```
GET /api/v1/users?page=1&limit=50
```

**响应**:
```json
{
  "success": true,
  "data": {
    "users": [
      {
        "id": 1,
        "username": "admin",
        "email": "admin@example.com",
        "role": "admin",
        "created_at": "2026-03-31T10:00:00Z",
        "updated_at": "2026-03-31T10:00:00Z"
      }
    ],
    "total": 1,
    "page": 1,
    "limit": 50
  }
}
```

---

### 更新用户

#### PUT /api/v1/users/:id
#### PATCH /api/v1/users/:id

更新用户信息。

**请求体**:
```json
{
  "email": "newemail@example.com",
  "password": "new-password",
  "role": "admin"
}
```

**响应**:
```json
{
  "success": true,
  "data": {
    "id": 1,
    "username": "admin",
    "email": "newemail@example.com",
    "role": "admin",
    "created_at": "2026-03-31T10:00:00Z",
    "updated_at": "2026-03-31T14:30:00Z"
  }
}
```

---

### 删除用户

#### DELETE /api/v1/users/:id

删除用户（需要管理员权限）。

**响应**:
```json
{
  "success": true,
  "message": "User deleted successfully"
}
```

---

## 💾 数据库端点

### 执行 SQL

#### POST /api/v1/databases/:id/sql

执行 SQL 查询。

**路径参数**:
- `id` (integer) - 数据库 ID

**请求头**:
```
Content-Type: application/json
Authorization: Bearer <token>
```

**请求体**:
```json
{
  "query": "SELECT * FROM users LIMIT 10",
  "timeout_ms": 5000
}
```

| 字段 | 类型 | 必需 | 描述 |
|------|------|------|------|
| query | string | ✅ | SQL 查询 |
| timeout_ms | integer | ❌ | 超时毫秒（默认: 30000，范围: 1000-300000） |

**响应**:
```json
{
  "success": true,
  "data": {
    "columns": ["id", "username", "email", "role"],
    "rows": [
      {
        "id": 1,
        "username": "admin",
        "email": "admin@example.com",
        "role": "admin"
      }
    ],
    "row_count": 1
  }
}
```

**SQL 白名单**:
- `SELECT`
- `SHOW`
- `DESCRIBE`
- `EXPLAIN`

---

## 📸 快照端点

### 创建快照

#### POST /api/v1/databases/:id/snapshots

创建数据库快照（异步操作）。

**路径参数**:
- `id` (integer) - 数据库 ID

**请求体**:
```json
{
  "name": "backup-2026-03-31"
}
```

**响应** (202):
```json
{
  "success": true,
  "data": {
    "id": "snap-uuid",
    "database_id": "1",
    "name": "backup-2026-03-31",
    "size_bytes": 0,
    "storage_path": "/data/db9-snapshots/1/snap-uuid.sql.gz",
    "status": "creating",
    "created_at": "2026-03-31T14:00:00Z"
  }
}
```

---

### 列出快照

#### GET /api/v1/databases/:id/snapshots

列出所有快照。

**响应**:
```json
{
  "success": true,
  "data": {
    "snapshots": [
      {
        "id": "snap-uuid-1",
        "database_id": "1",
        "name": "backup-2026-03-31",
        "size_bytes": 104857600,
        "storage_path": "/data/db9-snapshots/1/snap-uuid-1.sql.gz",
        "status": "ready",
        "created_at": "2026-03-31T10:00:00Z"
      }
    ],
    "count": 1
  }
}
```

---

### 获取快照

#### GET /api/v1/databases/:id/snapshots/:sid

获取快照详情。

**路径参数**:
- `id` (integer) - 数据库 ID
- `sid` (string) - 快照 ID

**响应**:
```json
{
  "success": true,
  "data": {
    "id": "snap-uuid",
    "database_id": "1",
    "name": "backup-2026-03-31",
    "size_bytes": 104857600,
    "storage_path": "/data/db9-snapshots/1/snap-uuid.sql.gz",
    "status": "ready",
    "created_at": "2026-03-31T12:00:00Z",
    "expires_at": "2026-04-30T12:00:00Z"
  }
}
```

---

### 删除快照

#### DELETE /api/v1/databases/:id/snapshots/:sid

删除快照。

**路径参数**:
- `id` (integer) - 数据库 ID
- `sid` (string) - 快照 ID

**响应**:
```json
{
  "success": true,
  "message": "Snapshot deleted successfully"
}
```

---

### 恢复快照

#### POST /api/v1/databases/:id/snapshots/:sid/restore

从快照恢复数据库（异步操作）。

**路径参数**:
- `id` (integer) - 数据库 ID
- `sid` (string) - 快照 ID

**响应** (202):
```json
{
  "success": true,
  "message": "Snapshot restoration initiated",
  "data": {
    "snapshot_id": "snap-uuid",
    "status": "restoring"
  }
}
```

---

## 🌿 分支端点

### 创建分支

#### POST /api/v1/databases/:id/branches

创建数据库分支。

**路径参数**:
- `id` (integer) - 数据库 ID

**请求体**:
```json
{
  "name": "feature-x"
}
```

**响应** (201):
```json
{
  "success": true,
  "data": {
    "id": "branch-uuid",
    "name": "feature-x-timestamp",
    "parent_id": "parent-db-uuid",
    "snapshot_id": "snap-uuid",
    "created_at": "2026-03-31T14:00:00Z",
    "status": "active"
  }
}
```

---

### 列出分支

#### GET /api/v1/databases/:id/branches

列出所有分支。

**响应**:
```json
{
  "success": true,
  "data": {
    "branches": [
      {
        "id": "branch-uuid-1",
        "name": "feature-x",
        "parent_id": "parent-db-uuid",
        "snapshot_id": "snap-uuid-1",
        "created_at": "2026-03-31T10:00:00Z",
        "status": "active"
      }
    ],
    "count": 1
  }
}
```

---

### 删除分支

#### DELETE /api/v1/databases/:id/branches/:bid

删除分支。

**路径参数**:
- `id` (integer) - 父数据库 ID
- `bid` (string) - 分支 ID

**响应**:
```json
{
  "success": true,
  "message": "Branch deleted successfully"
}
```

---

## 📁 文件管理端点

### 上传文件

#### POST /api/v1/databases/:id/files

上传文件到数据库。

**路径参数**:
- `id` (integer) - 数据库 ID

**请求**:
- Content-Type: `multipart/form-data`
- 最大文件大小: 100MB

**表单字段**:
- `file` (file) - 要上传的文件（必需）
- `path` (string) - 文件路径（可选，默认: `/filename`）

**cURL 示例**:
```bash
curl -X POST http://localhost:8080/api/v1/databases/1/files \
  -H "Authorization: Bearer <token>" \
  -F "file=@/path/to/file.txt" \
  -F "path=/documents"
```

**响应** (201):
```json
{
  "success": true,
  "data": {
    "id": 1,
    "database_id": 1,
    "path": "/documents/file.txt",
    "name": "file.txt",
    "size": 1024,
    "content_type": "text/plain",
    "checksum": "abc123...",
    "storage_key": "1/documents/file.txt",
    "created_at": "2026-03-31T14:00:00Z"
  }
}
```

---

### 列出文件

#### GET /api/v1/databases/:id/files

列出文件。

**路径参数**:
- `id` (integer) - 数据库 ID

**查询参数**:
- `path` (string) - 路径前缀过滤（可选）
- `limit` (integer) - 最大结果数（默认: 100，最大: 1000）

**请求**:
```
GET /api/v1/databases/1/files?path=/documents&limit=50
```

**响应**:
```json
{
  "success": true,
  "data": {
    "files": [
      {
        "id": 1,
        "database_id": 1,
        "path": "/documents/file1.txt",
        "name": "file1.txt",
        "size": 1024,
        "content_type": "text/plain",
        "checksum": "abc123...",
        "created_at": "2026-03-31T12:00:00Z"
      }
    ],
    "total": 1
  }
}
```

---

### 下载文件

#### GET /api/v1/databases/:id/files/:path

下载文件。

**路径参数**:
- `id` (integer) - 数据库 ID
- `path` (string) - 文件路径

**响应**:
- Content-Type: 文件的内容类型
- Content-Disposition: `attachment; filename="filename"`
- Content-Length: 文件大小
- Body: 文件二进制内容

**cURL 示例**:
```bash
curl -X GET http://localhost:8080/api/v1/databases/1/files/documents/file1.txt \
  -H "Authorization: Bearer <token>" \
  -o downloaded-file.txt
```

---

### 复制文件

#### POST /api/v1/databases/:id/files/copy

复制文件（服务端操作）。

**路径参数**:
- `id` (integer) - 数据库 ID

**请求体**:
```json
{
  "src_path": "/documents/file1.txt",
  "dst_path": "/documents/file1-backup.txt"
}
```

**响应**:
```json
{
  "success": true,
  "data": {
    "id": 2,
    "database_id": 1,
    "path": "/documents/file1-backup.txt",
    "name": "file1-backup.txt",
    "size": 1024,
    "content_type": "text/plain",
    "checksum": "abc123...",
    "created_at": "2026-03-31T14:00:00Z"
  },
  "message": "File copied successfully"
}
```

---

### 删除文件

#### DELETE /api/v1/databases/:id/files/:path

删除文件。

**路径参数**:
- `id` (integer) - 数据库 ID
- `path` (string) - 文件路径

**响应**:
```json
{
  "success": true,
  "message": "File deleted successfully"
}
```

---

## 🔧 扩展端点

### 列出扩展

#### GET /api/v1/databases/:id/extensions

列出可用的数据库扩展。

**路径参数**:
- `id` (integer) - 数据库 ID

**响应**:
```json
{
  "success": true,
  "data": {
    "extensions": [
      {
        "name": "pgvector",
        "version": "0.5.0",
        "enabled": true
      },
      {
        "name": "pg_cron",
        "version": "1.6.0",
        "enabled": false
      }
    ]
  }
}
```

---

### 启用扩展

#### POST /api/v1/databases/:id/extensions/enable

启用数据库扩展。

**路径参数**:
- `id` (integer) - 数据库 ID

**请求体**:
```json
{
  "extension": "pgvector"
}
```

**响应**:
```json
{
  "success": true,
  "message": "Extension 'pgvector' enabled successfully"
}
```

---

### 禁用扩展

#### POST /api/v1/databases/:id/extensions/disable

禁用数据库扩展。

**请求体**:
```json
{
  "extension": "pgvector"
}
```

**响应**:
```json
{
  "success": true,
  "message": "Extension 'pgvector' disabled successfully"
}
```

---

## 🧪 cURL 示例

### 完整工作流

```bash
# 1. 登录获取 Token
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/users/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"secret"}' \
  | jq -r '.data.token')

# 2. 执行 SQL 查询
curl -X POST http://localhost:8080/api/v1/databases/1/sql \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"query":"SELECT version()"}'

# 3. 创建快照
curl -X POST http://localhost:8080/api/v1/databases/1/snapshots \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"backup-2026-03-31"}'

# 4. 上传文件
curl -X POST http://localhost:8080/api/v1/databases/1/files \
  -H "Authorization: Bearer $TOKEN" \
  -F "file=@./data.csv" \
  -F "path=/imports"

# 5. 创建分支
curl -X POST http://localhost:8080/api/v1/databases/1/branches \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"experiment-1"}'

# 6. 复制文件
curl -X POST http://localhost:8080/api/v1/databases/1/files/copy \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"src_path":"/data.csv","dst_path":"/data-backup.csv"}'
```

---

## 📚 SDK 和客户端库

### Go

```go
package main

import (
    "bytes"
    "encoding/json"
    "net/http"
)

type Client struct {
    BaseURL    string
    Token      string
    HTTPClient *http.Client
}

func (c *Client) ExecuteSQL(dbID int, query string) (*SQLResponse, error) {
    body := map[string]interface{}{
        "query": query,
    }
    jsonBody, _ := json.Marshal(body)

    req, _ := http.NewRequest("POST",
        c.BaseURL + fmt.Sprintf("/api/v1/databases/%d/sql", dbID),
        bytes.NewBuffer(jsonBody))
    req.Header.Set("Authorization", "Bearer " + c.Token)
    req.Header.Set("Content-Type", "application/json")

    resp, err := c.HTTPClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var result struct {
        Success bool        `json:"success"`
        Data    SQLResponse `json:"data"`
    }
    json.NewDecoder(resp.Body).Decode(&result)

    return &result.Data, nil
}
```

---

## 🔄 更新日志

| 版本 | 日期 | 更新内容 |
|------|------|----------|
| 1.0.0 | 2026-03-31 | 初始版本，包含所有核心功能 |

---

**文档版本**: 1.0
**最后更新**: 2026-03-31

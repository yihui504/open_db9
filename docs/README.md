# Open-DB9 文档

欢迎使用 Open-DB9 文档！本文档提供了完整的使用指南和 API 参考。

---

## 📑 目录

1. [快速开始](#快速开始)
2. [安装指南](#安装指南)
3. [配置说明](#配置说明)
4. [CLI 参考](#cli-参考)
5. [API 参考](#api-参考)
6. [安全指南](#安全指南)
7. [部署指南](#部署指南)
8. [故障排除](#故障排除)

---

## 快速开始

### 5 分钟快速上手

```bash
# 1. 构建项目
make build

# 2. 启动文件服务
./build/fs9-service &

# 3. 启动 API 服务器
./build/server &

# 4. 执行 SQL 查询
./build/db9 db sql 1 "SELECT version()"

# 5. 创建快照
./build/db9 db snapshot create 1 --name "initial"
```

---

## 安装指南

### 系统要求

- **Go**: 1.25.0 或更高
- **PostgreSQL**: 12+ (用于控制平面数据库)
- **操作系统**: Linux、macOS、Windows

### 从源码安装

```bash
# 克隆仓库
git clone https://github.com/open-db9/db9.git
cd db9

# 安装依赖
go mod download

# 构建所有组件
make build

# (可选) 安装到 $GOPATH/bin
make install
```

### Docker 安装

```bash
# 使用 Docker Compose
docker-compose up -d

# 或单独构建镜像
docker build -t open-db9/server -f deployments/docker/Dockerfile.server .
docker build -t open-db9/cli -f deployments/docker/Dockerfile.cli .
```

---

## 配置说明

### 服务器配置

服务器支持通过环境变量或配置文件进行配置。

#### 环境变量

```bash
export DB9_HOST="0.0.0.0"
export DB9_PORT="8080"
export DB9_DB_HOST="localhost"
export DB9_DB_PORT="5432"
export DB9_DB_USER="postgres"
export DB9_DB_PASSWORD="secret"
export DB9_DB_NAME="db9"
export DB9_JWT_SECRET="your-very-secret-key-at-least-32-chars-long"
export DB9_FS9_URL="http://localhost:9090"
```

#### 配置文件

创建 `~/.db9/config.yaml`:

```yaml
server:
  host: "0.0.0.0"
  port: 8080
  timeout: 30
  read_timeout: 30
  write_timeout: 30
  idle_timeout: 120

database:
  host: "localhost"
  port: 5432
  user: "postgres"
  password: "${DB_PASSWORD}"
  database: "db9"
  ssl_mode: "disable"
  pool:
    max_connections: 25
    min_connections: 5
    max_conn_lifetime: 0
    max_conn_idle_time: 0
    health_check_period: "1m"

jwt:
  secret: "${JWT_SECRET}"
  expires_in: 3600
  issuer: "open-db9"

rate_limit:
  enabled: true
  max_requests: 100
  window: 60
  cleanup_interval: 300

cors:
  enabled: true
  allowed_origins:
    - "*"
  allowed_methods:
    - GET
    - POST
    - PUT
    - DELETE
    - PATCH
    - OPTIONS
  allowed_headers:
    - Content-Type
    - Authorization
    - X-Requested-With
  exposed_headers:
    - X-Total-Count
    - X-Page-Limit
  allow_credentials: false
  max_age: 86400

logging:
  level: "info"
  format: "json"
```

### CLI 配置

CLI 配置存储在 `~/.db9/config.yaml`:

```yaml
api:
  base_url: "http://localhost:8080"
  timeout: 30

credentials:
  # 凭证通过 `db9 auth login` 安全存储
  # 不在此文件中明文存储

output:
  format: "table"  # table, json, csv
  pager: "less"
```

---

## CLI 参考

### 全局选项

```bash
db9 [全局选项] 命令 [命令选项]

全局选项:
  --config string      配置文件路径 (default: ~/.db9/config.yaml)
  -v, --verbose int    详细程度 (0-3) (default 0)
  -h, --help          显示帮助信息
  --version           显示版本信息
```

### 数据库命令

```bash
# SQL 查询
db9 db sql <database-id> "<query>"

# 交互式 SQL
db9 db sql <database-id> --interactive

# 快照管理
db9 db snapshot create <id> --name <name>
db9 db snapshot list <id>
db9 db snapshot get <id> <snapshot-id>
db9 db snapshot delete <id> <snapshot-id> [--yes]
db9 db snapshot restore <id> <snapshot-id>

# 分支管理
db9 db branch create <id> --name <name>
db9 db branch list <id>
db9 db branch delete <id> --branch <branch-id> [--yes]

# 扩展管理
db9 db extension list <id>
db9 db extension enable <id> <extension-name>
db9 db extension disable <id> <extension-name>

# 类型生成
db9 gen types <output-dir> --language (typescript|python)
```

### 文件系统命令

```bash
# 文件操作
db9 fs upload <database-id> <local-file> <remote-path>
db9 fs download <database-id> <remote-path> <local-file>
db9 fs list <database-id> [path]
db9 fs rm <database-id> <path>
db9 fs cp <database-id> <src-path> <dst-path>
```

### Agent 命令

```bash
# 安装 AI Agent 技能
db9 onboard --agent claude           # Claude Code
db9 onboard --agent cursor          # Cursor IDE
db9 onboard --agent cline            # VSCode Cline
db9 onboard --agent codex           # OpenAI Codex
db9 onboard --agent opencode         # OpenCode

# 管理已安装技能
db9 onboard --list                   # 列出所有
db9 onboard --uninstall claude       # 卸载
```

### 认证命令

```bash
# 登录服务
db9 auth login <service> --username <user> --password <pass>
db9 auth login <service> --token <api-token>

# 查看状态
db9 auth status

# 登出
db9 auth logout [service]
```

---

## API 参考

完整的 API 文档请参考 [API.md](API.md)。

### 主要端点

| 端点 | 方法 | 描述 |
|------|------|------|
| `/health` | GET | 健康检查 |
| `/api/v1/users/login` | POST | 用户登录 |
| `/api/v1/databases/:id/sql` | POST | 执行 SQL |
| `/api/v1/databases/:id/snapshots` | GET/POST | 列出/创建快照 |
| `/api/v1/databases/:id/branches` | GET/POST | 列出/创建分支 |
| `/api/v1/databases/:id/files` | GET/POST | 列出/上传文件 |

### Schema 内省 API (Phase 7 修复)

提供数据库结构的完整内省能力，返回表、列、索引、约束等详细信息。

| 端点 | 方法 | 描述 | 参数 |
|------|------|------|------|
| `/api/v1/databases/:id/schema` | GET | 列出所有 schema | - |
| `/api/v1/databases/:id/schema/tables` | GET | 列出指定 schema 的表 | `schema` (默认 public) |
| `/api/v1/databases/:id/schema/table` | GET | 获取表完整详情 | `schema`, `table` (必填) |

**返回数据包含**: `columns[]`, `indexes[]`, `constraints[]`, `triggers[]`, `row_count_estimate`, `size_bytes`

### Metrics 可观测性 API (Phase 7 新增)

提供数据库运行时的统计和监控数据。

| 端点 | 方法 | 描述 | 参数 |
|------|------|------|------|
| `/api/v1/databases/:id/metrics/stats` | GET | 数据库统计概览 | - |
| `/api/v1/databases/:id/metrics/slow` | GET | 慢查询列表 | `threshold`(ms), `limit`(1-100) |
| `/api/v1/databases/:id/metrics/queries` | GET | Top N 查询统计 | `limit`, `sort`(total_time/calls/rows) |

**stats 返回字段**: `total_connections`, `active_queries`, `total_tables`, `total_size_bytes`, `uptime_seconds`, `cache_hit_ratio`

### RAG API

基于检索增强生成的文档智能查询子系统。

| 端点 | 方法 | 描述 |
|------|------|------|
| `/api/v1/databases/:id/rag/documents` | POST/GET/DELETE/PATCH | 文档管理 |
| `/api/v1/databases/:id/rag/query` | POST | 自然语言查询 |

### 匿名认证 API (Phase 7 新增)

支持匿名用户注册和后续 SSO 认领账户的完整流程。

| 端点 | 方法 | 描述 | 认证 |
|------|------|------|------|
| `/api/v1/auth/anonymous` | POST | 匿名注册 | 公开 |
| `/api/v1/auth/claim` | POST | SSO 认领账户 | 需认证 |

**匿名注册请求/响应示例**:

```json
// Request
POST /api/v1/auth/anonymous
{"session_id": "unique-session-id"}

// Response 201
{
  "success": true,
  "data": {
    "token": "eyJ...",
    "tenant_id": "uuid",
    "capabilities": {"max_databases": 5},
    "expires_at": "2026-01-03T00:00:00Z"
  }
}
```

**认领请求/响应示例**:

```json
// Request (需 Bearer Token)
POST /api/v1/auth/claim
{"session_id": "unique-session-id"}

// Response 200
{
  "success": true,
  "data": {
    "token": "eyJ...(new-full-permissions)",
    "message": "Account claimed successfully"
  }
}
```

### HTTP 扩展 API (Phase 7 新增)

允许通过 API 安全地发起 HTTP 请求，适用于数据管道和 Webhook 场景。

| 端点 | 方法 | 描述 | 安全限制 |
|------|------|------|----------|
| `/api/v1/databases/:id/http/get` | POST | 发起 HTTP GET 请求 | SSRF 防护, HTTPS-only |
| `/api/v1/databases/:id/http/post` | POST | 发起 HTTP POST 请求 | 同上 |

**HTTP GET 请求示例**:

```json
// Request
POST /api/v1/databases/1/http/get
{"url": "https://api.example.com/data", "headers": {"Authorization": "Bearer xxx"}}

// Response 200
{
  "status_code": 200,
  "body": "{...response...}",
  "headers": {"content-type": "application/json"}
}
```

> **安全限制说明**
>
> - 仅允许 **HTTPS** 协议
> - **禁止私有 IP 地址**: `10.x` / `172.16-31.x` / `192.168.x` / `127.x`
> - DNS Rebinding 防护
> - 响应体最大 **1MB**, 请求体最大 **256KB**
> - **5 秒**超时
> - 速率限制: **10 请求/分钟/租户**

---

## 安全指南

完整的安全指南请参考 [SECURITY.md](SECURITY.md)。

### 安全特性

- ✅ JWT 认证保护所有 API 端点
- ✅ 速率限制防止滥用
- ✅ CORS 策略控制跨域访问
- ✅ 凭证 AES-256-GCM 加密存储
- ✅ SQL 注入防护（参数化查询）
- ✅ 输入验证和大小限制
- ✅ 敏感信息脱敏

### 最佳实践

1. **使用强密码**: JWT 密钥至少 32 字符
2. **启用 HTTPS**: 生产环境必须使用 TLS
3. **限制 CORS**: 仅允许可信的源
4. **定期轮换密钥**: 定期更新 JWT 密钥
5. **监控日志**: 关注异常访问模式

---

## 部署指南

完整的部署指南请参考 [DEPLOYMENT.md](DEPLOYMENT.md)。

### 生产环境检查清单

- [ ] 使用 HTTPS/TLS
- [ ] 配置防火墙规则
- [ ] 设置速率限制
- [ ] 配置 CORS 策略
- [ ] 启用日志记录
- [ ] 设置监控告警
- [ ] 定期备份
- [ ] 安全审计

### Docker 部署

```bash
# 使用 Docker Compose
docker-compose -f deployments/docker/docker-compose.yml up -d

# 检查服务状态
docker-compose ps
docker-compose logs -f
```

### Kubernetes 部署

```bash
# 应用 Kubernetes 配置
kubectl apply -f deployments/k8s/

# 检查部署状态
kubectl get pods -l app=open-db9
kubectl get services
```

---

## 故障排除

### 常见问题

#### 1. 数据库连接失败

```
Error: connection refused
```

**解决方案**:
- 检查数据库是否运行
- 验证连接参数（主机、端口、用户名、密码）
- 确认网络连接和防火墙设置

#### 2. JWT 认证失败

```
Error: invalid token
```

**解决方案**:
- 检查 JWT 密钥配置
- 确认 Token 未过期
- 验证 Authorization 头格式

#### 3. 文件上传失败

```
Error: file too large
```

**解决方案**:
- 文件大小限制为 100MB
- 压缩大文件后再上传
- 或调整配置中的大小限制

#### 4. 速率限制

```
Error: rate limit exceeded
```

**解决方案**:
- 等待 1 分钟后重试
- 或调整配置中的速率限制

### 调试模式

```bash
# 启用详细日志
DB9_LOG_LEVEL=debug ./build/server

# CLI 详细模式
db9 -vvv db sql 1 "SELECT 1"
```

### 日志位置

- 服务器日志: `stdout/stderr`（可配置文件输出）
- CLI 日志: `~/.db9/logs/`

---

## MCP Server 使用说明

MCP Server 让 AI Agent 通过 Model Context Protocol 直接操作 open-db9。

### 启动

```bash
# 设置环境变量
export DB9_API_URL=http://localhost:8080
export DB9_API_TOKEN=your-jwt-token
export DB9_RAG_URL=http://localhost:8001

# 运行
go run ./cmd/mcp-server/
# 或
./build/db9-mcp
```

### Claude Desktop 配置

编辑 `~config/claude/settings.json`:

```json
{
  "mcpServers": {
    "open-db9": {
      "command": "./db9-mcp",
      "env": {
        "DB9_API_URL": "http://localhost:8080",
        "DB9_API_TOKEN": "your-token"
      }
    }
  }
}
```

### 可用工具 (11 个)

`execute_sql`, `list_databases`, `create_database`, `list_files`, `upload_file`, `download_file`, `query_rag`, `get_schema`, `create_snapshot`, `list_snapshots`, `restore_snapshot`

---

## Agent Onboarding CLI

为 AI Agent 安装 db9 技能包，快速上手 open-db9 的完整能力。

### 安装命令

```bash
# 为 Claude Code 安装 (用户级)
db9 onboard --agent claude

# 为 Cursor IDE 安装 (项目级)
db9 onboard --agent cursor --scope project

# 同时安装到用户和项目
db9 onboard --agent cline --scope both

# 查看已安装的技能
db9 onboard --list

# 卸载
db9 onboard --uninstall claude
```

### 支持的 Agent

| Agent | 安装路径 | Scope |
|-------|---------|-------|
| Claude Code | `~/.claude/commands/db9.md` | user + project |
| Cursor | `.cursor/rules/db9.md` | project only |
| Cline | `.cline/rules/db9.md` | user + project |
| Codex | `~/.codex/commands/db9.md` | user + project |
| OpenCode | `~/.config/opencode/commands/db9.md` | user + project |

---

## 获取帮助

- 📖 [API 文档](API.md)
- 🔒 [安全指南](SECURITY.md)
- 🚀 [部署指南](DEPLOYMENT.md)
- 🐛 [问题反馈](https://github.com/open-db9/db9/issues)
- 💬 [Discussions](https://github.com/open-db9/db9/discussions)

---

**文档版本**: 2.0
**最后更新**: 2026-04-03

# Open-DB9

> **The Open-Source, Self-Hosted "Postgres for Agents"** — db9.ai, but open & yours.

[![Go Version](https://img.shields.io/badge/Go-1.25-blue)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)
[![Security](https://img.shields.io/badge/security-A-brightgreen)](docs/SECURITY.md)

---

## 为什么选择 Open-DB9?

Open-DB9 是 **db9.ai** 的开源自托管替代品，专为 AI Agent 时代设计的 PostgreSQL 平台。它不仅提供传统的多租户数据库管理、分支和文件存储能力，更在 Phase 7 中引入了 **RAG 智能检索、MCP Server、Agent Onboarding** 等 AI 原生功能，让 Claude、Cursor、Cline 等 AI 编码助手能够直接与数据库交互。

**自托管意味着完全掌控数据** — 无需将数据库凭证交给第三方，所有数据存储在你自己的 PostgreSQL 实例中。无论是个人开发者搭建 AI 工作流，还是团队构建 Agent 基础设施，Open-DB9 都提供了从数据库操作到向量检索的一站式解决方案。

---

## 架构总览

```
┌─────────────────────────────────────────────────────────────────────┐
│                        Open-DB9 Platform                           │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐           │
│  │   CLI    │  │ API      │  │  FS9     │  │   RAG    │           │
│  │  (db9)   │  │ Server   │  │ Service  │  │  Server  │           │
│  │          │  │ :8080    │  │ :9090    │  │ :8001    │           │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘           │
│       │             │             │             │                  │
│       └─────────────┴─────────────┴─────────────┘                  │
│                            │                                       │
│                    ┌───────▼────────┐                             │
│                    │  MCP Server    │ ◄── Claude / Cursor / Cline  │
│                    │  (stdio)       │                              │
│                    └────────────────┘                              │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
                            │
                    ┌───────▼────────┐
                    │  PostgreSQL    │
                    │  + pgvector    │
                    └────────────────┘
```

---

## 核心特性

### P0 基础能力 (Phase 1-6)

| 特性 | 描述 |
|------|------|
| **多租户 PostgreSQL 管理** | 支持多个独立租户，每个租户拥有隔离的数据库实例 |
| **数据库分支与快照** | 基于 pg_dump 的快照机制，支持创建实验性分支 |
| **FS9 文件存储系统** | 内置文件系统抽象，支持上传/下载/列表/复制/删除 |
| **REST API + CLI 工具** | 完整的 RESTful API 和强大的命令行工具 |
| **JWT 认证 + 安全防护** | Bearer Token 认证、速率限制、SQL 注入防护 |
| **Schema 内省 + Metrics 可观测性** | 数据库结构查询和 Prometheus 指标导出 |

### P1 AI 能力 (Phase 7 新增) 🚀

| 特性 | 描述 |
|------|------|
| 🧠 **RAG 智能检索** | 基于 LangChain + pgvector 的文档检索增强生成系统，支持 PDF/TXT/MD 多格式文档 |
| 🔌 **MCP Server** | 遵循 Model Context Protocol 标准，Claude Desktop / Cursor 原生集成 |
| 🤖 **Agent Onboarding** | `db9 onboard --agent claude` 一键为 AI 编码助手安装 DB9 技能文档 |
| 🌐 **HTTP-from-SQL 扩展** | 安全的外部 API 调用扩展，内置 SSRF 防护和大小限制 |
| 👤 **匿名使用** | 无需注册即可开始使用，24 小时临时会话，后续可升级为正式账户 |

---

## 快速开始

### 安装

#### Docker 一键启动 (推荐)

```bash
# 克隆项目
git clone https://github.com/open-db9/db9.git
cd db9

# 使用 Docker Compose 启动全部服务
docker compose -f deployments/docker/docker-compose.yml up -d
```

#### 从源码构建

```bash
# 安装 Go 依赖
go mod download

# 构建 CLI 和服务
make build

# 或单独安装
go install github.com/open-db9/db9/cmd/db9@latest
go install github.com/open-db9/db9/cmd/server@latest
go install github.com/open-db9/db9/cmd/fs9-service@latest
go install github.com/open-db9/db9/cmd/mcp-server@latest

# 安装 Python RAG 依赖
cd rag && pip install -e . && cd ..
```

### 五步上手

```bash
# 1️⃣ 启动 FS9 文件存储服务 (端口 9090)
./build/fs9-service &
# 或 Docker: docker run -p 9090:9090 open-db9/fs9-service

# 2️⃣ 启动 API 服务器 (端口 8080)
./build/server &
# 或 Docker: docker run -p 8080:8080 open-db9/api-server

# 3️⃣ 启动 RAG Server [Phase 7 新增] (端口 8001)
cd rag && uvicorn src.api.server:app --host 0.0.0.0 --port 8001 &

# 4️⃣ 创建数据库 (或使用匿名注册跳过此步)
export TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/users/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"secret"}' | jq -r '.data.token')

curl -X POST http://localhost:8080/api/v1/databases \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"mydb","type":"postgres"}'

# 5️⃣ 开始查询 / 使用 MCP / 接入 Agent
./build/db9 db sql 1 "SELECT 1 as test"
```

---

## 使用示例

### CLI 命令

```bash
# ====== 基础命令 ======

# SQL 查询
db9 db sql 1 "SELECT * FROM users LIMIT 10"

# 快照管理
db9 db snapshot create 1 --name "backup-before-migration"
db9 db snapshot list 1
db9 db snapshot restore 1 <snapshot-id>

# 分支管理
db9 db branch create 1 --name "feature-auth-v2"
db9 db branch list 1
db9 db branch delete 1 --branch <branch-id> --yes

# 文件操作
db9 fs upload 1 ./report.pdf /documents/report.pdf
db9 fs download 1 /documents/report.pdf ./output.pdf
db9 fs list 1 /documents/
db9 fs copy 1 /src/file.txt /dst/file.txt

# 类型生成
db9 gen types ./output --language typescript

# 认证管理
db9 auth login postgres --username admin --password secret
db9 auth status
db9 auth logout postgres

# ====== Phase 7 新增命令 ======

# Agent Onboarding - 为 AI 编码助手安装 DB9 技能
db9 onboard --agent claude              # Claude Code
db9 onboard --agent cursor --scope project  # Cursor IDE
db9 onboard --agent cline               # VSCode Cline
db9 onboard --agent codex               # OpenAI Codex
db9 onboard --list                      # 列出已安装的 Agent
```

### API 调用

```bash
# ====== 传统认证流程 ======

# 登录获取 Token
curl -X POST http://localhost:8080/api/v1/users/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"secret"}'

# 执行 SQL
curl -X POST http://localhost:8080/api/v1/databases/1/sql \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"query":"SELECT * FROM users LIMIT 10"}'

# 创建快照
curl -X POST http://localhost:8080/api/v1/databases/1/snapshots \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"backup-2026-04-03"}'

# 上传文件
curl -X POST http://localhost:8080/api/v1/databases/1/files \
  -H "Authorization: Bearer $TOKEN" \
  -F "file=@./file.txt" \
  -F "path=/documents"

# ====== Phase 7 新增端点 ======

# 匿名注册 (24小时临时会话，最多5个数据库)
curl -X POST http://localhost:8080/api/v1/auth/anonymous \
  -H "Content-Type: application/json" \
  -d '{"session_id":"unique-session-123"}'
# 返回: { token, tenant_id, capabilities: {"max_databases": 5}, expires_at }

# 升级匿名账户为正式账户
curl -X POST http://localhost:8080/api/v1/auth/claim \
  -H "Authorization: Bearer $REGISTERED_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"session_id":"unique-session-123"}'

# RAG 文档上传 (需要 RAG Server 运行中)
curl -X POST http://localhost:8001/api/v1/databases/1/rag/documents \
  -H "Authorization: Bearer $TOKEN" \
  -F "file=@./document.pdf" \
  -F 'title="API Documentation"'

# RAG 智能查询
curl -X POST http://localhost:8001/api/v1/databases/1/rag/query \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"query":"How does authentication work?", "k": 4}'
```

### MCP 集成

MCP (Model Context Protocol) 让 Claude Desktop、Cursor 等 AI 工具原生访问 Open-DB9。

#### Claude Desktop 配置

编辑 `~/Library/Application Support/Claude/claude_desktop_config.json` (macOS) 或 `%APPDATA%\Claude\claude_desktop_config.json` (Windows):

```json
{
  "mcpServers": {
    "open-db9": {
      "command": "mcp-server",
      "args": [],
      "env": {
        "DB9_API_URL": "http://localhost:8080",
        "DB9_API_TOKEN": "your-jwt-token-here",
        "DB9_RAG_URL": "http://localhost:8001"
      }
    }
  }
}
```

配置完成后，重启 Claude Desktop 即可在对话中使用以下 MCP 工具：
- `db9_list_databases` - 列出所有数据库
- `db9_execute_sql` - 执行 SQL 查询
- `db9_create_snapshot` - 创建快照
- `db9_upload_file` - 上传文件到 FS9
- `rag_query` - RAG 智能检索
- `rag_ingest_document` - 上传文档到 RAG 系统

#### Cursor IDE 配置

Cursor 通过 `.cursor/rules/db9.md` 规则文件集成：

```bash
# 一键安装 Cursor 规则
db9 onboard --agent cursor --scope project
```

---

## 项目结构

```
open_db9/
├── cmd/                          # 可执行程序入口
│   ├── db9/                     # CLI 工具
│   │   └── main.go
│   ├── server/                   # API 服务器 (:8080)
│   │   └── main.go
│   ├── fs9-service/              # 文件存储服务 (:9090)
│   │   ├── main.go
│   │   └── README.md
│   ├── mcp-server/              # MCP Server [Phase 7] (stdio)
│   │   ├── main.go              # 服务入口
│   │   ├── handlers.go          # MCP Tool 注册
│   │   └── server_test.go
│   └── secrets-migrate/         # 凭证迁移工具
│       └── main.go
│
├── internal/                     # 核心业务逻辑
│   ├── api/                      # REST API 层
│   │   ├── handlers/
│   │   │   ├── handlers.go      # 核心请求处理器
│   │   │   ├── sql.go           # SQL 执行
│   │   │   ├── snapshots.go     # 快照管理
│   │   │   ├── branches.go      # 分支管理
│   │   │   ├── files.go         # 文件操作
│   │   │   ├── rag.go           # RAG 代理 [Phase 7]
│   │   │   ├── http_ext.go     # HTTP-from-SQL [Phase 7]
│   │   │   ├── anonymous.go    # 匿名注册 [Phase 7]
│   │   │   ├── extensions.go   # 扩展管理
│   │   │   ├── metrics.go      # Prometheus 指标
│   │   │   ├── schema.go       # Schema 内省
│   │   │   └── health.go       # 健康检查
│   │   ├── middleware/
│   │   │   ├── auth.go         # JWT 认证中间件
│   │   │   ├── ratelimit.go    # 速率限制
│   │   │   ├── cors.go         # CORS 配置
│   │   │   ├── anon_limit.go   # 匿名用户限制 [Phase 7]
│   │   │   └── tracing.go      # 请求追踪
│   │   └── router/
│   │       └── router.go       # 路由定义
│   │
│   ├── cli/                     # CLI 命令层
│   │   ├── root.go              # 根命令
│   │   ├── config.go            # 配置加载
│   │   ├── auth.go              # 认证命令
│   │   ├── cmd/
│   │   │   └── onboard.go      # Agent Onboarding [Phase 7]
│   │   ├── commands/
│   │   │   └── commands.go     # FS 命令
│   │   └── db/                  # 数据库子命令
│   │       ├── db.go            # 数据库 CRUD
│   │       ├── sql.go           # SQL 执行
│   │       ├── snapshot.go      # 快照操作
│   │       ├── branch.go        # 分支操作
│   │       ├── extensions.go    # 扩展管理
│   │       ├── gen.go           # 类型生成
│   │       ├── inspect.go       # Schema 检查
│   │       └── metrics.go       # 指标查看
│   │
│   ├── database/                # 数据库管理层
│   │   ├── manager.go           # 连接池管理
│   │   ├── pool.go              # 租户连接池
│   │   ├── snapshot.go          # 快照实现
│   │   ├── branch_test.go       # 分支测试
│   │   ├── extensions.go        # 扩展注册
│   │   ├── introspection.go     # Schema 内省
│   │   └── metrics.go           # 数据库指标
│   │
│   ├── extensions/
│   │   └── http/               # HTTP 扩展 [Phase 7]
│   │       ├── client.go       # 安全 HTTP 客户端 (SSRF 防护)
│   │       ├── ratelimit.go    # HTTP 速率限制
│   │       └── client_test.go
│   │
│   ├── auth/                    # JWT 认证模块
│   │   └── auth.go
│   ├── secrets/                 # AES-256-GCM 凭证加密
│   │   └── secrets.go
│   ├── filesystem/              # 文件系统抽象
│   │   ├── storage.go
│   │   ├── local.go
│   │   └── disk.go
│   ├── models/                  # 数据模型
│   │   ├── models.go
│   │   └── user.go
│   ├── config/                  # 配置管理
│   │   └── config.go
│   ├── validate/                # 输入验证
│   │   └── validate.go
│   ├── typescript/              # TypeScript 类型生成器
│   │   └── generator.go
│   └── python/                  # Python 类型生成器
│       └── generator.go
│
├── rag/                         # RAG 智能检索系统 [Phase 7]
│   ├── src/
│   │   ├── api/                 # FastAPI 服务
│   │   │   ├── server.py        # 应用入口
│   │   │   ├── routes.py        # API 路由
│   │   │   ├── models.py        # 请求/响应模型
│   │   │   └── auth.py          # JWT 认证
│   │   ├── ingestion/           # 文档处理管道
│   │   │   ├── pipeline.py      # ETL 管道
│   │   │   ├── parsers.py       # PDF/TXT/MD 解析器
│   │   │   └── chunkers.py      # 智能分块策略
│   │   ├── query/               # 检索引擎
│   │   │   └── engine.py        # RAG 查询引擎
│   │   ├── vectorstore/         # 向量存储
│   │   │   ├── db9_vectorstore.py  # 自定义 LangChain VectorStore
│   │   │   └── embeddings.py    # Embedding 生成器
│   │   ├── config/              # 配置管理
│   │   │   └── settings.py
│   │   ├── cli/                 # RAG CLI 工具
│   │   │   └── commands.py
│   │   └── dev/                 # 开发工具
│   │       └── bootstrap.py     # 本地环境初始化
│   ├── tests/
│   │   ├── unit/                # 单元测试
│   │   ├── integration/         # 集成测试
│   │   └── e2e/                 # 端到端测试
│   ├── scripts/                 # 辅助脚本
│   ├── pyproject.toml           # Python 依赖
│   ├── Dockerfile               # RAG 服务容器
│   └── README.md                # RAG 文档
│
├── migrations/control/          # 数据库迁移脚本
│   ├── 001_init_schema.up.sql
│   ├── 002_migrations.up.sql
│   ├── 003_fs9_metadata.up.sql
│   ├── 004_observability.up.sql
│   ├── 005_fs9_metadata.up.sql
│   ├── 006_add_branch_support.up.sql
│   ├── 007_connection_pools.up.sql
│   └── 008_rag_tables.up.sql    # RAG 表 [Phase 7]
│
├── deployments/
│   ├── docker/
│   │   ├── docker-compose.yml   # 编排文件
│   │   ├── Dockerfile           # 主服务镜像
│   │   ├── Dockerfile.cli       # CLI 镜像
│   │   ├── postgres-init/
│   │   │   └── init-db.sh       # PG 初始化脚本
│   │   └── .env.example         # 环境变量模板
│   └── config.yaml.example      # 配置文件模板
│
├── e2e/                         # E2E 测试
│   ├── suite_test.go
│   └── setup.go
│
├── docs/                        # 项目文档
│   ├── API.md                   # REST API 参考
│   ├── DEPLOYMENT.md            # 生产部署指南
│   ├── SECURITY.md              # 安全指南
│   ├── DISASTER_RECOVERY.md     # 灾难恢复
│   ├── INCIDENT_RESPONSE.md     # 应急响应
│   └── README.md                # 文档索引
│
├── pkg/                         # 公共包
│   ├── logger/logger.go
│   └── utils/utils.go
│
├── scripts/                     # 快速启动脚本
│   ├── quickstart.sh            # Linux/macOS
│   └── quickstart.ps1           # Windows
│
├── Makefile                     # 构建任务
├── go.mod                       # Go 模块定义
├── go.sum                       # 依赖校验
├── Dockerfile                   # 根容器
├── install.sh                   # 安装脚本
└── LICENSE                      # MIT 许可证
```

---

## 配置

### 环境变量

| 变量 | 描述 | 默认值 | 适用组件 |
|------|------|--------|----------|
| **服务器** ||||
| `PORT` | API 服务器端口 | `8080` | Server |
| `HOST` | API 服务器主机 | `localhost` | Server |
| `TIMEOUT` | 请求超时(秒) | `30` | Server |
| **数据库** ||||
| `DB_HOST` | PostgreSQL 主机 | `localhost` | Server |
| `DB_PORT` | PostgreSQL 端口 | `5432` | Server |
| `DB_USER` | PostgreSQL 用户 | `db9` | Server |
| `DB_PASSWORD` | PostgreSQL 密码 | *(必填)* | Server |
| `DB_NAME` | 数据库名称 | `db9` | Server |
| **认证** ||||
| `AUTH_SECRET` | JWT 密钥 (>=32字符) | *(必填)* | Server |
| `AUTH_EXPIRES_IN` | JWT 过期时间(秒) | `3600` | Server |
| **CORS** ||||
| `CORS_ALLOWED_ORIGINS` | 允许的来源 | `http://localhost:3000,http://localhost:8080` | Server |
| `CORS_ALLOWED_METHODS` | 允许的方法 | `GET,POST,PUT,DELETE,OPTIONS` | Server |
| **速率限制** ||||
| `RATE_LIMIT_ENABLED` | 是否启用限流 | `false` | Server |
| `RATE_LIMIT_RPM` | 每分钟请求数 | `100` | Server |
| `RATE_LIMIT_BURST` | 突发请求数 | `20` | Server |
| **FS9 文件存储** ||||
| `DB9_FS9_URL` | FS9 服务 URL | `http://localhost:9090` | Server |
| **RAG 系统 [Phase 7]** ||||
| `RAG_DB_HOST` | RAG 用 PG 主机 | `localhost` | RAG Server |
| `RAG_DB_PORT` | RAG 用 PG 端口 | `5432` | RAG Server |
| `RAG_DB_USER` | RAG 用 PG 用户 | `postgres` | RAG Server |
| `RAG_DB_PASSWORD` | RAG 用 PG 密码 | - | RAG Server |
| `RAG_DB_NAME` | RAG 用数据库名 | `db9` | RAG Server |
| `RAG_DB9_API_URL` | Open-DB9 API URL | `http://localhost:8080` | RAG Server |
| `RAG_FS9_URL` | FS9 服务 URL | `http://localhost:9090` | RAG Server |
| **MCP Server [Phase 7]** ||||
| `DB9_API_URL` | API 服务器 URL | `http://localhost:8080` | MCP Server |
| `DB9_API_TOKEN` | JWT Token | *(必填)* | MCP Server |
| `DB9_RAG_URL` | RAG 服务 URL | `http://localhost:8001` | MCP Server |

### 配置文件

创建 `$HOME/.open-db9.yaml`:

```yaml
server:
  port: "8080"
  host: "localhost"
  timeout: 30

database:
  driver: "postgres"
  host: localhost
  port: 5432
  user: db9
  password: "${DB_PASSWORD}"
  database: db9

auth:
  secret: "${AUTH_SECRET}"
  expires_in: 3600

cors:
  allowed_origins: "http://localhost:3000,http://localhost:8080"
  allowed_methods: "GET,POST,PUT,DELETE,OPTIONS"
  allowed_headers: "Content-Type,Authorization"
  exposed_headers: ""
  allow_credentials: false
  max_age: 86400

rate_limit:
  enabled: false
  requests_per_minute: 100
  burst: 20
```

---

## 技术栈

| 层级 | 技术 | 说明 |
|------|------|------|
| **后端框架** | Go 1.25 | 高性能 API Server + CLI |
| **Web 框架** | net/http 标准库 | HTTP 路由和中间件 |
| **数据库** | PostgreSQL 15+ | 主存储引擎 |
| **向量扩展** | pgvector | 相似度搜索 |
| **认证** | golang-jwt/v5 | JWT Token 管理 |
| **CLI 框架** | Cobra | 命令行界面 |
| **配置** | Viper | 多源配置管理 |
| **可观测性** | Prometheus | Metrics 导出 |
| **RAG 引擎** | Python 3.10 + FastAPI | 文档检索服务 |
| **RAG 框架** | LangChain | 向量检索链路 |
| **Embedding** | OpenAI API | 文本向量化 |
| **MCP 协议** | mcp-go | Model Context Protocol |
| **加密** | AES-256-GCM | 凭证静态加密 |
| **容器化** | Docker + Compose | 一键部署 |

---

## 测试

```bash
# 运行 Go 单元测试 (150+ tests, 80%+ coverage)
make test

# 运行 E2E 测试
make test-e2e

# 运行性能基准测试
make benchmark

# 测试覆盖率报告
make test-coverage

# 运行 RAG 系统测试 [Phase 7]
cd rag
pytest tests/unit/                          # 单元测试
pytest tests/integration/                   # 集成测试
pytest tests/e2e/                           # 端到端测试
pytest --cov=rag.src --cov-report=html      # 覆盖率报告
```

---

## 文档链接

| 文档 | 描述 |
|------|------|
| [docs/README.md](docs/README.md) | 文档索引 |
| [docs/API.md](docs/API.md) | REST API 完整参考 |
| [docs/SECURITY.md](docs/SECURITY.md) | 安全特性和最佳实践 |
| [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md) | 生产环境部署指南 |
| [docs/DISASTER_RECOVERY.md](docs/DISASTER_RECOVERY.md) | 备份恢复方案 |
| [docs/INCIDENT_RESPONSE.md](docs/INCIDENT_RESPONSE.md) | 应急响应手册 |
| [rag/README.md](rag/README.md) | RAG 系统详细文档 |
| [CONTRIBUTING.md](CONTRIBUTING.md) | 贡献指南 |

---

## License

MIT License — 详见 [LICENSE](LICENSE) 文件。

---

**Made with commitment by the Open-DB9 Team**

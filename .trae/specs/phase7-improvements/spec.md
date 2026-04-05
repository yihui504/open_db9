# Open-DB9 Phase 7 改进方案 Spec

## Why

基于定位分析（"db9.ai 的开源自托管替代品"），当前 open-db9 已具备 67% 核心功能，但存在关键缺口：Schema/Metrics API 返回 501、匿名流程未打通、RAG 缺测试、缺少 Agent 集成能力。本 Phase 旨在补齐这些短板，使产品达到可对外展示的 MVP 状态。

## What Changes

### P0 — 补齐核心体验（定义最小可用产品）

- **完成 Schema 内省 API**: `GetTableDetails`/`ListTables`/`ListSchemas` 已有完整实现但路由注册可能有问题，需验证并修复 501 问题
- **完成 Metrics 可观测性 API**: `MetricsHandler` 当前返回 501 Not Implemented，需实现数据库统计、查询性能、慢查询等端点
- **打通匿名使用流程**: `anonymous_accounts` 表已存在于 schema 中，需实现匿名注册→建库→SSO 认领的完整链路
- **补充 RAG 子系统测试**: e2e/integration 测试目录为空，仅有 3 个 unit test

### P1 — 差异化能力（拉开与 Supabase 距离）

- **HTTP-from-SQL 扩展**: 实现 SQL 层 HTTP 调用能力（对标 db9.ai 的 `http_get()`/`http_post()`）
- **Agent Onboarding CLI 命令**: `db9 onboard --agent <name>` 让 AI Agent 学会使用 DB9
- **MCP Server**: 通过 Model Context Protocol 让 Claude/Cursor 等 AI 工具直接操作 open-db9

## Impact

- Affected specs: project-understanding 定位分析
- Affected code:
  - `internal/api/handlers/metrics.go` — 重写 MetricsHandler
  - `internal/api/handlers/schema.go` — 验证并修复（已有实现）
  - `internal/api/router/router.go` — 可能需要调整路由注册
  - `internal/cli/commands/` — 新增 onboard 命令
  - `internal/database/manager.go` — 匿名账户 CRUD
  - `rag/tests/e2e/`, `rag/tests/integration/` — 补充测试
  - 新增: `internal/extensions/http/` — HTTP-from-SQL 模块
  - 新增: `cmd/mcp-server/` — MCP Server 入口

---

# ADDED Requirements

## Requirement: Schema Introspection API 完整可用

系统 SHALL 提供完整的数据库 Schema 内省 REST API，返回表结构、列信息、索引、约束、触发器等元数据。

#### Scenario: 获取表详情成功
- **WHEN** 用户 GET `/api/v1/databases/:id/schema/table?schema=public&table=users`
- **THEN** 返回 200 JSON，包含 columns[] / indexes[] / constraints[] / triggers[] / row_count_estimate / size_bytes

#### Scenario: 列出所有表成功
- **WHEN** 用户 GET `/api/v1/databases/:id/schema/tables?schema=public`
- **THEN** 返回 200 JSON，包含 tables[{name, type}] 和 count

## Requirement: Metrics 可观测性 API

系统 SHALL 提供 REST API 端点暴露数据库运行时指标，包括连接池状态、查询统计和慢查询列表。

#### Scenario: 获取数据库统计概览
- **WHEN** 用户 GET `/api/v1/databases/:id/metrics/stats`
- **THEN** 返回 200 JSON，包含 total_connections / active_queries / total_tables / total_size_bytes / uptime_seconds

#### Scenario: 获取慢查询列表
- **WHEN** 用户 GET `/api/v1/databases/:id/metrics/slow?threshold=1000&limit=20`
- **THEN** 返回 200 JSON，包含 slow_queries[{query, duration_ms, calls, timestamp}]，按 duration_ms 降序排列

#### Scenario: 获取查询统计 Top N
- **WHEN** 用户 GET `/api/v1/databases/:id/metrics/queries?limit=10&sort=total_time`
- **THEN** 返回 200 JSON，包含 queries[{query_hash, query_text, calls, total_time_ms, avg_time_ms, rows}]

## Requirement: 匿名使用流程

系统 SHALL 允许用户在未注册状态下创建匿名账户并使用有限功能的数据库服务。

#### Scenario: 匿名用户创建账户和数据库
- **WHEN** 匿名用户 POST `/api/v1/auth/anonymous` 并提供 session_id
- **THEN** 创建 anonymous_accounts 记录，返回 JWT token 和临时 tenant_id，该账户最多创建 5 个数据库

#### Scenario: SSO 认领匿名账户
- **WHEN** 已登录用户 POST `/api/v1/auth/claim` 并提供 session_id
- **THEN** 将 anonymous_accounts 关联到正式 users 记录，转移所有权，解除 5 库限制

## Requirement: HTTP-from-SQL 扩展

系统 SHALL 提供 SQL 函数或 UDF 能力，允许在 SQL 查询中发起 HTTPS 请求获取外部数据。

#### Scenario: 从 SQL 发起 GET 请求
- **WHEN** 用户执行 `SELECT http_get('https://api.example.com/data')`
- **THEN** 返回 JSON 包含 status_code / body / headers，仅允许 HTTPS 目标

#### Scenario: 安全限制生效
- **WHEN** 用户尝试对私有 IP 或非 HTTPS 地址调用 http_get()
- **THEN** 返回安全错误 "SSRF protection: private IP addresses are blocked"

## Requirement: Agent Onboarding CLI 命令

系统 SHALL 提供 `db9 onboard` CLI 命令，将 open-db9 安装为 AI Agent 的技能包。

#### Scenario: 为 Claude Code 安装技能
- **WHEN** 用户执行 `db9 onboard --agent claude --scope user`
- **THEN** 在 ~/.claude/commands/ 目录生成 db9 相关命令文件，Agent 可自主调用 db9 CLI

#### Scenario: 为 Cursor 安装技能
- **WHEN** 用户执行 `db9 onboard --agent cursor --scope project`
- **THEN** 在 .cursor/rules/ 目录生成 db9 使用规则

## Requirement: MCP Server

系统 SHALL 提供 Model Context Protocol (MCP) Server，让支持 MCP 的 AI Agent 直接通过协议操作 open-db9。

#### Scenario: MCP 列出工具
- **WHEN** MCP Client 发送 tools/list 请求
- **THEN** 返回工具列表: execute_sql / list_databases / create_database / list_files / upload_file / query_rag 等

#### Scenario: MCP 调用 execute_sql
- **WHEN** MCP Client 发送 tools/call 请求，参数 {sql: "SELECT * FROM users LIMIT 5", database_id: "abc123"}
- **THEN** 执行 SQL 并返回结果集 JSON

---

# MODIFIED Requirements

## Requirement: RAG 子系统测试覆盖

原有 RAG 子系统仅有 3 个 unit test（parsers + vectorstore），无 e2e 和 integration 测试。Phase 7 SHALL 补充：

- IngestionPipeline Saga 事务测试（正常+回滚场景）
- QueryEngine 端到端测试（mock LLM）
- DB9VectorStore CTE 查询正确性验证
- API server handler 集成测试

---

# REMOVED Requirements

(无)

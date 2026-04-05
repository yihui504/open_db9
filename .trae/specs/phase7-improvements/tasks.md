# Tasks

## P0 — 补齐核心体验

- [x] Task 1: 验证并修复 Schema 内省 API (501 → 200)
  - [x] 1.1 确认 schema.go 中 GetTableDetails/ListTables/ListSchemas 实现完整性
  - [x] 1.2 检查 router.go 中路由注册是否正确映射到 SchemaHandler 方法
  - [x] 1.3 验证 validate 包中 IsValidSchemaName / IsValidTableName 是否已实现
  - [x] 1.4 手动测试 GET `/api/v1/databases/:id/schema/tables` 和 `/api/v1/databases/:id/schema/table` 端点
  - [x] 1.5 如有 501 问题，定位根因（handler 未注册 / 方法签名不匹配 / 缺少依赖注入）并修复

- [x] Task 2: 实现 Metrics 可观测性 API (替换 501 占位)
  - [x] 2.1 重写 `GetDatabaseStats` handler: 查询 pg_stat_database 获取连接数/事务数/缓存命中率
  - [x] 2.2 实现 `GetSlowQueries` handler: 查询 pg_stat_statements 按 mean_exec_time 排序，支持 threshold 和 limit 参数
  - [x] 2.3 实现 `GetQueryStats` handler: Top N 查询统计，支持 sort(total_time|calls|rows) 参数
  - [x] 2.4 在 MetricsHandler 中统一数据库连接获取逻辑（复用 schema.go 的连接模式）
  - [x] 2.5 添加请求参数校验和错误处理
  - [x] 2.6 编写单元测试覆盖正常/空结果/无效参数场景

- [x] Task 3: 打通匿名使用流程
  - [x] 3.1 创建 `internal/api/handlers/anonymous.go`: POST `/api/v1/auth/anonymous` handler
    - 接收 session_id，查询/创建 anonymous_accounts 记录
    - 调用 auth.Manager 为匿名用户生成 JWT
    - 返回 token + tenant_id + capabilities
  - [x] 3.2 创建 `POST /api/v1/auth/claim` handler:
    - 验证已登录状态（从 JWT context 获取 user_id）
    - 接收 session_id，查找对应 anonymous_account
    - 将 anonymous_account.tenant_id 关联到 users 表
    - 删除 anonymous_accounts 记录（或标记已认领）
    - 返回新 token（包含完整权限）
  - [x] 3.3 在 router.go 中注册匿名路由到公开路径组（无需认证）
  - [x] 3.4 实现匿名账户中间件：限制匿名用户每 tenant 最多 5 个数据库
  - [x] 3.5 编写测试：匿名创建 → 建库(≤5成功/>5失败) → SSO认领 → 权限升级验证

- [x] Task 4: 补充 RAG 子系统测试
  - [x] 4.1 编写 IngestionPipeline Saga 测试:
    - 正常流程: parse→upload→create_doc→chunk→update_count 全部成功
    - 回滚流程: chunk 步骤失败，验证前 3 步补偿事务执行
  - [x] 4.2 编写 QueryEngine 集成测试 (mock LLM):
    - 有文档时返回带 source 的回答
    - 无文档时返回降级回答
    - LLM 不可用时返回原始检索片段
  - [x] 4.3 编写 DB9VectorStore CTE 正确性测试:
    - add_texts 后 similarity_search 能找到刚添加的文本
    - metadata filter 正确过滤结果
    - delete 后搜索不再返回已删文本
  - [x] 4.4 编写 RAG API server handler 测试:
    - POST /documents 上传+索引
    - GET /documents 列表
    - DELETE /documents/:id 删除+清理向量

## P1 — 差异化能力

- [x] Task 5: 实现 HTTP-from-SQL 扩展
  - [x] 5.1 设计 `internal/extensions/http/client.go`:
    - HttpClient 结构体，封装 net.http.Client
    - 安全配置: Timeout=5s, MaxResponseBodyBytes=1MB, MaxRequestBodyBytes=256KB
    - SSRF 防护: 禁止私有 IP (10.x / 172.16-31.x / 192.168.x / 127.x / 169.254.x)
    - 仅允许 HTTPS 协议
  - [x] 5.2 实现核心函数:
    - `HttpGet(ctx, url, headers) -> {status, body, headers}`
    - `HttpPost(ctx, url, body, contentType, headers) -> {status, body, headers}`
  - [x] 5.3 创建 SQL UDF 包装层 `internal/extensions/http/udf.go`:
    - 注册为 PostgreSQL 函数（通过 CREATE FUNCTION 或在应用层实现）
    - 应用层方案: 提供 REST API `POST /api/v1/databases/:id/http/get` 和 `/http/post`
  - [x] 5.4 实现速率限制: 每语句最多 100 请求, 每租户并发上限 20
  - [x] 5.5 编写测试: 正常请求 / SSRF 阻止 / 超时处理 / 大响应截断 / 速率限制触发

- [x] Task 6: 实现 Agent Onboarding CLI 命令
  - [x] 6.1 创建 `internal/cli/commands/onboard.go`:
    - Cobra 子命令: `db9 onboard --agent <name> [--scope user|project|both]`
    - 支持 agent 类型: claude, codex, cursor, cline, opencode
  - [x] 6.2 实现各 Agent 的技能文件模板:
    - Claude Code (~/.claude/commands/db9.md): db9 create/sql/fs/branch 等命令说明
    - Cursor (.cursor/rules/db9.md): 使用规则和最佳实践
    - Cline (.cline/rules/db9.md): 类似 Cursor 格式
    - OpenCode (config.yaml 片段)
  - [x] 6.3 实现安装逻辑:
    - 检测目标 Agent 是否已安装
    - 写入技能文件到正确路径
    - scope=user 时写入全局目录, scope=project 时写入当前项目
    - 验证写入成功
  - [x] 6.4 实现 `db9 onboard --list` 列出已安装的 Agent 技能
  - [x] 6.5 实现 `db9 onboard --uninstall <agent>` 卸载技能

- [x] Task 7: 实现 MCP Server
  - [x] 7.1 创建 `cmd/mcp-server/main.go`:
    - 使用 MCP SDK (github.com/mark3labs/mcp-go 或类似库)
    - stdio 传输模式（适合 AI Agent 子进程调用）
    - 配置从环境变量读取 (DB9_API_URL, DB9_TOKEN)
  - [x] 7.2 实现工具注册:
    - `execute_sql`: 参数 {database_id, sql} → 连接目标 DB 执行 SQL 返回结果
    - `list_databases`: 列出当前用户可访问的数据库
    - `create_database`: 创建新数据库
    - `list_files`: 列出数据库关联的文件
    - `upload_file`: 上传文件到 FS9
    - `query_rag`: 对 RAG 索引进行自然语言查询
    - `get_schema`: 获取数据库表结构
    - `create_snapshot`: 创建快照
    - `list_snapshots`: 列出快照
  - [x] 7.3 实现工具调用逻辑: 每个 tool handler 调用对应的 API Server REST endpoint
  - [x] 7.4 错误处理: API 错误转换为 MCP error response，包含可操作的错误信息
  - [x] 7.5 编写 MCP client 测试: 用 mock server 验证 tools/list 和 tools/call

# Task Dependencies

- [Task 2] depends on [Task 1] — Metrics 复用 Schema 的连接模式，先确认 Schema 正常 ✅
- [Task 3] depends on [Task 1] — 匿名路由注册需确认路由系统工作正常 ✅
- [Task 4] 可与 [Task 1][2][3] 并行 — RAG 测试独立于 API 改进 ✅
- [Task 5] 可与 [Task 1-4] 并行 — HTTP 扩展是新增模块 ✅
- [Task 6] 可与 [Task 1-5] 并行 — CLI 命令独立 ✅
- [Task 7] depends on [Task 1] partially — MCP 的 execute_sql/get_schema 需要 Schema API 可用 ✅

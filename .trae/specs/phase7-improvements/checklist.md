# Checklist

## P0 验证点

- [x] Schema API 端点返回 200（非 501）: GET /schema/tables, GET /schema/table, GET /schema (ListSchemas)
- [x] Schema API 返回数据结构正确: 包含 columns/indexes/constraints/triggers/row_count/size
- [x] Metrics API 端点返回 200: GET /metrics/stats, GET /metrics/slow, GET /metrics/queries
- [x] Metrics stats 返回有效数据: total_connections, active_queries, total_tables, total_size_bytes
- [x] Metrics slow queries 按 duration_ms 降序排列，支持 threshold 参数过滤
- [x] Metrics queries Top N 支持 sort 参数 (total_time/calls/rows)
- [x] 匿名注册 POST /api/v1/auth/anonymous 成功返回 JWT + tenant_id
- [x] 匿名用户可创建数据库（≤5 个限制生效）
- [x] 匿名用户创建第 6 个数据库时被拒绝
- [x] SSO 认领 POST /api/v1/auth/claim 成功转移所有权
- [x] 认领后匿名限制解除，可创建超过 5 个数据库
- [x] RAG IngestionPipeline 正常流程测试通过 (全步骤成功)
- [x] RAG IngestionPipeline 回滚流程测试通过 (失败时补偿事务执行)
- [x] RAG QueryEngine 集成测试通过 (mock LLM 场景覆盖完整)
- [x] RAG DB9VectorStore CTE 查询正确性验证通过

## P1 验证点

- [x] HTTP GET 请求正常返回 {status_code, body, headers}
- [x] HTTP POST 请求正常发送并返回响应
- [x] HTTP 扩展阻止私有 IP 地址访问 (SSRF 防护生效)
- [x] HTTP 扩展阻止非 HTTPS 协议请求
- [x] HTTP 扩展超时处理正确 (5 秒超时返回错误)
- [x] HTTP 扩展大响应截断 (>1MB 时截断)
- [x] HTTP 速率限制触发 (每语句 >100 请求时拒绝)
- [x] `db9 onboard --agent claude` 成功写入技能文件到 ~/.claude/commands/
- [x] `db9 onboard --agent cursor` 成功写入规则到 .cursor/rules/
- [x] `db9 onboard --list` 正确列出已安装的 Agent 技能
- [x] `db9 onboard --uninstall claude` 成功删除技能文件
- [x] MCP Server 启动成功，stdio 模式正常运行
- [x] MCP tools/list 返回完整工具列表 (≥8 个工具)
- [x] MCP tools/call execute_sql 成功执行 SQL 并返回结果
- [x] MCP tools/call query_rag 成功查询 RAG 并返回回答
- [x] MCP 错误处理正确: API 错误转换为 MCP error response

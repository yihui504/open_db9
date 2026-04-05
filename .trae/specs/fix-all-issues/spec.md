# 全盘修复与演示验证 Spec

## Why

演示运行 (`run-demo.sh`) 存在多个功能失败问题，同时 README.md 包含与实际代码不符的描述。需要系统性修复所有 Bug，确保演示完全按预期成功运行，并修正项目文档。

## What Changes

### Bug 修复（代码层面）

1. **files.go Sscanf URL 解析 Bug（3处）** — 与已修复的 sql.go 相同的问题，`fmt.Sscanf` 无法正确解析含斜杠的 URL 路径，导致文件上传、列表、复制全部返回 400 错误
2. **metrics.go PathValue() Bug（3处）** — 使用 Gin/Chi 的 `r.PathValue("id")` 方法，但项目实际使用标准库 `net/http` 的 `http.ServeMux`，该方法始终返回空字符串，导致 Metrics 端点全部失败
3. **http_ext.go PathValue() Bug（2处）** — 同上，HTTP 扩展端点无法获取数据库 ID
4. **RAG 端点 URL 不匹配** — demo 脚本中 RAG 调用路径与 Python FastAPI 服务实际挂载路由不一致
5. **匿名注册流程问题** — 需要排查并修复匿名账户创建失败的根本原因

### 文档修正（README.md）

6. **Web 框架描述错误** — 声称使用 Gin，实际使用标准库 `net/http`
7. **其他与实际代码不一致之处** — 需全盘检查并修正

## Impact

- Affected specs: 所有现有 spec（phase7-improvements, showcase-project 等）
- Affected code:
  - `internal/api/handlers/files.go` — UploadFileHandler, ListFilesHandler, CopyFileHandler
  - `internal/api/handlers/metrics.go` — GetDatabaseStatsHandler, GetSlowQueriesHandler, GetQueryStatsHandler
  - `internal/api/handlers/http_ext.go` — GetRequest, PostRequest
  - `demo/run-demo.sh` — RAG 端点 URL 修正
  - `README.md` — 多处事实性错误修正

## ADDED Requirements

### Requirement: URL 路径解析统一使用 strings.Split

系统 SHALL 在所有 Handler 中使用 `strings.Split` 解析 URL 路径提取数据库 ID，禁止使用 `fmt.Sscanf` 或 `r.PathValue()`。

#### Scenario: 文件上传成功
- **WHEN** 用户 POST `/api/v1/databases/1/files` 并附带文件
- **THEN** 返回 200 和文件元数据 JSON

#### Scenario: Metrics 统计正常返回
- **WHEN** 用户 GET `/api/v1/databases/1/metrics/stats`
- **THEN** 返回包含 total_connections、total_tables 等统计数据的 JSON

#### Scenario: HTTP 扩展调用成功
- **WHEN** 用户 POST `/api/v1/databases/1/http/get` 并提供 URL
- **THEN** 返回外部 API 响应数据

### Requirement: RAG 端点 URL 与 FastAPI 路由一致

Demo 脚本中的 RAG 调用 URL SHALL 匹配 Python FastAPI 服务的实际路由挂载路径：
- 文档列表: `GET /api/v1/databases/{id}/rag/documents`
- RAG 查询: `POST /api/v1/databases/{id}/rag/query`

#### Scenario: RAG 文档列表查询
- **WHEN** RAG 服务运行中且用户请求文档列表
- **THEN** 返回文档数组或空数组（非 404 Not Found）

### Requirement: README.md 内容准确性

README.md 中所有技术描述 SHALL 与实际代码实现一致。

#### Scenario: 框架描述正确
- **WHEN** 用户阅读 README 技术栈表格
- **THEN** Web 框架显示为标准库 `net/http`（而非 Gin）

## MODIFIED Requirements

### Requirement: 演示脚本完整通过

修改后的 run-demo.sh SHALL 使全部 13 个步骤无关键错误完成（允许非核心功能的降级提示）。

## REMOVED Requirements

无

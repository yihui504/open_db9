# Tasks

- [x] Task 1: 修复 files.go 中 3 处 Sscanf URL 解析 Bug
  - [x] 1.1 修复 UploadFileHandler (line 78) — 改用 strings.Split
  - [x] 1.2 修复 ListFilesHandler (line 415) — 改用 strings.Split
  - [x] 1.3 修复 CopyFileHandler (line 534) — 改用 strings.Split

- [x] Task 2: 修复 metrics.go 中 3 处 PathValue() Bug
  - [x] 2.1 修复 GetDatabaseStatsHandler (line 26) — 改用 strings.Split 提取 dbID
  - [x] 2.2 修复 GetSlowQueriesHandler (line 91) — 同上
  - [x] 2.3 修复 GetQueryStatsHandler (line 195) — 同上

- [x] Task 3: 修复 http_ext.go 中 2 处 PathValue() Bug
  - [x] 3.1 修复 GetRequest (line 26) — 改用 strings.Split
  - [x] 3.2 修复 PostRequest (line 75) — 同上

- [x] Task 4: 修复 demo/run-demo.sh 中 RAG 端点 URL 不匹配问题
  - [x] 4.1 修正 RAG 文档列表 URL 为 `/api/v1/databases/$DB_ID/rag/documents`
  - [x] 4.2 修正 RAG 查询 URL 为 `/api/v1/databases/$DB_ID/rag/query`

- [x] Task 5: 排查并修复匿名注册流程失败问题
  - [x] 5.1 分析 anonymous_accounts 表是否存在及 schema 是否正确
  - [x] 5.2 修复匿名注册 Handler 或数据库初始化脚本（添加 updated_at 列）

- [x] Task 6: 全盘检查并修正 README.md 所有不符合实际之处
  - [x] 6.1 修正 Web 框架描述（Gin → net/http 标准库）
  - [x] 6.2 检查并修正其他技术栈描述、API 路径、命令示例等
  - [x] 6.3 确保所有文档与代码实现一致

- [x] Task 7: 重新构建 Docker 并运行完整演示验证
  - [x] 7.1 重建 Docker 镜像（含迁移脚本更新）
  - [x] 7.2 运行 run-demo.sh 确认全部步骤通过

# Task Dependencies
- [Task 1, 2, 3] 可并行执行（独立文件）
- [Task 4, 5] 可并行执行
- [Task 6] 可与上述任务并行
- [Task 7] 依赖 [Task 1-6] 全部完成

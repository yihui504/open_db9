# Tasks

- [x] Task 1: 重写 README.md 为 Phase 7 完整状态
  - [x] 1.1 更新 Slogan 和项目定位描述（"db9.ai 的开源自托管替代品"）
  - [x] 1.2 更新特性列表：新增 RAG / MCP Server / Agent Onboarding / HTTP-from-SQL / 匿名使用 5 大特性
  - [x] 1.3 更新快速开始：补充 RAG Server + MCP Server 启动步骤
  - [x] 1.4 更新 CLI 命令示例：补充 `db9 onboard` 命令
  - [x] 1.5 更新项目结构树：包含 cmd/mcp-server/, internal/extensions/http/, internal/cli/cmd/, rag/, internal/api/handlers/ 新文件
  - [x] 1.6 更新环境变量表：补充 RAG/MCP 相关变量
  - [x] 1.7 更新 API 端点索引：补充 Schema/Metrics/RAG/Anonymous/HTTP 端点
  - [x] 1.8 更新测试命令：补充 RAG 测试和 MCP 测试

- [x] Task 2: 更新 PROGRESS.md 至 Phase 7 完成
  - [x] 2.1 新增 "Phase 7: MVP 完善" 章节（7 个 Task 概述及状态）
  - [x] 2.2 更新测试覆盖表（40+ → 150+，按模块细分）
  - [x] 2.3 更新技术栈表（增加 Python/FastAPI/LangChain/mcp-go/pgvector）
  - [x] 2.4 更新项目结构树（与 README 同步）
  - [x] 2.5 将原 "Phase 7 待实现" 替换为已完成内容摘要
  - [x] 2.6 新增 "Phase 8: 路线图" 未来规划章节（pgwire/Cron/多存储后端/Helm Chart/SDK）

- [x] Task 3: 更新 docs/README.md 补充新端点和功能文档
  - [x] 3.1 新增 RAG API 端点组文档（POST/GET/DELETE /rag/documents, POST /rag/query）
  - [x] 3.2 新增匿名认证端点文档（POST /auth/anonymous, POST /auth/claim，含请求/响应示例）
  - [x] 3.3 新增 HTTP 扩展端点文档（POST /http/get, POST /http/post，含安全限制说明）
  - [x] 3.4 新增 Schema 内省完整端点文档（GET /schema, /schema/tables, /schema/table）
  - [x] 3.5 新增 Metrics 可观测性端点文档（GET /metrics/stats, /metrics/slow, /metrics/queries）
  - [x] 3.6 新增 MCP Server 使用说明（配置、启动、Claude Desktop 集成示例）
  - [x] 3.7 新增 Agent Onboarding CLI 文档（支持列表、安装示例、scope 说明）
  - [x] 3.8 更新 CLI 参考部分：补充 onboard 子命令

- [x] Task 4: 更新 rag/README.md 反映 Phase 7 测试状态
  - [x] 4.1 更新测试数量: 3 → 34（26 passed + 6 skipped async）
  - [x] 4.2 列出新增测试文件及其覆盖范围
  - [x] 4.3 补充 IngestionPipeline Saga 测试说明
  - [x] 4.4 补充 QueryEngine 集成测试说明

- [x] Task 5: 整理散落文件并更新构建配置
  - [x] 5.1 创建 `docs/archive/` 目录
  - [x] 5.2 移动 `P1-2-FS9-HEALTH-CHECK-IMPLEMENTATION.md` → `docs/archive/`
  - [x] 5.3 移动 `security_review_phase5.md` → `docs/archive/`
  - [x] 5.4 移动 `基本信息来源.txt` → `docs/archive/`
  - [x] 5.5 更新 Makefile: 新增 build-mcp, test-rag, build-all target，更新 help
  - [x] 5.6 更新 .gitignore: 新增 .omc/, __pycache__/, *.pyc 忽略规则

# Task Dependencies

- 所有任务互相独立，可并行执行 ✅
- [Task 1] 和 [Task 2] 内容有重叠（项目结构树），已保持一致 ✅

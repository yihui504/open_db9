# Checklist

## 文档更新验证

- [x] README.md 包含新的 Slogan ("The Open-Source, Self-Hosted 'Postgres for Agents'")
- [x] README.md 特性列表包含 RAG/MCP/Onboarding/HTTP扩展/匿名使用 5 项新增特性
- [x] README.md 快速开始包含 RAG Server 和 MCP Server 启动步骤
- [x] README.md 项目结构树包含 cmd/mcp-server/, internal/extensions/http/, internal/cli/cmd/, rag/
- [x] README.md CLI 示例包含 db9 onboard 命令
- [x] PROGRESS.md 包含 Phase 7 完整记录（7 个 Task 全部标记完成）
- [x] PROGRESS.md 测试数量已更新为 150+
- [x] PROGRESS.md 技术栈包含 Python/FastAPI/LangChain/mcp-go
- [x] PROGRESS.md 包含 Phase 8 路线图规划
- [x] docs/README.md 包含 RAG API 端点文档
- [x] docs/README.md 包含匿名认证端点文档（含请求/响应示例）
- [x] docs/README.md 包含 HTTP 扩展端点文档（含安全限制说明）
- [x] docs/README.md 包含 Schema 内省端点文档
- [x] docs/README.md 包含 Metrics 可观测性端点文档
- [x] docs/README.md 包含 MCP Server 使用说明
- [x] docs/README.md 包含 Agent Onboarding CLI 文档
- [x] rag/README.md 测试数量更新为 34（含新增文件清单）

## 文件整理验证

- [x] 根目录不再有 P1-2-FS9-HEALTH-CHECK-IMPLEMENTATION.md
- [x] 根目录不再有 security_review_phase5.md
- [x] 根目录不再有 基本信息来源.txt
- [x] docs/archive/ 目录存在且包含上述 3 个文件
- [x] Makefile 包含 build-mcp target
- [x] Makefile 包含 test-rag target
- [x] Makefile 包含 build-all target (cli+server+fs9+mcp)
- [x] Makefile help 输出包含新 target
- [x] .gitignore 包含 .omc/ 忽略规则
- [x] .gitignore 包含 __pycache__/ 忽略规则
- [x] .gitignore 包含 *.pyc 忽略规则

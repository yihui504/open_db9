# 项目文档更新与文件整理 Spec

## Why

Phase 7 已完成全部 7 个 Task（35/35 验证点通过），但项目文档（README.md、PROGRESS.md、docs/README.md）仍停留在 Phase 6 时代，未反映新增的 RAG 子系统、MCP Server、Agent Onboarding CLI、HTTP 扩展、匿名使用流程等能力。同时项目根目录存在散落的临时文件需要整理。

## What Changes

### 文档更新（3 个核心文件 + 1 个辅助文件）

- **README.md** — 重写为反映 Phase 7 完成状态的最新面貌，包含新定位叙事
- **PROGRESS.md** — 更新至 Phase 7 完成，补充所有新增模块和测试数据
- **docs/README.md** — 补充 RAG / MCP / 匿名认证 / HTTP 扩展等新端点文档
- **rag/README.md** — 补充 Phase 7 新增测试信息

### 文件整理

- 根目录散落文件归类或清理
- Makefile 补充新构建目标（mcp-server）
- .gitignore 补充忽略规则（.omc/, __pycache__/）

## Impact

- Affected specs: project-understanding（定位参考）, phase7-improvements（已完成记录）
- Affected code: README.md, PROGRESS.md, docs/README.md, rag/README.md, Makefile, .gitignore
- 不影响任何运行时代码

---

# ADDED Requirements

## Requirement: README.md 反映 Phase 7 完整状态

README SHALL 作为项目门面，准确反映 open-db9 当前全部能力。

#### 场景: 新访客 30 秒理解项目
- **WHEN** 开发者打开仓库根目录
- **THEN** README 展示: 定位口号(Slogan) → 架构图 → 核心特性(含 Phase 7 新增) → 快速开始 → 项目结构(含新目录)

具体新增内容:
1. **Slogan**: "The Open-Source, Self-Hosted 'Postgres for Agents' — db9.ai, but open & yours."
2. **新增特性列表**: RAG 智能检索 / MCP Server / Agent Onboarding / HTTP-from-SQL / 匿名使用
3. **新增启动步骤**: RAG API Server (:8001) 和 MCP Server 启动方式
4. **更新项目结构树**: 包含 cmd/mcp-server/, internal/extensions/http/, internal/cli/cmd/, rag/
5. **新增 CLI 命令示例**: `db9 onboard --agent claude`
6. **新增环境变量**: DB9_RAG_URL, DB9_MCP_*

## Requirement: PROGRESS.md 更新至 Phase 7

PROGRESS.md SHALL 记录完整开发历程，包含 Phase 7 所有交付物。

具体更新:
1. 新增 "Phase 7: MVP 完善" 章节，列出 7 个 Task 及其状态
2. 更新测试覆盖表（从 40+ → 150+）
3. 更新技术栈表（增加 Python 3.10+, FastAPI, LangChain, mcp-go）
4. 更新项目结构树（同 README）
5. 将原 "待实现的 Phase 7" 替换为已完成的实际内容
6. 新增 "Phase 8: 路线图" 作为未来规划

## Requirement: docs/README.md 补充新 API 文档

docs/README.md SHALL 包含所有可用 API 端点的完整索引。

具体补充:
1. 新增 "RAG API" 端点组（/rag/documents, /rag/query）
2. 新增 "匿名认证" 端点组（/auth/anonymous, /auth/claim）
3. 新增 "HTTP 扩展" 端点组（/http/get, /http/post）
4. 新增 "Schema 内省" 端点组（/schema/*）
5. 新增 "Metrics 可观测性" 端点组（/metrics/stats, /metrics/slow, /metrics/queries）
6. 新增 "MCP Server" 使用说明章节
7. 新增 "Agent Onboarding" CLI 使用说明
8. 更新 CLI 参考中新增 onboard 命令

## Requirement: rag/README.md 更新测试信息

rag/README.md SHALL 反映 Phase 7 测试补充结果。

具体更新:
1. 测试数量从 3 → 34（含 26 passed + 6 skipped async + 新增文件清单）
2. 列出新增测试文件: test_pipeline.py, test_query_engine.py, test_vectorstore_advanced.py, test_api_handlers.py

---

# MODIFIED Requirements

## Requirement: Makefile 补充新目标

Makefile SHALL 支持 Phase 7 新组件的构建和测试。

新增 target:
- `build-mcp`: 构建 mcp-server 二进制
- `test-rag`: 运行 RAG Python 测试
- `build-all`: 构建 cli + server + fs9 + mcp 四个组件
- help 目标更新

## Requirement: .gitignore 补充忽略规则

.gitignore SHALL 忽略工具生成的临时文件。

新增忽略:
- `.omc/` (OMC 工具状态目录)
- `__pycache__/` (Python 缓存)
- `*.pyc` (Python 编译文件)

## Requirement: 散落文件整理

根目录 SHALL 仅保留必要的项目元文件。

处理方案:
| 文件 | 操作 |
|------|------|
| `P1-2-FS9-HEALTH-CHECK-IMPLEMENTATION.md` | 移入 `docs/archive/` |
| `security_review_phase5.md` | 移入 `docs/archive/` |
| `基本信息来源.txt` | 移入 `docs/archive/` |
| `cmd/secrets-migrate/` | 保留（有效工具），在 docs 中备注 |

---

# REMOVED Requirements

(无)

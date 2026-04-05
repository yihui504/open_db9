# 项目展示方案 Spec

## Why

课题组需要直观了解 open-db9 项目的工作成果、技术价值、核心能力和应用场景。当前项目虽有 README 和 PROGRESS.md，但它们是面向开发者的技术文档，不适合作为学术汇报/课题展示材料。需要一套结构化的展示素材，能在 10-15 分钟内让听众理解"做了什么、为什么有价值、能做到什么、用在哪里"。

## What Changes

- **新建展示脚本**: `demo/demo.sh` — 一键端到端演示所有核心能力
- **新建演示文档**: `docs/showcase/SHOWCASE.md` — 课题汇报用主文档
- **新建架构图**: `docs/showcase/architecture.md` — ASCII + Mermaid 双格式架构图
- **新建对比分析**: `docs/showcase/comparison.md` — 与 db9.ai/Supabase/Neon 的竞品对标
- **新建应用场景**: `docs/showcase/use-cases.md` — 4-5 个具体应用场景深度解析
- **新建性能数据**: `docs/showcase/performance.md` — 测试覆盖、安全评级、性能指标

## Impact

- Affected specs: project-understanding（定位参考）, phase7-improvements（成果来源）
- Affected code: 纯新增文件，不修改任何现有代码
- 新增目录: demo/, docs/showcase/

---

# ADDED Requirements

## Requirement: 一键演示脚本 (demo/demo.sh)

系统 SHALL 提供一个可执行的演示脚本，按顺序自动展示项目的全部核心能力。

#### 场景: 课题组成员运行演示
- **WHEN** 执行者运行 `bash demo/demo.sh`
- **THEN** 脚本依次执行以下演示步骤，每步输出清晰的标题和结果：

1. **环境检查**: 检查 Docker/Go/Python 是否就绪
2. **服务启动**: 按序启动 postgres → fs9-service → server → rag-api (Docker Compose)
3. **CLI 创建数据库**: `db9 create --name demo-db` 展示即时配置
4. **SQL 执行**: 创建表、插入数据、查询，展示 SQL 能力
5. **Schema 内省**: 调用 Schema API 展示表结构查询
6. **Metrics 可观测性**: 调用 Metrics API 展示数据库统计
7. **文件操作**: 上传 CSV、下载、列表，展示 FS9 文件系统
8. **快照与分支**: 创建快照、创建分支、列出分支
9. **匿名使用**: 匿名注册 → 建库 → 展示匿名流程
10. **RAG 演示**: 上传 PDF → 索引 → 自然语言问答
11. **HTTP 扩展**: 从 SQL 调用外部 API
12. **Agent 集成**: 展示 MCP Server 工具列表
13. **性能汇总**: 输出测试覆盖率、安全评级、功能完成度

每个步骤之间暂停 2 秒，输出带 emoji 的进度标记。

## Requirement: 课题展示主文档 (docs/showcase/SHOWCASE.md)

系统 SHALL 提供一份适合学术汇报的主文档，结构清晰、图文并茂。

#### 文档结构要求:

```
# Open-DB9 项目展示

## 一、项目概述 (1 页)
- 一句话定位 + Slogan
- 核心问题: db9.ai 闭源不可自托管 → 开源替代的必要性
- 目标用户: AI Agent 开发者 / 需要数据主权的企业 / 研究机构

## 二、我们做了什么 (2-3 页)
### Phase 1-6 基础设施 (已完成)
- 多租户 PostgreSQL 管理 (Go + pgx)
- 数据库快照与分支 (pg_dump + COW 设计)
- FS9 文件存储服务 (独立 Go 服务 :9090)
- REST API + CLI 工具 (Gin + Cobra)
- JWT 认证 + A 级安全防护 (36 个安全修复)

### Phase 7 MVP 完善 (本次重点)
- Schema/Metrics API 从 501 → 完整可用
- 匿名使用全流程打通
- RAG 子系统测试 3 → 34 个
- HTTP-from-SQL 扩展 (SSRF 防护)
- Agent Onboarding CLI (5 种 Agent)
- MCP Server (11 个工具)

## 三、技术架构 (1-2 页)
[引用 architecture.md]

## 四、核心能力演示 (3-4 页)
[每项配截图/命令输出示例]
4.1 即时创建数据库
4.2 SQL 执行与 Schema 内省
4.3 文件系统操作
4.4 快照与分支
4.5 RAG 智能检索
4.6 HTTP-from-SQL
4.7 Agent 集成 (MCP)

## 五、竞品对比 (1 页)
[引用 comparison.md]

## 六、应用场景 (2-3 页)
[引用 use-cases.md]

## 七、工程品质 (1 页)
- 测试: ~160 个测试用例, ~89% 功能覆盖率
- 安全: A 级评级, 36 个安全修复
- 文档: 完整 API 文档 + 架构文档
- 部署: Docker Compose 一键启动

## 八、后续规划 (0.5 页)
[Phase 8 路线图]
```

## Requirement: 架构图 (docs/showcase/architecture.md)

系统 SHALL 提供双格式架构图：ASCII (终端可读) + Mermaid (可渲染)。

包含以下视图:
1. **系统全景图**: 四层架构 (接入层→业务逻辑层→数据层→存储层)
2. **数据流图**: 典型请求从 CLI/API 到 PostgreSQL 的完整路径
3. **RAG 子系统图**: 文档摄入→分块→向量存储→检索→LLM 生成
4. **多租户隔离图**: tenants → databases → connections 的层级关系

## Requirement: 竞品对比分析 (docs/showcase/comparison.md)

系统 SHALL 提供与主要竞品的客观对比分析。

对比维度:
| 维度 | Open-DB9 | db9.ai | Supabase | Neon | Pigsty |
|------|----------|--------|----------|------|--------|
| 开源 | ✅ MIT | ❌ | ✅ | ⚠️ 部分 | ✅ |
| 自托管 | ✅ | ❌ | ✅ | ❌ | ✅ |
| AI/Agent 友好 | ✅ RAG+MCP+Onboard | ✅ 原生 | 一般 | 加速中 | ❌ |
| 向量搜索 | ✅ pgvector | ✅ 内置 | 需装扩展 | ✅ | ✅ |
| 文件系统 | ✅ FS9 | ✅ fs9 | ✅ Storage | ❌ | ⚠️ |
| HTTP from SQL | ✅ Phase7 | ✅ | ❌ | ❌ | ❌ |
| 数据库分支 | ✅ 快照模式 | ✅ 零拷贝 | ❌ | ✅ | ❌ |
| 匿名使用 | ✅ Phase7 | ✅ | ❌ | ❌ | ❌ |
| Agent 技能 | ✅ 5种Agent | ✅ | ❌ | ❌ | ❌ |
| 部署复杂度 | 低(4服务) | N/A | 高(10+服务) | 中 | 中(Ansible) |

## Requirement: 应用场景深度解析 (docs/showcase/use-cases.md)

系统 SHALL 提供 4-5 个具体的应用场景，每个场景包含: 背景→问题→解决方案→效果。

### 场景 1: AI 研究助手 (RAG 核心)
- **背景**: 研究团队有大量 PDF 论文和技术文档，需要快速检索和问答
- **方案**: 上传文档到 open-db9 → 自动分块+向量化 → 自然语言提问
- **价值**: 无需外部向量数据库，PostgreSQL 统一管理数据和知识

### 场景 2: Agent 记忆系统 (MCP 核心)
- **背景**: Claude/Cursor 等 AI Agent 需要持久化记忆和状态
- **方案**: 通过 MCP 协议连接 open-db9 → Agent 自主读写数据库
- **价值**: 一个数据库同时存结构化状态 (tables) 和非结构化上下文 (files)

### 场景 3: 敏感数据自托管平台 (安全核心)
- **背景**: 企业/研究机构无法将数据放在云端 (合规/隐私)
- **方案**: Docker Compose 一键部署 open-db9 → 数据完全本地
- **价值**: 对标 db9.ai 的能力但数据不出内网

### 场景 4: 多实验并行 (分支核心)
- **背景**: 研究人员需要在同一数据集上尝试不同分析方法
- **方案**: 主库 → 分支A (方法1) / 分支B (方法2) → 对比结果
- **价值**: 类似 Git 的数据库版本控制

### 场景 5: 快速原型开发 (匿名核心)
- **背景**: 学生/研究者想快速试用，不想注册配置
- **方案**: 打开终端 → 匿名建库 → 5 分钟出结果
- **价值**: 零摩擦入门

## Requirement: 工程品质报告 (docs/showcase/performance.md)

系统 SHALL 提供量化的工程质量数据。

内容:
- **测试矩阵**: 按模块列出的测试数量和覆盖率
- **安全审计**: A 级评级的详细说明 (36 个修复的分类)
- **功能完成度**: vs db9.ai 的 16/18 核心能力对照
- **性能指标**: 连接池参数、API 响应时间目标、并发支持
- **代码统计**: Go 代码行数 / Python 代码行数 / 文件数 / 模块数

---

# MODIFIED Requirements

(无)

# REMOVED Requirements

(无)

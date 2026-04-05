# Open-DB9 项目全面理解与定位 Spec

## Why

用户要求通读项目并结合对原版 db9 商业产品的深入调研，确认开源项目 open-db9 的最佳市场定位。

## What Changes

* 本 Spec 为**纯分析性文档**，不涉及代码变更
* 核心新增：原版 db9.ai 产品调研、竞品格局分析、能力差距对比、定位方案推荐

## Impact

* 涉及范围：整个代码库（Go 后端 + Python RAG 子系统）
* 不影响任何现有代码或功能
* 输出将决定项目后续发展方向

***

# Open-DB9 项目完整理解与战略定位报告

## 一、原版 db9.ai 是什么？

### 1.1 产品身份

| 属性 | 详情 |
|------|------|
| **产品名** | [db9.ai](https://db9.ai) |
| **Slogan** | "Postgres but for agents" — 为 AI Agent 设计的 Serverless PostgreSQL |
| **公司** | Tiger Data |
| **技术栈** | Rust (Tokio) + TiKV (分布式 KV 存储) + 自研 SQL 引擎 |
| **开源状态** | ❌ **闭源** |
| **自托管** | ❌ **不支持** (仅云服务) |
| **核心创新** | 将 Agent 所需的能力（向量、文件、HTTP、Cron）**编译进数据库** |

### 1.2 核心功能矩阵

```
┌─────────────────────────────────────────────────────────────┐
│                   db9.ai 功能全景                           │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  🧠 内置 AI 能力                                            │
│  ├── embedding() SQL 函数 — 服务端生成向量，无需外部 API      │
│  ├── 向量搜索 — pgvector 风格的 <-> 操作符，HNSW 索引        │
│  └── 每租户并发限制 (默认 5)                                  │
│                                                             │
│  📁 fs9 — SQL 层文件系统                                     │
│  ├── SELECT * FROM fs9('/data.csv') — CSV/JSONL/Parquet 挂载  │
│  ├── SELECT fs9_write('/out.json', '...') — 写入文件         │
│  └── 单文件 100MB, 单次读取预算 128MB                         │
│                                                             │
│  🌐 HTTP from SQL                                            │
│  ├── http_get() / http_post() — 从 SQL 调外部 API            │
│  ├── 仅 HTTPS, SSRF 防护 (阻止私有/回环 IP)                  │
│  └── 每语句 100 请求, 响应最大 1MB                            │
│                                                             │
│  🌿 数据库分支                                                │
│  ├── 零拷贝分支 (TiKV COW 特性)                               │
│  ├── 包含: 表数据 + 文件 + Cron + 权限 + 扩展                 │
│  └── 用途: 预览环境 / Schema 实验 / 回滚点                    │
│                                                             │
│  ⏰ 定时任务 (pg_cron)                                       │
│  └── 分布式调度, 无空闲超时                                    │
│                                                             │
│  🤖 Agent Onboarding                                         │
│  ├── db9 onboard --agent claude                              │
│  ├── 支持 Claude Code / Codex / Cursor / Cline 等            │
│  └── 教会 Agent 自主安装、认证、使用 DB9                       │
│                                                             │
│  ⚡ 即时配置                                                 │
│  ├── 匿名使用, 无需注册                                       │
│  ├── 匿名账户可创建 5 个数据库                                │
│  └── SSO 认领后解除限制                                       │
│                                                             │
│  🔐 多租户隔离                                               │
│  ├── Keyspace 级隔离 (TiKV 原生)                             │
│  ├── 用户名格式: tenant_id.role                               │
│  └── 内存配额 / 连接数配额                                    │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### 1.3 架构设计哲学 (6 大原则)

1. **"能力内置，而非外挂"** — Embeddings/文件/HTTP/Cron 全部在 SQL 层，减少服务间调用
2. **"标准协议优先"** — 完全兼容 PostgreSQL wire protocol，所有 ORM 开箱即用
3. **"匿名优先，渐进式认证"** — 先体验后注册，降低初始摩擦
4. **"分支即原语"** — 分支不是附加功能，而是核心设计 (TiKV COW)
5. **"SQL 作为编排层"** — SQL 不只是查询语言，也是工作流编排语言
6. **"Agent 技能安装"** — `db9 onboard` 让 AI Agent 学会使用数据库

### 1.4 竞争定位 (来自官方)

| 特性 | db9.ai | Neon | Supabase |
|------|--------|------|----------|
| 目标用户 | **AI Agent** | 开发者 | 全栈开发者 |
| 即时配置 | ✅ 秒级 | ✅ | ✅ |
| **内置 Embeddings** | ✅ | ✗ | ✗ |
| **内置文件系统** | ✅ | ✗ | ✗ (Storage) |
| **HTTP from SQL** | ✅ | ✗ | ✗ |
| **数据库分支** | ✅ | ✅ | ✗ |
| **内置 Cron** | ✅ | ✗ | ✗ |
| **匿名使用** | ✅ | ✗ | ✗ |
| **Agent 技能** | ✅ | ✗ | ✗ |
| **自托管** | ✗ | ✗ | ✅ |

---

## 二、开源竞品格局分析

### 2.1 竞品分类地图

```
                        数据库管理平台
                    ┌───────────────────────┐
                    │                       │
              全栈 BaaS               纯数据库平台
           ┌──────────────┐        ┌──────────────────┐
           │              │        │                  │
       云托管          自托管    云托管 (Serverless)  自托管
    ┌─────────┐   ┌──────────┐  ┌──────┐  ┌───────┐  ┌──────────┐
    │Supabase │   │Supabase  │  │Neon  │  │db9.ai │  │??? 空缺 │
    │ Cloud   │   │ Self-Host│  │Cloud │  │(闭源)  │  │ ← 我们!  │
    └─────────┘   └──────────┘  └──────┘  └───────┘  └──────────┘
         ▲              ▲           ▲         ▲            ▲
         │              │           │         │            │
     94K stars     Docker/K8s   分支技术   Postgres+AI   ???
     Auth+Storage  复杂部署    pgvector   Rust+TiKV    Go+PG
```

### 2.2 关键发现：市场空缺

**⚠️ 核心发现：不存在 "db9.ai 的开源替代品"！**

| 维度 | Supabase (自托管) | Neon (开源部分) | Pigsty | **open-db9 (我们)** |
|------|-------------------|-----------------|--------|---------------------|
| 是否开源 | ✅ MIT | ⚠️ 部分 | ✅ | ✅ |
| 是否可自托管 | ✅ | ⚠️ 有限 | ✅ | ✅ |
| 目标用户 | 全栈开发者 | 开发者 | DBA/运维 | **AI Agent 开发者** |
| 内置向量搜索 | 需自行装扩展 | ✅ pgvector | ✅ pgvector | **✅ RAG 完整流水线** |
| 文件存储 | ✅ Storage | ✗ | ✗ (MinIO模块) | **✅ FS9 服务** |
| 数据库分支 | ✗ | ✅ (核心特性) | ✗ | **✅ 快照分支** |
| HTTP from SQL | ✗ | ✗ | ✗ | 📋 待实现 |
| 内置 Cron | ✗ (需Edge Func) | ✗ | ✗ | 📋 待实现 |
| Agent 友好 | 一般 | 正在加 | 否 | **✅ RAG + CLI** |
| 部署复杂度 | 高 (10+ 服务) | 中 | 中 (Ansible) | **低 (Docker Compose)** |
| 技术栈 | TypeScript | Rust/TypeScript | Shell/Ansible | **Go + Python** |

---

## 三、能力差距分析：open-db9 vs db9.ai

### 3.1 功能覆盖对比

| db9.ai 原版功能 | open-db9 当前状态 | 差距评估 |
|-----------------|-------------------|----------|
| **即时配置 (秒级建库)** | ✅ Manager.Create + API | **已具备** |
| **多租户隔离** | ✅ tenants 表 + RLS | **已具备** |
| **SQL 执行** | ✅ ExecuteSQLHandler | **已具备** |
| **数据库分支** | ✅ CreateBranch (快照模式) | **已具备** (非零拷贝) |
| **快照管理** | ✅ SnapshotManager (pg_dump) | **已具备** |
| **文件系统 (fs9)** | ✅ FS9 Service + API/CLI | **已具备** |
| **类型生成** | ✅ TS/Python generator | **已具备** |
| **CLI 工具** | ✅ Cobra CLI | **已具备** |
| **REST API** | ✅ Gin Server | **已具备** |
| **JWT 认证** | ✅ golang-jwt | **已具备** |
| **内置 Embeddings** | ⚠️ 通过 RAG 子系统间接支持 | **部分具备** (非 SQL 函数) |
| **向量搜索** | ✅ pgvector + LangChain VectorStore | **已具备** |
| **RAG 流水线** | ✅ IngestionPipeline + QueryEngine | **已具备** |
| **HTTP from SQL** | ❌ 未实现 | **❌ 缺失** |
| **Cron 定时任务** | ❌ 未实现 | **❌ 缺失** |
| **匿名使用** | ⚠️ anonymous_accounts 表已有 | **骨架具备** |
| **Agent Onboarding** | ❌ 未实现 | **❌ 缺失** |
| **pgwire 协议** | ❌ 使用 REST API 代替 | **架构差异** |
| **零拷贝分支** | ❌ 基于 pg_dump 的物理拷贝 | **性能差距** |
| **Schema 内省 API** | ⚠️ 路由已注册但返回 501 | **🔧 未完成** |
| **Metrics/可观测性 API** | ⚠️ 路由已注册但返回 501 | **🔧 未完成** |

### 3.2 架构差异总结

```
┌──────────────────────────────────────────────────────────────┐
│                    架构对比                                   │
├──────────────────────┬───────────────────────────────────────┤
│      db9.ai (原版)   │        open-db9 (我们)                │
├──────────────────────┼───────────────────────────────────────┤
│ 语言: Rust           │ 语言: Go 1.25 + Python 3.10           │
│ 存储: TiKV (分布式)  │ 存储: PostgreSQL (单节点)             │
│ 协议: pgwire (5433) │ 协议: REST API (8080)                 │
│ 分支: 零拷贝 COW     │ 分支: pg_dump 物理拷贝                │
│ 部署: 云端 SaaS      │ 部署: Docker Compose 自托管           │
│ 许可: 闭源商业       │ 许可: MIT 开源                         │
│ AI: SQL 内嵌         │ AI: RAG 子系统 (LangChain)             │
│ 复杂度: 极高         │ 复杂度: 中等                          │
└──────────────────────┴───────────────────────────────────────┘
```

### 3.3 我们的优势 (vs 原版)

1. **✅ 开源自由**: MIT 许可，无 vendor lock-in
2. **✅ 可自托管**: 数据完全自主可控，适合企业/合规场景
3. **✅ 部署简单**: `docker-compose up -d` vs 无法自托管
4. **✅ 标准 PostgreSQL**: 直接用原生 PG，生态兼容性最强
5. **✅ RAG 完整流水线**: Saga 事务 + LangChain + 双 LLM
6. **✅ 安全评级 A**: 36 个安全修复，纵深防御
7. **✅ Go 生态**: 比 Rust 更广泛的开发者基础

### 3.4 我们的劣势 (vs 原版)

1. **❌ 无零拷贝分支**: pg_dump 方式慢且重
2. **❌ 无 pgwire 协议**: 无法用标准 PG 客户端直连
3. **❌ 无 SQL 内嵌函数**: embedding()/http_get()/fs9() 不是 SQL 函数
4. **❌ 无 Cron**: 缺少定时任务调度
5. **❌ 性能差距**: 单节点 PG vs 分布式 TiKV
6. **❌ 匿名流程未打通**: 表结构有但流程未完成

---

## 四、定位方案推荐

### 方案 A：🎯 "db9.ai 的开源自托管替代品" （强烈推荐）

**一句话定位**: **The Open-Source, Self-Hosted "Postgres for Agents"**

```
Tagline: "db9.ai, but open & yours."
```

**目标用户**:
- 需要 AI Agent 数据库但不想被云厂商 lock-in 的团队
- 有数据主权/合规要求的企业 (金融、医疗、政府)
- 想要学习 "Postgres for Agents" 架构的开发者
- 自托管爱好者 / 开源社区

**核心叙事**:
> db9.ai 开创了 "Postgres for Agents" 品类，但它闭源且不可自托管。
> **Open-DB9 是第一个开源、可自托管的 "Postgres for Agents" 实现。**
> 我们复刻了 db9.ai 的核心理念 — 数据库分支、文件系统、向量搜索、RAG —
> 但用 Go + PostgreSQL 构建，让你完全拥有自己的数据。

**差异化壁垒**:
- **唯一**同时满足: 开源 + 自托管 + AI-Agent-friendly + Postgres-native
- 社区驱动: 贡献者可以参与核心开发 (不像闭源的 db9.ai)
- 渐进式兼容: 可以与现有 Supabase/Neon 部署共存

**发展路径**:
```
Phase 7 (当前): 补齐短板 → Schema/Metrics API + HTTP-from-SQL 扩展
Phase 8:        Agent 集成   → MCP Server + onboard 命令 + Claude/Cursor 技能包
Phase 9:        性能优化    → 连接池优化 + pgwire 可选层 + 分支加速
Phase 10:       生态扩展    → 插件系统 + 多 LLM 后端 + 更多 Parser
```

---

### 方案 B：📦 "带 AI 的轻量级 Supabase"

**一句话定位**: **Self-Hosted Postgres Platform with Built-in AI/RAG**

```
Tagline: "Like Supabase, but with RAG built in. 3 services, not 12."
```

**目标用户**:
- 觉得 Supabase 自托管太重的开发者
- 需要在应用中集成 RAG/AI 但不想维护独立向量库的团队
- 想要一个 "开箱即用的 AI 后端" 的全栈开发者

**优缺点**:
- ✅ 更广泛的受众 (不止 Agent 场景)
- ✅ 部署简单是强卖点
- ❌ 与 Supabase 直接竞争，差异化不够尖锐
- ❌ 可能被视为 "Supabase 的阉割版"

---

### 方案 C：🔧 "PostgreSQL AI 编排层"

**一句话定位**: **An AI-Native Orchestration Layer for PostgreSQL**

```
Tagline: "Supercharge your Postgres with AI: Vectors, RAG, Files, Branching."
```

**定位**: 不做完整平台，而是作为现有 PostgreSQL 部署的**增强层/扩展包**

**优缺点**:
- ✅ 安装门槛极低 (attach 到现有 PG)
- ✅ 与 Pigsty/云 RDS 互补而非替代
- ❌ 身份模糊 — 是工具还是平台？
- ❌ 价值感知弱于完整解决方案

---

### 最终推荐：方案 A

**选择理由**:

1. **蓝海定位**: "db9.ai 开源替代" 是当前**无人占据**的市场位置
2. **叙事有力**: "Open & Yours" 直击 db9.ai 最大痛点 (闭源+不可自托管)
3. **与现状匹配**: 我们已有的代码 (分支+FS9+RAG+多租户) 天然对齐这个定位
4. **天花板高**: AI Agent 是 2025-2026 最热赛道之一
5. **社区吸引力**: 开源社区喜欢 "把闭源产品开源化" 的故事

**Slogan 候选**:
- `"db9.ai, but open & yours."` (直接对标)
- `"The Open-Source Postgres Platform for AI Agents"` (描述式)
- `"Self-Hosted Postgres + RAG + Branching. Zero vendor lock-in."` (利益驱动)

---

## 五、行动建议 (按优先级排序)

### P0 — 立即补齐 (定义核心体验)

1. **完成 Metrics/Schema API** — 路由已注册，只需实现 Handler 逻辑
2. **完成匿名使用流程** — anonymous_accounts 表已有，打通注册→建库→SSO 认领
3. **RAG 子系统完善** — 补齐 e2e/integration 测试

### P1 — 差异化功能 (拉开与 Supabase 距离)

4. **HTTP-from-SQL 扩展** — 实现 `http_get()`/`http_post()` SQL 函数或 UDF
5. **Agent Onboarding** — `db9 onboard --agent <name>` 命令
6. **MCP Server** — 让 AI Agent 通过 MCP 协议操作 open-db9
7. **内置 Embedding SQL 函数** — `embedding('text')::vector`

### P2 — 性能与体验 (追赶 db9.ai 核心体验)

8. **pgwire 兼容层** (可选) — 让标准 PG 客户端直连
9. **分支加速** — 从 pg_dump 迁移到更快的方式 (如 WAL-level 复制)
10. **Cron 任务调度** — pg_cron 或自研调度器

### P3 — 生态建设 (长期)

11. **插件/扩展系统** — 第三方扩展接入
12. **多存储后端** — S3/GCS/MinIO 支持
13. **Kubernetes Helm Chart** — 一键 K8s 部署
14. **SDK** — TypeScript/Python/Go SDK (`get-db9` 开源版)

---

## 六、附录：项目技术全貌 (保留原有理解)

*(以下章节保持不变，详见前版: 系统架构总览、三大组件、核心模块、RAG子系统、数据库Schema、安全体系、可观测性、项目成熟度、部署架构、设计亮点)*

### 关键架构图

```
┌─────────────────────────────────────────────────────────────┐
│  Open-DB9: The Open-Source "Postgres for Agents"           │
│                                                             │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐    │
│  │  CLI     │  │ REST API │  │ FS9 Svc  │  │ RAG API  │    │
│  │  (Go)    │  │  (Go)    │  │  (Go)    │  │ (Python) │    │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘    │
│       └──────────────┼──────────────┼──────────────┘         │
│                      ▼              ▼                        │
│              ┌──────────────────────────┐                    │
│              │   PostgreSQL + pgvector  │                    │
│              │  Multi-tenant + RLS      │                    │
│              └──────────────────────────┘                    │
└─────────────────────────────────────────────────────────────┘
```

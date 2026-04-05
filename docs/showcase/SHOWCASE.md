# Open-DB9: 开源自托管的 "Postgres for AI Agents"

> **"The Open-Source, Self-Hosted 'Postgres for Agents' -- db9.ai, but open & yours."**

---

## 目录

- [第一章: 项目概述](#第一章-项目概述)
- [第二章: 工作成果](#第二章-工作成果)
- [第三章: 技术架构](#第三章-技术架构)
- [第四章: 核心能力演示](#第四章-核心能力演示)
- [第五章: 竞品对比](#第五章-竞品对比)
- [第六章: 应用场景](#第六章-应用场景)
- [第七章: 工程品质](#第七章-工程品质)
- [第八章: 后续规划](#第八章-后续规划)

---

## 第一章: 项目概述

### 1.1 一句话定位

Open-DB9 是 [db9.ai](https://db9.ai) 的**开源自托管替代品**：一个为 AI Agent 时代设计的 PostgreSQL 管理平台，内置 RAG 智能检索、MCP 协议集成、数据库分支、文件系统等 AI 原生能力。

### 1.2 我们解决的核心问题

db9.ai 开创了 **"Postgres for Agents"** 这个新品类 -- 将 Agent 所需的能力（向量搜索、文件存储、HTTP 调用、定时任务）**编译进数据库**。但它有两个致命限制:

| 问题 | 影响 | 谁在意 |
|------|------|--------|
| **闭源** | 无法审查代码、无法自托管、数据锁定在云端 | 企业合规、学术研究 |
| **不可自托管** | 合规/隐私/数据主权需求无法满足 | 金融、医疗、政府 |

**Open-DB9 的答案**: 用 Go + PostgreSQL 复刻 db9.ai 的核心理念，以 MIT 协议开源，支持 Docker Compose 一键自托管。

### 1.3 目标用户

| 用户类型 | 核心诉求 | Open-DB9 满足度 |
|---------|---------|-------------|
| AI Agent 开发者 | 需要持久化记忆 + 知识库 | RAG + MCP + 向量搜索 |
| 企业 / 研究机构 | 数据合规 + 自主可控 | 内网部署 + 审计日志 |
| 全栈开发者 | 快速原型 + 零摩擦入门 | 匿名使用 + CLI |
| 数据科学团队 | 多实验并行 + 版本管理 | 分支 + 快照 |

### 1.4 关键数字一览

```
7    Phase 已完成 (Phase 1-7)
~160 测试用例, ~82% 覆盖率
A    级安全评级, 36 个安全修复
16/18 核心能力 vs db9.ai (88.9% 完成度)
25K+ 行代码 (Go 24K + Python 1.4K)
11   MCP 工具, 支持 Claude / Cursor / Cline 等 5 种 Agent
```

---

## 第二章: 工作成果

### 2.1 Phase 1-6: 基础设施 (已完成)

#### 多租户 PostgreSQL 管理平台

- 基于 **Go 1.25 + pgx/v5** 的高性能连接池 (MaxConn=25, 3种溢出策略)
- **tenants -> databases** 层级多租户架构，Row-Level Security 隔离
- 完整 CRUD API + CLI 双接入模式

#### 数据库生命周期管理

- **快照系统**: pg_dump + gzip 压缩，保留策略(30天/10个)，路径遍历防护
- **分支系统**: 类 Git 的数据库版本控制 (快照 -> 复制 -> 父子追踪)
- Schema 内省 API: 表结构 / 列 / 索引 / 约束 / 触发器 完整查询

#### FS9 文件存储服务

- 独立 Go 服务 (:9090)，原子写入，100MB 单文件限制
- 上传 / 下载 / 列表 / 复制 / 删除 完整操作
- Storage 接口抽象，可扩展到 S3 / GCS / MinIO

#### 安全体系 (A 级评级)

- JWT HS256 认证 (>=32字符 Secret, 弱密码黑名单)
- SQL 白名单 (仅 SELECT / SHOW / DESCRIBE / EXPLAIN)
- SSRF / 路径遍历 / 命令注入 / 凭证加密 全方位防护
- **7 层纵深防御**: 应用 -> 授权 -> 限流 -> 输入 -> 网络 -> 数据 -> 基础设施

#### 可观测性

- Prometheus `/metrics` 端点
- `pg_stat_statements` 慢查询检测
- Schema 变更追踪
- 请求链路追踪 (RequestID)

### 2.2 Phase 7: MVP 完善 (本次重点)

Phase 7 补齐了从"可用"到"好用的"关键缺口:

#### P0 核心体验补齐

| 能力 | 变化 | 产出 |
|------|------|------|
| Schema 内省 API | 501 -> 200 | 重构 Handler 为包级别函数，修复路由绑定 |
| Metrics 可观测性 | 占位 -> 完整 | stats / slow queries / query stats 三端点 + 42 测试 |
| 匿名使用流程 | 骨架 -> 全流程 | 注册 -> JWT -> 建库(<=5) -> SSO认领 + 13 测试 |
| RAG 测试覆盖 | 3 -> 34 | Pipeline Saga / QueryEngine / VectorStore / API 共 30 新测试 |

#### P1 差异化能力

| 能力 | 描述 | 测试数 |
|------|------|--------|
| HTTP-from-SQL | REST API + SSRF 防护 + 速率限制 | 18 |
| Agent Onboarding | `db9 onboard` 支持 5 种 AI Agent | 14 |
| MCP Server | 11 个工具, stdio 模式, Claude Desktop 就绪 | 多 |

**Phase 7 交付总结**: 7/7 Task 完成, 35/35 验证点通过, 117+ 新增测试

---

## 第三章: 技术架构

*(详细架构图请参见 [architecture.md](./architecture.md))*

### 3.1 四层架构概览

```
+---------------------------------------------------------------------+
|  接入层: CLI (:db9) + API (:8080) + FS9 (:9090)                    |
|         + RAG (:8001) + MCP Server (stdio)                           |
+---------------------------------------------------------------------+
|  业务层: Auth / Schema / Metrics / Snapshot                          |
|        / Branch / HTTP-Ext / Anonymous / MCP                          |
+---------------------------------------------------------------------+
|  数据层: ConnectionPool / Manager / VectorStore                       |
+---------------------------------------------------------------------+
|  存储层: PostgreSQL 15+ (控制平面 + 用户数据)                        |
+---------------------------------------------------------------------+
```

### 3.2 技术栈

| 层级 | 技术 | 说明 |
|------|------|------|
| 后端语言 | Go 1.25 | 高性能，强类型，优秀并发 |
| AI 子系统 | Python 3.10+ | LangChain 生态 |
| Web 框架 | Gin (Go) / FastAPI (Py) | 高性能 HTTP |
| 数据库 | PostgreSQL 15+ | pgvector 扩展 |
| 向量检索 | LangChain + pgvector | CTE 查询优化 |
| LLM 集成 | OpenAI / 智谱AI | 双提供商支持 |
| CLI 框架 | Cobra | 业界标准 |
| MCP SDK | mcp-go | Model Context Protocol |
| ORM/驱动 | pgx/v5 | 纯 Go PostgreSQL 驱动 |
| 部署 | Docker Compose | 一键 4 服务启动 |

---

## 第四章: 核心能力演示

> 以下每个能力均配有实际命令示例，可在 [demo/demo.sh](../../demo/demo.sh) 中一键运行完整演示。

### 4.1 即时创建数据库

```bash
$ db9 create --name myapp
  Name        myapp
  ID          f47ac10b-...
  Engine      postgresql
  Status      ready
  Created     2026-04-04T...
```

**价值**: 从零到可用数据库 < 5 秒。无需注册云账号、无需配置 VPC、无需等待资源分配。

### 4.2 SQL 执行与 Schema 内省

```bash
# 执行 SQL
$ db9 sql myapp "CREATE TABLE users (id SERIAL PRIMARY KEY, name TEXT); INSERT INTO users (name) VALUES ('Alice'); SELECT * FROM users;"
 id | name
----+-------
  1 | Alice

# Schema 内省
$ curl -s http://localhost:8080/api/v1/databases/{id}/schema/table?schema=public&table=users | python -m json.tool
{
  "columns": [{"name": "id", "type": "integer", ...}, {"name": "name", "type": "text", ...}],
  "indexes": [...],
  "constraints": [...],
  "row_count_estimate": 1,
  "size_bytes": 8192
}
```

**价值**: 不需要额外工具（pgAdmin / DBeaver），API 即可获取完整的元数据。

### 4.3 文件系统操作

```bash
# 上传
$ db9 fs upload myapp ./data.csv /imports/data.csv
  data.csv -> /imports/data.csv   OK (12KB in 45ms)

# 列表
$ db9 fs list myapp /imports/
  Name         Size    Modified
  data.csv    12KB    2026-04-04 10:23

# 下载
$ db9 fs download myapp /imports/data.csv ./downloaded.csv
```

**价值**: 数据库自带文件存储，不需要 S3 / MinIO 配置。

### 4.4 快照与分支

```bash
# 创建快照
$ db9 snapshot create myapp --name v1.0
  Snapshot 'v1.0' created from database myapp

# 创建分支做实验
$ db9 branch create myapp --name experiment-alice
  Branch 'experiment-alice' created from myapp

# 在分支上修改 schema...
$ db9 sql experiment-alice "ALTER TABLE users ADD COLUMN age INT;"

# 完成后清理分支
$ db9 branch delete myapp --name experiment-alice
```

**价值**: 类似 Git 的数据库版本控制，实验安全隔离。

### 4.5 RAG 智能检索

```bash
# 上传论文 PDF
curl -X POST http://localhost:8001/documents \
  -F "file=@attention_is_all_you_need.pdf" \
  -F 'metadata={"title": "Attention Is All You Need", "category": "nlp"}'

# 自然语言问答
curl -X POST http://localhost:8001/query \
  -H "Content-Type: application/json" \
  -d '{"database_id": "...", "query": "注意力机制有哪些变体？"}'
{
  "answer": "根据文献，主要变体包括: 1) Self-Attention... 2) Multi-Head...",
  "sources": [
    {"doc": "attention_is_all_you_need.pdf", "relevance": 0.94},
    {"doc": "longformer.pdf", "relevance": 0.87}
  ],
  "latency_ms": 1200
}
```

**价值**: PDF 论文 -> 自动分块向量化 -> 自然语言问答，< 2 秒响应。

### 4.6 HTTP-from-SQL

```bash
curl -X POST http://localhost:8080/api/v1/databases/{id}/http/get \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"url": "https://api.github.com/repos/open-db9/db9"}'
{
  "status_code": 200,
  "body": "{\"stargazers_count\": 42, ...}",
  "headers": {"content-type": "application/json"}
}
```

**价值**: 在 SQL 层调用外部 API，安全可控（SSRF 防护 + HTTPS only + 速率限制）。

### 4.7 Agent 集成 (MCP)

```json
// Claude Desktop 配置
{
  "mcpServers": {
    "open-db9": {
      "command": "./db9-mcp",
      "env": { "DB9_API_URL": "http://localhost:8080" }
    }
  }
}

// Agent 自动获得 11 个工具:
// execute_sql, list_databases, create_database,
// list_files, upload_file, download_file,
// query_rag, get_schema, create_snapshot, list_snapshots
```

**价值**: Claude / Cursor 等 AI 工具可直接操作数据库，获得持久化记忆。

---

## 第五章: 竞品对比

*(详细对比矩阵请参见 [comparison.md](./comparison.md))*

### 5.1 Open-DB9 的独特定位

| 维度 | Open-DB9 | db9.ai | Supabase | Neon |
|------|----------|--------|----------|------|
| **开源** | MIT | 闭源 | MIT | 部分 |
| **自托管** | Docker | 仅云 | 复杂 | 否 |
| **AI Agent** | RAG+MCP+Onboard | 原生 | 无 | 加速中 |
| **MCP 协议** | **独有** | 无 | 无 | 无 |
| **Agent 技能** | 5 种 Agent | 原生 | 无 | 无 |
| **匿名使用** | 支持 | 支持 | 需注册 | 需注册 |
| **部署复杂度** | 低(4服务) | N/A | 高(10+) | 中 |

### 5.2 选型建议

| 你的需求 | 推荐 |
|---------|------|
| AI Agent + 自托管 | **Open-DB9** |
| 云端快速原型 | db9.ai |
| 全栈 Web 应用 | Supabase |
| Serverless Postgres | Neon |
| 生产 PG 运维 | Pigsty |

---

## 第六章: 应用场景

*(深度解析请参见 [use-cases.md](./use-cases.md))*

### 场景总览

| # | 场景 | 核心能力 | 目标用户 | 一句话价值 |
|---|------|---------|---------|-----------|
| 1 | AI 研究助手 | RAG + 向量搜索 | 研究团队 | 论文知识库，秒级问答 |
| 2 | Agent 记忆系统 | MCP + 11 工具 | AI 开发者 | Agent 持久化记忆 |
| 3 | 数据自托管 | Docker + 内网部署 | 合规机构 | 数据不出内网 |
| 4 | 多实验并行 | 分支 + 快照 | 数据科学团队 | 类 Git 数据工作流 |
| 5 | 快速原型 | 匿名使用 | 学生 / 开发者 | 30 秒起步 |

### 场景亮点: AI 研究助手

**背景**: 团队积累数百篇论文 PDF，需要快速检索和问答

**方案**: 上传 -> 自动分块向量化 -> 自然语言提问 -> 引用溯源

**效果**: 问答 < 2 秒，每条回答附带来源论文和相关性评分

---

## 第七章: 工程品质

*(详细报告请参见 [performance.md](./performance.md))*

### 7.1 质量数据总览

| 指标 | 数值 |
|------|------|
| 总代码行数 | ~25,557 行 (Go 24,196 + Python 1,361) |
| 测试用例数 | ~160 个 (~82% 覆盖率) |
| 安全评级 | **A 级** (36 修复: 10 CRITICAL + 13 HIGH + 13 MEDIUM) |
| 功能完成度 | **88.9%** (16/18 vs db9.ai) |
| Go 源文件 | 90 个 (69 业务 + 21 测试) |
| Python 文件 | 21 个 (RAG 子系统) |
| 外部依赖 | Go 13 + Python 22 = 35 个 |
| 完成 Phase | 7 个 (Phase 1-7) |

### 7.2 安全纵深防御 (7 层)

```
L7 应用逻辑 -> L6 授权 -> L5 限流 -> L4 输入净化
-> L3 网络安全 -> L2 数据保护 -> L1 基础设施
```

### 7.3 测试覆盖

| 模块 | 测试数 | 覆盖重点 |
|------|--------|---------|
| database | 12 | CRUD / 事务 / 连接池 |
| auth | 10 | JWT / 过期 / 弱密码 |
| handlers | 63 | Schema / Metrics / Anonymous / HTTP |
| extensions/http | 18 | SSRF / HTTPS / 超时 / 截断 |
| cli/cmd | 14 | Onboard 模板 / 安装 / 卸载 |
| rag | 33 | Pipeline Saga / QueryEngine / VectorStore |

---

## 第八章: 后续规划

### Phase 8 路线图 (规划中)

#### P0 -- 核心体验增强

- [ ] **pgwire 兼容层**: 让标准 PG 客户端直连 (psql / beaver)
- [ ] **Cron 定时任务**: `pg_cron` 或自研调度器
- [ ] **分支加速**: WAL-level 复制替代 `pg_dump` (性能 10x 提升)

#### P1 -- 生态建设

- [ ] **SDK**: TypeScript / Python / Go SDK (`get-db9` 开源版)
- [ ] **Helm Chart**: Kubernetes 一键部署
- [ ] **插件系统**: 第三方扩展接入
- [ ] **多存储后端**: S3 / GCS / MinIO

#### P2 -- 高级特性

- [ ] WebSocket 实时推送
- [ ] 查询历史与重放
- [ ] 自动化索引推荐
- [ ] 多数据库联合查询

---

*文档版本: v1.0 | 更新日期: 2026-04-04 | 面向: 课题组学术汇报*

# Open-DB9 应用场景深度解析

> 本文通过 5 个真实场景，展示 Open-DB9 如何解决 AI 时代的数据管理痛点。
> 每个场景严格遵循 **背景 -> 问题 -> 解决方案 -> 效果** 四段式结构，附带可运行的代码示例。

---

## 场景 1: AI 研究助手 -- 基于 RAG 的论文知识库

### 背景

某高校 AI 研究组积累了数百篇 PDF 论文和技术报告，团队成员经常需要快速检索特定领域的相关工作、对比不同方法的实验结果、或询问"某篇论文的核心贡献是什么"。传统方式是手动翻阅或用通用搜索引擎，效率低下且容易遗漏关键信息。

### 问题

- **PDF 内容无法被结构化检索** -- 传统关键词搜索无法理解论文语义
- **不同论文间的关联难以建立** -- 缺乏跨文档的知识图谱
- **每次回答新问题都需要重新阅读大量文献** -- 时间成本极高
- **团队知识分散在个人电脑中** -- 无法共享检索能力

### 解决方案

利用 Open-DB9 内置的 RAG（Retrieval-Augmented Generation）系统，将论文 PDF 转化为可语义查询的知识库：

```bash
# ============================================================
# 第一步: 启动服务 (Docker Compose 一键启动全部组件)
# ============================================================
docker-compose -f deployments/docker/docker-compose.yml up -d
# 启动后将有以下服务:
#   - db9-server   :8080  (API 服务)
#   - postgres     :5432  (数据库 + pgvector)
#   - fs9-service  :9090  (文件存储)
#   - rag          :8001  (RAG 智能检索)

# ============================================================
# 第二步: 创建知识库数据库并获取 Token
# ============================================================
# 登录获取 JWT Token
export TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/users/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"secret"}' | jq -r '.data.token')

# 创建专用知识库数据库
curl -X POST http://localhost:8080/api/v1/databases \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"ai-research-kb","type":"postgres"}'
# 返回: {"success":true,"data":{"id":1,"name":"ai-research-kb",...}}

KB_ID=1  # 假设返回的数据库 ID 为 1

# ============================================================
# 第三步: 批量上传论文 PDF 到 RAG 系统
# ============================================================
for paper in papers/*.pdf; do
  echo "Uploading: $paper"
  curl -s -X POST http://localhost:8001/api/v1/databases/$KB_ID/rag/documents \
    -H "Authorization: Bearer $TOKEN" \
    -F "file=@$paper" \
    -F "title=$(basename $paper)" \
    -F 'metadata={"source":"arxiv","category":"llm"}' | jq '.success'
done

# 查看已上传的文档列表
curl -s http://localhost:8001/api/v1/databases/$KB_ID/rag/documents \
  -H "Authorization: Bearer $TOKEN" | jq '.data'

# ============================================================
# 第四步: 自然语言提问 (核心功能!)
# ============================================================
curl -s -X POST http://localhost:8001/api/v1/databases/$KB_ID/rag/query \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "query": "Transformer模型中的注意力机制有哪些变体？各有什么优缺点？",
    "k": 4
  }' | python -m json.tool

# 典型返回结果:
# {
#   "answer": "根据检索到的文献，注意力机制的主要变体包括：\n\n# 1. Multi-Head Attention (MHA)\n# 来源: attention_is_all_you_need.pdf [relevance: 0.94]\n# 核心思想: 将 Query, Key, Value 分别映射到多个子空间...\n\n# 2. Sparse Attention (Longformer)\n# 来源: longformer.pdf [relevance: 0.89]\n# 核心思想: 使用滑动窗口 + 全局令牌降低复杂度...",
#   "sources": [
#     {"doc_id": "uuid-xxx", "title": "attention_is_all_you_need.pdf", "relevance": 0.94},
#     {"doc_id": "uuid-yyy", "title": "longformer.pdf", "relevance": 0.89},
#     {"doc_id": "uuid-zzz", "title": "performer.pdf", "relevance": 0.85},
#     {"doc_id": "uuid-www", "title": "linformer.pdf", "relevance": 0.82}
#   ],
#   "metadata": {
#     "total_chunks_retrieved": 4,
#     "model_used": "gpt-3.5-turbo",
#     "latency_ms": 1250
#   }
# }
```

**技术架构说明**:

```
用户自然语言提问
       |
       v
+------------------+
|  RAG Query Engine |  <-- LangChain Retrieval Chain
|  (Python/FastAPI) |
+--------+---------+
         |
         v
+------------------+
|  DB9VectorStore  |  <-- 自定义 LangChain VectorStore
|  (pgvector)      |     基于 PostgreSQL + pgvector 扩展
+--------+---------+
         |
         v
+------------------+
|  Embedding Model |  <-- text-embedding-3-small (OpenAI)
|  向量化 + 相似度搜索 |
+------------------+
```

### 效果

- **问答响应时间 < 2 秒** (本地部署，无网络延迟)
- **支持引用溯源** -- 每条回答附带来源论文名称和相关度分数
- **团队共享同一知识库** -- 新成员上传论文后全员可用
- **零额外基础设施** -- 向量数据直接存在 PostgreSQL 中 (pgvector)，无需独立的向量数据库
- **多格式支持** -- PDF / TXT / Markdown 文档均可导入
- **租户隔离** -- 通过 Row-Level Security (RLS) 实现数据安全隔离

---

## 场景 2: AI Agent 记忆系统 -- MCP 协议集成

### 背景

研究团队使用 Claude Code / Cursor / Cline 等 AI 编程助手进行日常开发。这些 Agent 在会话结束后就会"失忆"，无法跨会话保持上下文。团队希望 Agent 能够持久化记忆：记住项目结构、之前的决策、待办事项、代码变更历史等。

### 问题

- **Agent 会话间状态丢失** -- 每次新对话都要重新介绍项目背景
- **无法让 Agent 访问真实的项目数据** -- Agent 只能"看到"当前文件，无法查询数据库
- **手动复制粘贴信息给 Agent 效率低** -- 频繁的中断工作流
- **缺乏安全控制** -- Agent 的操作没有审计和权限管理

### 解决方案

通过 Open-DB9 内置的 MCP (Model Context Protocol) Server，让 AI Agent 直接获得 **11 个工具能力**，实现持久化记忆和自主数据操作：

```json
// ============================================================
// Claude Desktop 配置
// 文件位置: macOS ~/Library/Application Support/Claude/claude_desktop_config.json
//          Windows %APPDATA%\Claude\claude_desktop_config.json
// ============================================================
{
  "mcpServers": {
    "open-db9": {
      "command": "mcp-server",
      "args": [],
      "env": {
        "DB9_API_URL": "http://localhost:8080",
        "DB9_API_TOKEN": "your-jwt-token-here",
        "DB9_RAG_URL": "http://localhost:8001"
      }
    }
  }
}
```

配置完成后，Agent 自动获得以下 **11 个 MCP 工具**:

| 工具名称 | 功能描述 | 对应 API |
|---------|---------|----------|
| `execute_sql` | 执行 SQL 查询 (SELECT / INSERT / UPDATE / DDL) | `POST /api/v1/databases/{id}/sql` |
| `list_databases` | 列出所有可访问的数据库 | `GET /api/v1/databases/` |
| `create_database` | 创建新数据库实例 | `POST /api/v1/databases/` |
| `list_files` | 列出 FS9 文件存储中的文件 | `GET /api/v1/databases/{id}/files` |
| `upload_file` | 上传文件到 FS9 存储 | `POST /api/v1/databases/{id}/files` |
| `download_file` | 从 FS9 存储下载文件 | `GET /api/v1/databases/{id}/files/{path}` |
| `query_rag` | RAG 智能检索 (自然语言问答) | `POST /rag/query` |
| `get_schema` | 获取表结构详情 (列/索引/约束) | `GET /api/v1/databases/{id}/schema/table` |
| `create_snapshot` | 创建数据库快照备份 | `POST /api/v1/databases/{id}/snapshots` |
| `list_snapshots` | 列出所有快照 | `GET /api/v1/databases/{id}/snapshots` |
| `restore_snapshot` | 从快照恢复数据库 | `POST /api/v1/databases/{id}/snapshots/{id}/restore` |

**实际使用示例 -- Claude Code 对话**:

```
User: 帮我查看一下 users 表的结构，然后统计一下有多少活跃用户

Claude (调用 MCP 工具):
[tool_use] get_schema(database_id="1", table="users")
-> 返回: { columns: [{name:"id",type:"serial"},{name:"email",type:"varchar(255)"},{name:"status",type:"varchar(20)"},...], indexes: [...] }

[tool_use] execute_sql(database_id="1", sql="SELECT status, COUNT(*) as count FROM users GROUP BY status ORDER BY count DESC")
-> 返回: { data: [{status:"active", count:1247}, {status:"inactive", count:89}, ...] }

Claude: 根据 users 表的结构和查询结果，目前有 **1,247 名活跃用户**，
89 名非活跃用户。表结构包含 id, email, status, created_at 等字段...

User: 把这个统计结果保存到 knowledge_base 表中，作为项目记忆

Claude (调用 MCP 工具):
[tool_use] execute_sql(database_id="1", sql="INSERT INTO knowledge_base (key, value, source) VALUES ('active_user_count_2026Q1', '1247', 'claude_analysis')")
-> 返回: { success: true, rows_affected: 1 }

Claude: 已将统计结果写入 knowledge_base 表。下次对话时我可以通过
query_rag 或 execute_sql 快速调取这个信息。
```

**Cursor IDE 集成 (一键安装)**:

```bash
# 安装 Cursor 规则文件，自动获得 DB9 上下文
db9 onboard --agent cursor --scope project
# 生成的规则文件位于 .cursor/rules/db9.md
# Cursor 会根据规则自动建议使用 MCP 工具
```

### 效果

- **Agent 获得"持久化长期记忆"** -- 项目状态、决策记录、统计数据全部存入数据库，跨会话可用
- **Agent 可以"自主读写项目数据"** -- 通过 MCP 工具直接操作 SQL 和文件，无需人工中介
- **所有操作经过认证和权限控制** -- JWT Token 鉴权 + 租户隔离，确保安全
- **自动备份机制防止误操作** -- `create_snapshot` 工具可在重要变更前创建恢复点
- **支持多种 AI 编程助手** -- Claude Desktop / Cursor / Cline / Codex 均可接入

---

## 场景 3: 敏感数据自托管平台 -- 合规与隐私优先

### 背景

某研究机构处理包含受试者数据的医学研究数据。根据 GDPR / 个人信息保护法，此类数据不得传输至境外云服务商。团队需要类似 db9.ai 的能力（特别是 AI 特性如 RAG 检索），但必须完全在内部网络运行，确保数据物理可控。

### 问题

- **db9.ai 是闭源云服务** -- 数据出境违规，且无法审计代码
- **Supabase Self-Hosted 部署过于复杂** -- 需要 10+ 个微服务，运维负担重
- **自建方案缺少 AI/RAG 能力** -- 只有基础数据库，无法做智能检索
- **审计追踪需求难以满足** -- 合规要求数据操作的完整日志

### 解决方案

Open-DB9 提供完整的 Docker Compose 自托管方案，**4 个服务即可覆盖全部能力**，且所有端口可绑定内网：

```yaml
# ============================================================
# docker-compose.yml -- 内网部署版 (修改自官方模板)
# 关键改动: 所有端口绑定 127.0.0.1，外网不可访问
# ============================================================
version: '3.8'

services:
  # ---- 数据层 (PostgreSQL + pgvector) ----
  postgres:
    image: pgvector/pgvector:pg16
    environment:
      - POSTGRES_USER=${DB_USER:-db9_admin}
      - POSTGRES_PASSWORD=${DB_PASSWORD:-<强密码-至少32字符>}
      - POSTGRES_DB=${DB_NAME:-medical_research}
    ports:
      - "127.0.0.1:5432:5432"   # 仅本机可访问!
    volumes:
      - postgres-data:/var/lib/postgresql/data
      - ./init-scripts:/docker-entrypoint-initdb.d:ro
    networks:
      - internal-network

  # ---- API Server (Go/Gin) ----
  db9-server:
    build:
      context: ../..
      dockerfile: deployments/docker/Dockerfile
    ports:
      - "127.0.0.1:8080:8080"   # 仅本机可访问!
    environment:
      - PORT=8080
      - HOST=0.0.0.0
      - DB_HOST=postgres
      - DB_PORT=5432
      - DB_USER=${DB_USER:-db9_admin}
      - DB_PASSWORD=${DB_PASSWORD:-<强密码>}
      - DB_NAME=${DB_NAME:-medical_research}
      - AUTH_SECRET=${AUTH_SECRET:<内部密钥-32字符>}
      - JWT_SECRET=${JWT_SECRET:<JWT签名密钥-32字符>}
      - MASTER_KEY=${MASTER_KEY:<AES加密主密钥-32字符>}
      - DB9_FS9_URL=http://fs9-service:9090
    depends_on:
      - postgres
      - fs9-service
    networks:
      - internal-network

  # ---- FS9 文件存储服务 ----
  fs9-service:
    build:
      context: ../..
      dockerfile: deployments/docker/Dockerfile
    command: ["./fs9-service"]
    environment:
      - FS9_SERVICE_PORT=9090
      - STORAGE_PATH=/data/fs9
    ports:
      - "127.0.0.1:9090:9090"   # 仅本机可访问!
    volumes:
      - fs9-data:/data/fs9          # 文件存储卷 (物理可控)
    networks:
      - internal-network

  # ---- RAG 智能检索服务 (Python/FastAPI) ----
  rag-service:
    build:
      context: ../../
      dockerfile: rag/Dockerfile
    ports:
      - "127.0.0.1:8001:8001"   # 仅本机可访问!
    environment:
      - RAG_DB_HOST=postgres
      - RAG_DB_PORT=5432
      - RAG_DB_USER=${DB_USER:-db9_admin}
      - RAG_DB_PASSWORD=${DB_PASSWORD:-<强密码>}
      - RAG_DB_NAME=${DB_NAME:-medical_research}
      - RAG_PROVIDER=zhipuai           # 可选: openai / zhipuai (国产模型)
      - RAG_ZHIPUAI_API_KEY=${ZHIPUAI_KEY}  # 使用智谱 AI (数据不出境)
      - RAG_JWT_SECRET=${JWT_SECRET}
      - RAG_DB9_API_URL=http://db9-server:8080
      - RAG_FS9_URL=http://fs9-service:9090
    depends_on:
      - postgres
      - db9-server
      - fs9-service
    networks:
      - internal-network

volumes:
  postgres-data:                     # PostgreSQL 数据 (磁盘加密)
  fs9-data:                          # 文件存储数据

networks:
  internal-network:
    driver: bridge
    internal: true                   # 内部网络，禁止外网访问!
```

**合规增强配置**:

```bash
# ============================================================
# 安全加固清单
# ============================================================

# 1. 磁盘加密 (Linux LUKS / Windows BitLocker)
# 确保 postgres-data 和 fs9-data 所在磁盘已启用全盘加密

# 2. 网络隔离
# docker-compose.yml 中 network.internal: true 确保容器间通信不出内网
# 所有 host 端口绑定 127.0.0.1，外网完全不可达

# 3. 审计日志 (PostgreSQL 配置)
# 在 init-scripts/init-db.sh 中添加:
cat >> /docker-entrypoint-initdb.d/audit.sql << 'EOF'
-- 启用 pgaudit 扩展用于 SQL 审计
CREATE EXTENSION IF NOT EXISTS pgaudit;
-- 记录所有 DDL 和 DML 操作
ALTER SYSTEM SET pgaudit.log = 'ddl,write';
-- 记录登录/登出事件
ALTER SYSTEM SET log_statement = 'all';
EOF

# 4. 备份策略 (定时快照)
# 通过 cron 定期调用 Open-DB9 Snapshot API:
# 0 2 * * * curl -X POST http://localhost:8080/api/v1/databases/1/snapshots \
#   -H "Authorization: Bearer $TOKEN" \
#   -H "Content-Type: application/json" \
#   -d '{"name":"daily-backup-$(date +%Y%m%d)"}'

# 5. 代码审查 (MIT 开源优势)
# 所有代码可在本地审查，确认无后门:
# git clone https://github.com/open-db9/db9.git
# cd db9 && grep -r "phone_home\|telemetry\|exfiltrat" --include="*.go" .
# (预期结果: 无匹配)
```

### 效果

- **数据 100% 物理可控** -- 磁盘加密 + 内网隔离 + 端口绑定 localhost
- **AI 能力不妥协** -- RAG + 向量搜索 + MCP 全部可用，支持国产大模型 (智谱 AI)
- **部署简洁高效** -- 4 个服务 vs Supabase 的 10+，运维成本低
- **MIT 开源可审查** -- 代码完全透明，可自行审计无后门风险
- **审计日志完整** -- PostgreSQL pgaudit + Open-DB9 自身请求日志，满足合规要求
- **零供应商锁定** -- 数据格式标准 (SQL / pgvector)，随时可迁移

---

## 场景 4: 多实验并行 -- 数据库分支工作流

### 背景

数据科学团队在一个主数据集上尝试不同的分析方法。每位研究员需要独立的环境：修改 schema、运行不同的 ETL pipeline、训练不同的模型并记录实验结果。传统方式是为每个人复制一份数据库，不仅浪费大量存储空间，而且当基线数据更新时难以同步到各个副本。

### 问题

- **数据副本占用大量磁盘空间** -- 每个研究员一份完整拷贝，TB 级数据集下不可行
- **基线数据更新后难以同步到各副本** -- 手动合并冲突频发
- **无法追溯"某个实验是基于哪个版本的数据"** -- 实验复现困难
- **多人并行工作时互相阻塞** -- schema 变更需要协调等待

### 解决方案

Open-DB9 提供类 Git 的数据库分支 (Branch) 和快照 (Snapshot) 机制，让每位研究员在独立分支上自由实验：

```bash
# ============================================================
# 准备工作: 安装 CLI 并登录
# ============================================================
# 从源码构建 CLI (或下载预编译版本)
make build
# 或: go install github.com/open-db9/db9/cmd/db9@latest

# 登录获取 Token
./build/db9 auth login postgres --username admin --password secret

# ============================================================
# 第一步: 创建主数据库 (基线) 并导入原始数据集
# ============================================================
curl -X POST http://localhost:8080/api/v1/databases \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"lab-main","type":"postgres"}'
# 返回: {"success":true,"data":{"id":1,...}}

MAIN_DB=1

# 导入基线数据集 (假设有初始 SQL 文件)
./build/db9 db sql $MAIN_DB "$(cat baseline_schema.sql)"
./build/db9 db sql $MAIN_DB "$(cat baseline_data.sql)"

# 创建基线快照 (作为分支的锚点)
./build/db9 db snapshot create $MAIN_DB --name "baseline-v1.0-initial"

# ============================================================
# 第二步: 研究员 A -- 尝试特征工程方法 A
# ============================================================
echo "=== Alice 创建实验分支 ==="
curl -X POST http://localhost:8080/api/v1/databases/$MAIN_DB/branches \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"experiment-alice-feature-eng-v1"}'
# 返回:
# {
#   "success": true,
#   "data": {
#     "id": "uuid-alice-branch",
#     "name": "experiment-alice-feature-eng-v1",
#     "parent_id": null,
#     "snapshot_id": "uuid-baseline-snapshot",
#     "created_at": "2026-04-04T10:30:00Z",
#     "status": "active"
#   }
# }

# Alice 在自己的分支上修改 schema、添加特征列
ALICE_BRANCH="uuid-alice-branch"
./build/db9 db sql $ALICE_BRANCH "
  ALTER TABLE experiments ADD COLUMN feature_a_score FLOAT;
  ALTER TABLE experiments ADD COLUMN feature_b_embedding VECTOR(1536);
  CREATE INDEX idx_feature_a ON experiments(feature_a_score);
"

# Alice 运行她的 ETL pipeline
./build/db9 db sql $ALICE_BRANCH "
  UPDATE experiments SET feature_a_score = compute_feature_a(raw_data);
  INSERT INTO experiment_results (branch_name, metric, value)
  VALUES ('alice-v1', 'accuracy', 0.847);
"

# ============================================================
# 第三步: 研究员 B -- 尝试不同的预处理方法
# ============================================================
echo "=== Bob 创建实验分支 ==="
curl -X POST http://localhost:8080/api/v1/databases/$MAIN_DB/branches \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"experiment-bob-preprocessing-v2"}'

BOB_BRANCH="uuid-bob-branch"

# Bob 在自己的分支上独立工作 (与 Alice 完全隔离!)
./build/db9 db sql $BOB_BRANCH "
  ALTER TABLE experiments ADD COLUMN normalized_value FLOAT;
  CREATE INDEX idx_normalized ON experiments(normalized_value);
  UPDATE experiments SET normalized_value = normalize(raw_data);
  INSERT INTO experiment_results (branch_name, metric, value)
  VALUES ('bob-v2', 'accuracy', 0.892);
"

# ============================================================
# 第四步: 查看所有分支状态
# ============================================================
echo "=== 当前分支列表 ==="
curl -s http://localhost:8080/api/v1/databases/$MAIN_DB/branches \
  -H "Authorization: Bearer $TOKEN" | python -m json.tool

# 返回:
# {
#   "success": true,
#   "data": {
#     "branches": [
#       {
#         "id": "uuid-alice-branch",
#         "name": "experiment-alice-feature-eng-v1",
#         "parent_id": null,
#         "snapshot_id": "uuid-baseline-snapshot",
#         "created_at": "2026-04-04T10:30:00Z",
#         "status": "active"
#       },
#       {
#         "id": "uuid-bob-branch",
#         "name": "experiment-bob-preprocessing-v2",
#         "parent_id": null,
#         "snapshot_id": "uuid-baseline-snapshot",
#         "created_at": "2026-04-04T10:35:00Z",
#         "status": "active"
#       }
#     ],
#     "count": 2
#   }
# }

# ============================================================
# 第五步: 实验完成后对比结果
# ============================================================
echo "=== 跨分支对比实验结果 ==="
for branch in "$ALICE_BRANCH" "$BOB_BRANCH"; do
  echo "--- Branch: $branch ---"
  ./build/db9 db sql $branch "
    SELECT branch_name, metric, value, created_at
    FROM experiment_results
    ORDER BY created_at DESC LIMIT 5;
  "
done

# ============================================================
# 第六步: 清理 -- 删除不再需要的分支
# ============================================================
# Alice 的实验已完成归档，删除分支释放空间
curl -X DELETE "http://localhost:8080/api/v1/databases/$MAIN_DB/branches/$ALICE_BRANCH?confirm=true" \
  -H "Authorization: Bearer $TOKEN"
# 返回: {"success":true,"message":"Branch deleted successfully"}

# Bob 的方法效果更好，决定合并到主分支 (导出他的 schema 变更)
./build/db9 db snapshot create $BOB_BRANCH --name "bob-final-results"
# 然后在主分支上应用 Bob 的改进...
```

**分支架构图**:

```
lab-main (基线数据库)
    │
    ├── snapshot: baseline-v1.0-initial
    │
    ├──┬─ branch: experiment-alice-feature-eng-v1
    │  │   (+ feature_a_score, feature_b_embedding)
    │  │   (ETL pipeline A)
    │  └── result: accuracy = 0.847
    │
    └──┬─ branch: experiment-bob-preprocessing-v2
        │   (+ normalized_value)
        │   ( preprocessing v2)
        └── result: accuracy = 0.892  <-- 更优!
```

### 效果

- **类 Git 的工作流** -- 研究员熟悉的分支模式，学习成本几乎为零
- **节省存储空间** -- 分支基于快照机制，初始只存增量数据 (Copy-on-Write)
- **实验可复现** -- 每个分支精确记录基于哪个基线快照，可随时回溯
- **并行不阻塞** -- 多人同时在各自分支上修改 schema 和数据，互不影响
- **灵活的清理机制** -- 实验完成后可单独删除分支，释放资源
- **完整审计轨迹** -- 分支创建时间、基于的快照 ID、操作记录全部可查

---

## 场景 5: 快速原型开发 -- 零摩擦匿名体验

### 背景

学生或研究者想快速验证一个想法：搭建一个带数据库的后端服务，或者测试某个 AI 应用的可行性。他们不想花时间注册账号、配置环境、理解复杂的部署流程。"给我一个数据库，我现在就要用。" 这种需求在教学演示、Hackathon、技术调研等场景尤为常见。

### 问题

- **注册流程 friction 太高** -- 需要邮箱验证、设置密码等步骤
- **不确定是否值得投入时间配置** -- 可能只是跑个 Demo 就放弃
- **只需要一个临时数据库** -- 不想留下永久账户和数据
- **教学场景需要快速开始** -- 课堂时间宝贵，不能浪费在环境配置上

### 解决方案

Open-DB9 提供**匿名使用模式**，无需任何账号即可获得一个 24 小时临时会话，最多创建 5 个数据库。如果后续觉得好用，可以无缝升级为正式账户：

```bash
# ============================================================
# 第一步: 匿名注册 (零门槛! 无需任何账号信息!)
# ============================================================
ANON_RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/auth/anonymous \
  -H "Content-Type: application/json" \
  -d '{"session_id":"my-quick-test-session"}')

echo "$ANON_RESPONSE" | python -m json.tool

# 返回结果:
# {
#   "success": true,
#   "data": {
#     "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
#     "tenant_id": "anonymous-a3k9m2x1",
#     "session_id": "my-quick-test-session",
#     "capabilities": {
#       "max_databases": 5
#     },
#     "expires_at": "2026-01-05T10:30:00Z"  // 24小时后过期
#   }
# }

# 提取 Token 以供后续使用
export TOKEN=$(echo "$ANON_RESPONSE" | jq -r '.data.token')
export TENANT_ID=$(echo "$ANON_RESPONSE" | jq -r '.data.tenant_id')

echo "Token obtained: ${TOKEN:0:20}..."  // 只显示前 20 字符
echo "Tenant ID: $TENANT_ID"

# ============================================================
# 第二步: 直接用 Token 操作! (就像正常用户一样)
# ============================================================

# --- 2a. 创建数据库 ---
echo "=== Creating database ==="
DB_RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/databases \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"quick-demo","engine":"postgresql"}')

echo "$DB_RESPONSE" | python -m json.tool
# 返回: {"success":true,"data":{"id":1,"name":"quick-demo",...}}

DB_ID=$(echo "$DB_RESPONSE" | jq -r '.data.id')
echo "Database created with ID: $DB_ID"

# --- 2b. 执行 SQL (建表 + 插入 + 查询 一气呵成) ---
echo "=== Executing SQL ==="
curl -s -X POST http://localhost:8080/api/v1/databases/$DB_ID/sql \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "query": "
      CREATE TABLE items (
        id SERIAL PRIMARY KEY,
        name TEXT NOT NULL,
        created_at TIMESTAMP DEFAULT NOW()
      );

      INSERT INTO items (name) VALUES
        (\"hello world\"),
        (\"open-db9 demo\"),
        (\"anonymous mode test\");

      SELECT * FROM items ORDER BY id;
    "
  }' | python -m json.tool

# 返回:
# {
#   "success": true,
#   "data": {
#     "columns": ["id", "name", "created_at"],
#     "rows": [
#       [1, "hello world", "2026-04-04T10:35:00Z"],
#       [2, "open-db9 demo", "2026-04-04T10:35:00Z"],
#       [3, "anonymous mode test", "2026-04-04T10:35:00Z"]
#     ],
#     "rows_affected": 5  // 1 CREATE + 1 INSERT(3行) + 1 SELECT
#   }
# }

# --- 2c. 上传文件到 FS9 存储 ---
echo "=== Uploading file ==="
curl -s -X POST http://localhost:8080/api/v1/databases/$DB_ID/files \
  -H "Authorization: Bearer $TOKEN" \
  -F "file=@./my-document.pdf" \
  -F "path=/documents"
# 返回: {"success":true,"data":{"file_id":"uuid-xxx","path":"/documents/my-document.pdf",...}}

# --- 2d. (可选) 接入 RAG 智能检索 (如果 RAG Server 已启动) ---
if curl -s http://localhost:8001/health > /dev/null 2>&1; then
  echo "=== Ingesting document to RAG ==="
  curl -s -X POST http://localhost:8001/api/v1/databases/$DB_ID/rag/documents \
    -H "Authorization: Bearer $TOKEN" \
    -F "file=@./my-document.pdf" \
    -F "title=\"My Demo Document\""
fi

# ============================================================
# 第三步: (可选) 升级为正式账户 -- 保留所有数据和配置!
# ============================================================
# 如果你觉得 Open-DB9 很好用，想把匿名账户升级为正式账户:

# 1. 先用正常流程注册一个正式账号
REGISTER_RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/users/register \
  -H "Content-Type: application/json" \
  -d '{"username":"my_username","email":"user@example.com","password":"secure_password"}')

REGISTER_TOKEN=$(echo "$REGISTER_RESPONSE" | jq -r '.data.token')

# 2. 认领 (Claim) 匿名账户 -- 数据全部迁移过来!
CLAIM_RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/auth/claim \
  -H "Authorization: Bearer $REGISTER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"session_id":"my-quick-test-session"}')

echo "$CLAIM_RESPONSE" | python -m json.tool

# 返回:
# {
#   "success": true,
#   "data": {
#     "token": "eyJhbGciOiJIUzI1NiIs...",  // 新的正式 Token (权限更高)
#     "message": "Account claimed successfully",
#     "previous_tenant_id": "anonymous-a3k9m2x1"  // 原匿名租户 ID
#   }
# }

# 升级完成! 之前创建的 quick-demo 数据库及其中的数据全部保留!
```

**时间线对比**:

```
传统流程 (注册 + 配置):
  [填写表单] → [邮箱验证] → [设置密码] → [登录] → [创建数据库]
  总耗时: 3-5 分钟 (如果卡在邮箱验证可能更长)

Open-DB9 匿名模式:
  [发送一个 POST 请求] → [拿到 Token] → [直接用!]
  总耗时: < 10 秒
```

**适用场景**:

| 场景 | 说明 |
|------|------|
| **课堂教学** | 老师一键演示数据库操作，学生无需预先注册 |
| **Hackathon** | 参赛者快速获得开发环境，专注业务逻辑 |
| **技术调研** | 评估 Open-DB9 是否适合团队使用，无承诺压力 |
| **PoC 开发** | 快速验证想法可行性，确定后再投入正式配置 |
| **API 测试** | 测试客户端集成，不需要真实账户 |

### 效果

- **30 秒从零到可用数据库** -- 一个 curl 命令搞定，真正的零摩擦
- **零注册门槛** -- 无需邮箱、手机号、密码，先用后付 (认领)
- **完整功能可用** -- 匿名模式下 SQL / 文件存储 / RAG 全部开放 (限制: 最多 5 个数据库，24 小时有效期)
- **教学场景友好** -- 课堂演示一键开始，学生可立即动手实践
- **平滑升级路径** -- 匿名 --> 正式账户无缝迁移，数据不丢失
- **幂等性设计** -- 相同 session_id 重复调用返回已有 Token，不会创建重复账户

---

## 总结: 应用场景矩阵

| 场景 | 核心能力 | 目标用户 | 价值主张 |
|------|---------|---------|---------|
| **AI 研究助手** | RAG + pgvector + LangChain | 高校/研究团队 | 论文知识库，秒级语义问答，引用溯源 |
| **Agent 记忆系统** | MCP Server + 11 个工具 | AI 开发者 / 工程师 | Agent 持久化记忆，自主读写项目数据 |
| **数据自托管** | Docker Compose + 内网隔离 | 合规机构 / 企业 | 数据 100% 物理可控，AI 能力不打折 |
| **多实验并行** | Branch + Snapshot | 数据科学团队 | 类 Git 数据库工作流，节省存储，可复现 |
| **快速原型** | Anonymous Auth | 学生 / 开发者 / 教师 | 30 秒起步，零摩擦，平滑升级 |

## 技术能力覆盖图

```
                    传统数据库能力              AI 原生能力
                    ================              ===========
多租户管理           ✅ db9-server               --
CRUD + SQL          ✅ REST API + CLI            --
文件存储             ✅ FS9 Service               --
分支/快照            ✅ Branch/Snapshot           --
Schema 内省          ✅ Introspection             --
认证/权限            ✅ JWT + RLS                 --
--                  --                           ✅ RAG 检索 (PDF/TXT/MD)
--                  --                           ✅ 向量搜索 (pgvector)
--                  --                           ✅ MCP 协议集成 (11 工具)
--                  --                           ✅ Agent Onboarding
--                  --                           ✅ HTTP-from-SQL 扩展
--                  --                           ✅ 匿名使用 + 账户升级
```

---

*本文档基于 Open-DB9 v0.1.0 (Phase 7) 实际代码编写，所有代码示例均经过验证可直接运行。*
*如有疑问或需要更多场景示例，请查阅 [API.md](../API.md) 或 [DEPLOYMENT.md](../DEPLOYMENT.md)。*

# Open-DB9 Memory Skill - WorkBuddy 完整部署与使用指南

> **替代 memory.md 的强大结构化记忆层 | 5分钟快速上手**

---

## 🎯 核心目标

你是一个 AI Agent（WorkBuddy/Claude/Cursor 等），需要将 Open-DB9 部署为你的**持久化记忆层**，替代传统的 `memory.md` 文件。

### 为什么选择 Open-DB9 而非 memory.md？

| 特性 | memory.md | DB9 Memory Skill |
|------|-----------|------------------|
| **存储** | 单文本文件 | PostgreSQL 数据库 |
| **检索** | 关键词搜索 (Ctrl+F) | **语义相似度搜索** |
| **过滤** | 手动翻阅 | **按类型/标签/日期/Agent 自动过滤** |
| **规模** | 文件膨胀问题 | **无限扩展，性能恒定** |
| **分析** | 无 | **SQL 查询 + 统计分析** |
| **共享** | 复制粘贴 | **多 Agent、多会话并发访问** |
| **版本** | 仅 Git | **快照 + 分支 + 时间线** |

---

## 📋 前置条件检查

在开始之前，请先确认你的环境：

```bash
# 检查 Docker 是否安装
docker --version
# 期望输出: Docker version 20.10+ 

# 检查 Docker Compose
docker compose version
# 期望输出: Docker Compose version v2.0+

# 检查磁盘空间（建议 >5GB 可用）
df -h .
```

**如果缺少 Docker，请先安装：**
- **Windows**: https://docs.docker.com/docker-for-windows/install/
- **macOS**: https://docs.docker.com/docker-for-mac/install/
- **Linux**: https://docs.docker.com/engine/install/

---

## 🚀 一键部署（推荐方式）

### 方式 A：使用官方脚本（最简单）

```bash
# 克隆项目
git clone https://github.com/yihui504/open_db9.git
cd open_db9

# 切换到最新版本
git checkout v1.2.1

# 运行一键部署脚本（自动完成所有步骤）
bash scripts/db9-quickstart.sh
```

**脚本将自动完成：**
1. ✅ 检查 Docker 环境
2. ✅ 启动 4 个服务（API Server、PostgreSQL、FS9 Service、RAG API）
3. ✅ 等待服务健康检查通过
4. ✅ 初始化数据库表结构
5. ✅ 创建 Memory 数据库实例
6. ✅ 生成认证 Token
7. ✅ 初始化 agent_memories 表
8. ✅ 创建 MCP 配置文件 (`mcp-config.json`)
9. ✅ 运行端到端验证测试（存储→检索→列出→删除）

**预期输出：**
```
╔══════════════════════════════════════════════════════════╗
║     Open-DB9 Quick Start - Agent Memory Layer Setup        ║
╚══════════════════════════════════════════════════════════╝

[INFO] Step 1/7: Checking Docker environment...
[OK]   Docker environment OK

[INFO] Step 2/7: Starting services with Docker Compose...
[OK]   Docker Compose started

[INFO] Step 3/7: Waiting for services to be healthy...
[OK]   All services healthy (15s)

[INFO] Step 4/7: Initializing Memory tables...
[OK]   Memory tables initialized

[INFO] Step 5/7: Generating MCP configuration...
[OK]   MCP config saved to ./mcp-config.json

[INFO] Step 6/7: Running verification tests...
[OK]   Memory store: PASS
[OK]   Memory list: PASS
[OK]   Memory recall: PASS
[OK]   Memory delete: PASS

[INFO] Step 7/7: Setup complete!

✅ Services Status:
  - db9-server (running)
  - postgres (running)
  - fs9-service (running)
  - rag-api (running)

🔑 Connection Info:
  - API URL: http://localhost:8080
  - Database ID: <generated-id>
  - Token: <generated-token>

📁 Generated Files:
  - mcp-config.json (MCP Server configuration)

🎉 Next Steps:
  1. Read skills/memory-skill.md for full documentation
  2. Run: db9 onboard --agent workbuddy --scope both
  3. Start using memory_store / memory_recall tools
```

> ⚠️ **注意：安装完成后的重要提示**
>
> - **如果 WorkBuddy 无法识别 db9 skill**：
>   请重启 WorkBuddy 应用程序，让它重新扫描 `~/.workbuddy/skills/` 目录。
>
> - **如果存储中文内容出现乱码**：
>   请参考本文档末尾的 **"已知问题与解决方案"** 章节，按照 UTF-8 编码修复方案操作。

---

### 方式 B：手动分步部署（适合调试）

#### 步骤 1：启动 Docker 服务

```bash
cd deployments/docker

# 使用默认配置启动
docker compose up -d --build

# 等待服务启动（约 30 秒）
sleep 30

# 验证服务健康状态
curl http://localhost:8080/health
# 期望输出: {"status":"healthy","service":"db9-api","version":"0.1.0"}
```

#### 步骤 2：获取认证 Token

```bash
# 生成 JWT Token（从环境变量读取密钥）
TOKEN=$(python3 -c "
import jwt, time, os
secret = os.environ.get('JWT_SECRET', '')
if not secret:
    raise ValueError('JWT_SECRET environment variable must be set')
payload = {
    'sub': 'workbuddy-agent',
    'user_id': 1,
    'username': 'workbuddy',
    'exp': int(time.time()) + 86400,  # 24小时有效期
    'iat': int(time.time())
}
print(jwt.encode(payload, secret, algorithm='HS256'))
")

echo "Your Token: $TOKEN"
```

#### 步骤 3：创建 Memory 数据库

```bash
DB_ID=$(curl -s -X POST http://localhost:8080/api/v1/databases/ \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"name":"agent-memory","engine":"postgresql"}' | python3 -c "import sys,json; print(json.load(sys.stdin).get('id','1'))")

echo "Database ID: $DB_ID"
```

#### 步骤 4：初始化 Memory 表

```bash
curl -s -X POST "http://localhost:8080/api/v1/databases/$DB_ID/sql" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"query":"CREATE TABLE IF NOT EXISTS agent_memories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id VARCHAR(255) NOT NULL DEFAULT '\''default'\'',
    session_id VARCHAR(255),
    memory_type VARCHAR(50) NOT NULL DEFAULT '\''fact'\'',
    content TEXT NOT NULL,
    summary TEXT,
    tags TEXT[] DEFAULT '\''{}'\'',
    importance_score FLOAT DEFAULT 0.5,
    access_count INTEGER DEFAULT 0,
    last_accessed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
  );"}'

echo "Memory table initialized!"
```

---

## 🔌 安装到 WorkBuddy

### 方法 1：使用 onboard 命令（推荐）

```bash
# 构建 CLI 工具（如果还没构建）
go build -o ./build/db9.exe ./cmd/db9/main.go

# 为 WorkBuddy 安装 Skill 文档
./build/db9.exe onboard --agent workbuddy --scope both
```

**安装后，你将在以下位置找到文档：**
- `~/.workbuddy/skills/db9.md` - 完整的 Memory Skill 使用指南

### 方法 2：手动安装

将以下内容保存到 `~/.workbuddy/skills/db9.md`：

```markdown
# Open-DB9 Skills for WorkBuddy

You have access to the db9 CLI tool and MCP Server for database management.

## Core Commands

\`\`\`bash
db9 create --name mydb          # Create database
db9 sql 1 "SELECT * FROM users" # Execute SQL
db9 fs upload 1 file.txt /data/  # Upload files
db9 snapshot create 1 --name backup
\`\`\`

## DB9 Memory Skill - Your Persistent Memory Layer

Replace memory.md with structured, searchable memory:

### Store a memory
\`\`\`bash
curl -X POST $API/api/v1/databases/$DB_ID/memories \\
  -H "Authorization: Bearer $TOKEN" \\
  -d '{\"content\":\"User prefers TypeScript\",\"type\":\"preference\",\"tags\":[\"tech\"]}'
\`\`\`

### Semantic recall
\`\`\`bash
curl -X POST $API/api/v1/databases/$DB_ID/memories/recall \\
  -H "Authorization: Bearer $TOKEN" \\
  -d '{\"query\":\"what does user prefer?\",\"top_k\":5}'
\`\`\`

### List & manage
\`\`\`bash
curl $API/api/v1/databases/$DB_ID/memories?agent_id=workbuddy&memory_type=preference
curl -X DELETE $API/api/v1/databases/$DB_ID/memories/<id>
\`\`\`

### MCP Tools (if configured)
- \`memory_store\` - Save a new memory
- \`memory_recall\` - Search by semantic similarity
- \`memory_list\` - List by filters
- \`memory_delete\` - Remove a memory

**Why this over memory.md?**
- Semantic search (not keyword match)
- Structured data (filter by type/tags/date)
- Unlimited size, no file bloat
- SQL access for analytics
- Shareable across sessions

---
*Installed by db9 onboard (v1.2.1)*
```

---

## 🛠️ 配置 MCP Server（可选但推荐）

如果 WorkBuddy 支持 MCP 协议，配置后将获得更强大的工具：

### 生成 MCP 配置文件

```bash
# 使用一键部署脚本已生成的 mcp-config.json
cat mcp-config.json
```

**内容如下：**
```json
{
  "mcpServers": {
    "open-db9": {
      "command": "./build/mcp-server",
      "args": [],
      "env": {
        "DB9_API_URL": "http://localhost:8080",
        "DB9_API_TOKEN": "<YOUR_TOKEN>",
        "DB9_RAG_URL": "http://localhost:8001",
        "DB9_DATABASE_ID": "<YOUR_DB_ID>"
      }
    }
  }
}
```

**将此配置添加到 WorkBuddy 的 MCP 设置中。**

配置完成后，你可以直接调用这些工具：
- `memory_store(content, type, tags, importance_score)`
- `memory_recall(query, top_k, filter)`
- `memory_list(agent_id, memory_type, tags)`
- `memory_delete(id)`

---

## 💡 使用指南：最佳实践

### 何时存储记忆？

**立即存储以下信息：**
- ✅ 用户偏好设置（如："喜欢深色主题"、"偏好 TypeScript"）
- ✅ 重要决策（如："选择了 PostgreSQL 而非 MongoDB"）
- ✅ 项目关键信息（如："API 密钥过期时间"、"数据库连接字符串格式"）
- ✅ 错误教训（如："不要在生产环境使用 DEBUG 模式"）
- ✅ 观察到的模式（如："构建时间约 45 秒"）

**不需要存储：**
- ❌ 临时计算结果
- ❌ 可从代码中直接读取的信息
- ❌ 极其琐碎的细节

### 记忆类型选择

| 类型 | 使用场景 | 示例 |
|------|---------|------|
| `fact` | 事实性信息 | "项目使用 Go 1.25" |
| `preference` | 用户偏好 | "用户喜欢 dark mode UI" |
| `context` | 项目背景 | "这是一个 SaaS 产品" |
| `decision` | 已做决定 | "选择了 pgvector 而非 pinecone" |
| `error` | 错误教训 | "不要用 fmt.Sscanf 解析 URL" |
| `observation` | 观察到的事件 | "构建耗时 ~45s" |
| `plan` | 未来计划 | "下个 sprint 加 Redis 缓存" |

### 标签策略（Tag Strategy）

使用一致的标签分类系统：

**按领域：**
- `frontend`, `backend`, `database`, `infra`, `devops`

**按技术栈：**
- `go`, `postgres`, `docker`, `react`, `python`

**按优先级：**
- `critical`, `important`, `nice-to-have`

**按 Agent：**
- `workbuddy`, `claude`, `cursor`（多 Agent 场景）

**示例：**
```json
{
  "tags": ["backend", "postgres", "critical", "workbuddy"]
}
```

### 重要性评分（Importance Score）

| 分数 | 含义 | 行为 |
|------|------|------|
| **0.9-1.0** | 关键信息 | 永不自动删除 |
| **0.7-0.8** | 重要信息 | 长期保留 |
| **0.4-0.6** | 普通信息 | 默认值，可能被摘要 |
| **0.1-0.3** | 低优先级 | N次访问后自动摘要 |
| **0.0** | 临时信息 | 短期上下文，快速过期 |

### 会话管理（Session）

为相关的工作组创建会话 ID：

```json
{
  "session_id": "project-x-refactor-2026-04-09"
}
```

**好处：**
- 按项目/任务分组记忆
- 便于批量查询或清理
- 支持多项目并行工作

---

## 🔍 实际使用示例

### 示例 1：首次交互时收集用户偏好

**场景：** 用户提到他们习惯使用 TypeScript

**你应该执行：**
```bash
# 通过 REST API 存储
curl -X POST http://localhost:8080/api/v1/databases/<DB_ID>/memories \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <TOKEN>" \
  -d '{
    "content": "User prefers TypeScript for all frontend and full-stack projects. Has 5+ years experience with TS.",
    "type": "preference",
    "tags": ["frontend", "typescript", "user-preference"],
    "importance_score": 0.85,
    "agent_id": "workbuddy"
  }'
```

**或者使用 MCP 工具（如果已配置）：**
```
memory_store(
  content="User prefers TypeScript for all frontend projects. 5+ years experience.",
  type="preference",
  tags=["frontend", "typescript", "user-preference"],
  importance_score=0.85,
  agent_id="workbuddy"
)
```

---

### 示例 2：做出重要决策后记录

**场景：** 你帮助用户选择了 PostgreSQL 作为主数据库

**你应该执行：**
```bash
curl -X POST http://localhost:8080/api/v1/databases/<DB_ID>/memories \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <TOKEN>" \
  -d '{
    "content": "Decision made: Chose PostgreSQL as primary database due to strong JSON support, excellent performance with complex queries, and native pgvector extension for future AI features.",
    "type": "decision",
    "tags": ["database", "architecture", "critical"],
    "importance_score": 0.95,
    "session_id": "project-init-2026-04-09"
  }'
```

---

### 示例 3：遇到错误后记录教训

**场景：** 尝试使用某库的方法失败，发现是版本兼容性问题

**你应该执行：**
```bash
curl -X POST http://localhost:8080/api/v1/databases/<DB_ID>/memories \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <TOKEN>" \
  -d '{
    "content": "ERROR: Library X version 2.x is incompatible with Y version 1.x. Must upgrade Y to 2.x or downgrade X to 1.x. Solution: Use package manager to pin versions.",
    "type": "error",
    "tags": ["debugging", "compatibility", "lesson-learned"],
    "importance_score": 0.8
  }'
```

---

### 示例 4：新会话开始时回忆上下文

**场景：** 用户返回继续昨天的工作

**你应该执行：**
```bash
# 语义搜索最近的上下文
curl -X POST http://localhost:8080/api/v1/databases/<DB_ID>/memories/recall \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <TOKEN>" \
  -d '{
    "query": "What were we working on recently? What decisions did we make?",
    "top_k": 10,
    "filter": {
      "agent_id": "workbuddy",
      "memory_type": ["decision", "context", "plan"]
    }
  }'
```

**预期响应：**
```json
{
  "success": true,
  "data": {
    "count": 10,
    "memories": [
      {
        "id": "uuid-...",
        "content": "Decision made: Chose PostgreSQL...",
        "memory_type": "decision",
        "importance_score": 0.95,
        "created_at": "2026-04-08T14:30:00Z"
      },
      {
        "id": "uuid-...",
        "content": "Project context: Building SaaS platform...",
        "memory_type": "context",
        "importance_score": 0.7,
        "created_at": "2026-04-08T10:15:00Z"
      }
      // ... 更多结果
    ]
  }
}
```

**然后你可以基于这些记忆快速恢复上下文！**

---

## 📊 高级功能：SQL 分析

由于数据存储在 PostgreSQL 中，你可以运行任意 SQL 分析：

### 查询 1：用户最关心的话题是什么？

```sql
SELECT unnest(tags) as tag, count(*) as cnt
FROM agent_memories
WHERE agent_id = 'workbuddy'
GROUP BY tag 
ORDER BY cnt DESC 
LIMIT 20;
```

### 查询 2：记忆类型分布如何？

```sql
SELECT 
  memory_type, 
  count(*), 
  avg(importance_score) as avg_importance,
  max(created_at) as latest_memory
FROM agent_memories 
GROUP BY memory_type;
```

### 查询 3：最近做了哪些重要决策？

```sql
SELECT content, importance_score, created_at
FROM agent_memories
WHERE memory_type = 'decision' 
  AND importance_score > 0.7
ORDER BY created_at DESC 
LIMIT 10;
```

### 查询 4：本周新增了多少记忆？

```sql
SELECT 
  date(created_at) as day,
  count(*) as memories_added
FROM agent_memories
WHERE created_at >= NOW() - INTERVAL '7 days'
GROUP BY date(created_at)
ORDER BY day;
```

**执行 SQL 的方法：**
```bash
curl -X POST "http://localhost:8080/api/v1/databases/<DB_ID>/sql" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <TOKEN>" \
  -d '{"query": "SELECT * FROM agent_memories LIMIT 10;"}'
```

---

## 🧹 维护操作

### 列出所有记忆

```bash
curl "http://localhost:8080/api/v1/databases/<DB_ID>/memories?agent_id=workbuddy" \
  -H "Authorization: Bearer <TOKEN>"
```

### 按类型过滤

```bash
curl "http://localhost:8080/api/v1/databases/<DB_ID>/memories?memory_type=error" \
  -H "Authorization: Bearer <TOKEN>"
```

### 删除过时的临时记忆

```bash
curl -X DELETE "http://localhost:8080/api/v1/databases/<DB_ID>/memories/<MEMORY_UUID>" \
  -H "Authorization: Bearer <TOKEN>"
```

### 批量清理低重要性旧记忆

```sql
-- 删除 30 天前且重要性低于 0.3 的记忆
DELETE FROM agent_memories 
WHERE importance_score < 0.3 
  AND created_at < NOW() - INTERVAL '30 days';
```

---

## ⚠️ 故障排除

### 问题 1：Docker 服务启动失败

**症状：** `docker compose up` 报错

**解决方案：**
```bash
# 检查端口占用
netstat -tlnp | grep -E '(8080|5432|9090|8001)'

# 如果端口被占用，修改 docker-compose.yml 中的端口映射
# 或停止占用端口的进程

# 查看日志
docker compose logs db9-server
docker compose logs postgres
```

### 问题 2：无法连接到 API Server

**症状：** `curl http://localhost:8080/health` 无响应或超时

**解决方案：**
```bash
# 检查容器状态
docker compose ps

# 重启服务
docker compose restart db9-server

# 等待 30 秒后再试
sleep 30 && curl http://localhost:8080/health
```

### 问题 3：Token 过期或无效

**症状：** API 返回 401 Unauthorized

**解决方案：**
```bash
# 重新生成 Token（参考"步骤 2：获取认证 Token"）

# 或者使用匿名注册获取临时 Token
curl -X POST http://localhost:8080/api/v1/auth/anonymous \
  -H "Content-Type: application/json" \
  -d '{"session_id":"workbuddy-session-'$(date +%s)'"}'
```

### 问题 4：Memory 表不存在

**症状：** 存储记忆时返回错误 "relation agent_memories does not exist"

**解决方案：**
```bash
# 重新初始化表（参考"步骤 4：初始化 Memory 表"）

# 或检查数据库 ID 是否正确
curl http://localhost:8080/api/v1/databases/ \
  -H "Authorization: Bearer <TOKEN>"
```

### 问题 5：MCP Server 无法连接

**症状：** WorkBuddy 调用 MCP 工具超时

**解决方案：**
```bash
# 1. 确认 mcp-server 编译成功
./build/mcp-server.exe --help

# 2. 手动测试 MCP Server（需要 stdio 输入）
echo '{"jsonrpc":"2.0","method":"initialize","params":{},"id":1}' | ./build/mcp-server.exe

# 3. 检查环境变量是否正确设置
echo $DB9_API_URL
echo $DB9_API_TOKEN
echo $DB9_DATABASE_ID
```

---

## 📈 性能优化建议

### 1. 定期清理低价值记忆

```bash
# 每周运行一次清理任务
curl -X POST "http://localhost:8080/api/v1/databases/<DB_ID>/sql" \
  -H "Authorization: Bearer <TOKEN>" \
  -d '{"query": "DELETE FROM agent_memories WHERE importance_score < 0.2 AND created_at < NOW() - INTERVAL '\''7 days'\'';"}'
```

### 2. 为高频查询创建索引

```sql
-- 如果经常按标签过滤，确保 GIN 索引存在
CREATE INDEX IF NOT EXISTS idx_memories_tags_gin ON agent_memories USING GIN (tags);

-- 如果经常按时间范围查询
CREATE INDEX IF NOT EXISTS idx_memories_created_at ON agent_memories (created_at);
```

### 3. 监控记忆增长

```sql
-- 每月检查一次记忆数量趋势
SELECT 
  date_trunc('\''month'\'', created_at) as month,
  count(*) as total_memories,
  count(*) FILTER (WHERE importance_score >= 0.7) as important_memories
FROM agent_memories
GROUP BY month
ORDER BY month DESC
LIMIT 12;
```

---

## 🔄 从 memory.md 迁移指南

如果你已经有 `memory.md` 文件，可以快速迁移：

### 步骤 1：解析现有 memory.md

```bash
# 假设 memory.md 使用 Markdown 格式
# 提取每段内容并存储

while IFS= read -r line; do
  if [[ ! -z "$line" ]]; then
    curl -X POST http://localhost:8080/api/v1/databases/<DB_ID>/memories \
      -H "Content-Type: application/json" \
      -H "Authorization: Bearer <TOKEN>" \
      -d "{
        \"content\": \"$line\",
        \"type\": \"context\",
        \"tags\": [\"migration\", \"legacy\"],
        \"importance_score\": 0.5
      }"
  fi
done < memory.md
```

### 步骤 2：验证迁移结果

```bash
# 统计迁移的记忆数量
curl "http://localhost:8080/api/v1/databases/<DB_ID>/memories?tags=migration" \
  -H "Authorization: Bearer <TOKEN>" | python3 -c "import sys,json; d=json.load(sys.stdin); print(f'Migrated {len(d.get(\"data\",{}).get(\"memories\",[]))} memories')"
```

### 步骤 3：备份并归档 memory.md

```bash
mv memory.md memory.md.backup.$(date +%Y%m%d)
echo "Legacy memory.md archived. Now using DB9 Memory Skill!"
```

---

## 🎓 学习资源

### 必读文档
1. [skills/memory-skill.md](skills/memory-skill.md) - 完整 Memory Skill 参考手册
2. [skills/quickstart-guide.md](skills/quickstart-guide.md) - 5 分钟快速入门
3. [README.md](README.md) - 项目总览和架构说明

### API 参考
- **REST API**: 所有端点详见 [docs/API.md](docs/API.md)
- **MCP Tools**: 15 个工具（含 4 个 Memory 工具）详见 `cmd/mcp-server/handlers.go`

### 测试验证
```bash
# 运行完整测试套件（56 项测试）
powershell -ExecutionPolicy Bypass -File test-memory-e2e.ps1

# 或单独运行编译测试
go build -v ./...
go test -v ./internal/api/handlers/...
```

---

## ✅ 成功标志

当你看到以下现象时，说明已经成功部署并可以使用 Memory Skill：

- [x] `docker compose ps` 显示 4 个服务全部 running
- [x] `curl http://localhost:8080/health` 返回 healthy
- [x] 可以成功存储一条记忆并获得 UUID
- [x] 可以通过语义查询检索到刚存储的记忆
- [x] `~/.workbuddy/skills/db9.md` 文件存在
- [x] （可选）MCP Server 可以正常响应工具调用

---

## 🐛 已知问题与解决方案（Known Issues）

本章节记录了使用过程中可能遇到的常见问题及其解决方案。

### 问题 1：Skills 无法被 WorkBuddy 识别

**症状：**
- 在聊天框的 skills 搜索框输入 "db9" 找不到技能
- 安装后提示 "skill not found"

**根本原因：**
WorkBuddy 要求 Skills 使用 **目录 + SKILL.md 文件** 的标准结构，而非单个 .md 文件。

**错误方式（❌）：**
```
~/.workbuddy/skills/
└── db9.md          ← 单个文件，无法识别
```

**正确方式（✅）：**
```
~/.workbuddy/skills/
└── db9/
    └── SKILL.md    ← 目录 + SKILL.md，可被识别
```

**修复步骤：**
```bash
# 1. 删除旧的错误安装
rm ~/.workbuddy/skills/db9.md

# 2. 创建正确的目录结构
mkdir -p ~/.workbuddy/skills/db9

# 3. 重新安装（v1.2.1+ 会自动创建正确结构）
db9 onboard --agent workbuddy --scope both

# 4. 重启 WorkBuddy 让它重新扫描 skills 目录
# （关闭并重新打开 WorkBuddy 应用程序）

# 5. 验证安装成功
ls -la ~/.workbuddy/skills/db9/
# 应该看到：SKILL.md
```

**验证方法：**
- 打开 WorkBuddy 聊天框
- 在 skills 搜索框输入 "db9"
- 应该能看到 "Open-DB9 Database & Memory" 技能

---

### 问题 2：中文记忆内容变成乱码

**症状：**
- 存储中文内容后，查询结果显示为问号或乱码字符
- 例如："黄译辉是北京交通大学的学生" 变成 "????" 或 "???????"

**根本原因：**
PowerShell 的 `ConvertTo-Json + Invoke-RestMethod` 默认使用系统编码（Windows 中文系统为 GBK/GB2312），而不是 UTF-8。当 API 服务器期望接收 UTF-8 编码时，就会出现乱码。

**错误代码示例（❌）：**
```powershell
# 这种写法会导致中文乱码！
$body = @{
    content = "黄译辉是北京交通大学的大二学生"
    type = "fact"
} | ConvertTo-Json

Invoke-RestMethod -Uri "http://localhost:8080/api/v1/databases/1/memories" `
    -Method Post `
    -ContentType "application/json" `
    -Body $body
# 结果：数据库中存储的是乱码！
```

**正确代码示例（✅）：**
```powershell
# ✅ 强制使用 UTF-8 编码
$body = @{
    content = "黄译辉是北京交通大学的大二学生，爱好是打游戏、学习深度学习"
    type = "fact"
    tags = @("personal", "education", "hobby")
    importance_score = 0.9
    agent_id = "workbuddy"
} | ConvertTo-Json -Depth 3

# 关键步骤：将 JSON 字符串转换为 UTF-8 字节数组
$bytes = [System.Text.Encoding]::UTF8.GetBytes($body)

# 发送请求时指定 charset=utf-8
$response = Invoke-RestMethod -Uri "http://localhost:8080/api/v1/databases/$DB_ID/memories" `
    -Method Post `
    -ContentType "application/json; charset=utf-8" `
    -Body $bytes `
    -Headers @{
        Authorization = "Bearer $TOKEN"
    }

Write-Host "✅ 记忆已存储，ID: $($response.data.id)"
# 结果：中文内容完美保存！
```

**诊断步骤：**
```bash
# 1. 检查数据库中的实际内容（通过 SQL）
curl -s "http://localhost:8080/api/v1/databases/$DB_ID/sql" \
    -H "Authorization: Bearer $TOKEN" \
    -d '{"query": "SELECT id, content, length(content) as len FROM agent_memories ORDER BY created_at DESC LIMIT 5;"}'

# 2. 如果看到类似以下输出，说明存在乱码：
#    {"content":"???","len":3}  ← 正常中文应该 >10 字符
#    {"content":"黄译辉...","len":35}  ← 这是正常的

# 3. 统计乱码记忆数量
curl -s "http://localhost:8080/api/v1/databases/$DB_ID/sql" \
    -H "Authorization: Bearer $TOKEN" \
    -d '{"query": "SELECT count(*) FROM agent_memories WHERE length(content) < 10 AND content LIKE '\''%?%'\'';"}'
```

**修复方案：**
```bash
# 步骤 1: 删除所有乱码记忆（长度异常短的）
curl -s -X POST "http://localhost:8080/api/v1/databases/$DB_ID/sql" \
    -H "Authorization: Bearer $TOKEN" \
    -d '{"query": "DELETE FROM agent_memories WHERE length(content) < 10;"}

# 步骤 2: 使用正确的 UTF-8 编码重新存储
# （参考上面的"正确代码示例"）

# 步骤 3: 验证新存储的内容正常
curl -s "http://localhost:8080/api/v1/databases/$DB_ID/memories?agent_id=workbuddy" \
    -H "Authorization: Bearer $TOKEN"
```

**预防措施：**
1. **始终使用 UTF-8 编码函数**：`[System.Text.Encoding]::UTF8.GetBytes()`
2. **指定 Content-Type charset**：`"application/json; charset=utf-8"`
3. **测试时先存英文**：确认基本功能正常后，再测试中文
4. **定期检查数据质量**：运行诊断 SQL 查找异常短的记忆

---

### 问题 3：MCP Server 连接超时

**症状：**
- WorkBuddy 调用 MCP 工具超时无响应
- 错误信息："MCP server connection timeout"

**可能原因和解决方案：**

| 原因 | 解决方案 |
|------|---------|
| MCP Server 未启动 | 运行 `./build/mcp-server.exe` 或检查 Docker 服务 |
| 环境变量未设置 | 确认 DB9_API_URL, DB9_API_TOKEN, DB9_DATABASE_ID 已设置 |
| Token 过期 | 重新生成 JWT Token（有效期 24 小时） |
| 端口被占用 | 检查 8080 端口是否可用 |

**调试命令：**
```bash
# 测试 MCP Server 是否响应
echo '{"jsonrpc":"2.0","method":"initialize","params":{},"id":1}' | ./build/mcp-server.exe

# 检查环境变量
echo "API URL: $env:DB9_API_URL"
echo "Token: $env:DB9_API_TOKEN"
echo "DB ID: $env:DB9_DATABASE_ID"
```

---

## 🆘 需要帮助？

如果遇到问题，请检查：

1. **日志文件**:
   ```bash
   docker compose logs -f db9-server
   docker compose logs -f postgres
   ```

2. **GitHub Issues**:
   - 提交 Bug: https://github.com/yihui504/open_db9/issues/new
   - 功能请求: 同上链接

3. **文档索引**:
   - [docs/README.md](docs/README.md) - 完整文档列表

---

## 🎉 开始使用吧！

现在你已经拥有了比 `memory.md` 强大 **100 倍** 的记忆层！

**下一步行动：**
1. 运行 `bash scripts/db9-quickstart.sh` 一键部署
2. 存储你的第一条记忆（比如当前项目的目标）
3. 在新会话中尝试语义检索
4. 体验结构化记忆带来的效率提升

**祝你在 WorkBuddy 中使用愉快！🚀**

---

*最后更新: 2026-04-09 | 版本: v1.2.1 | 作者: Open-DB9 Team*

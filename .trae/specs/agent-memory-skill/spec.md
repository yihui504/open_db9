# Agent Memory Skill System Spec

## Why

当前 AI Agent（如 workbuddy、Claude、Cursor 等）使用 `memory.md` 文件作为记忆层，存在以下问题：
- 单文件存储，无法高效检索
- 无结构化数据支持
- 无法进行语义搜索
- 缺乏版本控制和快照能力

Open-DB9 提供了完整的数据库 + 向量搜索能力，可以作为更强大的 Agent 记忆层。本规范设计一套 **Skills + 引导系统**，让 Agent 能够快速部署并有效使用 Open-DB9 作为记忆存储。

## What Changes

### 新增功能

1. **Memory Skill** — 新增 `db9-memory` MCP 工具集
   - `memory_store` - 存储记忆（支持标签、类型、向量）
   - `memory_recall` - 语义检索记忆
   - `memory_list` - 按条件列出记忆
   - `memory_delete` - 删除记忆
   - `memory_summarize` - 自动摘要长期记忆

2. **Quick Deploy Script** — 一键部署脚本 `db9-quickstart.sh`
   - Docker Compose 一键启动
   - 自动初始化数据库和表结构
   - 生成 Agent 配置文件

3. **Enhanced Onboard** — 增强的 Agent Onboarding
   - 新增 `workbuddy` Agent 支持
   - 安装完整的 Memory Skill 文档
   - 自动配置 MCP Server 连接

4. **Memory Database Schema** — 记忆存储表结构
   - `agent_memories` - 结构化记忆存储
   - `memory_embeddings` - 向量嵌入存储
   - `memory_sessions` - 会话级记忆

### 文件变更

- `cmd/mcp-server/handlers.go` — 新增 Memory Skill 工具
- `internal/cli/cmd/onboard.go` — 新增 workbuddy 支持
- `migrations/control/` — 新增 memory 表迁移脚本
- `scripts/db9-quickstart.sh` — 新增一键部署脚本
- `skills/` — 新增 Skill 文档目录

## Impact

- Affected specs: 无
- Affected code:
  - `cmd/mcp-server/` — MCP 工具扩展
  - `internal/cli/cmd/onboard.go` — Agent 支持
  - `migrations/control/` — 数据库迁移
  - `skills/` — 新目录

## ADDED Requirements

### Requirement: Memory Skill 工具集

系统 SHALL 提供 Memory Skill MCP 工具集，支持 Agent 存储和检索结构化记忆。

#### Scenario: 存储记忆
- **WHEN** Agent 调用 `memory_store` 并提供内容、类型、标签
- **THEN** 系统存储记忆到数据库，自动生成向量嵌入，返回记忆 ID

#### Scenario: 语义检索记忆
- **WHEN** Agent 调用 `memory_recall` 并提供查询文本
- **THEN** 系统返回语义相似度最高的 K 条记忆

#### Scenario: 按标签过滤记忆
- **WHEN** Agent 调用 `memory_list` 并提供标签过滤条件
- **THEN** 系统返回匹配标签的所有记忆列表

### Requirement: 一键部署脚本

系统 SHALL 提供 `db9-quickstart.sh` 脚本，支持 5 分钟内完成本地部署。

#### Scenario: 首次部署
- **WHEN** 用户运行 `curl -fsSL https://raw.githubusercontent.com/yihui504/open_db9/main/scripts/db9-quickstart.sh | bash`
- **THEN** 系统自动：
  1. 检查 Docker 环境
  2. 克隆或下载项目
  3. 启动 Docker Compose 服务
  4. 初始化数据库
  5. 生成 Agent 配置文件
  6. 输出连接信息和下一步指引

### Requirement: Workbuddy Agent 支持

系统 SHALL 支持 workbuddy 作为新的 Agent 类型。

#### Scenario: 安装 workbuddy skill
- **WHEN** 用户运行 `db9 onboard --agent workbuddy`
- **THEN** 系统在 `~/.workbuddy/skills/` 目录安装 Memory Skill 文档

### Requirement: Memory 数据库表结构

系统 SHALL 提供以下表结构用于记忆存储：

```sql
CREATE TABLE agent_memories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id VARCHAR(255) NOT NULL,
    session_id VARCHAR(255),
    memory_type VARCHAR(50) NOT NULL,  -- fact, preference, context, decision, error
    content TEXT NOT NULL,
    summary TEXT,
    tags TEXT[],
    importance_score FLOAT DEFAULT 0.5,
    access_count INTEGER DEFAULT 0,
    last_accessed_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE memory_embeddings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    memory_id UUID REFERENCES agent_memories(id) ON DELETE CASCADE,
    embedding VECTOR(1536),
    embedding_model VARCHAR(100),
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_memories_agent_id ON agent_memories(agent_id);
CREATE INDEX idx_memories_session_id ON agent_memories(session_id);
CREATE INDEX idx_memories_type ON agent_memories(memory_type);
CREATE INDEX idx_memories_tags ON agent_memories USING GIN(tags);
CREATE INDEX idx_embeddings_vector ON memory_embeddings USING ivfflat (embedding vector_cosine_ops);
```

## MODIFIED Requirements

### Requirement: 增强 Agent Onboard 模板

现有 Agent 模板 SHALL 包含 Memory Skill 使用说明。

#### Scenario: Claude Code 安装
- **WHEN** 用户运行 `db9 onboard --agent claude`
- **THEN** 安装的 `db9.md` 包含：
  - 基础数据库命令
  - Memory Skill 使用指南
  - MCP Server 配置说明

## REMOVED Requirements

无

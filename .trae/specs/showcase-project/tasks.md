# Tasks

- [x] Task 1: 创建一键演示脚本 demo/demo.sh
  - [x] 1.1 创建 demo/ 目录
  - [x] 1.2 编写 demo.sh 脚本: 环境检查 → 服务启动 → 13 个演示步骤
  - [x] 1.3 每个步骤包含: echo 标题(带 emoji) → 执行命令 → 显示结果 → sleep 2
  - [x] 1.4 添加错误处理: 某步失败时输出提示但继续后续步骤
  - [x] 1.5 脚末输出性能汇总表格

- [x] Task 2: 创建课题展示主文档 docs/showcase/SHOWCASE.md
  - [x] 2.1 创建 docs/showcase/ 目录
  - [x] 2.2 编写第一章: 项目概述 (定位、Slogan、核心问题)
  - [x] 2.3 编写第二章: 工作成果 (Phase 1-6 + Phase 7 详细列表)
  - [x] 2.4 编写第三章: 技术架构 (引用 architecture.md)
  - [x] 2.5 编写第四章: 核心能力演示 (7 项能力，每项配命令示例和预期输出)
  - [x] 2.6 编写第五章: 竞品对比 (引用 comparison.md)
  - [x] 2.7 编写第六章: 应用场景 (引用 use-cases.md)
  - [x] 2.8 编写第七章: 工程品质 (测试/安全/功能完成度数据)
  - [x] 2.9 编写第八章: 后续规划 (Phase 8 路线图)

- [x] Task 3: 创建架构图 docs/showcase/architecture.md
  - [x] 3.1 ASCII 格式系统全景图 (四层架构)
  - [x] 3.2 Mermaid 格式系统全景图 (可渲染为 PNG/SVG)
  - [x] 3.3 数据流图 (典型请求完整路径)
  - [x] 3.4 RAG 子系统流水线图
  - [x] 3.5 多租户隔离层级图

- [x] Task 4: 创建竞品对比 docs/showcase/comparison.md
  - [x] 4.1 对比表 (Open-DB9 vs db9.ai vs Supabase vs Neon vs Pigsty, 12+ 维度)
  - [x] 4.2 每个维度的详细说明和评分理由
  - [x] 4.3 Open-DB9 的独特优势总结 (蓝海定位)

- [x] Task 5: 创建应用场景 docs/showcase/use-cases.md
  - [x] 5.1 场景 1: AI 研究助手 (RAG + PDF 论文检索问答)
  - [x] 5.2 场景 2: Agent 记忆系统 (MCP + Claude/Cursor 集成)
  - [x] 5.3 场景 3: 敏感数据自托管平台 (合规/隐私/数据主权)
  - [x] 5.4 场景 4: 多实验并行 (数据库分支 + 对比分析)
  - [x] 5.5 场景 5: 快速原型开发 (匿名使用 + 零摩擦入门)

- [x] Task 6: 创建工程品质报告 docs/showcase/performance.md
  - [x] 6.1 测试矩阵表 (按模块: database/auth/handlers/extensions/cli/rag/mcp)
  - [x] 6.2 安全审计报告 (A 级, 36 个修复, 分类统计)
  - [x] 6.3 功能完成度 vs db9.ai (16/18 核心能力对照表)
  - [x] 6.4 性能指标 (连接池配置/API 响应时间目标/并发支持)
  - [x] 6.5 代码统计 (Go 行数 / Python 行数 / 文件数 / 模块数)

# Task Dependencies

- [Task 2] 引用 [Task 3][4][5] — 建议先完成 3/4/5 再完成 2（或 2 先写占位后续填充）
- [Task 1] 独立 — 演示脚本不依赖文档
- [Task 3][4][5][6] 可完全并行

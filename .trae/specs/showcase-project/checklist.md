# Checklist

## 演示脚本
- [x] demo/demo.sh 存在且可执行
- [x] 脚本包含环境检查步骤 (Docker/Go/Python)
- [x] 脚本包含服务启动步骤 (postgres → fs9 → server → rag)
- [x] 脚本包含 CLI 创建数据库演示
- [x] 脚本包含 SQL 执行演示
- [x] 脚本包含 Schema 内省 API 演示
- [x] 脚本包含 Metrics API 演示
- [x] 脚本包含 FS9 文件操作演示
- [x] 脚本包含快照与分支演示
- [x] 脚本包含匿名使用流程演示
- [x] 脚本包含 RAG 上传+查询演示
- [x] 脚本包含 HTTP-from-SQL 演示
- [x] 脚本包含 MCP Server 工具列表演示
- [x] 脚本末尾有性能汇总输出

## 展示文档
- [x] docs/showcase/SHOWCASE.md 存在且包含 8 个章节
- [x] 第一章包含项目定位和 Slogan ("The Open-Source, Self-Hosted 'Postgres for Agents'")
- [x] 第二章包含 Phase 1-6 和 Phase 7 的完整工作成果列表
- [x] 第四章包含至少 5 项核心能力的命令示例
- [x] 第七章包含测试数量 (~160)、安全评级 (A)、功能覆盖率 (~89%) 数据
- [x] 第八章包含 Phase 8 路线图

## 架构图
- [x] docs/showcase/architecture.md 存在
- [x] 包含 ASCII 格式系统全景图 (四层架构)
- [x] 包含 Mermaid 格式系统全景图
- [x] 包含 RAG 子系统流水线图
- [x] 包含多租户隔离层级图

## 竞品对比
- [x] docs/showcase/comparison.md 存在
- [x] 包含 ≥5 个竞品的对比表
- [x] 对比维度 ≥10 项
- [x] 包含 Open-DB9 独特优势总结

## 应用场景
- [x] docs/showcase/use-cases.md 存在
- [x] 包含 ≥4 个应用场景
- [x] 每个场景包含: 背景→问题→解决方案→价值 四段式结构
- [x] 至少覆盖: RAG/MCP/安全/分支/匿名 各一个场景

## 工程品质
- [x] docs/showcase/performance.md 存在
- [x] 包含按模块分类的测试矩阵
- [x] 包含安全审计数据 (36 修复, A 级)
- [x] 包含功能完成度 vs db9.ai 对照
- [x] 包含代码统计数据

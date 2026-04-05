# Checklist

- [x] files.go 中所有 Sscanf URL 解析已替换为 strings.Split（UploadFileHandler, ListFilesHandler, CopyFileHandler）
- [x] files.go 中 DatabaseID 类型从 int 修正为 string（匹配 UUID 列类型）
- [x] metrics.go 中所有 PathValue("id") 调用已替换为 strings.Split 路径解析（GetDatabaseStatsHandler, GetSlowQueriesHandler, GetQueryStatsHandler）
- [x] metrics.go 中 connection_string 查询已移除，改用 manager.GetPool() 直接执行
- [x] http_ext.go 中所有 PathValue("id") 调用已替换为 strings.Split 路径解析（GetRequest, PostRequest）
- [x] demo/run-demo.sh 中 RAG 文档列表和查询 URL 已修正为 FastAPI 实际路由路径
- [x] demo/run-demo.sh 中 RAG Token 提取逻辑已修复（处理 json.RawMessage 格式）
- [x] demo/run-demo.sh 中 RAG 回退 Token 从 demo-token 改为主 TOKEN
- [x] anonymous_accounts 表已添加 updated_at 列（运行数据库 + 迁移脚本）
- [x] fs9_files 表已添加 storage_key 列（运行数据库）
- [x] 匿名注册流程可正常执行并返回有效 JWT Token
- [x] README.md Web 框架描述已从 Gin 修正为 net/http 标准库
- [x] README.md docker-compose 命令已修正为 docker compose（新版格式）
- [x] README.md 环境变量表格已按实际 config.go 重写
- [x] README.md 配置文件路径和示例已修正
- [x] README.md 不存在的 CLI 命令已移除
- [x] Docker 重新构建成功，无编译错误
- [x] run-demo.sh 全部 13 个步骤成功完成（核心功能全部正常）

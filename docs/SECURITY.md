# Open-DB9 安全指南

本文档描述 Open-DB9 的安全特性、最佳实践和建议配置。

---

## 🔒 安全特性概述

Open-DB9 实现了多层安全防护，达到 **安全评级 A**：

| 特性 | 状态 | 描述 |
|------|------|------|
| JWT 认证 | ✅ | Bearer Token 认证所有 API 端点 |
| 速率限制 | ✅ | 100 请求/分钟，防止滥用 |
| CORS 策略 | ✅ | 可配置的跨域资源共享 |
| 凭证加密 | ✅ | AES-256-GCM 加密存储 |
| SQL 注入防护 | ✅ | 参数化查询 + 白名单验证 |
| 输入验证 | ✅ | 全面的输入验证和大小限制 |
| 密码哈希 | ✅ | bcrypt 哈希算法 |
| 敏感信息脱敏 | ✅ | 日志和错误消息脱敏 |

---

## 🔐 认证与授权

### JWT 认证

Open-DB9 使用 JWT (JSON Web Token) 进行 API 认证。

#### 配置

```yaml
jwt:
  secret: "${JWT_SECRET}"    # 最少 32 字符
  expires_in: 3600            # Token 有效期（秒）
  issuer: "open-db9"          # 发行者标识
```

#### 安全要求

- **密钥长度**: 最少 32 字符
- **签名算法**: HS256 (HMAC-SHA256)
- **Token 有效期**: 默认 1 小时，可配置

#### 最佳实践

```bash
# 生成强密钥
openssl rand -base64 32

# 或使用
head -c 32 /dev/urandom | base64
```

---

### 用户角色

| 角色 | 权限 |
|------|------|
| `admin` | 完全访问权限，包括用户管理 |
| `user` | 标准 API 访问权限 |

---

## 🚦 速率限制

### 配置

```yaml
rate_limit:
  enabled: true
  max_requests: 100      # 每分钟请求数
  window: 60             # 时间窗口（秒）
  cleanup_interval: 300   # 清理间隔（秒）
```

### 速率限制头

```http
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1617200000
```

### 最佳实践

- **生产环境**: 启用速率限制
- **公共 API**: 降低限制（如 60 请求/分钟）
- **内部 API**: 提高限制或禁用

---

## 🌐 CORS 策略

### 默认配置（开发）

```yaml
cors:
  allowed_origins: ["*"]
  allowed_methods: ["GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"]
  allowed_headers: ["Content-Type", "Authorization", "X-Requested-With"]
  allow_credentials: false
  max_age: 86400
```

### 生产配置

```yaml
cors:
  allowed_origins:
    - "https://example.com"
    - "https://app.example.com"
  allowed_methods:
    - "GET"
    - "POST"
    - "PUT"
    - "DELETE"
  allowed_headers:
    - "Content-Type"
    - "Authorization"
  allow_credentials: true
  max_age: 3600
```

### 安全建议

- ❌ **避免**: `allowed_origins: ["*"]` + `allow_credentials: true`
- ✅ **推荐**: 明确指定允许的源
- ✅ **推荐**: 生产环境启用 `allow_credentials` 时限制源

---

## 🔐 凭证存储

### 加密存储

用户凭证使用 **AES-256-GCM** 加密存储：

- **算法**: AES-256-GCM
- **密钥派生**: SHA-256 从主密钥派生
- **存储位置**: `~/.config/db9/credentials.sec`
- **文件权限**: 0600（仅所有者可读写）

### 主密钥

主密钥从以下来源获取（按优先级）：

1. 环境变量 `DB9_MASTER_KEY`
2. 从用户主目录派生（备用方案）

```bash
# 设置主密钥
export DB9_MASTER_KEY="your-very-secure-master-key-here"

# 或使用密钥管理服务
export DB9_MASTER_KEY=$(aws secretsmanager get-secret-value --secret-id db9/master-key --query SecretString --output text)
```

---

## 🛡️ SQL 注入防护

### 防护措施

1. **参数化查询**: 所有 SQL 查询使用参数化
2. **白名单验证**: 仅允许安全的 SQL 命令
3. **查询大小限制**: 最大 100KB

### SQL 白名单

```go
allowedCommands := map[string]bool{
    "SELECT":  true,
    "SHOW":    true,
    "DESCRIBE": true,
    "EXPLAIN":  true,
}
```

### 查询验证

```go
// 验证查询大小
if len(query) > 100*1024 { // 100KB
    return errors.New("query too large")
}

// 验证命令类型
if !isAllowedCommand(query) {
    return errors.New("command not allowed")
}
```

---

## 📏 输入验证

### 请求大小限制

| 端点 | 最大大小 |
|------|----------|
| SQL 查询 | 100KB |
| 文件上传 | 100MB |
| JSON 请求 | 1MB |

### 路径验证

文件路径验证防止路径遍历攻击：

```go
// 清理路径
filepath.Clean(path)

// 验证路径在允许的目录内
filepath.IsAbs(path)
strings.Contains(path, "..")
```

### 敏感信息脱敏

日志和错误消息中脱敏敏感信息：

- 密码
- Token
- API 密钥
- 数据库连接字符串

---

## 🔒 密码安全

### 密码哈希

用户密码使用 **bcrypt** 哈希：

```go
// 成本因子
bcrypt.DefaultCost = 10

// 哈希密码
hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

// 验证密码
err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
```

### 密码策略

建议在生产环境实施：

- 最小长度: 8 字符
- 包含大写字母
- 包含小写字母
- 包含数字
- 包含特殊字符

---

## 🚢 生产安全检查清单

### 部署前

- [ ] 使用强 JWT 密钥（最少 32 字符）
- [ ] 设置强主密钥用于凭证加密
- [ ] 配置适当的 CORS 策略
- [ ] 启用速率限制
- [ ] 配置 HTTPS/TLS
- [ ] 设置防火墙规则
- [ ] 启用日志记录
- [ ] 配置监控告警

### 运行时

- [ ] 定期轮换密钥
- [ ] 监控异常访问模式
- [ ] 定期审计访问日志
- [ ] 更新依赖包
- [ ] 备份数据库

---

## 🚨 安全事件响应

### 发现安全漏洞

如果您发现安全漏洞，请：

1. **不要**公开 issue
2. **发送邮件至**: security@open-db9.org
3. **包含详情**:
   - 漏洞描述
   - 影响范围
   - 复现步骤
   - 建议修复方案

### 响应时间

- **确认**: 2 个工作日内
- **修复评估**: 5 个工作日内
- **安全补丁**: 根据严重程度，在合理时间内发布

---

## 📋 安全审计

### 已通过的安全审计

| 审计项 | 状态 | 备注 |
|--------|------|------|
| SQL 注入防护 | ✅ 通过 | 参数化查询 + 白名单 |
| 路径遍历防护 | ✅ 通过 | 路径验证和清理 |
| 命令注入防护 | ✅ 通过 | 输入验证和转义 |
| 文件上传安全 | ✅ 通过 | 大小限制 + 类型检查 |
| 密码存储 | ✅ 通过 | bcrypt 哈希 |
| 敏感信息脱敏 | ✅ 通过 | 日志脱敏 |

### 安全评级

**总体评级**: ⭐⭐⭐⭐⭐ **A 级**

- **认证**: A
- **授权**: A
- **输入验证**: A
- **数据保护**: A
- **日志审计**: B+

---

## 📖 参考资料

- [OWASP Top 10](https://owasp.org/www-project-top-ten/)
- [OWASP API Security Top 10](https://owasp.org/www-project-api-security/)
- [CWE-89: SQL Injection](https://cwe.mitre.org/data/definitions/89.html)
- [JWT Best Practices](https://tools.ietf.org/html/rfc8725)

---

**文档版本**: 1.0
**最后更新**: 2026-03-31
**安全联系人**: security@open-db9.org

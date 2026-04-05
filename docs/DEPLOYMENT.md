# Open-DB9 部署指南

本文档提供 Open-DB9 的完整部署指南，包括开发、测试和生产环境。

---

## 📋 部署前检查清单

### 系统要求

| 组件 | 最低要求 | 推荐配置 |
|------|----------|----------|
| **操作系统** | Linux 4.x+ | Ubuntu 22.04 LTS |
| **CPU** | 2 核 | 4+ 核 |
| **内存** | 2 GB | 4+ GB |
| **磁盘** | 20 GB | 50+ GB SSD |
| **Go** | 1.25.0 | 最新稳定版 |
| **PostgreSQL** | 12+ | 15+ |

### 网络要求

- **API 服务器端口**: 8080（可配置）
- **FS9 服务端口**: 9090（可配置）
- **PostgreSQL 端口**: 5432
- **防火墙**: 允许入站连接到上述端口

---

## 🐳 Docker 部署

### 快速启动

```bash
# 使用 Docker Compose 启动所有服务
docker-compose up -d

# 查看日志
docker-compose logs -f

# 停止服务
docker-compose down
```

### Docker Compose 配置

```yaml
version: '3.8'

services:
  postgres:
    image: postgres:15-alpine
    container_name: db9-postgres
    environment:
      POSTGRES_DB: db9
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: ${DB9_DB_PASSWORD}
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./migrations/control:/docker-entrypoint-initdb.d
    restart: unless-stopped

  fs9-service:
    image: open-db9/fs9-service:latest
    container_name: db9-fs9
    ports:
      - "9090:9090"
    volumes:
      - fs9_data:/data/storage
    restart: unless-stopped

  api-server:
    image: open-db9/server:latest
    container_name: db9-api
    ports:
      - "8080:8080"
    environment:
      DB9_DB_HOST: postgres
      DB9_DB_PORT: 5432
      DB9_DB_USER: postgres
      DB9_DB_PASSWORD: ${DB9_DB_PASSWORD}
      DB9_DB_NAME: db9
      DB9_JWT_SECRET: ${DB9_JWT_SECRET}
      DB9_FS9_URL: http://fs9-service:9090
      DB9_MASTER_KEY: ${DB9_MASTER_KEY}
    depends_on:
      - postgres
      - fs9-service
    restart: unless-stopped

volumes:
  postgres_data:
  fs9_data:
```

---

## ☸️ Kubernetes 部署

### 命名空间

```bash
kubectl create namespace db9
```

### ConfigMap

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: db9-config
  namespace: db9
data:
  DB9_DB_HOST: "postgres-service"
  DB9_DB_PORT: "5432"
  DB9_DB_NAME: "db9"
  DB9_FS9_URL: "http://fs9-service:9090"
```

### Secret

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: db9-secrets
  namespace: db9
type: Opaque
stringData:
  DB9_DB_PASSWORD: "your-secure-password"
  DB9_JWT_SECRET: "your-very-secure-jwt-secret-at-least-32-chars"
  DB9_MASTER_KEY: "your-secure-master-key"
```

### PostgreSQL 部署

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: postgres
  namespace: db9
spec:
  replicas: 1
  selector:
    matchLabels:
      app: postgres
  template:
    metadata:
      labels:
        app: postgres
    spec:
      containers:
      - name: postgres
        image: postgres:15-alpine
        ports:
        - containerPort: 5432
        env:
        - name: POSTGRES_DB
          value: "db9"
        - name: POSTGRES_USER
          value: "postgres"
        - name: POSTGRES_PASSWORD
          valueFrom:
            secretKeyRef:
              name: db9-secrets
              key: DB9_DB_PASSWORD
        volumeMounts:
        - name: postgres-storage
          mountPath: /var/lib/postgresql/data
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
      volumes:
      - name: postgres-storage
        persistentVolumeClaim:
          claimName: postgres-pvc
---
apiVersion: v1
kind: Service
metadata:
  name: postgres-service
  namespace: db9
spec:
  selector:
    app: postgres
  ports:
  - port: 5432
    targetPort: 5432
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: postgres-pvc
  namespace: db9
spec:
  accessModes:
  - ReadWriteOnce
  resources:
    requests:
      storage: 10Gi
```

### API 服务器部署

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: api-server
  namespace: db9
spec:
  replicas: 2
  selector:
    matchLabels:
      app: api-server
  template:
    metadata:
      labels:
        app: api-server
    spec:
      containers:
      - name: api-server
        image: open-db9/server:latest
        ports:
        - containerPort: 8080
        envFrom:
        - configMapRefRef:
            name: db9-config
        - secretRef:
            name: db9-secrets
        resources:
          requests:
            memory: "128Mi"
            cpu: "100m"
          limits:
            memory: "256Mi"
            cpu: "200m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 30
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 10
---
apiVersion: v1
kind: Service
metadata:
  name: api-server
  namespace: db9
spec:
  selector:
    app: api-server
  ports:
  - port: 8080
    targetPort: 8080
  type: LoadBalancer
```

### Ingress

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: db9-ingress
  namespace: db9
  annotations:
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
spec:
  tls:
  - hosts:
    - api.db9.example.com
    secretName: db9-tls
  rules:
  - host: api.db9.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: api-server
            port:
              number: 8080
```

---

## 🖥️ 传统部署

### 系统服务 (Systemd)

#### 创建服务文件

```bash
sudo nano /etc/systemd/system/db9-api.service
```

```ini
[Unit]
Description=Open-DB9 API Server
After=network.target postgresql.service

[Service]
Type=simple
User=db9
Group=db9
WorkingDirectory=/opt/db9
Environment="DB9_DB_HOST=localhost"
Environment="DB9_DB_PORT=5432"
Environment="DB9_DB_NAME=db9"
EnvironmentFile=/opt/db9/.env
ExecStart=/opt/db9/server
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

#### FS9 服务

```bash
sudo nano /etc/systemd/system/db9-fs9.service
```

```ini
[Unit]
Description=Open-DB9 File Storage Service
After=network.target

[Service]
Type=simple
User=db9
Group=db9
WorkingDirectory=/opt/db9
ExecStart=/opt/db9/fs9-service
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

#### 启用服务

```bash
# 重载 systemd
sudo systemctl daemon-reload

# 启动服务
sudo systemctl start db9-fs9
sudo systemctl start db9-api

# 启用开机自启
sudo systemctl enable db9-fs9
sudo systemctl enable db9-api

# 查看状态
sudo systemctl status db9-api
```

---

## 🔧 环境配置

### 生产环境变量

```bash
# /etc/db9/environment
export DB9_HOST="0.0.0.0"
export DB9_PORT="8080"

# 数据库配置
export DB9_DB_HOST="localhost"
export DB9_DB_PORT="5432"
export DB9_DB_USER="postgres"
export DB9_DB_PASSWORD="<secure-password>"
export DB9_DB_NAME="db9"

# 安全配置
export DB9_JWT_SECRET="<your-very-secure-jwt-secret-at-least-32-chars>"
export DB9_MASTER_KEY="<your-secure-master-key>"

# FS9 服务
export DB9_FS9_URL="http://localhost:9090"

# 日志级别
export DB9_LOG_LEVEL="info"
```

---

## 🔄 数据库迁移

### 运行迁移

```bash
# 手动运行迁移
db9 migrate up

# 或使用 API
curl -X POST http://localhost:8080/api/v1/migrations/up \
  -H "Authorization: Bearer <token>"
```

### 迁移文件

迁移文件位于 `migrations/control/`:

```
001_init_schema.up.sql
002_migrations.up.sql
003_fs9_metadata.up.sql
004_observability.up.sql
005_fs9_metadata.up.sql
006_add_branch_support.up.sql
007_connection_pools.up.sql
```

---

## 📊 监控和日志

### 日志配置

```yaml
logging:
  level: "info"        # debug, info, warn, error
  format: "json"       # json, text
  output: "stdout"     # stdout, file
  file:
    path: "/var/log/db9/api.log"
    max_size: 100      # MB
    max_backups: 10
    max_age: 30        # days
    compress: true
```

### Prometheus 指标

```yaml
# 在配置中启用 Prometheus
metrics:
  enabled: true
  path: "/metrics"
  port: 9091
```

### 健康检查

```bash
# 基本健康检查
curl http://localhost:8080/health

# 详细健康检查
curl http://localhost:8080/health?verbose=true
```

---

## 🔒 生产安全配置

### TLS/HTTPS

使用 Nginx 反向代理：

```nginx
server {
    listen 443 ssl http2;
    server_name api.db9.example.com;

    ssl_certificate /etc/ssl/certs/api.db9.example.com.crt;
    ssl_certificate_key /etc/ssl/private/api.db9.example.com.key;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}

server {
    listen 80;
    server_name api.db9.example.com;
    return 301 https://$server_name$request_uri;
}
```

### 防火墙配置

```bash
# UFW (Ubuntu)
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw allow from 10.0.0.0/8 to any port 8080
sudo ufw enable

# firewalld (CentOS/RHEL)
sudo firewall-cmd --permanent --add-port=8080/tcp
sudo firewall-cmd --permanent --add-port=9090/tcp
sudo firewall-cmd --reload
```

---

## 🚀 性能优化

### 连接池配置

```yaml
database:
  pool:
    max_connections: 25
    min_connections: 5
    max_conn_lifetime: 3600    # 秒
    max_conn_idle_time: 300     # 秒
    health_check_period: 60     # 秒
```

### 缓存配置

```yaml
cache:
  enabled: true
  ttl: 300              # 秒
  max_size: 1000        # 条目
```

### 速率限制

```yaml
rate_limit:
  enabled: true
  max_requests: 100
  window: 60
```

---

## 🔄 高可用部署

### 负载均衡

使用 Nginx 负载均衡多个 API 服务器实例：

```nginx
upstream db9_api {
    least_conn;
    server api-server-1:8080 max_fails=3 fail_timeout=30s;
    server api-server-2:8080 max_fails=3 fail_timeout=30s;
    server api-server-3:8080 max_fails=3 fail_timeout=30s;
}

server {
    listen 443 ssl;
    server_name api.db9.example.com;

    location / {
        proxy_pass http://db9_api;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }
}
```

### PostgreSQL 高可用

使用 PostgreSQL 主从复制或 Patroni：

```yaml
# docker-compose-ha.yml
services:
  postgres-primary:
    image: postgres:15-alpine
    environment:
      POSTGRES_REPLICATION_MODE: master
      POSTGRES_REPLICATION_USER: replicator
      POSTGRES_REPLICATION_PASSWORD: <password>
    volumes:
      - postgres_primary_data:/var/lib/postgresql/data

  postgres-standby:
    image: postgres:15-alpine
    environment:
      POSTGRES_REPLICATION_MODE: slave
      POSTGRES_REPLICATION_HOST: postgres-primary
      POSTGRES_REPLICATION_USER: replicator
      POSTGRES_REPLICATION_PASSWORD: <password>
    depends_on:
      - postgres-primary
```

---

## 🧪 测试部署

### 健康检查测试

```bash
# 测试所有服务
curl http://localhost:8080/health
curl http://localhost:9090/health

# 测试 API
curl -X POST http://localhost:8080/api/v1/users/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"test"}'

# 测试数据库连接
curl -X POST http://localhost:8080/api/v1/databases/1/sql \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"query":"SELECT version()"}'
```

### 性能测试

```bash
# 使用 Apache Bench
ab -n 1000 -c 10 http://localhost:8080/health

# 使用 wrk
wrk -t12 -c400 -d30s http://localhost:8080/health
```

---

## 🐛 故障排除

### 常见问题

#### 1. 服务无法启动

```bash
# 检查日志
journalctl -u db9-api -n 50 --no-pager

# 检查端口占用
sudo netstat -tulpn | grep :8080
```

#### 2. 数据库连接失败

```bash
# 测试数据库连接
psql -h localhost -U postgres -d db9

# 检查 PostgreSQL 状态
sudo systemctl status postgresql
```

#### 3. 权限错误

```bash
# 检查文件权限
ls -la /opt/db9

# 修复权限
sudo chown -R db9:db9 /opt/db9
sudo chmod -R 755 /opt/db9
```

---

## 📚 参考资料

- [Docker 部署](https://docs.docker.com/engine/deploy/)
- [Kubernetes 部署](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/)
- [Systemd 服务](https://www.freedesktop.org/software/systemd/man/systemd.service.html)
- [Nginx 反向代理](https://nginx.org/en/docs/http/ngx_http_proxy_module.html)

---

**文档版本**: 1.0
**最后更新**: 2026-03-31

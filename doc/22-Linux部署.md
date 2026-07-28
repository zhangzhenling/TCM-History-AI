# 22 Linux 原生部署指南

本章提供 TCM-History-AI 在 Linux 服务器（以 Ubuntu 22.04 LTS 为例）上的原生部署方案，不依赖 Docker，适用于以下场景：
- 服务器资源有限，无法运行容器化基础设施
- 需要直接管理系统进程，便于与现有运维体系集成
- 开发调试阶段需要直接访问各组件日志和端口
- 政务、金融等对内网物理机部署有硬性要求的场景

**建议**：如无特殊限制，优先使用 [第十五章 Docker 部署](./15-Docker部署.md) 或 [第十六章 Kubernetes 部署](./16-Kubernetes部署.md)，容器化方案在环境一致性、依赖隔离、升级回滚方面更具优势。

---

## 1 系统要求

| 组件 | 最低配置 | 推荐配置 | 说明 |
|------|---------|---------|------|
| OS | Ubuntu 20.04 | Ubuntu 22.04 LTS | 其他发行版（CentOS 8/Debian 12）命令略有差异 |
| CPU | 4 核 | 8 核 | AI Service 与 Knowledge Service 为 CPU 密集型 |
| 内存 | 8 GB | 16 GB | Neo4j 与 Milvus 占用较大 |
| 磁盘 | 100 GB SSD | 200 GB SSD | 包含数据库、向量索引、对象存储 |
| 网络 | 内网可访问 | 公网带宽 >= 10Mbps | 如需调用外部 LLM API |

所有命令默认以 `root` 执行，或前置 `sudo`。

---

## 2 基础环境安装

### 2.1 系统依赖

```bash
apt update && apt upgrade -y
apt install -y git curl wget build-essential cmake pkg-config \
    libssl-dev zlib1g-dev libbz2-dev libreadline-dev libsqlite3-dev \
    llvm libncurses5-dev libncursesw5-dev xz-utils tk-dev libffi-dev \
    liblzma-dev python3-openssl nginx gettext-base jq
```

### 2.2 Go 1.22+

```bash
GO_VERSION=1.22.5
wget https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz
rm -rf /usr/local/go && tar -C /usr/local -xzf go${GO_VERSION}.linux-amd64.tar.gz
ln -sf /usr/local/go/bin/go /usr/local/bin/go
ln -sf /usr/local/go/bin/gofmt /usr/local/bin/gofmt

go version  # go version go1.22.5 linux/amd64

# 配置 Go 环境变量（写入 /etc/profile.d）
cat > /etc/profile.d/go.sh << 'EOF'
export GOPATH=/opt/go
export GOBIN=$GOPATH/bin
export PATH=$PATH:/usr/local/go/bin:$GOBIN
export GOPROXY=https://goproxy.cn,direct
export GO111MODULE=on
EOF
chmod +x /etc/profile.d/go.sh
source /etc/profile.d/go.sh
```

国内环境建议配置 `GOPROXY` 加速依赖下载。

### 2.3 Node.js 20 + pnpm 9

```bash
curl -fsSL https://deb.nodesource.com/setup_20.x | bash -
apt install -y nodejs
node -v  # v20.x

npm install -g pnpm@9
pnpm -v  # 9.x
```

### 2.4 其他工具

```bash
# golangci-lint（后端代码检查）
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b /usr/local/bin v1.59.1

# Wire（依赖注入代码生成）
go install github.com/google/wire/cmd/wire@latest

# migrate（数据库迁移，可选，也可用项目内置 Flyway）
curl -L https://github.com/golang-migrate/migrate/releases/download/v4.17.0/migrate.linux-amd64.tar.gz | tar xvz -C /usr/local/bin migrate
```

---

## 3 数据库与中间件安装

以下组件全部安装在单机上，通过不同端口区分。生产环境应分散到不同节点并启用主从/集群。

### 3.1 PostgreSQL 16

```bash
# 添加官方仓库
apt install -y postgresql-common
/usr/share/postgresql-common/pgdg/apt.postgresql.org.sh -y
apt install -y postgresql-16 postgresql-client-16 postgresql-contrib-16

# 启动并设置开机自启
systemctl enable postgresql
systemctl start postgresql

# 创建数据库与用户
su - postgres -c "psql -c \"CREATE USER tcm WITH PASSWORD 'tcm_pass';\""
su - postgres -c "psql -c \"CREATE DATABASE tcm_history OWNER tcm;\""
su - postgres -c "psql -c \"GRANT ALL PRIVILEGES ON DATABASE tcm_history TO tcm;\""
su - postgres -c "psql -d tcm_history -c \"CREATE EXTENSION IF NOT EXISTS pg_trgm;\""

# 允许远程/本地连接（按需）
sed -i "s/#listen_addresses = 'localhost'/listen_addresses = '*'/" /etc/postgresql/16/main/postgresql.conf
echo "host  all  all  127.0.0.1/32  scram-sha-256" >> /etc/postgresql/16/main/pg_hba.conf
systemctl restart postgresql
```

### 3.2 Redis 7

```bash
apt install -y redis-server
sed -i 's/^# requirepass .*/requirepass redis_pass/' /etc/redis/redis.conf
sed -i 's/^appendonly no/appendonly yes/' /etc/redis/redis.conf
systemctl enable redis-server
systemctl restart redis-server
redis-cli -a redis_pass ping  # PONG
```

### 3.3 Neo4j 5

```bash
# 安装 Java 17
apt install -y openjdk-17-jre-headless

# 添加 Neo4j 仓库
wget -O - https://debian.neo4j.com/neotechnology.gpg.key | apt-key add -
echo 'deb https://debian.neo4j.com stable 5' | tee /etc/apt/sources.list.d/neo4j.list
apt update
apt install -y neo4j=1:5.19.0

# 配置认证与内存
sed -i 's/#dbms.security.auth_enabled=true/dbms.security.auth_enabled=true/' /etc/neo4j/neo4j.conf
sed -i 's/#server.memory.heap.initial_size=.*/server.memory.heap.initial_size=512m/' /etc/neo4j/neo4j.conf
sed -i 's/#server.memory.heap.max_size=.*/server.memory.heap.max_size=1G/' /etc/neo4j/neo4j.conf
sed -i 's/#server.memory.pagecache.size=.*/server.memory.pagecache.size=512m/' /etc/neo4j/neo4j.conf

# 设置初始密码
echo "neo4j:neo4j_pass" > /var/lib/neo4j/conf/initial_password
cystemctl enable neo4j
systemctl restart neo4j

# 安装 APOC 插件
wget -P /var/lib/neo4j/plugins https://github.com/neo4j/apoc/releases/download/5.19.0/apoc-5.19.0-core.jar
systemctl restart neo4j
```

### 3.4 MinIO

```bash
wget https://dl.min.io/server/minio/release/linux-amd64/minio
chmod +x minio
mv minio /usr/local/bin/

# 创建用户与数据目录
useradd -r -s /bin/false minio
mkdir -p /opt/minio/data
chown -R minio:minio /opt/minio

# systemd 服务
cat > /etc/systemd/system/minio.service << 'EOF'
[Unit]
Description=MinIO Object Storage
After=network.target

[Service]
Type=simple
User=minio
Group=minio
Environment="MINIO_ROOT_USER=minio"
Environment="MINIO_ROOT_PASSWORD=minio_pass"
ExecStart=/usr/local/bin/minio server /opt/minio/data --console-address :9001
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable minio
systemctl start minio
```

创建 bucket：
```bash
wget https://dl.min.io/client/mc/release/linux-amd64/mc
chmod +x mc && mv mc /usr/local/bin/
mc alias set local http://127.0.0.1:9000 minio minio_pass
mc mb local/tcm-assets
mc mb local/tcm-milvus
```

### 3.5 Milvus 2.4（Standalone 模式）

```bash
# 安装 etcd（Milvus 依赖）
ETCD_VER=v3.5.5
wget https://github.com/etcd-io/etcd/releases/download/${ETCD_VER}/etcd-${ETCD_VER}-linux-amd64.tar.gz
tar xzvf etcd-${ETCD_VER}-linux-amd64.tar.gz -C /usr/local/bin --strip-components=1 etcd-${ETCD_VER}-linux-amd64/etcd etcd-${ETCD_VER}-linux-amd64/etcdctl
rm etcd-${ETCD_VER}-linux-amd64.tar.gz

# etcd systemd 服务
cat > /etc/systemd/system/etcd.service << 'EOF'
[Unit]
Description=etcd key-value store
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/etcd \
  --listen-client-urls http://127.0.0.1:2379 \
  --advertise-client-urls http://127.0.0.1:2379 \
  --data-dir /var/lib/etcd
Restart=always

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable etcd
systemctl start etcd

# 安装 Milvus Standalone
wget https://github.com/milvus-io/milvus/releases/download/v2.4.0/milvus-standalone-docker-compose.yml -O /opt/milvus/docker-compose.yml
# 或者直接下载二进制（推荐不依赖 Docker 的方式）
```

> **注意**：Milvus Standalone 官方提供 Docker Compose 方式最为稳定。如果必须完全脱离 Docker，可参考 Milvus 源码编译部署，但复杂度较高。建议此处保留 Docker 仅用于 Milvus，或改用 [Faiss](https://github.com/facebookresearch/faiss) 作为纯 CPU 向量检索替代方案（需修改 Knowledge Service 代码适配）。

### 3.6 Meilisearch

```bash
wget https://github.com/meilisearch/meilisearch/releases/download/v1.6.0/meilisearch-linux-amd64
chmod +x meilisearch-linux-amd64
mv meilisearch-linux-amd64 /usr/local/bin/meilisearch

# systemd 服务
cat > /etc/systemd/system/meilisearch.service << 'EOF'
[Unit]
Description=Meilisearch Search Engine
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/meilisearch --master-key meili_master_key_dev --http-addr 127.0.0.1:7700
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable meilisearch
systemctl start meilisearch
```

### 3.7 RabbitMQ 3.13

```bash
apt install -y rabbitmq-server
rabbitmq-plugins enable rabbitmq_management

# 创建用户
rabbitmqctl add_user tcm tcm_pass
rabbitmqctl set_user_tags tcm administrator
rabbitmqctl add_vhost /tcm
rabbitmqctl set_permissions -p /tcm tcm ".*" ".*" ".*"

systemctl enable rabbitmq-server
systemctl restart rabbitmq-server
```

### 3.8 端口汇总

| 服务 | 端口 | 说明 |
|------|------|------|
| PostgreSQL | 5432 | 关系数据库 |
| Redis | 6379 | 缓存/会话 |
| Neo4j HTTP | 7474 | 图数据库浏览器 |
| Neo4j Bolt | 7687 | 图数据库驱动连接 |
| etcd | 2379 | Milvus 元数据 |
| MinIO API | 9000 | 对象存储 S3 API |
| MinIO Console | 9001 | 管理控制台 |
| Milvus | 19530 | 向量数据库 gRPC |
| Meilisearch | 7700 | 全文检索 |
| RabbitMQ AMQP | 5672 | 消息队列 |
| RabbitMQ Mgmt | 15672 | 管理控制台 |

---

## 4 后端编译与配置

### 4.1 获取代码

```bash
mkdir -p /opt/tcm-history-ai
cd /opt/tcm-history-ai
git clone https://github.com/zhangzhenling/TCM-History-AI.git .
```

### 4.2 编译全部服务

```bash
cd /opt/tcm-history-ai/backend
source /etc/profile.d/go.sh
make build
```

产物输出到 `backend/bin/` 目录：
```
bin/
├── gateway
├── user-service
├── history-service
├── knowledge-service
├── graph-service
├── ai-service
└── learning-service
```

### 4.3 配置文件

每个服务通过环境变量或 `configs/` 目录下的 YAML 文件加载配置。创建统一的环境变量文件 `/opt/tcm-history-ai/.env`：

```bash
cat > /opt/tcm-history-ai/.env << 'EOF'
# ===== 数据库 =====
DB_HOST=127.0.0.1
DB_PORT=5432
DB_USER=tcm
DB_PASSWORD=tcm_pass
DB_NAME=tcm_history
DB_SSLMODE=disable

# ===== Redis =====
REDIS_HOST=127.0.0.1
REDIS_PORT=6379
REDIS_PASSWORD=redis_pass
REDIS_DB=0

# ===== Neo4j =====
NEO4J_URI=bolt://127.0.0.1:7687
NEO4J_USER=neo4j
NEO4J_PASSWORD=neo4j_pass

# ===== MinIO =====
MINIO_ENDPOINT=127.0.0.1:9000
MINIO_ACCESS_KEY=minio
MINIO_SECRET_KEY=minio_pass
MINIO_USE_SSL=false
MINIO_BUCKET_TCM=tcm-assets
MINIO_BUCKET_MILVUS=tcm-milvus

# ===== Milvus =====
MILVUS_HOST=127.0.0.1
MILVUS_PORT=19530

# ===== Meilisearch =====
MEILISEARCH_HOST=http://127.0.0.1:7700
MEILISEARCH_API_KEY=meili_master_key_dev

# ===== RabbitMQ =====
RABBITMQ_HOST=127.0.0.1
RABBITMQ_PORT=5672
RABBITMQ_USER=tcm
RABBITMQ_PASSWORD=tcm_pass
RABBITMQ_VHOST=/tcm

# ===== 应用通用 =====
APP_ENV=production
LOG_LEVEL=info
LOG_FORMAT=json
JWT_SECRET=CHANGE_ME_TO_A_LONG_RANDOM_STRING_AT_LEAST_32_CHARS

# ===== LLM（按需配置） =====
LLM_ENABLED=false
LLM_PROVIDER=openai
LLM_API_KEY=sk-xxx
LLM_BASE_URL=https://api.openai.com/v1
LLM_MODEL=gpt-4o-mini

# ===== 服务端口 =====
GATEWAY_HTTP_PORT=8080
USER_SERVICE_RPC_PORT=9001
HISTORY_SERVICE_RPC_PORT=9002
KNOWLEDGE_SERVICE_RPC_PORT=9003
GRAPH_SERVICE_RPC_PORT=9004
AI_SERVICE_RPC_PORT=9005
LEARNING_SERVICE_RPC_PORT=9006
EOF
```

> **安全提醒**：生产环境务必将 `JWT_SECRET`、`LLM_API_KEY`、`DB_PASSWORD` 等敏感值替换为强密码，且该 `.env` 文件权限设为 `600`。

### 4.4 数据库迁移

```bash
cd /opt/tcm-history-ai/backend

# 方式一：使用 golang-migrate（如果有 migrate 文件）
# migrate -path ./history-service/migrations -database "postgres://tcm:tcm_pass@127.0.0.1:5432/tcm_history?sslmode=disable" up

# 方式二：各服务首次启动时自动执行 Flyway/GORM AutoMigrate
# 本项目的 history-service、user-service 等内置了 migration，启动时自动建表
```

---

## 5 使用 systemd 管理后端服务

为每个微服务创建 systemd 服务单元，实现开机自启、自动重启、日志统一收集。

### 5.1 创建通用服务模板

```bash
cat > /etc/systemd/system/tcm-gateway.service << 'EOF'
[Unit]
Description=TCM History AI - Gateway
After=network.target postgresql.service redis.service
Wants=postgresql.service redis.service

[Service]
Type=simple
User=www-data
Group=www-data
WorkingDirectory=/opt/tcm-history-ai/backend
EnvironmentFile=/opt/tcm-history-ai/.env
ExecStart=/opt/tcm-history-ai/backend/bin/gateway
Restart=always
RestartSec=5
StandardOutput=append:/var/log/tcm/gateway.log
StandardError=append:/var/log/tcm/gateway.log

[Install]
WantedBy=multi-user.target
EOF

cat > /etc/systemd/system/tcm-user-service.service << 'EOF'
[Unit]
Description=TCM History AI - User Service
After=network.target postgresql.service redis.service

[Service]
Type=simple
User=www-data
Group=www-data
WorkingDirectory=/opt/tcm-history-ai/backend
EnvironmentFile=/opt/tcm-history-ai/.env
ExecStart=/opt/tcm-history-ai/backend/bin/user-service
Restart=always
RestartSec=5
StandardOutput=append:/var/log/tcm/user-service.log
StandardError=append:/var/log/tcm/user-service.log

[Install]
WantedBy=multi-user.target
EOF

cat > /etc/systemd/system/tcm-history-service.service << 'EOF'
[Unit]
Description=TCM History AI - History Service
After=network.target postgresql.service

[Service]
Type=simple
User=www-data
Group=www-data
WorkingDirectory=/opt/tcm-history-ai/backend
EnvironmentFile=/opt/tcm-history-ai/.env
ExecStart=/opt/tcm-history-ai/backend/bin/history-service
Restart=always
RestartSec=5
StandardOutput=append:/var/log/tcm/history-service.log
StandardError=append:/var/log/tcm/history-service.log

[Install]
WantedBy=multi-user.target
EOF

cat > /etc/systemd/system/tcm-knowledge-service.service << 'EOF'
[Unit]
Description=TCM History AI - Knowledge Service
After=network.target postgresql.service

[Service]
Type=simple
User=www-data
Group=www-data
WorkingDirectory=/opt/tcm-history-ai/backend
EnvironmentFile=/opt/tcm-history-ai/.env
ExecStart=/opt/tcm-history-ai/backend/bin/knowledge-service
Restart=always
RestartSec=5
StandardOutput=append:/var/log/tcm/knowledge-service.log
StandardError=append:/var/log/tcm/knowledge-service.log

[Install]
WantedBy=multi-user.target
EOF

cat > /etc/systemd/system/tcm-graph-service.service << 'EOF'
[Unit]
Description=TCM History AI - Graph Service
After=network.target neo4j.service

[Service]
Type=simple
User=www-data
Group=www-data
WorkingDirectory=/opt/tcm-history-ai/backend
EnvironmentFile=/opt/tcm-history-ai/.env
ExecStart=/opt/tcm-history-ai/backend/bin/graph-service
Restart=always
RestartSec=5
StandardOutput=append:/var/log/tcm/graph-service.log
StandardError=append:/var/log/tcm/graph-service.log

[Install]
WantedBy=multi-user.target
EOF

cat > /etc/systemd/system/tcm-ai-service.service << 'EOF'
[Unit]
Description=TCM History AI - AI Service
After=network.target redis.service

[Service]
Type=simple
User=www-data
Group=www-data
WorkingDirectory=/opt/tcm-history-ai/backend
EnvironmentFile=/opt/tcm-history-ai/.env
ExecStart=/opt/tcm-history-ai/backend/bin/ai-service
Restart=always
RestartSec=5
StandardOutput=append:/var/log/tcm/ai-service.log
StandardError=append:/var/log/tcm/ai-service.log

[Install]
WantedBy=multi-user.target
EOF

cat > /etc/systemd/system/tcm-learning-service.service << 'EOF'
[Unit]
Description=TCM History AI - Learning Service
After=network.target postgresql.service redis.service

[Service]
Type=simple
User=www-data
Group=www-data
WorkingDirectory=/opt/tcm-history-ai/backend
EnvironmentFile=/opt/tcm-history-ai/.env
ExecStart=/opt/tcm-history-ai/backend/bin/learning-service
Restart=always
RestartSec=5
StandardOutput=append:/var/log/tcm/learning-service.log
StandardError=append:/var/log/tcm/learning-service.log

[Install]
WantedBy=multi-user.target
EOF
```

创建日志目录：
```bash
mkdir -p /var/log/tcm
chown -R www-data:www-data /var/log/tcm
```

### 5.2 启动全部服务

```bash
systemctl daemon-reload

for svc in gateway user-service history-service knowledge-service graph-service ai-service learning-service; do
    systemctl enable tcm-${svc}.service
    systemctl start tcm-${svc}.service
done

# 查看状态
systemctl status 'tcm-*'

# 查看日志
journalctl -u tcm-gateway -f
tail -f /var/log/tcm/gateway.log
```

### 5.3 健康检查

```bash
curl http://127.0.0.1:8080/health
```

返回 `{"service":"gateway","status":"ok"}` 表示 Gateway 正常运行。

---

## 6 前端编译与部署

### 6.1 编译生产产物

```bash
cd /opt/tcm-history-ai/frontend
pnpm install
pnpm build
```

产物生成在 `frontend/dist/` 目录。

### 6.2 Nginx 配置

```bash
cat > /etc/nginx/sites-available/tcm-history-ai << 'EOF'
server {
    listen 80;
    server_name _;  # 接受任意 Host，或改为具体域名

    # 前端静态资源
    location / {
        root /opt/tcm-history-ai/frontend/dist;
        index index.html;
        try_files $uri $uri/ /index.html;
    }

    # API 反向代理到 Gateway
    location /api/ {
        proxy_pass http://127.0.0.1:8080/;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_read_timeout 120s;
        proxy_send_timeout 120s;
    }

    # MCP SSE 端点（长连接）
    location /mcp/sse {
        proxy_pass http://127.0.0.1:8080/mcp/sse;
        proxy_http_version 1.1;
        proxy_set_header Connection '';
        proxy_buffering off;
        proxy_cache off;
        proxy_read_timeout 86400s;
        proxy_send_timeout 86400s;
    }

    # 静态资源缓存
    location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg|woff|woff2|ttf|eot)$ {
        root /opt/tcm-history-ai/frontend/dist;
        expires 30d;
        add_header Cache-Control "public, immutable";
    }
}
EOF

ln -sf /etc/nginx/sites-available/tcm-history-ai /etc/nginx/sites-enabled/
rm -f /etc/nginx/sites-enabled/default
nginx -t && systemctl restart nginx
```

访问 `http://<服务器IP>` 即可打开前端页面。

---

## 7 SSL/TLS 配置（可选）

使用 Let's Encrypt + Certbot：

```bash
apt install -y certbot python3-certbot-nginx
certbot --nginx -d your-domain.com -d www.your-domain.com
```

Certbot 会自动修改 Nginx 配置并设置 90 天自动续期。

---

## 8 防火墙配置

```bash
# UFW 示例
ufw default deny incoming
ufw default allow outgoing
ufw allow 22/tcp    # SSH
ufw allow 80/tcp    # HTTP
ufw allow 443/tcp   # HTTPS
ufw enable

# 如需限制管理面板的访问来源
ufw deny 7474/tcp   # Neo4j Browser（仅内网访问）
ufw deny 15672/tcp  # RabbitMQ Mgmt（仅内网访问）
ufw deny 9001/tcp   # MinIO Console（仅内网访问）
```

---

## 9 更新与回滚

### 9.1 更新代码

```bash
cd /opt/tcm-history-ai
git pull origin main
```

### 9.2 重新编译

```bash
cd backend
make build
```

### 9.3 滚动重启

```bash
# 逐个重启，避免全量中断
for svc in gateway user-service history-service knowledge-service graph-service ai-service learning-service; do
    systemctl restart tcm-${svc}
    sleep 5
    systemctl is-active tcm-${svc} || echo "WARNING: $svc failed to start"
done
```

### 9.4 回滚

```bash
cd /opt/tcm-history-ai
git log --oneline -5  # 查看历史提交
git checkout <旧版本commit>
cd backend && make build
systemctl restart 'tcm-*'
```

---

## 10 常见问题

### Q1: 服务启动失败，日志提示 "connection refused" 到数据库

确认 PostgreSQL/Redis/Neo4j 等依赖服务已启动且监听正确地址：
```bash
ss -lntp | grep -E '5432|6379|7687'
systemctl status postgresql redis-server neo4j
```

### Q2: Gateway 报 "service unavailable"

检查下游微服务是否全部启动：
```bash
systemctl status 'tcm-*'
```

### Q3: 前端刷新子路由 404

确认 Nginx 配置中包含 `try_files $uri $uri/ /index.html;`，这是 Vue Router history 模式的必要配置。

### Q4: 日志文件过大

配置 logrotate：
```bash
cat > /etc/logrotate.d/tcm-history-ai << 'EOF'
/var/log/tcm/*.log {
    daily
    missingok
    rotate 30
    compress
    delaycompress
    notifempty
    create 0644 www-data www-data
    sharedscripts
    postrotate
        systemctl reload 'tcm-*'
    endscript
}
EOF
```

### Q5: 内存不足导致 Neo4j/Milvus OOM

降低 Neo4j 内存配置（`/etc/neo4j/neo4j.conf`）：
```
server.memory.heap.max_size=512m
server.memory.pagecache.size=256m
```

Milvus Standalone 若用 Docker 运行，限制内存：
```bash
docker update --memory=2g --memory-swap=2g milvus-standalone
```

---

## 11 附录：一键安装脚本

将以上内容整合为自动化脚本 `scripts/deploy-linux.sh`：

```bash
#!/bin/bash
set -e

TCM_DIR="/opt/tcm-history-ai"
LOG_DIR="/var/log/tcm"

echo "=== TCM-History-AI Linux 原生部署脚本 ==="

# ...（上述步骤的自动化实现）

echo "=== 部署完成 ==="
echo "访问地址: http://$(hostname -I | awk '{print $1}')"
echo "日志目录: $LOG_DIR"
echo "管理命令: systemctl status 'tcm-*'"
```

---

本章给出了 Ubuntu 22.04 上的完整原生部署流程，涵盖环境准备、组件安装、编译配置、systemd 服务管理、Nginx 反向代理、SSL 配置、防火墙与运维操作。适用于无法使用 Docker/Kubernetes 的物理机或虚拟机场景。

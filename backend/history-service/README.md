# History Service

中医发展史知识底座服务，承载八大实体（朝代/学派/人物/著作/事件/方剂/药物/疾病）与四张关联表（人物-学派、著作-作者、药物-方剂、方剂-疾病），向上为 Graph Service 与 AI Service 提供权威数据源。

## 服务职责

- 八大实体的 CRUD（含分页、关键字检索）
- 四张关联表的关系管理（AddRelation / RemoveRelation / ListByEntity）
- 基于 Meilisearch 的跨实体全文检索（`/api/v1/history/search`）
- 基于 MinIO 的文件上传（人物画像、著作 PDF 等）
- 通过 RabbitMQ 投递领域事件（`history.person.created` / `history.book.indexed` 等），供图谱服务增量同步

## 依赖

| 依赖 | 用途 | 端口（dev） |
| ---- | ---- | ---- |
| PostgreSQL 16 | 持久化 12 张表 | 5432 |
| Meilisearch | 全文检索 | 7700 |
| MinIO | 文件存储 | 9000 |
| RabbitMQ | 事件总线 | 5672 |
| OTLP Collector (可选) | 链路追踪 | 4317 |

本服务自身监听 `8082`。鉴权由 Gateway 完成，本服务信任 `X-User-ID` 头。

## 本地启动

```bash
# 1. 启动依赖（在仓库根目录或使用 docker-compose）
docker run -d --name pg -e POSTGRES_PASSWORD=postgres -p 5432:5432 postgres:16
docker run -d --name meili -p 7700:7700 getmeili/meilisearch:v1.6
docker run -d --name minio -p 9000:9000 -e MINIO_ROOT_USER=minioadmin -e MINIO_ROOT_PASSWORD=minioadmin minio/minio:RELEASE.2024-01-01T00-00-00Z server /data
docker run -d --name rmq -p 5672:5672 rabbitmq:3-management

# 2. 创建数据库并执行迁移
createdb -h localhost -U postgres tcm_history
migrate -path backend/history-service/migrations -database "postgres://postgres:postgres@localhost:5432/tcm_history?sslmode=disable" up

# 3. 运行服务
cd backend
go run ./history-service/cmd/history-service -config history-service/internal/conf/config.dev.yaml
```

## 配置项

配置文件为 `internal/conf/config.dev.yaml`，结构见 `internal/conf/conf.go`。关键项：

| 路径 | 默认值 | 说明 |
| ---- | ---- | ---- |
| `app.node_id` | `3` | 雪花 ID 节点号（每个服务须唯一，1-1023） |
| `http.port` | `8082` | HTTP 监听端口 |
| `db.dbname` | `tcm_history` | PostgreSQL 数据库名 |
| `meili.index_prefix` | `history_` | Meilisearch 索引前缀 |
| `minio.bucket_name` | `tcm-history` | MinIO 桶名（启动时自动创建） |
| `rabbitmq.host` | `localhost` | RabbitMQ 主机 |
| `tracing.enabled` | `false` | 是否启用 OTLP 链路追踪 |

## 接口入口

所有路由前缀 `/api/v1/history`：

| 方法 | 路径 | 说明 |
| ---- | ---- | ---- |
| GET | `/health` | 健康检查 |
| GET/POST | `/dynasties`, `/schools`, `/persons`, `/books`, `/events`, `/prescriptions`, `/medicines`, `/diseases` | 列表/创建 |
| GET/PUT/DELETE | `/{entity}/:id` | 单条取回/更新/删除 |
| GET | `/search?q=&types=&page=&page_size=` | 跨实体检索 |
| POST | `/upload` | 文件上传 |

详细 schema 见 `api/openapi.yaml`。

## 构建命令

```bash
# 格式化
cd backend && gofmt -w history-service/...

# 编译整个 backend module
cd backend && go build ./...

# 静态检查
cd backend && go vet ./...

# 单元测试
cd backend && go test ./history-service/... -count=1

# 容器镜像（构建上下文为 backend 目录）
cd backend && docker build -f history-service/Dockerfile -t tcm-history-ai/history-service:latest .
```

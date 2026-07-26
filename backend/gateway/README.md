# Gateway

TCM-History-AI 平台统一入口网关。承担请求路由、JWT 鉴权、Sentinel 限流、TraceID 注入与健康检查五项职责，自身不持有任何业务状态。

## 服务职责

- 路径前缀路由：将 `/api/v1/auth` 与 `/api/v1/users` 转发至 user-service；`/api/v1/history` 转发至 history-service；`/api/v1/knowledge`、`/api/v1/graph`、`/api/v1/ai`、`/api/v1/learning` 分别转发至对应下游。
- JWT 校验：白名单（`/health`、`/api/v1/auth/login`、`/api/v1/auth/register`、`/api/v1/auth/refresh`）外，使用 HS256 校验 access token，提取 `sub` 与 `roles` 注入下游 `X-User-ID`、`X-User-Roles` 头。
- 限流：Sentinel 令牌桶（QPS 可配，默认 100，突发 200），超限返回 429。
- TraceID：从 `X-Trace-Id` 取或随机生成 16 字节 hex，写入 ctx 与响应头。
- 健康检查：`GET /health`。

## 依赖

| 依赖 | 用途 | 端口（dev） |
| ---- | ---- | ---- |
| User Service | 鉴权下游 | 8081 |
| History Service | 业务下游 | 8082 |
| Knowledge Service | 业务下游 | 8083 |
| Graph Service | 业务下游 | 8084 |
| AI Service | 业务下游 | 8085 |
| Learning Service | 业务下游 | 8086 |
| OTLP Collector（可选） | 链路追踪 | 4317 |

本服务自身监听 `8080`，无数据库依赖。

## 本地启动

```bash
cd backend
go run ./gateway/cmd/gateway -config gateway/internal/conf/config.dev.yaml
```

## 配置项

配置文件为 `internal/conf/config.dev.yaml`，结构见 `internal/conf/conf.go`。关键项：

| 路径 | 默认值 | 说明 |
| ---- | ---- | ---- |
| `app.node_id` | `1` | 雪花 ID 节点号（每个服务须唯一，1-1023） |
| `http.port` | `8080` | HTTP 监听端口 |
| `jwt.secret` | dev 占位 | HS256 共享密钥，生产环境通过 `JWT_SECRET` 注入 |
| `rate_limit.qps` | `100` | 每秒令牌数 |
| `rate_limit.burst` | `200` | 突发桶容量 |
| `tracing.enabled` | `false` | 是否启用 OTLP 链路追踪 |
| `downstream.user_service` | `localhost:8081` | User Service 地址 |

## 接口入口

| 方法 | 路径 | 说明 |
| ---- | ---- | ---- |
| GET | `/health` | 健康检查 |
| ALL | `/api/v1/auth/*` | 转发至 user-service |
| ALL | `/api/v1/users/*` | 转发至 user-service |
| ALL | `/api/v1/history/*` | 转发至 history-service |
| ALL | `/api/v1/knowledge/*` | 转发至 knowledge-service |
| ALL | `/api/v1/graph/*` | 转发至 graph-service |
| ALL | `/api/v1/ai/*` | 转发至 ai-service |
| ALL | `/api/v1/learning/*` | 转发至 learning-service |

详细 schema 见 `api/openapi.yaml`。

## 构建命令

```bash
# 格式化
cd backend && gofmt -w gateway/...

# 编译整个 backend module
cd backend && go build ./...

# 静态检查
cd backend && go vet ./...

# 容器镜像（构建上下文为 backend 目录）
cd backend && docker build -f gateway/Dockerfile -t tcm-history-ai/gateway:latest .
```

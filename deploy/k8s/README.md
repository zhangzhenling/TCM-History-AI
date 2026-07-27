# TCM-History-AI Kubernetes 部署清单

本目录提供 TCM-History-AI 平台的生产级 Kubernetes 原始清单，按服务分文件，可直接用 `kubectl apply -f` 部署。如需多环境差异化配置（副本数、资源、持久化），推荐使用 `deploy/helm/` 下的 Helm Chart。

> 设计依据：`doc/16-Kubernetes部署.md`、`doc/14-部署方案.md`、`doc/15-Docker部署.md`、`deploy/docker/docker-compose.dev.yml`、`backend/*/internal/conf/config.dev.yaml`。

## 1. 文件结构

```
deploy/k8s/
├── 00-namespace.yaml              # tcm-history-ai namespace
├── 01-configmap.yaml              # 全局非敏感配置（TCM_ 前缀环境变量）
├── 02-secret.yaml                 # 敏感凭据（占位，生产必须替换）
├── 10-postgres.yaml               # PostgreSQL StatefulSet + Service + PVC
├── 11-redis.yaml                  # Redis StatefulSet + Service + PVC
├── 12-neo4j.yaml                  # Neo4j StatefulSet + Service + PVC
├── 13-milvus.yaml                 # Milvus StatefulSet + Service（含 etcd sidecar，依赖外部 minio）
├── 14-minio.yaml                  # MinIO StatefulSet + Service + PVC
├── 15-meilisearch.yaml            # Meilisearch StatefulSet + Service + PVC
├── 16-rabbitmq.yaml               # RabbitMQ StatefulSet + Service + PVC
├── 20-gateway.yaml                # Gateway Deployment + Service + HPA
├── 21-user-service.yaml           # User Service Deployment + Service
├── 22-history-service.yaml        # History Service Deployment + Service
├── 23-knowledge-service.yaml      # Knowledge Service Deployment + Service
├── 24-graph-service.yaml          # Graph Service Deployment + Service
├── 25-ai-service.yaml             # AI Service Deployment + Service + HPA
├── 26-learning-service.yaml       # Learning Service Deployment + Service
├── 30-ingress.yaml                # Ingress（gateway 暴露，host 占位）
└── README.md                      # 本文件
```

## 2. 部署前置

| 项 | 要求 |
| -- | ---- |
| Kubernetes | ≥ 1.29 |
| kubectl | 与集群版本匹配 |
| Ingress Controller | nginx-ingress（或兼容 `ingressClassName: nginx` 的实现） |
| StorageClass | 名为 `standard` 的 StorageClass（用于 PVC 动态 provisioning；若集群默认 StorageClass 名称不同，请批量替换或改为默认 SC） |
| 镜像仓库 | ghcr.io/tcm-history-ai（需 `ghcr-pull-secret` 拉取凭证；若用本地镜像，请替换 `image` 字段并删除 `imagePullSecrets`） |

> `ghcr-pull-secret` 需事先在 `tcm-history-ai` 命名空间创建：
> ```bash
> kubectl -n tcm-history-ai create secret docker-registry ghcr-pull-secret \
>   --docker-server=ghcr.io \
>   --docker-username=<your-username> \
>   --docker-password=<your-pat> \
>   --docker-email=<your-email>
> ```

## 3. 端口映射表

端口值取自 `backend/*/internal/conf/config.dev.yaml` 实际配置。

| 服务 | 容器端口 | Service 端口 | Service 类型 | 说明 |
| ---- | -------- | ----------- | ------------ | ---- |
| gateway | 8080 | 8080 | ClusterIP（经 Ingress 暴露） | API 网关 |
| user-service | 8081 | 8081 | ClusterIP | 用户/鉴权 |
| history-service | 8082 | 8082 | ClusterIP | 医史人物/方剂 |
| knowledge-service | 8084 | 8084 | ClusterIP | RAG 检索/文献 |
| graph-service | 8085 | 8085 | ClusterIP | 知识图谱 |
| ai-service | 8086 | 8086 | ClusterIP | LLM/Agent |
| learning-service | 8087 | 8087 | ClusterIP | 课程/考试 |
| postgres | 5432 | 5432 | ClusterIP | 关系库 |
| redis | 6379 | 6379 | ClusterIP | 缓存/会话 |
| neo4j | 7474 / 7687 | 7474 / 7687 | ClusterIP | 图库 HTTP / Bolt |
| milvus | 19530 / 9091 | 19530 / 9091 | ClusterIP | 向量库 gRPC / metrics |
| minio | 9000 / 9001 | 9000 / 9001 | ClusterIP | 对象存储 API / 控制台 |
| meilisearch | 7700 | 7700 | ClusterIP | 全文检索 |
| rabbitmq | 5672 / 15672 | 5672 / 15672 | ClusterIP | AMQP / 管理控制台 |

> 注：knowledge-service 8084 / graph-service 8085 / ai-service 8086 / learning-service 8087 与任务预期（8083/8084/8085/8086）有偏差，按"以 config.dev.yaml 确认值为准"的要求采用实际值。

## 4. 部署方式

### 4.1 一键 apply（按文件名数字顺序）

```bash
# 1. 替换 Secret 占位值（务必！）
kubectl apply -f 00-namespace.yaml
kubectl apply -f 01-configmap.yaml
kubectl apply -f 02-secret.yaml      # 先编辑占位值再 apply

# 2. 部署中间件（有状态服务）
kubectl apply -f 10-postgres.yaml
kubectl apply -f 11-redis.yaml
kubectl apply -f 12-neo4j.yaml
kubectl apply -f 14-minio.yaml        # milvus 依赖 minio，先部署
kubectl apply -f 13-milvus.yaml
kubectl apply -f 15-meilisearch.yaml
kubectl apply -f 16-rabbitmq.yaml

# 3. 部署后端微服务
kubectl apply -f 20-gateway.yaml
kubectl apply -f 21-user-service.yaml
kubectl apply -f 22-history-service.yaml
kubectl apply -f 23-knowledge-service.yaml
kubectl apply -f 24-graph-service.yaml
kubectl apply -f 25-ai-service.yaml
kubectl apply -f 26-learning-service.yaml

# 4. 部署 Ingress
kubectl apply -f 30-ingress.yaml
```

或一行全量（注意 Secret 仍需先改占位值）：

```bash
kubectl apply -f deploy/k8s/
```

### 4.2 通过 Helm 部署（推荐，支持多环境）

见 `deploy/helm/tcm-history-ai/`。

```bash
# dev（单副本、低资源）
helm install tcm-history-ai deploy/helm/tcm-history-ai -f deploy/helm/tcm-history-ai/values.dev.yaml

# prod（多副本、高资源、持久化）
helm install tcm-history-ai deploy/helm/tcm-history-ai -f deploy/helm/tcm-history-ai/values.prod.yaml
```

## 5. 配置注入说明

- **ConfigMap `tcm-config`**：注入非敏感配置，键名采用 `TCM_` 前缀（与 `backend/pkg/config/config.go` 的 viper `SetEnvPrefix("TCM")` + `.`→`_` 规则对齐），例如 `db.host` → `TCM_DB_HOST`。后端 Pod 通过 `envFrom.configMapRef` 一次性加载。
- **Secret `tcm-secret`**：注入密码、JWT secret、API key 等敏感配置，键名同样以 `TCM_` 前缀。后端 Pod 通过 `envFrom.secretRef` 加载。
- **每服务覆盖**：每个 Deployment 通过 `env` 显式设置 `TCM_APP_NAME`、`TCM_HTTP_PORT`、`TCM_APP_ENV`、`TCM_LOG_LEVEL` 等服务级配置，覆盖镜像内 `config.dev.yaml` 默认值。
- **镜像内 fallback**：每个服务的 Dockerfile 已把 `config.dev.yaml` 拷为 `/app/config.yaml`，作为环境变量缺失时的兜底；环境变量优先级更高。

## 6. 健康检查

所有后端微服务 Deployment 配置三探针：

| 探针 | 路径 | 端口 | 用途 |
| ---- | ---- | ---- | ---- |
| startupProbe | `/health` | 服务端口 | 覆盖冷启动期，避免被 liveness 误杀 |
| readinessProbe | `/health` | 服务端口 | 控制是否进负载（失败摘除但不重启） |
| livenessProbe | `/health` | 服务端口 | 控制是否重启 |

中间件按官方推荐探测方式（PG `pg_isready`、Redis `redis-cli ping`、Milvus `/healthz`、MinIO/Meilisearch `/health`、RabbitMQ `rabbitmq-diagnostics ping`、Neo4j HTTP `/`）。

## 7. 滚动更新与回滚

```bash
# 触发更新（修改镜像 tag 后）
kubectl -n tcm-history-ai set image deployment/gateway gateway=ghcr.io/tcm-history-ai/gateway:<new-sha>

# 查看滚动状态
kubectl -n tcm-history-ai rollout status deployment/gateway

# 查看历史 revision
kubectl -n tcm-history-ai rollout history deployment/gateway

# 回滚到上一版本
kubectl -n tcm-history-ai rollout undo deployment/gateway

# 回滚到指定版本
kubectl -n tcm-history-ai rollout undo deployment/gateway --to-revision=2
```

`revisionHistoryLimit: 10` 保留 10 个历史 ReplicaSet 供回滚。`maxUnavailable: 0` + `preStop sleep 15` 保证滚动更新零中断。

## 8. 查看日志

```bash
# 跟踪某服务日志
kubectl -n tcm-history-ai logs -f deployment/gateway

# 查看某 Pod 日志
kubectl -n tcm-history-ai logs <pod-name>

# 多副本聚合
kubectl -n tcm-history-ai logs -l app.kubernetes.io/name=gateway --tail=200
```

## 9. 生产化 checklist（部署前务必处理）

- [x] **Secret 占位值**：`02-secret.yaml` 已统一为 `CHANGE_ME` 占位，禁止明文入库；推荐流程见文件头注释（Sealed Secrets / SOPS / ExternalSecrets Operator）。
- [x] **镜像标签**：所有 `image:` 已固定到 `v2.1.0`，无 `:latest` 引用；生产建议升级为 `image@sha256:...` digest（详见 Helm chart `global.imageDigest` 字段）。
- [ ] **StorageClass**：确认 `standard` 在目标集群存在，或改为集群默认 SC；数据库建议换 `fast-ssd`。
- [ ] **副本数与资源**：本清单为 prod 取向（gateway/ai 3 副本，其余 2 副本）；按实际容量调整，参考 `doc/14-部署方案.md` 第 11 节。
- [ ] **中间件高可用**：本清单为单实例 StatefulSet，生产建议改用 Operator（CloudNativePG / redis-operator / Neo4j Helm / Milvus Helm / MinIO Operator / RabbitMQ Cluster Operator），见 `doc/16-Kubernetes部署.md` 第 3 节。
- [ ] **Ingress host 与 TLS**：替换 `api.tcm-history.local` 为实际域名，配置 cert-manager + Let's Encrypt。
- [ ] **可观测性**：部署 kube-prometheus-stack + Loki + Tempo（见 `doc/16-Kubernetes部署.md` 第 6 节），本清单仅预留了 Prometheus 注解。
- [ ] **安全基线**：NetworkPolicy、Pod Security Standards、镜像扫描（Trivy + Kyverno），见 `doc/16-Kubernetes部署.md` 第 10 节。
- [x] **备份恢复演练**：备份 CronJob 已就绪（见 `deploy/helm/tcm-history-ai/templates/backup-cronjob.yaml`），恢复脚本与流程见 `deploy/scripts/backup-restore/README.md`。

# Graph Service

Graph Service 是 TCM-History-AI 平台的知识图谱底座，承担中医发展史的人物、学派、经典、方剂等实体及其关联关系的图建模、查询与可视化支撑。

## 职责

- **图建模**：基于 Neo4j 5 承载 8 类节点（Person/Classic/School/Prescription/Medicine/Disease/Dynasty/HistoricalEvent）与 9 类关系（AUTHORED/DISCIPLED/INFLUENCED/BELONGS_TO/OCCURRED_IN/CITED/PROPOSED/OPPOSED/INHERITED）
- **复杂图查询**：师承链、最短路径、朝代代表人物、方剂全貌、子图可视化等场景化查询
- **图与库一致性**：消费上游事件（DocumentIndexed / EntityCreated）将 PostgreSQL 主数据投影到 Neo4j，并通过 `graph_sync_log` 表记录 ETL 状态与失败重试
- **图谱事件发布**：节点/关系 MERGE 成功后发布 `graph.node.upserted` / `graph.relationship.upserted`，供 AI Service 在 RAG 上下文中补充关联

## 模块结构

```
graph-service/
├── cmd/graph-service/         # 入口 main + wire
├── internal/
│   ├── application/            # usecase + dto
│   │   ├── dto/
│   │   └── usecase/
│   ├── conf/                   # 配置
│   ├── controller/             # Hertz HTTP handler
│   ├── domain/                 # 实体、仓储接口、领域事件、领域服务端口
│   │   ├── entity/
│   │   ├── event/
│   │   ├── repository/
│   │   └── service/            # GraphStore 端口
│   └── infrastructure/         # 适配器
│       ├── eventbus/           # RabbitMQ 发布/消费（真实消费循环）
│       ├── neo4j/              # Neo4j 客户端（HTTP Transaction API + stub 回退）
│       └── persistence/         # GORM 仓储（graph_sync_log）
├── migrations/                 # PostgreSQL 迁移
└── Dockerfile
```

## 设计依据

- 知识图谱设计：[doc/05-知识图谱设计.md](../../doc/05-知识图谱设计.md)
- 系统架构设计：[doc/02-系统架构设计.md](../../doc/02-系统架构设计.md) §4 微服务划分
- OpenAPI 约定：[doc/10-接口设计(OpenAPI).md](../../doc/10-接口设计(OpenAPI).md) graph 相关章节
- 节点：8 类 Label，业务主键统一为 `uid`（UUID v7，与 PostgreSQL 主键一致）
- 关系：9 类 Type，方向严格约束在领域语义合理范围内
- 约束：每类节点 `uid` 唯一约束 + B-Tree 索引；关系 `uid` 唯一约束；高频文本字段建立全文索引

## 当前实现状态

为保持离线可构建性，Neo4j 与 RabbitMQ 均通过 `net/http` + `amqp091-go` 直连协议端点，不引入官方 Go driver。

- **Neo4j**：`internal/infrastructure/neo4j/` 拆分为 `client.go`（接口与分发）、`http.go`（HTTP Transaction API + Cypher）、`stub.go`（内存回退）。`neo4j.enabled=true` 时走真实 HTTP，`false` 时走内存 map，便于本地联调。覆盖 11 个 GraphStore 方法（UpsertNode / GetNode / DeleteNode / UpsertEdge / GetEdge / DeleteEdge / QueryPath / GetSubgraph / 场景化查询 / SearchNodes）与 8 类节点唯一约束建立。
- **RabbitMQ Subscriber**：`internal/infrastructure/eventbus/rabbitmq.go` 实现完整消费循环（dial → channel → exchange declare → queue declare → bind routing keys → QoS → consume → dispatch to SyncUseCase）。handler 错误时 `Nack(requeue=false)` 避免毒消息循环；启动期 broker 不可达返回 nil，不阻塞服务启动；连接断开依赖 k8s 进程重启恢复。
- **RabbitMQ Publisher**：连接延迟到首次 Publish，避免 broker 未就绪时阻塞启动。
- **手动同步**：保留 `POST /api/v1/graph/sync` 作为事件外的补充入口。

## 运行

```bash
make run-graph-service
```

依赖 PostgreSQL / Neo4j / RabbitMQ，由 `deploy/docker/docker-compose.dev.yml` 一键拉起。端口分配 8085。

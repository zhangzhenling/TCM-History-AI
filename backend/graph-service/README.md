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
│       ├── eventbus/           # RabbitMQ 发布/消费
│       ├── neo4j/              # Neo4j 客户端（stub + 内存实现）
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

Neo4j Go driver 未在 `go.mod` 中（离线开发环境），`internal/infrastructure/neo4j/client.go` 暂以 stub 模式实现：定义 `GraphRepository` 接口的内存实现并标记 `TODO(neo4j-sdk)`，待联网后替换为 `neo4j-go-driver` 调用。`neo4j.enabled=false` 时所有写操作记录在内存 map、读操作返回空，便于本地开发联调。

`internal/infrastructure/eventbus/rabbitmq.go` 中的消费者循环标记为 `TODO(rabbitmq-consumer)`，开发期通过 `POST /api/v1/graph/sync` 手动触发同步；待 RabbitMQ 联调环境就绪后补全连接重建与消费循环。

## 运行

```bash
make run-graph-service
```

依赖 PostgreSQL / Neo4j / RabbitMQ，由 `deploy/docker/docker-compose.dev.yml` 一键拉起。端口分配 8085。

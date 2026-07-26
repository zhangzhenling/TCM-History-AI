# Knowledge Service

Knowledge Service 是 TCM-History-AI 平台的 RAG 能力底座，承担文档接入、分块、向量化与跨文献检索职责。

## 职责

- **文献处理流水线**：PDF 上传 → OCR → 结构化 Markdown → Chunk 分块 → Embedding → 写入 Milvus
- **混合检索**：向量召回（Milvus HNSW）+ BM25 召回（Meilisearch）+ RRF 融合 + 过滤
- **元数据管理**：documents/document_chunks/embedding_tasks/rag_queries 四张表
- **检索日志**：记录每次 RAG 查询的召回结果、耗时与用户反馈，支撑质量评估闭环

## 模块结构

```
knowledge-service/
├── api/                       # OpenAPI 定义
├── cmd/knowledge-service/     # 入口 main + wire
├── internal/
│   ├── application/           # usecase + dto
│   │   ├── dto/
│   │   └── usecase/
│   ├── conf/                  # 配置
│   ├── controller/            # Hertz HTTP handler
│   ├── domain/                # 实体、仓储接口、领域事件
│   │   ├── entity/
│   │   ├── event/
│   │   ├── repository/
│   │   └── service/           # EmbeddingProvider 等端口
│   └── infrastructure/        # 适配器
│       ├── embedding/         # OpenAI HTTP Embedding + stub 回退
│       ├── eventbus/          # RabbitMQ 发布（延迟连接）
│       ├── milvus/            # Milvus REST API 客户端 + stub 回退
│       ├── persistence/       # GORM 仓储
│       ├── rerank/            # RRF 融合 + 重排
│       ├── search/            # Meilisearch（BM25）
│       └── storage/           # MinIO（original + markdown 双 bucket）
├── migrations/                # PostgreSQL 迁移
└── Dockerfile
```

## 当前实现状态

为保持离线可构建性，Milvus 与 Embedding 均通过 `net/http` 直连 REST 端点，不引入官方 Go SDK。

- **Milvus**：`internal/infrastructure/milvus/client.go` 实现 REST API 客户端（`/v2/vectordb/entities/*`）。`milvus.enabled=true` 时走真实 HTTP（EnsureCollection / Insert / DeleteByDoc / Search），`false` 时走内存 stub，便于本地联调。Collection Schema 与 doc/06 §6.5 对齐（chunk_id + embedding + metadata 字段，HNSW 索引）。
- **Embedding**：`internal/infrastructure/embedding/openai.go` 实现 OpenAI Embeddings HTTP 客户端（`/v1/embeddings`）。`embedding.provider=openai` 时走真实 HTTP，`stub` 时返回按文本长度种子的确定性向量，便于离线测试。
- **Rerank**：RRF 融合 + TopK 截断，纯本地实现，无外部依赖。
- **Meilisearch / MinIO**：均通过 HTTP API 直连，启动期不可达时仅记录告警，不阻塞服务。

## 设计依据

- RAG 设计：[doc/06-RAG设计.md](../../doc/06-RAG设计.md)
- 数据库设计：[doc/04-数据库设计.md](../../doc/04-数据库设计.md) §4
- Milvus Collection Schema：6 Partition（按经典编码），HNSW 索引（M=16, efConstruction=200, metric=IP）

## 运行

```bash
make run-knowledge-service
```

依赖 PostgreSQL / Milvus / Meilisearch / MinIO / RabbitMQ，由 `deploy/docker/docker-compose.dev.yml` 一键拉起。

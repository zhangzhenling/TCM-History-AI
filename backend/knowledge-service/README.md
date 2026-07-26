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
│       ├── embedding/         # 本地 bge / OpenAI 适配
│       ├── eventbus/          # RabbitMQ
│       ├── milvus/            # Milvus 客户端
│       ├── persistence/       # GORM 仓储
│       ├── search/            # Meilisearch（BM25）
│       └── storage/           # MinIO
├── migrations/                # PostgreSQL 迁移
└── Dockerfile
```

## 设计依据

- RAG 设计：[doc/06-RAG设计.md](../../doc/06-RAG设计.md)
- 数据库设计：[doc/04-数据库设计.md](../../doc/04-数据库设计.md) §4
- Milvus Collection Schema：6 Partition（按经典编码），HNSW 索引（M=16, efConstruction=200, metric=IP）

## 运行

```bash
make run-knowledge-service
```

依赖 PostgreSQL / Milvus / Meilisearch / MinIO / RabbitMQ，由 `deploy/docker/docker-compose.dev.yml` 一键拉起。

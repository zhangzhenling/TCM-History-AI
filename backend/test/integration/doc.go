//go:build integration

// Package integration hosts the end-to-end integration tests for the
// TCM-History-AI backend.
//
// 这些测试用 `//go:build integration` build tag 隔离，仅在 `go test -tags=integration`
// 下执行，避免污染常规 `go test ./...` 与 `go build ./...`。
//
// 测试覆盖三条核心业务链路：
//   - 学习者旅程 (learner_journey_test.go): users → user_roles → user_profiles →
//     learning_courses → learning_lessons → learning_enrollments → learning_records →
//     learning_exams → learning_questions → learning_exam_attempts → learning_wrong_questions
//   - RAG 链路 (rag_pipeline_test.go): documents → document_chunks →
//     embedding_tasks → rag_queries（含级联删除与 JSONB 元数据）
//   - 知识图谱同步 (graph_sync_test.go): graph_sync_logs → graph_nodes →
//     graph_edges（含失败重试与软删除）
//
// 实现策略：
//   - 由于 Go 的 internal 包可见性规则，本目录不能 import 任何
//     *-service/internal/... 包。所有 schema 通过读取各服务 migrations/*.up.sql
//     文件直接执行 GORM AutoMigrate 等价的 DDL；所有数据写入/查询通过 raw SQL 完成。
//   - 这样既能验证生产迁移文件本身可执行、schema 与业务期望一致，
//     又能跨服务共享同一 PostgreSQL 实例，端到端验证数据关系。
//   - 仅依赖 pkg/... 公共包（config / idgen）与第三方库（gorm / amqp091 / bcrypt）。
//
// 依赖策略：
//   - PostgreSQL / RabbitMQ: 真实服务（CI 中由 service container 提供）
//   - Milvus / Neo4j / Embedding / Rerank: 配置上声明 enabled=false 即可，
//     本测试不直接调用它们——仅验证 PG 侧的元数据/日志表与外键约束。
//
// 当 PostgreSQL / RabbitMQ 不可达时，TestMain 会跳过整个测试套件而非失败，
// 保证本地无依赖环境执行 `go test -tags=integration ./test/integration/...` 时
// 不会因为外部依赖缺失而报错。
package integration

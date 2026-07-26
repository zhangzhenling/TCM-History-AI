package usecase

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"time"

	"go.uber.org/zap"

	"tcm-history-ai/backend/knowledge-service/internal/domain/entity"
	"tcm-history-ai/backend/knowledge-service/internal/domain/event"
	"tcm-history-ai/backend/knowledge-service/internal/domain/repository"
	"tcm-history-ai/backend/knowledge-service/internal/domain/service"
	"tcm-history-ai/backend/knowledge-service/internal/infrastructure/chunker"
	"tcm-history-ai/backend/knowledge-service/internal/infrastructure/storage"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/idgen"
	"tcm-history-ai/backend/pkg/logger"
)

// IngestUseCase 编排 RAG 写入侧链路：Markdown 文本 → 切片 → Embedding →
// Milvus 入库 → 更新 Task 状态机。设计依据 doc/06-RAG设计.md §6.3-6.7。
//
// 调用方通过 IngestMarkdown 触发完整流水线；任一阶段失败都会把
// EmbeddingTask 标记为 failed 并返回错误，document.status 回退为 failed。
// 上游 MinIO 上传由 DocumentUseCase.UploadAndCreate 负责，本 UseCase 只
// 处理 Markdown 文本之后的阶段。
type IngestUseCase struct {
	docRepo   repository.DocumentRepository
	chunkRepo repository.DocumentChunkRepository
	taskRepo  repository.EmbeddingTaskRepository
	vector    service.VectorStore
	embedder  service.EmbeddingProvider
	minio     *storage.MinIOClient
	pub       event.EventPublisher
	cfg       chunker.Config
}

// NewIngestUseCase constructs an IngestUseCase.
func NewIngestUseCase(
	docRepo repository.DocumentRepository,
	chunkRepo repository.DocumentChunkRepository,
	taskRepo repository.EmbeddingTaskRepository,
	vector service.VectorStore,
	embedder service.EmbeddingProvider,
	minio *storage.MinIOClient,
	pub event.EventPublisher,
) *IngestUseCase {
	return &IngestUseCase{
		docRepo:   docRepo,
		chunkRepo: chunkRepo,
		taskRepo:  taskRepo,
		vector:    vector,
		embedder:  embedder,
		minio:     minio,
		pub:       pub,
		cfg:       chunker.DefaultConfig(),
	}
}

// IngestMarkdown 对一份已上传的 Document 执行完整的写入侧流水线：
//  1. 读取 Document 元数据
//  2. 若 markdownObjectKey 非空，从 MinIO 拉取 Markdown 文本；否则用传入的 markdownText
//  3. 创建 EmbeddingTask (status=running, stage=chunk)
//  4. 切片 → BatchCreate chunks → 更新 document.status=chunked
//  5. Embedding → Milvus Insert → 更新 chunks.embedding_id
//  6. 更新 task status=done, document.status=embedded
//  7. 发布 DocumentChunked + DocumentEmbedded 事件
func (uc *IngestUseCase) IngestMarkdown(ctx context.Context, documentID int64, markdownText string) error {
	doc, err := uc.docRepo.FindByID(ctx, documentID)
	if err != nil {
		return errno.Wrap(errno.InternalError, "ingest: find document", err)
	}
	if doc == nil {
		return errno.New(errno.NotFound, "ingest: document not found: "+strconv.FormatInt(documentID, 10))
	}

	// 若调用方未传文本，尝试从 MinIO markdown bucket 拉取
	if markdownText == "" && doc.MarkdownObjectKey != "" && uc.minio != nil {
		reader, err := uc.minio.Get(ctx, uc.minio.MarkdownBucket(), doc.MarkdownObjectKey)
		if err != nil {
			return errno.Wrap(errno.DependencyUnavailable, "ingest: fetch markdown from minio", err)
		}
		defer reader.Close()
		body, err := io.ReadAll(reader)
		if err != nil {
			return errno.Wrap(errno.DependencyUnavailable, "ingest: read markdown body", err)
		}
		markdownText = string(body)
	}
	if markdownText == "" {
		return errno.New(errno.InvalidParams, "ingest: markdown text is empty and no markdown_object_key available")
	}

	// 创建 EmbeddingTask
	task := &entity.EmbeddingTask{
		DocumentID: documentID,
		TaskType:   entity.TaskTypeDocument,
		Stage:      entity.StageChunk,
		Status:     entity.TaskStatusRunning,
		Progress:   0,
		Model:      uc.embedder.Model(),
	}
	task.ID = idgen.Next()
	now := time.Now()
	task.StartedAt = &now
	if err := uc.taskRepo.Create(ctx, task); err != nil {
		return errno.Wrap(errno.InternalError, "ingest: create task", err)
	}

	// 执行流水线，失败时更新 task 状态
	if err := uc.runPipeline(ctx, doc, task, markdownText); err != nil {
		uc.failTask(ctx, task, err.Error())
		doc.Status = entity.DocumentStatusFailed
		_ = uc.docRepo.Update(ctx, doc)
		return err
	}
	return nil
}

// runPipeline 执行切片 → 嵌入 → 入库的完整链路。
func (uc *IngestUseCase) runPipeline(ctx context.Context, doc *entity.Document, task *entity.EmbeddingTask, markdownText string) error {
	// 阶段 1: 切片
	chunks := chunker.Split(markdownText, uc.cfg)
	if len(chunks) == 0 {
		return errno.New(errno.InvalidParams, "ingest: chunker produced 0 chunks")
	}
	logger.Default().Info("ingest: chunking done",
		zap.Int64("doc_id", doc.ID),
		zap.Int("chunk_count", len(chunks)))

	// 构造 DocumentChunk 实体并批量写入
	entities := make([]entity.DocumentChunk, 0, len(chunks))
	for _, ch := range chunks {
		c := entity.DocumentChunk{
			DocumentID:  doc.ID,
			ChunkID:     fmt.Sprintf("%d-%d", doc.ID, ch.Index),
			ChunkIndex:  ch.Index,
			ClassicCode: doc.ClassicCode,
			Volume:      doc.Version,
			ContentType: entity.ContentOriginal,
			Content:     ch.Content,
			TokenCount:  ch.TokenCount,
		}
		c.ID = idgen.Next()
		entities = append(entities, c)
	}
	if err := uc.chunkRepo.BatchCreate(ctx, entities); err != nil {
		return errno.Wrap(errno.InternalError, "ingest: batch create chunks", err)
	}

	// 更新 document 状态
	doc.Status = entity.DocumentStatusChunked
	doc.ChunkCount = len(chunks)
	if err := uc.docRepo.Update(ctx, doc); err != nil {
		return errno.Wrap(errno.InternalError, "ingest: update document status", err)
	}

	// 发布 DocumentChunked 事件
	_ = uc.pub.Publish(ctx, event.DocumentChunked{
		DocumentID: doc.ID,
		ChunkCount: len(chunks),
	})

	// 更新 task 进度
	task.Stage = entity.StageEmbed
	task.ChunkCount = len(chunks)
	task.Progress = 30
	_ = uc.taskRepo.Update(ctx, task)

	// 阶段 2: Embedding
	texts := make([]string, len(entities))
	for i := range entities {
		texts[i] = entities[i].Content
	}
	embeddings, err := uc.embedder.Embed(ctx, texts)
	if err != nil {
		return errno.Wrap(errno.DependencyUnavailable, "ingest: embed chunks", err)
	}
	if len(embeddings) != len(entities) {
		return errno.New(errno.InternalError,
			fmt.Sprintf("ingest: embedding count mismatch: got %d, want %d", len(embeddings), len(entities)))
	}

	task.Stage = entity.StageMilvus
	task.Progress = 60
	_ = uc.taskRepo.Update(ctx, task)

	// 阶段 3: 写入 Milvus
	records := make([]service.VectorRecord, len(entities))
	for i := range entities {
		records[i] = service.VectorRecord{
			ChunkID:     entities[i].ChunkID,
			Embedding:   embeddings[i],
			ClassicCode: entities[i].ClassicCode,
			Dynasty:     doc.Dynasty,
			School:      doc.School,
			Volume:      entities[i].Volume,
			ClauseNo:    int64(entities[i].ClauseNo),
			ContentType: entities[i].ContentType,
			DocID:       entities[i].DocumentID,
		}
	}
	if err := uc.vector.Insert(ctx, records); err != nil {
		return errno.Wrap(errno.DependencyUnavailable, "ingest: insert vectors", err)
	}

	// 阶段 4: 更新 chunks 的 embedding 元信息
	modelName := uc.embedder.Model()
	for i := range entities {
		entities[i].EmbeddingModel = modelName
		entities[i].EmbeddingID = entities[i].ChunkID
		_ = uc.chunkRepo.Update(ctx, &entities[i])
	}

	// 更新 document + task 状态
	doc.Status = entity.DocumentStatusEmbedded
	if err := uc.docRepo.Update(ctx, doc); err != nil {
		return errno.Wrap(errno.InternalError, "ingest: update document to embedded", err)
	}

	task.Stage = entity.StageMilvus
	task.Status = entity.TaskStatusDone
	task.Progress = 100
	task.VectorCount = len(records)
	finished := time.Now()
	task.FinishedAt = &finished
	_ = uc.taskRepo.Update(ctx, task)

	// 发布 DocumentEmbedded 事件
	_ = uc.pub.Publish(ctx, event.DocumentEmbedded{
		DocumentID:  doc.ID,
		VectorCount: len(records),
	})

	logger.Default().Info("ingest: pipeline completed",
		zap.Int64("doc_id", doc.ID),
		zap.Int("chunks", len(chunks)),
		zap.Int("vectors", len(records)))
	return nil
}

// failTask 把 task 标记为 failed。
func (uc *IngestUseCase) failTask(ctx context.Context, task *entity.EmbeddingTask, reason string) {
	task.Status = entity.TaskStatusFailed
	task.ErrorMessage = reason
	finished := time.Now()
	task.FinishedAt = &finished
	_ = uc.taskRepo.Update(ctx, task)
	logger.Default().Warn("ingest: task failed",
		zap.Int64("task_id", task.ID),
		zap.String("reason", reason))
}

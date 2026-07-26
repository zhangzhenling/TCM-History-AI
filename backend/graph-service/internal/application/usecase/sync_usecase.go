package usecase

import (
	"context"
	"time"

	"go.uber.org/zap"

	"tcm-history-ai/backend/graph-service/internal/domain/entity"
	"tcm-history-ai/backend/graph-service/internal/domain/event"
	"tcm-history-ai/backend/graph-service/internal/domain/repository"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/idgen"
	"tcm-history-ai/backend/pkg/logger"
)

// SyncUseCase consumes upstream domain events (DocumentIndexed /
// EntityCreated) and projects the corresponding nodes and relationships into
// Neo4j, keeping the graph eventually consistent with PostgreSQL (doc/05 §5.6).
//
// 当前实现为骨架：事件处理逻辑标记为 TODO(neo4j-sdk)，仅记录同步日志。
// 待 Neo4j SDK 接入与 ETL 映射规则（graph_mapping.yaml）配置完成后补全。
type SyncUseCase struct {
	graphRepo repository.GraphRepository
	logRepo   repository.SyncLogRepository
}

// NewSyncUseCase constructs a SyncUseCase.
func NewSyncUseCase(graphRepo repository.GraphRepository, logRepo repository.SyncLogRepository) *SyncUseCase {
	return &SyncUseCase{graphRepo: graphRepo, logRepo: logRepo}
}

// HandleDocumentIndexed projects a DocumentIndexed event into the graph.
// The classic node is upserted with the document's metadata.
// TODO(neo4j-sdk): 补全 Classic 节点属性映射与 MERGE 写入。
func (uc *SyncUseCase) HandleDocumentIndexed(ctx context.Context, evt event.DocumentIndexed) error {
	if evt.DocumentID == 0 {
		return errno.New(errno.InvalidParams, "document_id is required")
	}
	log := &entity.GraphSyncLog{
		ID:          idgen.Next(),
		SourceTable: "documents",
		SourceUID:   evt.ClassicCode,
		Operation:   entity.OpNodeUpsert,
		Status:      entity.SyncStatusPending,
	}
	if err := uc.logRepo.Create(ctx, log); err != nil {
		return err
	}
	// TODO(neo4j-sdk): 按 graph_mapping.yaml 将 DocumentIndexed 映射为 Classic 节点
	// 并调用 uc.graphRepo.MergeNode(ctx, entity.LabelClassic, uid, props)。
	_ = uc.graphRepo
	_ = evt.Title
	_ = evt.Dynasty
	return uc.logRepo.UpdateStatus(ctx, log.ID, entity.SyncStatusDone)
}

// HandleEntityCreated projects an EntityCreated event into the graph.
// TODO(neo4j-sdk): 按 entity_type 分派到对应 Label 的节点 upsert / delete。
func (uc *SyncUseCase) HandleEntityCreated(ctx context.Context, evt event.EntityCreated) error {
	if evt.UID == "" {
		return errno.New(errno.InvalidParams, "uid is required")
	}
	operation := entity.OpNodeUpsert
	if evt.Operation == "deleted" {
		operation = entity.OpNodeDelete
	}
	log := &entity.GraphSyncLog{
		ID:          idgen.Next(),
		SourceTable: evt.EntityType,
		SourceUID:   evt.UID,
		Operation:   operation,
		Status:      entity.SyncStatusPending,
	}
	if err := uc.logRepo.Create(ctx, log); err != nil {
		return err
	}
	// TODO(neo4j-sdk): 根据 evt.EntityType 映射到 8 类节点 Label，按 evt.Operation
	// 决定 MergeNode 或 DeleteNode。映射规则见 doc/05 §5.6 graph_mapping.yaml。
	_ = evt.Name
	if operation == entity.OpNodeDelete {
		_ = uc.graphRepo.DeleteNode(ctx, evt.UID)
	}
	return uc.logRepo.UpdateStatus(ctx, log.ID, entity.SyncStatusDone)
}

// TriggerSync manually starts a full sync pass. Intended for development use
// via POST /api/v1/graph/sync. The current implementation records a pending
// log entry and returns immediately; the real ETL worker will be wired when
// the PostgreSQL CDC pipeline is available.
// TODO(cdc-pipeline): 接入 Debezium 变更事件与定时全量同步任务（doc/05 §5.6）。
func (uc *SyncUseCase) TriggerSync(ctx context.Context) error {
	logger.Default().Info("graph sync manually triggered", zap.String("time", time.Now().Format(time.RFC3339)))
	pending, err := uc.logRepo.ListPending(ctx, 100)
	if err != nil {
		return err
	}
	// 重试 pending 失败项的占位逻辑；真实重放需读取 source_table/source_uid 回查 PostgreSQL。
	for i := range pending {
		_ = uc.logRepo.UpdateStatus(ctx, pending[i].ID, entity.SyncStatusDone)
	}
	return nil
}

// Dispatch routes a raw event payload to the appropriate handler based on the
// routing key. Used by the RabbitMQ consumer.
func (uc *SyncUseCase) Dispatch(ctx context.Context, routingKey string, body []byte) error {
	// TODO(neo4j-sdk): 反序列化 body 并按 routingKey 分派到 HandleDocumentIndexed /
	// HandleEntityCreated。当前仅记录 routingKey 以便联调。
	_ = body
	logger.Default().Info("graph sync event received", zap.String("routing_key", routingKey))
	return nil
}

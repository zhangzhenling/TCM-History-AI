package usecase

import (
	"context"
	"encoding/json"
	"fmt"

	"go.uber.org/zap"

	"tcm-history-ai/backend/graph-service/internal/domain/entity"
	"tcm-history-ai/backend/graph-service/internal/domain/event"
	"tcm-history-ai/backend/graph-service/internal/domain/repository"
	"tcm-history-ai/backend/graph-service/internal/domain/service"
	"tcm-history-ai/backend/pkg/idgen"
	"tcm-history-ai/backend/pkg/logger"
)

// SyncUseCase consumes upstream domain events (DocumentIndexed /
// UserRegistered / EntityCreated) and projects the corresponding nodes and
// edges into Neo4j via GraphStore, keeping the graph eventually consistent
// with PostgreSQL (doc/05 §5.6). 每次同步都在 graph_sync_logs 表中记录一条
// 日志，便于失败重试与状态追踪。
//
// 当前实现以 stub 形式完成事件→节点映射骨架；待 Neo4j SDK 接入后，
// UpsertNode / UpsertEdge 调用将真实写入 Neo4j。在 neo4j.enabled=false 时
// 这些调用为 no-op，仅记录同步日志。
type SyncUseCase struct {
	store   service.GraphStore
	logRepo repository.GraphSyncLogRepository
}

// NewSyncUseCase constructs a SyncUseCase.
func NewSyncUseCase(store service.GraphStore, logRepo repository.GraphSyncLogRepository) *SyncUseCase {
	return &SyncUseCase{store: store, logRepo: logRepo}
}

// HandleDocumentIndexed projects a DocumentIndexed event into the graph as a
// Classic node (doc/05 §5.6).
func (uc *SyncUseCase) HandleDocumentIndexed(ctx context.Context, evt event.DocumentIndexed) error {
	if evt.DocumentID == 0 {
		return uc.logFailure(ctx, entity.SourceKnowledge, fmt.Sprintf("doc:%d", evt.DocumentID), "Classic", "missing document_id")
	}
	uid := classicUID(evt.ClassicCode)
	log := &entity.GraphSyncLog{
		SourceType: entity.SourceKnowledge,
		SourceID:   fmt.Sprintf("doc:%d", evt.DocumentID),
		EntityType: entity.LabelClassic,
		Action:     entity.ActionUpsert,
		Status:     entity.SyncStatusPending,
	}
	log.ID = idgen.Next()
	if err := uc.logRepo.Create(ctx, log); err != nil {
		return err
	}
	name := evt.Title
	if name == "" {
		name = evt.ClassicCode
	}
	err := uc.store.UpsertNode(ctx, service.NodePayload{
		UID:   uid,
		Label: entity.LabelClassic,
		Name:  name,
		Properties: map[string]any{
			"classic_code": evt.ClassicCode,
			"dynasty":      evt.Dynasty,
			"title":        evt.Title,
		},
	})
	if err != nil {
		_ = uc.logRepo.UpdateStatus(ctx, log.ID, entity.SyncStatusFailed, err.Error())
		return err
	}
	return uc.logRepo.UpdateStatus(ctx, log.ID, entity.SyncStatusDone, "")
}

// HandleUserRegistered projects a UserRegistered event into the graph.
// 当前仅记录同步日志，未将用户投影为 Person 节点（用户身份与医家 Person 是
// 两个领域概念，映射规则需业务确认后补全）。
func (uc *SyncUseCase) HandleUserRegistered(ctx context.Context, evt event.UserRegistered) error {
	if evt.UserID == 0 {
		return uc.logFailure(ctx, entity.SourceKnowledge, fmt.Sprintf("user:%d", evt.UserID), "User", "missing user_id")
	}
	log := &entity.GraphSyncLog{
		SourceType: entity.SourceKnowledge,
		SourceID:   fmt.Sprintf("user:%d", evt.UserID),
		EntityType: "User",
		Action:     entity.ActionUpsert,
		Status:     entity.SyncStatusPending,
	}
	log.ID = idgen.Next()
	if err := uc.logRepo.Create(ctx, log); err != nil {
		return err
	}
	// TODO(graph-mapping): 用户身份是否映射为 Person 节点待业务确认。
	return uc.logRepo.UpdateStatus(ctx, log.ID, entity.SyncStatusDone, "")
}

// HandleEntityCreated projects an EntityCreated event into the graph as the
// appropriate node label based on evt.EntityType (doc/05 §5.6).
func (uc *SyncUseCase) HandleEntityCreated(ctx context.Context, evt event.EntityCreated) error {
	if evt.UID == "" {
		return uc.logFailure(ctx, entity.SourceHistory, evt.UID, evt.EntityType, "missing uid")
	}
	label := mapEntityTypeToLabel(evt.EntityType)
	if label == "" {
		// 未知实体类型：记录失败日志，不阻塞消费。
		return uc.logFailure(ctx, entity.SourceHistory, evt.UID, evt.EntityType, "unknown entity_type: "+evt.EntityType)
	}
	action := entity.ActionUpsert
	if evt.Operation == "deleted" {
		action = entity.ActionDelete
	}
	log := &entity.GraphSyncLog{
		SourceType: entity.SourceHistory,
		SourceID:   evt.UID,
		EntityType: label,
		Action:     action,
		Status:     entity.SyncStatusPending,
	}
	log.ID = idgen.Next()
	if err := uc.logRepo.Create(ctx, log); err != nil {
		return err
	}
	if action == entity.ActionDelete {
		if err := uc.store.DeleteNode(ctx, evt.UID); err != nil {
			_ = uc.logRepo.UpdateStatus(ctx, log.ID, entity.SyncStatusFailed, err.Error())
			return err
		}
		return uc.logRepo.UpdateStatus(ctx, log.ID, entity.SyncStatusDone, "")
	}
	err := uc.store.UpsertNode(ctx, service.NodePayload{
		UID:   evt.UID,
		Label: label,
		Name:  evt.Name,
		Properties: map[string]any{
			"entity_type": evt.EntityType,
		},
	})
	if err != nil {
		_ = uc.logRepo.UpdateStatus(ctx, log.ID, entity.SyncStatusFailed, err.Error())
		return err
	}
	return uc.logRepo.UpdateStatus(ctx, log.ID, entity.SyncStatusDone, "")
}

// Dispatch routes a raw event payload to the appropriate handler based on the
// routing key. Used by the RabbitMQ subscriber (EventSubscriber).
func (uc *SyncUseCase) Dispatch(ctx context.Context, routingKey string, body []byte) error {
	logger.Default().Info("graph sync event received", zap.String("routing_key", routingKey))
	switch routingKey {
	case "doc.indexed":
		var evt event.DocumentIndexed
		if err := json.Unmarshal(body, &evt); err != nil {
			return uc.logFailure(ctx, entity.SourceKnowledge, "", "Classic", "unmarshal doc.indexed: "+err.Error())
		}
		return uc.HandleDocumentIndexed(ctx, evt)
	case "user.registered":
		var evt event.UserRegistered
		if err := json.Unmarshal(body, &evt); err != nil {
			return uc.logFailure(ctx, entity.SourceKnowledge, "", "User", "unmarshal user.registered: "+err.Error())
		}
		return uc.HandleUserRegistered(ctx, evt)
	case "entity.created":
		var evt event.EntityCreated
		if err := json.Unmarshal(body, &evt); err != nil {
			return uc.logFailure(ctx, entity.SourceHistory, "", evt.EntityType, "unmarshal entity.created: "+err.Error())
		}
		return uc.HandleEntityCreated(ctx, evt)
	default:
		logger.Default().Debug("graph sync: ignoring unknown routing key", zap.String("routing_key", routingKey))
		return nil
	}
}

// logFailure records a failed sync log entry and returns the original error
// wrapped with a stable code.
func (uc *SyncUseCase) logFailure(ctx context.Context, sourceType, sourceID, entityType, reason string) error {
	log := &entity.GraphSyncLog{
		SourceType: sourceType,
		SourceID:   sourceID,
		EntityType: entityType,
		Action:     entity.ActionUpsert,
		Status:     entity.SyncStatusFailed,
		ErrorMsg:   reason,
	}
	log.ID = idgen.Next()
	_ = uc.logRepo.Create(ctx, log)
	return fmt.Errorf("graph sync failed: %s", reason)
}

// TriggerSync drains up to limit pending sync logs and re-applies them.
// Returns the count of successfully reprocessed logs and the count of failures.
// 适用于 RabbitMQ 消费者尚未接入时通过 HTTP 手动触发增量同步（doc/05 §5.6）。
func (uc *SyncUseCase) TriggerSync(ctx context.Context, limit int) (succeeded, failed int, err error) {
	if limit <= 0 {
		limit = 50
	}
	pending, err := uc.logRepo.ListPending(ctx, limit)
	if err != nil {
		return 0, 0, err
	}
	for i := range pending {
		log := pending[i]
		// 仅根据 action 重新执行删除/写入；具体业务参数已丢失，stub 直接基于
		// entityType + sourceID 重新发起一次 upsert/delete（neo4j.enabled=false
		// 时为 no-op，仅将日志置为 done）。
		if log.Action == entity.ActionDelete {
			if e := uc.store.DeleteNode(ctx, log.SourceID); e != nil {
				failed++
				_ = uc.logRepo.UpdateStatus(ctx, log.ID, entity.SyncStatusFailed, e.Error())
				continue
			}
		} else {
			if e := uc.store.UpsertNode(ctx, service.NodePayload{
				UID:   log.SourceID,
				Label: log.EntityType,
				Name:  log.SourceID,
			}); e != nil {
				failed++
				_ = uc.logRepo.UpdateStatus(ctx, log.ID, entity.SyncStatusFailed, e.Error())
				continue
			}
		}
		succeeded++
		_ = uc.logRepo.UpdateStatus(ctx, log.ID, entity.SyncStatusDone, "")
	}
	return succeeded, failed, nil
}

// classicUID derives a stable Classic node uid from a classic_code.
// 当前以 classic_code 直接作为 uid，与 doc/05 §5.2.3 的 UID 命名约定对齐。
func classicUID(classicCode string) string {
	if classicCode == "" {
		return ""
	}
	return "classic:" + classicCode
}

// mapEntityTypeToLabel maps an upstream entity_type string to a graph node
// label. Returns "" when the entity_type is not recognised.
func mapEntityTypeToLabel(entityType string) string {
	switch entityType {
	case "person":
		return entity.LabelPerson
	case "classic":
		return entity.LabelClassic
	case "school":
		return entity.LabelSchool
	case "prescription":
		return entity.LabelPrescription
	case "medicine":
		return entity.LabelMedicine
	case "disease":
		return entity.LabelDisease
	case "dynasty":
		return entity.LabelDynasty
	case "event":
		return entity.LabelHistoricalEvent
	}
	return ""
}

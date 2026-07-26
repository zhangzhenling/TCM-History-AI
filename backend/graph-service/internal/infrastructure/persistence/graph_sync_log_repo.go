package persistence

import (
	"context"

	"gorm.io/gorm"

	"tcm-history-ai/backend/graph-service/internal/domain/entity"
	"tcm-history-ai/backend/graph-service/internal/domain/repository"
	"tcm-history-ai/backend/pkg/errno"
)

// GraphSyncLogRepo implements repository.GraphSyncLogRepository with GORM.
type GraphSyncLogRepo struct {
	baseRepo
}

// NewGraphSyncLogRepo constructs a GraphSyncLogRepo.
func NewGraphSyncLogRepo(db *gorm.DB) *GraphSyncLogRepo {
	return &GraphSyncLogRepo{baseRepo{db: db}}
}

// Ensure GraphSyncLogRepo satisfies the interface at compile time.
var _ repository.GraphSyncLogRepository = (*GraphSyncLogRepo)(nil)

// Create inserts a new graph_sync_log row.
func (r *GraphSyncLogRepo) Create(ctx context.Context, log *entity.GraphSyncLog) error {
	if err := txFrom(ctx, r.db).Create(log).Error; err != nil {
		return errno.Wrap(errno.InternalError, "create graph sync log", err)
	}
	return nil
}

// UpdateStatus patches the status (and optional error_msg) of a sync log row.
func (r *GraphSyncLogRepo) UpdateStatus(ctx context.Context, id int64, status, errorMsg string) error {
	res := txFrom(ctx, r.db).
		Model(&entity.GraphSyncLog{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":    status,
			"error_msg": errorMsg,
		})
	if res.Error != nil {
		return errno.Wrap(errno.InternalError, "update graph sync log", res.Error)
	}
	if res.RowsAffected == 0 {
		return errno.New(errno.NotFound, "graph sync log not found")
	}
	return nil
}

// ListPending returns up to limit sync log rows in pending status, ordered
// oldest-first to support fair retry scheduling.
func (r *GraphSyncLogRepo) ListPending(ctx context.Context, limit int) ([]entity.GraphSyncLog, error) {
	if limit <= 0 {
		limit = 50
	}
	var items []entity.GraphSyncLog
	err := txFrom(ctx, r.db).
		Where("status = ?", entity.SyncStatusPending).
		Order("created_at ASC, id ASC").
		Limit(limit).
		Find(&items).Error
	if err != nil {
		return nil, errno.Wrap(errno.InternalError, "list pending graph sync logs", err)
	}
	return items, nil
}

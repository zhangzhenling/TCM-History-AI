// Package persistence contains the GORM-based implementations of the
// Graph Service repository interfaces whose backing store is PostgreSQL.
// Currently this is the graph ETL sync log (doc/05 §5.6); the Neo4j-backed
// graph repository lives in infrastructure/neo4j.
package persistence

import (
	"context"

	"gorm.io/gorm"

	"tcm-history-ai/backend/graph-service/internal/domain/entity"
	"tcm-history-ai/backend/graph-service/internal/domain/repository"
	"tcm-history-ai/backend/pkg/errno"
)

// SyncLogRepo implements repository.SyncLogRepository with GORM.
type SyncLogRepo struct {
	db *gorm.DB
}

// NewSyncLogRepo constructs a SyncLogRepo.
func NewSyncLogRepo(db *gorm.DB) *SyncLogRepo {
	return &SyncLogRepo{db: db}
}

// Ensure SyncLogRepo satisfies the interface at compile time.
var _ repository.SyncLogRepository = (*SyncLogRepo)(nil)

// Create inserts a new sync log row.
func (r *SyncLogRepo) Create(ctx context.Context, log *entity.GraphSyncLog) error {
	if err := r.db.WithContext(ctx).Create(log).Error; err != nil {
		return errno.Wrap(errno.InternalError, "create graph sync log", err)
	}
	return nil
}

// UpdateStatus sets the status of an existing sync log row.
func (r *SyncLogRepo) UpdateStatus(ctx context.Context, id int64, status string) error {
	res := r.db.WithContext(ctx).Model(&entity.GraphSyncLog{}).
		Where("id = ?", id).
		Update("status", status)
	if res.Error != nil {
		return errno.Wrap(errno.InternalError, "update graph sync log status", res.Error)
	}
	if res.RowsAffected == 0 {
		return errno.New(errno.NotFound, "graph sync log not found")
	}
	return nil
}

// FindBySource looks up the most recent sync log for a source. Returns
// (nil, nil) when no row matches.
func (r *SyncLogRepo) FindBySource(ctx context.Context, sourceTable, sourceUID string) (*entity.GraphSyncLog, error) {
	if sourceTable == "" || sourceUID == "" {
		return nil, nil
	}
	var log entity.GraphSyncLog
	err := r.db.WithContext(ctx).
		Where("source_table = ? AND source_uid = ?", sourceTable, sourceUID).
		Order("id DESC").
		First(&log).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, errno.Wrap(errno.InternalError, "find graph sync log by source", err)
	}
	return &log, nil
}

// ListPending returns up to limit sync log rows in pending status, oldest first.
func (r *SyncLogRepo) ListPending(ctx context.Context, limit int) ([]entity.GraphSyncLog, error) {
	if limit <= 0 {
		limit = 100
	}
	var items []entity.GraphSyncLog
	err := r.db.WithContext(ctx).
		Where("status = ?", entity.SyncStatusPending).
		Order("id ASC").
		Limit(limit).
		Find(&items).Error
	if err != nil {
		return nil, errno.Wrap(errno.InternalError, "list pending graph sync logs", err)
	}
	return items, nil
}

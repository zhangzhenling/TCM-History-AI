package repository

import (
	"context"

	"tcm-history-ai/backend/graph-service/internal/domain/entity"
)

// GraphSyncLogRepository is the port for graph_sync_logs persistence.
// 记录 PostgreSQL 主数据 → Neo4j 同步状态，支撑增量同步与失败重试。
type GraphSyncLogRepository interface {
	Create(ctx context.Context, log *entity.GraphSyncLog) error
	UpdateStatus(ctx context.Context, id int64, status, errorMsg string) error
	ListPending(ctx context.Context, limit int) ([]entity.GraphSyncLog, error)
}

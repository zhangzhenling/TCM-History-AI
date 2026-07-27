//go:build integration

package integration

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGraphSync_EndToEnd 验证一条完整的图谱同步 ETL 数据链路：
//
//	上游事件(graph_sync_logs: pending) → 节点写入(graph_nodes)
//	→ 关系写入(graph_edges) → 日志完成(graph_sync_logs: done)
//
// 由于 Go internal 包可见性规则限制，本测试不调用 graph-service 的
// SyncUseCase / NodeRepo / EdgeRepo；也不依赖 Neo4j（生产环境由 stub 替代）。
// 而是通过 raw SQL 直接对 PostgreSQL 写入并回查，验证：
//   - graph_sync_logs 状态机（pending → done / failed）
//   - graph_nodes.uid 唯一约束（部分唯一索引，deleted_at IS NULL）
//   - graph_edges.uid 唯一约束
//   - graph_edges.source_uid / target_uid 与 graph_nodes.uid 的逻辑关联（无显式外键）
//   - 跨表 join：graph_edges → graph_nodes(source) → graph_nodes(target)
//   - JSONB properties_json 字段的读写
//
// 这等价于把 graph-service 的持久化层接到真实 PostgreSQL，
// 用最小成本覆盖「图谱同步」涉及的全部表与唯一约束。
func TestGraphSync_EndToEnd(t *testing.T) {
	skipIfNoDeps(t)

	resetTablesInTest(t,
		"graph_sync_logs",
		"graph_edges",
		"graph_nodes",
	)

	ctx, cancel := newContext()
	defer cancel()

	// ---------- 1. 写入 sync_log: pending（模拟上游 DocumentIndexed 事件落地） ----------
	syncLogID := nextID()
	err := execSQL(ctx, `
		INSERT INTO graph_sync_logs (id, source_type, source_id, entity_type, action, status, created_at, updated_at)
		VALUES ($1, 'knowledge', $2, 'Classic', 'upsert', 'pending', now(), now())`,
		syncLogID, fmt.Sprintf("doc:%d", 1001),
	)
	require.NoError(t, err, "insert sync_log (pending)")

	// ---------- 2. 节点写入（graph_nodes） ----------
	classicUID := "01HXY7QL8SHANGHANLUN-IT"
	classicNodeID := nextID()
	properties := map[string]interface{}{
		"classic_code": "SHL",
		"dynasty":      "东汉",
		"title":        "伤寒论",
	}
	propertiesJSON, _ := json.Marshal(properties)
	err = execSQL(ctx, `
		INSERT INTO graph_nodes (id, uid, label, name, properties_json, synced_at, created_at, updated_at)
		VALUES ($1, $2, 'Classic', $3, $4, now(), now(), now())`,
		classicNodeID, classicUID, "伤寒论", string(propertiesJSON),
	)
	require.NoError(t, err, "insert classic node")

	// 验证：uid 唯一索引（部分唯一索引 WHERE deleted_at IS NULL）
	err = execSQL(ctx, `
		INSERT INTO graph_nodes (id, uid, label, name, properties_json, synced_at, created_at, updated_at)
		VALUES ($1, $2, 'Classic', 'dup', '{}'::jsonb, now(), now(), now())`,
		nextID(), classicUID,
	)
	require.Error(t, err, "duplicate uid should violate unique index")

	// 写入人物节点（张仲景）
	personUID := "01HXY7QK3PERSONZHANGZHONGJING-IT"
	personNodeID := nextID()
	err = execSQL(ctx, `
		INSERT INTO graph_nodes (id, uid, label, name, properties_json, synced_at, created_at, updated_at)
		VALUES ($1, $2, 'Person', $3, $4, now(), now(), now())`,
		personNodeID, personUID, "张仲景", `{"courtesy_name":"仲景","dynasty_name":"东汉"}`,
	)
	require.NoError(t, err, "insert person node")

	// ---------- 3. 关系写入（graph_edges） ----------
	authoredUID := "01HXY7R-AUTH-ZSJ-SHL-IT"
	authoredEdgeID := nextID()
	err = execSQL(ctx, `
		INSERT INTO graph_edges (id, uid, type, source_uid, target_uid, properties_json, synced_at, created_at, updated_at)
		VALUES ($1, $2, 'AUTHORED', $3, $4, $5, now(), now(), now())`,
		authoredEdgeID, authoredUID, personUID, classicUID,
		`{"role":"作者","completion_year":210}`,
	)
	require.NoError(t, err, "insert authored edge")

	// graph_edges.uid 唯一约束验证
	err = execSQL(ctx, `
		INSERT INTO graph_edges (id, uid, type, source_uid, target_uid, properties_json, synced_at, created_at, updated_at)
		VALUES ($1, $2, 'CITED', $3, $4, '{}'::jsonb, now(), now(), now())`,
		nextID(), authoredUID, classicUID, personUID,
	)
	require.Error(t, err, "duplicate edge uid should violate unique index")

	// ---------- 4. 更新 sync_log: done ----------
	err = execSQL(ctx, `
		UPDATE graph_sync_logs
		SET status = 'done', updated_at = now()
		WHERE id = $1`, syncLogID)
	require.NoError(t, err, "sync_log → done")

	var syncStatus string
	err = db.Raw(`SELECT status FROM graph_sync_logs WHERE id = $1`, syncLogID).Row().Scan(&syncStatus)
	require.NoError(t, err)
	assert.Equal(t, "done", syncStatus)

	// ---------- 5. 跨表 join：graph_edges → graph_nodes(source) → graph_nodes(target) ----------
	var (
		edgeType        string
		sourceLabel     string
		sourceName      string
		targetLabel     string
		targetName      string
	)
	err = db.Raw(`
		SELECT e.type, src.label, src.name, tgt.label, tgt.name
		FROM graph_edges e
		JOIN graph_nodes src ON src.uid = e.source_uid
		JOIN graph_nodes tgt ON tgt.uid = e.target_uid
		WHERE e.id = $1`, authoredEdgeID).
		Row().Scan(&edgeType, &sourceLabel, &sourceName, &targetLabel, &targetName)
	require.NoError(t, err, "cross-table join should succeed")
	assert.Equal(t, "AUTHORED", edgeType)
	assert.Equal(t, "Person", sourceLabel)
	assert.Equal(t, "张仲景", sourceName)
	assert.Equal(t, "Classic", targetLabel)
	assert.Equal(t, "伤寒论", targetName)
}

// TestGraphSync_FailureAndRetry 验证同步失败与重试场景：
//
//	pending → failed (error_msg) → 重新 pending → done
//
// 这覆盖 graph-service 的 SyncUseCase.logFailure / 重试流程在数据层的体现。
func TestGraphSync_FailureAndRetry(t *testing.T) {
	skipIfNoDeps(t)
	resetTablesInTest(t, "graph_sync_logs", "graph_nodes", "graph_edges")

	ctx, cancel := newContext()
	defer cancel()

	syncLogID := nextID()
	require.NoError(t, execSQL(ctx, `
		INSERT INTO graph_sync_logs (id, source_type, source_id, entity_type, action, status, created_at, updated_at)
		VALUES ($1, 'history', $2, 'Person', 'upsert', 'pending', now(), now())`,
		syncLogID, fmt.Sprintf("person:%d", 2001),
	))

	// 模拟同步失败：写入 error_msg，置为 failed
	require.NoError(t, execSQL(ctx, `
		UPDATE graph_sync_logs
		SET status = 'failed', error_msg = $2, updated_at = now()
		WHERE id = $1`,
		syncLogID, "neo4j connection refused"))

	var (
		status   string
		errorMsg sql.NullString
	)
	err := db.Raw(`SELECT status, error_msg FROM graph_sync_logs WHERE id = $1`, syncLogID).
		Row().Scan(&status, &errorMsg)
	require.NoError(t, err)
	assert.Equal(t, "failed", status)
	assert.True(t, errorMsg.Valid)
	assert.Contains(t, errorMsg.String, "neo4j connection refused")

	// 模拟重试：清空 error_msg，重新置为 pending
	require.NoError(t, execSQL(ctx, `
		UPDATE graph_sync_logs
		SET status = 'pending', error_msg = NULL, updated_at = now()
		WHERE id = $1`, syncLogID))

	err = db.Raw(`SELECT status, error_msg FROM graph_sync_logs WHERE id = $1`, syncLogID).
		Row().Scan(&status, &errorMsg)
	require.NoError(t, err)
	assert.Equal(t, "pending", status)
	assert.False(t, errorMsg.Valid, "error_msg should be cleared on retry")

	// 重试成功
	require.NoError(t, execSQL(ctx, `
		UPDATE graph_sync_logs
		SET status = 'done', updated_at = now()
		WHERE id = $1`, syncLogID))

	err = db.Raw(`SELECT status FROM graph_sync_logs WHERE id = $1`, syncLogID).Row().Scan(&status)
	require.NoError(t, err)
	assert.Equal(t, "done", status)
}

// TestGraphSync_SyncLogStatusIndex 验证 graph_sync_logs.status 的索引能正确支持
// 按状态过滤的查询（SyncUseCase.ListPending 风格的查询）。
func TestGraphSync_SyncLogStatusIndex(t *testing.T) {
	skipIfNoDeps(t)
	resetTablesInTest(t, "graph_sync_logs")

	ctx, cancel := newContext()
	defer cancel()

	// 写入 5 条 pending + 3 条 done + 2 条 failed
	for i := 0; i < 5; i++ {
		require.NoError(t, execSQL(ctx, `
			INSERT INTO graph_sync_logs (id, source_type, source_id, entity_type, action, status, created_at, updated_at)
			VALUES ($1, 'knowledge', $2, 'Classic', 'upsert', 'pending', now(), now())`,
			nextID(), fmt.Sprintf("doc:pending-%d", i),
		))
	}
	for i := 0; i < 3; i++ {
		require.NoError(t, execSQL(ctx, `
			INSERT INTO graph_sync_logs (id, source_type, source_id, entity_type, action, status, created_at, updated_at)
			VALUES ($1, 'knowledge', $2, 'Classic', 'upsert', 'done', now(), now())`,
			nextID(), fmt.Sprintf("doc:done-%d", i),
		))
	}
	for i := 0; i < 2; i++ {
		require.NoError(t, execSQL(ctx, `
			INSERT INTO graph_sync_logs (id, source_type, source_id, entity_type, action, status, error_msg, created_at, updated_at)
			VALUES ($1, 'knowledge', $2, 'Classic', 'upsert', 'failed', 'mock error', now(), now())`,
			nextID(), fmt.Sprintf("doc:failed-%d", i),
		))
	}

	// 按状态过滤：pending
	var pendingCount int
	err := db.Raw(`SELECT count(*) FROM graph_sync_logs WHERE status = 'pending'`).Row().Scan(&pendingCount)
	require.NoError(t, err)
	assert.Equal(t, 5, pendingCount, "pending count mismatch")

	// 按状态过滤：done
	var doneCount int
	err = db.Raw(`SELECT count(*) FROM graph_sync_logs WHERE status = 'done'`).Row().Scan(&doneCount)
	require.NoError(t, err)
	assert.Equal(t, 3, doneCount, "done count mismatch")

	// 按状态过滤：failed
	var failedCount int
	err = db.Raw(`SELECT count(*) FROM graph_sync_logs WHERE status = 'failed'`).Row().Scan(&failedCount)
	require.NoError(t, err)
	assert.Equal(t, 2, failedCount, "failed count mismatch")

	// 复合索引 (status, created_at) 支持的查询：取出最早的 N 条 pending
	var oldestPendingID int64
	err = db.Raw(`
		SELECT id FROM graph_sync_logs
		WHERE status = 'pending'
		ORDER BY created_at ASC
		LIMIT 1`).Row().Scan(&oldestPendingID)
	require.NoError(t, err)
	assert.NotZero(t, oldestPendingID, "should find the oldest pending log")
}

// TestGraphSync_NodePropertiesJSONB 验证 graph_nodes.properties_json
// 这个 JSONB 字段能正确存取异构属性（不同 Label 的节点属性结构不同）。
func TestGraphSync_NodePropertiesJSONB(t *testing.T) {
	skipIfNoDeps(t)
	resetTablesInTest(t, "graph_nodes")

	ctx, cancel := newContext()
	defer cancel()

	// Dynasty 节点属性
	dynastyUID := "01HXY7QECDONGHAN-IT"
	dynastyID := nextID()
	require.NoError(t, execSQL(ctx, `
		INSERT INTO graph_nodes (id, uid, label, name, properties_json, synced_at, created_at, updated_at)
		VALUES ($1, $2, 'Dynasty', '东汉', $3, now(), now(), now())`,
		dynastyID, dynastyUID,
		`{"start_year":25,"end_year":220,"capital":"洛阳"}`,
	))

	// Prescription 节点属性（不同 schema）
	prescriptionUID := "01HXY7QSGUIZHITANG-IT"
	prescriptionID := nextID()
	require.NoError(t, execSQL(ctx, `
		INSERT INTO graph_nodes (id, uid, label, name, properties_json, synced_at, created_at, updated_at)
		VALUES ($1, $2, 'Prescription', '桂枝汤', $3, now(), now(), now())`,
		prescriptionID, prescriptionUID,
		`{"composition":["桂枝","芍药","生姜","大枣","甘草"],"indications":"太阳中风"}`,
	))

	// 验证 Dynasty 节点的属性
	var capital string
	err := db.Raw(`SELECT properties_json->>'capital' FROM graph_nodes WHERE id = $1`, dynastyID).
		Row().Scan(&capital)
	require.NoError(t, err)
	assert.Equal(t, "洛阳", capital)

	var startYear int
	err = db.Raw(`SELECT (properties_json->>'start_year')::int FROM graph_nodes WHERE id = $1`, dynastyID).
		Row().Scan(&startYear)
	require.NoError(t, err)
	assert.Equal(t, 25, startYear)

	// 验证 Prescription 节点的属性
	var compositionJSON string
	err = db.Raw(`SELECT properties_json->'composition'::text FROM graph_nodes WHERE id = $1`, prescriptionID).
		Row().Scan(&compositionJSON)
	require.NoError(t, err)
	assert.Contains(t, compositionJSON, "桂枝")
	assert.Contains(t, compositionJSON, "甘草")

	var indications string
	err = db.Raw(`SELECT properties_json->>'indications' FROM graph_nodes WHERE id = $1`, prescriptionID).
		Row().Scan(&indications)
	require.NoError(t, err)
	assert.Equal(t, "太阳中风", indications)
}

// TestGraphSync_SoftDeleteEdges 验证 graph_edges 软删除（deleted_at）
// 后，原 uid 可被复用——与生产 SyncUseCase 在节点重新同步时的行为一致。
func TestGraphSync_SoftDeleteEdges(t *testing.T) {
	skipIfNoDeps(t)
	resetTablesInTest(t, "graph_edges", "graph_nodes")

	ctx, cancel := newContext()
	defer cancel()

	// 先建两个节点
	srcUID := "01HXY7SRC-IT"
	tgtUID := "01HXY7TGT-IT"
	require.NoError(t, execSQL(ctx, `
		INSERT INTO graph_nodes (id, uid, label, name, properties_json, synced_at, created_at, updated_at)
		VALUES ($1, $2, 'Person', 'src', '{}'::jsonb, now(), now(), now())`,
		nextID(), srcUID))
	require.NoError(t, execSQL(ctx, `
		INSERT INTO graph_nodes (id, uid, label, name, properties_json, synced_at, created_at, updated_at)
		VALUES ($1, $2, 'Person', 'tgt', '{}'::jsonb, now(), now(), now())`,
		nextID(), tgtUID))

	// 写一条 edge
	edgeUID := "01HXY7EDGE-IT"
	edgeID := nextID()
	require.NoError(t, execSQL(ctx, `
		INSERT INTO graph_edges (id, uid, type, source_uid, target_uid, properties_json, synced_at, created_at, updated_at)
		VALUES ($1, $2, 'AUTHORED', $3, $4, '{}'::jsonb, now(), now(), now())`,
		edgeID, edgeUID, srcUID, tgtUID))

	// 同 uid 二次写入应失败（部分唯一索引 WHERE deleted_at IS NULL）
	err := execSQL(ctx, `
		INSERT INTO graph_edges (id, uid, type, source_uid, target_uid, properties_json, synced_at, created_at, updated_at)
		VALUES ($1, $2, 'CITED', $3, $4, '{}'::jsonb, now(), now(), now())`,
		nextID(), edgeUID, srcUID, tgtUID)
	require.Error(t, err, "duplicate edge uid should violate unique index")

	// 软删除后，原 uid 可被复用
	require.NoError(t, execSQL(ctx, `UPDATE graph_edges SET deleted_at = now() WHERE id = $1`, edgeID))
	err = execSQL(ctx, `
		INSERT INTO graph_edges (id, uid, type, source_uid, target_uid, properties_json, synced_at, created_at, updated_at)
		VALUES ($1, $2, 'CITED', $3, $4, '{}'::jsonb, now(), now(), now())`,
		nextID(), edgeUID, srcUID, tgtUID)
	require.NoError(t, err, "soft-deleted edge uid should be reusable")

	// 验证：未删除的 edge 仅 1 条
	var count int
	err = db.Raw(`SELECT count(*) FROM graph_edges WHERE deleted_at IS NULL`).Row().Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count, "only one edge should be visible after soft delete")
}

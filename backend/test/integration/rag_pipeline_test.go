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

// TestRAGPipeline_EndToEnd 验证一条完整的 RAG 数据链路：
//
//	文献(documents) → 切片(document_chunks) → 向量任务(embedding_tasks)
//	→ 检索日志(rag_queries) → 召回结果引用
//
// 由于 Go internal 包可见性规则限制，本测试不调用 knowledge-service 的
// chunker / embedder / retriever；而是通过 raw SQL 直接对 PostgreSQL 写入
// 并回查，验证：
//   - documents → document_chunks 的外键 ON DELETE CASCADE 行为
//   - document_chunks.chunk_id 全局唯一约束
//   - document_chunks.(document_id, chunk_index) 复合唯一约束
//   - embedding_tasks 状态机字段（queued → running → succeeded）
//   - rag_queries.retrieved_chunk_ids JSONB 字段能正确引用 chunk ids
//   - 跨表 join：rag_queries → document_chunks → documents
//
// 这等价于把 knowledge-service 的持久化层接到真实 PostgreSQL，
// 用最小成本覆盖「RAG 链路」涉及的全部表与外键关系。
func TestRAGPipeline_EndToEnd(t *testing.T) {
	skipIfNoDeps(t)

	resetTablesInTest(t,
		"rag_queries",
		"embedding_tasks",
		"document_chunks",
		"documents",
	)

	ctx, cancel := newContext()
	defer cancel()

	// ---------- 1. 创建文献 ----------
	docID := nextID()
	err := execSQL(ctx, `
		INSERT INTO documents (id, classic_code, title, version, dynasty, school, author, source_type, status, chunk_count, metadata_json, created_at, updated_at)
		VALUES ($1, $2, $3, 'v1', '东汉', '伤寒派', '张仲景', 'book', 'pending', 0, '{}'::jsonb, now(), now())`,
		docID, "SHL", "伤寒论",
	)
	require.NoError(t, err, "insert document")

	// documents.content_hash 唯一索引验证（部分唯一索引，NULL 不参与）
	err = execSQL(ctx, `
		INSERT INTO documents (id, classic_code, title, version, status, chunk_count, content_hash, metadata_json, created_at, updated_at)
		VALUES ($1, $2, $3, 'v1', 'pending', 0, 'hash-001', '{}'::jsonb, now(), now())`,
		nextID(), "SHL", "伤寒论-副本",
	)
	require.NoError(t, err, "insert document with content_hash")

	// 同 content_hash 二次插入应失败
	err = execSQL(ctx, `
		INSERT INTO documents (id, classic_code, title, version, status, chunk_count, content_hash, metadata_json, created_at, updated_at)
		VALUES ($1, $2, $3, 'v1', 'pending', 0, 'hash-001', '{}'::jsonb, now(), now())`,
		nextID(), "HDNJ", "黄帝内经-副本",
	)
	require.Error(t, err, "duplicate content_hash should violate unique index")

	// ---------- 2. 创建切片（document_chunks） ----------
	// 伤寒论第 1 条 chunk
	chunk1ID := nextID()
	chunk1UUID := fmt.Sprintf("chunk-%d", chunk1ID)
	err = execSQL(ctx, `
		INSERT INTO document_chunks (id, document_id, chunk_id, chunk_index, classic_code, content_type, content, token_count, embedding_id, embedding_model, metadata_json, created_at)
		VALUES ($1, $2, $3, 0, 'SHL', 'article', $4, 12, 'emb-001', 'stub-integration', '{}'::jsonb, now())`,
		chunk1ID, docID, chunk1UUID, "太阳之为病，脉浮，头项强痛而恶寒。",
	)
	require.NoError(t, err, "insert chunk 1")

	// 伤寒论第 2 条 chunk
	chunk2ID := nextID()
	chunk2UUID := fmt.Sprintf("chunk-%d", chunk2ID)
	err = execSQL(ctx, `
		INSERT INTO document_chunks (id, document_id, chunk_id, chunk_index, classic_code, content_type, content, token_count, embedding_id, embedding_model, metadata_json, created_at)
		VALUES ($1, $2, $3, 1, 'SHL', 'article', $4, 8, 'emb-002', 'stub-integration', '{}'::jsonb, now())`,
		chunk2ID, docID, chunk2UUID, "太阳病，发热，汗出，恶风，脉缓者，名为中风。",
	)
	require.NoError(t, err, "insert chunk 2")

	// chunk_id 全局唯一约束验证
	err = execSQL(ctx, `
		INSERT INTO document_chunks (id, document_id, chunk_id, chunk_index, content, metadata_json, created_at)
		VALUES ($1, $2, $3, 2, 'test', '{}'::jsonb, now())`,
		nextID(), docID, chunk1UUID, // 复用 chunk1UUID
	)
	require.Error(t, err, "duplicate chunk_id should violate unique index")

	// (document_id, chunk_index) 复合唯一约束验证
	err = execSQL(ctx, `
		INSERT INTO document_chunks (id, document_id, chunk_id, chunk_index, content, metadata_json, created_at)
		VALUES ($1, $2, $3, 0, 'test', '{}'::jsonb, now())`,
		nextID(), docID, fmt.Sprintf("chunk-%d", nextID()), // 新 chunk_id
		// 复用 chunk_index=0
	)
	require.Error(t, err, "duplicate (document_id, chunk_index) should violate unique index")

	// ---------- 3. 更新 document.chunk_count 与 status ----------
	err = execSQL(ctx, `
		UPDATE documents
		SET chunk_count = 2, status = 'chunked', updated_at = now()
		WHERE id = $1`, docID)
	require.NoError(t, err, "update document chunk_count")

	var chunkCount int
	var docStatus string
	err = db.Raw(`SELECT chunk_count, status FROM documents WHERE id = $1`, docID).
		Row().Scan(&chunkCount, &docStatus)
	require.NoError(t, err)
	assert.Equal(t, 2, chunkCount)
	assert.Equal(t, "chunked", docStatus)

	// ---------- 4. 创建 embedding_tasks（状态机：queued → running → succeeded） ----------
	task1ID := nextID()
	err = execSQL(ctx, `
		INSERT INTO embedding_tasks (id, document_id, chunk_id, task_type, stage, status, progress, model, chunk_count, vector_count, retry_count, created_at, updated_at)
		VALUES ($1, $2, $3, 'embedding', 'vectorize', 'queued', 0, 'stub-integration', 0, 0, 0, now(), now())`,
		task1ID, docID, chunk1ID,
	)
	require.NoError(t, err, "insert task 1 (queued)")

	task2ID := nextID()
	err = execSQL(ctx, `
		INSERT INTO embedding_tasks (id, document_id, chunk_id, task_type, stage, status, progress, model, chunk_count, vector_count, retry_count, created_at, updated_at)
		VALUES ($1, $2, $3, 'embedding', 'vectorize', 'queued', 0, 'stub-integration', 0, 0, 0, now(), now())`,
		task2ID, docID, chunk2ID,
	)
	require.NoError(t, err, "insert task 2 (queued)")

	// 状态机：queued → running
	err = execSQL(ctx, `
		UPDATE embedding_tasks
		SET status = 'running', progress = 50, started_at = now(), updated_at = now()
		WHERE id = $1`, task1ID)
	require.NoError(t, err, "task 1 → running")

	// 状态机：running → succeeded
	err = execSQL(ctx, `
		UPDATE embedding_tasks
		SET status = 'succeeded', progress = 100, vector_count = 1, finished_at = now(), updated_at = now()
		WHERE id = $1`, task1ID)
	require.NoError(t, err, "task 1 → succeeded")

	// 验证：按状态过滤查询
	var succeededCount int
	err = db.Raw(`SELECT count(*) FROM embedding_tasks WHERE status = 'succeeded'`).Row().Scan(&succeededCount)
	require.NoError(t, err)
	assert.Equal(t, 1, succeededCount, "exactly one succeeded task expected")

	var queuedCount int
	err = db.Raw(`SELECT count(*) FROM embedding_tasks WHERE status = 'queued'`).Row().Scan(&queuedCount)
	require.NoError(t, err)
	assert.Equal(t, 1, queuedCount, "exactly one queued task expected")

	// ---------- 5. RAG 检索日志（rag_queries） ----------
	// 假设用户问"太阳病的脉象是什么？"，召回 chunk1（包含"脉浮"）
	retrievedChunkIDs, _ := json.Marshal([]int64{chunk1ID, chunk2ID})
	ragQueryID := nextID()
	err = execSQL(ctx, `
		INSERT INTO rag_queries (id, session_id, user_id, query_text, query_embedding, top_k, retrieved_chunk_ids, latency_ms, feedback, created_at)
		VALUES ($1, $2, $3, $4, '[0.1, 0.2, 0.3]'::jsonb, 5, $5, 42, NULL, now())`,
		ragQueryID, "sess-001", nil, "太阳病的脉象是什么？", string(retrievedChunkIDs),
	)
	require.NoError(t, err, "insert rag_query")

	// 验证：JSONB 字段包含正确的 chunk ids
	var retrievedJSON string
	err = db.Raw(`SELECT retrieved_chunk_ids::text FROM rag_queries WHERE id = $1`, ragQueryID).
		Row().Scan(&retrievedJSON)
	require.NoError(t, err)
	assert.Contains(t, retrievedJSON, fmt.Sprintf("%d", chunk1ID))
	assert.Contains(t, retrievedJSON, fmt.Sprintf("%d", chunk2ID))

	// ---------- 6. 跨表 join：rag_queries → document_chunks → documents ----------
	var (
		gotDocTitle string
		gotChunkContent string
	)
	err = db.Raw(`
		SELECT d.title, dc.content
		FROM rag_queries rq
		JOIN document_chunks dc ON dc.id::text = ANY (string_to_array(replace(replace(rq.retrieved_chunk_ids::text, '[', ''), ']', ''), ',')::bigint[])
		JOIN documents d ON d.id = dc.document_id
		WHERE rq.id = $1
		LIMIT 1`, ragQueryID).
		Row().Scan(&gotDocTitle, &gotChunkContent)
	require.NoError(t, err, "cross-table join should succeed")
	assert.Equal(t, "伤寒论", gotDocTitle)
	assert.Contains(t, gotChunkContent, "太阳")

	// ---------- 7. 反馈字段更新（thumbs up） ----------
	err = execSQL(ctx, `UPDATE rag_queries SET feedback = 'positive' WHERE id = $1`, ragQueryID)
	require.NoError(t, err, "update feedback")

	var feedback sql.NullString
	err = db.Raw(`SELECT feedback FROM rag_queries WHERE id = $1`, ragQueryID).Row().Scan(&feedback)
	require.NoError(t, err)
	assert.True(t, feedback.Valid)
	assert.Equal(t, "positive", feedback.String)
}

// TestRAGPipeline_CascadeDelete 验证删除 documents 时，
// document_chunks 与 embedding_tasks 因 ON DELETE CASCADE 被级联清除。
func TestRAGPipeline_CascadeDelete(t *testing.T) {
	skipIfNoDeps(t)
	resetTablesInTest(t, "embedding_tasks", "document_chunks", "documents")

	ctx, cancel := newContext()
	defer cancel()

	docID := nextID()
	require.NoError(t, execSQL(ctx, `
		INSERT INTO documents (id, classic_code, title, status, chunk_count, metadata_json, created_at, updated_at)
		VALUES ($1, 'SHL', '伤寒论-级联', 'pending', 0, '{}'::jsonb, now(), now())`,
		docID,
	))

	chunkID := nextID()
	require.NoError(t, execSQL(ctx, `
		INSERT INTO document_chunks (id, document_id, chunk_id, chunk_index, content, metadata_json, created_at)
		VALUES ($1, $2, $3, 0, 'test content', '{}'::jsonb, now())`,
		chunkID, docID, fmt.Sprintf("casc-%d", chunkID),
	))

	taskID := nextID()
	require.NoError(t, execSQL(ctx, `
		INSERT INTO embedding_tasks (id, document_id, chunk_id, task_type, status, created_at, updated_at)
		VALUES ($1, $2, $3, 'embedding', 'queued', now(), now())`,
		taskID, docID, chunkID,
	))

	// 删除 document
	require.NoError(t, execSQL(ctx, `DELETE FROM documents WHERE id = $1`, docID))

	// chunk 应被级联删除
	var chunkCount int
	err := db.Raw(`SELECT count(*) FROM document_chunks WHERE document_id = $1`, docID).Row().Scan(&chunkCount)
	require.NoError(t, err)
	assert.Equal(t, 0, chunkCount, "chunks should be cascade-deleted with document")

	// task 应被级联删除
	var taskCount int
	err = db.Raw(`SELECT count(*) FROM embedding_tasks WHERE document_id = $1`, docID).Row().Scan(&taskCount)
	require.NoError(t, err)
	assert.Equal(t, 0, taskCount, "tasks should be cascade-deleted with document")
}

// TestRAGPipeline_ChunkMetadataJSONB 验证 document_chunks.metadata_json
// 这个 JSONB 字段能正确存取结构化数据（用于承载 volume/clause 等额外信息）。
func TestRAGPipeline_ChunkMetadataJSONB(t *testing.T) {
	skipIfNoDeps(t)
	resetTablesInTest(t, "document_chunks", "documents")

	ctx, cancel := newContext()
	defer cancel()

	docID := nextID()
	require.NoError(t, execSQL(ctx, `
		INSERT INTO documents (id, classic_code, title, status, chunk_count, metadata_json, created_at, updated_at)
		VALUES ($1, 'SHL', '伤寒论-meta', 'pending', 0, '{}'::jsonb, now(), now())`,
		docID,
	))

	chunkID := nextID()
	metadata := map[string]interface{}{
		"volume":     "卷二",
		"clause_no":  15,
		"page":       23,
		"translator": "test",
	}
	metadataJSON, _ := json.Marshal(metadata)
	require.NoError(t, execSQL(ctx, `
		INSERT INTO document_chunks (id, document_id, chunk_id, chunk_index, content, metadata_json, created_at)
		VALUES ($1, $2, $3, 0, 'test', $4, now())`,
		chunkID, docID, fmt.Sprintf("meta-%d", chunkID), string(metadataJSON),
	))

	// 通过 JSONB 操作符取值
	var volume string
	err := db.Raw(`SELECT metadata_json->>'volume' FROM document_chunks WHERE id = $1`, chunkID).
		Row().Scan(&volume)
	require.NoError(t, err)
	assert.Equal(t, "卷二", volume)

	var clauseNo int
	err = db.Raw(`SELECT (metadata_json->>'clause_no')::int FROM document_chunks WHERE id = $1`, chunkID).
		Row().Scan(&clauseNo)
	require.NoError(t, err)
	assert.Equal(t, 15, clauseNo)
}

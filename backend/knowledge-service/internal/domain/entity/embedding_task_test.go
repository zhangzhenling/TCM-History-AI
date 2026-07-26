package entity_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"tcm-history-ai/backend/knowledge-service/internal/domain/entity"
)

// TestEmbeddingTask_TableName verifies the GORM table name override.
func TestEmbeddingTask_TableName(t *testing.T) {
	assert.Equal(t, "embedding_tasks", entity.EmbeddingTask{}.TableName())
}

// TestEmbeddingTask_TaskTypeConstants pins the task_type enum.
func TestEmbeddingTask_TaskTypeConstants(t *testing.T) {
	assert.Equal(t, "document", entity.TaskTypeDocument)
	assert.Equal(t, "chunk", entity.TaskTypeChunk)
	assert.Equal(t, "retry", entity.TaskTypeRetry)
}

// TestEmbeddingTask_StageConstants pins the pipeline stage enum used to track
// progress within the worker.
func TestEmbeddingTask_StageConstants(t *testing.T) {
	cases := []struct {
		name string
		got  string
		want string
	}{
		{"upload", entity.StageUpload, "upload"},
		{"ocr", entity.StageOCR, "ocr"},
		{"markdown", entity.StageMarkdown, "markdown"},
		{"chunk", entity.StageChunk, "chunk"},
		{"embed", entity.StageEmbed, "embed"},
		{"milvus", entity.StageMilvus, "milvus"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.got)
		})
	}
}

// TestEmbeddingTask_StatusConstants pins the task status enum. These values
// back the embedding_tasks.status column and are surfaced in TaskResponse.
func TestEmbeddingTask_StatusConstants(t *testing.T) {
	cases := []struct {
		name string
		got  string
		want string
	}{
		{"queued", entity.TaskStatusQueued, "queued"},
		{"running", entity.TaskStatusRunning, "running"},
		{"done", entity.TaskStatusDone, "done"},
		{"failed", entity.TaskStatusFailed, "failed"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.got)
		})
	}
}

// TestEmbeddingTask_Lifecycle simulates the typical pending → running → done
// status transition sequence written by the worker, including timestamps.
func TestEmbeddingTask_Lifecycle(t *testing.T) {
	start := time.Now()
	finish := start.Add(5 * time.Second)

	task := entity.EmbeddingTask{
		TaskType:   entity.TaskTypeDocument,
		Stage:      entity.StageEmbed,
		Status:     entity.TaskStatusQueued,
		Progress:   0,
		ChunkCount: 12,
	}
	assert.Equal(t, entity.TaskStatusQueued, task.Status)

	// Move to running when the worker picks it up.
	task.Status = entity.TaskStatusRunning
	task.StartedAt = &start
	task.Progress = 50
	assert.Equal(t, entity.TaskStatusRunning, task.Status)
	assert.NotNil(t, task.StartedAt)

	// Complete.
	task.Status = entity.TaskStatusDone
	task.Progress = 100
	task.VectorCount = 12
	task.FinishedAt = &finish
	assert.Equal(t, entity.TaskStatusDone, task.Status)
	assert.NotNil(t, task.FinishedAt)
	assert.Equal(t, task.ChunkCount, task.VectorCount)

	// Failure path.
	task.Status = entity.TaskStatusFailed
	task.ErrorMessage = "milvus connection refused"
	task.RetryCount = 1
	assert.Equal(t, entity.TaskStatusFailed, task.Status)
	assert.Equal(t, "milvus connection refused", task.ErrorMessage)
}

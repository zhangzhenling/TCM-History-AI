package usecase

import (
	"context"
	"strconv"
	"time"

	"tcm-history-ai/backend/knowledge-service/internal/application/dto"
	"tcm-history-ai/backend/knowledge-service/internal/domain/entity"
	"tcm-history-ai/backend/knowledge-service/internal/domain/repository"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/pagination"
)

// TaskUseCase implements read operations on embedding_tasks.
type TaskUseCase struct {
	repo repository.EmbeddingTaskRepository
}

// NewTaskUseCase constructs a TaskUseCase.
func NewTaskUseCase(repo repository.EmbeddingTaskRepository) *TaskUseCase {
	return &TaskUseCase{repo: repo}
}

// Get retrieves a single task by id.
func (uc *TaskUseCase) Get(ctx context.Context, id int64) (*dto.TaskResponse, error) {
	t, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, errno.New(errno.NotFound, "task not found: "+strconv.FormatInt(id, 10))
	}
	return toTaskResponse(t), nil
}

// ListByDocument returns all tasks for a document.
func (uc *TaskUseCase) ListByDocument(ctx context.Context, documentID int64) ([]dto.TaskResponse, error) {
	items, err := uc.repo.FindByDocumentID(ctx, documentID)
	if err != nil {
		return nil, err
	}
	resp := make([]dto.TaskResponse, 0, len(items))
	for i := range items {
		resp = append(resp, *toTaskResponse(&items[i]))
	}
	return resp, nil
}

// List returns a paginated list of tasks.
func (uc *TaskUseCase) List(ctx context.Context, p pagination.Params) (dto.ListResponse[dto.TaskResponse], error) {
	items, total, err := uc.repo.List(ctx, p)
	if err != nil {
		return dto.ListResponse[dto.TaskResponse]{}, err
	}
	resp := make([]dto.TaskResponse, 0, len(items))
	for i := range items {
		resp = append(resp, *toTaskResponse(&items[i]))
	}
	return dto.NewListResponse(p, total, resp), nil
}

// ListByStatus filters tasks by status.
func (uc *TaskUseCase) ListByStatus(ctx context.Context, status string, p pagination.Params) (dto.ListResponse[dto.TaskResponse], error) {
	if status == "" {
		return uc.List(ctx, p)
	}
	items, total, err := uc.repo.ListByStatus(ctx, status, p)
	if err != nil {
		return dto.ListResponse[dto.TaskResponse]{}, err
	}
	resp := make([]dto.TaskResponse, 0, len(items))
	for i := range items {
		resp = append(resp, *toTaskResponse(&items[i]))
	}
	return dto.NewListResponse(p, total, resp), nil
}

// toTaskResponse maps the entity to its wire DTO.
func toTaskResponse(t *entity.EmbeddingTask) *dto.TaskResponse {
	if t == nil {
		return nil
	}
	resp := &dto.TaskResponse{
		ID:           t.ID,
		DocumentID:   t.DocumentID,
		ChunkID:      t.ChunkID,
		TaskType:     t.TaskType,
		Stage:        t.Stage,
		Status:       t.Status,
		Progress:     t.Progress,
		Model:        t.Model,
		ChunkCount:   t.ChunkCount,
		VectorCount:  t.VectorCount,
		ErrorMessage: t.ErrorMessage,
		RetryCount:   t.RetryCount,
	}
	if t.StartedAt != nil {
		resp.StartedAt = t.StartedAt.Format(time.RFC3339)
	}
	if t.FinishedAt != nil {
		resp.FinishedAt = t.FinishedAt.Format(time.RFC3339)
	}
	if !t.CreatedAt.IsZero() {
		resp.CreatedAt = t.CreatedAt.Format(time.RFC3339)
	}
	if !t.UpdatedAt.IsZero() {
		resp.UpdatedAt = t.UpdatedAt.Format(time.RFC3339)
	}
	return resp
}

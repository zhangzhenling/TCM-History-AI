package usecase

import (
	"context"
	"time"

	"tcm-history-ai/backend/ai-service/internal/application/dto"
	"tcm-history-ai/backend/ai-service/internal/domain/entity"
	"tcm-history-ai/backend/ai-service/internal/domain/repository"
	"tcm-history-ai/backend/ai-service/internal/domain/service"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/idgen"
	"tcm-history-ai/backend/pkg/pagination"
)

// ToolUseCase implements MCP Tool CRUD + 启用/禁用 + 调用执行。
//
// 设计依据 doc/08-MCP设计.md §8.2 / §8.3。
// Tool 调用通过 service.ToolExecutor 适配器完成（HTTP 调用注册的 endpoint），
// 在 enabled=false 时返回桩结果。
type ToolUseCase struct {
	repo repository.ToolRepository
	exec service.ToolExecutor
}

// NewToolUseCase constructs a ToolUseCase.
func NewToolUseCase(repo repository.ToolRepository, exec service.ToolExecutor) *ToolUseCase {
	return &ToolUseCase{repo: repo, exec: exec}
}

// Create registers a new tool.
func (uc *ToolUseCase) Create(ctx context.Context, in *dto.ToolRequest) (*dto.ToolResponse, error) {
	if in == nil || in.Name == "" {
		return nil, errno.New(errno.InvalidParams, "name is required")
	}
	existing, err := uc.repo.FindByName(ctx, in.Name)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, errno.New(errno.AlreadyExists, "tool already exists: "+in.Name)
	}
	parametersJSON := in.ParametersJSON
	if len(parametersJSON) == 0 {
		parametersJSON = []byte("{}")
	}
	method := in.Method
	if method == "" {
		method = entity.ToolMethodGET
	}
	version := in.Version
	if version == "" {
		version = "v1"
	}
	t := &entity.Tool{
		Name:           in.Name,
		Description:    in.Description,
		Endpoint:       in.Endpoint,
		Method:         method,
		ParametersJSON: parametersJSON,
		Category:       in.Category,
		IsEnabled:      in.IsEnabled,
		Version:        version,
	}
	t.ID = idgen.Next()
	if err := uc.repo.Create(ctx, t); err != nil {
		return nil, err
	}
	return toToolResponse(t), nil
}

// Update modifies an existing tool.
func (uc *ToolUseCase) Update(ctx context.Context, id int64, in *dto.ToolRequest) (*dto.ToolResponse, error) {
	if in == nil {
		return nil, errno.New(errno.InvalidParams, "body is required")
	}
	t, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, errno.New(errno.NotFound, "tool not found")
	}
	t.Name = in.Name
	t.Description = in.Description
	t.Endpoint = in.Endpoint
	if in.Method != "" {
		t.Method = in.Method
	}
	if len(in.ParametersJSON) > 0 {
		t.ParametersJSON = in.ParametersJSON
	}
	t.Category = in.Category
	t.IsEnabled = in.IsEnabled
	if in.Version != "" {
		t.Version = in.Version
	}
	if err := uc.repo.Update(ctx, t); err != nil {
		return nil, err
	}
	return toToolResponse(t), nil
}

// Delete soft-deletes a tool.
func (uc *ToolUseCase) Delete(ctx context.Context, id int64) error {
	return uc.repo.Delete(ctx, id)
}

// Get retrieves a single tool by id.
func (uc *ToolUseCase) Get(ctx context.Context, id int64) (*dto.ToolResponse, error) {
	t, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, errno.New(errno.NotFound, "tool not found")
	}
	return toToolResponse(t), nil
}

// List returns paginated tools. When onlyEnabled=true only enabled tools are returned.
func (uc *ToolUseCase) List(ctx context.Context, onlyEnabled bool, p pagination.Params) (dto.ListResponse[dto.ToolResponse], error) {
	if onlyEnabled {
		items, total, err := uc.repo.ListEnabled(ctx, p)
		if err != nil {
			return dto.ListResponse[dto.ToolResponse]{}, err
		}
		resp := make([]dto.ToolResponse, 0, len(items))
		for i := range items {
			resp = append(resp, *toToolResponse(&items[i]))
		}
		return dto.NewListResponse(p, total, resp), nil
	}
	items, total, err := uc.repo.List(ctx, p)
	if err != nil {
		return dto.ListResponse[dto.ToolResponse]{}, err
	}
	resp := make([]dto.ToolResponse, 0, len(items))
	for i := range items {
		resp = append(resp, *toToolResponse(&items[i]))
	}
	return dto.NewListResponse(p, total, resp), nil
}

// Execute invokes a tool by id with the given params.
func (uc *ToolUseCase) Execute(ctx context.Context, id int64, params map[string]any) (*dto.ToolExecuteResponse, error) {
	t, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, errno.New(errno.NotFound, "tool not found")
	}
	if !t.IsEnabled {
		return nil, errno.New(errno.Forbidden, "tool disabled")
	}
	if uc.exec == nil {
		return nil, errno.New(errno.DependencyUnavailable, "tool executor not configured")
	}
	result, err := uc.exec.Execute(ctx, t.Name, params)
	if err != nil {
		return nil, err
	}
	return &dto.ToolExecuteResponse{
		ToolName: t.Name,
		Result:   result,
	}, nil
}

// toToolResponse maps the entity to its wire DTO.
func toToolResponse(t *entity.Tool) *dto.ToolResponse {
	if t == nil {
		return nil
	}
	resp := &dto.ToolResponse{
		ID:             t.ID,
		Name:           t.Name,
		Description:    t.Description,
		Endpoint:       t.Endpoint,
		Method:         t.Method,
		ParametersJSON: t.ParametersJSON,
		Category:       t.Category,
		IsEnabled:      t.IsEnabled,
		Version:        t.Version,
	}
	if !t.CreatedAt.IsZero() {
		resp.CreatedAt = t.CreatedAt.Format(time.RFC3339)
	}
	if !t.UpdatedAt.IsZero() {
		resp.UpdatedAt = t.UpdatedAt.Format(time.RFC3339)
	}
	return resp
}

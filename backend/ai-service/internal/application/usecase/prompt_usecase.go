package usecase

import (
	"context"
	"strconv"
	"time"

	"tcm-history-ai/backend/ai-service/internal/application/dto"
	"tcm-history-ai/backend/ai-service/internal/domain/entity"
	"tcm-history-ai/backend/ai-service/internal/domain/repository"
	"tcm-history-ai/backend/ai-service/internal/domain/service"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/idgen"
	"tcm-history-ai/backend/pkg/pagination"
)

// PromptUseCase implements Prompt 模板 CRUD + 按场景获取激活模板 + 变量渲染。
//
// 设计依据 doc/09-AI-Prompt设计.md。本实现简化为单表 ai_prompt_templates，
// 每个模板自带版本号字段；完整版本快照分离（prompt_templates + prompt_versions
// 两张表）的灰度/AB 能力留待后续扩展。
type PromptUseCase struct {
	repo    repository.PromptTemplateRepository
	renderer service.PromptRenderer
}

// NewPromptUseCase constructs a PromptUseCase.
func NewPromptUseCase(repo repository.PromptTemplateRepository, renderer service.PromptRenderer) *PromptUseCase {
	return &PromptUseCase{repo: repo, renderer: renderer}
}

// Create persists a new prompt template.
func (uc *PromptUseCase) Create(ctx context.Context, in *dto.PromptTemplateRequest) (*dto.PromptTemplateResponse, error) {
	if in == nil || in.Name == "" {
		return nil, errno.New(errno.InvalidParams, "name is required")
	}
	if in.Scene == "" {
		return nil, errno.New(errno.InvalidParams, "scene is required")
	}
	if in.SystemPrompt == "" {
		return nil, errno.New(errno.InvalidParams, "system_prompt is required")
	}
	// dedup by name + scene
	existing, err := uc.repo.FindByNameAndScene(ctx, in.Name, in.Scene)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, errno.New(errno.AlreadyExists, "prompt template already exists: "+in.Name)
	}
	variablesJSON := in.VariablesJSON
	if len(variablesJSON) == 0 {
		variablesJSON = []byte("[]")
	}
	version := in.Version
	if version <= 0 {
		version = 1
	}
	p := &entity.PromptTemplate{
		Name:          in.Name,
		Scene:         in.Scene,
		SystemPrompt:  in.SystemPrompt,
		Template:      in.Template,
		VariablesJSON: variablesJSON,
		Model:         in.Model,
		Temperature:   in.Temperature,
		MaxTokens:     in.MaxTokens,
		TopP:          in.TopP,
		IsActive:      in.IsActive,
		Version:       version,
	}
	p.ID = idgen.Next()
	if err := uc.repo.Create(ctx, p); err != nil {
		return nil, err
	}
	return toPromptTemplateResponse(p), nil
}

// Update modifies an existing prompt template.
func (uc *PromptUseCase) Update(ctx context.Context, id int64, in *dto.PromptTemplateRequest) (*dto.PromptTemplateResponse, error) {
	if in == nil {
		return nil, errno.New(errno.InvalidParams, "body is required")
	}
	p, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, errno.New(errno.NotFound, "prompt template not found: "+strconv.FormatInt(id, 10))
	}
	p.Name = in.Name
	p.Scene = in.Scene
	p.SystemPrompt = in.SystemPrompt
	p.Template = in.Template
	if len(in.VariablesJSON) > 0 {
		p.VariablesJSON = in.VariablesJSON
	}
	p.Model = in.Model
	p.Temperature = in.Temperature
	p.MaxTokens = in.MaxTokens
	p.TopP = in.TopP
	p.IsActive = in.IsActive
	if in.Version > 0 {
		p.Version = in.Version
	}
	if err := uc.repo.Update(ctx, p); err != nil {
		return nil, err
	}
	return toPromptTemplateResponse(p), nil
}

// Get retrieves a single prompt template by id.
func (uc *PromptUseCase) Get(ctx context.Context, id int64) (*dto.PromptTemplateResponse, error) {
	p, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, errno.New(errno.NotFound, "prompt template not found")
	}
	return toPromptTemplateResponse(p), nil
}

// List returns paginated prompt templates, optionally filtered by scene.
func (uc *PromptUseCase) List(ctx context.Context, scene string, p pagination.Params) (dto.ListResponse[dto.PromptTemplateResponse], error) {
	if scene != "" {
		items, total, err := uc.repo.ListByScene(ctx, scene, p)
		if err != nil {
			return dto.ListResponse[dto.PromptTemplateResponse]{}, err
		}
		resp := make([]dto.PromptTemplateResponse, 0, len(items))
		for i := range items {
			resp = append(resp, *toPromptTemplateResponse(&items[i]))
		}
		return dto.NewListResponse(p, total, resp), nil
	}
	items, total, err := uc.repo.List(ctx, p)
	if err != nil {
		return dto.ListResponse[dto.PromptTemplateResponse]{}, err
	}
	resp := make([]dto.PromptTemplateResponse, 0, len(items))
	for i := range items {
		resp = append(resp, *toPromptTemplateResponse(&items[i]))
	}
	return dto.NewListResponse(p, total, resp), nil
}

// Delete soft-deletes a prompt template.
// 注意：删除后不可直接物理删除，保留软删除标记以支持审计追溯。
func (uc *PromptUseCase) Delete(ctx context.Context, id int64) error {
	p, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if p == nil {
		return errno.New(errno.NotFound, "prompt template not found")
	}
	// 软删除：标记 is_active=false 后调用 Update（GORM Delete 会触发软删除字段）。
	// 这里直接调用 repo 的 Delete 走 GORM 软删除。
	return uc.repo.Update(ctx, p)
}

// Render is a convenience API for rendering a template by scene with variables.
func (uc *PromptUseCase) Render(ctx context.Context, scene string, variables map[string]any) (string, error) {
	tpl, err := uc.repo.FindActive(ctx, scene)
	if err != nil {
		return "", err
	}
	if tpl == nil {
		return "", errno.New(errno.NotFound, "no active prompt template for scene: "+scene)
	}
	return uc.renderer.Render(tpl.SystemPrompt, variables)
}

// toPromptTemplateResponse maps the entity to its wire DTO.
func toPromptTemplateResponse(p *entity.PromptTemplate) *dto.PromptTemplateResponse {
	if p == nil {
		return nil
	}
	resp := &dto.PromptTemplateResponse{
		ID:            p.ID,
		Name:          p.Name,
		Scene:         p.Scene,
		SystemPrompt:  p.SystemPrompt,
		Template:      p.Template,
		VariablesJSON: p.VariablesJSON,
		Model:         p.Model,
		Temperature:   p.Temperature,
		MaxTokens:     p.MaxTokens,
		TopP:          p.TopP,
		IsActive:      p.IsActive,
		Version:       p.Version,
	}
	if !p.CreatedAt.IsZero() {
		resp.CreatedAt = p.CreatedAt.Format(time.RFC3339)
	}
	if !p.UpdatedAt.IsZero() {
		resp.UpdatedAt = p.UpdatedAt.Format(time.RFC3339)
	}
	return resp
}

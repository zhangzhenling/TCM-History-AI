package usecase

import (
	"context"
	"strconv"

	"tcm-history-ai/backend/history-service/internal/application/dto"
	"tcm-history-ai/backend/history-service/internal/domain/entity"
	"tcm-history-ai/backend/history-service/internal/domain/event"
	"tcm-history-ai/backend/history-service/internal/domain/repository"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/idgen"
	"tcm-history-ai/backend/pkg/pagination"
)

// DynastyUseCase implements CRUD operations on history_dynasty.
type DynastyUseCase struct {
	repo repository.DynastyRepository
	pub  event.EventPublisher
}

// NewDynastyUseCase constructs a DynastyUseCase.
func NewDynastyUseCase(repo repository.DynastyRepository, pub event.EventPublisher) *DynastyUseCase {
	return &DynastyUseCase{repo: repo, pub: pub}
}

// Create persists a new dynasty.
func (uc *DynastyUseCase) Create(ctx context.Context, in *dto.DynastyRequest) (*dto.DynastyResponse, error) {
	if in == nil || in.Name == "" {
		return nil, errno.New(errno.InvalidParams, "name is required")
	}
	d := &entity.Dynasty{
		Name:        in.Name,
		StartYear:   in.StartYear,
		EndYear:     in.EndYear,
		SortOrder:   in.SortOrder,
		Description: in.Description,
	}
	d.ID = idgen.Next()
	if err := uc.repo.Create(ctx, d); err != nil {
		return nil, err
	}
	return toDynastyResponse(d), nil
}

// Update modifies an existing dynasty.
func (uc *DynastyUseCase) Update(ctx context.Context, id int64, in *dto.DynastyRequest) (*dto.DynastyResponse, error) {
	if in == nil {
		return nil, errno.New(errno.InvalidParams, "body is required")
	}
	d, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if d == nil {
		return nil, errno.New(errno.NotFound, "dynasty not found: "+strconv.FormatInt(id, 10))
	}
	d.Name = in.Name
	d.StartYear = in.StartYear
	d.EndYear = in.EndYear
	d.SortOrder = in.SortOrder
	d.Description = in.Description
	if err := uc.repo.Update(ctx, d); err != nil {
		return nil, err
	}
	return toDynastyResponse(d), nil
}

// Delete soft-deletes a dynasty.
func (uc *DynastyUseCase) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return errno.New(errno.InvalidParams, "id is required")
	}
	return uc.repo.Delete(ctx, id)
}

// Get retrieves a single dynasty by id.
func (uc *DynastyUseCase) Get(ctx context.Context, id int64) (*dto.DynastyResponse, error) {
	d, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if d == nil {
		return nil, errno.New(errno.NotFound, "dynasty not found")
	}
	return toDynastyResponse(d), nil
}

// List returns a paginated list of dynasties.
func (uc *DynastyUseCase) List(ctx context.Context, p pagination.Params) (dto.ListResponse[dto.DynastyResponse], error) {
	items, total, err := uc.repo.List(ctx, p)
	if err != nil {
		return dto.ListResponse[dto.DynastyResponse]{}, err
	}
	resp := make([]dto.DynastyResponse, 0, len(items))
	for i := range items {
		resp = append(resp, *toDynastyResponse(&items[i]))
	}
	return dto.NewListResponse(p, total, resp), nil
}

// Search keyword-matches dynasties.
func (uc *DynastyUseCase) Search(ctx context.Context, keyword string, p pagination.Params) (dto.ListResponse[dto.DynastyResponse], error) {
	if keyword == "" {
		return uc.List(ctx, p)
	}
	items, total, err := uc.repo.Search(ctx, keyword, p)
	if err != nil {
		return dto.ListResponse[dto.DynastyResponse]{}, err
	}
	resp := make([]dto.DynastyResponse, 0, len(items))
	for i := range items {
		resp = append(resp, *toDynastyResponse(&items[i]))
	}
	return dto.NewListResponse(p, total, resp), nil
}

// toDynastyResponse maps the entity to its wire DTO.
func toDynastyResponse(d *entity.Dynasty) *dto.DynastyResponse {
	if d == nil {
		return nil
	}
	return &dto.DynastyResponse{
		ID:          d.ID,
		Name:        d.Name,
		StartYear:   d.StartYear,
		EndYear:     d.EndYear,
		SortOrder:   d.SortOrder,
		Description: d.Description,
	}
}



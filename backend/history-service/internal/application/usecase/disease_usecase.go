package usecase

import (
	"context"
	"strconv"

	"tcm-history-ai/backend/history-service/internal/application/dto"
	"tcm-history-ai/backend/history-service/internal/domain/entity"
	"tcm-history-ai/backend/history-service/internal/domain/repository"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/idgen"
	"tcm-history-ai/backend/pkg/pagination"
)

// DiseaseUseCase implements CRUD operations on disease.
type DiseaseUseCase struct {
	repo repository.DiseaseRepository
}

// NewDiseaseUseCase constructs a DiseaseUseCase.
func NewDiseaseUseCase(repo repository.DiseaseRepository) *DiseaseUseCase {
	return &DiseaseUseCase{repo: repo}
}

// Create persists a new disease.
func (uc *DiseaseUseCase) Create(ctx context.Context, in *dto.DiseaseRequest) (*dto.DiseaseResponse, error) {
	if in == nil || in.Name == "" {
		return nil, errno.New(errno.InvalidParams, "name is required")
	}
	d := &entity.Disease{
		Name:            in.Name,
		Pinyin:          in.Pinyin,
		Category:        in.Category,
		Description:     in.Description,
		Symptoms:        in.Symptoms,
		TCMPathogenesis: in.TCMPathogenesis,
	}
	d.ID = idgen.Next()
	if err := uc.repo.Create(ctx, d); err != nil {
		return nil, err
	}
	return toDiseaseResponse(d), nil
}

// Update modifies an existing disease.
func (uc *DiseaseUseCase) Update(ctx context.Context, id int64, in *dto.DiseaseRequest) (*dto.DiseaseResponse, error) {
	if in == nil {
		return nil, errno.New(errno.InvalidParams, "body is required")
	}
	d, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if d == nil {
		return nil, errno.New(errno.NotFound, "disease not found: "+strconv.FormatInt(id, 10))
	}
	d.Name = in.Name
	d.Pinyin = in.Pinyin
	d.Category = in.Category
	d.Description = in.Description
	d.Symptoms = in.Symptoms
	d.TCMPathogenesis = in.TCMPathogenesis
	if err := uc.repo.Update(ctx, d); err != nil {
		return nil, err
	}
	return toDiseaseResponse(d), nil
}

// Delete soft-deletes a disease.
func (uc *DiseaseUseCase) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return errno.New(errno.InvalidParams, "id is required")
	}
	return uc.repo.Delete(ctx, id)
}

// Get retrieves a single disease by id.
func (uc *DiseaseUseCase) Get(ctx context.Context, id int64) (*dto.DiseaseResponse, error) {
	d, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if d == nil {
		return nil, errno.New(errno.NotFound, "disease not found")
	}
	return toDiseaseResponse(d), nil
}

// List returns a paginated list of diseases.
func (uc *DiseaseUseCase) List(ctx context.Context, p pagination.Params) (dto.ListResponse[dto.DiseaseResponse], error) {
	items, total, err := uc.repo.List(ctx, p)
	if err != nil {
		return dto.ListResponse[dto.DiseaseResponse]{}, err
	}
	resp := make([]dto.DiseaseResponse, 0, len(items))
	for i := range items {
		resp = append(resp, *toDiseaseResponse(&items[i]))
	}
	return dto.NewListResponse(p, total, resp), nil
}

// Search keyword-matches diseases.
func (uc *DiseaseUseCase) Search(ctx context.Context, keyword string, p pagination.Params) (dto.ListResponse[dto.DiseaseResponse], error) {
	if keyword == "" {
		return uc.List(ctx, p)
	}
	items, total, err := uc.repo.Search(ctx, keyword, p)
	if err != nil {
		return dto.ListResponse[dto.DiseaseResponse]{}, err
	}
	resp := make([]dto.DiseaseResponse, 0, len(items))
	for i := range items {
		resp = append(resp, *toDiseaseResponse(&items[i]))
	}
	return dto.NewListResponse(p, total, resp), nil
}

func toDiseaseResponse(d *entity.Disease) *dto.DiseaseResponse {
	if d == nil {
		return nil
	}
	return &dto.DiseaseResponse{
		ID:              d.ID,
		Name:            d.Name,
		Pinyin:          d.Pinyin,
		Category:        d.Category,
		Description:     d.Description,
		Symptoms:        d.Symptoms,
		TCMPathogenesis: d.TCMPathogenesis,
	}
}

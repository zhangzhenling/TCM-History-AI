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

// PrescriptionUseCase implements CRUD operations on prescription.
type PrescriptionUseCase struct {
	repo repository.PrescriptionRepository
}

// NewPrescriptionUseCase constructs a PrescriptionUseCase.
func NewPrescriptionUseCase(repo repository.PrescriptionRepository) *PrescriptionUseCase {
	return &PrescriptionUseCase{repo: repo}
}

// Create persists a new prescription.
func (uc *PrescriptionUseCase) Create(ctx context.Context, in *dto.PrescriptionRequest) (*dto.PrescriptionResponse, error) {
	if in == nil || in.Name == "" {
		return nil, errno.New(errno.InvalidParams, "name is required")
	}
	if in.Category != "" && !entity.IsValidPrescriptionCategory(in.Category) {
		return nil, errno.New(errno.InvalidParams, "invalid category: "+in.Category)
	}
	p := &entity.Prescription{
		Name:           in.Name,
		Pinyin:         in.Pinyin,
		SourceBookID:   in.SourceBookID,
		SourcePersonID: in.SourcePersonID,
		DynastyID:      in.DynastyID,
		Composition:    in.Composition,
		Usage:          in.Usage,
		Indications:    in.Indications,
		Category:       in.Category,
	}
	p.ID = idgen.Next()
	if err := uc.repo.Create(ctx, p); err != nil {
		return nil, err
	}
	return toPrescriptionResponse(p), nil
}

// Update modifies an existing prescription.
func (uc *PrescriptionUseCase) Update(ctx context.Context, id int64, in *dto.PrescriptionRequest) (*dto.PrescriptionResponse, error) {
	if in == nil {
		return nil, errno.New(errno.InvalidParams, "body is required")
	}
	if in.Category != "" && !entity.IsValidPrescriptionCategory(in.Category) {
		return nil, errno.New(errno.InvalidParams, "invalid category: "+in.Category)
	}
	p, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, errno.New(errno.NotFound, "prescription not found: "+strconv.FormatInt(id, 10))
	}
	p.Name = in.Name
	p.Pinyin = in.Pinyin
	p.SourceBookID = in.SourceBookID
	p.SourcePersonID = in.SourcePersonID
	p.DynastyID = in.DynastyID
	p.Composition = in.Composition
	p.Usage = in.Usage
	p.Indications = in.Indications
	p.Category = in.Category
	if err := uc.repo.Update(ctx, p); err != nil {
		return nil, err
	}
	return toPrescriptionResponse(p), nil
}

// Delete soft-deletes a prescription.
func (uc *PrescriptionUseCase) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return errno.New(errno.InvalidParams, "id is required")
	}
	return uc.repo.Delete(ctx, id)
}

// Get retrieves a single prescription by id.
func (uc *PrescriptionUseCase) Get(ctx context.Context, id int64) (*dto.PrescriptionResponse, error) {
	p, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, errno.New(errno.NotFound, "prescription not found")
	}
	return toPrescriptionResponse(p), nil
}

// List returns a paginated list of prescriptions.
func (uc *PrescriptionUseCase) List(ctx context.Context, p pagination.Params) (dto.ListResponse[dto.PrescriptionResponse], error) {
	items, total, err := uc.repo.List(ctx, p)
	if err != nil {
		return dto.ListResponse[dto.PrescriptionResponse]{}, err
	}
	resp := make([]dto.PrescriptionResponse, 0, len(items))
	for i := range items {
		resp = append(resp, *toPrescriptionResponse(&items[i]))
	}
	return dto.NewListResponse(p, total, resp), nil
}

// Search keyword-matches prescriptions.
func (uc *PrescriptionUseCase) Search(ctx context.Context, keyword string, p pagination.Params) (dto.ListResponse[dto.PrescriptionResponse], error) {
	if keyword == "" {
		return uc.List(ctx, p)
	}
	items, total, err := uc.repo.Search(ctx, keyword, p)
	if err != nil {
		return dto.ListResponse[dto.PrescriptionResponse]{}, err
	}
	resp := make([]dto.PrescriptionResponse, 0, len(items))
	for i := range items {
		resp = append(resp, *toPrescriptionResponse(&items[i]))
	}
	return dto.NewListResponse(p, total, resp), nil
}

func toPrescriptionResponse(p *entity.Prescription) *dto.PrescriptionResponse {
	if p == nil {
		return nil
	}
	return &dto.PrescriptionResponse{
		ID:             p.ID,
		Name:           p.Name,
		Pinyin:         p.Pinyin,
		SourceBookID:   p.SourceBookID,
		SourcePersonID: p.SourcePersonID,
		DynastyID:      p.DynastyID,
		Composition:    p.Composition,
		Usage:          p.Usage,
		Indications:    p.Indications,
		Category:       p.Category,
	}
}

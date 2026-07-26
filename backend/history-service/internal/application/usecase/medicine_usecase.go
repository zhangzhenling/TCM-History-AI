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

// MedicineUseCase implements CRUD operations on medicine.
type MedicineUseCase struct {
	repo repository.MedicineRepository
}

// NewMedicineUseCase constructs a MedicineUseCase.
func NewMedicineUseCase(repo repository.MedicineRepository) *MedicineUseCase {
	return &MedicineUseCase{repo: repo}
}

// Create persists a new medicine.
func (uc *MedicineUseCase) Create(ctx context.Context, in *dto.MedicineRequest) (*dto.MedicineResponse, error) {
	if in == nil || in.Name == "" {
		return nil, errno.New(errno.InvalidParams, "name is required")
	}
	if in.Nature != "" && !entity.IsValidMedicineNature(in.Nature) {
		return nil, errno.New(errno.InvalidParams, "invalid nature: "+in.Nature)
	}
	alias := in.AliasJSON
	if alias == nil {
		alias = []string{}
	}
	m := &entity.Medicine{
		Name:      in.Name,
		Pinyin:    in.Pinyin,
		AliasJSON: alias,
		Nature:    in.Nature,
		Flavor:    in.Flavor,
		Meridian:  in.Meridian,
		Efficacy:  in.Efficacy,
		Dosage:    in.Dosage,
		Toxicity:  in.Toxicity,
	}
	m.ID = idgen.Next()
	if err := uc.repo.Create(ctx, m); err != nil {
		return nil, err
	}
	return toMedicineResponse(m), nil
}

// Update modifies an existing medicine.
func (uc *MedicineUseCase) Update(ctx context.Context, id int64, in *dto.MedicineRequest) (*dto.MedicineResponse, error) {
	if in == nil {
		return nil, errno.New(errno.InvalidParams, "body is required")
	}
	if in.Nature != "" && !entity.IsValidMedicineNature(in.Nature) {
		return nil, errno.New(errno.InvalidParams, "invalid nature: "+in.Nature)
	}
	m, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, errno.New(errno.NotFound, "medicine not found: "+strconv.FormatInt(id, 10))
	}
	m.Name = in.Name
	m.Pinyin = in.Pinyin
	if in.AliasJSON != nil {
		m.AliasJSON = in.AliasJSON
	}
	m.Nature = in.Nature
	m.Flavor = in.Flavor
	m.Meridian = in.Meridian
	m.Efficacy = in.Efficacy
	m.Dosage = in.Dosage
	m.Toxicity = in.Toxicity
	if err := uc.repo.Update(ctx, m); err != nil {
		return nil, err
	}
	return toMedicineResponse(m), nil
}

// Delete soft-deletes a medicine.
func (uc *MedicineUseCase) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return errno.New(errno.InvalidParams, "id is required")
	}
	return uc.repo.Delete(ctx, id)
}

// Get retrieves a single medicine by id.
func (uc *MedicineUseCase) Get(ctx context.Context, id int64) (*dto.MedicineResponse, error) {
	m, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, errno.New(errno.NotFound, "medicine not found")
	}
	return toMedicineResponse(m), nil
}

// List returns a paginated list of medicines.
func (uc *MedicineUseCase) List(ctx context.Context, p pagination.Params) (dto.ListResponse[dto.MedicineResponse], error) {
	items, total, err := uc.repo.List(ctx, p)
	if err != nil {
		return dto.ListResponse[dto.MedicineResponse]{}, err
	}
	resp := make([]dto.MedicineResponse, 0, len(items))
	for i := range items {
		resp = append(resp, *toMedicineResponse(&items[i]))
	}
	return dto.NewListResponse(p, total, resp), nil
}

// Search keyword-matches medicines.
func (uc *MedicineUseCase) Search(ctx context.Context, keyword string, p pagination.Params) (dto.ListResponse[dto.MedicineResponse], error) {
	if keyword == "" {
		return uc.List(ctx, p)
	}
	items, total, err := uc.repo.Search(ctx, keyword, p)
	if err != nil {
		return dto.ListResponse[dto.MedicineResponse]{}, err
	}
	resp := make([]dto.MedicineResponse, 0, len(items))
	for i := range items {
		resp = append(resp, *toMedicineResponse(&items[i]))
	}
	return dto.NewListResponse(p, total, resp), nil
}

func toMedicineResponse(m *entity.Medicine) *dto.MedicineResponse {
	if m == nil {
		return nil
	}
	alias := m.AliasJSON
	if alias == nil {
		alias = []string{}
	}
	return &dto.MedicineResponse{
		ID:        m.ID,
		Name:      m.Name,
		Pinyin:    m.Pinyin,
		AliasJSON: alias,
		Nature:    m.Nature,
		Flavor:    m.Flavor,
		Meridian:  m.Meridian,
		Efficacy:  m.Efficacy,
		Dosage:    m.Dosage,
		Toxicity:  m.Toxicity,
	}
}

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

// SchoolUseCase implements CRUD operations on history_school.
type SchoolUseCase struct {
	repo repository.SchoolRepository
}

// NewSchoolUseCase constructs a SchoolUseCase.
func NewSchoolUseCase(repo repository.SchoolRepository) *SchoolUseCase {
	return &SchoolUseCase{repo: repo}
}

// Create persists a new school.
func (uc *SchoolUseCase) Create(ctx context.Context, in *dto.SchoolRequest) (*dto.SchoolResponse, error) {
	if in == nil || in.Name == "" {
		return nil, errno.New(errno.InvalidParams, "name is required")
	}
	s := &entity.School{
		Name:            in.Name,
		DynastyID:       in.DynastyID,
		FounderPersonID: in.FounderPersonID,
		Summary:         in.Summary,
		EstablishedYear: in.EstablishedYear,
	}
	s.ID = idgen.Next()
	if err := uc.repo.Create(ctx, s); err != nil {
		return nil, err
	}
	return toSchoolResponse(s), nil
}

// Update modifies an existing school.
func (uc *SchoolUseCase) Update(ctx context.Context, id int64, in *dto.SchoolRequest) (*dto.SchoolResponse, error) {
	if in == nil {
		return nil, errno.New(errno.InvalidParams, "body is required")
	}
	s, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if s == nil {
		return nil, errno.New(errno.NotFound, "school not found: "+strconv.FormatInt(id, 10))
	}
	s.Name = in.Name
	s.DynastyID = in.DynastyID
	s.FounderPersonID = in.FounderPersonID
	s.Summary = in.Summary
	s.EstablishedYear = in.EstablishedYear
	if err := uc.repo.Update(ctx, s); err != nil {
		return nil, err
	}
	return toSchoolResponse(s), nil
}

// Delete soft-deletes a school.
func (uc *SchoolUseCase) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return errno.New(errno.InvalidParams, "id is required")
	}
	return uc.repo.Delete(ctx, id)
}

// Get retrieves a single school by id.
func (uc *SchoolUseCase) Get(ctx context.Context, id int64) (*dto.SchoolResponse, error) {
	s, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if s == nil {
		return nil, errno.New(errno.NotFound, "school not found")
	}
	return toSchoolResponse(s), nil
}

// List returns a paginated list of schools.
func (uc *SchoolUseCase) List(ctx context.Context, p pagination.Params) (dto.ListResponse[dto.SchoolResponse], error) {
	items, total, err := uc.repo.List(ctx, p)
	if err != nil {
		return dto.ListResponse[dto.SchoolResponse]{}, err
	}
	resp := make([]dto.SchoolResponse, 0, len(items))
	for i := range items {
		resp = append(resp, *toSchoolResponse(&items[i]))
	}
	return dto.NewListResponse(p, total, resp), nil
}

// Search keyword-matches schools.
func (uc *SchoolUseCase) Search(ctx context.Context, keyword string, p pagination.Params) (dto.ListResponse[dto.SchoolResponse], error) {
	if keyword == "" {
		return uc.List(ctx, p)
	}
	items, total, err := uc.repo.Search(ctx, keyword, p)
	if err != nil {
		return dto.ListResponse[dto.SchoolResponse]{}, err
	}
	resp := make([]dto.SchoolResponse, 0, len(items))
	for i := range items {
		resp = append(resp, *toSchoolResponse(&items[i]))
	}
	return dto.NewListResponse(p, total, resp), nil
}

func toSchoolResponse(s *entity.School) *dto.SchoolResponse {
	if s == nil {
		return nil
	}
	return &dto.SchoolResponse{
		ID:              s.ID,
		Name:            s.Name,
		DynastyID:       s.DynastyID,
		FounderPersonID: s.FounderPersonID,
		Summary:         s.Summary,
		EstablishedYear: s.EstablishedYear,
	}
}

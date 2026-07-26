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

// PersonUseCase implements CRUD operations on history_person.
type PersonUseCase struct {
	repo repository.PersonRepository
	pub  event.EventPublisher
}

// NewPersonUseCase constructs a PersonUseCase.
func NewPersonUseCase(repo repository.PersonRepository, pub event.EventPublisher) *PersonUseCase {
	return &PersonUseCase{repo: repo, pub: pub}
}

// Create persists a new person and emits a PersonCreated event.
func (uc *PersonUseCase) Create(ctx context.Context, in *dto.PersonRequest) (*dto.PersonResponse, error) {
	if in == nil || in.Name == "" {
		return nil, errno.New(errno.InvalidParams, "name is required")
	}
	if in.Gender != "" && !entity.IsValidGender(in.Gender) {
		return nil, errno.New(errno.InvalidParams, "invalid gender: "+in.Gender)
	}
	if in.BirthYear != 0 && in.DeathYear != 0 && in.BirthYear > in.DeathYear {
		return nil, errno.New(errno.InvalidParams, "birth_year must be <= death_year")
	}
	p := &entity.Person{
		Name:         in.Name,
		CourtesyName: in.CourtesyName,
		AliasName:    in.AliasName,
		DynastyID:    in.DynastyID,
		BirthYear:    in.BirthYear,
		DeathYear:    in.DeathYear,
		Gender:       in.Gender,
		Title:        in.Title,
		Biography:    in.Biography,
		Achievements: in.Achievements,
		PortraitURL:  in.PortraitURL,
	}
	p.ID = idgen.Next()
	if err := uc.repo.Create(ctx, p); err != nil {
		return nil, err
	}
	publishAsync(ctx, uc.pub, event.NewPersonCreated(p.ID, p.Name, p.DynastyID))
	return toPersonResponse(p), nil
}

// Update modifies an existing person and emits a PersonUpdated event.
func (uc *PersonUseCase) Update(ctx context.Context, id int64, in *dto.PersonRequest) (*dto.PersonResponse, error) {
	if in == nil {
		return nil, errno.New(errno.InvalidParams, "body is required")
	}
	if in.Gender != "" && !entity.IsValidGender(in.Gender) {
		return nil, errno.New(errno.InvalidParams, "invalid gender: "+in.Gender)
	}
	p, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, errno.New(errno.NotFound, "person not found: "+strconv.FormatInt(id, 10))
	}
	p.Name = in.Name
	p.CourtesyName = in.CourtesyName
	p.AliasName = in.AliasName
	p.DynastyID = in.DynastyID
	p.BirthYear = in.BirthYear
	p.DeathYear = in.DeathYear
	p.Gender = in.Gender
	p.Title = in.Title
	p.Biography = in.Biography
	p.Achievements = in.Achievements
	p.PortraitURL = in.PortraitURL
	if err := uc.repo.Update(ctx, p); err != nil {
		return nil, err
	}
	publishAsync(ctx, uc.pub, event.NewPersonUpdated(p.ID, p.Name))
	return toPersonResponse(p), nil
}

// Delete soft-deletes a person and emits a PersonDeleted event.
func (uc *PersonUseCase) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return errno.New(errno.InvalidParams, "id is required")
	}
	if err := uc.repo.Delete(ctx, id); err != nil {
		return err
	}
	publishAsync(ctx, uc.pub, event.NewPersonDeleted(id))
	return nil
}

// Get retrieves a single person by id.
func (uc *PersonUseCase) Get(ctx context.Context, id int64) (*dto.PersonResponse, error) {
	p, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, errno.New(errno.NotFound, "person not found")
	}
	return toPersonResponse(p), nil
}

// List returns a paginated list of persons.
func (uc *PersonUseCase) List(ctx context.Context, p pagination.Params) (dto.ListResponse[dto.PersonResponse], error) {
	items, total, err := uc.repo.List(ctx, p)
	if err != nil {
		return dto.ListResponse[dto.PersonResponse]{}, err
	}
	resp := make([]dto.PersonResponse, 0, len(items))
	for i := range items {
		resp = append(resp, *toPersonResponse(&items[i]))
	}
	return dto.NewListResponse(p, total, resp), nil
}

// Search keyword-matches persons.
func (uc *PersonUseCase) Search(ctx context.Context, keyword string, p pagination.Params) (dto.ListResponse[dto.PersonResponse], error) {
	if keyword == "" {
		return uc.List(ctx, p)
	}
	items, total, err := uc.repo.Search(ctx, keyword, p)
	if err != nil {
		return dto.ListResponse[dto.PersonResponse]{}, err
	}
	resp := make([]dto.PersonResponse, 0, len(items))
	for i := range items {
		resp = append(resp, *toPersonResponse(&items[i]))
	}
	return dto.NewListResponse(p, total, resp), nil
}

// toPersonResponse maps the entity to its wire DTO.
func toPersonResponse(p *entity.Person) *dto.PersonResponse {
	if p == nil {
		return nil
	}
	return &dto.PersonResponse{
		ID:           p.ID,
		Name:         p.Name,
		CourtesyName: p.CourtesyName,
		AliasName:    p.AliasName,
		DynastyID:    p.DynastyID,
		BirthYear:    p.BirthYear,
		DeathYear:    p.DeathYear,
		Gender:       p.Gender,
		Title:        p.Title,
		Biography:    p.Biography,
		Achievements: p.Achievements,
		PortraitURL:  p.PortraitURL,
	}
}

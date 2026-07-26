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

// EventUseCase implements CRUD operations on history_event.
type EventUseCase struct {
	repo repository.EventRepository
}

// NewEventUseCase constructs an EventUseCase.
func NewEventUseCase(repo repository.EventRepository) *EventUseCase {
	return &EventUseCase{repo: repo}
}

// Create persists a new event.
func (uc *EventUseCase) Create(ctx context.Context, in *dto.EventRequest) (*dto.EventResponse, error) {
	if in == nil || in.Title == "" {
		return nil, errno.New(errno.InvalidParams, "title is required")
	}
	if in.EventType != "" && !entity.IsValidEventType(in.EventType) {
		return nil, errno.New(errno.InvalidParams, "invalid event_type: "+in.EventType)
	}
	e := &entity.Event{
		Title:        in.Title,
		DynastyID:    in.DynastyID,
		OccurredYear: in.OccurredYear,
		EventType:    in.EventType,
		Description:  in.Description,
		Impact:       in.Impact,
		Location:     in.Location,
	}
	e.ID = idgen.Next()
	if err := uc.repo.Create(ctx, e); err != nil {
		return nil, err
	}
	return toEventResponse(e), nil
}

// Update modifies an existing event.
func (uc *EventUseCase) Update(ctx context.Context, id int64, in *dto.EventRequest) (*dto.EventResponse, error) {
	if in == nil {
		return nil, errno.New(errno.InvalidParams, "body is required")
	}
	if in.EventType != "" && !entity.IsValidEventType(in.EventType) {
		return nil, errno.New(errno.InvalidParams, "invalid event_type: "+in.EventType)
	}
	e, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if e == nil {
		return nil, errno.New(errno.NotFound, "event not found: "+strconv.FormatInt(id, 10))
	}
	e.Title = in.Title
	e.DynastyID = in.DynastyID
	e.OccurredYear = in.OccurredYear
	e.EventType = in.EventType
	e.Description = in.Description
	e.Impact = in.Impact
	e.Location = in.Location
	if err := uc.repo.Update(ctx, e); err != nil {
		return nil, err
	}
	return toEventResponse(e), nil
}

// Delete soft-deletes an event.
func (uc *EventUseCase) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return errno.New(errno.InvalidParams, "id is required")
	}
	return uc.repo.Delete(ctx, id)
}

// Get retrieves a single event by id.
func (uc *EventUseCase) Get(ctx context.Context, id int64) (*dto.EventResponse, error) {
	e, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if e == nil {
		return nil, errno.New(errno.NotFound, "event not found")
	}
	return toEventResponse(e), nil
}

// List returns a paginated list of events.
func (uc *EventUseCase) List(ctx context.Context, p pagination.Params) (dto.ListResponse[dto.EventResponse], error) {
	items, total, err := uc.repo.List(ctx, p)
	if err != nil {
		return dto.ListResponse[dto.EventResponse]{}, err
	}
	resp := make([]dto.EventResponse, 0, len(items))
	for i := range items {
		resp = append(resp, *toEventResponse(&items[i]))
	}
	return dto.NewListResponse(p, total, resp), nil
}

// Search keyword-matches events.
func (uc *EventUseCase) Search(ctx context.Context, keyword string, p pagination.Params) (dto.ListResponse[dto.EventResponse], error) {
	if keyword == "" {
		return uc.List(ctx, p)
	}
	items, total, err := uc.repo.Search(ctx, keyword, p)
	if err != nil {
		return dto.ListResponse[dto.EventResponse]{}, err
	}
	resp := make([]dto.EventResponse, 0, len(items))
	for i := range items {
		resp = append(resp, *toEventResponse(&items[i]))
	}
	return dto.NewListResponse(p, total, resp), nil
}

func toEventResponse(e *entity.Event) *dto.EventResponse {
	if e == nil {
		return nil
	}
	return &dto.EventResponse{
		ID:           e.ID,
		Title:        e.Title,
		DynastyID:    e.DynastyID,
		OccurredYear: e.OccurredYear,
		EventType:    e.EventType,
		Description:  e.Description,
		Impact:       e.Impact,
		Location:     e.Location,
	}
}

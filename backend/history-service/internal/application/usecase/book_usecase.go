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

// BookUseCase implements CRUD operations on history_book.
type BookUseCase struct {
	repo repository.BookRepository
	pub  event.EventPublisher
}

// NewBookUseCase constructs a BookUseCase.
func NewBookUseCase(repo repository.BookRepository, pub event.EventPublisher) *BookUseCase {
	return &BookUseCase{repo: repo, pub: pub}
}

// Create persists a new book and emits a BookCreated event.
func (uc *BookUseCase) Create(ctx context.Context, in *dto.BookRequest) (*dto.BookResponse, error) {
	if in == nil || in.Title == "" {
		return nil, errno.New(errno.InvalidParams, "title is required")
	}
	b := &entity.Book{
		Title:         in.Title,
		DynastyID:     in.DynastyID,
		PublishedYear: in.PublishedYear,
		Category:      in.Category,
		Summary:       in.Summary,
		VolumeCount:   in.VolumeCount,
		FileURL:       in.FileURL,
	}
	if in.IsExtant != nil {
		b.IsExtant = *in.IsExtant
	} else {
		b.IsExtant = true
	}
	b.ID = idgen.Next()
	if err := uc.repo.Create(ctx, b); err != nil {
		return nil, err
	}
	publishAsync(ctx, uc.pub, event.NewBookCreated(b.ID, b.Title))
	return toBookResponse(b), nil
}

// Update modifies an existing book.
func (uc *BookUseCase) Update(ctx context.Context, id int64, in *dto.BookRequest) (*dto.BookResponse, error) {
	if in == nil {
		return nil, errno.New(errno.InvalidParams, "body is required")
	}
	b, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if b == nil {
		return nil, errno.New(errno.NotFound, "book not found: "+strconv.FormatInt(id, 10))
	}
	b.Title = in.Title
	b.DynastyID = in.DynastyID
	b.PublishedYear = in.PublishedYear
	b.Category = in.Category
	b.Summary = in.Summary
	b.VolumeCount = in.VolumeCount
	b.FileURL = in.FileURL
	if in.IsExtant != nil {
		b.IsExtant = *in.IsExtant
	}
	if err := uc.repo.Update(ctx, b); err != nil {
		return nil, err
	}
	return toBookResponse(b), nil
}

// Delete soft-deletes a book.
func (uc *BookUseCase) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return errno.New(errno.InvalidParams, "id is required")
	}
	return uc.repo.Delete(ctx, id)
}

// Get retrieves a single book by id.
func (uc *BookUseCase) Get(ctx context.Context, id int64) (*dto.BookResponse, error) {
	b, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if b == nil {
		return nil, errno.New(errno.NotFound, "book not found")
	}
	return toBookResponse(b), nil
}

// List returns a paginated list of books.
func (uc *BookUseCase) List(ctx context.Context, p pagination.Params) (dto.ListResponse[dto.BookResponse], error) {
	items, total, err := uc.repo.List(ctx, p)
	if err != nil {
		return dto.ListResponse[dto.BookResponse]{}, err
	}
	resp := make([]dto.BookResponse, 0, len(items))
	for i := range items {
		resp = append(resp, *toBookResponse(&items[i]))
	}
	return dto.NewListResponse(p, total, resp), nil
}

// Search keyword-matches books.
func (uc *BookUseCase) Search(ctx context.Context, keyword string, p pagination.Params) (dto.ListResponse[dto.BookResponse], error) {
	if keyword == "" {
		return uc.List(ctx, p)
	}
	items, total, err := uc.repo.Search(ctx, keyword, p)
	if err != nil {
		return dto.ListResponse[dto.BookResponse]{}, err
	}
	resp := make([]dto.BookResponse, 0, len(items))
	for i := range items {
		resp = append(resp, *toBookResponse(&items[i]))
	}
	return dto.NewListResponse(p, total, resp), nil
}

func toBookResponse(b *entity.Book) *dto.BookResponse {
	if b == nil {
		return nil
	}
	return &dto.BookResponse{
		ID:            b.ID,
		Title:         b.Title,
		DynastyID:     b.DynastyID,
		PublishedYear: b.PublishedYear,
		Category:      b.Category,
		Summary:       b.Summary,
		VolumeCount:   b.VolumeCount,
		IsExtant:      b.IsExtant,
		FileURL:       b.FileURL,
	}
}

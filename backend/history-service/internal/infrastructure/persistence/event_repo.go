package persistence

import (
	"context"

	"gorm.io/gorm"

	"tcm-history-ai/backend/history-service/internal/domain/entity"
	"tcm-history-ai/backend/history-service/internal/domain/repository"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/pagination"
)

// EventRepo implements repository.EventRepository with GORM.
type EventRepo struct {
	baseRepo
}

// NewEventRepo constructs an EventRepo.
func NewEventRepo(db *gorm.DB) *EventRepo {
	return &EventRepo{baseRepo{db: db}}
}

var _ repository.EventRepository = (*EventRepo)(nil)

// Create inserts a new history_event row.
func (r *EventRepo) Create(ctx context.Context, e *entity.Event) error {
	if err := txFrom(ctx, r.db).Create(e).Error; err != nil {
		return errno.Wrap(errno.InternalError, "create event", err)
	}
	return nil
}

// Update saves changes to an existing history_event row.
func (r *EventRepo) Update(ctx context.Context, e *entity.Event) error {
	res := txFrom(ctx, r.db).Save(e)
	if res.Error != nil {
		return errno.Wrap(errno.InternalError, "update event", res.Error)
	}
	if res.RowsAffected == 0 {
		return errno.New(errno.NotFound, "event not found")
	}
	return nil
}

// Delete soft-deletes a history_event by id.
func (r *EventRepo) Delete(ctx context.Context, id int64) error {
	res := txFrom(ctx, r.db).Where("id = ?", id).Delete(&entity.Event{})
	if res.Error != nil {
		return errno.Wrap(errno.InternalError, "delete event", res.Error)
	}
	if res.RowsAffected == 0 {
		return errno.New(errno.NotFound, "event not found")
	}
	return nil
}

// FindByID fetches a single history_event by id.
func (r *EventRepo) FindByID(ctx context.Context, id int64) (*entity.Event, error) {
	var e entity.Event
	err := txFrom(ctx, r.db).First(&e, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, errno.Wrap(errno.InternalError, "find event", err)
	}
	return &e, nil
}

// List returns a paginated list of history_event rows.
func (r *EventRepo) List(ctx context.Context, p pagination.Params) ([]entity.Event, int, error) {
	_, pageSize, offset := p.Normalise()
	var items []entity.Event
	var total int64
	db := txFrom(ctx, r.db).Model(&entity.Event{})
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "count event", err)
	}
	if err := db.Order("occurred_year ASC, id ASC").Limit(pageSize).Offset(offset).Find(&items).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "list event", err)
	}
	return items, int(total), nil
}

// Search keyword-matches history_event rows on title, description, location.
func (r *EventRepo) Search(ctx context.Context, keyword string, p pagination.Params) ([]entity.Event, int, error) {
	_, pageSize, offset := p.Normalise()
	pattern := "%" + keyword + "%"
	var items []entity.Event
	var total int64
	db := txFrom(ctx, r.db).Model(&entity.Event{}).
		Where("title ILIKE ? OR description ILIKE ? OR location ILIKE ?", pattern, pattern, pattern)
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "count event search", err)
	}
	if err := db.Order("occurred_year ASC, id ASC").Limit(pageSize).Offset(offset).Find(&items).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "search event", err)
	}
	return items, int(total), nil
}

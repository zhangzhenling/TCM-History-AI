package controller_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"tcm-history-ai/backend/history-service/internal/application/usecase"
	"tcm-history-ai/backend/history-service/internal/controller"
	"tcm-history-ai/backend/history-service/internal/domain/entity"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/pagination"
)

// ============================================================================
// SchoolController
// ============================================================================

func newSchoolController(repo *mockSchoolRepo) *controller.SchoolController {
	uc := usecase.NewSchoolUseCase(repo)
	return controller.NewSchoolController(uc)
}

func seedSchool(repo *mockSchoolRepo, id int64, name string) *entity.School {
	s := &entity.School{Name: name}
	s.ID = id
	repo.items[id] = s
	return s
}

func TestSchoolController_List(t *testing.T) {
	t.Run("returns 200 with seeded schools", func(t *testing.T) {
		repo := newMockSchoolRepo()
		seedSchool(repo, 1, "Jian'an School")
		ctrl := newSchoolController(repo)

		rc := newRC()
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/history/schools?page=1&page_size=10")

		ctrl.List(ctx(), rc)
		body := assertStatusCode(t, rc, http.StatusOK)
		assert.Equal(t, int(errno.OK), body.Code)
	})

	t.Run("returns 200 on empty result", func(t *testing.T) {
		repo := newMockSchoolRepo()
		ctrl := newSchoolController(repo)

		rc := newRC()
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/history/schools")

		ctrl.List(ctx(), rc)
		assertStatusCode(t, rc, http.StatusOK)
	})

	t.Run("propagates repo error as 500", func(t *testing.T) {
		repo := newMockSchoolRepo()
		repo.list = func(_ pagination.Params) ([]entity.School, int, error) {
			return nil, 0, errno.New(errno.InternalError, "db down")
		}
		ctrl := newSchoolController(repo)

		rc := newRC()
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/history/schools")

		ctrl.List(ctx(), rc)
		assert.Equal(t, http.StatusInternalServerError, rc.Response.StatusCode())
	})
}

func TestSchoolController_Create(t *testing.T) {
	t.Run("happy path returns 201", func(t *testing.T) {
		repo := newMockSchoolRepo()
		ctrl := newSchoolController(repo)

		rc := newRC()
		rc.Request.SetMethod("POST")
		rc.Request.SetRequestURI("/api/v1/history/schools")
		rc.Request.SetBody([]byte(`{"name":"Jian'an School","dynasty_id":1,"established_year":196}`))

		ctrl.Create(ctx(), rc)
		body := assertStatusCode(t, rc, http.StatusCreated)
		assert.Equal(t, "created", body.Message)
	})

	t.Run("invalid JSON returns 400", func(t *testing.T) {
		repo := newMockSchoolRepo()
		ctrl := newSchoolController(repo)

		rc := newRC()
		rc.Request.SetMethod("POST")
		rc.Request.SetRequestURI("/api/v1/history/schools")
		rc.Request.SetBody([]byte(`{not-json`))

		ctrl.Create(ctx(), rc)
		assertStatusCode(t, rc, http.StatusBadRequest)
	})

	t.Run("empty name returns 400", func(t *testing.T) {
		repo := newMockSchoolRepo()
		ctrl := newSchoolController(repo)

		rc := newRC()
		rc.Request.SetMethod("POST")
		rc.Request.SetRequestURI("/api/v1/history/schools")
		rc.Request.SetBody([]byte(`{"name":""}`))

		ctrl.Create(ctx(), rc)
		assertStatusCode(t, rc, http.StatusBadRequest)
	})
}

func TestSchoolController_Get(t *testing.T) {
	t.Run("found returns 200", func(t *testing.T) {
		repo := newMockSchoolRepo()
		seedSchool(repo, 1, "Jian'an School")
		ctrl := newSchoolController(repo)

		rc := newRC()
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/history/schools/1")
		setParam(rc, "id", "1")

		ctrl.Get(ctx(), rc)
		assertStatusCode(t, rc, http.StatusOK)
	})

	t.Run("invalid id returns 400", func(t *testing.T) {
		repo := newMockSchoolRepo()
		ctrl := newSchoolController(repo)

		rc := newRC()
		setParam(rc, "id", "abc")

		ctrl.Get(ctx(), rc)
		assertStatusCode(t, rc, http.StatusBadRequest)
	})

	t.Run("not found returns 404", func(t *testing.T) {
		repo := newMockSchoolRepo()
		ctrl := newSchoolController(repo)

		rc := newRC()
		setParam(rc, "id", "999")

		ctrl.Get(ctx(), rc)
		assertStatusCode(t, rc, http.StatusNotFound)
	})
}

func TestSchoolController_Update(t *testing.T) {
	t.Run("happy path returns 200", func(t *testing.T) {
		repo := newMockSchoolRepo()
		seedSchool(repo, 1, "Jian'an School")
		ctrl := newSchoolController(repo)

		rc := newRC()
		rc.Request.SetMethod("PUT")
		rc.Request.SetRequestURI("/api/v1/history/schools/1")
		setParam(rc, "id", "1")
		rc.Request.SetBody([]byte(`{"name":"Updated School"}`))

		ctrl.Update(ctx(), rc)
		assertStatusCode(t, rc, http.StatusOK)
	})

	t.Run("invalid id returns 400", func(t *testing.T) {
		repo := newMockSchoolRepo()
		ctrl := newSchoolController(repo)

		rc := newRC()
		setParam(rc, "id", "0")

		ctrl.Update(ctx(), rc)
		assertStatusCode(t, rc, http.StatusBadRequest)
	})

	t.Run("not found returns 404", func(t *testing.T) {
		repo := newMockSchoolRepo()
		ctrl := newSchoolController(repo)

		rc := newRC()
		setParam(rc, "id", "999")
		rc.Request.SetBody([]byte(`{"name":"X"}`))

		ctrl.Update(ctx(), rc)
		assertStatusCode(t, rc, http.StatusNotFound)
	})

	t.Run("invalid JSON returns 400", func(t *testing.T) {
		repo := newMockSchoolRepo()
		seedSchool(repo, 1, "Jian'an School")
		ctrl := newSchoolController(repo)

		rc := newRC()
		setParam(rc, "id", "1")
		rc.Request.SetBody([]byte(`{not-json`))

		ctrl.Update(ctx(), rc)
		assertStatusCode(t, rc, http.StatusBadRequest)
	})
}

func TestSchoolController_Delete(t *testing.T) {
	t.Run("happy path returns 204", func(t *testing.T) {
		repo := newMockSchoolRepo()
		seedSchool(repo, 1, "Jian'an School")
		ctrl := newSchoolController(repo)

		rc := newRC()
		rc.Request.SetMethod("DELETE")
		rc.Request.SetRequestURI("/api/v1/history/schools/1")
		setParam(rc, "id", "1")

		ctrl.Delete(ctx(), rc)
		assertStatusCode(t, rc, http.StatusNoContent)
	})

	t.Run("invalid id returns 400", func(t *testing.T) {
		repo := newMockSchoolRepo()
		ctrl := newSchoolController(repo)

		rc := newRC()
		setParam(rc, "id", "abc")

		ctrl.Delete(ctx(), rc)
		assertStatusCode(t, rc, http.StatusBadRequest)
	})

	t.Run("not found returns 404", func(t *testing.T) {
		repo := newMockSchoolRepo()
		ctrl := newSchoolController(repo)

		rc := newRC()
		setParam(rc, "id", "999")

		ctrl.Delete(ctx(), rc)
		assertStatusCode(t, rc, http.StatusNotFound)
	})
}

// ============================================================================
// BookController
// ============================================================================

func newBookController(repo *mockBookRepo) *controller.BookController {
	uc := usecase.NewBookUseCase(repo, nil)
	return controller.NewBookController(uc)
}

func seedBook(repo *mockBookRepo, id int64, title string) *entity.Book {
	b := &entity.Book{Title: title}
	b.ID = id
	repo.items[id] = b
	return b
}

func TestBookController_List(t *testing.T) {
	t.Run("returns 200 with seeded books", func(t *testing.T) {
		repo := newMockBookRepo()
		seedBook(repo, 1, "Shanghan Lun")
		ctrl := newBookController(repo)

		rc := newRC()
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/history/books?page=1&page_size=10")

		ctrl.List(ctx(), rc)
		body := assertStatusCode(t, rc, http.StatusOK)
		assert.Equal(t, int(errno.OK), body.Code)
	})

	t.Run("returns 200 on empty result", func(t *testing.T) {
		repo := newMockBookRepo()
		ctrl := newBookController(repo)

		rc := newRC()
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/history/books")

		ctrl.List(ctx(), rc)
		assertStatusCode(t, rc, http.StatusOK)
	})

	t.Run("propagates repo error as 500", func(t *testing.T) {
		repo := newMockBookRepo()
		repo.list = func(_ pagination.Params) ([]entity.Book, int, error) {
			return nil, 0, errno.New(errno.InternalError, "db down")
		}
		ctrl := newBookController(repo)

		rc := newRC()
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/history/books")

		ctrl.List(ctx(), rc)
		assert.Equal(t, http.StatusInternalServerError, rc.Response.StatusCode())
	})
}

func TestBookController_Create(t *testing.T) {
	t.Run("happy path returns 201", func(t *testing.T) {
		repo := newMockBookRepo()
		ctrl := newBookController(repo)

		rc := newRC()
		rc.Request.SetMethod("POST")
		rc.Request.SetRequestURI("/api/v1/history/books")
		rc.Request.SetBody([]byte(`{"title":"Shanghan Lun","dynasty_id":1,"category":"classic"}`))

		ctrl.Create(ctx(), rc)
		body := assertStatusCode(t, rc, http.StatusCreated)
		assert.Equal(t, "created", body.Message)
	})

	t.Run("invalid JSON returns 400", func(t *testing.T) {
		repo := newMockBookRepo()
		ctrl := newBookController(repo)

		rc := newRC()
		rc.Request.SetMethod("POST")
		rc.Request.SetRequestURI("/api/v1/history/books")
		rc.Request.SetBody([]byte(`{not-json`))

		ctrl.Create(ctx(), rc)
		assertStatusCode(t, rc, http.StatusBadRequest)
	})

	t.Run("empty title returns 400", func(t *testing.T) {
		repo := newMockBookRepo()
		ctrl := newBookController(repo)

		rc := newRC()
		rc.Request.SetMethod("POST")
		rc.Request.SetRequestURI("/api/v1/history/books")
		rc.Request.SetBody([]byte(`{"title":""}`))

		ctrl.Create(ctx(), rc)
		assertStatusCode(t, rc, http.StatusBadRequest)
	})
}

func TestBookController_Get(t *testing.T) {
	t.Run("found returns 200", func(t *testing.T) {
		repo := newMockBookRepo()
		seedBook(repo, 1, "Shanghan Lun")
		ctrl := newBookController(repo)

		rc := newRC()
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/history/books/1")
		setParam(rc, "id", "1")

		ctrl.Get(ctx(), rc)
		assertStatusCode(t, rc, http.StatusOK)
	})

	t.Run("invalid id returns 400", func(t *testing.T) {
		repo := newMockBookRepo()
		ctrl := newBookController(repo)

		rc := newRC()
		setParam(rc, "id", "abc")

		ctrl.Get(ctx(), rc)
		assertStatusCode(t, rc, http.StatusBadRequest)
	})

	t.Run("not found returns 404", func(t *testing.T) {
		repo := newMockBookRepo()
		ctrl := newBookController(repo)

		rc := newRC()
		setParam(rc, "id", "999")

		ctrl.Get(ctx(), rc)
		assertStatusCode(t, rc, http.StatusNotFound)
	})
}

func TestBookController_Update(t *testing.T) {
	t.Run("happy path returns 200", func(t *testing.T) {
		repo := newMockBookRepo()
		seedBook(repo, 1, "Shanghan Lun")
		ctrl := newBookController(repo)

		rc := newRC()
		rc.Request.SetMethod("PUT")
		rc.Request.SetRequestURI("/api/v1/history/books/1")
		setParam(rc, "id", "1")
		rc.Request.SetBody([]byte(`{"title":"Updated Title"}`))

		ctrl.Update(ctx(), rc)
		assertStatusCode(t, rc, http.StatusOK)
	})

	t.Run("invalid id returns 400", func(t *testing.T) {
		repo := newMockBookRepo()
		ctrl := newBookController(repo)

		rc := newRC()
		setParam(rc, "id", "0")

		ctrl.Update(ctx(), rc)
		assertStatusCode(t, rc, http.StatusBadRequest)
	})

	t.Run("not found returns 404", func(t *testing.T) {
		repo := newMockBookRepo()
		ctrl := newBookController(repo)

		rc := newRC()
		setParam(rc, "id", "999")
		rc.Request.SetBody([]byte(`{"title":"X"}`))

		ctrl.Update(ctx(), rc)
		assertStatusCode(t, rc, http.StatusNotFound)
	})

	t.Run("invalid JSON returns 400", func(t *testing.T) {
		repo := newMockBookRepo()
		seedBook(repo, 1, "Shanghan Lun")
		ctrl := newBookController(repo)

		rc := newRC()
		setParam(rc, "id", "1")
		rc.Request.SetBody([]byte(`{not-json`))

		ctrl.Update(ctx(), rc)
		assertStatusCode(t, rc, http.StatusBadRequest)
	})
}

func TestBookController_Delete(t *testing.T) {
	t.Run("happy path returns 204", func(t *testing.T) {
		repo := newMockBookRepo()
		seedBook(repo, 1, "Shanghan Lun")
		ctrl := newBookController(repo)

		rc := newRC()
		rc.Request.SetMethod("DELETE")
		rc.Request.SetRequestURI("/api/v1/history/books/1")
		setParam(rc, "id", "1")

		ctrl.Delete(ctx(), rc)
		assertStatusCode(t, rc, http.StatusNoContent)
	})

	t.Run("invalid id returns 400", func(t *testing.T) {
		repo := newMockBookRepo()
		ctrl := newBookController(repo)

		rc := newRC()
		setParam(rc, "id", "abc")

		ctrl.Delete(ctx(), rc)
		assertStatusCode(t, rc, http.StatusBadRequest)
	})

	t.Run("not found returns 404", func(t *testing.T) {
		repo := newMockBookRepo()
		ctrl := newBookController(repo)

		rc := newRC()
		setParam(rc, "id", "999")

		ctrl.Delete(ctx(), rc)
		assertStatusCode(t, rc, http.StatusNotFound)
	})
}

// ============================================================================
// EventController
// ============================================================================

func newEventController(repo *mockEventRepo) *controller.EventController {
	uc := usecase.NewEventUseCase(repo)
	return controller.NewEventController(uc)
}

func seedEvent(repo *mockEventRepo, id int64, title string) *entity.Event {
	e := &entity.Event{Title: title}
	e.ID = id
	repo.items[id] = e
	return e
}

func TestEventController_List(t *testing.T) {
	t.Run("returns 200 with seeded events", func(t *testing.T) {
		repo := newMockEventRepo()
		seedEvent(repo, 1, "Publication of Shanghan Lun")
		ctrl := newEventController(repo)

		rc := newRC()
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/history/events?page=1&page_size=10")

		ctrl.List(ctx(), rc)
		body := assertStatusCode(t, rc, http.StatusOK)
		assert.Equal(t, int(errno.OK), body.Code)
	})

	t.Run("returns 200 on empty result", func(t *testing.T) {
		repo := newMockEventRepo()
		ctrl := newEventController(repo)

		rc := newRC()
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/history/events")

		ctrl.List(ctx(), rc)
		assertStatusCode(t, rc, http.StatusOK)
	})

	t.Run("propagates repo error as 500", func(t *testing.T) {
		repo := newMockEventRepo()
		repo.list = func(_ pagination.Params) ([]entity.Event, int, error) {
			return nil, 0, errno.New(errno.InternalError, "db down")
		}
		ctrl := newEventController(repo)

		rc := newRC()
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/history/events")

		ctrl.List(ctx(), rc)
		assert.Equal(t, http.StatusInternalServerError, rc.Response.StatusCode())
	})
}

func TestEventController_Create(t *testing.T) {
	t.Run("happy path returns 201", func(t *testing.T) {
		repo := newMockEventRepo()
		ctrl := newEventController(repo)

		rc := newRC()
		rc.Request.SetMethod("POST")
		rc.Request.SetRequestURI("/api/v1/history/events")
		rc.Request.SetBody([]byte(`{"title":"Publication of Shanghan Lun","event_type":"publish","occurred_year":210}`))

		ctrl.Create(ctx(), rc)
		body := assertStatusCode(t, rc, http.StatusCreated)
		assert.Equal(t, "created", body.Message)
	})

	t.Run("invalid JSON returns 400", func(t *testing.T) {
		repo := newMockEventRepo()
		ctrl := newEventController(repo)

		rc := newRC()
		rc.Request.SetMethod("POST")
		rc.Request.SetRequestURI("/api/v1/history/events")
		rc.Request.SetBody([]byte(`{not-json`))

		ctrl.Create(ctx(), rc)
		assertStatusCode(t, rc, http.StatusBadRequest)
	})

	t.Run("empty title returns 400", func(t *testing.T) {
		repo := newMockEventRepo()
		ctrl := newEventController(repo)

		rc := newRC()
		rc.Request.SetMethod("POST")
		rc.Request.SetRequestURI("/api/v1/history/events")
		rc.Request.SetBody([]byte(`{"title":""}`))

		ctrl.Create(ctx(), rc)
		assertStatusCode(t, rc, http.StatusBadRequest)
	})
}

func TestEventController_Get(t *testing.T) {
	t.Run("found returns 200", func(t *testing.T) {
		repo := newMockEventRepo()
		seedEvent(repo, 1, "Publication")
		ctrl := newEventController(repo)

		rc := newRC()
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/history/events/1")
		setParam(rc, "id", "1")

		ctrl.Get(ctx(), rc)
		assertStatusCode(t, rc, http.StatusOK)
	})

	t.Run("invalid id returns 400", func(t *testing.T) {
		repo := newMockEventRepo()
		ctrl := newEventController(repo)

		rc := newRC()
		setParam(rc, "id", "abc")

		ctrl.Get(ctx(), rc)
		assertStatusCode(t, rc, http.StatusBadRequest)
	})

	t.Run("not found returns 404", func(t *testing.T) {
		repo := newMockEventRepo()
		ctrl := newEventController(repo)

		rc := newRC()
		setParam(rc, "id", "999")

		ctrl.Get(ctx(), rc)
		assertStatusCode(t, rc, http.StatusNotFound)
	})
}

func TestEventController_Update(t *testing.T) {
	t.Run("happy path returns 200", func(t *testing.T) {
		repo := newMockEventRepo()
		seedEvent(repo, 1, "Publication")
		ctrl := newEventController(repo)

		rc := newRC()
		rc.Request.SetMethod("PUT")
		rc.Request.SetRequestURI("/api/v1/history/events/1")
		setParam(rc, "id", "1")
		rc.Request.SetBody([]byte(`{"title":"Updated Event"}`))

		ctrl.Update(ctx(), rc)
		assertStatusCode(t, rc, http.StatusOK)
	})

	t.Run("invalid id returns 400", func(t *testing.T) {
		repo := newMockEventRepo()
		ctrl := newEventController(repo)

		rc := newRC()
		setParam(rc, "id", "0")

		ctrl.Update(ctx(), rc)
		assertStatusCode(t, rc, http.StatusBadRequest)
	})

	t.Run("not found returns 404", func(t *testing.T) {
		repo := newMockEventRepo()
		ctrl := newEventController(repo)

		rc := newRC()
		setParam(rc, "id", "999")
		rc.Request.SetBody([]byte(`{"title":"X"}`))

		ctrl.Update(ctx(), rc)
		assertStatusCode(t, rc, http.StatusNotFound)
	})

	t.Run("invalid JSON returns 400", func(t *testing.T) {
		repo := newMockEventRepo()
		seedEvent(repo, 1, "Publication")
		ctrl := newEventController(repo)

		rc := newRC()
		setParam(rc, "id", "1")
		rc.Request.SetBody([]byte(`{not-json`))

		ctrl.Update(ctx(), rc)
		assertStatusCode(t, rc, http.StatusBadRequest)
	})
}

func TestEventController_Delete(t *testing.T) {
	t.Run("happy path returns 204", func(t *testing.T) {
		repo := newMockEventRepo()
		seedEvent(repo, 1, "Publication")
		ctrl := newEventController(repo)

		rc := newRC()
		rc.Request.SetMethod("DELETE")
		rc.Request.SetRequestURI("/api/v1/history/events/1")
		setParam(rc, "id", "1")

		ctrl.Delete(ctx(), rc)
		assertStatusCode(t, rc, http.StatusNoContent)
	})

	t.Run("invalid id returns 400", func(t *testing.T) {
		repo := newMockEventRepo()
		ctrl := newEventController(repo)

		rc := newRC()
		setParam(rc, "id", "abc")

		ctrl.Delete(ctx(), rc)
		assertStatusCode(t, rc, http.StatusBadRequest)
	})

	t.Run("not found returns 404", func(t *testing.T) {
		repo := newMockEventRepo()
		ctrl := newEventController(repo)

		rc := newRC()
		setParam(rc, "id", "999")

		ctrl.Delete(ctx(), rc)
		assertStatusCode(t, rc, http.StatusNotFound)
	})
}

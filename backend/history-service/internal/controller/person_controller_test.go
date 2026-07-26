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

// newPersonController wires a PersonController to an in-memory repo.
func newPersonController(repo *mockPersonRepo) *controller.PersonController {
	uc := usecase.NewPersonUseCase(repo, nil)
	return controller.NewPersonController(uc)
}

func seedPerson(repo *mockPersonRepo, id int64, name string) *entity.Person {
	p := &entity.Person{Name: name}
	p.ID = id
	repo.items[id] = p
	return p
}

func TestPersonController_List(t *testing.T) {
	t.Run("returns 200 with seeded persons", func(t *testing.T) {
		repo := newMockPersonRepo()
		seedPerson(repo, 1, "Zhang Zhongjing")
		seedPerson(repo, 2, "Hua Tuo")
		ctrl := newPersonController(repo)

		rc := newRC()
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/history/persons?page=1&page_size=10")

		ctrl.List(ctx(), rc)
		body := assertStatusCode(t, rc, http.StatusOK)
		assert.Equal(t, int(errno.OK), body.Code)
	})

	t.Run("returns 200 on empty result", func(t *testing.T) {
		repo := newMockPersonRepo()
		ctrl := newPersonController(repo)

		rc := newRC()
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/history/persons")

		ctrl.List(ctx(), rc)
		assertStatusCode(t, rc, http.StatusOK)
	})

	t.Run("propagates repo error as 500", func(t *testing.T) {
		repo := newMockPersonRepo()
		repo.list = func(_ pagination.Params) ([]entity.Person, int, error) {
			return nil, 0, errno.New(errno.InternalError, "db down")
		}
		ctrl := newPersonController(repo)

		rc := newRC()
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/history/persons")

		ctrl.List(ctx(), rc)
		assert.Equal(t, http.StatusInternalServerError, rc.Response.StatusCode())
	})
}

func TestPersonController_Create(t *testing.T) {
	t.Run("happy path returns 201", func(t *testing.T) {
		repo := newMockPersonRepo()
		ctrl := newPersonController(repo)

		rc := newRC()
		rc.Request.SetMethod("POST")
		rc.Request.SetRequestURI("/api/v1/history/persons")
		rc.Request.SetBody([]byte(`{"name":"Zhang Zhongjing","dynasty_id":1,"gender":"male"}`))

		ctrl.Create(ctx(), rc)
		body := assertStatusCode(t, rc, http.StatusCreated)
		assert.Equal(t, "created", body.Message)
	})

	t.Run("invalid JSON returns 400", func(t *testing.T) {
		repo := newMockPersonRepo()
		ctrl := newPersonController(repo)

		rc := newRC()
		rc.Request.SetMethod("POST")
		rc.Request.SetRequestURI("/api/v1/history/persons")
		rc.Request.SetBody([]byte(`{not-json`))

		ctrl.Create(ctx(), rc)
		assertStatusCode(t, rc, http.StatusBadRequest)
	})

	t.Run("empty name returns 400", func(t *testing.T) {
		repo := newMockPersonRepo()
		ctrl := newPersonController(repo)

		rc := newRC()
		rc.Request.SetMethod("POST")
		rc.Request.SetRequestURI("/api/v1/history/persons")
		rc.Request.SetBody([]byte(`{"name":""}`))

		ctrl.Create(ctx(), rc)
		assertStatusCode(t, rc, http.StatusBadRequest)
	})

	t.Run("invalid gender returns 400", func(t *testing.T) {
		repo := newMockPersonRepo()
		ctrl := newPersonController(repo)

		rc := newRC()
		rc.Request.SetMethod("POST")
		rc.Request.SetRequestURI("/api/v1/history/persons")
		rc.Request.SetBody([]byte(`{"name":"X","gender":"invalid"}`))

		ctrl.Create(ctx(), rc)
		assertStatusCode(t, rc, http.StatusBadRequest)
	})
}

func TestPersonController_Get(t *testing.T) {
	t.Run("found returns 200", func(t *testing.T) {
		repo := newMockPersonRepo()
		seedPerson(repo, 1, "Hua Tuo")
		ctrl := newPersonController(repo)

		rc := newRC()
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/history/persons/1")
		setParam(rc, "id", "1")

		ctrl.Get(ctx(), rc)
		assertStatusCode(t, rc, http.StatusOK)
	})

	t.Run("invalid id returns 400", func(t *testing.T) {
		repo := newMockPersonRepo()
		ctrl := newPersonController(repo)

		rc := newRC()
		setParam(rc, "id", "abc")

		ctrl.Get(ctx(), rc)
		assertStatusCode(t, rc, http.StatusBadRequest)
	})

	t.Run("not found returns 404", func(t *testing.T) {
		repo := newMockPersonRepo()
		ctrl := newPersonController(repo)

		rc := newRC()
		setParam(rc, "id", "999")

		ctrl.Get(ctx(), rc)
		assertStatusCode(t, rc, http.StatusNotFound)
	})
}

func TestPersonController_Update(t *testing.T) {
	t.Run("happy path returns 200", func(t *testing.T) {
		repo := newMockPersonRepo()
		seedPerson(repo, 1, "Hua Tuo")
		ctrl := newPersonController(repo)

		rc := newRC()
		rc.Request.SetMethod("PUT")
		rc.Request.SetRequestURI("/api/v1/history/persons/1")
		setParam(rc, "id", "1")
		rc.Request.SetBody([]byte(`{"name":"Hua Tuo Updated"}`))

		ctrl.Update(ctx(), rc)
		assertStatusCode(t, rc, http.StatusOK)
	})

	t.Run("invalid id returns 400", func(t *testing.T) {
		repo := newMockPersonRepo()
		ctrl := newPersonController(repo)

		rc := newRC()
		setParam(rc, "id", "0")

		ctrl.Update(ctx(), rc)
		assertStatusCode(t, rc, http.StatusBadRequest)
	})

	t.Run("not found returns 404", func(t *testing.T) {
		repo := newMockPersonRepo()
		ctrl := newPersonController(repo)

		rc := newRC()
		setParam(rc, "id", "999")
		rc.Request.SetBody([]byte(`{"name":"X"}`))

		ctrl.Update(ctx(), rc)
		assertStatusCode(t, rc, http.StatusNotFound)
	})

	t.Run("invalid JSON returns 400", func(t *testing.T) {
		repo := newMockPersonRepo()
		seedPerson(repo, 1, "Hua Tuo")
		ctrl := newPersonController(repo)

		rc := newRC()
		setParam(rc, "id", "1")
		rc.Request.SetBody([]byte(`{not-json`))

		ctrl.Update(ctx(), rc)
		assertStatusCode(t, rc, http.StatusBadRequest)
	})

	t.Run("invalid gender returns 400", func(t *testing.T) {
		repo := newMockPersonRepo()
		seedPerson(repo, 1, "Hua Tuo")
		ctrl := newPersonController(repo)

		rc := newRC()
		setParam(rc, "id", "1")
		rc.Request.SetBody([]byte(`{"name":"X","gender":"bad"}`))

		ctrl.Update(ctx(), rc)
		assertStatusCode(t, rc, http.StatusBadRequest)
	})
}

func TestPersonController_Delete(t *testing.T) {
	t.Run("happy path returns 204", func(t *testing.T) {
		repo := newMockPersonRepo()
		seedPerson(repo, 1, "Hua Tuo")
		ctrl := newPersonController(repo)

		rc := newRC()
		rc.Request.SetMethod("DELETE")
		rc.Request.SetRequestURI("/api/v1/history/persons/1")
		setParam(rc, "id", "1")

		ctrl.Delete(ctx(), rc)
		assertStatusCode(t, rc, http.StatusNoContent)
	})

	t.Run("invalid id returns 400", func(t *testing.T) {
		repo := newMockPersonRepo()
		ctrl := newPersonController(repo)

		rc := newRC()
		setParam(rc, "id", "abc")

		ctrl.Delete(ctx(), rc)
		assertStatusCode(t, rc, http.StatusBadRequest)
	})

	t.Run("not found returns 404", func(t *testing.T) {
		repo := newMockPersonRepo()
		ctrl := newPersonController(repo)

		rc := newRC()
		setParam(rc, "id", "999")

		ctrl.Delete(ctx(), rc)
		assertStatusCode(t, rc, http.StatusNotFound)
	})
}

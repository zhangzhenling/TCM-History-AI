package controller_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tcm-history-ai/backend/history-service/internal/application/usecase"
	"tcm-history-ai/backend/history-service/internal/controller"
	"tcm-history-ai/backend/history-service/internal/domain/entity"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/pagination"
)

// newDynastyController builds a DynastyController wired to an in-memory repo.
func newDynastyController(repo *mockDynastyRepo) *controller.DynastyController {
	uc := usecase.NewDynastyUseCase(repo, nil)
	return controller.NewDynastyController(uc)
}

func seedDynasty(repo *mockDynastyRepo, id int64, name string) *entity.Dynasty {
	d := &entity.Dynasty{Name: name}
	d.ID = id
	repo.items[id] = d
	return d
}

// TestDynastyController_List covers the pagination list endpoint.
func TestDynastyController_List(t *testing.T) {
	t.Run("returns 200 with seeded dynasties", func(t *testing.T) {
		repo := newMockDynastyRepo()
		seedDynasty(repo, 1, "Han")
		seedDynasty(repo, 2, "Tang")
		ctrl := newDynastyController(repo)

		rc := newRC()
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/history/dynasties?page=1&page_size=10")

		ctrl.List(ctx(), rc)

		body := assertStatusCode(t, rc, http.StatusOK)
		assert.Equal(t, int(errno.OK), body.Code)
	})

	t.Run("returns 200 on empty result", func(t *testing.T) {
		repo := newMockDynastyRepo()
		ctrl := newDynastyController(repo)

		rc := newRC()
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/history/dynasties")

		ctrl.List(ctx(), rc)
		assertStatusCode(t, rc, http.StatusOK)
	})

	t.Run("propagates repo error as 500", func(t *testing.T) {
		repo := newMockDynastyRepo()
		repo.list = func(_ pagination.Params) ([]entity.Dynasty, int, error) {
			return nil, 0, errno.New(errno.InternalError, "db down")
		}
		ctrl := newDynastyController(repo)

		rc := newRC()
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/history/dynasties")

		ctrl.List(ctx(), rc)
		assert.Equal(t, http.StatusInternalServerError, rc.Response.StatusCode())
	})
}

// TestDynastyController_Create covers the create endpoint.
func TestDynastyController_Create(t *testing.T) {
	t.Run("happy path returns 201", func(t *testing.T) {
		repo := newMockDynastyRepo()
		ctrl := newDynastyController(repo)

		rc := newRC()
		rc.Request.SetMethod("POST")
		rc.Request.SetRequestURI("/api/v1/history/dynasties")
		rc.Request.SetBody([]byte(`{"name":"Han","start_year":-202,"end_year":220,"sort_order":5}`))

		ctrl.Create(ctx(), rc)

		body := assertStatusCode(t, rc, http.StatusCreated)
		assert.Equal(t, "created", body.Message)
	})

	t.Run("invalid JSON returns 400", func(t *testing.T) {
		repo := newMockDynastyRepo()
		ctrl := newDynastyController(repo)

		rc := newRC()
		rc.Request.SetMethod("POST")
		rc.Request.SetRequestURI("/api/v1/history/dynasties")
		rc.Request.SetBody([]byte(`{not-json`))

		ctrl.Create(ctx(), rc)
		assertStatusCode(t, rc, http.StatusBadRequest)
	})

	t.Run("validation error (empty name) returns 400", func(t *testing.T) {
		repo := newMockDynastyRepo()
		ctrl := newDynastyController(repo)

		rc := newRC()
		rc.Request.SetMethod("POST")
		rc.Request.SetRequestURI("/api/v1/history/dynasties")
		rc.Request.SetBody([]byte(`{"name":""}`))

		ctrl.Create(ctx(), rc)
		assertStatusCode(t, rc, http.StatusBadRequest)
	})
}

// TestDynastyController_Get covers get by id.
func TestDynastyController_Get(t *testing.T) {
	t.Run("found returns 200", func(t *testing.T) {
		repo := newMockDynastyRepo()
		seedDynasty(repo, 1, "Han")
		ctrl := newDynastyController(repo)

		rc := newRC()
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/history/dynasties/1")
		setParam(rc, "id", "1")

		ctrl.Get(ctx(), rc)
		assertStatusCode(t, rc, http.StatusOK)
	})

	t.Run("invalid id returns 400", func(t *testing.T) {
		repo := newMockDynastyRepo()
		ctrl := newDynastyController(repo)

		rc := newRC()
		setParam(rc, "id", "abc")

		ctrl.Get(ctx(), rc)
		assertStatusCode(t, rc, http.StatusBadRequest)
	})

	t.Run("not found returns 404", func(t *testing.T) {
		repo := newMockDynastyRepo()
		ctrl := newDynastyController(repo)

		rc := newRC()
		setParam(rc, "id", "999")

		ctrl.Get(ctx(), rc)
		assertStatusCode(t, rc, http.StatusNotFound)
	})
}

// TestDynastyController_Update covers update by id.
func TestDynastyController_Update(t *testing.T) {
	t.Run("happy path returns 200", func(t *testing.T) {
		repo := newMockDynastyRepo()
		seedDynasty(repo, 1, "Han")
		ctrl := newDynastyController(repo)

		rc := newRC()
		rc.Request.SetMethod("PUT")
		rc.Request.SetRequestURI("/api/v1/history/dynasties/1")
		setParam(rc, "id", "1")
		rc.Request.SetBody([]byte(`{"name":"Han Updated"}`))

		ctrl.Update(ctx(), rc)
		assertStatusCode(t, rc, http.StatusOK)
	})

	t.Run("invalid id returns 400", func(t *testing.T) {
		repo := newMockDynastyRepo()
		ctrl := newDynastyController(repo)

		rc := newRC()
		setParam(rc, "id", "0")

		ctrl.Update(ctx(), rc)
		assertStatusCode(t, rc, http.StatusBadRequest)
	})

	t.Run("not found returns 404", func(t *testing.T) {
		repo := newMockDynastyRepo()
		ctrl := newDynastyController(repo)

		rc := newRC()
		setParam(rc, "id", "999")
		rc.Request.SetBody([]byte(`{"name":"X"}`))

		ctrl.Update(ctx(), rc)
		assertStatusCode(t, rc, http.StatusNotFound)
	})

	t.Run("invalid JSON returns 400", func(t *testing.T) {
		repo := newMockDynastyRepo()
		seedDynasty(repo, 1, "Han")
		ctrl := newDynastyController(repo)

		rc := newRC()
		setParam(rc, "id", "1")
		rc.Request.SetBody([]byte(`{not-json`))

		ctrl.Update(ctx(), rc)
		assertStatusCode(t, rc, http.StatusBadRequest)
	})
}

// TestDynastyController_Delete covers delete by id.
func TestDynastyController_Delete(t *testing.T) {
	t.Run("happy path returns 204", func(t *testing.T) {
		repo := newMockDynastyRepo()
		seedDynasty(repo, 1, "Han")
		ctrl := newDynastyController(repo)

		rc := newRC()
		rc.Request.SetMethod("DELETE")
		rc.Request.SetRequestURI("/api/v1/history/dynasties/1")
		setParam(rc, "id", "1")

		ctrl.Delete(ctx(), rc)
		assertStatusCode(t, rc, http.StatusNoContent)
	})

	t.Run("invalid id returns 400", func(t *testing.T) {
		repo := newMockDynastyRepo()
		ctrl := newDynastyController(repo)

		rc := newRC()
		setParam(rc, "id", "abc")

		ctrl.Delete(ctx(), rc)
		assertStatusCode(t, rc, http.StatusBadRequest)
	})

	t.Run("not found returns 404", func(t *testing.T) {
		repo := newMockDynastyRepo()
		ctrl := newDynastyController(repo)

		rc := newRC()
		setParam(rc, "id", "999")

		ctrl.Delete(ctx(), rc)
		assertStatusCode(t, rc, http.StatusNotFound)
	})
}

// TestNewDynastyController_NotNil verifies the constructor returns a non-nil
// controller.
func TestNewDynastyController_NotNil(t *testing.T) {
	repo := newMockDynastyRepo()
	uc := usecase.NewDynastyUseCase(repo, nil)
	ctrl := controller.NewDynastyController(uc)
	require.NotNil(t, ctrl)
}

// TestDynastyController_BindAndValidate_DynastyRequest ensures the request
// struct tag uses the expected name field.
func TestDynastyController_BindAndValidate_DynastyRequest(t *testing.T) {
	repo := newMockDynastyRepo()
	ctrl := newDynastyController(repo)

	rc := newRC()
	rc.Request.SetMethod("POST")
	rc.Request.SetRequestURI("/api/v1/history/dynasties")
	rc.Request.SetBody([]byte(`{"name":"Sui","start_year":581,"end_year":618,"sort_order":3,"description":"Short-lived"}`))

	ctrl.Create(ctx(), rc)
	body := assertStatusCode(t, rc, http.StatusCreated)
	data, ok := body.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "Sui", data["name"])
	assert.EqualValues(t, 581, data["start_year"])
}

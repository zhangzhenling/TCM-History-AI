package controller_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"tcm-history-ai/backend/history-service/internal/application/usecase"
	"tcm-history-ai/backend/history-service/internal/controller"
	"tcm-history-ai/backend/pkg/errno"
)

// fakeSearchClient is a stub usecase.SearchClient for the controller tests.
type fakeSearchClient struct {
	byIndex map[string][]map[string]any
	err     error
	calls   int
}

func (f *fakeSearchClient) Search(_ context.Context, index, _ string, _ int) ([]map[string]any, int64, error) {
	f.calls++
	if f.err != nil {
		return nil, 0, f.err
	}
	docs := f.byIndex[index]
	return docs, int64(len(docs)), nil
}

func newSearchController(client *fakeSearchClient) *controller.SearchController {
	uc := usecase.NewSearchUseCase(client)
	return controller.NewSearchController(uc)
}

// TestSearchController_Search covers the happy path, empty query, and broker
// error scenarios for the unified search endpoint.
func TestSearchController_Search(t *testing.T) {
	t.Run("happy path returns 200 with aggregated hits", func(t *testing.T) {
		client := &fakeSearchClient{byIndex: map[string][]map[string]any{
			"person": {{"id": float64(1), "name": "Zhang Zhongjing"}},
			"book":   {{"id": float64(10), "title": "Shanghan Lun"}},
		}}
		ctrl := newSearchController(client)

		rc := newRC()
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/history/search?q=zhang&page=1&page_size=10")

		ctrl.Search(ctx(), rc)
		body := assertStatusCode(t, rc, http.StatusOK)
		assert.Equal(t, int(errno.OK), body.Code)
	})

	t.Run("empty query returns 400", func(t *testing.T) {
		client := &fakeSearchClient{}
		ctrl := newSearchController(client)

		rc := newRC()
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/history/search?q=")

		ctrl.Search(ctx(), rc)
		assertStatusCode(t, rc, http.StatusBadRequest)
	})

	t.Run("missing q param returns 400", func(t *testing.T) {
		client := &fakeSearchClient{}
		ctrl := newSearchController(client)

		rc := newRC()
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/history/search")

		ctrl.Search(ctx(), rc)
		assertStatusCode(t, rc, http.StatusBadRequest)
	})

	t.Run("broker error returns 500", func(t *testing.T) {
		client := &fakeSearchClient{err: assertError("meili down")}
		ctrl := newSearchController(client)

		rc := newRC()
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/history/search?q=zhang")

		ctrl.Search(ctx(), rc)
		assert.Equal(t, http.StatusInternalServerError, rc.Response.StatusCode())
	})

	t.Run("with types filter returns 200", func(t *testing.T) {
		client := &fakeSearchClient{byIndex: map[string][]map[string]any{
			"person": {{"id": float64(1)}},
		}}
		ctrl := newSearchController(client)

		rc := newRC()
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/history/search?q=zhang&types=person")

		ctrl.Search(ctx(), rc)
		assertStatusCode(t, rc, http.StatusOK)
		assert.Equal(t, 1, client.calls, "types filter should restrict fanout to one index")
	})

	t.Run("invalid types returns 400", func(t *testing.T) {
		client := &fakeSearchClient{}
		ctrl := newSearchController(client)

		rc := newRC()
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/history/search?q=zhang&types=bogus")

		ctrl.Search(ctx(), rc)
		assertStatusCode(t, rc, http.StatusBadRequest)
	})

	t.Run("non-numeric page yields zero page", func(t *testing.T) {
		client := &fakeSearchClient{byIndex: map[string][]map[string]any{}}
		ctrl := newSearchController(client)

		rc := newRC()
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/history/search?q=zhang&page=abc&page_size=xyz")

		ctrl.Search(ctx(), rc)
		assertStatusCode(t, rc, http.StatusOK)
	})
}

// TestNewSearchController_NotNil verifies the constructor returns a non-nil
// controller.
func TestNewSearchController_NotNil(t *testing.T) {
	uc := usecase.NewSearchUseCase(&fakeSearchClient{})
	ctrl := controller.NewSearchController(uc)
	assert.NotNil(t, ctrl)
}

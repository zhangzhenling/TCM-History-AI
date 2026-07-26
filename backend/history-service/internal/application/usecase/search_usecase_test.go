package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tcm-history-ai/backend/history-service/internal/application/dto"
	"tcm-history-ai/backend/history-service/internal/application/usecase"
	"tcm-history-ai/backend/pkg/errno"
)

// fakeSearchClient is a stub usecase.SearchClient used by the unit tests.
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

// TestSearchUseCase_HappyPath verifies the usecase fans out across multiple
// indices and aggregates hits.
func TestSearchUseCase_HappyPath(t *testing.T) {
	client := &fakeSearchClient{byIndex: map[string][]map[string]any{
		"person": {
			{"id": float64(1), "name": "Zhang Zhongjing"},
			{"id": float64(2), "name": "Hua Tuo"},
		},
		"book": {
			{"id": float64(10), "title": "Shanghan Lun"},
		},
	}}
	uc := usecase.NewSearchUseCase(client)

	resp, err := uc.Search(context.Background(), &dto.SearchRequest{
		Q:        "zhang",
		Page:     1,
		PageSize: 10,
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	// Two indices have hits, total should be 3.
	assert.Equal(t, int64(3), resp.Total)
	require.Len(t, resp.Items, 3)

	types := map[string]int{}
	for _, hit := range resp.Items {
		types[hit.Type]++
		if hit.Type == "person" {
			assert.NotZero(t, hit.ID)
		}
	}
	assert.Equal(t, 2, types["person"])
	assert.Equal(t, 1, types["book"])
}

// TestSearchUseCase_TypeFilter verifies the types= query restricts the fanout.
func TestSearchUseCase_TypeFilter(t *testing.T) {
	client := &fakeSearchClient{byIndex: map[string][]map[string]any{
		"person": {{"id": float64(1)}},
		"book":   {{"id": float64(2)}},
	}}
	uc := usecase.NewSearchUseCase(client)

	resp, err := uc.Search(context.Background(), &dto.SearchRequest{
		Q:     "x",
		Types: "person",
	})
	require.NoError(t, err)
	require.Len(t, resp.Items, 1)
	assert.Equal(t, "person", resp.Items[0].Type)
	// Only one index should have been queried.
	assert.Equal(t, 1, client.calls)
}

// TestSearchUseCase_InvalidType verifies unsupported types are dropped.
func TestSearchUseCase_InvalidType(t *testing.T) {
	client := &fakeSearchClient{}
	uc := usecase.NewSearchUseCase(client)

	resp, err := uc.Search(context.Background(), &dto.SearchRequest{
		Q:     "x",
		Types: "bogus",
	})
	require.Error(t, err)
	assert.Nil(t, resp)
	var e *errno.Error
	if errors.As(err, &e) {
		assert.Equal(t, errno.InvalidParams, e.Code)
	}
}

// TestSearchUseCase_EmptyQuery rejects empty queries.
func TestSearchUseCase_EmptyQuery(t *testing.T) {
	client := &fakeSearchClient{}
	uc := usecase.NewSearchUseCase(client)

	resp, err := uc.Search(context.Background(), &dto.SearchRequest{Q: ""})
	require.Error(t, err)
	assert.Nil(t, resp)
}

// TestSearchUseCase_BrokerError propagates errors from the search client.
func TestSearchUseCase_BrokerError(t *testing.T) {
	client := &fakeSearchClient{err: errors.New("meili down")}
	uc := usecase.NewSearchUseCase(client)

	resp, err := uc.Search(context.Background(), &dto.SearchRequest{Q: "x"})
	require.Error(t, err)
	assert.Nil(t, resp)
}

// TestSearchUseCase_AllTypesByDefault verifies that omitting types= queries
// every supported index.
func TestSearchUseCase_AllTypesByDefault(t *testing.T) {
	client := &fakeSearchClient{byIndex: map[string][]map[string]any{}}
	uc := usecase.NewSearchUseCase(client)

	_, err := uc.Search(context.Background(), &dto.SearchRequest{Q: "x"})
	require.NoError(t, err)
	// supportedSearchTypes has 8 entries; each should result in one call.
	assert.Equal(t, 8, client.calls)
}

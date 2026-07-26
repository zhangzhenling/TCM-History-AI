package usecase_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tcm-history-ai/backend/ai-service/internal/application/dto"
	"tcm-history-ai/backend/ai-service/internal/application/usecase"
	"tcm-history-ai/backend/ai-service/internal/domain/entity"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/idgen"
	"tcm-history-ai/backend/pkg/pagination"
)

// newToolUseCase wires up a ToolUseCase with a fresh mock repo and executor.
func newToolUseCase() (*usecase.ToolUseCase, *mockToolRepo, *mockToolExecutor) {
	repo := newMockToolRepo()
	exec := &mockToolExecutor{}
	return usecase.NewToolUseCase(repo, exec), repo, exec
}

// TestToolUseCase_Create_HappyPath verifies a tool is created with defaults.
func TestToolUseCase_Create_HappyPath(t *testing.T) {
	uc, repo, _ := newToolUseCase()

	resp, err := uc.Create(context.Background(), &dto.ToolRequest{
		Name:        "timeline",
		Description: "TCM timeline",
		IsEnabled:   true,
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotZero(t, resp.ID)
	assert.Equal(t, "timeline", resp.Name)
	assert.Equal(t, entity.ToolMethodGET, resp.Method)            // default GET
	assert.Equal(t, "v1", resp.Version)                          // default v1
	assert.Equal(t, json.RawMessage("{}"), resp.ParametersJSON) // default {}

	got, err := repo.FindByID(context.Background(), resp.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "timeline", got.Name)
}

// TestToolUseCase_Create_WithExplicitFields verifies explicit fields are honoured.
func TestToolUseCase_Create_WithExplicitFields(t *testing.T) {
	uc, _, _ := newToolUseCase()
	resp, err := uc.Create(context.Background(), &dto.ToolRequest{
		Name:           "search",
		Method:         entity.ToolMethodPOST,
		ParametersJSON: []byte(`{"q":"string"}`),
		Version:        "v2",
		IsEnabled:      true,
	})
	require.NoError(t, err)
	assert.Equal(t, entity.ToolMethodPOST, resp.Method)
	assert.Equal(t, "v2", resp.Version)
	assert.Equal(t, json.RawMessage(`{"q":"string"}`), resp.ParametersJSON)
}

// TestToolUseCase_Create_ValidationErrors covers input validations.
func TestToolUseCase_Create_ValidationErrors(t *testing.T) {
	uc, _, _ := newToolUseCase()
	cases := []struct {
		name string
		in   *dto.ToolRequest
		code errno.Errno
	}{
		{"nil request", nil, errno.InvalidParams},
		{"empty name", &dto.ToolRequest{}, errno.InvalidParams},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := uc.Create(context.Background(), c.in)
			require.Error(t, err)
			var e *errno.Error
			if errors.As(err, &e) {
				assert.Equal(t, c.code, e.Code)
			}
		})
	}
}

// TestToolUseCase_Create_Duplicate verifies the FindByName dedup path.
func TestToolUseCase_Create_Duplicate(t *testing.T) {
	uc, _, _ := newToolUseCase()
	_, err := uc.Create(context.Background(), &dto.ToolRequest{Name: "dup"})
	require.NoError(t, err)
	_, err = uc.Create(context.Background(), &dto.ToolRequest{Name: "dup"})
	require.Error(t, err)
	var e *errno.Error
	if errors.As(err, &e) {
		assert.Equal(t, errno.AlreadyExists, e.Code)
	}
}

// TestToolUseCase_Create_FindByNameError verifies repo errors during dedup propagate.
func TestToolUseCase_Create_FindByNameError(t *testing.T) {
	uc, repo, _ := newToolUseCase()
	repo.findByName = func(string) (*entity.Tool, error) {
		return nil, errors.New("db down")
	}
	_, err := uc.Create(context.Background(), &dto.ToolRequest{Name: "n"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "db down")
}

// TestToolUseCase_Create_RepoCreateError verifies Create errors propagate.
func TestToolUseCase_Create_RepoCreateError(t *testing.T) {
	uc, repo, _ := newToolUseCase()
	repo.create = func(*entity.Tool) error { return errors.New("write failed") }
	_, err := uc.Create(context.Background(), &dto.ToolRequest{Name: "n"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "write failed")
}

// TestToolUseCase_Update_HappyPath verifies updates overwrite fields.
func TestToolUseCase_Update_HappyPath(t *testing.T) {
	uc, _, _ := newToolUseCase()
	created, err := uc.Create(context.Background(), &dto.ToolRequest{Name: "n"})
	require.NoError(t, err)

	resp, err := uc.Update(context.Background(), created.ID, &dto.ToolRequest{
		Name:           "n2",
		Description:    "desc",
		Endpoint:       "http://example.com",
		Method:         entity.ToolMethodPOST,
		ParametersJSON: []byte(`{"a":1}`),
		Category:       "cat",
		IsEnabled:      true,
		Version:        "v3",
	})
	require.NoError(t, err)
	assert.Equal(t, "n2", resp.Name)
	assert.Equal(t, "desc", resp.Description)
	assert.Equal(t, "http://example.com", resp.Endpoint)
	assert.Equal(t, entity.ToolMethodPOST, resp.Method)
	assert.Equal(t, json.RawMessage(`{"a":1}`), resp.ParametersJSON)
	assert.Equal(t, "cat", resp.Category)
	assert.True(t, resp.IsEnabled)
	assert.Equal(t, "v3", resp.Version)
}

// TestToolUseCase_Update_NilRequest verifies nil body is rejected.
func TestToolUseCase_Update_NilRequest(t *testing.T) {
	uc, _, _ := newToolUseCase()
	_, err := uc.Update(context.Background(), 1, nil)
	require.Error(t, err)
	var e *errno.Error
	if errors.As(err, &e) {
		assert.Equal(t, errno.InvalidParams, e.Code)
	}
}

// TestToolUseCase_Update_NotFound verifies update on a missing row.
func TestToolUseCase_Update_NotFound(t *testing.T) {
	uc, _, _ := newToolUseCase()
	_, err := uc.Update(context.Background(), 9999, &dto.ToolRequest{Name: "n"})
	require.Error(t, err)
	var e *errno.Error
	if errors.As(err, &e) {
		assert.Equal(t, errno.NotFound, e.Code)
	}
}

// TestToolUseCase_Update_FindError verifies repo errors propagate.
func TestToolUseCase_Update_FindError(t *testing.T) {
	uc, repo, _ := newToolUseCase()
	repo.find = func(int64) (*entity.Tool, error) { return nil, errors.New("find err") }
	_, err := uc.Update(context.Background(), 1, &dto.ToolRequest{Name: "n"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "find err")
}

// TestToolUseCase_Update_RepoUpdateError verifies Update errors propagate.
func TestToolUseCase_Update_RepoUpdateError(t *testing.T) {
	uc, repo, _ := newToolUseCase()
	created, err := uc.Create(context.Background(), &dto.ToolRequest{Name: "n"})
	require.NoError(t, err)
	repo.update = func(*entity.Tool) error { return errors.New("update failed") }
	_, err = uc.Update(context.Background(), created.ID, &dto.ToolRequest{Name: "n2"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "update failed")
}

// TestToolUseCase_Delete covers the delete path.
func TestToolUseCase_Delete(t *testing.T) {
	uc, _, _ := newToolUseCase()
	created, err := uc.Create(context.Background(), &dto.ToolRequest{Name: "n"})
	require.NoError(t, err)

	// Delete twice; second should be silently OK (Delete on missing row).
	require.NoError(t, uc.Delete(context.Background(), created.ID))
	require.NoError(t, uc.Delete(context.Background(), created.ID))
}

// TestToolUseCase_Delete_Error verifies a forced delete error propagates.
func TestToolUseCase_Delete_Error(t *testing.T) {
	uc, repo, _ := newToolUseCase()
	repo.delete = func(int64) error { return errors.New("delete failed") }
	err := uc.Delete(context.Background(), 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "delete failed")
}

// TestToolUseCase_Get covers found / not-found / error paths.
func TestToolUseCase_Get(t *testing.T) {
	uc, repo, _ := newToolUseCase()
	created, err := uc.Create(context.Background(), &dto.ToolRequest{Name: "n"})
	require.NoError(t, err)

	t.Run("found", func(t *testing.T) {
		got, err := uc.Get(context.Background(), created.ID)
		require.NoError(t, err)
		assert.Equal(t, created.ID, got.ID)
	})
	t.Run("not found", func(t *testing.T) {
		_, err := uc.Get(context.Background(), 9999)
		require.Error(t, err)
		var e *errno.Error
		if errors.As(err, &e) {
			assert.Equal(t, errno.NotFound, e.Code)
		}
	})
	t.Run("find error", func(t *testing.T) {
		repo.find = func(int64) (*entity.Tool, error) {
			return nil, errors.New("find err")
		}
		_, err := uc.Get(context.Background(), 1)
		require.Error(t, err)
	})
}

// TestToolUseCase_List covers onlyEnabled true/false and error paths.
func TestToolUseCase_List(t *testing.T) {
	uc, repo, _ := newToolUseCase()
	// Seed two tools: one enabled, one disabled.
	a, err := uc.Create(context.Background(), &dto.ToolRequest{Name: "a", IsEnabled: true})
	require.NoError(t, err)
	b, err := uc.Create(context.Background(), &dto.ToolRequest{Name: "b", IsEnabled: false})
	require.NoError(t, err)
	_ = a
	_ = b

	t.Run("only enabled", func(t *testing.T) {
		resp, err := uc.List(context.Background(), true, pagination.Params{Page: 1, PageSize: 10})
		require.NoError(t, err)
		assert.Equal(t, 1, resp.Total)
		require.Len(t, resp.Items, 1)
		assert.Equal(t, "a", resp.Items[0].Name)
	})
	t.Run("all", func(t *testing.T) {
		resp, err := uc.List(context.Background(), false, pagination.Params{Page: 1, PageSize: 10})
		require.NoError(t, err)
		assert.Equal(t, 2, resp.Total)
	})
	t.Run("listEnabled error", func(t *testing.T) {
		repo.listEnabled = func(pagination.Params) ([]entity.Tool, int, error) {
			return nil, 0, errors.New("le err")
		}
		_, err := uc.List(context.Background(), true, pagination.Params{Page: 1, PageSize: 10})
		require.Error(t, err)
	})
	t.Run("list error", func(t *testing.T) {
		repo.list = func(pagination.Params) ([]entity.Tool, int, error) {
			return nil, 0, errors.New("l err")
		}
		_, err := uc.List(context.Background(), false, pagination.Params{Page: 1, PageSize: 10})
		require.Error(t, err)
	})
}

// TestToolUseCase_Execute covers happy / not-found / disabled / nil-exec / error paths.
func TestToolUseCase_Execute(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		uc, _, exec := newToolUseCase()
		created, err := uc.Create(context.Background(), &dto.ToolRequest{Name: "ask", IsEnabled: true})
		require.NoError(t, err)
		exec.result = map[string]any{"answer": "ok"}

		resp, err := uc.Execute(context.Background(), created.ID, map[string]any{"q": "x"})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "ask", resp.ToolName)
		assert.Equal(t, "ok", resp.Result["answer"])
		assert.Equal(t, 1, exec.calls)
	})

	t.Run("not found", func(t *testing.T) {
		uc, _, _ := newToolUseCase()
		_, err := uc.Execute(context.Background(), 9999, nil)
		require.Error(t, err)
		var e *errno.Error
		if errors.As(err, &e) {
			assert.Equal(t, errno.NotFound, e.Code)
		}
	})

	t.Run("disabled tool", func(t *testing.T) {
		uc, _, _ := newToolUseCase()
		created, err := uc.Create(context.Background(), &dto.ToolRequest{Name: "off", IsEnabled: false})
		require.NoError(t, err)
		_, err = uc.Execute(context.Background(), created.ID, nil)
		require.Error(t, err)
		var e *errno.Error
		if errors.As(err, &e) {
			assert.Equal(t, errno.Forbidden, e.Code)
		}
	})

	t.Run("find error", func(t *testing.T) {
		uc, repo, _ := newToolUseCase()
		repo.find = func(int64) (*entity.Tool, error) {
			return nil, errors.New("find err")
		}
		_, err := uc.Execute(context.Background(), 1, nil)
		require.Error(t, err)
	})

	t.Run("executor not configured", func(t *testing.T) {
		repo := newMockToolRepo()
		// Construct with a nil executor.
		uc := usecase.NewToolUseCase(repo, nil)
		tt := &entity.Tool{Name: "n", IsEnabled: true}
		tt.ID = idgen.Next()
		require.NoError(t, repo.Create(context.Background(), tt))
		_, err := uc.Execute(context.Background(), tt.ID, nil)
		require.Error(t, err)
		var e *errno.Error
		if errors.As(err, &e) {
			assert.Equal(t, errno.DependencyUnavailable, e.Code)
		}
	})

	t.Run("executor error", func(t *testing.T) {
		uc, _, exec := newToolUseCase()
		created, err := uc.Create(context.Background(), &dto.ToolRequest{Name: "n", IsEnabled: true})
		require.NoError(t, err)
		exec.err = errors.New("exec failed")
		_, err = uc.Execute(context.Background(), created.ID, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "exec failed")
	})
}

// TestToToolResponse_Timestamps verifies timestamps are formatted in the DTO.
func TestToToolResponse_Timestamps(t *testing.T) {
	uc, repo, _ := newToolUseCase()
	created, err := uc.Create(context.Background(), &dto.ToolRequest{Name: "n"})
	require.NoError(t, err)
	repo.mu.Lock()
	if tt, ok := repo.items[created.ID]; ok {
		tt.CreatedAt = time.Now()
		tt.UpdatedAt = time.Now()
	}
	repo.mu.Unlock()

	got, err := uc.Get(context.Background(), created.ID)
	require.NoError(t, err)
	assert.NotEmpty(t, got.CreatedAt)
	assert.NotEmpty(t, got.UpdatedAt)
}

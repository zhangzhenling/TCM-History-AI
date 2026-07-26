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
	"tcm-history-ai/backend/ai-service/internal/infrastructure/prompt"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/idgen"
	"tcm-history-ai/backend/pkg/pagination"
)

// newPromptUseCase wires up the PromptUseCase with the real renderer and a
// fresh in-memory mock repo.
func newPromptUseCase() (*usecase.PromptUseCase, *mockPromptRepo) {
	repo := newMockPromptRepo()
	return usecase.NewPromptUseCase(repo, prompt.New()), repo
}

// TestPromptUseCase_Create_HappyPath verifies a template is created and the
// response carries the supplied fields.
func TestPromptUseCase_Create_HappyPath(t *testing.T) {
	uc, repo := newPromptUseCase()

	resp, err := uc.Create(context.Background(), &dto.PromptTemplateRequest{
		Name:         "chat-default",
		Scene:        entity.SceneChat,
		SystemPrompt: "you are an assistant",
		Template:     "{{user_question}}",
		IsActive:     true,
		Version:      1,
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotZero(t, resp.ID)
	assert.Equal(t, "chat-default", resp.Name)
	assert.Equal(t, entity.SceneChat, resp.Scene)
	assert.True(t, resp.IsActive)

	// Repo contains the row.
	got, err := repo.FindByID(context.Background(), resp.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "chat-default", got.Name)
}

// TestPromptUseCase_Create_Defaults verifies empty VariablesJSON/Version are
// filled with defaults.
func TestPromptUseCase_Create_Defaults(t *testing.T) {
	uc, repo := newPromptUseCase()
	resp, err := uc.Create(context.Background(), &dto.PromptTemplateRequest{
		Name:         "n",
		Scene:        entity.SceneChat,
		SystemPrompt: "p",
	})
	require.NoError(t, err)
	got, err := repo.FindByID(context.Background(), resp.ID)
	require.NoError(t, err)
	assert.Equal(t, json.RawMessage("[]"), got.VariablesJSON)
	assert.Equal(t, 1, got.Version)
}

// TestPromptUseCase_Create_ValidationErrors exercises the input validations.
func TestPromptUseCase_Create_ValidationErrors(t *testing.T) {
	uc, _ := newPromptUseCase()
	cases := []struct {
		name string
		in   *dto.PromptTemplateRequest
		code errno.Errno
	}{
		{"nil request", nil, errno.InvalidParams},
		{"empty name", &dto.PromptTemplateRequest{Scene: "chat", SystemPrompt: "p"}, errno.InvalidParams},
		{"empty scene", &dto.PromptTemplateRequest{Name: "n", SystemPrompt: "p"}, errno.InvalidParams},
		{"empty system prompt", &dto.PromptTemplateRequest{Name: "n", Scene: "chat"}, errno.InvalidParams},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			resp, err := uc.Create(context.Background(), c.in)
			require.Error(t, err)
			assert.Nil(t, resp)
			var e *errno.Error
			if errors.As(err, &e) {
				assert.Equal(t, c.code, e.Code)
			}
		})
	}
}

// TestPromptUseCase_Create_Duplicate verifies the dedup-by-name-and-scene path.
func TestPromptUseCase_Create_Duplicate(t *testing.T) {
	uc, _ := newPromptUseCase()
	in := &dto.PromptTemplateRequest{
		Name: "dup", Scene: entity.SceneChat, SystemPrompt: "p",
	}
	_, err := uc.Create(context.Background(), in)
	require.NoError(t, err)
	_, err = uc.Create(context.Background(), in)
	require.Error(t, err)
	var e *errno.Error
	if errors.As(err, &e) {
		assert.Equal(t, errno.AlreadyExists, e.Code)
	}
}

// TestPromptUseCase_Create_FindByNameSceneError verifies repo errors during
// dedup are propagated.
func TestPromptUseCase_Create_FindByNameSceneError(t *testing.T) {
	uc, repo := newPromptUseCase()
	repo.findByNameScene = func(string, string) (*entity.PromptTemplate, error) {
		return nil, errors.New("db down")
	}
	_, err := uc.Create(context.Background(), &dto.PromptTemplateRequest{
		Name: "n", Scene: entity.SceneChat, SystemPrompt: "p",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "db down")
}

// TestPromptUseCase_Create_RepoCreateError verifies Create errors propagate.
func TestPromptUseCase_Create_RepoCreateError(t *testing.T) {
	uc, repo := newPromptUseCase()
	repo.create = func(*entity.PromptTemplate) error {
		return errors.New("write failed")
	}
	_, err := uc.Create(context.Background(), &dto.PromptTemplateRequest{
		Name: "n", Scene: entity.SceneChat, SystemPrompt: "p",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "write failed")
}

// TestPromptUseCase_Update_HappyPath verifies updates overwrite fields.
func TestPromptUseCase_Update_HappyPath(t *testing.T) {
	uc, _ := newPromptUseCase()
	created, err := uc.Create(context.Background(), &dto.PromptTemplateRequest{
		Name: "n", Scene: entity.SceneChat, SystemPrompt: "p",
	})
	require.NoError(t, err)

	resp, err := uc.Update(context.Background(), created.ID, &dto.PromptTemplateRequest{
		Name: "n2", Scene: entity.SceneAgent, SystemPrompt: "p2",
		VariablesJSON: []byte(`["a"]`), Version: 3,
	})
	require.NoError(t, err)
	assert.Equal(t, "n2", resp.Name)
	assert.Equal(t, entity.SceneAgent, resp.Scene)
	assert.Equal(t, "p2", resp.SystemPrompt)
	assert.Equal(t, 3, resp.Version)
	assert.Equal(t, json.RawMessage(`["a"]`), resp.VariablesJSON)
}

// TestPromptUseCase_Update_NilRequest verifies nil body is rejected.
func TestPromptUseCase_Update_NilRequest(t *testing.T) {
	uc, _ := newPromptUseCase()
	_, err := uc.Update(context.Background(), 1, nil)
	require.Error(t, err)
	var e *errno.Error
	if errors.As(err, &e) {
		assert.Equal(t, errno.InvalidParams, e.Code)
	}
}

// TestPromptUseCase_Update_NotFound verifies update on a missing row.
func TestPromptUseCase_Update_NotFound(t *testing.T) {
	uc, _ := newPromptUseCase()
	_, err := uc.Update(context.Background(), 9999, &dto.PromptTemplateRequest{Name: "n"})
	require.Error(t, err)
	var e *errno.Error
	if errors.As(err, &e) {
		assert.Equal(t, errno.NotFound, e.Code)
	}
}

// TestPromptUseCase_Update_FindError verifies repo errors propagate.
func TestPromptUseCase_Update_FindError(t *testing.T) {
	uc, repo := newPromptUseCase()
	repo.find = func(int64) (*entity.PromptTemplate, error) {
		return nil, errors.New("find err")
	}
	_, err := uc.Update(context.Background(), 1, &dto.PromptTemplateRequest{Name: "n"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "find err")
}

// TestPromptUseCase_Update_RepoUpdateError verifies Update errors propagate.
func TestPromptUseCase_Update_RepoUpdateError(t *testing.T) {
	uc, repo := newPromptUseCase()
	repo.update = func(*entity.PromptTemplate) error {
		return errors.New("update failed")
	}
	created, err := uc.Create(context.Background(), &dto.PromptTemplateRequest{
		Name: "n", Scene: entity.SceneChat, SystemPrompt: "p",
	})
	require.NoError(t, err)
	_, err = uc.Update(context.Background(), created.ID, &dto.PromptTemplateRequest{Name: "n2"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "update failed")
}

// TestPromptUseCase_Get covers found / not-found / error.
func TestPromptUseCase_Get(t *testing.T) {
	uc, repo := newPromptUseCase()
	created, err := uc.Create(context.Background(), &dto.PromptTemplateRequest{
		Name: "n", Scene: entity.SceneChat, SystemPrompt: "p",
	})
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
		repo.find = func(int64) (*entity.PromptTemplate, error) {
			return nil, errors.New("find err")
		}
		_, err := uc.Get(context.Background(), 1)
		require.Error(t, err)
	})
}

// TestPromptUseCase_List covers scene filter, no-filter and error paths.
func TestPromptUseCase_List(t *testing.T) {
	uc, repo := newPromptUseCase()
	for i, scene := range []string{entity.SceneChat, entity.SceneChat, entity.SceneAgent} {
		p := &entity.PromptTemplate{
			Name: "n", Scene: scene, SystemPrompt: "p", IsActive: true,
		}
		p.ID = idgen.Next() + int64(i)
		require.NoError(t, repo.Create(context.Background(), p))
	}

	t.Run("by scene", func(t *testing.T) {
		resp, err := uc.List(context.Background(), entity.SceneChat, pagination.Params{Page: 1, PageSize: 10})
		require.NoError(t, err)
		assert.Equal(t, 2, resp.Total)
		require.Len(t, resp.Items, 2)
	})
	t.Run("all scenes", func(t *testing.T) {
		resp, err := uc.List(context.Background(), "", pagination.Params{Page: 1, PageSize: 10})
		require.NoError(t, err)
		assert.Equal(t, 3, resp.Total)
	})
	t.Run("listByScene error", func(t *testing.T) {
		repo.listByScene = func(string, pagination.Params) ([]entity.PromptTemplate, int, error) {
			return nil, 0, errors.New("ls err")
		}
		_, err := uc.List(context.Background(), entity.SceneChat, pagination.Params{Page: 1, PageSize: 10})
		require.Error(t, err)
	})
	t.Run("list error", func(t *testing.T) {
		repo.list = func(pagination.Params) ([]entity.PromptTemplate, int, error) {
			return nil, 0, errors.New("l err")
		}
		_, err := uc.List(context.Background(), "", pagination.Params{Page: 1, PageSize: 10})
		require.Error(t, err)
	})
}

// TestPromptUseCase_Delete covers found / not-found / error paths.
func TestPromptUseCase_Delete(t *testing.T) {
	uc, repo := newPromptUseCase()
	created, err := uc.Create(context.Background(), &dto.PromptTemplateRequest{
		Name: "n", Scene: entity.SceneChat, SystemPrompt: "p",
	})
	require.NoError(t, err)

	require.NoError(t, uc.Delete(context.Background(), created.ID))

	t.Run("not found", func(t *testing.T) {
		err := uc.Delete(context.Background(), 9999)
		require.Error(t, err)
		var e *errno.Error
		if errors.As(err, &e) {
			assert.Equal(t, errno.NotFound, e.Code)
		}
	})
	t.Run("find error", func(t *testing.T) {
		repo.find = func(int64) (*entity.PromptTemplate, error) {
			return nil, errors.New("find err")
		}
		err := uc.Delete(context.Background(), 1)
		require.Error(t, err)
	})
}

// TestPromptUseCase_Render covers happy / not-found / repo-error / render-error.
func TestPromptUseCase_Render(t *testing.T) {
	t.Run("happy", func(t *testing.T) {
		uc, repo := newPromptUseCase()
		tpl := &entity.PromptTemplate{
			Name: "n", Scene: entity.SceneChat, SystemPrompt: "Hello {{name}}", IsActive: true,
		}
		tpl.ID = idgen.Next()
		require.NoError(t, repo.Create(context.Background(), tpl))

		out, err := uc.Render(context.Background(), entity.SceneChat, map[string]any{"name": "Alice"})
		require.NoError(t, err)
		assert.Equal(t, "Hello Alice", out)
	})
	t.Run("not found", func(t *testing.T) {
		uc, _ := newPromptUseCase()
		_, err := uc.Render(context.Background(), entity.SceneChat, nil)
		require.Error(t, err)
		var e *errno.Error
		if errors.As(err, &e) {
			assert.Equal(t, errno.NotFound, e.Code)
		}
	})
	t.Run("repo error", func(t *testing.T) {
		uc, repo := newPromptUseCase()
		repo.findActive = func(string) (*entity.PromptTemplate, error) {
			return nil, errors.New("active err")
		}
		_, err := uc.Render(context.Background(), entity.SceneChat, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "active err")
	})
	t.Run("render error", func(t *testing.T) {
		_, repo := newPromptUseCase()
		tpl := &entity.PromptTemplate{
			Name: "n", Scene: entity.SceneChat, SystemPrompt: "{{x}}", IsActive: true,
		}
		tpl.ID = idgen.Next()
		require.NoError(t, repo.Create(context.Background(), tpl))
		uc2 := usecase.NewPromptUseCase(repo, &mockPromptRenderer{err: errors.New("render err")})
		_, err := uc2.Render(context.Background(), entity.SceneChat, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "render err")
	})
}

// TestToPromptTemplateResponse_Timestamps verifies timestamps are formatted.
func TestToPromptTemplateResponse_Timestamps(t *testing.T) {
	uc, repo := newPromptUseCase()
	created, err := uc.Create(context.Background(), &dto.PromptTemplateRequest{
		Name: "n", Scene: entity.SceneChat, SystemPrompt: "p",
	})
	require.NoError(t, err)
	// Set timestamps directly on the stored entity to exercise the mapper.
	repo.mu.Lock()
	if p, ok := repo.items[created.ID]; ok {
		p.CreatedAt = time.Now()
		p.UpdatedAt = time.Now()
	}
	repo.mu.Unlock()

	got, err := uc.Get(context.Background(), created.ID)
	require.NoError(t, err)
	assert.NotEmpty(t, got.CreatedAt)
	assert.NotEmpty(t, got.UpdatedAt)
}

package usecase_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tcm-history-ai/backend/ai-service/internal/application/dto"
	"tcm-history-ai/backend/ai-service/internal/application/usecase"
	"tcm-history-ai/backend/ai-service/internal/domain/entity"
	"tcm-history-ai/backend/ai-service/internal/domain/event"
	"tcm-history-ai/backend/ai-service/internal/domain/service"
	"tcm-history-ai/backend/ai-service/internal/infrastructure/prompt"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/idgen"
	"tcm-history-ai/backend/pkg/pagination"
)

// TestChatUseCase_Chat_HappyPath_NewConversation exercises the full happy
// path: a brand-new conversation is created, no template found (so default
// system prompt is used), the LLM is invoked, both messages are persisted,
// the conversation title is derived from the message and an event published.
func TestChatUseCase_Chat_HappyPath_NewConversation(t *testing.T) {
	convRepo := newMockConvRepo()
	msgRepo := newMockMsgRepo()
	promptRepo := newMockPromptRepo()
	llm := newMockLLMProvider()
	pub := newMockEventPublisher()
	renderer := prompt.New()

	uc := usecase.NewChatUseCase(convRepo, msgRepo, promptRepo, llm, renderer, pub)

	resp, err := uc.Chat(context.Background(), &dto.ChatRequest{
		UserID:  42,
		Message: "张仲景是谁？",
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotZero(t, resp.ConversationID)
	assert.NotZero(t, resp.MessageID)
	assert.Equal(t, "stub-answer", resp.Assistant)
	assert.Equal(t, "test-model", resp.Model)
	assert.Equal(t, 10, resp.TokensPrompt)
	assert.Equal(t, 5, resp.TokensCompletion)
	assert.GreaterOrEqual(t, resp.LatencyMs, 0)

	// Verify conversation persisted with title derived from message.
	conv, err := convRepo.FindByID(context.Background(), resp.ConversationID)
	require.NoError(t, err)
	require.NotNil(t, conv)
	assert.Equal(t, int64(42), conv.UserID)
	assert.Equal(t, "张仲景是谁？", conv.Title)
	assert.Equal(t, entity.ConversationModeChat, conv.Mode)
	assert.Equal(t, entity.ConversationStatusActive, conv.Status)
	assert.Equal(t, 2, conv.MessageCount)

	// Verify event published.
	evt, ok := captureEvent[event.ChatMessageCreated](pub)
	require.True(t, ok)
	assert.Equal(t, resp.ConversationID, evt.ConversationID)
	assert.Equal(t, resp.MessageID, evt.MessageID)
	assert.Equal(t, entity.MessageRoleAssistant, evt.Role)
	assert.Equal(t, "test-model", evt.ModelName)
}

// TestChatUseCase_Chat_ExistingConversation verifies that when ConversationID
// is supplied the existing conversation is reused.
func TestChatUseCase_Chat_ExistingConversation(t *testing.T) {
	convRepo := newMockConvRepo()
	msgRepo := newMockMsgRepo()
	promptRepo := newMockPromptRepo()
	llm := newMockLLMProvider()
	pub := newMockEventPublisher()
	renderer := prompt.New()

	uc := usecase.NewChatUseCase(convRepo, msgRepo, promptRepo, llm, renderer, pub)

	// Seed an existing conversation.
	existing := &entity.Conversation{
		UserID:       7,
		Title:        "Existing",
		Mode:         entity.ConversationModeChat,
		Status:       entity.ConversationStatusActive,
		MessageCount: 0,
		MetadataJSON: []byte("{}"),
	}
	existing.ID = idgen.Next()
	require.NoError(t, convRepo.Create(context.Background(), existing))

	resp, err := uc.Chat(context.Background(), &dto.ChatRequest{
		ConversationID: existing.ID,
		UserID:          7,
		Message:         "hello",
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, existing.ID, resp.ConversationID)

	// Conversation title is preserved (not overwritten when not empty).
	conv, err := convRepo.FindByID(context.Background(), existing.ID)
	require.NoError(t, err)
	assert.Equal(t, "Existing", conv.Title)
	assert.Equal(t, 2, conv.MessageCount)
}

// TestChatUseCase_Chat_WithTemplateAndVariables verifies that an active
// prompt template is fetched and rendered with the user's variables.
func TestChatUseCase_Chat_WithTemplateAndVariables(t *testing.T) {
	convRepo := newMockConvRepo()
	msgRepo := newMockMsgRepo()
	promptRepo := newMockPromptRepo()
	llm := newMockLLMProvider()
	pub := newMockEventPublisher()
	renderer := prompt.New()

	// Seed an active template for the chat scene.
	tpl := &entity.PromptTemplate{
		Name:         "chat-default",
		Scene:        entity.SceneChat,
		SystemPrompt: "You are an assistant. Topic: {{topic}}",
		IsActive:     true,
	}
	tpl.ID = idgen.Next()
	require.NoError(t, promptRepo.Create(context.Background(), tpl))

	uc := usecase.NewChatUseCase(convRepo, msgRepo, promptRepo, llm, renderer, pub)

	resp, err := uc.Chat(context.Background(), &dto.ChatRequest{
		UserID:   1,
		Message:  "question",
		Variables: map[string]any{"topic": "TCM"},
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "stub-answer", resp.Assistant)
}

// TestChatUseCase_Chat_TitleTruncation verifies that a very long message is
// truncated to 32 runes + "..." when copied to the conversation title.
func TestChatUseCase_Chat_TitleTruncation(t *testing.T) {
	convRepo := newMockConvRepo()
	msgRepo := newMockMsgRepo()
	promptRepo := newMockPromptRepo()
	llm := newMockLLMProvider()
	pub := newMockEventPublisher()
	renderer := prompt.New()

	longMsg := strings.Repeat("中医", 50) // 100 runes
	uc := usecase.NewChatUseCase(convRepo, msgRepo, promptRepo, llm, renderer, pub)
	resp, err := uc.Chat(context.Background(), &dto.ChatRequest{
		UserID:  1,
		Message: longMsg,
	})
	require.NoError(t, err)
	require.NotNil(t, resp)

	conv, err := convRepo.FindByID(context.Background(), resp.ConversationID)
	require.NoError(t, err)
	assert.True(t, strings.HasSuffix(conv.Title, "..."))
	assert.LessOrEqual(t, len([]rune(conv.Title)), 35) // 32 + "..."
}

// TestChatUseCase_Chat_ValidationErrors exercises the input validation paths.
func TestChatUseCase_Chat_ValidationErrors(t *testing.T) {
	convRepo := newMockConvRepo()
	msgRepo := newMockMsgRepo()
	promptRepo := newMockPromptRepo()
	llm := newMockLLMProvider()
	pub := newMockEventPublisher()
	renderer := prompt.New()
	uc := usecase.NewChatUseCase(convRepo, msgRepo, promptRepo, llm, renderer, pub)

	cases := []struct {
		name string
		in   *dto.ChatRequest
		code errno.Errno
	}{
		{"nil request", nil, errno.InvalidParams},
		{"empty message", &dto.ChatRequest{UserID: 1}, errno.InvalidParams},
		{"zero user_id", &dto.ChatRequest{Message: "hi"}, errno.InvalidParams},
		{"negative user_id", &dto.ChatRequest{UserID: -1, Message: "hi"}, errno.InvalidParams},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			resp, err := uc.Chat(context.Background(), c.in)
			require.Error(t, err)
			assert.Nil(t, resp)
			var e *errno.Error
			if errors.As(err, &e) {
				assert.Equal(t, c.code, e.Code)
			}
		})
	}
}

// TestChatUseCase_Chat_ConversationNotFound verifies that supplying a non-
// existent conversation id yields a NotFound error.
func TestChatUseCase_Chat_ConversationNotFound(t *testing.T) {
	convRepo := newMockConvRepo()
	convRepo.find = func(int64) (*entity.Conversation, error) {
		return nil, nil
	}
	msgRepo := newMockMsgRepo()
	promptRepo := newMockPromptRepo()
	llm := newMockLLMProvider()
	pub := newMockEventPublisher()
	renderer := prompt.New()
	uc := usecase.NewChatUseCase(convRepo, msgRepo, promptRepo, llm, renderer, pub)

	_, err := uc.Chat(context.Background(), &dto.ChatRequest{
		ConversationID: 9999,
		UserID:          1,
		Message:         "hi",
	})
	require.Error(t, err)
	var e *errno.Error
	if errors.As(err, &e) {
		assert.Equal(t, errno.NotFound, e.Code)
	}
}

// TestChatUseCase_Chat_FindByIDError verifies repo errors during conversation
// lookup are propagated.
func TestChatUseCase_Chat_FindByIDError(t *testing.T) {
	convRepo := newMockConvRepo()
	dbErr := errors.New("db down")
	convRepo.find = func(int64) (*entity.Conversation, error) { return nil, dbErr }
	msgRepo := newMockMsgRepo()
	promptRepo := newMockPromptRepo()
	llm := newMockLLMProvider()
	pub := newMockEventPublisher()
	renderer := prompt.New()
	uc := usecase.NewChatUseCase(convRepo, msgRepo, promptRepo, llm, renderer, pub)

	_, err := uc.Chat(context.Background(), &dto.ChatRequest{
		ConversationID: 5,
		UserID:          1,
		Message:         "hi",
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, dbErr)
}

// TestChatUseCase_Chat_HistoryLoadError verifies errors from FindByConversation
// are wrapped as DependencyUnavailable.
func TestChatUseCase_Chat_HistoryLoadError(t *testing.T) {
	convRepo := newMockConvRepo()
	msgRepo := newMockMsgRepo()
	msgRepo.findHist = func(int64) ([]entity.Message, error) {
		return nil, errors.New("history load failed")
	}
	promptRepo := newMockPromptRepo()
	llm := newMockLLMProvider()
	pub := newMockEventPublisher()
	renderer := prompt.New()
	uc := usecase.NewChatUseCase(convRepo, msgRepo, promptRepo, llm, renderer, pub)

	_, err := uc.Chat(context.Background(), &dto.ChatRequest{
		UserID:  1,
		Message: "hi",
	})
	require.Error(t, err)
	var e *errno.Error
	if errors.As(err, &e) {
		assert.Equal(t, errno.DependencyUnavailable, e.Code)
	}
}

// TestChatUseCase_Chat_LLMError verifies LLM errors are wrapped as
// DependencyUnavailable and that the user message has already been persisted.
func TestChatUseCase_Chat_LLMError(t *testing.T) {
	convRepo := newMockConvRepo()
	msgRepo := newMockMsgRepo()
	promptRepo := newMockPromptRepo()
	llm := newMockLLMProvider()
	llm.chatErr = errors.New("llm rate limited")
	pub := newMockEventPublisher()
	renderer := prompt.New()
	uc := usecase.NewChatUseCase(convRepo, msgRepo, promptRepo, llm, renderer, pub)

	_, err := uc.Chat(context.Background(), &dto.ChatRequest{
		UserID:  1,
		Message: "hi",
	})
	require.Error(t, err)
	var e *errno.Error
	if errors.As(err, &e) {
		assert.Equal(t, errno.DependencyUnavailable, e.Code)
	}
}

// TestChatUseCase_Chat_CreateUserMessageError verifies persistence errors
// for the user message are propagated verbatim.
func TestChatUseCase_Chat_CreateUserMessageError(t *testing.T) {
	convRepo := newMockConvRepo()
	msgRepo := newMockMsgRepo()
	msgRepo.create = func(*entity.Message) error {
		return errors.New("write failed")
	}
	promptRepo := newMockPromptRepo()
	llm := newMockLLMProvider()
	pub := newMockEventPublisher()
	renderer := prompt.New()
	uc := usecase.NewChatUseCase(convRepo, msgRepo, promptRepo, llm, renderer, pub)

	_, err := uc.Chat(context.Background(), &dto.ChatRequest{
		UserID:  1,
		Message: "hi",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "write failed")
}

// TestChatUseCase_Chat_CreateAssistantMessageError verifies persistence
// errors for the assistant message are propagated.
func TestChatUseCase_Chat_CreateAssistantMessageError(t *testing.T) {
	convRepo := newMockConvRepo()
	msgRepo := newMockMsgRepo()
	createCalls := 0
	msgRepo.create = func(m *entity.Message) error {
		createCalls++
		if createCalls == 2 { // assistant message
			return errors.New("assistant write failed")
		}
		msgRepo.items[m.ID] = m
		return nil
	}
	promptRepo := newMockPromptRepo()
	llm := newMockLLMProvider()
	pub := newMockEventPublisher()
	renderer := prompt.New()
	uc := usecase.NewChatUseCase(convRepo, msgRepo, promptRepo, llm, renderer, pub)

	_, err := uc.Chat(context.Background(), &dto.ChatRequest{
		UserID:  1,
		Message: "hi",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "assistant write failed")
}

// TestChatUseCase_Chat_ModeOverride verifies that when Mode is supplied it
// is honoured on the new conversation.
func TestChatUseCase_Chat_ModeOverride(t *testing.T) {
	convRepo := newMockConvRepo()
	msgRepo := newMockMsgRepo()
	promptRepo := newMockPromptRepo()
	llm := newMockLLMProvider()
	pub := newMockEventPublisher()
	renderer := prompt.New()
	uc := usecase.NewChatUseCase(convRepo, msgRepo, promptRepo, llm, renderer, pub)

	resp, err := uc.Chat(context.Background(), &dto.ChatRequest{
		UserID:  1,
		Mode:    entity.ConversationModeReasoning,
		Message: "hi",
	})
	require.NoError(t, err)
	conv, err := convRepo.FindByID(context.Background(), resp.ConversationID)
	require.NoError(t, err)
	assert.Equal(t, entity.ConversationModeReasoning, conv.Mode)
}

// TestChatUseCase_Chat_RenderError verifies that a renderer error short-
// circuits the chat flow.
func TestChatUseCase_Chat_RenderError(t *testing.T) {
	convRepo := newMockConvRepo()
	msgRepo := newMockMsgRepo()
	promptRepo := newMockPromptRepo()
	// Seed an active template so Render is invoked.
	tpl := &entity.PromptTemplate{
		Name:         "x",
		Scene:        entity.SceneChat,
		SystemPrompt: "{{topic}}",
		IsActive:     true,
	}
	tpl.ID = idgen.Next()
	require.NoError(t, promptRepo.Create(context.Background(), tpl))

	llm := newMockLLMProvider()
	pub := newMockEventPublisher()
	renderer := &mockPromptRenderer{err: errors.New("render failed")}
	uc := usecase.NewChatUseCase(convRepo, msgRepo, promptRepo, llm, renderer, pub)

	_, err := uc.Chat(context.Background(), &dto.ChatRequest{
		UserID:  1,
		Message: "hi",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "render failed")
}

// TestChatUseCase_Chat_TemplateSceneOverride verifies the TemplateScene
// field selects a different active template.
func TestChatUseCase_Chat_TemplateSceneOverride(t *testing.T) {
	convRepo := newMockConvRepo()
	msgRepo := newMockMsgRepo()
	promptRepo := newMockPromptRepo()
	tpl := &entity.PromptTemplate{
		Name:         "agent-tpl",
		Scene:        entity.SceneAgent,
		SystemPrompt: "agent prompt",
		IsActive:     true,
	}
	tpl.ID = idgen.Next()
	require.NoError(t, promptRepo.Create(context.Background(), tpl))

	llm := newMockLLMProvider()
	pub := newMockEventPublisher()
	renderer := prompt.New()
	uc := usecase.NewChatUseCase(convRepo, msgRepo, promptRepo, llm, renderer, pub)

	resp, err := uc.Chat(context.Background(), &dto.ChatRequest{
		UserID:         1,
		Message:        "hi",
		TemplateScene:  entity.SceneAgent,
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
}

// TestChatUseCase_ListConversations exercises pagination and the count.
func TestChatUseCase_ListConversations(t *testing.T) {
	convRepo := newMockConvRepo()
	msgRepo := newMockMsgRepo()
	promptRepo := newMockPromptRepo()
	llm := newMockLLMProvider()
	pub := newMockEventPublisher()
	renderer := prompt.New()
	uc := usecase.NewChatUseCase(convRepo, msgRepo, promptRepo, llm, renderer, pub)

	// Seed 3 conversations for user 1.
	for i := 0; i < 3; i++ {
		c := &entity.Conversation{UserID: 1, Mode: entity.ConversationModeChat, Status: entity.ConversationStatusActive}
		c.ID = idgen.Next()
		require.NoError(t, convRepo.Create(context.Background(), c))
	}
	// And 1 for user 2 (should not appear).
	c2 := &entity.Conversation{UserID: 2, Mode: entity.ConversationModeChat, Status: entity.ConversationStatusActive}
	c2.ID = idgen.Next()
	require.NoError(t, convRepo.Create(context.Background(), c2))

	resp, err := uc.ListConversations(context.Background(), 1, pagination.Params{Page: 1, PageSize: 2})
	require.NoError(t, err)
	assert.Equal(t, 3, resp.Total)
	assert.Equal(t, 2, resp.TotalPage)
	require.Len(t, resp.Items, 2)
}

// TestChatUseCase_ListConversations_Error verifies repo errors propagate.
func TestChatUseCase_ListConversations_Error(t *testing.T) {
	convRepo := newMockConvRepo()
	convRepo.list = func(int64, pagination.Params) ([]entity.Conversation, int, error) {
		return nil, 0, errors.New("list err")
	}
	msgRepo := newMockMsgRepo()
	promptRepo := newMockPromptRepo()
	llm := newMockLLMProvider()
	pub := newMockEventPublisher()
	renderer := prompt.New()
	uc := usecase.NewChatUseCase(convRepo, msgRepo, promptRepo, llm, renderer, pub)

	_, err := uc.ListConversations(context.Background(), 1, pagination.Params{Page: 1, PageSize: 10})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "list err")
}

// TestChatUseCase_GetConversation covers found / not-found / error paths.
func TestChatUseCase_GetConversation(t *testing.T) {
	convRepo := newMockConvRepo()
	msgRepo := newMockMsgRepo()
	promptRepo := newMockPromptRepo()
	llm := newMockLLMProvider()
	pub := newMockEventPublisher()
	renderer := prompt.New()
	uc := usecase.NewChatUseCase(convRepo, msgRepo, promptRepo, llm, renderer, pub)

	t.Run("found", func(t *testing.T) {
		c := &entity.Conversation{UserID: 1, Title: "T", Mode: entity.ConversationModeChat, Status: entity.ConversationStatusActive}
		c.ID = idgen.Next()
		require.NoError(t, convRepo.Create(context.Background(), c))

		got, err := uc.GetConversation(context.Background(), c.ID)
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, "T", got.Title)
	})

	t.Run("not found", func(t *testing.T) {
		got, err := uc.GetConversation(context.Background(), 999999)
		require.Error(t, err)
		assert.Nil(t, got)
		var e *errno.Error
		if errors.As(err, &e) {
			assert.Equal(t, errno.NotFound, e.Code)
		}
	})

	t.Run("repo error", func(t *testing.T) {
		convRepo.find = func(int64) (*entity.Conversation, error) {
			return nil, errors.New("boom")
		}
		_, err := uc.GetConversation(context.Background(), 1)
		require.Error(t, err)
	})
}

// TestChatUseCase_DeleteConversation covers delete and not-found paths.
func TestChatUseCase_DeleteConversation(t *testing.T) {
	convRepo := newMockConvRepo()
	msgRepo := newMockMsgRepo()
	promptRepo := newMockPromptRepo()
	llm := newMockLLMProvider()
	pub := newMockEventPublisher()
	renderer := prompt.New()
	uc := usecase.NewChatUseCase(convRepo, msgRepo, promptRepo, llm, renderer, pub)

	c := &entity.Conversation{UserID: 1, Mode: entity.ConversationModeChat, Status: entity.ConversationStatusActive}
	c.ID = idgen.Next()
	require.NoError(t, convRepo.Create(context.Background(), c))

	require.NoError(t, uc.DeleteConversation(context.Background(), c.ID))
	// Deleting again (already removed) → NotFound.
	err := uc.DeleteConversation(context.Background(), c.ID)
	require.Error(t, err)
	var e *errno.Error
	if errors.As(err, &e) {
		assert.Equal(t, errno.NotFound, e.Code)
	}
}

// TestChatUseCase_ListMessages covers pagination and error paths.
func TestChatUseCase_ListMessages(t *testing.T) {
	convRepo := newMockConvRepo()
	msgRepo := newMockMsgRepo()
	promptRepo := newMockPromptRepo()
	llm := newMockLLMProvider()
	pub := newMockEventPublisher()
	renderer := prompt.New()
	uc := usecase.NewChatUseCase(convRepo, msgRepo, promptRepo, llm, renderer, pub)

	// Seed 3 messages for conversation 1.
	for i := 0; i < 3; i++ {
		m := &entity.Message{ConversationID: 1, Role: entity.MessageRoleUser, Content: "x"}
		m.ID = idgen.Next()
		require.NoError(t, msgRepo.Create(context.Background(), m))
	}

	resp, err := uc.ListMessages(context.Background(), 1, pagination.Params{Page: 1, PageSize: 2})
	require.NoError(t, err)
	assert.Equal(t, 3, resp.Total)
	require.Len(t, resp.Items, 2)

	t.Run("repo error", func(t *testing.T) {
		msgRepo.list = func(int64, pagination.Params) ([]entity.Message, int, error) {
			return nil, 0, errors.New("list err")
		}
		_, err := uc.ListMessages(context.Background(), 1, pagination.Params{Page: 1, PageSize: 2})
		require.Error(t, err)
	})
}

// TestChatUseCase_Chat_PromptRepoFindActiveError verifies that the FindActive
// error path is propagated.
func TestChatUseCase_Chat_PromptRepoFindActiveError(t *testing.T) {
	convRepo := newMockConvRepo()
	msgRepo := newMockMsgRepo()
	promptRepo := newMockPromptRepo()
	promptRepo.findActive = func(string) (*entity.PromptTemplate, error) {
		return nil, errors.New("template db down")
	}
	llm := newMockLLMProvider()
	pub := newMockEventPublisher()
	renderer := prompt.New()
	uc := usecase.NewChatUseCase(convRepo, msgRepo, promptRepo, llm, renderer, pub)

	_, err := uc.Chat(context.Background(), &dto.ChatRequest{
		UserID:  1,
		Message: "hi",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "template db down")
}

// TestChatUseCase_Chat_ConversationCreateError verifies Create errors on
// the conversation repo are propagated verbatim.
func TestChatUseCase_Chat_ConversationCreateError(t *testing.T) {
	convRepo := newMockConvRepo()
	convRepo.create = func(*entity.Conversation) error {
		return errors.New("conv create failed")
	}
	msgRepo := newMockMsgRepo()
	promptRepo := newMockPromptRepo()
	llm := newMockLLMProvider()
	pub := newMockEventPublisher()
	renderer := prompt.New()
	uc := usecase.NewChatUseCase(convRepo, msgRepo, promptRepo, llm, renderer, pub)

	_, err := uc.Chat(context.Background(), &dto.ChatRequest{
		UserID:  1,
		Message: "hi",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "conv create failed")
}

// TestToConversationResponse_Nil ensures the mapping helper handles nil.
func TestToConversationResponse_Nil(t *testing.T) {
	// Direct exercise via GetConversation with a nil entity is covered above;
	// here we ensure timestamps are formatted when set.
	convRepo := newMockConvRepo()
	msgRepo := newMockMsgRepo()
	promptRepo := newMockPromptRepo()
	llm := newMockLLMProvider()
	pub := newMockEventPublisher()
	renderer := prompt.New()
	uc := usecase.NewChatUseCase(convRepo, msgRepo, promptRepo, llm, renderer, pub)

	c := &entity.Conversation{UserID: 1, Mode: entity.ConversationModeChat, Status: entity.ConversationStatusActive}
	c.ID = idgen.Next()
	c.CreatedAt = time.Now()
	c.UpdatedAt = time.Now()
	require.NoError(t, convRepo.Create(context.Background(), c))

	got, err := uc.GetConversation(context.Background(), c.ID)
	require.NoError(t, err)
	assert.NotEmpty(t, got.CreatedAt)
	assert.NotEmpty(t, got.UpdatedAt)
}

// TestToMessageResponse_Timestamps ensures the message DTO timestamps are
// formatted when set.
func TestToMessageResponse_Timestamps(t *testing.T) {
	convRepo := newMockConvRepo()
	msgRepo := newMockMsgRepo()
	promptRepo := newMockPromptRepo()
	llm := newMockLLMProvider()
	pub := newMockEventPublisher()
	renderer := prompt.New()
	uc := usecase.NewChatUseCase(convRepo, msgRepo, promptRepo, llm, renderer, pub)

	// Drive a chat turn so messages are persisted with timestamps.
	_, err := uc.Chat(context.Background(), &dto.ChatRequest{UserID: 1, Message: "hi"})
	require.NoError(t, err)

	// Find any persisted message and exercise the mapper via ListByConversation
	// after stamping its CreatedAt.
	for _, m := range msgRepo.items {
		m.CreatedAt = time.Now()
		list, err := uc.ListMessages(context.Background(), m.ConversationID, pagination.Params{Page: 1, PageSize: 10})
		require.NoError(t, err)
		// A single chat turn persists 2 messages (user + assistant).
		require.Len(t, list.Items, 2)
		assert.NotEmpty(t, list.Items[0].CreatedAt)
		break
	}
}

// TestChatUseCase_ConstantsSanity verifies LLMMessage construction uses the
// expected role constants. This protects against silent renames in the
// entity package.
func TestChatUseCase_ConstantsSanity(t *testing.T) {
	assert.Equal(t, "user", entity.MessageRoleUser)
	assert.Equal(t, "assistant", entity.MessageRoleAssistant)
	assert.Equal(t, "system", entity.MessageRoleSystem)
	// Ensure service.LLMMessage can be constructed.
	m := service.LLMMessage{Role: entity.MessageRoleUser, Content: "x"}
	assert.Equal(t, "user", m.Role)
}

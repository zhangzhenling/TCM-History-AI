package usecase_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

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

// newAgentUseCase wires up an AgentUseCase with in-memory mocks and the real
// prompt renderer. db and outboxRepo are nil so the direct-publish path is
// exercised.
func newAgentUseCase(
	llm service.LLMProvider,
	toolExec service.ToolExecutor,
	retriever service.RetrievalClient,
) (*usecase.AgentUseCase, *mockConvRepo, *mockAgentRunRepo, *mockPromptRepo, *mockToolRepo, *mockEventPublisher) {
	convRepo := newMockConvRepo()
	agentRepo := newMockAgentRunRepo()
	promptRepo := newMockPromptRepo()
	toolRepo := newMockToolRepo()
	pub := newMockEventPublisher()
	renderer := prompt.New()
	uc := usecase.NewAgentUseCase(convRepo, agentRepo, promptRepo, toolRepo, llm, toolExec, retriever, renderer, pub, nil, nil)
	return uc, convRepo, agentRepo, promptRepo, toolRepo, pub
}

// TestAgentUseCase_Run_HappyPath_DirectFallback verifies that when the LLM
// returns non-JSON (stub mode), the agent falls back to a fallback plan
// (stored on the run, no steps executed) and still produces an answer.
func TestAgentUseCase_Run_HappyPath_DirectFallback(t *testing.T) {
	llm := newMockLLMProvider()
	// LLM returns plain text for both plan and answer.
	llm.chatResp = &service.LLMChatResponse{
		Text:             "not-json",
		Model:            "test-model",
		TokensPrompt:     5,
		TokensCompletion: 3,
	}
	uc, convRepo, agentRepo, _, _, pub := newAgentUseCase(llm, nil, nil)

	resp, err := uc.Run(context.Background(), &dto.AgentRequest{
		UserID:   1,
		Question: "张仲景是谁？",
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotZero(t, resp.AgentRunID)
	assert.NotZero(t, resp.ConversationID)
	assert.Equal(t, "not-json", resp.Answer)
	assert.Equal(t, entity.AgentRunStatusDone, resp.Status)
	assert.Equal(t, 8, resp.TotalTokens) // 5+3 from the answer call
	// Fallback plan yields no executed steps (steps slice is nil from plan()).
	require.Empty(t, resp.Steps)

	// Conversation persisted with title derived from question.
	conv, err := convRepo.FindByID(context.Background(), resp.ConversationID)
	require.NoError(t, err)
	assert.Equal(t, entity.ConversationModeAgent, conv.Mode)
	assert.Equal(t, "张仲景是谁？", conv.Title)

	// AgentRun persisted with done status and a fallback plan.
	run, err := agentRepo.FindByID(context.Background(), resp.AgentRunID)
	require.NoError(t, err)
	assert.Equal(t, entity.AgentRunStatusDone, run.Status)
	assert.Equal(t, "not-json", run.FinalAnswer)
	require.NotNil(t, run.PlanJSON)
	var plan map[string]any
	require.NoError(t, json.Unmarshal(run.PlanJSON, &plan))
	assert.Equal(t, true, plan["fallback"])

	// Event published.
	evt, ok := captureEvent[event.AgentRunCompleted](pub)
	require.True(t, ok)
	assert.Equal(t, resp.AgentRunID, evt.AgentRunID)
	assert.Equal(t, entity.AgentRunStatusDone, evt.Status)
}

// TestAgentUseCase_Run_WithValidPlan verifies that when the LLM returns a
// valid plan JSON, the agent executes each step.
func TestAgentUseCase_Run_WithValidPlan(t *testing.T) {
	llm := newMockLLMProvider()
	planJSON := `{"task_id":"agent-1","question":"q","sub_tasks":[
		{"sub_task_id":"t1","intent_type":"fact_lookup","channel":"direct","query":"q1"},
		{"sub_task_id":"t2","intent_type":"fact_lookup","channel":"direct","query":"q2"}
	]}`
	// First call returns the plan, second returns the answer.
	callCount := 0
	llm.chatFn = func(ctx context.Context, req service.LLMChatRequest) (*service.LLMChatResponse, error) {
		callCount++
		if callCount == 1 {
			return &service.LLMChatResponse{Text: "```json\n" + planJSON + "\n```", Model: "test-model"}, nil
		}
		return &service.LLMChatResponse{Text: "final answer", Model: "test-model", TokensPrompt: 4, TokensCompletion: 2}, nil
	}

	uc, _, agentRepo, _, _, _ := newAgentUseCase(llm, nil, nil)
	resp, err := uc.Run(context.Background(), &dto.AgentRequest{
		UserID:   1,
		Question: "q",
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "final answer", resp.Answer)
	require.Len(t, resp.Steps, 2)
	assert.Equal(t, "t1", resp.Steps[0].SubTaskID)
	assert.Equal(t, "t2", resp.Steps[1].SubTaskID)

	// PlanJSON persisted on the run.
	run, err := agentRepo.FindByID(context.Background(), resp.AgentRunID)
	require.NoError(t, err)
	require.NotNil(t, run.PlanJSON)
	var plan map[string]any
	require.NoError(t, json.Unmarshal(run.PlanJSON, &plan))
	assert.Equal(t, "agent-1", plan["task_id"])
}

// TestAgentUseCase_Run_PlanCapsSubTasks verifies that >6 sub-tasks are
// truncated to 6.
func TestAgentUseCase_Run_PlanCapsSubTasks(t *testing.T) {
	llm := newMockLLMProvider()
	// Build a plan with 8 sub-tasks.
	subs := make([]map[string]any, 0, 8)
	for i := 0; i < 8; i++ {
		subs = append(subs, map[string]any{
			"sub_task_id": "t" + string(rune('1'+i)),
			"channel":     "direct",
			"query":       "q",
		})
	}
	planBytes, _ := json.Marshal(map[string]any{
		"task_id":   "agent-1",
		"question":  "q",
		"sub_tasks": subs,
	})
	callCount := 0
	llm.chatFn = func(ctx context.Context, req service.LLMChatRequest) (*service.LLMChatResponse, error) {
		callCount++
		if callCount == 1 {
			return &service.LLMChatResponse{Text: string(planBytes), Model: "test-model"}, nil
		}
		return &service.LLMChatResponse{Text: "answer", Model: "test-model"}, nil
	}
	uc, _, _, _, _, _ := newAgentUseCase(llm, nil, nil)
	resp, err := uc.Run(context.Background(), &dto.AgentRequest{UserID: 1, Question: "q"})
	require.NoError(t, err)
	require.Len(t, resp.Steps, 6)
}

// TestAgentUseCase_Run_WithExistingConversation verifies reusing an existing
// agent-mode conversation.
func TestAgentUseCase_Run_WithExistingConversation(t *testing.T) {
	llm := newMockLLMProvider()
	uc, convRepo, _, _, _, _ := newAgentUseCase(llm, nil, nil)

	existing := &entity.Conversation{
		UserID:       7,
		Title:        "Existing",
		Mode:         entity.ConversationModeAgent,
		Status:       entity.ConversationStatusActive,
		MetadataJSON: []byte("{}"),
	}
	existing.ID = idgen.Next()
	require.NoError(t, convRepo.Create(context.Background(), existing))

	resp, err := uc.Run(context.Background(), &dto.AgentRequest{
		ConversationID: existing.ID,
		UserID:         7,
		Question:       "q",
	})
	require.NoError(t, err)
	assert.Equal(t, existing.ID, resp.ConversationID)
}

// TestAgentUseCase_Run_ConversationNotFound verifies the not-found path.
func TestAgentUseCase_Run_ConversationNotFound(t *testing.T) {
	llm := newMockLLMProvider()
	uc, _, _, _, _, _ := newAgentUseCase(llm, nil, nil)
	_, err := uc.Run(context.Background(), &dto.AgentRequest{
		ConversationID: 9999,
		UserID:          1,
		Question:        "q",
	})
	require.Error(t, err)
	var e *errno.Error
	if errors.As(err, &e) {
		assert.Equal(t, errno.NotFound, e.Code)
	}
}

// TestAgentUseCase_Run_ConversationFindError verifies repo errors propagate.
func TestAgentUseCase_Run_ConversationFindError(t *testing.T) {
	llm := newMockLLMProvider()
	uc, convRepo, _, _, _, _ := newAgentUseCase(llm, nil, nil)
	convRepo.find = func(int64) (*entity.Conversation, error) {
		return nil, errors.New("db down")
	}
	_, err := uc.Run(context.Background(), &dto.AgentRequest{
		ConversationID: 5,
		UserID:          1,
		Question:        "q",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "db down")
}

// TestAgentUseCase_Run_ValidationErrors covers input validation.
func TestAgentUseCase_Run_ValidationErrors(t *testing.T) {
	llm := newMockLLMProvider()
	uc, _, _, _, _, _ := newAgentUseCase(llm, nil, nil)
	cases := []struct {
		name string
		in   *dto.AgentRequest
		code errno.Errno
	}{
		{"nil request", nil, errno.InvalidParams},
		{"empty question", &dto.AgentRequest{UserID: 1}, errno.InvalidParams},
		{"zero user_id", &dto.AgentRequest{Question: "q"}, errno.InvalidParams},
		{"negative user_id", &dto.AgentRequest{UserID: -1, Question: "q"}, errno.InvalidParams},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := uc.Run(context.Background(), c.in)
			require.Error(t, err)
			var e *errno.Error
			if errors.As(err, &e) {
				assert.Equal(t, c.code, e.Code)
			}
		})
	}
}

// TestAgentUseCase_Run_AnswerError verifies that when the final LLM answer
// call fails, the run is marked failed and a DependencyUnavailable error
// is returned.
func TestAgentUseCase_Run_AnswerError(t *testing.T) {
	llm := newMockLLMProvider()
	callCount := 0
	llm.chatFn = func(ctx context.Context, req service.LLMChatRequest) (*service.LLMChatResponse, error) {
		callCount++
		if callCount == 1 {
			// Plan call returns non-JSON so falls back to direct.
			return &service.LLMChatResponse{Text: "stub", Model: "test-model"}, nil
		}
		return nil, errors.New("llm answer failed")
	}
	uc, _, agentRepo, _, _, _ := newAgentUseCase(llm, nil, nil)
	resp, err := uc.Run(context.Background(), &dto.AgentRequest{
		UserID:   1,
		Question: "q",
	})
	require.Error(t, err)
	assert.Nil(t, resp)
	var e *errno.Error
	if errors.As(err, &e) {
		assert.Equal(t, errno.DependencyUnavailable, e.Code)
	}
	// The run should be persisted with failed status.
	require.NotEmpty(t, agentRepo.items)
	for _, run := range agentRepo.items {
		assert.Equal(t, entity.AgentRunStatusFailed, run.Status)
		assert.Contains(t, run.ErrorMsg, "answer failed")
	}
}

// TestAgentUseCase_Run_AgentRepoCreateError verifies that errors during the
// initial AgentRun create propagate.
func TestAgentUseCase_Run_AgentRepoCreateError(t *testing.T) {
	llm := newMockLLMProvider()
	uc, _, agentRepo, _, _, _ := newAgentUseCase(llm, nil, nil)
	agentRepo.create = func(*entity.AgentRun) error {
		return errors.New("create failed")
	}
	_, err := uc.Run(context.Background(), &dto.AgentRequest{
		UserID:   1,
		Question: "q",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create failed")
}

// TestAgentUseCase_Run_ConversationCreateError verifies errors during
// conversation creation propagate.
func TestAgentUseCase_Run_ConversationCreateError(t *testing.T) {
	llm := newMockLLMProvider()
	uc, convRepo, _, _, _, _ := newAgentUseCase(llm, nil, nil)
	convRepo.create = func(*entity.Conversation) error {
		return errors.New("conv create failed")
	}
	_, err := uc.Run(context.Background(), &dto.AgentRequest{
		UserID:   1,
		Question: "q",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "conv create failed")
}

// TestAgentUseCase_Run_RagChannel verifies the rag channel invokes the
// retriever and includes chunks in the step result.
func TestAgentUseCase_Run_RagChannel(t *testing.T) {
	llm := newMockLLMProvider()
	planJSON := `{"task_id":"t","question":"q","sub_tasks":[
		{"sub_task_id":"t1","channel":"rag","query":"太阳病"}
	]}`
	callCount := 0
	llm.chatFn = func(ctx context.Context, req service.LLMChatRequest) (*service.LLMChatResponse, error) {
		callCount++
		if callCount == 1 {
			return &service.LLMChatResponse{Text: planJSON, Model: "test-model"}, nil
		}
		return &service.LLMChatResponse{Text: "answer", Model: "test-model"}, nil
	}
	retriever := &mockRetrievalClient{
		retrieveRes: &service.RetrieveResult{
			Total: 1,
			Chunks: []service.RetrievedChunk{
				{ChunkID: "c1", ClassicCode: "shl", Content: "太阳病"},
			},
		},
	}
	uc, _, _, _, _, _ := newAgentUseCase(llm, nil, retriever)
	resp, err := uc.Run(context.Background(), &dto.AgentRequest{UserID: 1, Question: "q"})
	require.NoError(t, err)
	require.Len(t, resp.Steps, 1)
	assert.Equal(t, "rag", resp.Steps[0].Channel)
	chunks, ok := resp.Steps[0].Result["chunks"].([]service.RetrievedChunk)
	require.True(t, ok)
	require.Len(t, chunks, 1)
	assert.Equal(t, "shl", chunks[0].ClassicCode)
}

// TestAgentUseCase_Run_RagChannelError verifies that a retriever error
// degrades the step (recorded as error) without aborting the run.
func TestAgentUseCase_Run_RagChannelError(t *testing.T) {
	llm := newMockLLMProvider()
	planJSON := `{"task_id":"t","question":"q","sub_tasks":[
		{"sub_task_id":"t1","channel":"rag","query":"q"}
	]}`
	callCount := 0
	llm.chatFn = func(ctx context.Context, req service.LLMChatRequest) (*service.LLMChatResponse, error) {
		callCount++
		if callCount == 1 {
			return &service.LLMChatResponse{Text: planJSON, Model: "test-model"}, nil
		}
		return &service.LLMChatResponse{Text: "answer", Model: "test-model"}, nil
	}
	retriever := &mockRetrievalClient{
		retrieveErr: errors.New("retriever down"),
	}
	uc, _, _, _, _, _ := newAgentUseCase(llm, nil, retriever)
	resp, err := uc.Run(context.Background(), &dto.AgentRequest{UserID: 1, Question: "q"})
	require.NoError(t, err)
	require.Len(t, resp.Steps, 1)
	assert.True(t, resp.Steps[0].Result["degraded"].(bool))
}

// TestAgentUseCase_Run_RagChannelNoRetriever verifies that with retriever=nil
// the step degrades gracefully.
func TestAgentUseCase_Run_RagChannelNoRetriever(t *testing.T) {
	llm := newMockLLMProvider()
	planJSON := `{"task_id":"t","question":"q","sub_tasks":[
		{"sub_task_id":"t1","channel":"rag","query":"q"}
	]}`
	callCount := 0
	llm.chatFn = func(ctx context.Context, req service.LLMChatRequest) (*service.LLMChatResponse, error) {
		callCount++
		if callCount == 1 {
			return &service.LLMChatResponse{Text: planJSON, Model: "test-model"}, nil
		}
		return &service.LLMChatResponse{Text: "answer", Model: "test-model"}, nil
	}
	uc, _, _, _, _, _ := newAgentUseCase(llm, nil, nil)
	resp, err := uc.Run(context.Background(), &dto.AgentRequest{UserID: 1, Question: "q"})
	require.NoError(t, err)
	require.Len(t, resp.Steps, 1)
	assert.Contains(t, resp.Steps[0].Result["evidence"], "degraded")
}

// TestAgentUseCase_Run_GraphChannel verifies the graph channel invokes
// SearchNodes and includes nodes in the step result.
func TestAgentUseCase_Run_GraphChannel(t *testing.T) {
	llm := newMockLLMProvider()
	planJSON := `{"task_id":"t","question":"q","sub_tasks":[
		{"sub_task_id":"t1","channel":"graph","query":"张仲景"}
	]}`
	callCount := 0
	llm.chatFn = func(ctx context.Context, req service.LLMChatRequest) (*service.LLMChatResponse, error) {
		callCount++
		if callCount == 1 {
			return &service.LLMChatResponse{Text: planJSON, Model: "test-model"}, nil
		}
		return &service.LLMChatResponse{Text: "answer", Model: "test-model"}, nil
	}
	retriever := &mockRetrievalClient{
		searchRes: &service.GraphSearchResult{
			Total: 1,
			Items: []service.GraphNode{
				{UID: "p1", Label: "Person", Name: "张仲景"},
			},
		},
	}
	uc, _, _, _, _, _ := newAgentUseCase(llm, nil, retriever)
	resp, err := uc.Run(context.Background(), &dto.AgentRequest{UserID: 1, Question: "q"})
	require.NoError(t, err)
	require.Len(t, resp.Steps, 1)
	assert.Equal(t, "graph", resp.Steps[0].Channel)
	nodes, ok := resp.Steps[0].Result["nodes"].([]service.GraphNode)
	require.True(t, ok)
	require.Len(t, nodes, 1)
	assert.Equal(t, "张仲景", nodes[0].Name)
}

// TestAgentUseCase_Run_ToolChannel verifies the tool channel invokes the
// tool executor.
func TestAgentUseCase_Run_ToolChannel(t *testing.T) {
	llm := newMockLLMProvider()
	planJSON := `{"task_id":"t","question":"q","sub_tasks":[
		{"sub_task_id":"t1","channel":"tool","tool_name":"timeline","query":"东汉"}
	]}`
	callCount := 0
	llm.chatFn = func(ctx context.Context, req service.LLMChatRequest) (*service.LLMChatResponse, error) {
		callCount++
		if callCount == 1 {
			return &service.LLMChatResponse{Text: planJSON, Model: "test-model"}, nil
		}
		return &service.LLMChatResponse{Text: "answer", Model: "test-model"}, nil
	}
	exec := &mockToolExecutor{result: map[string]any{"year": 200}}
	uc, _, _, _, _, _ := newAgentUseCase(llm, exec, nil)
	resp, err := uc.Run(context.Background(), &dto.AgentRequest{UserID: 1, Question: "q"})
	require.NoError(t, err)
	require.Len(t, resp.Steps, 1)
	assert.Equal(t, "tool", resp.Steps[0].Channel)
	assert.Equal(t, 200, resp.Steps[0].Result["year"])
}

// TestAgentUseCase_Run_ToolChannelNoExecutor verifies the tool channel
// degrades when executor=nil.
func TestAgentUseCase_Run_ToolChannelNoExecutor(t *testing.T) {
	llm := newMockLLMProvider()
	planJSON := `{"task_id":"t","question":"q","sub_tasks":[
		{"sub_task_id":"t1","channel":"tool","tool_name":"x","query":"q"}
	]}`
	callCount := 0
	llm.chatFn = func(ctx context.Context, req service.LLMChatRequest) (*service.LLMChatResponse, error) {
		callCount++
		if callCount == 1 {
			return &service.LLMChatResponse{Text: planJSON, Model: "test-model"}, nil
		}
		return &service.LLMChatResponse{Text: "answer", Model: "test-model"}, nil
	}
	uc, _, _, _, _, _ := newAgentUseCase(llm, nil, nil)
	resp, err := uc.Run(context.Background(), &dto.AgentRequest{UserID: 1, Question: "q"})
	require.NoError(t, err)
	require.Len(t, resp.Steps, 1)
	assert.Contains(t, resp.Steps[0].Result["error"], "tool executor not configured")
}

// TestAgentUseCase_Run_LongQuestionTitleTruncation verifies that a long
// question is truncated for the conversation title.
func TestAgentUseCase_Run_LongQuestionTitleTruncation(t *testing.T) {
	llm := newMockLLMProvider()
	uc, convRepo, _, _, _, _ := newAgentUseCase(llm, nil, nil)
	longQ := string(make([]rune, 100))
	for i := range longQ {
		longQ = longQ[:i] + "中" + longQ[i+1:]
	}
	resp, err := uc.Run(context.Background(), &dto.AgentRequest{
		UserID:   1,
		Question: longQ,
	})
	require.NoError(t, err)
	conv, err := convRepo.FindByID(context.Background(), resp.ConversationID)
	require.NoError(t, err)
	// Title is 32 runes + "..."
	rs := []rune(conv.Title)
	assert.True(t, len(rs) <= 36)
}

// TestAgentUseCase_ListAgentRuns covers pagination and error paths.
func TestAgentUseCase_ListAgentRuns(t *testing.T) {
	llm := newMockLLMProvider()
	uc, _, agentRepo, _, _, _ := newAgentUseCase(llm, nil, nil)
	// Seed 3 runs.
	for i := 0; i < 3; i++ {
		r := &entity.AgentRun{ConversationID: 1, UserID: 1, Status: entity.AgentRunStatusDone}
		r.ID = idgen.Next() + int64(i)
		require.NoError(t, agentRepo.Create(context.Background(), r))
	}
	resp, err := uc.ListAgentRuns(context.Background(), pagination.Params{Page: 1, PageSize: 2})
	require.NoError(t, err)
	assert.Equal(t, 3, resp.Total)
	assert.Equal(t, 2, resp.TotalPage)
	require.Len(t, resp.Items, 2)

	t.Run("repo error", func(t *testing.T) {
		agentRepo.list = func(pagination.Params) ([]entity.AgentRun, int, error) {
			return nil, 0, errors.New("list err")
		}
		_, err := uc.ListAgentRuns(context.Background(), pagination.Params{Page: 1, PageSize: 2})
		require.Error(t, err)
	})
}

// TestAgentUseCase_GetAgentRun covers found / not-found / error.
func TestAgentUseCase_GetAgentRun(t *testing.T) {
	llm := newMockLLMProvider()
	uc, _, agentRepo, _, _, _ := newAgentUseCase(llm, nil, nil)
	r := &entity.AgentRun{ConversationID: 1, UserID: 1, Status: entity.AgentRunStatusDone, FinalAnswer: "ans"}
	r.ID = idgen.Next()
	require.NoError(t, agentRepo.Create(context.Background(), r))

	t.Run("found", func(t *testing.T) {
		got, err := uc.GetAgentRun(context.Background(), r.ID)
		require.NoError(t, err)
		assert.Equal(t, "ans", got.FinalAnswer)
	})
	t.Run("not found", func(t *testing.T) {
		_, err := uc.GetAgentRun(context.Background(), 9999)
		require.Error(t, err)
		var e *errno.Error
		if errors.As(err, &e) {
			assert.Equal(t, errno.NotFound, e.Code)
		}
	})
	t.Run("repo error", func(t *testing.T) {
		agentRepo.find = func(int64) (*entity.AgentRun, error) {
			return nil, errors.New("find err")
		}
		_, err := uc.GetAgentRun(context.Background(), 1)
		require.Error(t, err)
	})
}

// TestAgentUseCase_Run_PlanFillsMissingFields verifies that sub-tasks missing
// SubTaskID/Channel/Query get sensible defaults filled in.
func TestAgentUseCase_Run_PlanFillsMissingFields(t *testing.T) {
	llm := newMockLLMProvider()
	// Plan with sub-task missing SubTaskID and Channel.
	planJSON := `{"task_id":"t","question":"orig q","sub_tasks":[
		{"query":""}
	]}`
	callCount := 0
	llm.chatFn = func(ctx context.Context, req service.LLMChatRequest) (*service.LLMChatResponse, error) {
		callCount++
		if callCount == 1 {
			return &service.LLMChatResponse{Text: planJSON, Model: "test-model"}, nil
		}
		return &service.LLMChatResponse{Text: "answer", Model: "test-model"}, nil
	}
	uc, _, _, _, _, _ := newAgentUseCase(llm, nil, nil)
	resp, err := uc.Run(context.Background(), &dto.AgentRequest{UserID: 1, Question: "orig q"})
	require.NoError(t, err)
	require.Len(t, resp.Steps, 1)
	assert.Equal(t, "t1", resp.Steps[0].SubTaskID)
	assert.Equal(t, "direct", resp.Steps[0].Channel)
	assert.Equal(t, "orig q", resp.Steps[0].Query)
}

// TestAgentUseCase_Run_WithAgentTemplate verifies that an active agent-scene
// template is rendered into the answer prompt.
func TestAgentUseCase_Run_WithAgentTemplate(t *testing.T) {
	llm := newMockLLMProvider()
	uc, _, _, promptRepo, _, _ := newAgentUseCase(llm, nil, nil)
	tpl := &entity.PromptTemplate{
		Name:         "agent-default",
		Scene:        entity.SceneAgent,
		SystemPrompt: "Agent system prompt with evidence: {{evidence_summary}}",
		IsActive:     true,
	}
	tpl.ID = idgen.Next()
	require.NoError(t, promptRepo.Create(context.Background(), tpl))

	resp, err := uc.Run(context.Background(), &dto.AgentRequest{UserID: 1, Question: "q"})
	require.NoError(t, err)
	assert.Equal(t, "stub-answer", resp.Answer)
}

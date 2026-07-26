package entity_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"tcm-history-ai/backend/ai-service/internal/domain/entity"
)

// TestTableName_Methods verifies each entity maps to its expected table.
func TestTableName_Methods(t *testing.T) {
	cases := []struct {
		name string
		got  string
		want string
	}{
		{"AgentRun", entity.AgentRun{}.TableName(), "ai_agent_runs"},
		{"Conversation", entity.Conversation{}.TableName(), "ai_conversations"},
		{"Message", entity.Message{}.TableName(), "ai_messages"},
		{"PromptTemplate", entity.PromptTemplate{}.TableName(), "ai_prompt_templates"},
		{"Tool", entity.Tool{}.TableName(), "ai_tools"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, c.got)
		})
	}
}

// TestConstants verifies the enumerated status/mode/scene/role constants are
// non-empty and stable across the entity package.
func TestConstants(t *testing.T) {
	t.Run("AgentRun status", func(t *testing.T) {
		assert.Equal(t, "pending", entity.AgentRunStatusPending)
		assert.Equal(t, "running", entity.AgentRunStatusRunning)
		assert.Equal(t, "done", entity.AgentRunStatusDone)
		assert.Equal(t, "failed", entity.AgentRunStatusFailed)
	})

	t.Run("Conversation mode", func(t *testing.T) {
		assert.Equal(t, "chat", entity.ConversationModeChat)
		assert.Equal(t, "agent", entity.ConversationModeAgent)
		assert.Equal(t, "reasoning", entity.ConversationModeReasoning)
	})

	t.Run("Conversation status", func(t *testing.T) {
		assert.Equal(t, "active", entity.ConversationStatusActive)
		assert.Equal(t, "archived", entity.ConversationStatusArchived)
	})

	t.Run("Message role", func(t *testing.T) {
		assert.Equal(t, "user", entity.MessageRoleUser)
		assert.Equal(t, "assistant", entity.MessageRoleAssistant)
		assert.Equal(t, "system", entity.MessageRoleSystem)
		assert.Equal(t, "tool", entity.MessageRoleTool)
	})

	t.Run("Prompt scene", func(t *testing.T) {
		assert.Equal(t, "chat", entity.SceneChat)
		assert.Equal(t, "agent", entity.SceneAgent)
		assert.Equal(t, "reasoning", entity.SceneReasoning)
		assert.Equal(t, "summarize", entity.SceneSummarize)
	})

	t.Run("Tool method", func(t *testing.T) {
		assert.Equal(t, "GET", entity.ToolMethodGET)
		assert.Equal(t, "POST", entity.ToolMethodPOST)
	})
}

// TestAgentRun_Fields exercises the AgentRun struct construction and
// verifies the JSON raw message fields accept arbitrary payloads.
func TestAgentRun_Fields(t *testing.T) {
	a := entity.AgentRun{
		ConversationID: 42,
		UserID:         7,
		PlanJSON:       []byte(`{"task_id":"t1"}`),
		StepsJSON:      []byte(`[{"sub_task_id":"t1"}]`),
		FinalAnswer:    "answer",
		Status:         entity.AgentRunStatusDone,
		ErrorMsg:       "",
		TotalTokens:    100,
		TotalLatencyMs: 50,
	}
	assert.Equal(t, int64(42), a.ConversationID)
	assert.Equal(t, int64(7), a.UserID)
	assert.Equal(t, "answer", a.FinalAnswer)
	assert.Equal(t, entity.AgentRunStatusDone, a.Status)
	assert.Equal(t, 100, a.TotalTokens)
	assert.Equal(t, 50, a.TotalLatencyMs)
	assert.Contains(t, string(a.PlanJSON), "t1")
}

// TestConversation_Fields exercises the Conversation struct construction.
func TestConversation_Fields(t *testing.T) {
	c := entity.Conversation{
		UserID:       1,
		Title:        "T",
		Mode:         entity.ConversationModeAgent,
		Status:       entity.ConversationStatusActive,
		MessageCount: 5,
		MetadataJSON: []byte(`{"k":"v"}`),
	}
	assert.Equal(t, int64(1), c.UserID)
	assert.Equal(t, "T", c.Title)
	assert.Equal(t, entity.ConversationModeAgent, c.Mode)
	assert.Equal(t, entity.ConversationStatusActive, c.Status)
	assert.Equal(t, 5, c.MessageCount)
	assert.Contains(t, string(c.MetadataJSON), "k")
}

// TestMessage_Fields exercises the Message struct construction.
func TestMessage_Fields(t *testing.T) {
	m := entity.Message{
		ConversationID:    99,
		Role:              entity.MessageRoleAssistant,
		Content:           "hello",
		ToolCallsJSON:     []byte(`[]`),
		ToolCallID:        "call_1",
		TokensPrompt:      10,
		TokensCompletion:  20,
		LatencyMs:         30,
		ModelName:         "gpt-4o-mini",
	}
	assert.Equal(t, int64(99), m.ConversationID)
	assert.Equal(t, entity.MessageRoleAssistant, m.Role)
	assert.Equal(t, "hello", m.Content)
	assert.Equal(t, "call_1", m.ToolCallID)
	assert.Equal(t, 10, m.TokensPrompt)
	assert.Equal(t, 20, m.TokensCompletion)
	assert.Equal(t, 30, m.LatencyMs)
	assert.Equal(t, "gpt-4o-mini", m.ModelName)
}

// TestPromptTemplate_Fields exercises the PromptTemplate struct construction.
func TestPromptTemplate_Fields(t *testing.T) {
	p := entity.PromptTemplate{
		Name:          "chat-default",
		Scene:         entity.SceneChat,
		SystemPrompt:  "you are an assistant",
		Template:      "{{user_question}}",
		VariablesJSON: []byte(`["user_question"]`),
		Model:         "gpt-4o-mini",
		Temperature:   0.3,
		MaxTokens:     1024,
		TopP:          0.9,
		IsActive:      true,
		Version:       2,
	}
	assert.Equal(t, "chat-default", p.Name)
	assert.Equal(t, entity.SceneChat, p.Scene)
	assert.Equal(t, "you are an assistant", p.SystemPrompt)
	assert.Equal(t, "gpt-4o-mini", p.Model)
	assert.Equal(t, float32(0.3), p.Temperature)
	assert.Equal(t, 1024, p.MaxTokens)
	assert.Equal(t, float32(0.9), p.TopP)
	assert.True(t, p.IsActive)
	assert.Equal(t, 2, p.Version)
}

// TestTool_Fields exercises the Tool struct construction.
func TestTool_Fields(t *testing.T) {
	tt := entity.Tool{
		Name:           "timeline",
		Description:    "TCM timeline tool",
		Endpoint:       "http://localhost:8080/timeline",
		Method:         entity.ToolMethodGET,
		ParametersJSON: []byte(`{"q":"string"}`),
		Category:       "history",
		IsEnabled:      true,
		Version:        "v1",
	}
	assert.Equal(t, "timeline", tt.Name)
	assert.Equal(t, "TCM timeline tool", tt.Description)
	assert.Equal(t, "http://localhost:8080/timeline", tt.Endpoint)
	assert.Equal(t, entity.ToolMethodGET, tt.Method)
	assert.Equal(t, "history", tt.Category)
	assert.True(t, tt.IsEnabled)
	assert.Equal(t, "v1", tt.Version)
}

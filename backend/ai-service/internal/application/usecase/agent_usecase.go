package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"tcm-history-ai/backend/ai-service/internal/application/dto"
	"tcm-history-ai/backend/ai-service/internal/domain/entity"
	"tcm-history-ai/backend/ai-service/internal/domain/event"
	"tcm-history-ai/backend/ai-service/internal/domain/repository"
	"tcm-history-ai/backend/ai-service/internal/domain/service"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/idgen"
	"tcm-history-ai/backend/pkg/pagination"
)

// AgentUseCase implements Agent 编排：Plan → Execute → Answer。
//
// 流程参考 doc/07-Agent设计.md §3 / §4，本实现为可运行的最小骨架：
//  1. Plan: 解析用户问题，生成简单步骤计划（stub：单步 direct）
//  2. Execute: 逐步执行（可选 ToolExecutor 调用，stub 模式下直接返回桩结果）
//  3. Answer: 调用 LLMProvider 整合证据生成最终答案
//  4. 持久化 AgentRun，发布 AgentRunCompleted 事件
//
// 真实接入 Planner/Reasoner/Reflection 多阶段链路在 doc/07 §3 详述，
// 此处保留接口边界，后续按阶段替换 stub 实现。
type AgentUseCase struct {
	convRepo   repository.ConversationRepository
	agentRepo  repository.AgentRunRepository
	promptRepo repository.PromptTemplateRepository
	toolRepo   repository.ToolRepository
	llm        service.LLMProvider
	toolExec   service.ToolExecutor
	renderer   service.PromptRenderer
	pub        event.EventPublisher
}

// NewAgentUseCase constructs an AgentUseCase.
func NewAgentUseCase(
	convRepo repository.ConversationRepository,
	agentRepo repository.AgentRunRepository,
	promptRepo repository.PromptTemplateRepository,
	toolRepo repository.ToolRepository,
	llm service.LLMProvider,
	toolExec service.ToolExecutor,
	renderer service.PromptRenderer,
	pub event.EventPublisher,
) *AgentUseCase {
	return &AgentUseCase{
		convRepo:   convRepo,
		agentRepo:  agentRepo,
		promptRepo: promptRepo,
		toolRepo:   toolRepo,
		llm:        llm,
		toolExec:   toolExec,
		renderer:   renderer,
		pub:        pub,
	}
}

// Run executes an Agent run for the given question.
func (uc *AgentUseCase) Run(ctx context.Context, in *dto.AgentRequest) (*dto.AgentResponse, error) {
	if in == nil || in.Question == "" {
		return nil, errno.New(errno.InvalidParams, "question is required")
	}
	userID := in.UserID
	if userID <= 0 {
		return nil, errno.New(errno.InvalidParams, "user_id is required")
	}
	start := time.Now()

	// 1. 加载或创建 Conversation（mode=agent）
	conv, err := uc.ensureConversation(ctx, in)
	if err != nil {
		return nil, err
	}

	// 2. 创建 AgentRun（pending）
	run := &entity.AgentRun{
		ConversationID: conv.ID,
		UserID:         userID,
		Status:         entity.AgentRunStatusPending,
	}
	run.ID = idgen.Next()
	if err := uc.agentRepo.Create(ctx, run); err != nil {
		return nil, err
	}
	// 标记 running
	run.Status = entity.AgentRunStatusRunning
	_ = uc.agentRepo.Update(ctx, run)

	// 3. Plan: 生成步骤计划（stub：单步 direct）
	plan, steps, planErr := uc.plan(ctx, in.Question)
	if planErr != nil {
		run.Status = entity.AgentRunStatusFailed
		run.ErrorMsg = "plan failed: " + planErr.Error()
		_ = uc.agentRepo.Update(ctx, run)
		return nil, errno.Wrap(errno.InternalError, "agent plan", planErr)
	}
	if planJSON, err := json.Marshal(plan); err == nil {
		run.PlanJSON = planJSON
	}

	// 4. Execute: 逐步执行
	executed := make([]dto.AgentStep, 0, len(steps))
	totalTokens := 0
	for _, st := range steps {
		result, exErr := uc.executeStep(ctx, st)
		if exErr != nil {
			// 单步失败不阻塞整体，记录降级结果
			result = map[string]any{
				"sub_task_id": st.SubTaskID,
				"error":       exErr.Error(),
				"degraded":    true,
			}
		}
		st.Result = result
		executed = append(executed, st)
		if t, ok := result["tokens"].(int); ok {
			totalTokens += t
		}
	}
	if stepsJSON, err := json.Marshal(executed); err == nil {
		run.StepsJSON = stepsJSON
	}

	// 5. Answer: 调用 LLM 整合证据
	systemPrompt, _ := uc.buildAnswerPrompt(ctx, in.Question, executed)
	answer, err := uc.llm.Chat(ctx, service.LLMChatRequest{
		Model:    uc.llm.Model(),
		System:   systemPrompt,
		Messages: []service.LLMMessage{{Role: entity.MessageRoleUser, Content: in.Question}},
	})
	if err != nil {
		run.Status = entity.AgentRunStatusFailed
		run.ErrorMsg = "answer failed: " + err.Error()
		_ = uc.agentRepo.Update(ctx, run)
		return nil, errno.Wrap(errno.DependencyUnavailable, "agent answer", err)
	}
	run.FinalAnswer = answer.Text
	run.TotalTokens = totalTokens + answer.TokensPrompt + answer.TokensCompletion
	run.TotalLatencyMs = int(time.Since(start).Milliseconds())
	run.Status = entity.AgentRunStatusDone
	_ = uc.agentRepo.Update(ctx, run)

	// 6. 发布事件（best-effort）
	_ = uc.pub.Publish(ctx, event.AgentRunCompleted{
		AgentRunID:     run.ID,
		ConversationID: conv.ID,
		UserID:         userID,
		Status:         run.Status,
	})

	return &dto.AgentResponse{
		AgentRunID:     run.ID,
		ConversationID: conv.ID,
		Answer:         answer.Text,
		Steps:          executed,
		TotalTokens:    run.TotalTokens,
		TotalLatencyMs: run.TotalLatencyMs,
		Status:         run.Status,
	}, nil
}

// plan produces a structured Plan + step list. stub 实现：单步 direct 通道。
//
// TODO(agent-sdk): 接入真实 Planner LLM 调用（强模型，强制 JSON Schema 输出），
// 参考 doc/07 §3.1：限制子任务数 3~6 个，支持意图类型 fact_lookup /
// thought_summary / relation_path / origin_cite / compare。
func (uc *AgentUseCase) plan(ctx context.Context, question string) (map[string]any, []dto.AgentStep, error) {
	step := dto.AgentStep{
		SubTaskID:  "t1",
		IntentType: "fact_lookup",
		Channel:    "direct",
		Query:      question,
	}
	plan := map[string]any{
		"task_id":   fmt.Sprintf("agent-%d", time.Now().UnixNano()),
		"question":  question,
		"sub_tasks": []dto.AgentStep{step},
	}
	return plan, []dto.AgentStep{step}, nil
}

// executeStep runs a single plan step. 在 stub 模式下直接返回桩结果。
//
// TODO(agent-sdk): 接入真实 Reasoner 路由决策（rag/graph/tool/direct），
// 对 rag 通道调用 Knowledge Service.HybridSearch Kitex RPC，
// 对 graph 通道调用 Graph Service.FindPath，
// 对 tool 通道调用 ToolExecutor。
func (uc *AgentUseCase) executeStep(ctx context.Context, st dto.AgentStep) (map[string]any, error) {
	switch st.Channel {
	case "tool":
		if uc.toolExec == nil {
			return map[string]any{"error": "tool executor not configured"}, nil
		}
		return uc.toolExec.Execute(ctx, st.ToolName, map[string]any{"query": st.Query})
	default:
		// direct / rag / graph: stub 返回桩证据
		return map[string]any{
			"sub_task_id": st.SubTaskID,
			"channel":     st.Channel,
			"query":        st.Query,
			"evidence":     "[stub-agent] 未接入 Knowledge/Graph Service，使用桩证据",
			"tokens":        0,
		}, nil
	}
}

// buildAnswerPrompt renders the active agent scene prompt.
func (uc *AgentUseCase) buildAnswerPrompt(ctx context.Context, question string, steps []dto.AgentStep) (string, error) {
	tpl, err := uc.promptRepo.FindActive(ctx, entity.SceneAgent)
	if err != nil {
		return "", err
	}
	if tpl == nil {
		return "你是一名中医发展史研究型 Agent，整合证据生成带来源标注的回答。", nil
	}
	vars := map[string]any{
		"user_question": question,
	}
	if stepsJSON, err := json.Marshal(steps); err == nil {
		vars["steps"] = string(stepsJSON)
	}
	return uc.renderer.Render(tpl.SystemPrompt, vars)
}

// ensureConversation loads or creates an agent-mode conversation.
func (uc *AgentUseCase) ensureConversation(ctx context.Context, in *dto.AgentRequest) (*entity.Conversation, error) {
	if in.ConversationID > 0 {
		c, err := uc.convRepo.FindByID(ctx, in.ConversationID)
		if err != nil {
			return nil, err
		}
		if c == nil {
			return nil, errno.New(errno.NotFound, "conversation not found: "+strconv.FormatInt(in.ConversationID, 10))
		}
		return c, nil
	}
	title := in.Question
	if len([]rune(title)) > 32 {
		title = string([]rune(title)[:32]) + "..."
	}
	c := &entity.Conversation{
		UserID:       in.UserID,
		Title:        title,
		Mode:         entity.ConversationModeAgent,
		Status:       entity.ConversationStatusActive,
		MetadataJSON: []byte("{}"),
	}
	c.ID = idgen.Next()
	if err := uc.convRepo.Create(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

// ListAgentRuns returns paginated agent runs.
func (uc *AgentUseCase) ListAgentRuns(ctx context.Context, p pagination.Params) (dto.ListResponse[dto.AgentRunResponse], error) {
	items, total, err := uc.agentRepo.List(ctx, p)
	if err != nil {
		return dto.ListResponse[dto.AgentRunResponse]{}, err
	}
	resp := make([]dto.AgentRunResponse, 0, len(items))
	for i := range items {
		resp = append(resp, *toAgentRunResponse(&items[i]))
	}
	return dto.NewListResponse(p, total, resp), nil
}

// GetAgentRun fetches a single agent run by id.
func (uc *AgentUseCase) GetAgentRun(ctx context.Context, id int64) (*dto.AgentRunResponse, error) {
	a, err := uc.agentRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if a == nil {
		return nil, errno.New(errno.NotFound, "agent run not found")
	}
	return toAgentRunResponse(a), nil
}

// toAgentRunResponse maps the entity to its wire DTO.
func toAgentRunResponse(a *entity.AgentRun) *dto.AgentRunResponse {
	if a == nil {
		return nil
	}
	resp := &dto.AgentRunResponse{
		ID:             a.ID,
		ConversationID: a.ConversationID,
		UserID:         a.UserID,
		PlanJSON:       a.PlanJSON,
		StepsJSON:      a.StepsJSON,
		FinalAnswer:    a.FinalAnswer,
		Status:         a.Status,
		ErrorMsg:       a.ErrorMsg,
		TotalTokens:    a.TotalTokens,
		TotalLatencyMs: a.TotalLatencyMs,
	}
	if !a.CreatedAt.IsZero() {
		resp.CreatedAt = a.CreatedAt.Format(time.RFC3339)
	}
	if !a.UpdatedAt.IsZero() {
		resp.UpdatedAt = a.UpdatedAt.Format(time.RFC3339)
	}
	return resp
}

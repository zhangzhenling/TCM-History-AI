package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
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
// 流程参考 doc/07-Agent设计.md §3 / §4：
//  1. Plan: 调用 LLM 解析用户问题，输出结构化子任务计划（JSON Schema 约束）
//  2. Execute: 逐步执行，按 channel 路由到 rag/graph/tool/direct
//     - rag   → RetrievalClient.Retrieve (Knowledge Service)
//     - graph → RetrievalClient.SearchNodes (Graph Service)
//     - tool  → ToolExecutor
//     - direct → 跳过检索，直接交由 LLM 生成
//  3. Answer: 调用 LLM 整合证据生成最终答案
//  4. 持久化 AgentRun，发布 AgentRunCompleted 事件
//
// 在 LLM 处于 stub 模式或上游服务未配置时，链路降级为桩证据，
// 仍然返回可运行的响应，保证本地开发联调可用。
type AgentUseCase struct {
	convRepo   repository.ConversationRepository
	agentRepo  repository.AgentRunRepository
	promptRepo repository.PromptTemplateRepository
	toolRepo   repository.ToolRepository
	llm        service.LLMProvider
	toolExec   service.ToolExecutor
	retriever  service.RetrievalClient
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
	retriever service.RetrievalClient,
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
		retriever:  retriever,
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

	// 3. Plan: 调用 LLM 生成结构化子任务计划
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

// planRequest is the JSON schema we ask the LLM to emit when planning.
// 子任务数限制 3~6 个，参考 doc/07 §3.1。
type planRequest struct {
	TaskID   string       `json:"task_id"`
	Question string       `json:"question"`
	SubTasks []dto.AgentStep `json:"sub_tasks"`
}

const planSystemPrompt = `你是一名中医发展史研究型 Agent 的规划器。
将用户问题拆解为 1~4 个子任务，每个子任务必须包含：
- sub_task_id: 字符串，形如 "t1" / "t2"
- intent_type: fact_lookup | thought_summary | relation_path | origin_cite | compare
- channel: rag | graph | tool | direct
  * rag   → 需要检索经典原文（如《伤寒论》《黄帝内经》片段）
  * graph → 需要查询人物/学派/朝代关联（如师承、影响）
  * tool  → 需要调用 MCP Tool（如 TimelineTool / PrescriptionTool）
  * direct → 无需检索，可直接回答（如常识性问题）
- query: 该子任务的检索或调用 query
- tool_name: 仅当 channel=tool 时填写，其余为空

输出严格的 JSON，不要包含任何解释性文字。格式：
{"task_id":"agent-<timestamp>","question":"<原问题>","sub_tasks":[...]}`

// plan produces a structured Plan + step list by calling the LLM.
//
// 在 LLM 处于 stub 模式时，stub 响应不是合法 JSON，本方法捕获错误后
// 回退到单步 direct 通道，保证链路可运行。
func (uc *AgentUseCase) plan(ctx context.Context, question string) (map[string]any, []dto.AgentStep, error) {
	resp, err := uc.llm.Chat(ctx, service.LLMChatRequest{
		System: planSystemPrompt,
		Messages: []service.LLMMessage{
			{Role: entity.MessageRoleUser, Content: question},
		},
		Temperature: 0.2,
	})
	if err != nil {
		// 降级：单步 direct
		return uc.fallbackPlan(question), nil, nil
	}

	// 尝试从响应中提取 JSON（LLM 可能包裹在 ```json ... ``` 中）
	jsonStr := extractJSON(resp.Text)
	var parsed planRequest
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil || len(parsed.SubTasks) == 0 {
		// 降级：单步 direct
		return uc.fallbackPlan(question), nil, nil
	}

	// 限制子任务数 1~6
	if len(parsed.SubTasks) > 6 {
		parsed.SubTasks = parsed.SubTasks[:6]
	}
	// 补全缺失字段
	for i := range parsed.SubTasks {
		if parsed.SubTasks[i].SubTaskID == "" {
			parsed.SubTasks[i].SubTaskID = fmt.Sprintf("t%d", i+1)
		}
		if parsed.SubTasks[i].Channel == "" {
			parsed.SubTasks[i].Channel = "direct"
		}
		if parsed.SubTasks[i].Query == "" {
			parsed.SubTasks[i].Query = question
		}
	}

	plan := map[string]any{
		"task_id":   parsed.TaskID,
		"question":  parsed.Question,
		"sub_tasks": parsed.SubTasks,
		"model":     resp.Model,
	}
	return plan, parsed.SubTasks, nil
}

// fallbackPlan returns a single direct-channel step when LLM planning fails.
func (uc *AgentUseCase) fallbackPlan(question string) map[string]any {
	step := dto.AgentStep{
		SubTaskID:  "t1",
		IntentType: "fact_lookup",
		Channel:    "direct",
		Query:      question,
	}
	return map[string]any{
		"task_id":   fmt.Sprintf("agent-%d", time.Now().UnixNano()),
		"question":  question,
		"sub_tasks": []dto.AgentStep{step},
		"fallback":  true,
	}
}

// executeStep runs a single plan step by dispatching to the right channel.
func (uc *AgentUseCase) executeStep(ctx context.Context, st dto.AgentStep) (map[string]any, error) {
	switch st.Channel {
	case "rag":
		if uc.retriever == nil {
			return map[string]any{
				"sub_task_id": st.SubTaskID,
				"channel":     st.Channel,
				"query":       st.Query,
				"evidence":    "[degraded] retriever not configured",
				"tokens":      0,
			}, nil
		}
		result, err := uc.retriever.Retrieve(ctx, st.Query, 5)
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"sub_task_id": st.SubTaskID,
			"channel":     st.Channel,
			"query":       st.Query,
			"chunks":      result.Chunks,
			"total":       result.Total,
			"latency_ms":  result.LatencyMs,
			"tokens":      0,
		}, nil
	case "graph":
		if uc.retriever == nil {
			return map[string]any{
				"sub_task_id": st.SubTaskID,
				"channel":     st.Channel,
				"query":       st.Query,
				"evidence":    "[degraded] retriever not configured",
				"tokens":      0,
			}, nil
		}
		result, err := uc.retriever.SearchNodes(ctx, st.Query, "", 10)
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"sub_task_id": st.SubTaskID,
			"channel":     st.Channel,
			"query":       st.Query,
			"nodes":       result.Items,
			"total":       result.Total,
			"tokens":      0,
		}, nil
	case "tool":
		if uc.toolExec == nil {
			return map[string]any{"error": "tool executor not configured"}, nil
		}
		return uc.toolExec.Execute(ctx, st.ToolName, map[string]any{"query": st.Query})
	default:
		// direct: 不检索，留给 Answer 阶段整合
		return map[string]any{
			"sub_task_id": st.SubTaskID,
			"channel":     st.Channel,
			"query":        st.Query,
			"evidence":     "[direct] no retrieval needed",
			"tokens":        0,
		}, nil
	}
}

// extractJSON strips Markdown code fences and surrounding prose so that
// LLM-emitted JSON can be parsed even when wrapped in ```json ... ``` blocks.
func extractJSON(s string) string {
	s = strings.TrimSpace(s)
	// Strip ```json ... ``` fences
	if strings.HasPrefix(s, "```") {
		// Remove opening fence (with optional language tag)
		if idx := strings.Index(s, "\n"); idx >= 0 {
			s = s[idx+1:]
		}
		// Remove closing fence
		if idx := strings.LastIndex(s, "```"); idx >= 0 {
			s = s[:idx]
		}
		s = strings.TrimSpace(s)
	}
	// Extract the largest {...} block if there is surrounding prose.
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start >= 0 && end > start {
		return s[start : end+1]
	}
	return s
}

// buildAnswerPrompt renders the active agent scene prompt, augmented with
// the evidence collected during step execution so the LLM can ground its
// final answer on the retrieved chunks / graph nodes.
func (uc *AgentUseCase) buildAnswerPrompt(ctx context.Context, question string, steps []dto.AgentStep) (string, error) {
	tpl, err := uc.promptRepo.FindActive(ctx, entity.SceneAgent)
	if err != nil {
		return "", err
	}
	vars := map[string]any{
		"user_question": question,
	}
	if stepsJSON, err := json.Marshal(steps); err == nil {
		vars["steps"] = string(stepsJSON)
	}
	// 拼装证据摘要供 LLM 参考
	vars["evidence_summary"] = summarizeEvidence(steps)

	if tpl == nil {
		return "你是一名中医发展史研究型 Agent，基于检索证据生成带来源标注的回答。", nil
	}
	return uc.renderer.Render(tpl.SystemPrompt, vars)
}

// summarizeEvidence produces a compact text summary of retrieved evidence
// suitable for injection into the answer prompt.
func summarizeEvidence(steps []dto.AgentStep) string {
	if len(steps) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("已检索证据：\n")
	for _, st := range steps {
		sb.WriteString(fmt.Sprintf("- [%s] %s\n", st.Channel, st.Query))
		if st.Result == nil {
			continue
		}
		if chunks, ok := st.Result["chunks"].([]service.RetrievedChunk); ok && len(chunks) > 0 {
			for i, c := range chunks {
				if i >= 3 {
					break
				}
				text := c.Content
				if c.TextOriginal != "" {
					text = c.TextOriginal
				}
				sb.WriteString(fmt.Sprintf("  • 来源:%s 内容:%s\n", c.ClassicCode, truncate(text, 200)))
			}
		}
		if nodes, ok := st.Result["nodes"].([]service.GraphNode); ok && len(nodes) > 0 {
			for i, n := range nodes {
				if i >= 3 {
					break
				}
				sb.WriteString(fmt.Sprintf("  • 节点:%s(%s)\n", n.Name, n.Label))
			}
		}
	}
	return sb.String()
}

// truncate clips a string to at most n runes, appending an ellipsis.
func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "..."
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

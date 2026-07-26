package usecase

import (
	"context"
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

// ChatUseCase implements single/multi-turn chat conversations.
//
// 流程：
//  1. 加载/创建 Conversation
//  2. 渲染 Prompt 模板（按 scene 取激活模板，叠加用户变量）
//  3. 拉取历史消息组装 LLM 上下文
//  4. 调用 LLMProvider 生成回复（stub 模式下返回桩响应）
//  5. 持久化 user/assistant 两条消息，更新 Conversation.message_count
//  6. 发布 ChatMessageCreated 事件
type ChatUseCase struct {
	convRepo  repository.ConversationRepository
	msgRepo   repository.MessageRepository
	promptRepo repository.PromptTemplateRepository
	llm       service.LLMProvider
	renderer  service.PromptRenderer
	pub       event.EventPublisher
}

// NewChatUseCase constructs a ChatUseCase.
func NewChatUseCase(
	convRepo repository.ConversationRepository,
	msgRepo repository.MessageRepository,
	promptRepo repository.PromptTemplateRepository,
	llm service.LLMProvider,
	renderer service.PromptRenderer,
	pub event.EventPublisher,
) *ChatUseCase {
	return &ChatUseCase{
		convRepo:   convRepo,
		msgRepo:    msgRepo,
		promptRepo: promptRepo,
		llm:        llm,
		renderer:   renderer,
		pub:        pub,
	}
}

// Chat runs a single chat turn.
func (uc *ChatUseCase) Chat(ctx context.Context, in *dto.ChatRequest) (*dto.ChatResponse, error) {
	if in == nil || in.Message == "" {
		return nil, errno.New(errno.InvalidParams, "message is required")
	}
	userID := in.UserID
	if userID <= 0 {
		return nil, errno.New(errno.InvalidParams, "user_id is required")
	}

	// 1. 加载或创建 Conversation
	conv, err := uc.ensureConversation(ctx, in)
	if err != nil {
		return nil, err
	}

	// 2. 渲染 system prompt
	systemPrompt, err := uc.buildSystemPrompt(ctx, in)
	if err != nil {
		return nil, err
	}

	// 3. 拉取历史消息（最近若干条）
	history, err := uc.msgRepo.FindByConversation(ctx, conv.ID)
	if err != nil {
		return nil, errno.Wrap(errno.DependencyUnavailable, "load chat history", err)
	}
	messages := make([]service.LLMMessage, 0, len(history)+2)
	if systemPrompt != "" {
		messages = append(messages, service.LLMMessage{Role: entity.MessageRoleSystem, Content: systemPrompt})
	}
	for i := range history {
		messages = append(messages, service.LLMMessage{
			Role:    history[i].Role,
			Content: history[i].Content,
		})
	}
	messages = append(messages, service.LLMMessage{Role: entity.MessageRoleUser, Content: in.Message})

	// 4. 持久化 user 消息
	userMsg := &entity.Message{
		ConversationID: conv.ID,
		Role:           entity.MessageRoleUser,
		Content:        in.Message,
	}
	userMsg.ID = idgen.Next()
	if err := uc.msgRepo.Create(ctx, userMsg); err != nil {
		return nil, err
	}

	// 5. 调用 LLM
	start := time.Now()
	resp, err := uc.llm.Chat(ctx, service.LLMChatRequest{
		Model:    uc.llm.Model(),
		System:   systemPrompt,
		Messages: messages,
	})
	latencyMs := int(time.Since(start).Milliseconds())
	if err != nil {
		return nil, errno.Wrap(errno.DependencyUnavailable, "llm chat", err)
	}

	// 6. 持久化 assistant 消息
	assistantMsg := &entity.Message{
		ConversationID:   conv.ID,
		Role:            entity.MessageRoleAssistant,
		Content:         resp.Text,
		TokensPrompt:    resp.TokensPrompt,
		TokensCompletion: resp.TokensCompletion,
		LatencyMs:       latencyMs,
		ModelName:       resp.Model,
	}
	assistantMsg.ID = idgen.Next()
	if err := uc.msgRepo.Create(ctx, assistantMsg); err != nil {
		return nil, err
	}

	// 7. 更新 Conversation
	conv.MessageCount += 2
	if conv.Title == "" && len(in.Message) > 0 {
		title := in.Message
		if len([]rune(title)) > 32 {
			title = string([]rune(title)[:32]) + "..."
		}
		conv.Title = title
	}
	_ = uc.convRepo.Update(ctx, conv)

	// 8. 发布事件（best-effort）
	_ = uc.pub.Publish(ctx, event.ChatMessageCreated{
		ConversationID: conv.ID,
		MessageID:      assistantMsg.ID,
		UserID:         userID,
		Role:           entity.MessageRoleAssistant,
		ModelName:      resp.Model,
	})

	return &dto.ChatResponse{
		ConversationID:   conv.ID,
		MessageID:        assistantMsg.ID,
		Assistant:         resp.Text,
		Model:            resp.Model,
		TokensPrompt:     resp.TokensPrompt,
		TokensCompletion: resp.TokensCompletion,
		LatencyMs:        latencyMs,
	}, nil
}

// ensureConversation loads an existing conversation or creates a new one.
func (uc *ChatUseCase) ensureConversation(ctx context.Context, in *dto.ChatRequest) (*entity.Conversation, error) {
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
	mode := in.Mode
	if mode == "" {
		mode = entity.ConversationModeChat
	}
	c := &entity.Conversation{
		UserID:  in.UserID,
		Title:   "",
		Mode:    mode,
		Status:  entity.ConversationStatusActive,
		MetadataJSON: []byte("{}"),
	}
	c.ID = idgen.Next()
	if err := uc.convRepo.Create(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

// buildSystemPrompt renders the active template for the requested scene.
func (uc *ChatUseCase) buildSystemPrompt(ctx context.Context, in *dto.ChatRequest) (string, error) {
	scene := in.TemplateScene
	if scene == "" {
		scene = entity.SceneChat
	}
	tpl, err := uc.promptRepo.FindActive(ctx, scene)
	if err != nil {
		return "", err
	}
	if tpl == nil {
		// 没有模板时退回默认 system prompt
		return "你是一名中医发展史领域的研究型导师，依据提供的上下文准确回答用户问题。", nil
	}
	variables := in.Variables
	if variables == nil {
		variables = map[string]any{}
	}
	variables["user_question"] = in.Message
	rendered, err := uc.renderer.Render(tpl.SystemPrompt, variables)
	if err != nil {
		return "", err
	}
	return rendered, nil
}

// ListConversations returns paginated conversations for a user.
func (uc *ChatUseCase) ListConversations(ctx context.Context, userID int64, p pagination.Params) (dto.ListResponse[dto.ConversationResponse], error) {
	items, total, err := uc.convRepo.ListByUser(ctx, userID, p)
	if err != nil {
		return dto.ListResponse[dto.ConversationResponse]{}, err
	}
	resp := make([]dto.ConversationResponse, 0, len(items))
	for i := range items {
		resp = append(resp, *toConversationResponse(&items[i]))
	}
	return dto.NewListResponse(p, total, resp), nil
}

// GetConversation fetches a single conversation by id.
func (uc *ChatUseCase) GetConversation(ctx context.Context, id int64) (*dto.ConversationResponse, error) {
	c, err := uc.convRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, errno.New(errno.NotFound, "conversation not found")
	}
	return toConversationResponse(c), nil
}

// DeleteConversation soft-deletes a conversation.
func (uc *ChatUseCase) DeleteConversation(ctx context.Context, id int64) error {
	return uc.convRepo.Delete(ctx, id)
}

// ListMessages returns paginated messages for a conversation.
func (uc *ChatUseCase) ListMessages(ctx context.Context, conversationID int64, p pagination.Params) (dto.ListResponse[dto.MessageResponse], error) {
	items, total, err := uc.msgRepo.ListByConversation(ctx, conversationID, p)
	if err != nil {
		return dto.ListResponse[dto.MessageResponse]{}, err
	}
	resp := make([]dto.MessageResponse, 0, len(items))
	for i := range items {
		resp = append(resp, *toMessageResponse(&items[i]))
	}
	return dto.NewListResponse(p, total, resp), nil
}

// toConversationResponse maps the entity to its wire DTO.
func toConversationResponse(c *entity.Conversation) *dto.ConversationResponse {
	if c == nil {
		return nil
	}
	resp := &dto.ConversationResponse{
		ID:           c.ID,
		UserID:       c.UserID,
		Title:        c.Title,
		Mode:         c.Mode,
		Status:       c.Status,
		MessageCount: c.MessageCount,
		MetadataJSON: c.MetadataJSON,
	}
	if !c.CreatedAt.IsZero() {
		resp.CreatedAt = c.CreatedAt.Format(time.RFC3339)
	}
	if !c.UpdatedAt.IsZero() {
		resp.UpdatedAt = c.UpdatedAt.Format(time.RFC3339)
	}
	return resp
}

// toMessageResponse maps the entity to its wire DTO.
func toMessageResponse(m *entity.Message) *dto.MessageResponse {
	if m == nil {
		return nil
	}
	resp := &dto.MessageResponse{
		ID:               m.ID,
		ConversationID:   m.ConversationID,
		Role:             m.Role,
		Content:          m.Content,
		ToolCallsJSON:    m.ToolCallsJSON,
		ToolCallID:       m.ToolCallID,
		TokensPrompt:     m.TokensPrompt,
		TokensCompletion: m.TokensCompletion,
		LatencyMs:        m.LatencyMs,
		ModelName:        m.ModelName,
	}
	if !m.CreatedAt.IsZero() {
		resp.CreatedAt = m.CreatedAt.Format(time.RFC3339)
	}
	return resp
}

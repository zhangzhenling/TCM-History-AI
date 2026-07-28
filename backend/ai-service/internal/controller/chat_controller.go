package controller

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"

	"tcm-history-ai/backend/ai-service/internal/application/dto"
	"tcm-history-ai/backend/ai-service/internal/application/usecase"
)

// ChatController exposes HTTP handlers for chat conversations.
type ChatController struct {
	uc *usecase.ChatUseCase
}

// NewChatController constructs a ChatController.
func NewChatController(uc *usecase.ChatUseCase) *ChatController {
	return &ChatController{uc: uc}
}

// Chat POST /api/v1/ai/chat
func (h *ChatController) Chat(ctx context.Context, c *app.RequestContext) {
	var req dto.ChatRequest
	if !bindAndValidate(ctx, c, &req) {
		return
	}
	// header X-User-ID 优先于 body user_id
	if uid := userIDFromHeader(c); uid > 0 {
		req.UserID = uid
	}
	resp, err := h.uc.Chat(ctx, &req)
	okOrFail(ctx, c, resp, err)
}

// ListConversations GET /api/v1/ai/conversations
func (h *ChatController) ListConversations(ctx context.Context, c *app.RequestContext) {
	userID := userIDFromHeader(c)
	p := pageParams(c)
	resp, err := h.uc.ListConversations(ctx, userID, p)
	okOrFail(ctx, c, resp, err)
}

// GetConversation GET /api/v1/ai/conversations/:id
func (h *ChatController) GetConversation(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(ctx, c)
	if !ok {
		return
	}
	resp, err := h.uc.GetConversation(ctx, id)
	okOrFail(ctx, c, resp, err)
}

// ListMessages GET /api/v1/ai/conversations/:id/messages
func (h *ChatController) ListMessages(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(ctx, c)
	if !ok {
		return
	}
	p := pageParams(c)
	resp, err := h.uc.ListMessages(ctx, id, p)
	okOrFail(ctx, c, resp, err)
}

// DeleteConversation DELETE /api/v1/ai/conversations/:id
func (h *ChatController) DeleteConversation(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(ctx, c)
	if !ok {
		return
	}
	err := h.uc.DeleteConversation(ctx, id)
	noContentOrFail(ctx, c, err)
}

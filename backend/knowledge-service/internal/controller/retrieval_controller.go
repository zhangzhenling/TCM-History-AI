package controller

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"

	"tcm-history-ai/backend/knowledge-service/internal/application/dto"
	"tcm-history-ai/backend/knowledge-service/internal/application/usecase"
)

// RetrievalController exposes HTTP handlers for RAG retrieval.
type RetrievalController struct {
	uc *usecase.RetrievalUseCase
}

// NewRetrievalController constructs a RetrievalController.
func NewRetrievalController(uc *usecase.RetrievalUseCase) *RetrievalController {
	return &RetrievalController{uc: uc}
}

// Retrieve POST /api/v1/knowledge/retrieve
func (h *RetrievalController) Retrieve(ctx context.Context, c *app.RequestContext) {
	var req dto.RetrieveRequest
	if !bindAndValidate(ctx, c, &req) {
		return
	}
	userID := userIDFromHeader(c)
	resp, err := h.uc.Retrieve(ctx, &req, userID)
	okOrFail(ctx, c, resp, err)
}

// Feedback POST /api/v1/knowledge/queries/:id/feedback
func (h *RetrievalController) Feedback(ctx context.Context, c *app.RequestContext) {
	queryLogID, ok := pathID(ctx, c)
	if !ok {
		return
	}
	var req dto.FeedbackRequest
	if !bindAndValidate(ctx, c, &req) {
		return
	}
	err := h.uc.Feedback(ctx, queryLogID, req.Feedback)
	noContentOrFail(ctx, c, err)
}

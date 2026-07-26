package controller

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"

	"tcm-history-ai/backend/knowledge-service/internal/application/usecase"
	"tcm-history-ai/backend/pkg/response"
)

// IngestController exposes HTTP handlers for the RAG ingestion pipeline.
type IngestController struct {
	uc *usecase.IngestUseCase
}

// NewIngestController constructs an IngestController.
func NewIngestController(uc *usecase.IngestUseCase) *IngestController {
	return &IngestController{uc: uc}
}

// Ingest POST /api/v1/knowledge/documents/:id/ingest
// 触发 RAG 写入侧流水线：切片 → Embedding → Milvus 入库。
// 请求体可选携带 markdown_text 字段；为空时从 MinIO markdown bucket 拉取。
func (h *IngestController) Ingest(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var body struct {
		MarkdownText string `json:"markdown_text,optional"`
	}
	_ = c.BindJSON(&body) // 允许空 body

	if err := h.uc.IngestMarkdown(ctx, id, body.MarkdownText); err != nil {
		response.Fail(ctx, c, err)
		return
	}
	response.OKWith(ctx, c, "ingestion completed", map[string]any{
		"document_id": id,
		"status":      "embedded",
	})
}

package controller

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"

	"tcm-history-ai/backend/history-service/internal/application/usecase"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/response"
)

// UploadController exposes the multipart upload endpoint for person
// portraits, book PDFs and other binary attachments.
type UploadController struct {
	uc *usecase.UploadUseCase
}

// NewUploadController constructs an UploadController.
func NewUploadController(uc *usecase.UploadUseCase) *UploadController {
	return &UploadController{uc: uc}
}

// Upload POST /api/v1/history/upload
//
// Multipart form fields:
//   - file:    the binary payload (required)
//   - purpose: storage prefix e.g. "portraits", "books" (optional, default "misc")
func (h *UploadController) Upload(ctx context.Context, c *app.RequestContext) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		response.Fail(ctx, c, errno.Wrap(errno.InvalidParams, "file field is required", err))
		return
	}
	purpose := string(c.FormValue("purpose"))
	src, err := fileHeader.Open()
	if err != nil {
		response.Fail(ctx, c, errno.Wrap(errno.InvalidParams, "open uploaded file", err))
		return
	}
	defer func() { _ = src.Close() }()

	resp, err := h.uc.Upload(ctx, purpose, fileHeader.Filename, fileHeader.Header.Get("Content-Type"), fileHeader.Size, src)
	okOrFail(ctx, c, resp, err)
}

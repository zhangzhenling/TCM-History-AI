package usecase

import (
	"context"
	"io"
	"strconv"
	"time"

	"tcm-history-ai/backend/knowledge-service/internal/application/dto"
	"tcm-history-ai/backend/knowledge-service/internal/domain/entity"
	"tcm-history-ai/backend/knowledge-service/internal/domain/event"
	"tcm-history-ai/backend/knowledge-service/internal/domain/repository"
	"tcm-history-ai/backend/knowledge-service/internal/infrastructure/storage"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/idgen"
	"tcm-history-ai/backend/pkg/pagination"
)

// DocumentUseCase implements CRUD operations on documents.
type DocumentUseCase struct {
	repo  repository.DocumentRepository
	pub   event.EventPublisher
	minio *storage.MinIOClient
}

// NewDocumentUseCase constructs a DocumentUseCase. minio 可为 nil（离线开发时）。
func NewDocumentUseCase(repo repository.DocumentRepository, pub event.EventPublisher, minio *storage.MinIOClient) *DocumentUseCase {
	return &DocumentUseCase{repo: repo, pub: pub, minio: minio}
}

// Create persists a new document. If content_hash matches an existing doc,
// returns the existing one (dedup).
func (uc *DocumentUseCase) Create(ctx context.Context, in *dto.DocumentRequest) (*dto.DocumentResponse, error) {
	if in == nil || in.Title == "" {
		return nil, errno.New(errno.InvalidParams, "title is required")
	}
	if in.ClassicCode == "" {
		return nil, errno.New(errno.InvalidParams, "classic_code is required")
	}
	// dedup by content_hash
	if in.ContentHash != "" {
		if existing, err := uc.repo.FindByContentHash(ctx, in.ContentHash); err != nil {
			return nil, err
		} else if existing != nil {
			return toDocumentResponse(existing), nil
		}
	}
	sourceType := in.SourceType
	if sourceType == "" {
		sourceType = entity.SourceBook
	}
	d := &entity.Document{
		ClassicCode:       in.ClassicCode,
		Title:             in.Title,
		Version:           in.Version,
		Dynasty:           in.Dynasty,
		School:            in.School,
		Author:            in.Author,
		SourceType:        sourceType,
		SourceRef:         in.SourceRef,
		FileURL:           in.FileURL,
		PDFObjectKey:      in.PDFObjectKey,
		MarkdownObjectKey: in.MarkdownObjectKey,
		MimeType:          in.MimeType,
		ContentHash:       in.ContentHash,
		Status:            entity.DocumentStatusPending,
		VolumeCount:       in.VolumeCount,
		ClauseCount:       in.ClauseCount,
		MetadataJSON:      in.MetadataJSON,
	}
	d.ID = idgen.Next()
	if d.MetadataJSON == nil {
		d.MetadataJSON = []byte("{}")
	}
	if err := uc.repo.Create(ctx, d); err != nil {
		return nil, err
	}
	// 发文献上传事件，触发下游 Worker 处理
	if d.PDFObjectKey != "" {
		bucket := ""
		if uc.minio != nil {
			bucket = uc.minio.OriginalBucket()
		}
		_ = uc.pub.Publish(ctx, event.DocumentUploaded{
			DocumentID:  d.ID,
			ClassicCode: d.ClassicCode,
			ObjectKey:   d.PDFObjectKey,
			Bucket:      bucket,
		})
	}
	return toDocumentResponse(d), nil
}

// UploadMarkdown 上传 Markdown 文本到 MinIO markdown bucket 并创建 Document。
// objectKey 为空时自动生成 docs/{docID}/markdown.md。
// 上传成功后 Document.MarkdownObjectKey 会被填充。
func (uc *DocumentUseCase) UploadMarkdown(ctx context.Context, in *dto.DocumentRequest, markdownText string, objectKey string) (*dto.DocumentResponse, error) {
	if in == nil || in.Title == "" {
		return nil, errno.New(errno.InvalidParams, "title is required")
	}
	if in.ClassicCode == "" {
		return nil, errno.New(errno.InvalidParams, "classic_code is required")
	}
	if markdownText == "" {
		return nil, errno.New(errno.InvalidParams, "markdown text is required")
	}

	// 先创建 Document 拿到 ID
	resp, err := uc.Create(ctx, in)
	if err != nil {
		return nil, err
	}

	if uc.minio == nil {
		// MinIO 未启用：MarkdownObjectKey 留空，调用方需自行通过 IngestMarkdown 传入文本
		return resp, nil
	}

	if objectKey == "" {
		objectKey = "docs/" + strconv.FormatInt(resp.ID, 10) + "/markdown.md"
	}

	if err := uc.minio.PutMarkdown(ctx, objectKey, stringReader(markdownText), int64(len(markdownText))); err != nil {
		return nil, err
	}

	// 更新 Document.MarkdownObjectKey
	d, err := uc.repo.FindByID(ctx, resp.ID)
	if err != nil || d == nil {
		return resp, err
	}
	d.MarkdownObjectKey = objectKey
	d.Status = entity.DocumentStatusMarked
	if err := uc.repo.Update(ctx, d); err != nil {
		return nil, err
	}
	return toDocumentResponse(d), nil
}

// stringReader 将 string 包装为 io.Reader。
func stringReader(s string) io.Reader { return &stringReaderImpl{s: s} }

type stringReaderImpl struct {
	s   string
	pos int
}

func (r *stringReaderImpl) Read(p []byte) (int, error) {
	if r.pos >= len(r.s) {
		return 0, io.EOF
	}
	n := copy(p, r.s[r.pos:])
	r.pos += n
	return n, nil
}

// Update modifies an existing document.
func (uc *DocumentUseCase) Update(ctx context.Context, id int64, in *dto.DocumentRequest) (*dto.DocumentResponse, error) {
	if in == nil {
		return nil, errno.New(errno.InvalidParams, "body is required")
	}
	d, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if d == nil {
		return nil, errno.New(errno.NotFound, "document not found: "+strconv.FormatInt(id, 10))
	}
	d.ClassicCode = in.ClassicCode
	d.Title = in.Title
	d.Version = in.Version
	d.Dynasty = in.Dynasty
	d.School = in.School
	d.Author = in.Author
	d.SourceRef = in.SourceRef
	d.FileURL = in.FileURL
	d.PDFObjectKey = in.PDFObjectKey
	d.MarkdownObjectKey = in.MarkdownObjectKey
	d.MimeType = in.MimeType
	d.ContentHash = in.ContentHash
	d.VolumeCount = in.VolumeCount
	d.ClauseCount = in.ClauseCount
	if in.MetadataJSON != nil {
		d.MetadataJSON = in.MetadataJSON
	}
	if err := uc.repo.Update(ctx, d); err != nil {
		return nil, err
	}
	return toDocumentResponse(d), nil
}

// Delete soft-deletes a document.
func (uc *DocumentUseCase) Delete(ctx context.Context, id int64) error {
	return uc.repo.Delete(ctx, id)
}

// Get retrieves a single document by id.
func (uc *DocumentUseCase) Get(ctx context.Context, id int64) (*dto.DocumentResponse, error) {
	d, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if d == nil {
		return nil, errno.New(errno.NotFound, "document not found")
	}
	return toDocumentResponse(d), nil
}

// List returns a paginated list of documents.
func (uc *DocumentUseCase) List(ctx context.Context, p pagination.Params) (dto.ListResponse[dto.DocumentResponse], error) {
	items, total, err := uc.repo.List(ctx, p)
	if err != nil {
		return dto.ListResponse[dto.DocumentResponse]{}, err
	}
	resp := make([]dto.DocumentResponse, 0, len(items))
	for i := range items {
		resp = append(resp, *toDocumentResponse(&items[i]))
	}
	return dto.NewListResponse(p, total, resp), nil
}

// ListByClassic filters documents by classic_code.
func (uc *DocumentUseCase) ListByClassic(ctx context.Context, classicCode string, p pagination.Params) (dto.ListResponse[dto.DocumentResponse], error) {
	if classicCode == "" {
		return uc.List(ctx, p)
	}
	items, total, err := uc.repo.ListByClassic(ctx, classicCode, p)
	if err != nil {
		return dto.ListResponse[dto.DocumentResponse]{}, err
	}
	resp := make([]dto.DocumentResponse, 0, len(items))
	for i := range items {
		resp = append(resp, *toDocumentResponse(&items[i]))
	}
	return dto.NewListResponse(p, total, resp), nil
}

// toDocumentResponse maps the entity to its wire DTO.
func toDocumentResponse(d *entity.Document) *dto.DocumentResponse {
	if d == nil {
		return nil
	}
	resp := &dto.DocumentResponse{
		ID:                d.ID,
		ClassicCode:       d.ClassicCode,
		Title:             d.Title,
		Version:           d.Version,
		Dynasty:           d.Dynasty,
		School:            d.School,
		Author:            d.Author,
		SourceType:        d.SourceType,
		SourceRef:         d.SourceRef,
		FileURL:           d.FileURL,
		PDFObjectKey:      d.PDFObjectKey,
		MarkdownObjectKey: d.MarkdownObjectKey,
		MimeType:          d.MimeType,
		ContentHash:       d.ContentHash,
		Status:            d.Status,
		ChunkCount:        d.ChunkCount,
		VolumeCount:       d.VolumeCount,
		ClauseCount:       d.ClauseCount,
		MetadataJSON:      d.MetadataJSON,
	}
	if !d.CreatedAt.IsZero() {
		resp.CreatedAt = d.CreatedAt.Format(time.RFC3339)
	}
	if !d.UpdatedAt.IsZero() {
		resp.UpdatedAt = d.UpdatedAt.Format(time.RFC3339)
	}
	return resp
}

package usecase

import (
	"context"
	"strconv"

	"tcm-history-ai/backend/knowledge-service/internal/application/dto"
	"tcm-history-ai/backend/knowledge-service/internal/domain/entity"
	"tcm-history-ai/backend/knowledge-service/internal/domain/repository"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/idgen"
	"tcm-history-ai/backend/pkg/pagination"
)

// ChunkUseCase implements operations on document chunks.
type ChunkUseCase struct {
	chunkRepo repository.DocumentChunkRepository
	docRepo   repository.DocumentRepository
}

// NewChunkUseCase constructs a ChunkUseCase.
func NewChunkUseCase(chunkRepo repository.DocumentChunkRepository, docRepo repository.DocumentRepository) *ChunkUseCase {
	return &ChunkUseCase{chunkRepo: chunkRepo, docRepo: docRepo}
}

// ListByDocument returns paginated chunks for a document.
func (uc *ChunkUseCase) ListByDocument(ctx context.Context, documentID int64, p pagination.Params) (dto.ListResponse[dto.ChunkResponse], error) {
	items, total, err := uc.chunkRepo.ListByDocument(ctx, documentID, p)
	if err != nil {
		return dto.ListResponse[dto.ChunkResponse]{}, err
	}
	resp := make([]dto.ChunkResponse, 0, len(items))
	for i := range items {
		resp = append(resp, *toChunkResponse(&items[i]))
	}
	return dto.NewListResponse(p, total, resp), nil
}

// Get retrieves a single chunk by id.
func (uc *ChunkUseCase) Get(ctx context.Context, id int64) (*dto.ChunkResponse, error) {
	c, err := uc.chunkRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, errno.New(errno.NotFound, "chunk not found: "+strconv.FormatInt(id, 10))
	}
	return toChunkResponse(c), nil
}

// Create persists a single chunk. Mainly used for manual chunk insertion
// during testing / seeding.
func (uc *ChunkUseCase) Create(ctx context.Context, documentID int64, in *dto.ChunkResponse) (*dto.ChunkResponse, error) {
	if in == nil {
		return nil, errno.New(errno.InvalidParams, "body is required")
	}
	if in.Content == "" {
		return nil, errno.New(errno.InvalidParams, "content is required")
	}
	c := &entity.DocumentChunk{
		DocumentID:      documentID,
		ChunkID:         in.ChunkID,
		ChunkIndex:      in.ChunkIndex,
		ClassicCode:     in.ClassicCode,
		Volume:          in.Volume,
		ClauseNo:        in.ClauseNo,
		ContentType:     in.ContentType,
		Content:         in.Content,
		TextOriginal:    in.TextOriginal,
		TextTranslation: in.TextTranslation,
		TokenCount:      in.TokenCount,
		EmbeddingModel:  in.EmbeddingModel,
	}
	c.ID = idgen.Next()
	if c.ChunkID == "" {
		c.ChunkID = strconv.FormatInt(c.ID, 10)
	}
	if c.ContentType == "" {
		c.ContentType = entity.ContentOriginal
	}
	if err := uc.chunkRepo.Create(ctx, c); err != nil {
		return nil, err
	}
	return toChunkResponse(c), nil
}

// toChunkResponse maps the entity to its wire DTO.
func toChunkResponse(c *entity.DocumentChunk) *dto.ChunkResponse {
	if c == nil {
		return nil
	}
	return &dto.ChunkResponse{
		ID:              c.ID,
		DocumentID:      c.DocumentID,
		ChunkID:         c.ChunkID,
		ChunkIndex:      c.ChunkIndex,
		ClassicCode:     c.ClassicCode,
		Volume:          c.Volume,
		ClauseNo:        c.ClauseNo,
		ContentType:     c.ContentType,
		Content:         c.Content,
		TextOriginal:    c.TextOriginal,
		TextTranslation: c.TextTranslation,
		TokenCount:      c.TokenCount,
		EmbeddingID:     c.EmbeddingID,
		EmbeddingModel:  c.EmbeddingModel,
	}
}

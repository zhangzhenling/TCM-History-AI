package usecase

import (
	"context"
	"strconv"
	"time"

	"tcm-history-ai/backend/knowledge-service/internal/application/dto"
	"tcm-history-ai/backend/knowledge-service/internal/domain/entity"
	"tcm-history-ai/backend/knowledge-service/internal/domain/repository"
	"tcm-history-ai/backend/knowledge-service/internal/domain/service"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/idgen"
)

// RetrievalUseCase implements RAG retrieval: vector + BM25 → RRF → rerank.
//
// 流程依据 doc/06-RAG设计.md §6.8：
//  1. 并行召回：向量 (Milvus) + BM25 (Meilisearch)，各 topK
//  2. RRF 融合 (k=60) → Top 30
//  3. 过滤（classic/dynasty/school/content_type）
//  4. Rerank (cross-encoder) → Top 5
//  5. 写入 rag_queries 日志
type RetrievalUseCase struct {
	chunkRepo  repository.DocumentChunkRepository
	queryRepo  repository.RagQueryRepository
	vector     service.VectorStore
	fullText   service.FullTextSearcher
	embedder   service.EmbeddingProvider
	reranker   service.Reranker
	topK       int // 召回路每路返回数
	rrfK       int // RRF 平滑常数
	rerankTopK int // rerank 后返回数
}

// NewRetrievalUseCase constructs a RetrievalUseCase.
func NewRetrievalUseCase(
	chunkRepo repository.DocumentChunkRepository,
	queryRepo repository.RagQueryRepository,
	vector service.VectorStore,
	fullText service.FullTextSearcher,
	embedder service.EmbeddingProvider,
	reranker service.Reranker,
	topK, rrfK, rerankTopK int,
) *RetrievalUseCase {
	if topK <= 0 {
		topK = 20
	}
	if rrfK <= 0 {
		rrfK = 60
	}
	if rerankTopK <= 0 {
		rerankTopK = 5
	}
	return &RetrievalUseCase{
		chunkRepo:  chunkRepo,
		queryRepo:  queryRepo,
		vector:     vector,
		fullText:   fullText,
		embedder:   embedder,
		reranker:   reranker,
		topK:       topK,
		rrfK:       rrfK,
		rerankTopK: rerankTopK,
	}
}

// Retrieve runs the full RAG retrieval pipeline and persists the query log.
func (uc *RetrievalUseCase) Retrieve(ctx context.Context, in *dto.RetrieveRequest, userID int64) (*dto.RetrieveResponse, error) {
	if in == nil || in.Query == "" {
		return nil, errno.New(errno.InvalidParams, "query is required")
	}
	start := time.Now()
	topK := in.TopK
	if topK <= 0 {
		topK = uc.rerankTopK
	}
	filters := service.SearchFilter{
		ClassicCodes: in.ClassicCodes,
		Dynasties:    in.Dynasties,
		Schools:      in.Schools,
		ContentTypes: in.ContentTypes,
	}

	// 1. Embed query
	qVec, err := uc.embedder.Embed(ctx, []string{in.Query})
	if err != nil {
		return nil, errno.Wrap(errno.DependencyUnavailable, "embed query", err)
	}
	var queryVec []float32
	if len(qVec) > 0 {
		queryVec = qVec[0]
	}

	// 2. 并行召回：vector + BM25
	type vectorHit struct {
		chunkID string
		score   float32
		docID   int64
		rank    int
	}
	type bm25Hit struct {
		chunkID string
		score   float64
		docID   int64
		rank    int
	}
	vectorHits := make([]vectorHit, 0, uc.topK)
	bm25Hits := make([]bm25Hit, 0, uc.topK)
	var vecErr, bm25Err error

	// 在 goroutine 中并行执行两路召回
	done := make(chan struct{}, 2)
	go func() {
		defer func() { done <- struct{}{} }()
		results, err := uc.vector.Search(ctx, queryVec, uc.topK, filters)
		if err != nil {
			vecErr = err
			return
		}
		for i, r := range results {
			vectorHits = append(vectorHits, vectorHit{
				chunkID: r.ChunkID,
				score:   r.Score,
				docID:   r.DocID,
				rank:    i + 1,
			})
		}
	}()
	go func() {
		defer func() { done <- struct{}{} }()
		results, err := uc.fullText.Search(ctx, in.Query, uc.topK, filters)
		if err != nil {
			bm25Err = err
			return
		}
		for i, r := range results {
			bm25Hits = append(bm25Hits, bm25Hit{
				chunkID: r.ChunkID,
				score:   r.Score,
				docID:   r.DocID,
				rank:    i + 1,
			})
		}
	}()
	<-done
	<-done

	// 任一路召回失败都视为依赖不可用，但记录已有结果
	if vecErr != nil && bm25Err != nil {
		return nil, errno.Wrap(errno.DependencyUnavailable, "both retrieval paths failed", vecErr)
	}

	// 3. RRF 融合
	rrfScores := make(map[string]float64)
	docIDs := make(map[string]int64)
	for _, h := range vectorHits {
		rrfScores[h.chunkID] += 1.0 / float64(uc.rrfK+h.rank)
		docIDs[h.chunkID] = h.docID
	}
	for _, h := range bm25Hits {
		rrfScores[h.chunkID] += 1.0 / float64(uc.rrfK+h.rank)
		if _, ok := docIDs[h.chunkID]; !ok {
			docIDs[h.chunkID] = h.docID
		}
	}

	// 按 RRF 分数降序取 Top (rerankTopK * 2) 作为 rerank 候选
	candidates := make([]service.RerankCandidate, 0, len(rrfScores))
	for chunkID, score := range rrfScores {
		candidates = append(candidates, service.RerankCandidate{
			ChunkID: chunkID,
			DocID:   docIDs[chunkID],
		})
		_ = score // RRF 分数排序后用于决定 rerank 候选数量
	}
	// 按分数降序排序候选
	sortCandidatesByRRF(candidates, rrfScores)
	if len(candidates) > uc.rerankTopK*4 {
		candidates = candidates[:uc.rerankTopK*4]
	}

	// 4. 拉取 chunk 文本（rerank 需要）
	chunkTexts := make(map[string]string, len(candidates))
	if len(candidates) > 0 {
		chunkIDs := make([]string, 0, len(candidates))
		for _, c := range candidates {
			chunkIDs = append(chunkIDs, c.ChunkID)
		}
		// 通过 chunk_id 逐条查询（仓储接口未提供 BatchByChunkIDs，用现有 FindByChunkID）
		for _, cid := range chunkIDs {
			ch, err := uc.chunkRepo.FindByChunkID(ctx, cid)
			if err != nil || ch == nil {
				continue
			}
			text := ch.Content
			if ch.TextOriginal != "" {
				text = ch.TextOriginal
			}
			if ch.TextTranslation != "" {
				text += "\n" + ch.TextTranslation
			}
			chunkTexts[cid] = text
		}
	}
	for i := range candidates {
		candidates[i].Text = chunkTexts[candidates[i].ChunkID]
	}

	// 5. Rerank → TopK
	reranked, err := uc.reranker.Rerank(ctx, in.Query, candidates, topK)
	if err != nil {
		// rerank 失败回退到 RRF 排序
		reranked = candidates
		if len(reranked) > topK {
			reranked = reranked[:topK]
		}
	}

	// 6. 拉取完整 chunk 数据组装响应
	chunks := make([]dto.RetrievedChunk, 0, len(reranked))
	retrievedIDs := make([]int64, 0, len(reranked))
	for i, rc := range reranked {
		ch, err := uc.chunkRepo.FindByChunkID(ctx, rc.ChunkID)
		if err != nil || ch == nil {
			continue
		}
		var score float32
		if i < len(vectorHits) {
			_ = vectorHits
		}
		// 用 RRF 分数作为最终 score
		score = float32(rrfScores[rc.ChunkID])
		source := "rerank"
		if len(reranked) == 0 {
			source = "rrf"
		}
		chunks = append(chunks, dto.RetrievedChunk{
			ChunkID:         ch.ChunkID,
			DocumentID:      ch.DocumentID,
			ClassicCode:     ch.ClassicCode,
			Volume:          ch.Volume,
			ClauseNo:        ch.ClauseNo,
			ContentType:     ch.ContentType,
			Content:         ch.Content,
			TextOriginal:    ch.TextOriginal,
			TextTranslation: ch.TextTranslation,
			Score:           score,
			Source:          source,
		})
		retrievedIDs = append(retrievedIDs, ch.ID)
	}

	latencyMs := int(time.Since(start).Milliseconds())

	// 7. 写入 rag_queries 日志
	logID := idgen.Next()
	retrievedJSON := encodeRetrievedIDs(retrievedIDs)
	q := &entity.RagQuery{
		ID:                logID,
		SessionID:         in.SessionID,
		UserID:            userID,
		QueryText:         in.Query,
		TopK:              topK,
		RetrievedChunkIDs: retrievedJSON,
		LatencyMs:         latencyMs,
	}
	_ = uc.queryRepo.Create(ctx, q)

	return &dto.RetrieveResponse{
		Query:      in.Query,
		TopK:       topK,
		LatencyMs:  latencyMs,
		Total:      len(chunks),
		Chunks:     chunks,
		QueryLogID: logID,
	}, nil
}

// Feedback records user feedback (good/bad) for a RAG query.
func (uc *RetrievalUseCase) Feedback(ctx context.Context, queryLogID int64, feedback string) error {
	if feedback != entity.FeedbackGood && feedback != entity.FeedbackBad {
		return errno.New(errno.InvalidParams, "feedback must be good or bad")
	}
	q, err := uc.queryRepo.FindByID(ctx, queryLogID)
	if err != nil {
		return err
	}
	if q == nil {
		return errno.New(errno.NotFound, "rag query not found: "+strconv.FormatInt(queryLogID, 10))
	}
	q.Feedback = feedback
	return uc.queryRepo.Update(ctx, q)
}

// sortCandidatesByRRF sorts rerank candidates by their RRF score descending.
func sortCandidatesByRRF(candidates []service.RerankCandidate, scores map[string]float64) {
	for i := 1; i < len(candidates); i++ {
		for j := i; j > 0 && scores[candidates[j].ChunkID] > scores[candidates[j-1].ChunkID]; j-- {
			candidates[j], candidates[j-1] = candidates[j-1], candidates[j]
		}
	}
}

// encodeRetrievedIDs serialises a slice of int64 ids to a JSON array.
func encodeRetrievedIDs(ids []int64) []byte {
	if len(ids) == 0 {
		return []byte("[]")
	}
	out := []byte("[")
	for i, id := range ids {
		if i > 0 {
			out = append(out, ',')
		}
		out = append(out, []byte(strconv.FormatInt(id, 10))...)
	}
	out = append(out, ']')
	return out
}

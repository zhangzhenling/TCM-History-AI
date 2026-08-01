package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tcm-history-ai/backend/knowledge-service/internal/application/usecase"
	"tcm-history-ai/backend/knowledge-service/internal/domain/entity"
	"tcm-history-ai/backend/knowledge-service/internal/domain/event"
	"tcm-history-ai/backend/knowledge-service/internal/domain/service"
	"tcm-history-ai/backend/pkg/errno"
)

// --- trackingVectorStore wraps the existing mockVectorStore to record
// inserted records for verification ---

type trackingVectorStore struct {
	mockVectorStore
	inserted []service.VectorRecord
}

func (t *trackingVectorStore) Insert(ctx context.Context, records []service.VectorRecord) error {
	if t.insertErr != nil {
		return t.insertErr
	}
	t.inserted = append(t.inserted, records...)
	return nil
}

// --- dynamicEmbedder wraps the existing mockEmbedder to generate
// embeddings dynamically based on input texts ---

type dynamicEmbedder struct {
	mockEmbedder
	embedCount int
}

func (d *dynamicEmbedder) Embed(_ context.Context, texts []string) ([][]float32, error) {
	if d.embedErr != nil {
		return nil, d.embedErr
	}
	count := len(texts)
	if d.embedCount > 0 {
		count = d.embedCount
	}
	result := make([][]float32, count)
	for i := 0; i < count; i++ {
		v := make([]float32, d.dimInt)
		v[0] = float32(i + 1)
		result[i] = v
	}
	return result, nil
}

func (d *dynamicEmbedder) Model() string { return d.modelStr }
func (d *dynamicEmbedder) Dim() int      { return d.dimInt }

// --- mock DocumentRepository with FindByID error support ---

type mockDocRepoWithFindErr struct {
	mockDocumentRepo
	findErr error
}

func (m *mockDocRepoWithFindErr) FindByID(ctx context.Context, id int64) (*entity.Document, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	return m.mockDocumentRepo.FindByID(ctx, id)
}

// --- helpers ---

type ingestTestDeps struct {
	uc       *usecase.IngestUseCase
	docRepo  *mockDocumentRepo
	chunkRepo *mockChunkRepo
	taskRepo  *mockTaskRepo
	vector   *trackingVectorStore
	embedder *dynamicEmbedder
	pub      *mockEventPublisher
}

func setupIngestUseCase() *ingestTestDeps {
	docRepo := newMockDocumentRepo()
	chunkRepo := newMockChunkRepo()
	taskRepo := newMockTaskRepo()
	vector := &trackingVectorStore{}
	embedder := &dynamicEmbedder{mockEmbedder: mockEmbedder{modelStr: "bge-large-zh-v1.5", dimInt: 1024}}
	pub := &mockEventPublisher{}

	uc := usecase.NewIngestUseCase(docRepo, chunkRepo, taskRepo, vector, embedder, nil, pub)
	return &ingestTestDeps{
		uc: uc, docRepo: docRepo, chunkRepo: chunkRepo,
		taskRepo: taskRepo, vector: vector, embedder: embedder, pub: pub,
	}
}

func seedDocument(docRepo *mockDocumentRepo, id int64, status string) *entity.Document {
	doc := &entity.Document{}
	doc.ID = id
	doc.ClassicCode = "HuangDiNeiJing"
	doc.Title = "黄帝内经"
	doc.Version = "v1"
	doc.Dynasty = "Han"
	doc.School = "Traditional"
	doc.Status = status
	doc.SourceType = entity.SourceBook
	docRepo.items[id] = doc
	return doc
}

// --- tests ---

func TestIngestUseCase_HappyPath(t *testing.T) {
	deps := setupIngestUseCase()
	seedDocument(deps.docRepo, 1001, entity.DocumentStatusPending)

	err := deps.uc.IngestMarkdown(context.Background(), 1001, "上古天真论曰：昔在黄帝，生而神灵，弱而能言，幼而徇齐，长而敦敏，成而登天。")
	require.NoError(t, err)

	doc := deps.docRepo.items[1001]
	require.NotNil(t, doc)
	assert.Equal(t, entity.DocumentStatusEmbedded, doc.Status)
	assert.Positive(t, doc.ChunkCount)

	require.Len(t, deps.taskRepo.items, 1)
	for _, task := range deps.taskRepo.items {
		assert.Equal(t, entity.TaskStatusDone, task.Status)
		assert.Equal(t, int64(1001), task.DocumentID)
		assert.Equal(t, entity.StageMilvus, task.Stage)
		assert.Equal(t, 100, task.Progress)
		assert.Positive(t, task.VectorCount)
		assert.NotNil(t, task.StartedAt)
		assert.NotNil(t, task.FinishedAt)
	}

	assert.NotEmpty(t, deps.chunkRepo.items)
	for _, c := range deps.chunkRepo.items {
		assert.Equal(t, int64(1001), c.DocumentID)
		assert.NotEmpty(t, c.ChunkID)
		assert.NotEmpty(t, c.Content)
		assert.NotEmpty(t, c.EmbeddingID)
		assert.NotEmpty(t, c.EmbeddingModel)
	}

	assert.NotEmpty(t, deps.vector.inserted)
	for _, r := range deps.vector.inserted {
		assert.NotEmpty(t, r.ChunkID)
		assert.NotEmpty(t, r.Embedding)
		assert.Equal(t, int64(1001), r.DocID)
	}

	require.Len(t, deps.pub.published, 2)
	chunkedEvent, ok := deps.pub.published[0].(event.DocumentChunked)
	require.True(t, ok)
	assert.Equal(t, int64(1001), chunkedEvent.DocumentID)
	assert.Positive(t, chunkedEvent.ChunkCount)

	embeddedEvent, ok := deps.pub.published[1].(event.DocumentEmbedded)
	require.True(t, ok)
	assert.Equal(t, int64(1001), embeddedEvent.DocumentID)
	assert.Positive(t, embeddedEvent.VectorCount)
}

func TestIngestUseCase_DocumentNotFound(t *testing.T) {
	deps := setupIngestUseCase()

	err := deps.uc.IngestMarkdown(context.Background(), 9999, "some text")
	requireError(t, err, errno.NotFound)
}

func TestIngestUseCase_EmptyMarkdownText(t *testing.T) {
	deps := setupIngestUseCase()
	seedDocument(deps.docRepo, 1002, entity.DocumentStatusPending)

	err := deps.uc.IngestMarkdown(context.Background(), 1002, "")
	requireError(t, err, errno.InvalidParams)
}

func TestIngestUseCase_ChunkingFailsZeroChunks(t *testing.T) {
	deps := setupIngestUseCase()
	seedDocument(deps.docRepo, 1003, entity.DocumentStatusPending)

	err := deps.uc.IngestMarkdown(context.Background(), 1003, "   ")
	require.Error(t, err)

	doc := deps.docRepo.items[1003]
	assert.Equal(t, entity.DocumentStatusFailed, doc.Status)

	for _, task := range deps.taskRepo.items {
		assert.Equal(t, entity.TaskStatusFailed, task.Status)
		assert.NotEmpty(t, task.ErrorMessage)
	}
}

func TestIngestUseCase_BatchCreateChunksFails(t *testing.T) {
	deps := setupIngestUseCase()
	seedDocument(deps.docRepo, 1004, entity.DocumentStatusPending)
	deps.chunkRepo.batchCreateErr = errors.New("db write error")

	err := deps.uc.IngestMarkdown(context.Background(), 1004, "一段足够长的文本内容，用来产生至少一个 chunk 以供测试。")
	require.Error(t, err)

	doc := deps.docRepo.items[1004]
	assert.Equal(t, entity.DocumentStatusFailed, doc.Status)

	for _, task := range deps.taskRepo.items {
		assert.Equal(t, entity.TaskStatusFailed, task.Status)
	}
}

func TestIngestUseCase_EmbeddingFails(t *testing.T) {
	deps := setupIngestUseCase()
	seedDocument(deps.docRepo, 1005, entity.DocumentStatusPending)
	deps.embedder.embedErr = errors.New("embedding service unavailable")

	err := deps.uc.IngestMarkdown(context.Background(), 1005, "黄帝内经曰：上古之人，其知道者，法于阴阳，和于术数。")
	require.Error(t, err)

	doc := deps.docRepo.items[1005]
	assert.Equal(t, entity.DocumentStatusFailed, doc.Status)

	for _, task := range deps.taskRepo.items {
		assert.Equal(t, entity.TaskStatusFailed, task.Status)
		assert.Contains(t, task.ErrorMessage, "embedding service unavailable")
	}
}

func TestIngestUseCase_EmbeddingCountMismatch(t *testing.T) {
	deps := setupIngestUseCase()
	seedDocument(deps.docRepo, 1006, entity.DocumentStatusPending)
	deps.embedder.embedCount = 1

	longText := "上古天真论曰：昔在黄帝，生而神灵，弱而能言，幼而徇齐，长而敦敏，成而登天。" +
		"乃问于天师曰：余闻上古之人，春秋皆度百岁，而动作不衰；今时之人，年半百而动作皆衰者，时世异耶？人将失之耶？" +
		"岐伯对曰：上古之人，其知道者，法于阴阳，和于术数，食饮有节，起居有常，不妄作劳，故能形与神俱，而尽终其天年，度百岁乃去。" +
		"今时之人不然也，以酒为浆，以妄为常，醉以入房，以欲竭其精，以耗散其真，不知持满，不时御神，务快其心，逆于生乐，起居无节，故半百而衰也。" +
		"夫上古圣人之教下也，皆谓之虚邪贼风，避之有时，恬淡虚无，真气从之，精神内守，病安从来。" +
		"是以志闲而少欲，心安而不惧，形劳而不倦，气从以顺，各从其欲，皆得所愿。" +
		"故美其食，任其服，乐其俗，高下不相慕，其民故曰朴。" +
		"是以嗜欲不能劳其目，淫邪不能惑其心，愚智贤不肖不惧于物，故合于道。" +
		"所以能年皆度百岁而动作不衰者，以其德全不危也。" +
		"黄帝曰：余闻上古有真人者，提挈天地，把握阴阳，呼吸精气，独立守神，肌肉若一，故能寿敝天地，无有终时，此其道生。" +
		"中古之时，有至人者，淳德全道，和于阴阳，调于四时，去世离俗，积精全神，游行天地之间，视听八达之外，此盖益其寿命而强者也，亦归于真人。" +
		"其次有圣人者，处天地之和，从八风之理，适嗜欲于世俗之间，无恚嗔之心，行不欲离于世，被服章，举不欲观于俗，外不劳形于事，内无思想之患，以恬愉为务，以自得为功，形体不敝，精神不散，亦可以百数。" +
		"其次有贤人者，法则天地，象似日月，辨列星辰，逆从阴阳，分别四时，将从上古合同于道，亦可使益寿而有极时。"

	err := deps.uc.IngestMarkdown(context.Background(), 1006, longText)
	require.Error(t, err)

	doc := deps.docRepo.items[1006]
	assert.Equal(t, entity.DocumentStatusFailed, doc.Status)

	for _, task := range deps.taskRepo.items {
		assert.Equal(t, entity.TaskStatusFailed, task.Status)
	}
}

func TestIngestUseCase_VectorInsertFails(t *testing.T) {
	deps := setupIngestUseCase()
	seedDocument(deps.docRepo, 1007, entity.DocumentStatusPending)
	deps.vector.insertErr = errors.New("milvus connection refused")

	err := deps.uc.IngestMarkdown(context.Background(), 1007, "黄帝内经相关文本内容，足够产生多个分块。")
	require.Error(t, err)

	doc := deps.docRepo.items[1007]
	assert.Equal(t, entity.DocumentStatusFailed, doc.Status)

	for _, task := range deps.taskRepo.items {
		assert.Equal(t, entity.TaskStatusFailed, task.Status)
		assert.Contains(t, task.ErrorMessage, "milvus connection refused")
	}
}

func TestIngestUseCase_DocumentFindByIDError(t *testing.T) {
	docRepo := &mockDocRepoWithFindErr{}
	docRepo.findErr = errors.New("database error")
	chunkRepo := newMockChunkRepo()
	taskRepo := newMockTaskRepo()
	vector := &trackingVectorStore{}
	embedder := &dynamicEmbedder{mockEmbedder: mockEmbedder{modelStr: "bge-large-zh-v1.5", dimInt: 1024}}
	pub := &mockEventPublisher{}

	uc := usecase.NewIngestUseCase(docRepo, chunkRepo, taskRepo, vector, embedder, nil, pub)

	err := uc.IngestMarkdown(context.Background(), 1008, "test text")
	require.Error(t, err)
	requireError(t, err, errno.InternalError)
}

func TestIngestUseCase_UpdateDocumentStatusFails(t *testing.T) {
	deps := setupIngestUseCase()
	seedDocument(deps.docRepo, 1009, entity.DocumentStatusPending)

	deps.docRepo.updateErr = errors.New("update failed")

	err := deps.uc.IngestMarkdown(context.Background(), 1009, "测试文本内容，黄帝内经相关。")
	require.Error(t, err)

	for _, task := range deps.taskRepo.items {
		assert.Equal(t, entity.TaskStatusFailed, task.Status)
	}
}

func TestIngestUseCase_AlreadyProcessedDocument(t *testing.T) {
	deps := setupIngestUseCase()
	seedDocument(deps.docRepo, 1010, entity.DocumentStatusEmbedded)

	err := deps.uc.IngestMarkdown(context.Background(), 1010, "重新处理的文本内容。")
	require.NoError(t, err)

	doc := deps.docRepo.items[1010]
	assert.Equal(t, entity.DocumentStatusEmbedded, doc.Status)
	assert.Positive(t, doc.ChunkCount)

	assert.NotEmpty(t, deps.chunkRepo.items)
	assert.NotEmpty(t, deps.vector.inserted)
	assert.Len(t, deps.pub.published, 2)

	for _, task := range deps.taskRepo.items {
		assert.Equal(t, entity.TaskStatusDone, task.Status)
	}
}

func TestIngestUseCase_TaskCreatedWithCorrectFields(t *testing.T) {
	deps := setupIngestUseCase()
	seedDocument(deps.docRepo, 1011, entity.DocumentStatusPending)

	err := deps.uc.IngestMarkdown(context.Background(), 1011, "测试文本。")
	require.NoError(t, err)

	require.Len(t, deps.taskRepo.items, 1)
	var task *entity.EmbeddingTask
	for _, t := range deps.taskRepo.items {
		task = t
	}
	require.NotNil(t, task)
	assert.Equal(t, int64(1011), task.DocumentID)
	assert.Equal(t, entity.TaskTypeDocument, task.TaskType)
	assert.Equal(t, entity.StageMilvus, task.Stage)
	assert.Equal(t, entity.TaskStatusDone, task.Status)
	assert.Equal(t, deps.embedder.Model(), task.Model)
	assert.NotNil(t, task.StartedAt)
	assert.NotNil(t, task.FinishedAt)
	assert.True(t, task.FinishedAt.After(*task.StartedAt) || task.FinishedAt.Equal(*task.StartedAt))
}

func TestIngestUseCase_SingleChunkDocument(t *testing.T) {
	deps := setupIngestUseCase()
	seedDocument(deps.docRepo, 1012, entity.DocumentStatusPending)

	err := deps.uc.IngestMarkdown(context.Background(), 1012, "这是一段很短的测试文本，只有一个分块。")
	require.NoError(t, err)

	assert.Len(t, deps.chunkRepo.items, 1)
	for _, c := range deps.chunkRepo.items {
		assert.Equal(t, int64(1012), c.DocumentID)
		assert.Equal(t, 0, c.ChunkIndex)
		assert.NotEmpty(t, c.Content)
	}

	require.Len(t, deps.pub.published, 2)
	chunkedEvent, ok := deps.pub.published[0].(event.DocumentChunked)
	require.True(t, ok)
	assert.Equal(t, 1, chunkedEvent.ChunkCount)
}

func TestIngestUseCase_CreateTaskFails(t *testing.T) {
	deps := setupIngestUseCase()
	seedDocument(deps.docRepo, 1013, entity.DocumentStatusPending)

	deps.taskRepo.createErr = errors.New("task table down")

	err := deps.uc.IngestMarkdown(context.Background(), 1013, "测试文本内容。")
	require.Error(t, err)
	requireError(t, err, errno.InternalError)
}

func TestIngestUseCase_FullPipeline_VerifiesAllIntermediateSteps(t *testing.T) {
	deps := setupIngestUseCase()
	seedDocument(deps.docRepo, 1014, entity.DocumentStatusPending)

	longText := "黄帝内经·上古天真论。\n\n" +
		"昔在黄帝，生而神灵，弱而能言，幼而徇齐，长而敦敏，成而登天。\n\n" +
		"乃问于天师曰：余闻上古之人，春秋皆度百岁，而动作不衰；" +
		"今时之人，年半百而动作皆衰者，时世异耶？人将失之耶？\n\n" +
		"岐伯对曰：上古之人，其知道者，法于阴阳，和于术数，" +
		"食饮有节，起居有常，不妄作劳，故能形与神俱，而尽终其天年，度百岁乃去。"

	err := deps.uc.IngestMarkdown(context.Background(), 1014, longText)
	require.NoError(t, err)

	doc := deps.docRepo.items[1014]
	require.NotNil(t, doc)
	assert.Equal(t, entity.DocumentStatusEmbedded, doc.Status)
	assert.Positive(t, doc.ChunkCount)

	assert.NotEmpty(t, deps.chunkRepo.items)
	for _, c := range deps.chunkRepo.items {
		assert.NotEmpty(t, c.EmbeddingID)
		assert.NotEmpty(t, c.EmbeddingModel)
		assert.Equal(t, "bge-large-zh-v1.5", c.EmbeddingModel)
	}

	for _, task := range deps.taskRepo.items {
		assert.Equal(t, entity.TaskStatusDone, task.Status)
		assert.Equal(t, task.ChunkCount, doc.ChunkCount)
		assert.Equal(t, task.VectorCount, doc.ChunkCount)
	}
}
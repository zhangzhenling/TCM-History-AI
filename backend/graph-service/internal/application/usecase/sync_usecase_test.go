package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tcm-history-ai/backend/graph-service/internal/application/usecase"
	"tcm-history-ai/backend/graph-service/internal/domain/entity"
	"tcm-history-ai/backend/graph-service/internal/domain/event"
	"tcm-history-ai/backend/pkg/errno"
)

// --- mock GraphSyncLogRepository ---

type mockSyncLogRepo struct {
	items       map[int64]*entity.GraphSyncLog
	createErr   error
	updateErr   error
	listPendingErr error
	pendingOut  []entity.GraphSyncLog

	createCalls   int
	updateCalls   int
	listPendingCalls int
}

func newMockSyncLogRepo() *mockSyncLogRepo {
	return &mockSyncLogRepo{items: map[int64]*entity.GraphSyncLog{}}
}

func (m *mockSyncLogRepo) Create(_ context.Context, log *entity.GraphSyncLog) error {
	m.createCalls++
	if m.createErr != nil {
		return m.createErr
	}
	m.items[log.ID] = log
	return nil
}

func (m *mockSyncLogRepo) UpdateStatus(_ context.Context, id int64, status, errorMsg string) error {
	m.updateCalls++
	if m.updateErr != nil {
		return m.updateErr
	}
	if log, ok := m.items[id]; ok {
		log.Status = status
		log.ErrorMsg = errorMsg
	}
	return nil
}

func (m *mockSyncLogRepo) ListPending(_ context.Context, limit int) ([]entity.GraphSyncLog, error) {
	m.listPendingCalls++
	if m.listPendingErr != nil {
		return nil, m.listPendingErr
	}
	if m.pendingOut != nil {
		out := make([]entity.GraphSyncLog, 0, len(m.pendingOut))
		for _, l := range m.pendingOut {
			if limit > 0 && len(out) >= limit {
				break
			}
			out = append(out, l)
		}
		return out, nil
	}
	// Default: return all pending logs stored in items.
	out := make([]entity.GraphSyncLog, 0)
	for _, l := range m.items {
		if l.Status == entity.SyncStatusPending {
			out = append(out, *l)
		}
	}
	return out, nil
}

// syncLog is a small helper to build a GraphSyncLog with a stable ID without
// having to spell out the embedded BaseModel field in every struct literal.
func syncLog(id int64, action, sourceID, entityType string) entity.GraphSyncLog {
	l := entity.GraphSyncLog{
		SourceType: entity.SourceHistory,
		SourceID:   sourceID,
		EntityType: entityType,
		Action:     action,
		Status:     entity.SyncStatusPending,
	}
	l.ID = id
	return l
}

// --- tests ---

func TestSyncUseCase_HandleDocumentIndexed(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		store := &mockGraphStore{}
		logRepo := newMockSyncLogRepo()
		uc := usecase.NewSyncUseCase(store, logRepo)

		err := uc.HandleDocumentIndexed(context.Background(), event.DocumentIndexed{
			DocumentID:  42,
			ClassicCode: "HuangDiNeiJing",
			Title:       "黃帝內經",
			Dynasty:     "Han",
		})
		require.NoError(t, err)
		require.Equal(t, 1, logRepo.createCalls)
		require.Equal(t, 1, logRepo.updateCalls)
		require.Len(t, store.upsertNodeCalls, 1)
		assert.Equal(t, "classic:HuangDiNeiJing", store.upsertNodeCalls[0].UID)
		assert.Equal(t, entity.LabelClassic, store.upsertNodeCalls[0].Label)
		assert.Equal(t, "黃帝內經", store.upsertNodeCalls[0].Name)
	})

	t.Run("uses classic_code as name when title missing", func(t *testing.T) {
		store := &mockGraphStore{}
		logRepo := newMockSyncLogRepo()
		uc := usecase.NewSyncUseCase(store, logRepo)

		err := uc.HandleDocumentIndexed(context.Background(), event.DocumentIndexed{
			DocumentID:  1,
			ClassicCode: "Code1",
		})
		require.NoError(t, err)
		require.Len(t, store.upsertNodeCalls, 1)
		assert.Equal(t, "Code1", store.upsertNodeCalls[0].Name)
	})

	t.Run("missing document_id logs failure", func(t *testing.T) {
		logRepo := newMockSyncLogRepo()
		uc := usecase.NewSyncUseCase(&mockGraphStore{}, logRepo)

		err := uc.HandleDocumentIndexed(context.Background(), event.DocumentIndexed{
			DocumentID: 0,
		})
		require.Error(t, err)
		require.Equal(t, 1, logRepo.createCalls)
		// Verify the failure log entry was persisted.
		var failure *entity.GraphSyncLog
		for _, l := range logRepo.items {
			failure = l
		}
		require.NotNil(t, failure)
		assert.Equal(t, entity.SyncStatusFailed, failure.Status)
		assert.Contains(t, failure.ErrorMsg, "missing document_id")
	})

	t.Run("store error marks log failed", func(t *testing.T) {
		store := &mockGraphStore{upsertNodeErr: errors.New("neo4j down")}
		logRepo := newMockSyncLogRepo()
		uc := usecase.NewSyncUseCase(store, logRepo)

		err := uc.HandleDocumentIndexed(context.Background(), event.DocumentIndexed{
			DocumentID: 1, ClassicCode: "X",
		})
		require.Error(t, err)
		require.Equal(t, 1, logRepo.updateCalls)
		// Verify the log was marked failed.
		var log *entity.GraphSyncLog
		for _, l := range logRepo.items {
			if l.Status == entity.SyncStatusFailed {
				log = l
			}
		}
		require.NotNil(t, log)
		assert.Contains(t, log.ErrorMsg, "neo4j down")
	})

	t.Run("logRepo Create error", func(t *testing.T) {
		logRepo := newMockSyncLogRepo()
		logRepo.createErr = errors.New("db down")
		uc := usecase.NewSyncUseCase(&mockGraphStore{}, logRepo)
		err := uc.HandleDocumentIndexed(context.Background(), event.DocumentIndexed{
			DocumentID: 1, ClassicCode: "X",
		})
		require.Error(t, err)
	})

	t.Run("logRepo UpdateStatus error after store success", func(t *testing.T) {
		logRepo := newMockSyncLogRepo()
		logRepo.updateErr = errors.New("update fail")
		uc := usecase.NewSyncUseCase(&mockGraphStore{}, logRepo)
		err := uc.HandleDocumentIndexed(context.Background(), event.DocumentIndexed{
			DocumentID: 1, ClassicCode: "X",
		})
		require.Error(t, err)
	})
}

func TestSyncUseCase_HandleUserRegistered(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		logRepo := newMockSyncLogRepo()
		uc := usecase.NewSyncUseCase(&mockGraphStore{}, logRepo)

		err := uc.HandleUserRegistered(context.Background(), event.UserRegistered{
			UserID: 5, Username: "u", Nickname: "n",
		})
		require.NoError(t, err)
		require.Equal(t, 1, logRepo.createCalls)
		require.Equal(t, 1, logRepo.updateCalls)
	})

	t.Run("missing user_id logs failure", func(t *testing.T) {
		logRepo := newMockSyncLogRepo()
		uc := usecase.NewSyncUseCase(&mockGraphStore{}, logRepo)

		err := uc.HandleUserRegistered(context.Background(), event.UserRegistered{
			UserID: 0,
		})
		require.Error(t, err)
		require.Equal(t, 1, logRepo.createCalls)
		var failure *entity.GraphSyncLog
		for _, l := range logRepo.items {
			failure = l
		}
		require.NotNil(t, failure)
		assert.Equal(t, entity.SyncStatusFailed, failure.Status)
	})

	t.Run("logRepo Create error", func(t *testing.T) {
		logRepo := newMockSyncLogRepo()
		logRepo.createErr = errors.New("db down")
		uc := usecase.NewSyncUseCase(&mockGraphStore{}, logRepo)
		err := uc.HandleUserRegistered(context.Background(), event.UserRegistered{UserID: 1})
		require.Error(t, err)
	})

	t.Run("logRepo UpdateStatus error after success", func(t *testing.T) {
		logRepo := newMockSyncLogRepo()
		logRepo.updateErr = errors.New("update fail")
		uc := usecase.NewSyncUseCase(&mockGraphStore{}, logRepo)
		err := uc.HandleUserRegistered(context.Background(), event.UserRegistered{UserID: 1})
		require.Error(t, err)
	})
}

func TestSyncUseCase_HandleEntityCreated(t *testing.T) {
	t.Run("happy path upsert", func(t *testing.T) {
		store := &mockGraphStore{}
		logRepo := newMockSyncLogRepo()
		uc := usecase.NewSyncUseCase(store, logRepo)

		err := uc.HandleEntityCreated(context.Background(), event.EntityCreated{
			EntityType: "person",
			UID:        "person:1",
			Name:       "Zhang",
			Operation:  "created",
		})
		require.NoError(t, err)
		require.Len(t, store.upsertNodeCalls, 1)
		assert.Equal(t, "person:1", store.upsertNodeCalls[0].UID)
		assert.Equal(t, entity.LabelPerson, store.upsertNodeCalls[0].Label)
	})

	t.Run("happy path delete", func(t *testing.T) {
		store := &mockGraphStore{}
		logRepo := newMockSyncLogRepo()
		uc := usecase.NewSyncUseCase(store, logRepo)

		err := uc.HandleEntityCreated(context.Background(), event.EntityCreated{
			EntityType: "person",
			UID:        "person:1",
			Operation:  "deleted",
		})
		require.NoError(t, err)
		require.Len(t, store.deleteNodeCalls, 1)
		assert.Equal(t, "person:1", store.deleteNodeCalls[0])
	})

	t.Run("all known entity types map to labels", func(t *testing.T) {
		cases := []struct {
			entityType string
			wantLabel  string
		}{
			{"person", entity.LabelPerson},
			{"classic", entity.LabelClassic},
			{"school", entity.LabelSchool},
			{"prescription", entity.LabelPrescription},
			{"medicine", entity.LabelMedicine},
			{"disease", entity.LabelDisease},
			{"dynasty", entity.LabelDynasty},
			{"event", entity.LabelHistoricalEvent},
		}
		for _, tc := range cases {
			t.Run(tc.entityType, func(t *testing.T) {
				store := &mockGraphStore{}
				logRepo := newMockSyncLogRepo()
				uc := usecase.NewSyncUseCase(store, logRepo)
				err := uc.HandleEntityCreated(context.Background(), event.EntityCreated{
					EntityType: tc.entityType,
					UID:        "uid:" + tc.entityType,
					Name:       "name",
				})
				require.NoError(t, err)
				require.Len(t, store.upsertNodeCalls, 1)
				assert.Equal(t, tc.wantLabel, store.upsertNodeCalls[0].Label)
			})
		}
	})

	t.Run("missing uid logs failure", func(t *testing.T) {
		logRepo := newMockSyncLogRepo()
		uc := usecase.NewSyncUseCase(&mockGraphStore{}, logRepo)
		err := uc.HandleEntityCreated(context.Background(), event.EntityCreated{
			EntityType: "person",
		})
		require.Error(t, err)
		require.Equal(t, 1, logRepo.createCalls)
	})

	t.Run("unknown entity type logs failure", func(t *testing.T) {
		logRepo := newMockSyncLogRepo()
		uc := usecase.NewSyncUseCase(&mockGraphStore{}, logRepo)
		err := uc.HandleEntityCreated(context.Background(), event.EntityCreated{
			EntityType: "robot",
			UID:        "robot:1",
		})
		require.Error(t, err)
		var failure *entity.GraphSyncLog
		for _, l := range logRepo.items {
			failure = l
		}
		require.NotNil(t, failure)
		assert.Equal(t, entity.SyncStatusFailed, failure.Status)
		assert.Contains(t, failure.ErrorMsg, "unknown entity_type")
	})

	t.Run("store upsert error marks log failed", func(t *testing.T) {
		store := &mockGraphStore{upsertNodeErr: errors.New("neo4j down")}
		logRepo := newMockSyncLogRepo()
		uc := usecase.NewSyncUseCase(store, logRepo)
		err := uc.HandleEntityCreated(context.Background(), event.EntityCreated{
			EntityType: "person", UID: "p:1",
		})
		require.Error(t, err)
	})

	t.Run("store delete error marks log failed", func(t *testing.T) {
		store := &mockGraphStore{deleteNodeErr: errors.New("neo4j down")}
		logRepo := newMockSyncLogRepo()
		uc := usecase.NewSyncUseCase(store, logRepo)
		err := uc.HandleEntityCreated(context.Background(), event.EntityCreated{
			EntityType: "person", UID: "p:1", Operation: "deleted",
		})
		require.Error(t, err)
	})

	t.Run("logRepo Create error", func(t *testing.T) {
		logRepo := newMockSyncLogRepo()
		logRepo.createErr = errors.New("db down")
		uc := usecase.NewSyncUseCase(&mockGraphStore{}, logRepo)
		err := uc.HandleEntityCreated(context.Background(), event.EntityCreated{
			EntityType: "person", UID: "p:1",
		})
		require.Error(t, err)
	})
}

func TestSyncUseCase_Dispatch(t *testing.T) {
	t.Run("doc.indexed routes to HandleDocumentIndexed", func(t *testing.T) {
		logRepo := newMockSyncLogRepo()
		uc := usecase.NewSyncUseCase(&mockGraphStore{}, logRepo)
		body := []byte(`{"document_id":1,"classic_code":"X"}`)
		err := uc.Dispatch(context.Background(), "doc.indexed", body)
		require.NoError(t, err)
		require.Equal(t, 1, logRepo.createCalls)
	})

	t.Run("user.registered routes to HandleUserRegistered", func(t *testing.T) {
		logRepo := newMockSyncLogRepo()
		uc := usecase.NewSyncUseCase(&mockGraphStore{}, logRepo)
		body := []byte(`{"user_id":1}`)
		err := uc.Dispatch(context.Background(), "user.registered", body)
		require.NoError(t, err)
		require.Equal(t, 1, logRepo.createCalls)
	})

	t.Run("entity.created routes to HandleEntityCreated", func(t *testing.T) {
		logRepo := newMockSyncLogRepo()
		uc := usecase.NewSyncUseCase(&mockGraphStore{}, logRepo)
		body := []byte(`{"entity_type":"person","uid":"person:1"}`)
		err := uc.Dispatch(context.Background(), "entity.created", body)
		require.NoError(t, err)
		require.Equal(t, 1, logRepo.createCalls)
	})

	t.Run("unknown routing key returns nil (no-op)", func(t *testing.T) {
		logRepo := newMockSyncLogRepo()
		uc := usecase.NewSyncUseCase(&mockGraphStore{}, logRepo)
		err := uc.Dispatch(context.Background(), "totally.unknown", []byte(`{}`))
		require.NoError(t, err)
		assert.Equal(t, 0, logRepo.createCalls)
	})

	t.Run("doc.indexed with invalid JSON logs failure", func(t *testing.T) {
		logRepo := newMockSyncLogRepo()
		uc := usecase.NewSyncUseCase(&mockGraphStore{}, logRepo)
		err := uc.Dispatch(context.Background(), "doc.indexed", []byte(`{not json`))
		require.Error(t, err)
		require.Equal(t, 1, logRepo.createCalls)
	})

	t.Run("user.registered with invalid JSON logs failure", func(t *testing.T) {
		logRepo := newMockSyncLogRepo()
		uc := usecase.NewSyncUseCase(&mockGraphStore{}, logRepo)
		err := uc.Dispatch(context.Background(), "user.registered", []byte(`{not json`))
		require.Error(t, err)
		require.Equal(t, 1, logRepo.createCalls)
	})

	t.Run("entity.created with invalid JSON logs failure", func(t *testing.T) {
		logRepo := newMockSyncLogRepo()
		uc := usecase.NewSyncUseCase(&mockGraphStore{}, logRepo)
		err := uc.Dispatch(context.Background(), "entity.created", []byte(`{not json`))
		require.Error(t, err)
		require.Equal(t, 1, logRepo.createCalls)
	})
}

func TestSyncUseCase_TriggerSync(t *testing.T) {
	t.Run("default limit when zero", func(t *testing.T) {
		store := &mockGraphStore{}
		logRepo := newMockSyncLogRepo()
		logRepo.pendingOut = []entity.GraphSyncLog{
			syncLog(1, entity.ActionUpsert, "x:1", entity.LabelPerson),
		}
		uc := usecase.NewSyncUseCase(store, logRepo)
		succeeded, failed, err := uc.TriggerSync(context.Background(), 0)
		require.NoError(t, err)
		assert.Equal(t, 1, succeeded)
		assert.Equal(t, 0, failed)
		require.Equal(t, 1, logRepo.listPendingCalls)
	})

	t.Run("upsert path success", func(t *testing.T) {
		store := &mockGraphStore{}
		logRepo := newMockSyncLogRepo()
		logRepo.pendingOut = []entity.GraphSyncLog{
			syncLog(1, entity.ActionUpsert, "x:1", entity.LabelPerson),
			syncLog(2, entity.ActionUpsert, "x:2", entity.LabelClassic),
		}
		uc := usecase.NewSyncUseCase(store, logRepo)
		succeeded, failed, err := uc.TriggerSync(context.Background(), 10)
		require.NoError(t, err)
		assert.Equal(t, 2, succeeded)
		assert.Equal(t, 0, failed)
		require.Len(t, store.upsertNodeCalls, 2)
	})

	t.Run("delete path success", func(t *testing.T) {
		store := &mockGraphStore{}
		logRepo := newMockSyncLogRepo()
		logRepo.pendingOut = []entity.GraphSyncLog{
			syncLog(1, entity.ActionDelete, "x:1", entity.LabelPerson),
		}
		uc := usecase.NewSyncUseCase(store, logRepo)
		succeeded, failed, err := uc.TriggerSync(context.Background(), 10)
		require.NoError(t, err)
		assert.Equal(t, 1, succeeded)
		assert.Equal(t, 0, failed)
		require.Len(t, store.deleteNodeCalls, 1)
	})

	t.Run("upsert failure counts as failed", func(t *testing.T) {
		store := &mockGraphStore{upsertNodeErr: errors.New("boom")}
		logRepo := newMockSyncLogRepo()
		logRepo.pendingOut = []entity.GraphSyncLog{
			syncLog(1, entity.ActionUpsert, "x:1", entity.LabelPerson),
		}
		uc := usecase.NewSyncUseCase(store, logRepo)
		succeeded, failed, err := uc.TriggerSync(context.Background(), 10)
		require.NoError(t, err)
		assert.Equal(t, 0, succeeded)
		assert.Equal(t, 1, failed)
	})

	t.Run("delete failure counts as failed", func(t *testing.T) {
		store := &mockGraphStore{deleteNodeErr: errors.New("boom")}
		logRepo := newMockSyncLogRepo()
		logRepo.pendingOut = []entity.GraphSyncLog{
			syncLog(1, entity.ActionDelete, "x:1", entity.LabelPerson),
		}
		uc := usecase.NewSyncUseCase(store, logRepo)
		succeeded, failed, err := uc.TriggerSync(context.Background(), 10)
		require.NoError(t, err)
		assert.Equal(t, 0, succeeded)
		assert.Equal(t, 1, failed)
	})

	t.Run("empty pending returns zeros", func(t *testing.T) {
		logRepo := newMockSyncLogRepo()
		uc := usecase.NewSyncUseCase(&mockGraphStore{}, logRepo)
		succeeded, failed, err := uc.TriggerSync(context.Background(), 10)
		require.NoError(t, err)
		assert.Equal(t, 0, succeeded)
		assert.Equal(t, 0, failed)
	})

	t.Run("ListPending error", func(t *testing.T) {
		logRepo := newMockSyncLogRepo()
		logRepo.listPendingErr = errors.New("db down")
		uc := usecase.NewSyncUseCase(&mockGraphStore{}, logRepo)
		succeeded, failed, err := uc.TriggerSync(context.Background(), 10)
		require.Error(t, err)
		assert.Equal(t, 0, succeeded)
		assert.Equal(t, 0, failed)
	})
}

// TestSyncUseCase_logFailure_ErrnoIndependence ensures logFailure never
// returns an *errno.Error — callers (RabbitMQ subscriber) rely on plain
// errors to nack without poisoning typed-error handling.
func TestSyncUseCase_logFailure_ErrnoIndependence(t *testing.T) {
	logRepo := newMockSyncLogRepo()
	uc := usecase.NewSyncUseCase(&mockGraphStore{}, logRepo)
	// Trigger via HandleDocumentIndexed with missing DocumentID — exercises logFailure.
	err := uc.HandleDocumentIndexed(context.Background(), event.DocumentIndexed{})
	require.Error(t, err)
	var e *errno.Error
	assert.False(t, errors.As(err, &e), "logFailure should return plain error, not *errno.Error")
}

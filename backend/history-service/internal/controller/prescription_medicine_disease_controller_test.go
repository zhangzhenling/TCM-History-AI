package controller_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"tcm-history-ai/backend/history-service/internal/application/usecase"
	"tcm-history-ai/backend/history-service/internal/controller"
	"tcm-history-ai/backend/history-service/internal/domain/entity"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/pagination"
)

// ============================================================================
// PrescriptionController
// ============================================================================

func newPrescriptionController(repo *mockPrescriptionRepo) *controller.PrescriptionController {
	uc := usecase.NewPrescriptionUseCase(repo)
	return controller.NewPrescriptionController(uc)
}

func seedPrescription(repo *mockPrescriptionRepo, id int64, name string) *entity.Prescription {
	p := &entity.Prescription{Name: name}
	p.ID = id
	repo.items[id] = p
	return p
}

func TestPrescriptionController_List(t *testing.T) {
	t.Run("returns 200 with seeded prescriptions", func(t *testing.T) {
		repo := newMockPrescriptionRepo()
		seedPrescription(repo, 1, "Mahuang Tang")
		ctrl := newPrescriptionController(repo)

		rc := newRC()
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/history/prescriptions?page=1&page_size=10")

		ctrl.List(ctx(), rc)
		body := assertStatusCode(t, rc, http.StatusOK)
		assert.Equal(t, int(errno.OK), body.Code)
	})

	t.Run("returns 200 on empty result", func(t *testing.T) {
		repo := newMockPrescriptionRepo()
		ctrl := newPrescriptionController(repo)

		rc := newRC()
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/history/prescriptions")

		ctrl.List(ctx(), rc)
		assertStatusCode(t, rc, http.StatusOK)
	})

	t.Run("propagates repo error as 500", func(t *testing.T) {
		repo := newMockPrescriptionRepo()
		repo.list = func(_ pagination.Params) ([]entity.Prescription, int, error) {
			return nil, 0, errno.New(errno.InternalError, "db down")
		}
		ctrl := newPrescriptionController(repo)

		rc := newRC()
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/history/prescriptions")

		ctrl.List(ctx(), rc)
		assert.Equal(t, http.StatusInternalServerError, rc.Response.StatusCode())
	})
}

func TestPrescriptionController_Create(t *testing.T) {
	t.Run("happy path returns 201", func(t *testing.T) {
		repo := newMockPrescriptionRepo()
		ctrl := newPrescriptionController(repo)

		rc := newRC()
		rc.Request.SetMethod("POST")
		rc.Request.SetRequestURI("/api/v1/history/prescriptions")
		rc.Request.SetBody([]byte(`{"name":"Mahuang Tang","category":"exterior_releasing"}`))

		ctrl.Create(ctx(), rc)
		body := assertStatusCode(t, rc, http.StatusCreated)
		assert.Equal(t, "created", body.Message)
	})

	t.Run("invalid JSON returns 400", func(t *testing.T) {
		repo := newMockPrescriptionRepo()
		ctrl := newPrescriptionController(repo)

		rc := newRC()
		rc.Request.SetMethod("POST")
		rc.Request.SetRequestURI("/api/v1/history/prescriptions")
		rc.Request.SetBody([]byte(`{not-json`))

		ctrl.Create(ctx(), rc)
		assertStatusCode(t, rc, http.StatusBadRequest)
	})

	t.Run("empty name returns 400", func(t *testing.T) {
		repo := newMockPrescriptionRepo()
		ctrl := newPrescriptionController(repo)

		rc := newRC()
		rc.Request.SetMethod("POST")
		rc.Request.SetRequestURI("/api/v1/history/prescriptions")
		rc.Request.SetBody([]byte(`{"name":""}`))

		ctrl.Create(ctx(), rc)
		assertStatusCode(t, rc, http.StatusBadRequest)
	})
}

func TestPrescriptionController_Get(t *testing.T) {
	t.Run("found returns 200", func(t *testing.T) {
		repo := newMockPrescriptionRepo()
		seedPrescription(repo, 1, "Mahuang Tang")
		ctrl := newPrescriptionController(repo)

		rc := newRC()
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/history/prescriptions/1")
		setParam(rc, "id", "1")

		ctrl.Get(ctx(), rc)
		assertStatusCode(t, rc, http.StatusOK)
	})

	t.Run("invalid id returns 400", func(t *testing.T) {
		repo := newMockPrescriptionRepo()
		ctrl := newPrescriptionController(repo)

		rc := newRC()
		setParam(rc, "id", "abc")

		ctrl.Get(ctx(), rc)
		assertStatusCode(t, rc, http.StatusBadRequest)
	})

	t.Run("not found returns 404", func(t *testing.T) {
		repo := newMockPrescriptionRepo()
		ctrl := newPrescriptionController(repo)

		rc := newRC()
		setParam(rc, "id", "999")

		ctrl.Get(ctx(), rc)
		assertStatusCode(t, rc, http.StatusNotFound)
	})
}

func TestPrescriptionController_Update(t *testing.T) {
	t.Run("happy path returns 200", func(t *testing.T) {
		repo := newMockPrescriptionRepo()
		seedPrescription(repo, 1, "Mahuang Tang")
		ctrl := newPrescriptionController(repo)

		rc := newRC()
		rc.Request.SetMethod("PUT")
		rc.Request.SetRequestURI("/api/v1/history/prescriptions/1")
		setParam(rc, "id", "1")
		rc.Request.SetBody([]byte(`{"name":"Updated Prescription"}`))

		ctrl.Update(ctx(), rc)
		assertStatusCode(t, rc, http.StatusOK)
	})

	t.Run("invalid id returns 400", func(t *testing.T) {
		repo := newMockPrescriptionRepo()
		ctrl := newPrescriptionController(repo)

		rc := newRC()
		setParam(rc, "id", "0")

		ctrl.Update(ctx(), rc)
		assertStatusCode(t, rc, http.StatusBadRequest)
	})

	t.Run("not found returns 404", func(t *testing.T) {
		repo := newMockPrescriptionRepo()
		ctrl := newPrescriptionController(repo)

		rc := newRC()
		setParam(rc, "id", "999")
		rc.Request.SetBody([]byte(`{"name":"X"}`))

		ctrl.Update(ctx(), rc)
		assertStatusCode(t, rc, http.StatusNotFound)
	})

	t.Run("invalid JSON returns 400", func(t *testing.T) {
		repo := newMockPrescriptionRepo()
		seedPrescription(repo, 1, "Mahuang Tang")
		ctrl := newPrescriptionController(repo)

		rc := newRC()
		setParam(rc, "id", "1")
		rc.Request.SetBody([]byte(`{not-json`))

		ctrl.Update(ctx(), rc)
		assertStatusCode(t, rc, http.StatusBadRequest)
	})
}

func TestPrescriptionController_Delete(t *testing.T) {
	t.Run("happy path returns 204", func(t *testing.T) {
		repo := newMockPrescriptionRepo()
		seedPrescription(repo, 1, "Mahuang Tang")
		ctrl := newPrescriptionController(repo)

		rc := newRC()
		rc.Request.SetMethod("DELETE")
		rc.Request.SetRequestURI("/api/v1/history/prescriptions/1")
		setParam(rc, "id", "1")

		ctrl.Delete(ctx(), rc)
		assertStatusCode(t, rc, http.StatusNoContent)
	})

	t.Run("invalid id returns 400", func(t *testing.T) {
		repo := newMockPrescriptionRepo()
		ctrl := newPrescriptionController(repo)

		rc := newRC()
		setParam(rc, "id", "abc")

		ctrl.Delete(ctx(), rc)
		assertStatusCode(t, rc, http.StatusBadRequest)
	})

	t.Run("not found returns 404", func(t *testing.T) {
		repo := newMockPrescriptionRepo()
		ctrl := newPrescriptionController(repo)

		rc := newRC()
		setParam(rc, "id", "999")

		ctrl.Delete(ctx(), rc)
		assertStatusCode(t, rc, http.StatusNotFound)
	})
}

// ============================================================================
// MedicineController
// ============================================================================

func newMedicineController(repo *mockMedicineRepo) *controller.MedicineController {
	uc := usecase.NewMedicineUseCase(repo)
	return controller.NewMedicineController(uc)
}

func seedMedicine(repo *mockMedicineRepo, id int64, name string) *entity.Medicine {
	m := &entity.Medicine{Name: name}
	m.ID = id
	repo.items[id] = m
	return m
}

func TestMedicineController_List(t *testing.T) {
	t.Run("returns 200 with seeded medicines", func(t *testing.T) {
		repo := newMockMedicineRepo()
		seedMedicine(repo, 1, "Mahuang")
		ctrl := newMedicineController(repo)

		rc := newRC()
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/history/medicines?page=1&page_size=10")

		ctrl.List(ctx(), rc)
		body := assertStatusCode(t, rc, http.StatusOK)
		assert.Equal(t, int(errno.OK), body.Code)
	})

	t.Run("returns 200 on empty result", func(t *testing.T) {
		repo := newMockMedicineRepo()
		ctrl := newMedicineController(repo)

		rc := newRC()
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/history/medicines")

		ctrl.List(ctx(), rc)
		assertStatusCode(t, rc, http.StatusOK)
	})

	t.Run("propagates repo error as 500", func(t *testing.T) {
		repo := newMockMedicineRepo()
		repo.list = func(_ pagination.Params) ([]entity.Medicine, int, error) {
			return nil, 0, errno.New(errno.InternalError, "db down")
		}
		ctrl := newMedicineController(repo)

		rc := newRC()
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/history/medicines")

		ctrl.List(ctx(), rc)
		assert.Equal(t, http.StatusInternalServerError, rc.Response.StatusCode())
	})
}

func TestMedicineController_Create(t *testing.T) {
	t.Run("happy path returns 201", func(t *testing.T) {
		repo := newMockMedicineRepo()
		ctrl := newMedicineController(repo)

		rc := newRC()
		rc.Request.SetMethod("POST")
		rc.Request.SetRequestURI("/api/v1/history/medicines")
		rc.Request.SetBody([]byte(`{"name":"Mahuang","nature":"warm"}`))

		ctrl.Create(ctx(), rc)
		body := assertStatusCode(t, rc, http.StatusCreated)
		assert.Equal(t, "created", body.Message)
	})

	t.Run("invalid JSON returns 400", func(t *testing.T) {
		repo := newMockMedicineRepo()
		ctrl := newMedicineController(repo)

		rc := newRC()
		rc.Request.SetMethod("POST")
		rc.Request.SetRequestURI("/api/v1/history/medicines")
		rc.Request.SetBody([]byte(`{not-json`))

		ctrl.Create(ctx(), rc)
		assertStatusCode(t, rc, http.StatusBadRequest)
	})

	t.Run("empty name returns 400", func(t *testing.T) {
		repo := newMockMedicineRepo()
		ctrl := newMedicineController(repo)

		rc := newRC()
		rc.Request.SetMethod("POST")
		rc.Request.SetRequestURI("/api/v1/history/medicines")
		rc.Request.SetBody([]byte(`{"name":""}`))

		ctrl.Create(ctx(), rc)
		assertStatusCode(t, rc, http.StatusBadRequest)
	})
}

func TestMedicineController_Get(t *testing.T) {
	t.Run("found returns 200", func(t *testing.T) {
		repo := newMockMedicineRepo()
		seedMedicine(repo, 1, "Mahuang")
		ctrl := newMedicineController(repo)

		rc := newRC()
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/history/medicines/1")
		setParam(rc, "id", "1")

		ctrl.Get(ctx(), rc)
		assertStatusCode(t, rc, http.StatusOK)
	})

	t.Run("invalid id returns 400", func(t *testing.T) {
		repo := newMockMedicineRepo()
		ctrl := newMedicineController(repo)

		rc := newRC()
		setParam(rc, "id", "abc")

		ctrl.Get(ctx(), rc)
		assertStatusCode(t, rc, http.StatusBadRequest)
	})

	t.Run("not found returns 404", func(t *testing.T) {
		repo := newMockMedicineRepo()
		ctrl := newMedicineController(repo)

		rc := newRC()
		setParam(rc, "id", "999")

		ctrl.Get(ctx(), rc)
		assertStatusCode(t, rc, http.StatusNotFound)
	})
}

func TestMedicineController_Update(t *testing.T) {
	t.Run("happy path returns 200", func(t *testing.T) {
		repo := newMockMedicineRepo()
		seedMedicine(repo, 1, "Mahuang")
		ctrl := newMedicineController(repo)

		rc := newRC()
		rc.Request.SetMethod("PUT")
		rc.Request.SetRequestURI("/api/v1/history/medicines/1")
		setParam(rc, "id", "1")
		rc.Request.SetBody([]byte(`{"name":"Updated Medicine"}`))

		ctrl.Update(ctx(), rc)
		assertStatusCode(t, rc, http.StatusOK)
	})

	t.Run("invalid id returns 400", func(t *testing.T) {
		repo := newMockMedicineRepo()
		ctrl := newMedicineController(repo)

		rc := newRC()
		setParam(rc, "id", "0")

		ctrl.Update(ctx(), rc)
		assertStatusCode(t, rc, http.StatusBadRequest)
	})

	t.Run("not found returns 404", func(t *testing.T) {
		repo := newMockMedicineRepo()
		ctrl := newMedicineController(repo)

		rc := newRC()
		setParam(rc, "id", "999")
		rc.Request.SetBody([]byte(`{"name":"X"}`))

		ctrl.Update(ctx(), rc)
		assertStatusCode(t, rc, http.StatusNotFound)
	})

	t.Run("invalid JSON returns 400", func(t *testing.T) {
		repo := newMockMedicineRepo()
		seedMedicine(repo, 1, "Mahuang")
		ctrl := newMedicineController(repo)

		rc := newRC()
		setParam(rc, "id", "1")
		rc.Request.SetBody([]byte(`{not-json`))

		ctrl.Update(ctx(), rc)
		assertStatusCode(t, rc, http.StatusBadRequest)
	})
}

func TestMedicineController_Delete(t *testing.T) {
	t.Run("happy path returns 204", func(t *testing.T) {
		repo := newMockMedicineRepo()
		seedMedicine(repo, 1, "Mahuang")
		ctrl := newMedicineController(repo)

		rc := newRC()
		rc.Request.SetMethod("DELETE")
		rc.Request.SetRequestURI("/api/v1/history/medicines/1")
		setParam(rc, "id", "1")

		ctrl.Delete(ctx(), rc)
		assertStatusCode(t, rc, http.StatusNoContent)
	})

	t.Run("invalid id returns 400", func(t *testing.T) {
		repo := newMockMedicineRepo()
		ctrl := newMedicineController(repo)

		rc := newRC()
		setParam(rc, "id", "abc")

		ctrl.Delete(ctx(), rc)
		assertStatusCode(t, rc, http.StatusBadRequest)
	})

	t.Run("not found returns 404", func(t *testing.T) {
		repo := newMockMedicineRepo()
		ctrl := newMedicineController(repo)

		rc := newRC()
		setParam(rc, "id", "999")

		ctrl.Delete(ctx(), rc)
		assertStatusCode(t, rc, http.StatusNotFound)
	})
}

// ============================================================================
// DiseaseController
// ============================================================================

func newDiseaseController(repo *mockDiseaseRepo) *controller.DiseaseController {
	uc := usecase.NewDiseaseUseCase(repo)
	return controller.NewDiseaseController(uc)
}

func seedDisease(repo *mockDiseaseRepo, id int64, name string) *entity.Disease {
	d := &entity.Disease{Name: name}
	d.ID = id
	repo.items[id] = d
	return d
}

func TestDiseaseController_List(t *testing.T) {
	t.Run("returns 200 with seeded diseases", func(t *testing.T) {
		repo := newMockDiseaseRepo()
		seedDisease(repo, 1, "Shanghan")
		ctrl := newDiseaseController(repo)

		rc := newRC()
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/history/diseases?page=1&page_size=10")

		ctrl.List(ctx(), rc)
		body := assertStatusCode(t, rc, http.StatusOK)
		assert.Equal(t, int(errno.OK), body.Code)
	})

	t.Run("returns 200 on empty result", func(t *testing.T) {
		repo := newMockDiseaseRepo()
		ctrl := newDiseaseController(repo)

		rc := newRC()
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/history/diseases")

		ctrl.List(ctx(), rc)
		assertStatusCode(t, rc, http.StatusOK)
	})

	t.Run("propagates repo error as 500", func(t *testing.T) {
		repo := newMockDiseaseRepo()
		repo.list = func(_ pagination.Params) ([]entity.Disease, int, error) {
			return nil, 0, errno.New(errno.InternalError, "db down")
		}
		ctrl := newDiseaseController(repo)

		rc := newRC()
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/history/diseases")

		ctrl.List(ctx(), rc)
		assert.Equal(t, http.StatusInternalServerError, rc.Response.StatusCode())
	})
}

func TestDiseaseController_Create(t *testing.T) {
	t.Run("happy path returns 201", func(t *testing.T) {
		repo := newMockDiseaseRepo()
		ctrl := newDiseaseController(repo)

		rc := newRC()
		rc.Request.SetMethod("POST")
		rc.Request.SetRequestURI("/api/v1/history/diseases")
		rc.Request.SetBody([]byte(`{"name":"Shanghan","category":"external_contraction"}`))

		ctrl.Create(ctx(), rc)
		body := assertStatusCode(t, rc, http.StatusCreated)
		assert.Equal(t, "created", body.Message)
	})

	t.Run("invalid JSON returns 400", func(t *testing.T) {
		repo := newMockDiseaseRepo()
		ctrl := newDiseaseController(repo)

		rc := newRC()
		rc.Request.SetMethod("POST")
		rc.Request.SetRequestURI("/api/v1/history/diseases")
		rc.Request.SetBody([]byte(`{not-json`))

		ctrl.Create(ctx(), rc)
		assertStatusCode(t, rc, http.StatusBadRequest)
	})

	t.Run("empty name returns 400", func(t *testing.T) {
		repo := newMockDiseaseRepo()
		ctrl := newDiseaseController(repo)

		rc := newRC()
		rc.Request.SetMethod("POST")
		rc.Request.SetRequestURI("/api/v1/history/diseases")
		rc.Request.SetBody([]byte(`{"name":""}`))

		ctrl.Create(ctx(), rc)
		assertStatusCode(t, rc, http.StatusBadRequest)
	})
}

func TestDiseaseController_Get(t *testing.T) {
	t.Run("found returns 200", func(t *testing.T) {
		repo := newMockDiseaseRepo()
		seedDisease(repo, 1, "Shanghan")
		ctrl := newDiseaseController(repo)

		rc := newRC()
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/history/diseases/1")
		setParam(rc, "id", "1")

		ctrl.Get(ctx(), rc)
		assertStatusCode(t, rc, http.StatusOK)
	})

	t.Run("invalid id returns 400", func(t *testing.T) {
		repo := newMockDiseaseRepo()
		ctrl := newDiseaseController(repo)

		rc := newRC()
		setParam(rc, "id", "abc")

		ctrl.Get(ctx(), rc)
		assertStatusCode(t, rc, http.StatusBadRequest)
	})

	t.Run("not found returns 404", func(t *testing.T) {
		repo := newMockDiseaseRepo()
		ctrl := newDiseaseController(repo)

		rc := newRC()
		setParam(rc, "id", "999")

		ctrl.Get(ctx(), rc)
		assertStatusCode(t, rc, http.StatusNotFound)
	})
}

func TestDiseaseController_Update(t *testing.T) {
	t.Run("happy path returns 200", func(t *testing.T) {
		repo := newMockDiseaseRepo()
		seedDisease(repo, 1, "Shanghan")
		ctrl := newDiseaseController(repo)

		rc := newRC()
		rc.Request.SetMethod("PUT")
		rc.Request.SetRequestURI("/api/v1/history/diseases/1")
		setParam(rc, "id", "1")
		rc.Request.SetBody([]byte(`{"name":"Updated Disease"}`))

		ctrl.Update(ctx(), rc)
		assertStatusCode(t, rc, http.StatusOK)
	})

	t.Run("invalid id returns 400", func(t *testing.T) {
		repo := newMockDiseaseRepo()
		ctrl := newDiseaseController(repo)

		rc := newRC()
		setParam(rc, "id", "0")

		ctrl.Update(ctx(), rc)
		assertStatusCode(t, rc, http.StatusBadRequest)
	})

	t.Run("not found returns 404", func(t *testing.T) {
		repo := newMockDiseaseRepo()
		ctrl := newDiseaseController(repo)

		rc := newRC()
		setParam(rc, "id", "999")
		rc.Request.SetBody([]byte(`{"name":"X"}`))

		ctrl.Update(ctx(), rc)
		assertStatusCode(t, rc, http.StatusNotFound)
	})

	t.Run("invalid JSON returns 400", func(t *testing.T) {
		repo := newMockDiseaseRepo()
		seedDisease(repo, 1, "Shanghan")
		ctrl := newDiseaseController(repo)

		rc := newRC()
		setParam(rc, "id", "1")
		rc.Request.SetBody([]byte(`{not-json`))

		ctrl.Update(ctx(), rc)
		assertStatusCode(t, rc, http.StatusBadRequest)
	})
}

func TestDiseaseController_Delete(t *testing.T) {
	t.Run("happy path returns 204", func(t *testing.T) {
		repo := newMockDiseaseRepo()
		seedDisease(repo, 1, "Shanghan")
		ctrl := newDiseaseController(repo)

		rc := newRC()
		rc.Request.SetMethod("DELETE")
		rc.Request.SetRequestURI("/api/v1/history/diseases/1")
		setParam(rc, "id", "1")

		ctrl.Delete(ctx(), rc)
		assertStatusCode(t, rc, http.StatusNoContent)
	})

	t.Run("invalid id returns 400", func(t *testing.T) {
		repo := newMockDiseaseRepo()
		ctrl := newDiseaseController(repo)

		rc := newRC()
		setParam(rc, "id", "abc")

		ctrl.Delete(ctx(), rc)
		assertStatusCode(t, rc, http.StatusBadRequest)
	})

	t.Run("not found returns 404", func(t *testing.T) {
		repo := newMockDiseaseRepo()
		ctrl := newDiseaseController(repo)

		rc := newRC()
		setParam(rc, "id", "999")

		ctrl.Delete(ctx(), rc)
		assertStatusCode(t, rc, http.StatusNotFound)
	})
}

package controller_test

import (
	"context"
	"sync"

	"tcm-history-ai/backend/history-service/internal/domain/entity"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/pagination"
)

// paginate returns the offset/page_size slice of `all` according to p, plus
// the total count and a nil error so it can be used in 3-value returns.
func paginate[T any](all []T, p pagination.Params) ([]T, int, error) {
	_, pageSize, offset := p.Normalise()
	total := len(all)
	if offset > total {
		offset = total
	}
	end := offset + pageSize
	if end > total {
		end = total
	}
	if offset > end {
		offset = end
	}
	return all[offset:end], total, nil
}

// setID sets the BaseModel.ID on the entity (which embeds gormutil.BaseModel).
// We use a small helper because the embedded struct literal cannot set ID
// directly when constructing the entity in tests.
func setDynastyID(d *entity.Dynasty, id int64) { d.ID = id }
func setPersonID(p *entity.Person, id int64)   { p.ID = id }
func setSchoolID(s *entity.School, id int64)   { s.ID = id }
func setBookID(b *entity.Book, id int64)       { b.ID = id }
func setEventID(e *entity.Event, id int64)     { e.ID = id }
func setPrescriptionID(p *entity.Prescription, id int64) { p.ID = id }
func setMedicineID(m *entity.Medicine, id int64)         { m.ID = id }
func setDiseaseID(d *entity.Disease, id int64)           { d.ID = id }

// ============================================================================
// DynastyRepository mock
// ============================================================================

type mockDynastyRepo struct {
	mu     sync.Mutex
	items  map[int64]*entity.Dynasty
	create func(*entity.Dynasty) error
	update func(*entity.Dynasty) error
	delete func(int64) error
	find   func(int64) (*entity.Dynasty, error)
	list   func(pagination.Params) ([]entity.Dynasty, int, error)
	search func(string, pagination.Params) ([]entity.Dynasty, int, error)
}

func newMockDynastyRepo() *mockDynastyRepo {
	return &mockDynastyRepo{items: map[int64]*entity.Dynasty{}}
}

func (m *mockDynastyRepo) Create(_ context.Context, d *entity.Dynasty) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.create != nil {
		return m.create(d)
	}
	m.items[d.ID] = d
	return nil
}
func (m *mockDynastyRepo) Update(_ context.Context, d *entity.Dynasty) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.update != nil {
		return m.update(d)
	}
	if _, ok := m.items[d.ID]; !ok {
		return errno.New(errno.NotFound, "dynasty not found")
	}
	m.items[d.ID] = d
	return nil
}
func (m *mockDynastyRepo) Delete(_ context.Context, id int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.delete != nil {
		return m.delete(id)
	}
	if _, ok := m.items[id]; !ok {
		return errno.New(errno.NotFound, "dynasty not found")
	}
	delete(m.items, id)
	return nil
}
func (m *mockDynastyRepo) FindByID(_ context.Context, id int64) (*entity.Dynasty, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.find != nil {
		return m.find(id)
	}
	if d, ok := m.items[id]; ok {
		clone := *d
		return &clone, nil
	}
	return nil, nil
}
func (m *mockDynastyRepo) List(_ context.Context, p pagination.Params) ([]entity.Dynasty, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.list != nil {
		return m.list(p)
	}
	all := make([]entity.Dynasty, 0, len(m.items))
	for _, d := range m.items {
		all = append(all, *d)
	}
	return paginate(all, p)
}
func (m *mockDynastyRepo) Search(_ context.Context, kw string, p pagination.Params) ([]entity.Dynasty, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.search != nil {
		return m.search(kw, p)
	}
	all := make([]entity.Dynasty, 0, len(m.items))
	for _, d := range m.items {
		all = append(all, *d)
	}
	return paginate(all, p)
}

// ============================================================================
// PersonRepository mock
// ============================================================================

type mockPersonRepo struct {
	mu     sync.Mutex
	items  map[int64]*entity.Person
	create func(*entity.Person) error
	update func(*entity.Person) error
	delete func(int64) error
	find   func(int64) (*entity.Person, error)
	list   func(pagination.Params) ([]entity.Person, int, error)
	search func(string, pagination.Params) ([]entity.Person, int, error)
}

func newMockPersonRepo() *mockPersonRepo {
	return &mockPersonRepo{items: map[int64]*entity.Person{}}
}

func (m *mockPersonRepo) Create(_ context.Context, p *entity.Person) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.create != nil {
		return m.create(p)
	}
	m.items[p.ID] = p
	return nil
}
func (m *mockPersonRepo) Update(_ context.Context, p *entity.Person) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.update != nil {
		return m.update(p)
	}
	if _, ok := m.items[p.ID]; !ok {
		return errno.New(errno.NotFound, "person not found")
	}
	m.items[p.ID] = p
	return nil
}
func (m *mockPersonRepo) Delete(_ context.Context, id int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.delete != nil {
		return m.delete(id)
	}
	if _, ok := m.items[id]; !ok {
		return errno.New(errno.NotFound, "person not found")
	}
	delete(m.items, id)
	return nil
}
func (m *mockPersonRepo) FindByID(_ context.Context, id int64) (*entity.Person, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.find != nil {
		return m.find(id)
	}
	if p, ok := m.items[id]; ok {
		clone := *p
		return &clone, nil
	}
	return nil, nil
}
func (m *mockPersonRepo) List(_ context.Context, p pagination.Params) ([]entity.Person, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.list != nil {
		return m.list(p)
	}
	all := make([]entity.Person, 0, len(m.items))
	for _, p := range m.items {
		all = append(all, *p)
	}
	return paginate(all, p)
}
func (m *mockPersonRepo) Search(_ context.Context, kw string, p pagination.Params) ([]entity.Person, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.search != nil {
		return m.search(kw, p)
	}
	all := make([]entity.Person, 0, len(m.items))
	for _, p := range m.items {
		all = append(all, *p)
	}
	return paginate(all, p)
}

// ============================================================================
// SchoolRepository mock
// ============================================================================

type mockSchoolRepo struct {
	mu     sync.Mutex
	items  map[int64]*entity.School
	create func(*entity.School) error
	update func(*entity.School) error
	delete func(int64) error
	find   func(int64) (*entity.School, error)
	list   func(pagination.Params) ([]entity.School, int, error)
	search func(string, pagination.Params) ([]entity.School, int, error)
}

func newMockSchoolRepo() *mockSchoolRepo {
	return &mockSchoolRepo{items: map[int64]*entity.School{}}
}

func (m *mockSchoolRepo) Create(_ context.Context, s *entity.School) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.create != nil {
		return m.create(s)
	}
	m.items[s.ID] = s
	return nil
}
func (m *mockSchoolRepo) Update(_ context.Context, s *entity.School) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.update != nil {
		return m.update(s)
	}
	if _, ok := m.items[s.ID]; !ok {
		return errno.New(errno.NotFound, "school not found")
	}
	m.items[s.ID] = s
	return nil
}
func (m *mockSchoolRepo) Delete(_ context.Context, id int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.delete != nil {
		return m.delete(id)
	}
	if _, ok := m.items[id]; !ok {
		return errno.New(errno.NotFound, "school not found")
	}
	delete(m.items, id)
	return nil
}
func (m *mockSchoolRepo) FindByID(_ context.Context, id int64) (*entity.School, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.find != nil {
		return m.find(id)
	}
	if s, ok := m.items[id]; ok {
		clone := *s
		return &clone, nil
	}
	return nil, nil
}
func (m *mockSchoolRepo) List(_ context.Context, p pagination.Params) ([]entity.School, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.list != nil {
		return m.list(p)
	}
	all := make([]entity.School, 0, len(m.items))
	for _, s := range m.items {
		all = append(all, *s)
	}
	return paginate(all, p)
}
func (m *mockSchoolRepo) Search(_ context.Context, kw string, p pagination.Params) ([]entity.School, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.search != nil {
		return m.search(kw, p)
	}
	all := make([]entity.School, 0, len(m.items))
	for _, s := range m.items {
		all = append(all, *s)
	}
	return paginate(all, p)
}

// ============================================================================
// BookRepository mock
// ============================================================================

type mockBookRepo struct {
	mu     sync.Mutex
	items  map[int64]*entity.Book
	create func(*entity.Book) error
	update func(*entity.Book) error
	delete func(int64) error
	find   func(int64) (*entity.Book, error)
	list   func(pagination.Params) ([]entity.Book, int, error)
	search func(string, pagination.Params) ([]entity.Book, int, error)
}

func newMockBookRepo() *mockBookRepo {
	return &mockBookRepo{items: map[int64]*entity.Book{}}
}

func (m *mockBookRepo) Create(_ context.Context, b *entity.Book) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.create != nil {
		return m.create(b)
	}
	m.items[b.ID] = b
	return nil
}
func (m *mockBookRepo) Update(_ context.Context, b *entity.Book) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.update != nil {
		return m.update(b)
	}
	if _, ok := m.items[b.ID]; !ok {
		return errno.New(errno.NotFound, "book not found")
	}
	m.items[b.ID] = b
	return nil
}
func (m *mockBookRepo) Delete(_ context.Context, id int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.delete != nil {
		return m.delete(id)
	}
	if _, ok := m.items[id]; !ok {
		return errno.New(errno.NotFound, "book not found")
	}
	delete(m.items, id)
	return nil
}
func (m *mockBookRepo) FindByID(_ context.Context, id int64) (*entity.Book, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.find != nil {
		return m.find(id)
	}
	if b, ok := m.items[id]; ok {
		clone := *b
		return &clone, nil
	}
	return nil, nil
}
func (m *mockBookRepo) List(_ context.Context, p pagination.Params) ([]entity.Book, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.list != nil {
		return m.list(p)
	}
	all := make([]entity.Book, 0, len(m.items))
	for _, b := range m.items {
		all = append(all, *b)
	}
	return paginate(all, p)
}
func (m *mockBookRepo) Search(_ context.Context, kw string, p pagination.Params) ([]entity.Book, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.search != nil {
		return m.search(kw, p)
	}
	all := make([]entity.Book, 0, len(m.items))
	for _, b := range m.items {
		all = append(all, *b)
	}
	return paginate(all, p)
}

// ============================================================================
// EventRepository mock
// ============================================================================

type mockEventRepo struct {
	mu     sync.Mutex
	items  map[int64]*entity.Event
	create func(*entity.Event) error
	update func(*entity.Event) error
	delete func(int64) error
	find   func(int64) (*entity.Event, error)
	list   func(pagination.Params) ([]entity.Event, int, error)
	search func(string, pagination.Params) ([]entity.Event, int, error)
}

func newMockEventRepo() *mockEventRepo {
	return &mockEventRepo{items: map[int64]*entity.Event{}}
}

func (m *mockEventRepo) Create(_ context.Context, e *entity.Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.create != nil {
		return m.create(e)
	}
	m.items[e.ID] = e
	return nil
}
func (m *mockEventRepo) Update(_ context.Context, e *entity.Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.update != nil {
		return m.update(e)
	}
	if _, ok := m.items[e.ID]; !ok {
		return errno.New(errno.NotFound, "event not found")
	}
	m.items[e.ID] = e
	return nil
}
func (m *mockEventRepo) Delete(_ context.Context, id int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.delete != nil {
		return m.delete(id)
	}
	if _, ok := m.items[id]; !ok {
		return errno.New(errno.NotFound, "event not found")
	}
	delete(m.items, id)
	return nil
}
func (m *mockEventRepo) FindByID(_ context.Context, id int64) (*entity.Event, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.find != nil {
		return m.find(id)
	}
	if e, ok := m.items[id]; ok {
		clone := *e
		return &clone, nil
	}
	return nil, nil
}
func (m *mockEventRepo) List(_ context.Context, p pagination.Params) ([]entity.Event, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.list != nil {
		return m.list(p)
	}
	all := make([]entity.Event, 0, len(m.items))
	for _, e := range m.items {
		all = append(all, *e)
	}
	return paginate(all, p)
}
func (m *mockEventRepo) Search(_ context.Context, kw string, p pagination.Params) ([]entity.Event, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.search != nil {
		return m.search(kw, p)
	}
	all := make([]entity.Event, 0, len(m.items))
	for _, e := range m.items {
		all = append(all, *e)
	}
	return paginate(all, p)
}

// ============================================================================
// PrescriptionRepository mock
// ============================================================================

type mockPrescriptionRepo struct {
	mu     sync.Mutex
	items  map[int64]*entity.Prescription
	create func(*entity.Prescription) error
	update func(*entity.Prescription) error
	delete func(int64) error
	find   func(int64) (*entity.Prescription, error)
	list   func(pagination.Params) ([]entity.Prescription, int, error)
	search func(string, pagination.Params) ([]entity.Prescription, int, error)
}

func newMockPrescriptionRepo() *mockPrescriptionRepo {
	return &mockPrescriptionRepo{items: map[int64]*entity.Prescription{}}
}

func (m *mockPrescriptionRepo) Create(_ context.Context, p *entity.Prescription) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.create != nil {
		return m.create(p)
	}
	m.items[p.ID] = p
	return nil
}
func (m *mockPrescriptionRepo) Update(_ context.Context, p *entity.Prescription) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.update != nil {
		return m.update(p)
	}
	if _, ok := m.items[p.ID]; !ok {
		return errno.New(errno.NotFound, "prescription not found")
	}
	m.items[p.ID] = p
	return nil
}
func (m *mockPrescriptionRepo) Delete(_ context.Context, id int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.delete != nil {
		return m.delete(id)
	}
	if _, ok := m.items[id]; !ok {
		return errno.New(errno.NotFound, "prescription not found")
	}
	delete(m.items, id)
	return nil
}
func (m *mockPrescriptionRepo) FindByID(_ context.Context, id int64) (*entity.Prescription, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.find != nil {
		return m.find(id)
	}
	if p, ok := m.items[id]; ok {
		clone := *p
		return &clone, nil
	}
	return nil, nil
}
func (m *mockPrescriptionRepo) List(_ context.Context, p pagination.Params) ([]entity.Prescription, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.list != nil {
		return m.list(p)
	}
	all := make([]entity.Prescription, 0, len(m.items))
	for _, p := range m.items {
		all = append(all, *p)
	}
	return paginate(all, p)
}
func (m *mockPrescriptionRepo) Search(_ context.Context, kw string, p pagination.Params) ([]entity.Prescription, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.search != nil {
		return m.search(kw, p)
	}
	all := make([]entity.Prescription, 0, len(m.items))
	for _, p := range m.items {
		all = append(all, *p)
	}
	return paginate(all, p)
}

// ============================================================================
// MedicineRepository mock
// ============================================================================

type mockMedicineRepo struct {
	mu     sync.Mutex
	items  map[int64]*entity.Medicine
	create func(*entity.Medicine) error
	update func(*entity.Medicine) error
	delete func(int64) error
	find   func(int64) (*entity.Medicine, error)
	list   func(pagination.Params) ([]entity.Medicine, int, error)
	search func(string, pagination.Params) ([]entity.Medicine, int, error)
}

func newMockMedicineRepo() *mockMedicineRepo {
	return &mockMedicineRepo{items: map[int64]*entity.Medicine{}}
}

func (m *mockMedicineRepo) Create(_ context.Context, e *entity.Medicine) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.create != nil {
		return m.create(e)
	}
	m.items[e.ID] = e
	return nil
}
func (m *mockMedicineRepo) Update(_ context.Context, e *entity.Medicine) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.update != nil {
		return m.update(e)
	}
	if _, ok := m.items[e.ID]; !ok {
		return errno.New(errno.NotFound, "medicine not found")
	}
	m.items[e.ID] = e
	return nil
}
func (m *mockMedicineRepo) Delete(_ context.Context, id int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.delete != nil {
		return m.delete(id)
	}
	if _, ok := m.items[id]; !ok {
		return errno.New(errno.NotFound, "medicine not found")
	}
	delete(m.items, id)
	return nil
}
func (m *mockMedicineRepo) FindByID(_ context.Context, id int64) (*entity.Medicine, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.find != nil {
		return m.find(id)
	}
	if e, ok := m.items[id]; ok {
		clone := *e
		return &clone, nil
	}
	return nil, nil
}
func (m *mockMedicineRepo) List(_ context.Context, p pagination.Params) ([]entity.Medicine, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.list != nil {
		return m.list(p)
	}
	all := make([]entity.Medicine, 0, len(m.items))
	for _, e := range m.items {
		all = append(all, *e)
	}
	return paginate(all, p)
}
func (m *mockMedicineRepo) Search(_ context.Context, kw string, p pagination.Params) ([]entity.Medicine, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.search != nil {
		return m.search(kw, p)
	}
	all := make([]entity.Medicine, 0, len(m.items))
	for _, e := range m.items {
		all = append(all, *e)
	}
	return paginate(all, p)
}

// ============================================================================
// DiseaseRepository mock
// ============================================================================

type mockDiseaseRepo struct {
	mu     sync.Mutex
	items  map[int64]*entity.Disease
	create func(*entity.Disease) error
	update func(*entity.Disease) error
	delete func(int64) error
	find   func(int64) (*entity.Disease, error)
	list   func(pagination.Params) ([]entity.Disease, int, error)
	search func(string, pagination.Params) ([]entity.Disease, int, error)
}

func newMockDiseaseRepo() *mockDiseaseRepo {
	return &mockDiseaseRepo{items: map[int64]*entity.Disease{}}
}

func (m *mockDiseaseRepo) Create(_ context.Context, d *entity.Disease) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.create != nil {
		return m.create(d)
	}
	m.items[d.ID] = d
	return nil
}
func (m *mockDiseaseRepo) Update(_ context.Context, d *entity.Disease) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.update != nil {
		return m.update(d)
	}
	if _, ok := m.items[d.ID]; !ok {
		return errno.New(errno.NotFound, "disease not found")
	}
	m.items[d.ID] = d
	return nil
}
func (m *mockDiseaseRepo) Delete(_ context.Context, id int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.delete != nil {
		return m.delete(id)
	}
	if _, ok := m.items[id]; !ok {
		return errno.New(errno.NotFound, "disease not found")
	}
	delete(m.items, id)
	return nil
}
func (m *mockDiseaseRepo) FindByID(_ context.Context, id int64) (*entity.Disease, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.find != nil {
		return m.find(id)
	}
	if d, ok := m.items[id]; ok {
		clone := *d
		return &clone, nil
	}
	return nil, nil
}
func (m *mockDiseaseRepo) List(_ context.Context, p pagination.Params) ([]entity.Disease, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.list != nil {
		return m.list(p)
	}
	all := make([]entity.Disease, 0, len(m.items))
	for _, d := range m.items {
		all = append(all, *d)
	}
	return paginate(all, p)
}
func (m *mockDiseaseRepo) Search(_ context.Context, kw string, p pagination.Params) ([]entity.Disease, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.search != nil {
		return m.search(kw, p)
	}
	all := make([]entity.Disease, 0, len(m.items))
	for _, d := range m.items {
		all = append(all, *d)
	}
	return paginate(all, p)
}

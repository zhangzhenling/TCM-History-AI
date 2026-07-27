// History API 模块：朝代/人物/学派/著作/事件/方剂/药物/疾病 CRUD + 跨实体检索。
// 端点对齐 backend/history-service/internal/controller/router.go：/api/v1/history/*。

import type { AxiosInstance } from 'axios';

import { buildQuery, type ListResponse, type PageParams } from '../types';
import type {
  Book,
  BookRequest,
  Disease,
  DiseaseRequest,
  Dynasty,
  DynastyRequest,
  HistoryEvent,
  EventRequest,
  Medicine,
  MedicineRequest,
  Person,
  PersonRequest,
  Prescription,
  PrescriptionRequest,
  School,
  SchoolRequest,
  SearchParams,
  SearchResponse,
} from './history-types';

export class HistoryApi {
  constructor(private http: AxiosInstance) {}

  // ---- Dynasties ----
  listDynasties(params?: PageParams): Promise<ListResponse<Dynasty>> {
    return this.http.get('/api/v1/history/dynasties', {
      params: buildQuery(params ?? {}),
    }) as unknown as Promise<ListResponse<Dynasty>>;
  }
  getDynasty(id: number | string): Promise<Dynasty> {
    return this.http.get(`/api/v1/history/dynasties/${id}`) as unknown as Promise<Dynasty>;
  }
  createDynasty(payload: DynastyRequest): Promise<Dynasty> {
    return this.http.post('/api/v1/history/dynasties', payload) as unknown as Promise<Dynasty>;
  }
  updateDynasty(id: number | string, payload: DynastyRequest): Promise<Dynasty> {
    return this.http.put(`/api/v1/history/dynasties/${id}`, payload) as unknown as Promise<Dynasty>;
  }
  deleteDynasty(id: number | string): Promise<void> {
    return this.http.delete(`/api/v1/history/dynasties/${id}`) as unknown as Promise<void>;
  }

  // ---- Persons ----
  listPersons(
    params?: PageParams & { dynasty_id?: number; keyword?: string },
  ): Promise<ListResponse<Person>> {
    return this.http.get('/api/v1/history/persons', {
      params: buildQuery(params ?? {}),
    }) as unknown as Promise<ListResponse<Person>>;
  }
  getPerson(id: number | string): Promise<Person> {
    return this.http.get(`/api/v1/history/persons/${id}`) as unknown as Promise<Person>;
  }
  createPerson(payload: PersonRequest): Promise<Person> {
    return this.http.post('/api/v1/history/persons', payload) as unknown as Promise<Person>;
  }
  updatePerson(id: number | string, payload: PersonRequest): Promise<Person> {
    return this.http.put(`/api/v1/history/persons/${id}`, payload) as unknown as Promise<Person>;
  }
  deletePerson(id: number | string): Promise<void> {
    return this.http.delete(`/api/v1/history/persons/${id}`) as unknown as Promise<void>;
  }

  // ---- Schools ----
  listSchools(params?: PageParams): Promise<ListResponse<School>> {
    return this.http.get('/api/v1/history/schools', {
      params: buildQuery(params ?? {}),
    }) as unknown as Promise<ListResponse<School>>;
  }
  getSchool(id: number | string): Promise<School> {
    return this.http.get(`/api/v1/history/schools/${id}`) as unknown as Promise<School>;
  }
  createSchool(payload: SchoolRequest): Promise<School> {
    return this.http.post('/api/v1/history/schools', payload) as unknown as Promise<School>;
  }
  updateSchool(id: number | string, payload: SchoolRequest): Promise<School> {
    return this.http.put(`/api/v1/history/schools/${id}`, payload) as unknown as Promise<School>;
  }
  deleteSchool(id: number | string): Promise<void> {
    return this.http.delete(`/api/v1/history/schools/${id}`) as unknown as Promise<void>;
  }

  // ---- Books ----
  listBooks(
    params?: PageParams & { dynasty_id?: number; category?: string },
  ): Promise<ListResponse<Book>> {
    return this.http.get('/api/v1/history/books', {
      params: buildQuery(params ?? {}),
    }) as unknown as Promise<ListResponse<Book>>;
  }
  getBook(id: number | string): Promise<Book> {
    return this.http.get(`/api/v1/history/books/${id}`) as unknown as Promise<Book>;
  }
  createBook(payload: BookRequest): Promise<Book> {
    return this.http.post('/api/v1/history/books', payload) as unknown as Promise<Book>;
  }
  updateBook(id: number | string, payload: BookRequest): Promise<Book> {
    return this.http.put(`/api/v1/history/books/${id}`, payload) as unknown as Promise<Book>;
  }
  deleteBook(id: number | string): Promise<void> {
    return this.http.delete(`/api/v1/history/books/${id}`) as unknown as Promise<void>;
  }

  // ---- Events ----
  listEvents(
    params?: PageParams & { dynasty_id?: number; event_type?: string },
  ): Promise<ListResponse<HistoryEvent>> {
    return this.http.get('/api/v1/history/events', {
      params: buildQuery(params ?? {}),
    }) as unknown as Promise<ListResponse<HistoryEvent>>;
  }
  getEvent(id: number | string): Promise<HistoryEvent> {
    return this.http.get(`/api/v1/history/events/${id}`) as unknown as Promise<HistoryEvent>;
  }
  createEvent(payload: EventRequest): Promise<HistoryEvent> {
    return this.http.post('/api/v1/history/events', payload) as unknown as Promise<HistoryEvent>;
  }
  updateEvent(id: number | string, payload: EventRequest): Promise<HistoryEvent> {
    return this.http.put(
      `/api/v1/history/events/${id}`,
      payload,
    ) as unknown as Promise<HistoryEvent>;
  }
  deleteEvent(id: number | string): Promise<void> {
    return this.http.delete(`/api/v1/history/events/${id}`) as unknown as Promise<void>;
  }

  // ---- Prescriptions ----
  listPrescriptions(
    params?: PageParams & { category?: string },
  ): Promise<ListResponse<Prescription>> {
    return this.http.get('/api/v1/history/prescriptions', {
      params: buildQuery(params ?? {}),
    }) as unknown as Promise<ListResponse<Prescription>>;
  }
  getPrescription(id: number | string): Promise<Prescription> {
    return this.http.get(`/api/v1/history/prescriptions/${id}`) as unknown as Promise<Prescription>;
  }
  createPrescription(payload: PrescriptionRequest): Promise<Prescription> {
    return this.http.post(
      '/api/v1/history/prescriptions',
      payload,
    ) as unknown as Promise<Prescription>;
  }
  updatePrescription(id: number | string, payload: PrescriptionRequest): Promise<Prescription> {
    return this.http.put(
      `/api/v1/history/prescriptions/${id}`,
      payload,
    ) as unknown as Promise<Prescription>;
  }
  deletePrescription(id: number | string): Promise<void> {
    return this.http.delete(`/api/v1/history/prescriptions/${id}`) as unknown as Promise<void>;
  }

  // ---- Medicines ----
  listMedicines(params?: PageParams): Promise<ListResponse<Medicine>> {
    return this.http.get('/api/v1/history/medicines', {
      params: buildQuery(params ?? {}),
    }) as unknown as Promise<ListResponse<Medicine>>;
  }
  getMedicine(id: number | string): Promise<Medicine> {
    return this.http.get(`/api/v1/history/medicines/${id}`) as unknown as Promise<Medicine>;
  }
  createMedicine(payload: MedicineRequest): Promise<Medicine> {
    return this.http.post('/api/v1/history/medicines', payload) as unknown as Promise<Medicine>;
  }
  updateMedicine(id: number | string, payload: MedicineRequest): Promise<Medicine> {
    return this.http.put(
      `/api/v1/history/medicines/${id}`,
      payload,
    ) as unknown as Promise<Medicine>;
  }
  deleteMedicine(id: number | string): Promise<void> {
    return this.http.delete(`/api/v1/history/medicines/${id}`) as unknown as Promise<void>;
  }

  // ---- Diseases ----
  listDiseases(params?: PageParams): Promise<ListResponse<Disease>> {
    return this.http.get('/api/v1/history/diseases', {
      params: buildQuery(params ?? {}),
    }) as unknown as Promise<ListResponse<Disease>>;
  }
  getDisease(id: number | string): Promise<Disease> {
    return this.http.get(`/api/v1/history/diseases/${id}`) as unknown as Promise<Disease>;
  }
  createDisease(payload: DiseaseRequest): Promise<Disease> {
    return this.http.post('/api/v1/history/diseases', payload) as unknown as Promise<Disease>;
  }
  updateDisease(id: number | string, payload: DiseaseRequest): Promise<Disease> {
    return this.http.put(
      `/api/v1/history/diseases/${id}`,
      payload,
    ) as unknown as Promise<Disease>;
  }
  deleteDisease(id: number | string): Promise<void> {
    return this.http.delete(`/api/v1/history/diseases/${id}`) as unknown as Promise<void>;
  }

  // ---- Search ----
  search(params: SearchParams): Promise<SearchResponse> {
    const q = buildQuery({
      q: params.q,
      types: params.types,
      page: params.page,
      page_size: params.page_size,
    });
    return this.http.get('/api/v1/history/search', {
      params: q,
    }) as unknown as Promise<SearchResponse>;
  }
}

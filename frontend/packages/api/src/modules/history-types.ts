// History Service 类型定义，对齐 backend/history-service/internal/application/dto。
// 字段命名与后端 json tag 完全一致（snake_case）。

export interface Dynasty {
  id: number;
  name: string;
  start_year: number;
  end_year: number;
  sort_order: number;
  description: string;
}

export interface DynastyRequest {
  name: string;
  start_year?: number;
  end_year?: number;
  sort_order?: number;
  description?: string;
}

export interface Person {
  id: number;
  name: string;
  courtesy_name: string;
  alias_name: string;
  dynasty_id: number;
  birth_year: number;
  death_year: number;
  gender: string;
  title: string;
  biography: string;
  achievements: string;
  portrait_url: string;
}

export interface PersonRequest {
  name: string;
  courtesy_name?: string;
  alias_name?: string;
  dynasty_id?: number;
  birth_year?: number;
  death_year?: number;
  gender?: string;
  title?: string;
  biography?: string;
  achievements?: string;
  portrait_url?: string;
}

export interface School {
  id: number;
  name: string;
  dynasty_id: number;
  founder_person_id: number;
  summary: string;
  established_year: number;
}

export interface SchoolRequest {
  name: string;
  dynasty_id?: number;
  founder_person_id?: number;
  summary?: string;
  established_year?: number;
}

export interface Book {
  id: number;
  title: string;
  dynasty_id: number;
  published_year: number;
  category: string;
  summary: string;
  volume_count: number;
  is_extant: boolean;
  file_url: string;
}

export interface BookRequest {
  title: string;
  dynasty_id?: number;
  published_year?: number;
  category?: string;
  summary?: string;
  volume_count?: number;
  is_extant?: boolean;
  file_url?: string;
}

export interface HistoryEvent {
  id: number;
  title: string;
  dynasty_id: number;
  occurred_year: number;
  event_type: string;
  description: string;
  impact: string;
  location: string;
}

export interface EventRequest {
  title: string;
  dynasty_id?: number;
  occurred_year?: number;
  event_type: string;
  description?: string;
  impact?: string;
  location?: string;
}

export interface Prescription {
  id: number;
  name: string;
  pinyin: string;
  source_book_id: number;
  source_person_id: number;
  dynasty_id: number;
  composition: string;
  usage: string;
  indications: string;
  category: string;
}

export interface PrescriptionRequest {
  name: string;
  pinyin?: string;
  source_book_id?: number;
  source_person_id?: number;
  dynasty_id?: number;
  composition?: string;
  usage?: string;
  indications?: string;
  category?: string;
}

export interface Medicine {
  id: number;
  name: string;
  pinyin: string;
  alias_json: string[] | string;
  nature: string;
  flavor: string;
  meridian: string;
  efficacy: string;
  dosage: string;
  toxicity: string;
}

export interface MedicineRequest {
  name: string;
  pinyin?: string;
  alias?: string[];
  nature?: string;
  flavor?: string;
  meridian?: string;
  efficacy?: string;
  dosage?: string;
  toxicity?: string;
}

export interface Disease {
  id: number;
  name: string;
  pinyin: string;
  category: string;
  description: string;
  symptoms: string;
  tcm_pathogenesis: string;
}

export interface DiseaseRequest {
  name: string;
  pinyin?: string;
  category?: string;
  description?: string;
  symptoms?: string;
  tcm_pathogenesis?: string;
}

export interface SearchHit {
  type: string;
  id: number;
  score?: number;
  source: Record<string, unknown>;
}

export interface SearchResponse {
  total: number;
  items: SearchHit[];
}

export interface SearchParams {
  q: string;
  types?: string[];
  page?: number;
  page_size?: number;
}

// 实体类型与朝代枚举：与后端 History Service 实体保持一致。

export type EntityType =
  'dynasty' | 'person' | 'school' | 'book' | 'event' | 'prescription' | 'medicine' | 'disease';

export const ENTITY_LABELS: Record<EntityType, string> = {
  dynasty: '朝代',
  person: '人物',
  school: '学派',
  book: '著作',
  event: '事件',
  prescription: '方剂',
  medicine: '药物',
  disease: '疾病',
};

/** 朝代列表，sort_order 与后端种子数据 history_dynasty 保持一致。 */
export interface DynastyOption {
  id: number;
  name: string;
  startYear: number;
  endYear: number;
  sortOrder: number;
  description?: string;
}

/** 将年份（可能为负，表示公元前）格式化为「公元前 150 年」/「公元 200 年」。 */
export function formatYear(year: number | null | undefined): string {
  if (year === null || year === undefined) return '—';
  if (year < 0) return `公元前 ${-year} 年`;
  if (year === 0) return '公元元年';
  return `公元 ${year} 年`;
}

/** 给定起止年份，返回「公元前 150 - 公元 219 年」式区间。 */
export function formatYearRange(start?: number, end?: number): string {
  const s = start !== null && start !== undefined ? formatYear(start) : '?';
  const e = end !== null && end !== undefined ? formatYear(end) : '?';
  return `${s} — ${e}`;
}

/** 默认分页参数。 */
export const DEFAULT_PAGE_SIZE = 20;
export const MAX_PAGE_SIZE = 100;

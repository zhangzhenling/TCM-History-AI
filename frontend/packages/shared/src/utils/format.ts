// 通用格式化工具：时间、ID 截断、文本省略。

/** 格式化 ISO 时间戳为「2026-07-25 14:30」。 */
export function formatDateTime(input: string | number | Date | null | undefined): string {
  if (!input) return '—';
  const d = new Date(input);
  if (Number.isNaN(d.getTime())) return '—';
  const pad = (n: number) => String(n).padStart(2, '0');
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

/** 格式化为日期「2026-07-25」。 */
export function formatDate(input: string | number | Date | null | undefined): string {
  if (!input) return '—';
  const d = new Date(input);
  if (Number.isNaN(d.getTime())) return '—';
  const pad = (n: number) => String(n).padStart(2, '0');
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
}

/** 截断长文本，超出 length 时以 … 收尾。 */
export function truncate(text: string, length = 80): string {
  if (!text) return '';
  return text.length > length ? text.slice(0, length) + '…' : text;
}

/** 将雪花 ID（bigint 字符串）展示为短形式「#…a3f9」。 */
export function shortId(id: string | number | null | undefined): string {
  if (id === null || id === undefined) return '—';
  const s = String(id);
  return s.length > 8 ? `#${s.slice(-8)}` : `#${s}`;
}

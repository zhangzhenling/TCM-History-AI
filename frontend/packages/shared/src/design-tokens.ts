// 设计 Token：中医风格配色、字号、圆角、间距、断点、阴影。
// 与文档 12-前端设计.md §9.1 保持一致，色值经 WCAG AA 对比度校验。

export const designTokens = {
  color: {
    primary: '#A23A30', // 朱砂红
    primaryHover: '#8E2E25',
    ink: '#1F1A17', // 墨黑
    paper: '#F7F2E8', // 宣纸米白
    paperDark: '#2A2520',
    celadon: '#5C8A6A', // 青瓷绿
    indigo: '#2C4A6B', // 藏青蓝
    gold: '#C9A24A',
    dynasty: {
      preQin: '#5C8A6A',
      qinHan: '#A23A30',
      weiJin: '#7B5BA0',
      suiTang: '#C9A24A',
      songYuan: '#3E7CB1',
      mingQing: '#8E2E25',
      modern: '#4A4A4A',
    },
  },
  fontSize: { xs: 12, sm: 13, base: 14, lg: 16, xl: 18, xxl: 22, display: 32 },
  radius: { sm: 4, base: 6, lg: 8, pill: 999 },
  spacing: { xs: 4, sm: 8, base: 12, lg: 16, xl: 24, xxl: 32 },
  breakpoint: { mobile: 768, tablet: 1024, desktop: 1280, wide: 1536 },
  shadow: {
    card: '0 2px 8px rgba(31, 26, 23, 0.08)',
    hover: '0 4px 16px rgba(31, 26, 23, 0.12)',
  },
} as const;

export type DesignTokens = typeof designTokens;

/** 将设计 Token 注入为 CSS 变量，供全局样式引用。 */
export function designTokensToCSSVars(prefix = '--tcm'): Record<string, string> {
  const vars: Record<string, string> = {};
  const walk = (obj: Record<string, unknown>, path: string) => {
    for (const [k, v] of Object.entries(obj)) {
      const key = `${path}-${k}`;
      if (v && typeof v === 'object') {
        walk(v as Record<string, unknown>, key);
      } else {
        vars[key] = String(v);
      }
    }
  };
  walk(designTokens as unknown as Record<string, unknown>, prefix);
  return vars;
}

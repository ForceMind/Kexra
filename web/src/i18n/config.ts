// ARTEX 简体中文版默认以 zh 构建，保留 ko 以供对照和兼容旧部署。
// 静态导出不使用 cookies()/headers()；语言在构建时由 NEXT_PUBLIC_LOCALE 决定。
export const LOCALES = ["zh", "ko"] as const;
export type Locale = (typeof LOCALES)[number];
export const DEFAULT_LOCALE: Locale = "zh";

// 未设置或配置了不支持的值时，始终回退到简体中文。
// 若有保留韩语界面的需求，构建时显式设置 NEXT_PUBLIC_LOCALE=ko。
export function resolveLocale(): Locale {
  const raw = process.env.NEXT_PUBLIC_LOCALE;
  return LOCALES.includes(raw as Locale) ? (raw as Locale) : DEFAULT_LOCALE;
}

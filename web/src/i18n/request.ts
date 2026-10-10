import { getRequestConfig } from "next-intl/server";

import { type Locale, resolveLocale } from "./config";

// 显式导入映射让打包器只加载目标语言。zh 是简体中文默认资源，
// ko 保留用于对照与旧版兼容；静态导出不使用动态路由或请求头识别语言。
const loaders: Record<Locale, () => Promise<{ default: Record<string, unknown> }>> = {
  zh: () => import("../../messages/zh.json"),
  ko: () => import("../../messages/ko.json"),
};

export default getRequestConfig(async () => {
  const locale = resolveLocale();
  const messages = (await loaders[locale]()).default;
  return { locale, messages };
});

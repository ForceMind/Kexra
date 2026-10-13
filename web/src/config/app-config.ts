import { resolveLocale } from "@/i18n/config";

import packageJson from "../../package.json";

const currentYear = new Date().getFullYear();

// 浏览器标题与 SEO 描述跟随构建语言（zh 默认，ko 可选）。
// 必须与 src/i18n/config.ts 的 locale 保持一致。
const META_BY_LOCALE = {
  ko: {
    title: "Kexra: 자율 침투 테스트 콘솔",
    description: "LLM 기반으로 자율 침투 테스트를 수행하는 시스템 콘솔",
  },
  zh: {
    title: "Kexra — 自主渗透测试控制台",
    description: "多智能体驱动的授权安全评估与证据管理平台",
  },
} as const;

export const APP_CONFIG = {
  name: "Kexra",
  version: packageJson.version,
  copyright: `© ${currentYear}, Kexra.`,
  meta: META_BY_LOCALE[resolveLocale()],
};

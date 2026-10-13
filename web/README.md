# Kexra Web 前端

**代码状态**：Kexra Logo、登录品牌与中文构建已进入 `main`；前端与本地服务已通过隔离环境检查。

Kexra 的 Web 管理界面基于 **Next.js 16、React 19、TypeScript、Tailwind、next-intl**。Go API 位于仓库根目录的 `server/` 等路径；本目录不是独立的后端服务。

## 开发

~~~bash
cd web
npm ci
npm run check
npm run dev
~~~

开发服务器实际端口以控制台为准，Next.js 默认常为 3000。若需访问真实 Go API，先在隔离环境准备本地数据库和后端；前端转发规则参见 `next.config.mjs`，默认调用 `http://localhost:8787`。

## 构建与中文验收

~~~bash
npm run build:static
cd ..
python3 -I scripts/check-i18n-keys.py
python3 -I scripts/check-web-zh.py
~~~

`build:static` 通过环境变量 `NEXT_EXPORT=1` 导出到 `web/out/`。默认使用简体中文 `zh`；如需韩语兼容构建，设置 `NEXT_PUBLIC_LOCALE=ko` 后重新构建。前端标识使用 Kexra SVG，历史 PNG 素材不应继续作为现行品牌资产。

## 关键结构

- `src/app/`：Next.js 页面及路由
- `src/components/`：UI 组件
- `src/lib/`：接口客户端、模型、展示数据
- `src/i18n/` 与 `messages/`：本地化
- `src/config/app-config.ts`：产品名、版本显示、浏览器标题
- `public/`：静态资源（实际使用的新 Logo 为 `logo.svg`）

## 兼容与安全

`kexra_token` 与旧 `artex_token` 处于迁移期。不要随意删除旧读取路径或真实会话数据。`NEXT_PUBLIC_MOCK=1` 仅用于模拟数据展示；它的通过不等于真实 Agent、数据库、审批系统通过验收。

完整说明见 [安装指南](../docs/INSTALLATION.md)、[开发与测试](../docs/DEVELOPMENT.md)、[中文化规范](../docs/LOCALIZATION.zh-CN.md)。

# ARTEX Web 前端

本目录是 ARTEX 的 Next.js / React / TypeScript 前端，不是一个独立部署的完整服务。后端位于仓库根目录的 `server/`、`cmd/artex/` 等路径。

## 常用操作

准备 Node.js 22，并在本目录执行：

```bash
npm ci
npm run check
npm run build:static
```

- `npm run check`：Biome 代码风格、静态质量与可访问性检查。
- `npm run build:static`：为 Go 单二进制构建导出静态前端到 `out/`。
- `npm run dev`：用于本地 UI 开发；连接实际 API 前请审查代理、鉴权和目标环境。
- `src/app/`：页面路由；`src/components/`：组件；`src/lib/`：客户端逻辑；`messages/`：多语言文案。
- `package-lock.json` 应随依赖变动同步提交；`node_modules/`、`.next/`、`out/` 均不应提交。

## 合并前校验

仓库中的 `.github/workflows/web.yml` 对 `web/**` 变动运行静态构建、TypeScript、Biome 和用户界面中文字符残留检查（原韩语本地化分支的约束）。不要擅自弱化这些门禁；新增或修改语言文案时同步验证 `messages/`。

完整项目的编译、内嵌前端和运行要求见 [中文维护指南](../docs/REPOSITORY_GUIDE.zh-CN.md)。

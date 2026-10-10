# ARTEX 简体中文本地化说明

**当前目标：** 让 ForceMind/Artex 的默认用户体验面向简体中文用户，同时保持旧版韩语资源可选，不破坏现有 API 协议、证据和原始安全声明。

## 默认语言
- 默认构建：`cd web && npm ci && npm run build:static`，使用简体中文（`zh`）。
- 显式中文构建：`NEXT_PUBLIC_LOCALE=zh npm run build:static`。
- 韩语兼容构建：`NEXT_PUBLIC_LOCALE=ko npm run build:static`。
- 语言是**构建时选项**，不是浏览器内即时切换；切换语言必须重新构建前端。
- 文案资源：`web/messages/zh.json`、`web/messages/ko.json`（两者的键必须保持一一对应）。

## 验收方式
1. `npm run check`、`npm run build:static`，核对 `web/out/index.html` 的 `<html lang="zh">` 和中文标题。
2. `python3 -I scripts/check-web-zh.py` 检测**静态页面**可见韩语残留。
3. 检查登录、任务列表、详情、聊天、资产、设置与报告；核对模型返回的内容、通知与运行时 mock 数据。
4. 检查动态插值、富文本标签（如 `<terms>`）、换行、Markdown 和安全免责声明。
5. 使用 `go test ./... -count=1` 验证后端输出改动没有破坏兼容性。

**已知边界：** 静态页面检查不会覆盖 JavaScript 运行时演示数据、聊天记录及 AI 模型本身生成的文字。后端/提示词改动应另行通过回归测试核实；未完成前不能宣称“所有输出都已中文化”。历史韩语说明保留为 `README.ko.md`，原作者版权与 AGPL-3.0 许可证不变。

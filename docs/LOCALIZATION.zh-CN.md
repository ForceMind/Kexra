# Kexra 中文化与术语规范

## 默认语言与构建

Kexra 前端使用 next-intl。`web/src/i18n/config.ts` 默认 locale 为 `zh`；保留 `ko` 用于兼容性比较。它属于**构建时语言选择**，而不是浏览器内即时切换。

~~~bash
cd web
npm ci
npm run build:static
# 若需韩语兼容构建，单独运行：
NEXT_PUBLIC_LOCALE=ko npm run build:static
cd ..
~~~

资源文件：`web/messages/zh.json`、`web/messages/ko.json`。新增文案须保持两个 JSON 的字段结构一致，键名不等于语言文字。

## 翻译范围

**应中文化**：导航、表单、状态、错误提示、通知展示文字、报告标题、非原文的 Agent 最终摘要和演示数据的解释性文本。

**必须保留原文**：命令、代码、日志、URL、请求/响应、漏洞证据、第三方许可证、检测 IOC、技术标识、原始安全样例和用户上传文本。

旧 `ARTEX_*` 配置、`artex_token`、数据库和 Go module 是协议兼容项，不能直接全文替换。软件历史版权信息必须保留。

## 术语建议

| 原词 | 中文展示 |
| --- | --- |
| Planner | 规划智能体 |
| Worker | 执行智能体 |
| Main Agent | 主智能体 |
| Finding | 安全发现 / 漏洞发现（按是否已验证区分） |
| Evidence | 证据 |
| Asset | 资产 |
| Intent | 探索意图 |
| Scope | 授权测试范围 |
| Retest | 复测 |

## 已合并的中文化增量

- [PR #4](https://github.com/ForceMind/Kexra/pull/4)：认证 API、数据库配置与 CLI 启动提示、通知模板和六种通知渠道错误消息已改为默认简体中文。
- 已有 Go 单元测试断言这些路径实际输出中文；其他 `server/`、`selfupdate/` 与第三方工具输出仍需单独检查，不能把本批迁移视为所有界面已翻译。

## 回归验证

~~~bash
cd web && npm run check && npm run build:static && cd ..
python3 -I scripts/check-i18n-keys.py
python3 -I scripts/check-web-zh.py
python3 -I scripts/check-doc-links.py
~~~

这些校验不能证明运行时所有韩语或英文都已清除。通知模块、历史数据和外部模型仍可能产生非中文内容，需逐条测验。对于较长的 AI 输出，中文语言约束不是绝对保证，结论还要与原始证据核对。

相关：[Web 子项目](../web/README.md) · [开发与测试](DEVELOPMENT.md)。

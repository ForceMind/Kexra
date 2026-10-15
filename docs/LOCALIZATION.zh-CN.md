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
- 已有 Go 单元测试断言上述路径实际输出中文；其余服务端模块与第三方工具输出仍需单独审查。

## 升级模块的中文化边界

- `selfupdate/`：发布检查、下载、SHA256 校验、解压、暂存、启动验证、自动/手动回滚的进度和错误文本使用简体中文，保留 JSON 字段、阶段值、HTTP 状态码和更新资产的名字。
- `server/update.go`：一键升级接口及 SSE 直接展示的状态文案使用简体中文；历史保留的 `ARTEX_SELFUPDATE_SMOKE`、`artex-selfupdate` User-Agent 属于兼容/技术标识，不进行批量更名。
- 对应测试位于 `selfupdate/selfupdate_localized_test.go` 和 `server/update_localized_test.go`，检查中文展示及缺失发布包、错误状态和更新期间禁止回滚等路径。
- **发布安全性独立评估**：只调整可见文案不证明真实 Release 下载、自更新/回滚端到端已经通过；正式镜像发布仍需供应链审计。

## 回归验证

~~~bash
cd web && npm run check && npm run build:static && cd ..
python3 -I scripts/check-i18n-keys.py
python3 -I scripts/check-web-zh.py
python3 -I scripts/check-doc-links.py
~~~

这些校验不能证明运行时所有韩语或英文都已清除。通知模块、历史数据和外部模型仍可能产生非中文内容，需逐条测验。对于较长的 AI 输出，中文语言约束不是绝对保证，结论还要与原始证据核对。

相关：[Web 子项目](../web/README.md) · [开发与测试](DEVELOPMENT.md)。

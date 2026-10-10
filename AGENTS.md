# Kexra · AI 编程代理协作规范

本文件适用于 Codex、AI 编程助手及参与维护的开发者。目标是让每次提交可验证、可审计、可回退。

## 项目与现状

- 仓库：`ForceMind/Kexra`，默认简体中文品牌 **Kexra（科巡）**。
- 代码：Go + PostgreSQL 后端、Next.js/React 前端，包含 Agent 编排、MCP/Skills、检测规则与证据管理。
- **品牌分支已完成源文件更名**：Go module 使用 `github.com/ForceMind/Kexra`，入口为 `cmd/kexra`；但 Go 1.26 编译、前端、数据库集成和发行流程仍需真实环境验证，不能据此宣布已发布。
- 现有代码基于 AGPL-3.0 等第三方许可，不得移除对应版权与原许可文本。参阅 [第三方声明](THIRD_PARTY_NOTICES.md)。

## 工作规则

1. 阅读 [文档索引](docs/README.md)、[开发与测试](docs/DEVELOPMENT.md)、[迁移状态](docs/MIGRATION.md)，再开始修改。
2. 不对未经明确授权的网络、系统或第三方资产执行主动测试、扫描、攻击或数据收集。
3. 保留现有 `ARTEX_*` 环境变量、数据库名称、Cookie 和历史路径的兼容性，除非已有可靠迁移方案与回归测试。
4. 不要批量修改 `detections/` 下的历史 IOC、签名、测试事件、流量和证据；品牌与检测内容必须分开管理。
5. 不提交 `.env`、`config.json`、JWT 密钥、流量数据库、客户信息、真实主机列表、云凭据和 API 密钥。
6. 用户可见文字以简体中文为默认；优先通过 next-intl 文案键翻译，不改变命令、URL、代码、协议字段及日志/证据原文。
7. 修改接口、配置、数据库或发行流程时先确定契约；修复同模块测试，避免一次性全仓库格式化。
8. 未验收的自动更新、Docker 发布和生产环境动作保持关闭，不修改当前运行中的数据库或服务。
9. 用新分支和 PR 提交小范围变更，明确写出已完成/未完成，不以通过一个阶段的构建替代全部验收。

## 测试门槛

~~~bash
go build ./...
go vet ./...
go test ./... -count=1
cd web && npm ci && npm run check && npm run build:static && cd ..
python3 -I scripts/check-i18n-keys.py
python3 -I scripts/check-web-zh.py
python3 -I scripts/check-doc-links.py
~~~

这些命令须在依赖与 `web/out` 已生成的环境下执行。数据库集成测试由 CI `ci.yml` 中的 PostgreSQL jobs 覆盖；检测规则由独立工作流覆盖。

如果 CI 因账户/平台限制在 runner 初始化阶段失败，应明确标记为“未运行”，不得写成“通过”或“代码失败”。

## 每次交接必须写明

- 当前分支、提交 SHA、差异范围、已批准的任务边界；
- 具体改动文件、风险、兼容路径与是否需要数据迁移；
- 测试命令、结果、失败原因和未跑的门禁；
- 尚未完成的事项、恢复方法与后续工作入口。

相关：[贡献指南](CONTRIBUTING.md) · [维护者指南](MAINTAINING.md) · [清理清单](docs/PROJECT_CLEANUP.md)。

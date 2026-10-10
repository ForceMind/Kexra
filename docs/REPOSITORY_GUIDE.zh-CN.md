# Kexra 仓库与工程维护总览

本文件是项目的工程入口；更细的指导以 [文档中心](README.md) 中的专题文档为准。

## 项目身份

- 当前仓库：**[ForceMind/Kexra](https://github.com/ForceMind/Kexra)**（已更名）。
- 目标：AI 多智能体协作、授权安全评估、资产/证据/报告工作台。
- 管理：以自己的 PR、Roadmap、Release 和 CI 为准，非旧上游的自动同步 fork。
- 许可：既有代码的 AGPL-3.0 与前端 MIT 适用条款保留，见 [第三方声明](../THIRD_PARTY_NOTICES.md)。
- 当前状态：品牌迁移仍在 Draft PR #1，未完成正式发布或生产安全验收。

## 主要模块与路线

| 模块 | 入口 |
| --- | --- |
| Go 启动 / API | `cmd/artex/`、`server/` |
| Agent 调度 | `agent/`、`llmpool/` |
| 数据及证据 | `db/`、`evidence/`、`report/` |
| 浏览器 UI | `web/`、`web/messages/zh.json` |
| 工具与集成 | `mcphttp/`、`skills/`、`notify/` |
| 流量和风险审查 | `traffic/`、`guard/`、`intercept/` |
| 安全检测规则 | `detections/`、`detections/tests/` |
| 代码发布 | `build.sh`、`Dockerfile`、`selfupdate/` |

细节见 [系统架构](ARCHITECTURE.md)。

## 使用流程与限制

1. 阅读 [安全边界](SECURITY_MODEL.md)，仅选择自有/授权隔离环境。
2. 依照 [安装指南](INSTALLATION.md) 设置工具链和独立数据库。
3. 根据 [配置文件](CONFIGURATION.md) 确定实际字段优先级。
4. 按 [开发与测试](DEVELOPMENT.md) 完成 Go/Web/文档/检测规则质量门禁。
5. 新分支提交 PR，记录已执行的验收、兼容性与回滚说明。
6. 发布前依照 [部署指南](DEPLOYMENT.md) 完成网络、镜像与自动更新核验。

## 容易误判的迁移事项

- `ForceMind/Artex` **已经更名**，不需要重复操作；本地 Git 需更新 remote。
- `go.mod` 和 `cmd/artex` 仍有历史命名，属代码迁移任务，不是合法可删的旧文件。
- 旧 `ARTEX_*` 配置、数据库用户/库名、Cookie 与 JWT 密钥可能仍被现有部署使用。
- `detections/` 内旧名称为技术指纹/回归样例的一部分，不能作为品牌批量替换。
- 旧 README 的 Docker/Releases/Demo 入口不是 Kexra 当前可用的安装源。
- PR #1 的完整 CI 结果和首次发布仍需验证。

参考：[迁移记录](MIGRATION.md) · [旧文件清理](PROJECT_CLEANUP.md) · [更新日志](../CHANGELOG.md)。

# Kexra 更新记录

Kexra 由 ForceMind 独立维护。以下记录不把继承的 ARTEX 历史版本冒称为 Kexra 自有发布。

## [未发布] — 2026-10-14，独立品牌基线已合入 main

- **[PR #1](https://github.com/ForceMind/Kexra/pull/1) 已合并**（合并提交 `9a3fd28e`）：Kexra 品牌、Go module `github.com/ForceMind/Kexra`、CLI `cmd/kexra/`、Web SVG、登录兼容、启动/构建/发布命名。
- 移除旧项目推广二维码、无效跳转 README、冗余占位与旧 Logo 图片；保留 AGPL-3.0、前端 MIT、原始来源及检测证据。
- 中文文档、配置、开发维护指南更新；保留现有 `ARTEX_*`、旧数据库及 Cookie 兼容，避免破坏已有数据。
- 验收记录：[Go](https://github.com/ForceMind/Kexra/actions/runs/38267626025)、[Web/文档](https://github.com/ForceMind/Kexra/actions/runs/38267953075)、[PostgreSQL](https://github.com/ForceMind/Kexra/actions/runs/38268254170)、[检测规则](https://github.com/ForceMind/Kexra/actions/runs/38269136070)、[跨平台打包](https://github.com/ForceMind/Kexra/actions/runs/38269849980)、[本地运行与登录](https://github.com/ForceMind/Kexra/actions/runs/38455771805)。

### 2026-10-19 开发增量

- 强化升级标记的语义校验、完整写入与换装前持久化；无法保存启动次数时立即尝试回滚。
- 新增缺陷复现与恢复回归测试，建立开发路线、当前状态与发布清单。
- 本次与历史验证范围分开记录，详见 [状态与交接](docs/WORK_STATUS.md)。尚无正式 Release。

### 仍需完成

- Docker 生产旧实例升级恢复、arm64 与发布供应链审计；隔离 amd64 镜像、会话和数据库恢复已有历史验证。
- 首个 Kexra 正式 Release 与自更新、回滚的端到端验收（目前没有发布）。
- 进一步中文化部分后端、通知、历史文案，同时保留技术证据与协议兼容字段。
- 公开仓库历史和依赖的安全审计。

## 原始代码导入

最初的可验证导入版本详见 [来源记录](docs/IMPORT_PROVENANCE.md)；继承的历史日志仅作参考，不属于 Kexra 自有发行版本。

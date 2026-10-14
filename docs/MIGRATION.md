# Kexra 迁移状态与后续兼容计划

## 已完成（2026-10-14）

- 仓库已公开、更名为 [ForceMind/Kexra](https://github.com/ForceMind/Kexra)。
- 独立品牌 [PR #1](https://github.com/ForceMind/Kexra/pull/1) 已合并至 `main`（提交 `9a3fd28e`）。
- Go 模块 `github.com/ForceMind/Kexra`、CLI `cmd/kexra/`、Windows/Linux 启动与跨平台包名称、Web Logo/界面品牌已迁移。
- 旧 `cmd/artex/`、无效推广二维码、旧图标及重复历史跳转页已清理，原版权和许可证保留。
- 旧管理员用户名 `ARTEX`、旧 `artex_token` Cookie、数据库/环境变量 `ARTEX_*` 仍保留兼容，防止旧安装和已有数据失效。
- 原始六套 GitHub Actions 工作流保持手动触发，不自动执行。

## 验收证据

| 范围 | 结果 | GitHub Actions |
| --- | --- | --- |
| Go 1.26 构建、vet、单测 | 成功 | [S1（Go job）](https://github.com/ForceMind/Kexra/actions/runs/38267626025) |
| 中文前端、Biome、内部文档链接 | 成功 | [S2](https://github.com/ForceMind/Kexra/actions/runs/38267953075) |
| 六模块 PostgreSQL 16 集成 | 成功 | [S3](https://github.com/ForceMind/Kexra/actions/runs/38268254170) |
| 合成检测规则、IOC、MISP 等回归 | 成功 | [S4b](https://github.com/ForceMind/Kexra/actions/runs/38269136070) |
| 5 平台 zip、SHA256、Compose 配置 | 成功 | [S5b](https://github.com/ForceMind/Kexra/actions/runs/38269849980) |
| 本地服务健康、初始化、登录兼容、JWT 重启 | 成功 | [S6](https://github.com/ForceMind/Kexra/actions/runs/38455771805) |

## 中文化增量（2026-10-14）

[PR #4](https://github.com/ForceMind/Kexra/pull/4) 已合并：认证、配置、CLI 及六类通知渠道的用户可见文本改为简体中文，对应 Go 全项目构建、vet 与单元测试通过。其他后端模块仍有遗留韩语，详见 Issue #3，不代表全产品中文化完成。

## 尚未完成（独立于本次代码合并）

1. **真实 Docker 镜像构建和启动验收**：默认仅本机监听与 JWT 私有持久卷已在开发分支修复，仍需在真实容器中复核、备份恢复、密钥迁移及完整启动。
2. **首次正式 Kexra Release**：验证 v* 标签、私有/公开下载、SHA256、软件自更新与回滚，在确认发布目标后才启用。
3. **后端中文化余项**：PR #4 已完成认证、通知及启动信息一批中文化；其他服务端及自动更新仍有历史韩语，须继续逐模块审查并保留证据/技术原文。
4. **安全审计**：真实公网部署前审查认证、模型和工具权限、出站网络限制及第三方依赖许可与 Git 历史。

不要因更名删除 `artex` 数据库、`pgdata`、`data/`、JWT 密钥、旧 Cookie 兼容或 `detections/` 中的历史指标。当前未发布经过端到端验收的正式镜像，也不应将这份迁移合并等同于可安全部署在公网上。

## 本地仓库 URL 更新

```bash
git remote set-url origin https://github.com/ForceMind/Kexra.git
git fetch origin
```

更多信息：[安装](INSTALLATION.md) · [部署](DEPLOYMENT.md) · [清理记录](PROJECT_CLEANUP.md) · [来源记录](IMPORT_PROVENANCE.md)。

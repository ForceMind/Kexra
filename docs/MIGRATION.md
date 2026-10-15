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

## Docker 隔离验收与安全修复（2026-10-14）

[PR #5](https://github.com/ForceMind/Kexra/pull/5) 已合并，默认 8787/8788 仅宿主机回环监听、JWT 使用不可浏览的独立持久卷，保留旧路径迁移与异常密钥拒绝静默轮换。隔离的 [Docker 镜像/容器/会话和数据库恢复验收](https://github.com/ForceMind/Kexra/actions/runs/38504238870) 全部成功，未发布镜像也未面向公网开放管理服务。

**注意**：真实生产旧容器可写层中的 `/app/jwt.key` 必须在删除旧容器前单独备份。验收记录只覆盖隔离测试环境，不等于生产环境和供应链审计通过。

## 自动更新标记完整性保护（待验收）

自更新启动路径新增两道失败保护：

- 暂存完成后如不能写入升级标记，立即删除待安装的二进制和校验和，并将更新报告为失败；不得重启进入无自动回滚保护的状态。
- Bootstrap 如果发现待安装文件但没有有效升级标记，拒绝换装，保留当前版本，并丢弃暂存文件。

`selfupdate/stage_safety_test.go` 在临时目录内模拟成功换装、标记写入失败、丢失和损坏；`selfupdate/selfupdate_test.go` 保持正确校验和及异常文件回归。该改动只增强本地更新状态保护，仍**不能替代**真实 GitHub Release 下载/更新/多次启动失败自动回滚端到端演练。

## 尚未完成（独立于本次代码合并）

1. **Docker 发布供应链与生产部署**：隔离 Docker 实际构建、启动、会话和数据库恢复已通过；NodeSource、npm 浮动依赖、镜像校验与已有生产实例升级恢复流程尚未完成审查，不能视为公网生产可用。
2. **首次正式 Kexra Release**：验证 v* 标签、私有/公开下载、SHA256、软件自更新与回滚，在确认发布目标后才启用。
3. **后端中文化余项**：PR #4 已完成认证、通知与启动信息。升级模块中文化正在分支独立实施；其他服务端模块、第三方工具输出仍需审查。保留证据原文、技术协议与兼容标识。
4. **安全审计**：真实公网部署前审查认证、模型和工具权限、出站网络限制及第三方依赖许可与 Git 历史。

不要因更名删除 `artex` 数据库、`pgdata`、`data/`、JWT 密钥、旧 Cookie 兼容或 `detections/` 中的历史指标。当前未发布经过端到端验收的正式镜像，也不应将这份迁移合并等同于可安全部署在公网上。

## 本地仓库 URL 更新

```bash
git remote set-url origin https://github.com/ForceMind/Kexra.git
git fetch origin
```

更多信息：[安装](INSTALLATION.md) · [部署](DEPLOYMENT.md) · [清理记录](PROJECT_CLEANUP.md) · [来源记录](IMPORT_PROVENANCE.md)。

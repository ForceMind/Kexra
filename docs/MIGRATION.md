# 从 Artex 仓库迁移到 Kexra

## 已确认事实

- GitHub 仓库名称已由 `ForceMind/Artex` 修改为 **`ForceMind/Kexra`**。
- `main` 尚保持此前中文化基线；新的品牌迁移代码位于 [Draft PR #1](https://github.com/ForceMind/Kexra/pull/1)。
- Web 产品名称、图标、登录名及若干启动/发布标识已在该 PR 中更新。
- **并非已完成独立发布的产品**：品牌迁移尚未合并，测试记录尚不充分。
- 原主体许可证、前端 MIT 组件版权、导入校验和原始数据必须保留。

## 本地仓库地址更新

~~~bash
git remote -v
git remote set-url origin https://github.com/ForceMind/Kexra.git
git remote -v
git fetch origin
~~~

GitHub 的旧 URL 重定向是兼容机制，**不应作为长期配置依赖**；CI、Webhook、部署系统、README 链接等均应改成新地址。

## 代码更名尚未闭环

| 事项 | 目前情况 | 处理方式 |
| --- | --- | --- |
| GitHub 仓库名 | 已改为 Kexra | 无需再次更名 |
| Web 品牌及 SVG | 已在品牌 PR 中修改 | 要求前端构建和视觉验收 |
| Go module | 品牌分支已改为 `github.com/ForceMind/Kexra` | 已同步内部 Go import，完整 Go 编译及依赖验证仍待执行 |
| Go CLI | 品牌分支已迁移至 `cmd/kexra` | 入口及测试文件迁移完成，仍待跨平台验证 |
| 构建产物 | 部分改为 `kexra` | 同步 Dockerfile、安装、更新与打包路径 |
| 环境变量 | `ARTEX_*` 兼容字段仍使用 | 不得批量移除；必要时引入新别名并测试 |
| 数据库 | 仍可能叫 `artex` | 无须因改品牌而改库名；不得重置历史数据 |
| 登录与 Cookie | 新 `Kexra` 与旧 `ARTEX` 兼容过渡 | 验证真实登录、会话与刷新行为 |
| 自动更新 | 源设为 `ForceMind/Kexra` | 首次发布、私有仓库权限、签名/校验和回滚尚待验收 |
| Docker | `kexra:local` 是本地构建标签 | 不存在可宣称的官方镜像 |

## 推荐迁移顺序

1. 已在品牌分支完成 Go module、内部导入和 CLI 命名迁移；下一步必须确认 `go build ./...` 与脚本产物实际成功。
2. 修复旧安装脚本、端口提示及 Go/前端本地开发流程。
3. 核验现有数据库和 JWT 密钥迁移，不进行任何自动清空。
4. 执行 Go + PostgreSQL、Web、文档、检测规则的全部 CI。
5. 审查安全边界与部署网络，再考虑合并 PR。
6. 通过可重复验收后，才建立首个 Kexra 版本标签与分发渠道。

## 数据迁移禁止事项

不要因“删除旧项目痕迹”而删除旧用户、`artex` 数据库、`pgdata`、`data/`、JWT 密钥、日志证据或新旧 Cookie 兼容逻辑。这些包含真实状态和使用中的协议信息。

来源、版权与合规说明： [导入记录](IMPORT_PROVENANCE.md) · [第三方许可](../THIRD_PARTY_NOTICES.md)。

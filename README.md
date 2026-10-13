# Kexra · 科巡

**面向授权安全评估的 AI 多智能体任务协作与证据管理平台**

Kexra 将目标管理、AI 任务规划、并行执行、资产与探索图谱、证据关联、人工审查和报告整理集中在一个 Web 工作台。前端默认使用简体中文；项目由 ForceMind 独立维护。

> **使用范围**：仅用于自有资产、书面授权的测试环境或隔离实验室。Kexra 的审批与拦截机制不等同于可信隔离边界，不应将其直接开放到公网。
>
> **项目状态**：已完成仓库重命名为 `ForceMind/Kexra`。[PR #1](https://github.com/ForceMind/Kexra/pull/1) 已合并至 `main`。Go、Web、PostgreSQL 集成、合成检测规则、五平台打包及隔离环境登录/重启验证均已通过；**尚无经过完整 Docker 部署与更新回滚验收的正式 Release 或官方镜像**。

## 功能概览

| 能力 | 说明 |
| --- | --- |
| 多 Agent 协作 | 对任务进行规划、调度、执行，并保存过程与结果 |
| 资产与探索图谱 | 关联资产、事实、任务节点及已有发现 |
| 证据与报告 | 整理安全发现、关联原始证据，生成报告并跟踪复测 |
| 人机协作 | 查看实时活动、交互指导、审批记录与任务状态 |
| 扩展能力 | 模型配置、MCP、Skills、通知和流量记录 |
| 中文界面 | 基于 next-intl 的简体中文默认构建，保留韩语兼容资源 |

**技术组成**：Go 1.26.x、Next.js 16 / React 19、PostgreSQL 16、TypeScript、MCP / Skills。有关实际依赖版本，以 `go.mod`、`web/package.json` 和 CI 定义为准。

## 验证与 CI 状态

为避免私有仓库 GitHub Actions 额度持续消耗，全部工作流已暂时切换为手动触发（`workflow_dispatch`）。当前迁移通过静态一致性检查准备验收，完整 Go 1.26、前端和数据库 CI 尚未运行；不要将手动模式理解为测试通过。

本地先运行 `python3 -I scripts/check-kexra-migration.py` 检查更名契约，再执行下方构建命令。

## 迁移验收记录

- [Go 构建/单测](https://github.com/ForceMind/Kexra/actions/runs/38267626025)（Go job 通过，第一轮文档断链随后修复）
- [前端中文构建与文档](https://github.com/ForceMind/Kexra/actions/runs/38267953075)
- [PostgreSQL 16 集成测试](https://github.com/ForceMind/Kexra/actions/runs/38268254170)
- [防御检测合成规则](https://github.com/ForceMind/Kexra/actions/runs/38269136070)
- [五平台构建包与校验](https://github.com/ForceMind/Kexra/actions/runs/38269849980)
- [本地服务启动、登录兼容与 JWT 持久化](https://github.com/ForceMind/Kexra/actions/runs/38455771805)

六套正式 GitHub Actions 工作流目前均为**手动触发**。仅在人工核实运行成本与发布目标后恢复自动执行。

## 开始使用

建议从**隔离的本地开发环境**入手，不要直接执行未知测试目标。

~~~bash
git clone https://github.com/ForceMind/Kexra.git
cd Kexra

# 前端依赖与基础检查
cd web
npm ci
npm run check
npm run build:static
cd ..

# Go 源码构建与单元测试
go build ./...
go test ./... -count=1
~~~

以上命令仅完成**源码构建与校验**，不会配置数据库，也不会证明产品部署、测试结果或安全控制已完成验收。主分支 Go 入口已迁移至 `cmd/kexra`，模块路径为 `github.com/ForceMind/Kexra`。源码编译与隔离本地启动已经通过测试，但公网部署、完整 Docker 镜像和真实版本更新/回滚仍待验收。

完整的数据库配置、本地运行方式、镜像前置条件和限制见 [安装与启动](docs/INSTALLATION.md)。

## 文档导航

- [文档总览](docs/README.md)：按使用角色定位资料
- [安装与启动](docs/INSTALLATION.md) · [配置说明](docs/CONFIGURATION.md)
- [系统架构](docs/ARCHITECTURE.md) · [开发与测试](docs/DEVELOPMENT.md)
- [安全边界](docs/SECURITY_MODEL.md) · [安全漏洞报告政策](SECURITY.md)
- [仓库迁移与兼容性](docs/MIGRATION.md) · [旧文件清理清单](docs/PROJECT_CLEANUP.md)
- [贡献指南](CONTRIBUTING.md) · [维护者指南](MAINTAINING.md) · [更新记录](CHANGELOG.md)
- [English](README.en.md) · [第三方许可与来源](THIRD_PARTY_NOTICES.md)

## 项目身份与许可

Kexra 是**基于已有开源代码持续维护、独立品牌化的项目**，并非全部代码从零原创。主体的 [AGPL-3.0 许可证](LICENSE)、前端所用第三方材料的 [MIT 许可证](web/LICENSE)、原始版权与来源记录必须保留。详情见 [第三方声明](THIRD_PARTY_NOTICES.md) 和 [可验证的导入来源](docs/IMPORT_PROVENANCE.md)。

重命名产品并不改变所继承代码的法律义务；既不表示原作者参与维护，也不代表原作者认可当前产品。

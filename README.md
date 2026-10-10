# Kexra · 科巡

**AI 多智能体安全评估与证据管理工作台**

Kexra 将任务规划、多 Agent 协作、可追溯证据、资产关系、人工审批和报告归档整合在一个工作台。面向拥有明确授权的测试环境；不支持对未知或未获授权的第三方系统开展自动测试。

> **品牌迁移状态：** 代码正在由 Artex 整理为 Kexra。目标仓库名 `ForceMind/Kexra`，在 GitHub 管理员执行仓库重命名之前，实际 GitHub 地址仍为 [ForceMind/Artex](https://github.com/ForceMind/Artex)。本仓库未发布可直接拉取的 Kexra 容器镜像或正式版二进制。

## 核心模块

| 模块 | 作用 |
| --- | --- |
| Agent 协作 | 规划者、执行者、主交互 Agent 的任务编排 |
| 探索与资产图谱 | 持久化记录资产、发现、事实和任务依赖 |
| 测试记录与报告 | 证据关联、漏洞审核、Markdown 报告与复测 |
| 管理与可视化 | Go 后端 + Next.js/React 前端、中文 UI |
| 模型与扩展 | 模型接口配置、MCP 和 Skills 扩展 |
| 风险控制 | 审批与拦截功能；需要独立安全审计，不能当作可靠的强制沙箱 |

## 开发与构建

需要 Go 1.26.x、Node.js 22、npm 与 PostgreSQL 16。源码构建不等于完成安全部署验收。

```bash
git clone https://github.com/ForceMind/Artex.git
cd Artex
cd web && npm ci && npm run build:static && cd ..
go build ./...
go test ./... -count=1
```

正在迁移的命令入口与发布名称以实际构建提交为准。详细开发约定见 [仓库维护指南](docs/REPOSITORY_GUIDE.zh-CN.md) 和 [Codex/Agent 规范](AGENTS.md)。

## 来源与许可

**Kexra 是在现有开源软件基础上持续开发的独立维护版本，不是全部代码从零原创。** 原始代码来源与版本固定记录在 [导入来源说明](docs/IMPORT_PROVENANCE.md)。主体仍按根目录 [AGPL-3.0 许可证](LICENSE) 管理；前端所含第三方 MIT 代码保留 [原许可](web/LICENSE)，版权及其他依赖声明见 [第三方声明](THIRD_PARTY_NOTICES.md)。这些法定信息不等于 Kexra 产品需要沿用原来的品牌设计。

## 安全与使用范围

仅在拥有明确授权的隔离环境中使用。未经审计不得面向公网部署；现有审批与拦截规则不能代替系统级隔离。原项目的某些检测特征、接口、数据库字段和环境变量可能为兼容旧数据而保留历史命名；不能自行把它们当成当前产品的品牌信息。

## 其他资料

- [中文化说明](docs/LOCALIZATION.zh-CN.md)
- [历史韩语说明](README.ko.md)
- [维护与品牌迁移记录](docs/KEXRA_BRAND_MIGRATION.md)

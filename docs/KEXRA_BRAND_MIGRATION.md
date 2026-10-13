# Kexra 品牌迁移记录

**2026-10-10 已完成的仓库管理变更：`ForceMind/Artex` → `ForceMind/Kexra`。**

本文件仅保存品牌更名的决策与兼容原则；当前实施状态以 [统一迁移说明](MIGRATION.md) 为准。

## 已确定的产品定位

- 产品名称：**Kexra（科巡）**。
- 面向中文用户的名称与界面统一使用 Kexra，原有旧品牌的营销素材不再作为当前产品宣传。
- 业务目标：AI 多智能体协作的授权安全评估、资产与证据管理。
- 不把旧项目代码冒称为全部从零原创；保留适用的版权、许可证及来源信息。

## 重要兼容决策

1. 不因仓库更名重置用户、数据库、运行数据或 JWT 密钥。
2. 对旧 `ARTEX_*` 环境变量和历史登录身份采用先兼容、后迁移的方式。
3. `main` 已迁移 Go module 和 `cmd/kexra`，旧 `cmd/artex` 已删除；Go、Web、PostgreSQL、检测规则、五平台打包和本地运行已通过，正式 Release 仍需端到端验收。
4. 原始安全检测规则和证据不进行“品牌文字替换”。
5. 自动更新必须指向新仓库、校验新发行包；正式发布前不启用。

## 下一步

品牌实现已通过 [PR #1](https://github.com/ForceMind/Kexra/pull/1) 合并。后续 Docker、自动更新、中文化余项和安全审查见 [迁移说明](MIGRATION.md) 与 [清理清单](PROJECT_CLEANUP.md)。

旧仓库本地克隆可以运行：

~~~bash
git remote set-url origin https://github.com/ForceMind/Kexra.git
~~~

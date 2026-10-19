# Kexra 文档中心

文档以**实际仓库代码和可验证配置**为准，区分「已经实现」「尚待验收」「历史资料」。默认阅读语言为简体中文。

## 选择你的入口

| 角色 / 目标 | 阅读顺序 |
| --- | --- |
| 初次了解项目 | [主页](../README.md) → [架构](ARCHITECTURE.md) → [安全边界](SECURITY_MODEL.md) |
| 本地开发 | [安装](INSTALLATION.md) → [配置](CONFIGURATION.md) → [开发与测试](DEVELOPMENT.md) |
| 容器与发布 | [部署说明](DEPLOYMENT.md) → [维护指南](../MAINTAINING.md) |
| 修改界面或文案 | [中文化约定](LOCALIZATION.zh-CN.md) → [Web 子项目](../web/README.md) |
| 维护 Agent 与测试 | [开发与测试](DEVELOPMENT.md) → [AGENTS](../AGENTS.md) |
| 安全审查 | [安全政策](../SECURITY.md) → [安全边界](SECURITY_MODEL.md) |
| 继续独立品牌化 | [迁移状态](MIGRATION.md) → [旧文件清理](PROJECT_CLEANUP.md) |
| 排查原始代码与法律来源 | [导入来源](IMPORT_PROVENANCE.md) → [第三方声明](../THIRD_PARTY_NOTICES.md) |

## 专题资料

- [开发路线](ROADMAP.md) · [当前状态与交接](WORK_STATUS.md) · [发布验收清单](RELEASE_CHECKLIST.md)
- [依赖安全审计与复现](DEPENDENCY_SECURITY.md)

- [检测规则维护](../detections/MAINTENANCE.zh-CN.md)；原有 [规则资料](../detections/README.md) 中的历史技术特征不能为了改品牌而擅自替换
- [Side Question 技术说明](../sidequestion/README.zh.md)；偏工程实现的历史资料，可能仍使用原模块命名
- [漏洞与流量证据的原始设计说明](finding-traffic-evidence-zh.md)；属于继承的参考资料，不等于新品牌功能已重新验收
- [原始版本导入记录](IMPORT_PROVENANCE.md)；固定校验信息不因仓库更名而改写

## 当前版本说明

- GitHub 仓库已改为 `ForceMind/Kexra`。
- 品牌迁移 [PR #1](https://github.com/ForceMind/Kexra/pull/1) 已合并至 `main`；Go module 与 CLI 均已改为 Kexra，数据库和部分 `ARTEX_*` 配置名为保留数据兼容而继续存在。
- 仓库不提供已经验证的官方镜像，也不能以既有上游 Demo 或截图作为 Kexra 的当前运行证明。
- 如文档与代码不符，优先核对具体提交、测试结果，然后修订文档。

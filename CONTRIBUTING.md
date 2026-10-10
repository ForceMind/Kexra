# 为 Kexra 贡献代码

Kexra 是由 ForceMind 独立维护的 AI 多智能体安全评估平台。欢迎修复缺陷、改善中文体验、补充测试与文档，但所有变更必须遵守授权使用和第三方许可。

## 参与前阅读

- [文档中心](docs/README.md) 与 [系统架构](docs/ARCHITECTURE.md)
- [安全政策](SECURITY.md) 与 [安全边界](docs/SECURITY_MODEL.md)
- [开发与测试](docs/DEVELOPMENT.md)
- [第三方版权与许可](THIRD_PARTY_NOTICES.md)

## 推荐流程

1. 先搜索已有 Issue / PR，再说明当前问题、预期行为和对应文件。
2. 从当前基线建立分支；每个 PR 尽量只解决一个问题。
3. 修改时保留旧数据与兼容契约，必要时先添加测试证明现有行为。
4. 提交前完成源码构建、目标包测试、格式检查、文档链接检查。
5. PR 中提供复现步骤、风险、已测试项和未测试项；不要填写虚假的通过结果。

## 本地验证

~~~bash
go build ./...
go vet ./...
go test ./... -count=1
cd web && npm ci && npm run check && npm run build:static && cd ..
python3 -I scripts/check-doc-links.py
~~~

改变中文界面时，另外运行 `python3 -I scripts/check-i18n-keys.py` 与 `python3 -I scripts/check-web-zh.py`（后者需要先生成 `web/out/`）。

## 行为和质量要求

- 仅在自有或明确授权的隔离环境进行动态测试；禁止把第三方目标、真实流量、凭据和隐私数据上传到仓库。
- Agent 内部工具命令、载荷、日志及证据原文不应翻译；用户可见文案默认用简体中文。
- 安全测试样例的原始检测签名和证据语义必须准确，禁止为了“去旧品牌”批量替换。
- 不随意扩大默认权限、关闭审批、删除测试或降低安全门禁。
- 保留适用的 AGPL-3.0 / MIT 许可和已有版权声明。
- 代码变更不代表发行完成；发布和自动更新需另行验收。

## Pull Request 说明

应写清楚：问题背景、解决方案、兼容影响、实际运行的测试、截图/日志（脱敏）、安全和数据风险、失败与待办。可参考仓库内的 [PR 模板](.github/PULL_REQUEST_TEMPLATE.md)。

发现 Kexra **自身**的严重安全漏洞，请勿公开披露利用细节，遵循 [安全报告政策](SECURITY.md)。社区沟通遵守 [行为准则](CODE_OF_CONDUCT.md)。

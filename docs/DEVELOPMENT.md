# Kexra 开发与测试

## 协作原则

- `main` 保留可回退版本；所有变更先在明确命名的分支和 PR 上提交。
- 先把“可运行性、兼容性与安全约束”作为门槛，再新增功能和优化视觉。
- 源码相关变更必须配套测试；涉及新字段时描述数据迁移与旧客户端兼容性。
- 仅在本地隔离环境或书面授权的环境执行动态安全测试。
- AI Agent/自动化不能在没有明确目标授权的情况下自动触发外网探测。

## 项目目录

| 目录 | 代码职责 |
| --- | --- |
| `cmd/kexra/` | `main` 中的 Go CLI 入口，Go 1.26 构建和测试已验证 |
| `server/` | HTTP API、会话和认证 |
| `agent/` | Agent 计划、执行、上下文与工具调用 |
| `db/`、`evidence/`、`report/` | 数据、证据和报告 |
| `llmpool/`、`llmrec/` | 模型调用与记录 |
| `web/` | Next.js 16、React 19、TypeScript、next-intl |
| `notify/`、`traffic/`、`intercept/` | 通知、流量与审查 |
| `detections/` | 防御规则与独立测试 |

## 快速验证命令

推荐构建工具链为 Go **1.26.9**（CI/Release 已固定）；Go 1.27 系列至少为 **1.27.2**。依赖漏洞结果、复现方法和未导入 OpenPGP 的保留告警见 [安全审计](DEPENDENCY_SECURITY.md)。

更名与许可证的低成本静态检查：`python3 -I scripts/check-kexra-migration.py`（不访问网络、不调用 GitHub Actions）。

~~~bash
# 根目录（不依赖本地 DB 的测试可能跳过 DB 集成路径）
go build ./...
go vet ./...
go test ./... -count=1

# 前端
cd web
npm ci
npm run check
npm run build:static
cd ..

# 中文资源与文档
python3 -I scripts/check-i18n-keys.py
python3 -I scripts/check-web-zh.py
python3 -I scripts/check-doc-links.py
python3 -I scripts/check-web-vendor.py
~~~

`check-web-zh.py` 需要先生成 `web/out`；其静态 HTML 检查**不能证明**运行时 API/Mock/Agent 输出已全部中文化。

**GitHub Actions 节省额度状态**：当前 `main` 的 CI、文档、前端、检测规则、外链周检及 Release 工作流仍为手动触发（`workflow_dispatch`）；不会因 push、PR 或定时任务自动运行。仓库已公开，但自动门禁的恢复需单独检查运行费用、安全边界与范围。

CI 通过 `.github/workflows/ci.yml`、`web.yml`、`docs.yml` 和 `detections.yml` 组织。使用 PostgreSQL 16 的集成测试 job 按包隔离，不能以无数据库的普通 `go test` 代替它们。

检测规则验证依赖测试镜像/工具，请参阅 [detections/tests/README.md](../detections/tests/README.md)，避免修改原始指纹或事件样例来“让测试通过”。

## 版本与提交

建议以 `feat:`、`fix:`、`docs:`、`test:`、`refactor:`、`chore:` 等语义开头，并在 PR 记录：

1. 背景、变更范围与不变的兼容协议；
2. 改动文件与截图（如影响 UI）；
3. 实际运行的命令及测试通过/失败/未执行；
4. 安全风险、数据迁移与回退方式；
5. 未完成事项与后续约束。

当前独立品牌工作的事实与剩余阻塞见 [迁移说明](MIGRATION.md)。

## 本地化注意事项

用户可见文本优先使用 `web/messages/zh.json` 的键；不得修改命令、payload、请求响应、证据原文。韩语资源仅作为兼容/历史参考，不能继续把中文界面生成过程约束为“不能出现汉字”。完整规则见 [本地化说明](LOCALIZATION.zh-CN.md)。

## 需要重点审查

- 已迁移的 `cmd/kexra` 及 `github.com/ForceMind/Kexra` 内部导入路径是否通过完整构建；
- Release 工作流、Dockerfile、启动脚本和二进制文件名；
- 登录名新旧兼容、Cookie 过渡、JWT 密钥位置；
- `config.example.json` 与 Docker 的不同默认端口；
- 各种 `ARTEX_*` 字段是否只作兼容，而非可立即删除；
- GitHub Actions 因平台原因在执行测试前失败时，不应将状态写成“测试通过”。

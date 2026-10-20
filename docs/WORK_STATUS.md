# Kexra 当前状态与交接

更新时间：2026-10-20，北京时间。此文件记录事实与恢复入口；长期任务见 [路线](ROADMAP.md)，发布判断见 [发布清单](RELEASE_CHECKLIST.md)。

## 本次接管基线

- 已回读 GitHub：`main` 为 `e4e5285698b075ccd8edba8cafcdb361b8d501c2`，与 10 月 15 日记录一致。
- 接管时没有开放 PR，也没有 Release；Issue [#2](https://github.com/ForceMind/Kexra/issues/2)（发布验收）、[#3](https://github.com/ForceMind/Kexra/issues/3)（中文化）仍开放。
- 六套正式工作流仅手动触发；未创建提醒、定时任务、发布标签或镜像。
- 最近 Docker 工具链和更新标记专项运行分别为 [38737617057](https://github.com/ForceMind/Kexra/actions/runs/38737617057)、[38737705235](https://github.com/ForceMind/Kexra/actions/runs/38737705235)，均成功。更早失败记录不改写。

## 已合并并验证：升级标记可靠性

PR [#9](https://github.com/ForceMind/Kexra/pull/9) 已合并，合并提交 `30783bfcfbf35bbbb8d73b8432fc1c15754cb8c8`；被验证的 PR 提交为 `f4f937838356591cad0a13b3566381ca7450191f`。

修改范围：`selfupdate/selfupdate.go`、`selfupdate/bootstrap.go`、新增 `selfupdate/marker_durability_test.go`，以及本组维护文档。

- 拒绝 `{}`、`null`、无效/倒退版本、负计数和越界次数等伪有效记录；兼容旧记录的零时间戳。
- 标记在同目录临时文件完整写入、Sync、Close 后替换，避免直接截断旧记录；新文件权限 0600。
- 最后一次标记写入在二进制换装之前完成；失败时取消换装并清除暂存件。
- 新版启动计数无法持久化时立即尝试恢复备份；双重失败保留诊断信息，需要人工修复。
- 无数据库结构、JWT、环境变量或证据数据迁移；回退使用 PR 父提交的源码重新构建。

## 已完成并验证

环境：macOS arm64，Go 1.27.0，更新模块仅依赖 Go 标准库；所有动态文件操作位于测试临时目录。

| 命令 / 方法 | 实际结果 |
| --- | --- |
| 原始基线 `go test ./selfupdate -count=1` | 通过 |
| 在原始源码运行新增关键回归测试 | 失败，成功复现空标记换装、写标记失败仍换装、计数失败不回滚 |
| 修改后 `go test ./selfupdate -count=1` | 通过 |
| 修改后 `go vet ./selfupdate` | 通过 |
| 修改后 `go test -race ./selfupdate -count=1` | 通过 |
| `git diff --check` | 通过 |
| [Linux 隔离验证](https://github.com/ForceMind/Kexra/actions/runs/39208895700) | 全仓库 build/vet/非 DB 单测、selfupdate race、go mod verify、文档与迁移检查通过 |
| 同次 Windows 原生测试 | 标记覆盖替换及不依赖 sh 的测试通过；shell 冒烟场景明确跳过 |

## 已实现并通过隔离验证：P0 Go / Web 依赖安全

分支 `codex/dependency-security-20261019`；跟踪 [Issue #10](https://github.com/ForceMind/Kexra/issues/10)。上述隔离运行的审计步骤仅作 inventory，并使用 continue-on-error：**整个运行成功不代表安全审计通过**。

- npm 实际失败：24 项，严重 2、高危 17、中危 5；前端 CycloneDX SBOM 已生成，但不是 Docker 镜像 SBOM。
- govulncheck v1.1.4 实际失败：报告 Go 1.26.3 标准库及 x/net v0.58.0 的可达漏洞；还需复核未调用的导入依赖。
- 修复提交 `5831ba106ef3403aa292c7a7bff2f839a3a46e58`，PR [#12](https://github.com/ForceMind/Kexra/pull/12)；[最终隔离验证](https://github.com/ForceMind/Kexra/actions/runs/39210043032) 的完整 Go/Web/中文/文档门禁通过。npm 24 项降至 0；Go 可达与导入包漏洞为 0，保留一项未导入 OpenPGP 的模块告警，详见 [依赖安全记录](DEPENDENCY_SECURITY.md)。
- 固定 Go 1.26.9、x/net v0.60.0、compress v1.18.7、Next.js 16.3.8 / PostCSS 8.5.23，原样 CSS 和 MIT 许可保留；新增第三方样式完整性门禁。没有数据或配置迁移。
- 首次修复复扫已归零，但前端 vendor CSS 格式检查失败，未运行构建；修复来源哈希检查与 vendor 格式排除后最终回归通过，失败证据保留。
- 下载限额、模拟更新与真实 Go 子进程回滚回归保存于 [PR #11](https://github.com/ForceMind/Kexra/pull/11)，精确代码提交 `6c7d48de899c5adfd1f0d31f3a6567bb02f4efc9` 已通过 [Linux/Windows 验证与五平台源码编译](https://github.com/ForceMind/Kexra/actions/runs/39210813606)。实际范围、失败恢复与边界见 [更新与回滚](UPDATING.md)。

## 已实现并验证：下载边界及原生更新回滚

PR #11 为压缩包、元数据及校验和设置有界读取，拒绝歧义条目；自动回滚失败时恢复当前程序，双重失败保留诊断。新增内存模拟发布、真实 Go 子进程、故障注入及业务手动回滚测试。无数据库/密钥/配置迁移。

本轮通过 macOS、Linux、Windows 的对应测试；完整 Go 非 DB 门禁及五平台 CLI 交叉编译通过。未完成真实 Release 下载、产品 HTTP 重启、数据库升级兼容、Docker 镜像升级、断电恢复或生产验收。

## 已实现并通过隔离验证：Docker 工具锁定

[PR #13](https://github.com/ForceMind/Kexra/pull/13)，精确代码提交 `4acfd073f6b8da1871a08889fffaf1fe47766c44`，[最终两架构验证](https://github.com/ForceMind/Kexra/actions/runs/39212481678)。直接与可见传递工具依赖使用独立锁文件，npm 审计为 0；保留默认 browser MCP 的 npx 调用，断网时直接进入锁定工具，其余 npx 参数交给原始 CLI。

Linux amd64/arm64 原生镜像真实构建、Chromium 无网络页面、命令入口、镜像锁文件一致性、新旧登录/Cookie、容器重建后 JWT、PostgreSQL 备份恢复均通过。工具 SBOM、系统包清单和镜像 ID 已保存到运行 artifact。未发布镜像，无生产资源/密钥/数据变更；完整镜像 CVE、digest 固定与 MCP 业务协议仍未验收。

## 已合并并验证：通知数据库集成中文断言

对当前基线执行了六个 PostgreSQL 16 包的独立数据库测试：[首次运行](https://github.com/ForceMind/Kexra/actions/runs/39212058774)。`agent/config/db/evidence/llmrec` 合计 353 个测试/子用例通过、零跳过；server 失败。[诊断](https://github.com/ForceMind/Kexra/actions/runs/39212572592) 确认六组通知集成测试仍要求历史韩语，而产品输出已经是中文。

修复通过 [PR #14](https://github.com/ForceMind/Kexra/pull/14) 合并，合并提交 `8e2577f09d9d1f5b39b2cb0ed0e560ea8bf39e92`。只更正通知展示断言和反向链接检查，不修改投递逻辑、状态码、原始证据或数据库。

精确测试提交 `1b94e759a078a405d4cce1dea7ea04bb816f6bda` 的 [server 复测](https://github.com/ForceMind/Kexra/actions/runs/39212902356) 为 393 pass / 0 fail / 1 skip；跳过的是 `TestLiveContextReview`，需要 `ARTEX_REVIEW_LIVE_CONFIG` 指定外部私有模型配置，本次未调用外部模型或消耗模型额度。其余五包 353 pass / 0 skip，共 746 个测试/子用例通过。失败与修复前诊断运行保留。

## 本次工程交接

- 当前主线从 GitHub 重新读取；不要把本文件所在提交的父提交当作永远不变的最新版本。
- 本会话 PR #9（标记可靠性）、#12（依赖修复）、#11（资源边界与原生回滚）、#14（通知 DB 中文回归）已合并；PR #13 已完成两架构实际镜像验证及代码审查，携带本次最终状态文档提交。
- 持久入口：[路线](ROADMAP.md)、[发布清单](RELEASE_CHECKLIST.md)、[依赖审计](DEPENDENCY_SECURITY.md)、[更新恢复](UPDATING.md)、[部署](DEPLOYMENT.md)。原六套工作流仍手动触发；没有 ChatGPT 定时任务、正式版本或镜像推送。
- 首要后续：完整镜像/浏览器/内嵌组件 SBOM 与 CVE、基础镜像 digest 和 apt 固定；随后验证真实 MCP 浏览器调用、真实 Release/HTTP 重启与数据升级兼容，继续 Issue #3 的后端中文化。
- 当前仍不能正式发布。对真实旧实例和生产环境的密钥/数据迁移，不以本轮新建隔离环境验收替代。

## 未验证与阻塞

- 本机终端 GitHub DNS 不可用，已通过 GitHub 连接读取固定提交并缓存必要源码；本地 Git 是源码子集的验证快照，不能把它的提交 SHA 当成远程完整仓库 SHA。
- 本机没有 Docker 或 PostgreSQL，缺少完整 Web/Go 依赖；全仓库、Web 与真实容器验证通过上述隔离运行执行，不冒充本机完成。
- Linux/Windows 标记替换已有上述专项验证；Windows shell 流程、断电/文件系统故障演练尚未完成。
- 镜像 arm64 原生隔离验证已完成；完整供应链、真实 Release 更新、数据库/配置升级兼容及生产环境仍未通过验收。

## 恢复步骤

1. 回读 GitHub `main`、开放 PR、Issue #2/#3 和最新验证运行；先核对是否已有后续工作。
2. 阅读 `AGENTS.md`、本文件和路线，避免把历史验收当成当前提交的测试结果。
3. 核对 PR #13/#14 的最终合并与验证状态，随后按路线优先完成完整镜像 SBOM/CVE 和基础层/digest 复核；保留未导入 OpenPGP 模块告警与 CLI/MCP 子依赖兼容记录。
4. 有客观阻塞时保留分支和真实结果，继续处理独立任务。六套正式 Actions 不恢复自动触发。

## 2026-10-20 增量：默认 MCP 实际运行

[PR #16](https://github.com/ForceMind/Kexra/pull/16) 修复默认 browser MCP 查找不存在的系统 Chrome，以及 root Chromium 需要显式 sandbox 参数的问题。代码提交 `8f2f796282deb8eecbde81944ed00cac96b1d4b1` 的 [两架构原生验证](https://github.com/ForceMind/Kexra/actions/runs/39359004374) 全部通过：完整镜像构建、真实无网络 MCP 初始化/工具/页面交互/中文快照、9 种自定义配置保留、常规 npx、新旧登录/Cookie、容器重建后 JWT 与数据库备份恢复。只改限定入口及验证脚本，无数据迁移、镜像推送或正式发布；root 浏览器关闭 Chromium 沙箱的边界见部署说明。

P0 完整镜像审计仍阻塞发布：[初审](https://github.com/ForceMind/Kexra/actions/runs/39268345885) 严重 8 / 高危 229；[候选复扫](https://github.com/ForceMind/Kexra/actions/runs/39358072868) 严重 4 / 高危 150。升级/固定方案和完整审计门禁保存在 `codex/image-supply-chain-20261020` / [草稿 PR #15](https://github.com/ForceMind/Kexra/pull/15)，没有合并失败的候选，也没有忽略漏洞。

工作空间符号链接越界已用临时文件复现，正在独立修复；本机 exec-server 连接中断后，本地编译/新文件写入不可用，后续通过 GitHub 保存代码并执行隔离验证。恢复时必须先回读远程 SHA，不依赖本地部分源码快照。

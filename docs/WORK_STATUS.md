# Kexra 当前状态与交接

更新时间：2026-10-19，北京时间。此文件记录事实与恢复入口；长期任务见 [路线](ROADMAP.md)，发布判断见 [发布清单](RELEASE_CHECKLIST.md)。

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
- 下载限额、模拟更新与真实 Go 子进程回滚回归保存于 [Draft PR #11](https://github.com/ForceMind/Kexra/pull/11)，macOS 单测/vet/race 通过；Linux/Windows 新增原生进程演练仍待执行。

## 未验证与阻塞

- 本机终端 GitHub DNS 不可用，已通过 GitHub 连接读取固定提交并缓存必要源码；本地 Git 是源码子集的验证快照，不能把它的提交 SHA 当成远程完整仓库 SHA。
- 本机没有 Docker 或 PostgreSQL，缺少完整 Web/Go 依赖，未在本机运行全仓库、Web、数据库或容器验收。
- Linux/Windows 标记替换已有上述专项验证；Windows shell 流程、断电/文件系统故障演练尚未完成。
- 发布供应链、镜像 arm64、真实 Release 更新、数据库/配置兼容与生产环境均未通过本次验收。

## 恢复步骤

1. 回读 GitHub `main`、开放 PR、Issue #2/#3 和最新验证运行；先核对是否已有后续工作。
2. 阅读 `AGENTS.md`、本文件和路线，避免把历史验收当成当前提交的测试结果。
3. 优先完成 Issue #10 的依赖修复、锁文件审计和完整回归，再恢复下载边界及模拟升级端到端测试。
4. 有客观阻塞时保留分支和真实结果，继续处理独立任务。六套正式 Actions 不恢复自动触发。

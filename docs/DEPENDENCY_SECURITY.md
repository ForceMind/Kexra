# 依赖安全审计与复现

## 2026-10-19 结果

验收代码提交：`5831ba106ef3403aa292c7a7bff2f839a3a46e58`，PR [#12](https://github.com/ForceMind/Kexra/pull/12)，跟踪 [Issue #10](https://github.com/ForceMind/Kexra/issues/10)。

| 范围 | 初次审计 | 修复后 | 证据 |
| --- | --- | --- | --- |
| Web npm 锁文件 | 24 项：严重 2 / 高危 17 / 中危 5 | **0 项**，所有严重级别均为 0 | [首次](https://github.com/ForceMind/Kexra/actions/runs/39208895700) / [复扫](https://github.com/ForceMind/Kexra/actions/runs/39210043032) |
| Go 可达调用 | 19 项报告，包含旧工具链和 x/net | **0 项**，已导入包告警也为 0 | 同上，govulncheck v1.1.4 |
| Go 未导入包所在模块 | 含旧压缩库及 OpenPGP 告警 | 1 项：GO-2026-5932，未导入的 `golang.org/x/crypto/openpgp` | 见下方保留事项 |
| Web CycloneDX SBOM | 已生成 | 基于修复后锁文件重新生成 | 复扫运行的 `dependency-security-repair-evidence` artifact |
| Docker / apt / Playwright 工具链 | 未审计 | **未审计** | 不在本次结论范围内 |

复扫使用 Go 1.26.9、Node 22.23.2。完整 Go build/vet/非数据库单测、selfupdate race、模块校验，以及 npm ci、Biome、Next 静态导出（30 页）、中韩键集、中文默认页面、第三方样式哈希、迁移和文档门禁均成功。此记录不包含数据库集成、Docker 镜像、生产或外部目标测试。

## 实际修复

- Go 最低版本与 CI/Release 构建工具链固定为 **1.26.9**，仍保留 Go 1.26 系列。使用 Go 1.27 系列时至少为 **1.27.2**；不能仅比较主/次版本，认为本机 1.27.0 一定比修复后的 1.26.9 安全。
- `golang.org/x/net` 升至 v0.60.0，关联 x/crypto、x/sys、x/text 由 Go 模块解析联动升级；`github.com/klauspost/compress` 升至 v1.18.7。
- Next.js 固定 16.3.8，PostCSS 固定 8.5.23；修复其他受影响传递依赖并更新锁文件。未使用 `npm audit fix --force` 或大版本回退。
- `shadcn` CLI 原先仅用于一份 CSS 导入；将该版本原样样式和 MIT 许可保存在仓库，删除 CLI 依赖树，保留实际使用的 `@shadcn/react` 组件。锁文件依赖总数由 824 降至 551（包括可选/开发依赖）。
- 原样第三方 CSS 单独使用 SHA256 和许可检查，不参与项目格式化；项目自有代码的 Biome 门禁保持不变。来源见 [第三方声明](../THIRD_PARTY_NOTICES.md)。

## 保留告警与边界

[GO-2026-5932](https://pkg.go.dev/vuln/GO-2026-5932) 涉及停止维护的 x/crypto/openpgp，扫描器标注没有修复版本。项目仍依赖 x/crypto 的其他功能；本次 `govulncheck -show verbose` 确认没有导入该包，也没有可达调用。**这不是“所有模块零告警”或忽略告警**。若今后引入该包，必须阻止合并并改用受维护实现。

首次运行的审计步骤是允许失败的 inventory，整个运行成功不代表审计通过；真实 npm / govulncheck outcome 均为 failure。第二次修复运行因 vendor CSS 格式检查失败，**未通过前端构建**。最终复扫已修复该问题，且增加了核对真实 outcome 的最终失败门禁。失败记录不删除、不改写。

上述 Web SBOM 不包含操作系统、Go 二进制、Playwright/浏览器或 apt，也不是完整镜像 SBOM。Actions artifact 有保留期，最终发布时须重新生成并与候选提交、锁文件摘要和包校验和一起归档。

## 复现命令

在完整仓库、已审查网络和相应工具链环境执行：

~~~bash
go version
go mod verify
go run golang.org/x/vuln/cmd/govulncheck@v1.1.4 -show verbose ./...
cd web
npm ci
npm audit --package-lock-only --ignore-scripts --json
npm sbom --package-lock-only --sbom-format cyclonedx
cd ..
python3 -I scripts/check-web-vendor.py
~~~

审计会查询公开漏洞数据库，结果随数据库更新而变化；每次保留工具版本、候选 SHA 和完整错误。不得把扫描器失败、网络错误或过滤严重级别后的结果记成零漏洞。Go 官方工具链版本可在 [下载清单](https://go.dev/dl/?mode=json) 核对。

# 依赖安全审计与复现

## 2026-10-19 结果

验收代码提交：`5831ba106ef3403aa292c7a7bff2f839a3a46e58`，PR [#12](https://github.com/ForceMind/Kexra/pull/12)，跟踪 [Issue #10](https://github.com/ForceMind/Kexra/issues/10)。

| 范围 | 初次审计 | 修复后 | 证据 |
| --- | --- | --- | --- |
| Web npm 锁文件 | 24 项：严重 2 / 高危 17 / 中危 5 | **0 项**，所有严重级别均为 0 | [首次](https://github.com/ForceMind/Kexra/actions/runs/39208895700) / [复扫](https://github.com/ForceMind/Kexra/actions/runs/39210043032) |
| Go 可达调用 | 19 项报告，包含旧工具链和 x/net | **0 项**，已导入包告警也为 0 | 同上，govulncheck v1.1.4 |
| Go 未导入包所在模块 | 含旧压缩库及 OpenPGP 告警 | 1 项：GO-2026-5932，未导入的 `golang.org/x/crypto/openpgp` | 见下方保留事项 |
| Web CycloneDX SBOM | 已生成 | 基于修复后锁文件重新生成 | 复扫运行的 `dependency-security-repair-evidence` artifact |
| Docker 浏览器工具 npm 锁文件 | 原先未形成锁文件 | **0 项**，已在两架构镜像验证 | [PR #13 验证](https://github.com/ForceMind/Kexra/actions/runs/39212481678) |
| 完整镜像已识别组件 | 初次全镜像扫描：严重 8 / 高危 229 | 修复复扫开发中，**未通过发布门槛** | [基线报告](https://github.com/ForceMind/Kexra/actions/runs/39268345885)，见下方覆盖边界 |

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

## 完整镜像审计（2026-10-20 开发中）

源码基线 `998d5082778b8a469fc48e3b6ec72fbb28eeeefb` 的 [隔离运行](https://github.com/ForceMind/Kexra/actions/runs/39268345885) 完成 Web/Go 及真实 amd64 镜像构建，再用 SHA256 核验的 Trivy **0.76.0** 扫描本地镜像，无 registry 推送、远程资产扫描或生产变更。

| 识别范围 | 包数 | 发现次数 |
| --- | --- | --- |
| Debian 12.15 系统包 | 261 | 946 |
| Node 包（包括 npm 自带依赖） | 205 | 27 |
| Python 包 | 1 | 6 |
| 实际 `/app/kexra` Go 二进制 | 51 | 1 |

总发现次数：严重 **8**、高危 **229**、中危 **341**、低危 **389**、未知 **13**。这是包与漏洞组合的出现次数，不能当作唯一 CVE 数量，也不能据此认定每项均可利用。报告保留已修复、未修复和发行版延后修复项，最后的高危/严重门禁真实失败。

`kexra-full-image-audit-amd64` artifact（14 天）保存：完整 Trivy JSON、521 组件的 SPDX、浏览器工具 CycloneDX、基础镜像 index/架构信息、镜像 ID、dpkg/运行时信息、Chromium 可执行文件版本与 SHA256、安装许可文件归档及校验清单。归档许可文件不等同于逐项许可兼容性审查。

**覆盖边界**：Trivy 的 Node 包扫描不是 Node 可执行文件内 V8/OpenSSL 等组件的完整审计；Playwright 下载的 Chromium 也不是 Debian 注册包。已记录浏览器版本 `156.0.8078.4` 及二进制哈希，但不能把它记为“无 CVE”。后续必须结合上游安全公告、组件版本、来源和独立浏览器验证复核。

[PR #15](https://github.com/ForceMind/Kexra/pull/15) 正在评估 Python 3.12.15 / Debian 13、固定多架构 digest、Debian 软件源快照和 npm 12.2.1 官方包。Node 22.23.3 来源为 [官方发布记录](https://github.com/nodejs/node/blob/v22.23.3/doc/changelogs/CHANGELOG_V22.md)，npm 包要求 Node `^22.22.2 || ^24.15.0 || >=26.0.0`；没有通过 npm 大版本强制改写浏览器工具锁文件。

初次候选 `1484a50c91d386a06f0a34399c7ea58c2ac1e385`（Node 22.23.2）[复扫](https://github.com/ForceMind/Kexra/actions/runs/39358072868) 为严重 **4** / 高危 **150** / 中危 239 / 低危 273 / 未知 13，SPDX 463 组件；Debian 251、Node 包 157、Python 包 1、Go 包 51。剩余严重/高危全部属于 Debian 系统层，报告均未给出修复版本；没有删除诊断工具来压低计数。该运行还复现默认 MCP 查找未安装系统 Chrome 的功能失败，实际 audit 与 MCP outcome 均为 failure（中间步骤使用 continue-on-error 收集证据，最后严格检查 outcome）。Node 22.23.3 及后续 MCP 修复不属于此次验证范围。

四项严重发现已回查 Debian 官方 tracker（2026-10-20）：

| CVE | 候选包 | 官方状态与下一步 |
| --- | --- | --- |
| [CVE-2019-16224](https://security-tracker.debian.org/tracker/CVE-2019-16224)、[16225](https://security-tracker.debian.org/tracker/CVE-2019-16225)、[16227](https://security-tracker.debian.org/tracker/CVE-2019-16227) | liblmdb0 0.9.31-1+b2 | trixie/unstable 仍受影响；上游未修复，触发涉及攻击者提供/损坏的 LMDB 文件。尚未完成产品可达性证明，继续阻止发布 |
| [CVE-2026-6653](https://security-tracker.debian.org/tracker/CVE-2026-6653) | libxml2 2.12.7+dfsg+really2.9.14-2.1+deb13u3 | trixie 仍受影响；不被版本字符串的 2.12.7 误导，实际回退到 2.9.14 系列。修复在更新的 upstream/unstable，需评估兼容补丁或替代基础层 |

Debian 将这些问题列为 no-dsa/minor 不等于软件已修复。本次不加入忽略项、不更改严重级别或排除未修复项；PR #15 继续草稿。替换系统共享库、从 unstable 混装或自行回移补丁可能破坏 ABI，需要单独可审查方案和回归，不能为扫描归零直接实施。

软件源快照只关闭归档元数据的有效期检查（`Check-Valid-Until`），继续由 Debian archive keyring 验证签名；不得使用 `trusted=yes` 或绕过签名。固定输入不会自动获得未来安全补丁，每次更新必须重新核对官方 index digest、npm 完整性值、快照日期及两架构真实构建/功能/漏洞报告。此方案固定依赖输入，不承诺镜像字节完全一致。

复现门禁（先在可信隔离环境安装已核验 Trivy，并实际构建 `kexra:local`）：

~~~bash
trivy image --scanners vuln,license --list-all-pkgs --format json --output trivy-image.json kexra:local
trivy image --format spdx-json --output image.spdx.json kexra:local
python3 -I scripts/check-image-audit.py trivy-image.json image.spdx.json --summary image-summary.json
python3 -I scripts/test-image-audit.py
python3 -I scripts/smoke-docker-mcp.py --image kexra:local
~~~

扫描器或网络失败不能生成安全通过结论；缺少系统、Node、Python、实际 Go 二进制覆盖或空 SBOM 时门禁直接报错。浏览器协议测试使用唯一命名的 `--network none` 临时容器，只操作 `about:blank` 和内存页面，不依赖外部模型或外部网站。

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

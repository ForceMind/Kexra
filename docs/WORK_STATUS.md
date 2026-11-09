# Kexra 当前状态与交接

更新时间为 2026-11-09（America/Los_Angeles）。长期任务见 [路线](ROADMAP.md)，发布判断见 [发布清单](RELEASE_CHECKLIST.md)。

## 当前恢复入口

聊天附件修复分支为 `codex/chat-upload-integrity-20261109`，基于联合验收分支 `codex/security-integration-20261108` 的 `9b64cec1affbab109b0def3e347d845bd3896ac8`。认证、出站、拦截策略和工具装配四项进展沿用 PR #23 的源提交与验收范围。

聊天附件原先在 `Stat` 检查文件名之后调用共享 `saveUpload` 创建文件，两个请求可取得同一路径；悬空符号链接也会被当成未占用名称。新上传专用函数通过 `os.Root.OpenFile` 的 `O_CREATE|O_EXCL` 排他分配名称，冲突时延续 `-1`、`-2` 命名，保留原文件和链接。复制字节数必须与 multipart 文件大小一致，复制或关闭失败后删除不完整文件。批量上传沿用逐文件提交，已经成功保存的文件在后续文件失败时仍存在。

附件的八项默认错误改为简体中文，包括任务不存在、无效 scope/id、任务删除中、创建目录、解析、缺少文件及保存失败。scope、id、file、JSON 字段、状态码和原始底层错误保留原协议；用户文件名和附件字节通过回归确认原样保存。工作空间直接上传仍调用原有共享保存函数，单独维护其文件管理语义。无数据库、环境变量或配置迁移。

| 本轮实际检查 | 结果 |
| --- | --- |
| 原始 chatUpload 的悬空链接 HTTP 回归 | 先复现 1 项失败，原处理函数返回了链接占用的名称 |
| 生产上传存储文件的独立标准库单测 | 7 条通过记录，含子例，零失败或跳过 |
| 实际附件与工作空间 handler 的裁剪联合回归 | 最终 race 运行 46 条通过记录，含父测试和子例，零失败或跳过 |
| 存储文件集与裁剪 handler 工程 vet | 通过 |
| Windows amd64 存储测试二进制 | 交叉编译通过，原生运行和权限尚待验收 |

裁剪工程为 `/tmp/kexra-chat-upload-check`。生产 chatupload、上传存储、workspace 与测试文件逐字节一致，Manager 的查询方法、Engine 的删除屏障/在跑计数以及 JSON helpers 从实际源码提取；支持声明只保留所需字段和 Activity 类型，数据库、SDK 与完整服务启动未进入本轮运行。HTTP 使用 httptest.NewRecorder，不绑定端口。回归包含 24 个同时开始的同名上传、已有文件/目录/悬空链接、截短的 multipart 临时文件、失败后的重试、零字节附件和中文/韩文文件名及二进制证据。

存储模块可在当前受限环境单独复现。

~~~bash
go test server/chat_upload_storage.go server/chat_upload_storage_test.go -count=1
go test -race server/chat_upload_storage.go server/chat_upload_storage_test.go -count=1
go vet server/chat_upload_storage.go server/chat_upload_storage_test.go
~~~

完整依赖环境继续运行。

~~~bash
go test -race ./server -run '^(TestChatUpload|TestWorkspace)' -count=1
go build ./...
go vet ./...
~~~

本轮再次确认 `github.com` 与 `proxy.golang.org` 的 DNS 不可用，当前无 PostgreSQL/Docker，GitHub 连接无手动 dispatch 能力。完整 Go/Web、SQL、真实网络与镜像复扫继续按下列联合验收入口恢复。最新源码仍需在固定 Go 1.26.9 或开发指南允许的补丁版本执行全部发布门禁。

## 联合验收入口（2026-11-08）

联合验证分支为 `codex/security-integration-20261108`，基于 main `70a468f87f9531f32ebdc63f0fdeee258d0f8e00`，整合以下固定提交。创建提交时以 main 为首父提交，四项源提交为附加父提交，保留研发历史。

| 来源 | 固定提交 | 验收余项 |
| --- | --- | --- |
| PR #19 | `33bb3e65624180ff8881503ebecbe100ceb2d01d` | PostgreSQL 初始化并发、完整服务、Windows 原生 |
| PR #20 | `fe1981dcce7a760350fc3b734c53724675a65a0d` | 完整 SDK、真实 socket/TLS/DNS、通知投递 |
| PR #21 | `c8820eaf5b84a23f74fb8e5e2f959e47100f49a2` | 完整工程、PostgreSQL 策略加载 |
| PR #22 | `18ab61aa3ccb6ee2227ed87e011c412f3679e84e` | SDK quota/session 包装、server 工具表故障回归 |

所有生产源码和 Go 测试按源 PR 的 blob SHA 原样纳入。共同文档的独立增量合并，路线与当前状态经人工协调。无新依赖、数据库结构或配置迁移。原 PR 各自保留审阅入口。供应链 PR #15 的最新镜像复扫继续阻塞发布。

## 本轮实际验证

环境为 macOS arm64、Go 1.27.0。临时裁剪工程 `/tmp/kexra-joint-check` 同时装入 guard/intercept、真实 assembly/finding 函数、JWT 文件模块、MCP 传输与通知源码。SDK 核心类型来自既有官方固定版本片段，数据库初始化和未用到的会话持久化被裁剪，装饰函数采用会立即 panic 的支持声明。支持文件只用于验证。

| 验证范围 | 实际结果 |
| --- | --- |
| 裁剪 guard/intercept、agent、JWT 文件模块 | 101 条测试通过记录，含父测试和子例，零失败/跳过 |
| 实际 MCP 传输文件集 | 99 条通过记录，含父测试和子例，零失败/跳过 |
| 通知精选无网络用例 | 151 条通过记录，含父测试和子例，零失败/跳过 |
| 上述裁剪四包 race | 通过 |
| MCP 文件集 race | 通过 |
| 通知同一精选用例 race | 通过 |
| 裁剪四包和通知 vet | 通过 |
| 所有 27 份生产 Go 源码及测试 | blob SHA 与源 PR 一致，gofmt 通过 |
| 本次合并的 9 份 Markdown 文档 | 相对链接目标存在，合并差异无空白错误 |
| 完整源码迁移检查 | 通过，扫描全部 401 份 Go 文件；此前部分快照的 14 项缺失已补齐 |
| 原有文档链接门禁 | 通过，79 份 Markdown、665 个文件引用、27 个锚点，零失效引用 |
| 中韩翻译键门禁 | 通过，2952 个可翻译字段一致 |
| 第三方样式与许可门禁 | 通过，固定 SHA256 与 MIT 许可声明一致 |
| 全部 Go 源码语法解析 | 401 份文件通过解析；55 份未改动主线文件存在 gofmt 差异，本轮 27 份改动文件格式通过 |
| 完整构建与依赖安装 | 已尝试，Go 模块和 npm registry 的 DNS 解析失败，未到达完整编译或 Web 检查/构建阶段 |
| 完整 Go/Web 回归、PostgreSQL、SDK、真实 socket/TLS、Docker | 未完成，仍待依赖和完整运行环境验收 |

源码验收固定在 `91919a87b8922208ce3788937e6c0963eded49e3`，本地目录为 `work/kexra-full-integration`。从固定远程 tree 逐项恢复文件，共 812 个 blob 中的 811 个已核对内容 SHA 与文件模式，包含全部源码、测试、文档和工作流。唯一未取回的 `web/media/dashboard.png` 为 1,246,155 字节历史截图，Contents 接口返回空内容，blob 接口按 UTF-8 解码失败；该图片未被文档门禁引用。源代码完整性与该图片的缺失分别记录，未声称整个本地 Git tree 与远程完全相同。

生产 Go 文件与源 PR 逐字节核对，裁剪中的 `db/settings.go` 仅保留策略测试所需部分。当前工具链低于开发指南推荐的 Go 1.27.2，最终验收使用仓库固定的 Go 1.26.9 或指南允许的补丁版本。

## 本轮完整源码检查命令

以下四项使用原仓库脚本，均已实际通过。

~~~bash
python3 -I scripts/check-kexra-migration.py
python3 -I scripts/check-doc-links.py
python3 -I scripts/check-i18n-keys.py
python3 -I scripts/check-web-vendor.py
~~~

`go mod download github.com/Autumn-27/norma` 和 `go build ./...` 均在下载阶段遇到 `lookup proxy.golang.org: no such host`。前端尝试 `npm ci --ignore-scripts --no-audit --fetch-retries=0 --fetch-timeout=10000 --cache /tmp/kexra-full-npm-cache`，返回 `ENOTFOUND registry.npmjs.org`；安装未完成，尚未运行 Biome、Next.js 构建或依赖审计。`ignore-scripts` 仅用于定位安装阻塞，正式前端验收继续使用开发指南中的正常安装命令。PostgreSQL 和 Docker 仍未安装，GitHub 连接当前无手动 workflow dispatch 能力。

## 下一验收命令

在完整分支 checkout、完整模块依赖与已生成的 Web 资源环境运行。`ARTEX_PG_DSN` 必须指向专用 PostgreSQL 16 测试库，并具有创建测试 schema 的权限。

~~~bash
python3 -I scripts/check-kexra-migration.py
go test ./db -run '^TestSetSettingIfEmpty' -count=1
go test -race ./server -run '^TestAuthInit' -count=1
go test ./guard ./intercept ./mcphttp ./notify -count=1
go test ./agent -run 'TestProvider|TestQuotaClientKeepsRedirectPolicy|TestResolved' -count=1
go test ./server -run '^TestToolsTableReadFailure' -count=1
go build ./...
go vet ./...
go test ./... -count=1
~~~

确认 PostgreSQL 并发用例实际运行且未跳过，继续执行 [开发指南](DEVELOPMENT.md) 的 Web、中文资源和文档门禁，再完成真实网络与镜像复扫。六套 Actions 保持手动，发布仍受 [发布清单](RELEASE_CHECKLIST.md) 约束。源码回退会恢复旧安全边界，生产恢复必须保留密钥与数据备份。

以下保留各源 PR 当时的验证记录。分支名称、当时环境和未完成事项属于对应固定提交的历史范围。

## PR #19 原分支验收记录（2026-11-07，认证启动安全）

- 已回读远程主线 `70a468f87f9531f32ebdc63f0fdeee258d0f8e00`，本轮基于该完整远程 tree 创建 `codex/auth-startup-safety-20261107`。本地 `work/Kexra` 是过时的源码子集，不用于覆盖主线。独立供应链草稿 PR #15 当前为 `e9ac6c3c501c9783363b8ca4347c19f2be8a5e45`，其审计门禁 10 项定向回归已通过，最新组合仍未完成真实镜像复扫。
- **已实现，PostgreSQL 集成待验证**：管理员初始化由 `db.SetSettingIfEmpty` 单条条件写入确定唯一胜者，兼容缺失或空值，已有非空密码不能被初始化覆盖；读取失败返回 500，竞争失败返回原有中文 403，只有成功写入者获得 Token。新增多连接池数据库回归及 8 个 Server 的并发 HTTP 回归；表锁用于确保所有请求到达竞争写入点，不依靠随机延时碰撞。测试仅创建随机 schema，不操作原有密码或真实服务。
- **已实现并通过本地定向验证**：JWT 目录使用实际目录身份、逐段目录句柄和打开前后身份核对，拒绝工作目录别名、密钥文件链接和根目录别名；安全系统路径别名仍可使用。完整临时文件经同步后以无覆盖硬链接发布，并发首次启动使用唯一持久化密钥。每次启动都核对两个遗留位置，匹配才清理，冲突或清理失败拒绝启动；64 KiB 上限防止把大文件当作密钥读取。
- 修改文件：`db/settings.go`、`db/settings_test.go`、`server/auth.go`、`server/auth_keys.go`、`server/auth_keys_security_test.go`、`server/auth_key_persistence_test.go`、`server/auth_init_concurrency_test.go`、`server/auth_localized_test.go`，以及配置/安全/状态文档和更新记录。无新依赖或数据库结构迁移；不改变已有账号、密码哈希和新旧 Cookie/JWT 兼容路径。新增严格检查可能使不安全、损坏或冲突的旧配置拒绝启动，不会自动轮换密钥。

| 本轮检查 | 实际结果 |
| --- | --- |
| macOS arm64 / Go 1.27.0，实际 `auth_keys.go` 与原有/新增密钥测试 | 19 个主测试 + 5 个子例，**24 pass / 0 fail / 0 skip** |
| 同一真实文件集 `go test -race` | 通过 |
| 同一真实文件集 `go vet` | 通过 |
| Windows amd64 测试二进制交叉编译 | 通过；未在 Windows 原生执行，也未验证 ACL |
| 所有改动 Go 文件的 gofmt/语法解析 | 通过 |
| 真实 PostgreSQL 初始化/设置并发、完整 Go 构建/vet/单测、Web、Docker | **未执行**；本机无 PostgreSQL/Docker，缺完整模块依赖，终端 GitHub DNS 不可用；不能将只读审查或交叉编译当作集成通过 |

复现本地定向验证：

~~~bash
GOCACHE=/tmp/kexra-auth-key-cache go test server/auth_keys.go server/auth_key_persistence_test.go server/auth_keys_security_test.go -count=1
GOCACHE=/tmp/kexra-auth-key-cache go test -race server/auth_keys.go server/auth_key_persistence_test.go server/auth_keys_security_test.go -count=1
GOCACHE=/tmp/kexra-auth-key-cache go vet server/auth_keys.go server/auth_key_persistence_test.go server/auth_keys_security_test.go
~~~

下一验收必须在完整仓库、已修复的推荐 Go 1.26.9 工具链和专用 PostgreSQL 16 测试库执行：

~~~bash
# ARTEX_PG_DSN 指向有 CREATE SCHEMA 权限的专用测试库，不使用生产连接。
go test ./db -run '^TestSetSettingIfEmpty' -count=1
go test -race ./server -run '^TestAuthInit' -count=1
go build ./...
go vet ./db ./server
~~~

数据库未配置时集成测试会跳过；配置后连接、锁同步或 SQL 故障均会失败。验收时必须确认上述并发用例实际执行且没有跳过。未以临时 push 触发工作流绕过手动策略：六套原 Actions 保持原样，本轮未触发 Actions。当前连接未提供工作流手动 dispatch 能力，不能声称云端验证已运行。

父目录/挂载须可信，`os.Root` 不隔离既有硬链接、bind mount 或同权限进程。遗留副本删除后未同步源目录，尚不声明断电迁移恢复；Windows ACL、真实旧实例升级及生产恢复另行验证。回退代码会恢复旧认证边界，不作为安全生产回退；始终保留密钥和数据库备份。本轮只提交草稿供审查，不合并、发布镜像、创建正式标签或改变运行中的服务。

完成本轮 PostgreSQL/完整工程验收后，继续工具调用与出站权限边界，以及 PR #15 的最新镜像复扫、系统补丁兼容性和浏览器/Node 内嵌组件审计；这些未完成项继续阻塞正式发布。

## PR #20 原分支验收记录（2026-11-07，远程 MCP 与通知出站）

- main 为 `70a468f87f9531f32ebdc63f0fdeee258d0f8e00`。认证草稿 [PR #19](https://github.com/ForceMind/Kexra/pull/19) 仍为 `33bb3e65624180ff8881503ebecbe100ceb2d01d`，未合并、没有新增 review thread 或 Actions；其 PostgreSQL/完整工程门禁继续保留。供应链草稿 PR #15 为 `e9ac6c3c501c9783363b8ca4347c19f2be8a5e45`，仍阻塞正式发布。
- 本分支 `codex/remote-egress-safety-20261107` 独立基于 main，完成 MCP 同来源凭据边界、完整有界 SSE endpoint 帧，以及通知 HTTPS 降级阻断与非法 Location 错误脱敏。保留合法 HTTP/内网部署、现有通知拨号策略和 SDK 工具适配接口；无新依赖、配置字段或数据库迁移。不同来源重定向/endpoint、含 userinfo/fragment 的 MCP 地址会明确失败。
- MCP 实际传输文件的 15 个顶层/88 个叶用例通过，含父测试 99 个 pass，零失败/跳过；race、vet 与 Windows amd64 交叉编译通过。通知 98 个精选顶层/151 个测试事件通过，同一选择集 race 及整包 vet 通过。反例均先在旧代码复现；范围、兼容约束和命令见 [出站边界](OUTBOUND_SECURITY.md)。
- HTTP 回归运行真实 net/http.Client 和内存 RoundTripper，SSE 用 io.Pipe；当前沙箱 bind 拒绝，未执行真实 socket/TLS/DNS/代理集成，也未运行完整 SDK、Go/Web/PostgreSQL/Docker。交叉编译不等于 Windows 原生通过，本轮不恢复或触发 Actions，不合并或发布。
- 后续继续工具策略加载故障阻断的独立变更，模型 HTTP 客户端、其他工具权限和完整出站授权；与 PR #19 集成验收、PR #15 镜像复扫分别保留门禁。回退会恢复旧凭据边界，不视为安全生产回退。

## PR #21 原分支验收记录（2026-11-07，工具策略加载故障）

- 固定主线为 `70a468f87f9531f32ebdc63f0fdeee258d0f8e00`；本分支 `codex/tool-policy-fail-closed-20261107` 独立保存策略异常阻断，不覆盖认证草稿 [PR #19](https://github.com/ForceMind/Kexra/pull/19) 或供应链草稿 PR #15 的未合并工作。
- `guard/guard.go` 对启用检查/规则匹配的加载错误立即阻断；`intercept/intercept.go` 两个接口显式返回 error，规则与工具集合完整成功后才发布缓存，拒绝损坏配置/null 元素/启用的非法正则。显式 `[]`、缺省工具和配置 FailAction=allow 保留，无数据库迁移。
- 新增真实策略代码的 SQL 故障注入回归，旧源码先复现失败；修改后含既有用例共 49 条通过记录，race/vet/格式检查通过。生产调用点已扫描全部 385 个 Go 文件，只有 Guard 两处，已同步修改。
- 验证来自临时裁剪工程：保留真实策略/官方 SDK 类型，裁剪 DB 初始化与未用会话持久化；交付三文件与运行文件逐字节相同。未执行完整工程 build、真实 PostgreSQL、server/Web、网络或 CI，不将裁剪验证写成全工程通过。详情与复现入口见 [策略故障边界](TOOL_POLICY_SECURITY.md)。
- 此分支保持草稿待完整集成。并行的 MCP/通知出站变更、认证 PR #19 数据库门禁、PR #15 完整镜像复扫独立保留；下一步继续其他工具表/模型网络客户端和审批/出站授权。六套正式 Actions 原样且未触发，没有发布、生产服务或真实数据变更。

## PR #22 原分支验收记录（2026-11-08，模型 HTTP 与工具装配）

- main 仍为 `70a468f87f9531f32ebdc63f0fdeee258d0f8e00`；PR #19/#20/#21 均开放且保持草稿，最新 head 分别为 `33bb3e6` / `fe1981d` / `c8820ea`，没有新增 review thread。本分支 `codex/runtime-boundaries-20261108` 独立基于 main，保留这些未合并工作，不以本轮替代其验收。
- 模型 HTTP 工厂限定同源重定向并隐藏非法代理地址中的凭据，保留直连/合法代理、HTTP 内网、SDK quota/录制/会话包装。工厂标准库测试 22 条通过，race/vet 通过，Windows amd64 仅交叉编译通过。
- 工具表读取失败返回显式空列表；装配完成后防止空列表变为 nil 触发 SDK DefaultTools，并清除不再允许工具的延迟元数据。真实装配函数的裁剪回归 6 条通过，race/vet 通过；原行为已先复现失败。
- 新增完整 SDK 包装回归及 server 闭数据库句柄回归，尚未执行；完整 Go/Web/PG/网络/Docker 仍受当前环境限制。本轮没有运行外部模型、代理或扫描，也未触发 Actions、合并、发布或修改生产数据。
- 具体改动、命令、兼容性和裁剪范围见 [运行时边界](RUNTIME_BOUNDARIES.md)。下一阶段完成完整 SDK/server/数据库集成、现有草稿的联合验证，以及镜像/生产发布门禁；不能把单模块验证转移到完整工程。

## 历史恢复记录（2026-10-20）

- 已合并并验证：PR #16（默认 MCP 实际运行，合并 `6287962a736114ca270d8d074baafae2170ea1ac`）、PR #17（工作空间/附件文件边界，合并 `300f1958518cddd601e7438ca0bfeae0a41e78e4`）。恢复时必须重新回读 GitHub main，不能把这些历史 SHA 当成永远最新。
- 工作空间默认 10 条 API 错误的中文化保存在 [PR #18](https://github.com/ForceMind/Kexra/pull/18)。代码 `5fe6ad790c43ed98d90f62258bcfb53e56909a62` 的 [专项验证](https://github.com/ForceMind/Kexra/actions/runs/39360407124) 为 **22 pass / 0 fail / 0 skip**，格式与 server vet 通过。真实 HTTP 固定中文断言和韩语原始证据/文件名原样保留均验证；不代表其余后端已全部中文化。
- 未合并的供应链工作保存在 `codex/image-supply-chain-20261020` / [草稿 PR #15](https://github.com/ForceMind/Kexra/pull/15)。完整基线严重 8 / 高危 229，首个 Debian 13/npm 候选严重 4 / 高危 150，均未通过发布安全门槛；最新 Node 22.23.3 和后续合并主线没有重新执行镜像复扫，不能转移旧结果。
- 本机 exec-server 连接已中断，最终分支/文档/验证都保存在 GitHub；本地是未同步的部分源码快照，不是完整远程仓库。不要据此覆盖远程，也不要将本地 synthetic SHA 当作远程 SHA。
- 下一 P0：JWT 配置目录的真实路径与符号链接、管理员初始化并发、工具调用/出站权限；并继续 PR #15 的可修复系统补丁方案、浏览器/Node 内嵌组件及许可复核。P2 继续聊天附件及其他默认后端文案。
- 仍不可正式发布；六套原工作流仍手动，没有镜像推送、正式标签、生产迁移或 ChatGPT 定时任务。

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

## 已实现并通过隔离验证：工作空间与附件文件边界

[PR #17](https://github.com/ForceMind/Kexra/pull/17)，代码 `055a3fbeef1e8f6f19d7f09dbe92b6f5e1241519`，[最终验证](https://github.com/ForceMind/Kexra/actions/runs/39360052527)。所有工作空间文件 I/O 与聊天附件创建/上传都通过 `os.Root`；下载使用受限文件句柄，不再由 ServeFile 重新打开路径；相对内部符号链接、中文文件名、二进制证据、Range 下载与正常文件生命周期保留。

- 完整 server 上原始 handler 的 8 个越界回归全部失败，真实复现；修复后 Linux 全仓库 build/vet、server PostgreSQL 16 的 **407 pass / 0 fail / 1 skip**、工作空间/附件 race 回归通过。唯一跳过仍是需要私有外部模型配置的 `TestLiveContextReview`，没有调用外部模型。
- Windows 原生完整 server 包的工作空间/附件定向回归通过，包括符号链接越界、嵌套目录和文件生命周期。没有以交叉编译代替运行。
- [首次集成失败](https://github.com/ForceMind/Kexra/actions/runs/39359798489) 为共享 saveUpload 函数的聊天附件调用未同步；已修复并保留失败证据。回读 Go 1.26.9 标准库确认 Windows 的 FS 路径分隔约束后，目录列表改用受限目录句柄。
- 本机仅在 exec-server 断开前执行实际 workspace 源码的临时 HTTP harness：原始 8 项失败、初次受限实现 8 项通过；最终完整集成结果来自上述隔离运行，不能冒称最终源码已在本机全仓库测试。

无数据、证据、JWT 或配置迁移；绝对或越界符号链接会被拒绝，相对内部链接保留。`os.Root` 不代替 Agent/命令/MCP 的独立操作系统隔离，也不隔离 bind mount、设备文件或同一卷中的硬链接。回退源码会恢复原先文件边界风险，不能将其作为生产安全回退。

后续：PR #15 的未修复系统漏洞与浏览器/Node 内嵌组件审计继续阻塞发布；继续工作空间中文化，并验证 JWT 配置目录的真实路径/符号链接、初始化并发和工具权限等独立 P0 边界。

## 已实现并通过专项验证：工作空间默认中文

PR #18 修改 `server/workspace.go` 的 10 条默认错误及 `server/workspace_localized_test.go`。纠正强制韩语的旧测试；5 条实际 HTTP 错误直接与固定中文展示比较，另验证包含韩语的文件名与原始证据内容不被翻译。专项结果为 22 个测试/子用例通过、零失败/跳过；不改状态码、表单、路径、原始底层错误或存储内容。完整 server DB 的 407 pass / 1 skip 属于 PR #17 的代码范围，不能自动当作新增本地化测试已执行全仓库/DB。

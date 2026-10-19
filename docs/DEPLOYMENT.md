# Kexra 部署与发布约束

> **版本范围：2026-10-19 开发基线。PR #13 使用独立 npm 锁文件安装浏览器工具，并完成 [Linux amd64/arm64 原生镜像、断网命令/Chromium、登录、JWT 持久化与数据库恢复验证](https://github.com/ForceMind/Kexra/actions/runs/39212481678)。**
>
> **状态：尚未完成正式发布及生产部署验收。** 当前不存在官方可下载的 Kexra Docker 镜像或已验证可自动升级的正式 Release。仅用于授权的隔离实验室，不建议直接面向公网部署。

## 1. 首先区分三种运行方式

| 方式 | 当前用途 | 状态 |
| --- | --- | --- |
| 本地 Go + Next.js 开发 | 调试界面、配置、接口和构建 | 可在自行配置 PostgreSQL 后尝试 |
| 本地自构建 Docker 镜像 | 在受控实验室运行完整服务 | amd64/arm64 原生镜像和下方隔离流程已验证；生产部署未验收 |
| GitHub Release / 自动更新 | 多平台分发与版本更新 | **尚未验收，不得视为可用** |

## 2. 手动构建 Linux amd64/arm64 本地镜像（隔离测试）

Go 编译/单测与上述两种原生架构的隔离 Docker 流程已通过；**多架构镜像公开发布、真实版本升级/数据库兼容与生产环境仍未验收**。以下命令供 Linux amd64 受控实验室构建，Go 使用 1.26.9（1.27 系列至少 1.27.2）：

~~~bash
cd web
npm ci
npm run build:static
cd ..

mkdir -p server/webui/dist dist/amd64
cp -a web/out/. server/webui/dist/
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -tags embedui -o dist/amd64/kexra ./cmd/kexra

docker build --build-arg TARGETARCH=amd64 -t kexra:local .
~~~

在原生 arm64 主机上，将命令中的 `dist/amd64`、`GOARCH=amd64` 和 `TARGETARCH=amd64` 分别改为 `dist/arm64`、`GOARCH=arm64` 和 `TARGETARCH=arm64`。本次 arm64 使用真实 ARM 运行器，未以 QEMU 构建代替原生运行证据。

Dockerfile 采用运行时镜像，**不会自行编译应用二进制**，并预装额外浏览器/工具依赖。构建应在可信环境执行，核对依赖来源、架构、镜像体积及权限，不要把容器当成隔离安全保证。

## 3. Compose 配置

检查 `docker-compose.yml`：服务名为 `kexra`，默认镜像 `kexra:local`；数据库仍沿用兼容字段 `POSTGRES_USER` / `POSTGRES_DB` 和 `ARTEX_PG_DSN`。必须先审查 `.env`，设置真实本地密码。

~~~bash
cp .env.example .env
# 编辑 .env，使用随机数据库口令、审阅模型密钥及允许的网络范围
docker image inspect kexra:local
docker compose config
~~~

在镜像、端口、数据挂载和测试范围确认前，不应直接开启 Compose。Compose 已将宿主机 `8787` 管理界面和 `8788` 流量代理均绑定在 `127.0.0.1`。如需反向代理，请另行配置可信访问控制和 TLS；不要取消本机监听限制后直接公开端口。

## 4. 持久化与备份

- `pgdata`：PostgreSQL 持久卷；不要使用 `docker compose down -v` 清除它。
- `./data`：SQLite、工作数据与临时工件，可能通过文件管理器访问；**不允许存储 JWT 私钥**。
- `jwtkeys`：独立的 Compose 命名卷，挂载在 `/app/private`，持久化 `/app/private/jwt.key`；容器重建时保留，不能使用 `docker compose down -v` 删除。
- `./skills`：正在使用的技能目录；更新前备份自定义修改。
- 新容器通过 `KEXRA_JWT_KEY_DIR=/app/private` 将 JWT 密钥放到独立私有目录（权限 0700，密钥 0600）。裸机原有默认路径仍保持在可执行文件旁。
- **升级已有容器前先备份旧容器的 `/app/jwt.key`**。旧容器的可写层不会随镜像重建保留；若旧密钥只在容器层，必须先安全恢复到新命名卷 `/app/private/jwt.key`，否则历史 JWT 会话失效。不要将备份放到 `./data`、Git 或公开目录。
- 更换数据库名、用户名、端口或密钥之前，先建立可验证的恢复点并使用副本演练。

## 5. Docker 持久化与本地访问验收清单

1. 在确认已备份现有容器密钥及数据库后，构建固定版本本地镜像；先执行 `docker compose config` 审查配置，确认 8787、8788 均仅绑定宿主机回环地址。
2. 启动隔离 Compose 并观察 `/api/health`，验证管理员初始化、旧用户名兼容、正常登录、容器重启后会话有效。
3. 验证 `pgdata`、`./data`、`jwtkeys` 三类数据在容器删除重建后仍存在；用数据库及 JWT 密钥的备份副本演练恢复，不能在唯一真实数据上尝试。
4. 检查 `/app/private/jwt.key` 权限和文件管理器不可访问性；对故意损坏的密钥应拒绝启动，而不是静默重建。
5. Node 运行时来自 `node:22.23.2-bookworm-slim`；浏览器工具的直接与可见传递依赖由 `docker/browser-tools/package-lock.json` 固定，`npm ci --omit=dev --ignore-scripts` 安装，不执行 NodeSource 脚本或无锁的全局 npm 安装。**基础镜像 digest、apt 及完整浏览器供应链仍未完全固定或审计**。

## 6. Docker 工具链复核

工具链的初步静态门禁：

~~~bash
python3 -I scripts/check-docker-toolchain.py
~~~

2026-10-19 的 [两架构验证](https://github.com/ForceMind/Kexra/actions/runs/39212481678) 使用代码提交 `4acfd073f6b8da1871a08889fffaf1fe47766c44`，分别在 x86_64 / aarch64 原生主机完成：镜像构建、Node/Playwright 命令、断网 Chromium 页面、内置 `npx @playwright/mcp --headless --help`、常规 `npx --version`、镜像锁文件逐字节比对、登录/旧 Cookie、重建后的 JWT 与 PostgreSQL 恢复。

浏览器工具安装在 `/opt/kexra-browser-tools`，CLI 通过 PATH 使用。内置 browser MCP 的历史默认命令 `npx @playwright/mcp` 由一个限定入口直接交给已锁定的 `playwright-mcp`；其他包、显式版本和 npx 参数交给原始 CLI，不改数据库或用户配置。

这只验证 MCP/CLI 入口与独立浏览器启动，**未验证完整 MCP 浏览器协议/业务调用**。锁文件保留当前 CLI/MCP 自带的固定 alpha Playwright 依赖，不把顶层 `playwright@1.64.1` 的成功当作全部子组件兼容证明。

工具锁文件 npm 审计各严重级别均为 0。运行 artifact 保存工具 CycloneDX SBOM、dpkg 清单、Go 构建信息和本地镜像 ID；这些不是完整镜像 CVE 通过证明，也不覆盖所有打包在组件内部的代码。基础镜像 digest、apt、浏览器/嵌入组件及完整镜像 SBOM/CVE 仍为发布门槛。

## 7. 发布与升级

`.github/workflows/release.yml` 仅允许手动触发，且只在选择 `v*` 标签时执行发布任务；构建仍使用历史 `ARTEX_TARGET_*` 构建变量，Docker 推送需要明确的 `KEXRA_DOCKER_IMAGE` 和密钥配置。**不要为了尝试打包而随意推送正式标签**。

`selfupdate/github.go` 使用 `ForceMind/Kexra`；模拟 Release 下载与 Linux/Windows 原生回滚已有验证，见 [更新与回滚](UPDATING.md)。真实 Release、产品 HTTP 重启、数据库升级兼容和生产升级仍需逐项验收，不能直接在现有服务上演练。

正式发布的最低门槛：Go 构建/测试、数据库集成、Web 编译/静态导出、文档链接、检测规则、容器/回滚演练、密钥审计和网络隔离检查全部通过，并留存 CI 结果。

延伸：[安装](INSTALLATION.md) · [配置](CONFIGURATION.md) · [安全边界](SECURITY_MODEL.md) · [迁移问题](MIGRATION.md)。

# Kexra 部署与发布约束

> **版本范围：本指南适用于 Kexra 当前主线。PR #5 已完成隔离环境中的 Docker 构建、容器启动、登录/会话与 PostgreSQL 备份恢复验收；本轮工具链版本调整又通过了 [Linux amd64 实际镜像及无网络 Chromium 冒烟测试](https://github.com/ForceMind/Kexra/actions/runs/38737617057)。**
>
> **状态：尚未完成正式发布及生产部署验收。** 当前不存在官方可下载的 Kexra Docker 镜像或已验证可自动升级的正式 Release。仅用于授权的隔离实验室，不建议直接面向公网部署。

## 1. 首先区分三种运行方式

| 方式 | 当前用途 | 状态 |
| --- | --- | --- |
| 本地 Go + Next.js 开发 | 调试界面、配置、接口和构建 | 可在自行配置 PostgreSQL 后尝试 |
| 本地自构建 Docker 镜像 | 在受控实验室运行完整服务 | 隔离 Linux amd64 构建与浏览器启动已验证；arm64 和生产部署未验收 |
| GitHub Release / 自动更新 | 多平台分发与版本更新 | **尚未验收，不得视为可用** |

## 2. 手动构建 Linux amd64 本地镜像（隔离测试）

`main` 的 Go 编译和单元测试已经通过，PR #5 的隔离 Docker 测试与本轮固定工具链下的 Linux amd64 镜像/浏览器冒烟测试都已通过；**arm64、多架构发布、真实升级回滚与生产环境仍未验收**。以下命令供受控实验室手动构建：

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
5. Dockerfile 在本轮工具链调整中改用 `node:22.23.2-bookworm-slim` 运行时层，并锁定直接安装的 `@playwright/mcp@0.0.83`、`@playwright/cli@0.1.22`、`playwright@1.64.1`，不再运行 NodeSource 远程安装脚本；**该直接依赖组合已通过 Linux amd64 的实际镜像构建和离线 Chromium 启动；npm 传递依赖、基础镜像 digest 与 apt 包仍未完全固定**。

## 6. Docker 工具链复核

工具链的初步静态门禁：

~~~bash
python3 -I scripts/check-docker-toolchain.py
~~~

[2026-10-15 专项验证](https://github.com/ForceMind/Kexra/actions/runs/38737617057) 已在 Linux amd64 镜像上实际完成 `node --version`、直接工具包版本、Playwright CLI/MCP 帮助命令和 Chromium 无网络启动。**这只证明 Playwright 单独运行时可启动 Chromium，尚未证明 MCP/CLI 的全部浏览器功能或 arm64 镜像运行可靠**。

本轮直接固定的版本减少了依赖漂移，但不等于有完整的锁文件。后续发布必须审查 npm 传递依赖、浏览器及系统二进制、apt 索引、官方基础镜像 digest、SBOM 和 CVE。

## 7. 发布与升级

`.github/workflows/release.yml` 仅允许手动触发，且只在选择 `v*` 标签时执行发布任务；构建仍使用历史 `ARTEX_TARGET_*` 构建变量，Docker 推送需要明确的 `KEXRA_DOCKER_IMAGE` 和密钥配置。**不要为了尝试打包而随意推送正式标签**。

`selfupdate/github.go` 目前已转向 `ForceMind/Kexra`，但包名、下载验证、回滚、首次发布和私有仓库访问仍需逐项验证；在通过前应禁用/回避 UI 自动更新操作。

正式发布的最低门槛：Go 构建/测试、数据库集成、Web 编译/静态导出、文档链接、检测规则、容器/回滚演练、密钥审计和网络隔离检查全部通过，并留存 CI 结果。

延伸：[安装](INSTALLATION.md) · [配置](CONFIGURATION.md) · [安全边界](SECURITY_MODEL.md) · [迁移问题](MIGRATION.md)。

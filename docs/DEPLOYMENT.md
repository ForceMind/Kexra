# Kexra 部署与发布约束

> **版本范围：本指南对应已经合入 `main` 的 Kexra 代码。五平台打包、Compose 配置及隔离本地二进制运行已经验证；完整 Docker 镜像部署尚未验证。**
>
> **状态：未完成部署验收。** 当前仓库尚未确认存在可直接拉取的官方 Kexra Docker 镜像或可自动升级的正式 Release。仅用于授权的隔离实验室，不建议直接面向公网部署。

## 1. 首先区分三种运行方式

| 方式 | 当前用途 | 状态 |
| --- | --- | --- |
| 本地 Go + Next.js 开发 | 调试界面、配置、接口和构建 | 可在自行配置 PostgreSQL 后尝试 |
| 本地自构建 Docker 镜像 | 在受控实验室运行完整服务 | 需构建镜像并人工验证 |
| GitHub Release / 自动更新 | 多平台分发与版本更新 | **尚未验收，不得视为可用** |

## 2. 手动构建 Linux amd64 本地镜像（隔离测试）

`main` 的 Go 编译和单元测试已经通过，**但 Docker 镜像和升级回滚仍未完成验收**。以下命令仅为受控实验室的待验收构建步骤：

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
5. Dockerfile 仍依赖 NodeSource 和 npm `@latest` 浏览器工具：**供应链版本尚未固定，也未完成真实镜像构建及完整镜像安全审计**。正式发布前必须单独解决。

## 6. 发布与升级

`.github/workflows/release.yml` 仅允许手动触发，且只在选择 `v*` 标签时执行发布任务；构建仍使用历史 `ARTEX_TARGET_*` 构建变量，Docker 推送需要明确的 `KEXRA_DOCKER_IMAGE` 和密钥配置。**不要为了尝试打包而随意推送正式标签**。

`selfupdate/github.go` 目前已转向 `ForceMind/Kexra`，但包名、下载验证、回滚、首次发布和私有仓库访问仍需逐项验证；在通过前应禁用/回避 UI 自动更新操作。

正式发布的最低门槛：Go 构建/测试、数据库集成、Web 编译/静态导出、文档链接、检测规则、容器/回滚演练、密钥审计和网络隔离检查全部通过，并留存 CI 结果。

延伸：[安装](INSTALLATION.md) · [配置](CONFIGURATION.md) · [安全边界](SECURITY_MODEL.md) · [迁移问题](MIGRATION.md)。

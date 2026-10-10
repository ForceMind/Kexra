# Kexra 部署与发布约束

> **状态：未完成部署验收。** 当前仓库尚未确认存在可直接拉取的官方 Kexra Docker 镜像或可自动升级的正式 Release。仅用于授权的隔离实验室，不建议直接面向公网部署。

## 1. 首先区分三种运行方式

| 方式 | 当前用途 | 状态 |
| --- | --- | --- |
| 本地 Go + Next.js 开发 | 调试界面、配置、接口和构建 | 可在自行配置 PostgreSQL 后尝试 |
| 本地自构建 Docker 镜像 | 在受控实验室运行完整服务 | 需构建镜像并人工验证 |
| GitHub Release / 自动更新 | 多平台分发与版本更新 | **尚未验收，不得视为可用** |

## 2. 手动构建 Linux amd64 本地镜像（隔离测试）

本次品牌迁移的 `build.sh` 已切换产物为 `kexra`，但目前仍引用未创建的 `cmd/kexra` 路径；因此它**不能在修复前被认定为可执行的发布脚本**。以下示例直接使用仓库现存 Go 入口 `cmd/artex`，只作为人工构建检查路径：

~~~bash
cd web
npm ci
npm run build:static
cd ..

mkdir -p server/webui/dist dist/amd64
cp -a web/out/. server/webui/dist/
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -tags embedui -o dist/amd64/kexra ./cmd/artex

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

在镜像、端口、数据挂载和测试范围确认前，不应直接开启 Compose。`8787` 管理界面在默认 Compose 配置下可能绑定到宿主机所有地址，必须使用网络层限制或显式改为只监听回环/内网；`8788` 是流量代理端口，不能开放给未知来源。

## 4. 持久化与备份

- `pgdata`：PostgreSQL 持久卷；不要使用 `docker compose down -v` 清除它。
- `./data`：服务运行数据、密钥及临时工件；改名不等于可以删掉。
- `./skills`：正在使用的技能目录；更新前备份自定义修改。
- 部署前核对 JWT 密钥存储位置（部分版本迁移到工作区之外）；备份时加密并限制访问。
- 更换数据库名、用户名、端口或密钥之前，先建立可验证的恢复点并使用副本演练。

## 5. 发布与升级

`.github/workflows/release.yml` 以 `v*` 标签触发并使用历史 `ARTEX_TARGET_*` 构建变量，Docker 推送需要明确的 `KEXRA_DOCKER_IMAGE` 和密钥配置。**不要为了尝试打包而随意推送正式标签**。

`selfupdate/github.go` 目前已转向 `ForceMind/Kexra`，但包名、下载验证、回滚、首次发布和私有仓库访问仍需逐项验证；在通过前应禁用/回避 UI 自动更新操作。

正式发布的最低门槛：Go 构建/测试、数据库集成、Web 编译/静态导出、文档链接、检测规则、容器/回滚演练、密钥审计和网络隔离检查全部通过，并留存 CI 结果。

延伸：[安装](INSTALLATION.md) · [配置](CONFIGURATION.md) · [安全边界](SECURITY_MODEL.md) · [迁移问题](MIGRATION.md)。

# Kexra 安装与本地启动

本说明以已完成品牌迁移的当前 `main` 为依据，面向**自有、隔离的开发环境**。Go 二进制与中文前端已通过隔离运行测试；尚无经过完整 Docker 镜像验收的正式部署方案。

## 1. 环境要求

- Go：项目 `go.mod` 声明 1.26.3；运行 `go version` 核对实际工具链。
- Node.js：CI 使用 22；前端 Next.js 16 / React 19，推荐与 CI 一致。
- npm：项目根目录下的 `web/package-lock.json` 作为安装锁定来源。
- PostgreSQL：CI 使用 `postgres:16-alpine`。
- 本地开发的授权资产样例与凭据；不得使用真实客户数据作为演示数据。

## 2. 克隆与安全检查

~~~bash
git clone https://github.com/ForceMind/Kexra.git
cd Kexra
git status
go version
node --version
~~~

若你要测试尚未合入的独立品牌版本，明确切换到 `feat/kexra-independent-brand`；`main` 在迁移验收前仍可能使用历史产品标识。

## 3. 配置数据库

~~~bash
cp config.example.json config.json
~~~

打开 `config.json` 填写 PostgreSQL 连接地址、端口、用户名、密码及 `sslmode`。样例默认端口为 **5433**，并非所有环境都适用：常见本地 PostgreSQL 或 Compose 服务可能使用 5432。也可以通过 `ARTEX_PG_DSN` 覆盖连接参数，详细优先级见 [配置说明](CONFIGURATION.md)。

> `config.json` 内不要放生产密码并提交 Git。原来的 `ARTEX_*` 环境变量保留是兼容行为，不是改名失败。

如果仅需要独立启动本地 PostgreSQL，可在审查 `.env` 后使用自己的数据库服务；不要误以为 `docker compose up -d postgres` 会自动把服务端口映射给宿主机：当前 Compose **没有** PostgreSQL 的宿主机端口映射。

## 4. 安装与构建前端

~~~bash
cd web
npm ci
npm run check
npm run build:static
cd ..
~~~

静态导出位于 `web/out/`。更改语言须重建：默认简体中文 `zh`；兼容韩语可设置 `NEXT_PUBLIC_LOCALE=ko`。

## 5. 启动本地开发服务

品牌迁移分支现使用 `cmd/kexra` 作为 Go 启动入口。旧 `cmd/artex` 已从该分支移除；不要用旧路径构建。

~~~bash
# 先在独立终端启动 PostgreSQL，并确保连接参数已经配置。
go run ./cmd/kexra -addr 127.0.0.1:8787 -proxy ""
~~~

需要浏览器开发热更新时，在另一个终端执行：

~~~bash
cd web
npm run dev
~~~

Next.js 默认开发端口通常为 **3000**（实际以控制台输出为准）。`dev.sh` 仍有旧的固定端口文案，应在品牌迁移验收时同步修正。首次使用可能进入 `/setup` 初始化管理员口令；切勿把测试环境暴露到公网。

## 6. 检查与关闭

~~~bash
go build ./...
go vet ./...
go test ./... -count=1
python3 -I scripts/check-doc-links.py
~~~

关闭进程、按需备份本地数据目录。上述命令不能代替带 PostgreSQL 的集成测试、Web 静态验证或授权运行环境的验收。完整门禁见 [开发与测试](DEVELOPMENT.md)。

## Docker

当前 `docker-compose.yml` 的应用服务使用 `KEXRA_IMAGE`，默认 `kexra:local`，**不存在可默认拉取的官方镜像**。先阅读 [部署说明](DEPLOYMENT.md)，确认本地镜像已经按对应架构构建；不要运行旧版“一键安装”而默认从上游拉取镜像。

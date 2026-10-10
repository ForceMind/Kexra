# ForceMind/Artex：仓库开发与维护指南

> 这是中文入口和本仓库的维护基线，而不是对原开发团队身份、功能、法律责任的重新声明。工具具有自动化安全测试能力，仅用于拥有明确授权的隔离测试环境。

## 1. 项目定位与来源

- **仓库：** [ForceMind/Artex](https://github.com/ForceMind/Artex)，以私有仓库独立维护。
- **来源：** [jiwoochris/artex-ko](https://github.com/jiwoochris/artex-ko)，属于 [Autumn-27/ARTEX](https://github.com/Autumn-27/ARTEX) 的韩语本地化版本；完整固定提交见 [IMPORT_PROVENANCE.md](IMPORT_PROVENANCE.md)。
- **许可：** 保留原作者署名、版权标记与根目录 [LICENSE](../LICENSE)（AGPL-3.0）；后续修改和分发须继续核对许可证义务。
- **语言：** 原始主 [README.md](../README.md) 为韩语，另有 [README.en.md](../README.en.md) 与 [README.zh.md](../README.zh.md)。中文 README 的部分链接指向最初上游，请以本指南的当前仓库路径为准。
- **代码身份：** Go module 暂时仍为 `github.com/Autumn-27/artex`。这是源码兼容性约束，不代表本仓库属于原作者，也不应只为重命名仓库而批量替换导入路径。

## 2. 核心技术与目录

| 路径 | 用途 |
| --- | --- |
| `cmd/artex/` | Go 程序入口 |
| `server/` | HTTP API、会话与服务端逻辑，含内嵌前端资源入口 |
| `agent/`、`llmpool/`、`sidequestion/` | 智能体调度、模型供应商接口、会话交互 |
| `db/`、`evidence/`、`report/` | PostgreSQL 持久化、证据与报告 |
| `web/` | Next.js + React + TypeScript 前端 |
| `detections/` | 防御检测规则与相应测试 |
| `skills/` | 工具技能说明与扩展 |
| `.github/workflows/` | Go / Web / 文档 / 检测规则 / 发布工作流 |
| `scripts/`、`docs/` | 校验脚本、防御指南、维护记录 |

## 3. 本地开发准备

建议准备 **Go 1.26.3**（见 `go.mod`）、**Node.js 22**（见工作流）、npm、**PostgreSQL 16**。不要把示例中的默认密码用于公网部署。

在拥有权限的隔离开发环境中，从本仓库获取源码：

```bash
git clone https://github.com/ForceMind/Artex.git
cd Artex
cp .env.example .env
# 根据本地环境填写 .env 和 config.json，绝不提交真实密钥
cp config.example.json config.json
```

注意：`config.example.json` 内的默认端口和路径仅为示例；DB 连接参数须与实际环境一致。也可通过 `ARTEX_PG_DSN` 指定连接。若只想启动 Compose 中的 PostgreSQL 服务，可审阅 `.env` 后使用 `docker compose up -d postgres`，无需拉取失效的上游 `artex` 镜像。**注意原 Compose 没有把数据库端口映射给宿主机**：宿主机上直接运行的 Go 程序不能凭此访问数据库；需要独立本地 PostgreSQL，或自行增加仅绑定 `127.0.0.1` 的端口映射并同步修改 DSN。

**后端检查（需要匹配的 Go 工具链及可访问的依赖缓存）：**

```bash
go build ./...
go vet ./...
go test ./... -count=1
```

**前端检查（在 `web/` 目录执行）：**

```bash
cd web
npm ci
npm run check
npm run build:static
cd ..
```

如需合并前端静态产物并构建单一可执行文件，按原 README 的约定执行：

```bash
rm -rf server/webui/dist
mkdir -p server/webui/dist
cp -a web/out/. server/webui/dist/
CGO_ENABLED=0 go build -tags embedui -o artex ./cmd/artex
```

前端构建输出和可执行文件已被 `.gitignore` 排除，不应作为源码提交。运行前必须完成配置、数据库和授权范围校验；不要在互联网或第三方资产上直接运行测试任务。

## 4. 已知限制与发布规则

- **Docker 一键启动尚不能直接使用：** 导入版本的 `docker-compose.yml` 指向 `autumn27/artex`，原本地化文档记录该镜像在 2026-10-09 无法拉取。这个状态尚未在当前仓库重新发布与验收，切勿把 Compose 文件视为可用部署方案。
- **发布与镜像：** `release.yml` 仍按 `v*` 标签触发二进制发布。Docker Hub 发布需显式配置 Actions 变量 `ARTEX_DOCKER_IMAGE`（完整命名空间/镜像名）及 Secrets `DOCKERHUB_USERNAME`、`DOCKERHUB_TOKEN`，否则跳过；不得默认推送到原作者的 `autumn27/artex`。
- **内置一键更新风险：** `selfupdate/github.go` 仍硬编码 `Autumn-27/artex` 的 GitHub Release；使用界面中的一键更新/回滚功能前，必须先审计发布源与版本兼容性，否则可能用原上游二进制覆盖本地化版本。不要未经验证就启用该路径。
- **CI：** `ci.yml`、`web.yml`、`docs.yml`、`detections.yml` 持续校验源码与文档；另有外部链接检查，部分依赖外网。
- **秘钥与数据：** `.env`、`config.json`、`jwt.key`、运行目录、令牌、真实授权凭据和抓包数据保持本地；请在每次提交前检查 `git status` 与 diff。不要将真实渗透测试资产、客户信息、流量或证据上传到 GitHub。
- **网络与版本：** 未经明确验证，不宣称生产就绪、自动更新安全或上游发行版兼容。升级依赖/自动更新目标应单独审查。

## 5. 日常协作与维护顺序

1. 以 `main` 作为导入基线；功能变更优先短期分支与小范围 PR，保持 Go、Web、检测规则的测试独立可追踪。
2. 先修复构建和 CI，再处理 Docker 镜像、版本发行及多语言文档同步。不要在未确认运行环境前启用自动发布。
3. 代码改动同步更新对应测试与文档，运行 `python3 -I scripts/check-doc-links.py` 检查内部 Markdown 链接。
4. 引入上游变更前，比较源提交与本仓库修改，保留授权使用声明、原始版权署名和许可证。

有关 AI 编程助手的仓库工作规范见根目录 [AGENTS.md](../AGENTS.md)。

# Kexra 配置与兼容字段

> **注意**：`KEXRA_IMAGE` / `kexra:local` 等配置已进入 `main`；历史 `ARTEX_*` 环境变量与数据库名继续保留兼容。Docker 完整部署仍需独立验收。

## 配置来源

项目采用 `config.json`、环境变量、部分运行时数据库设置并存的方式。以当前 `config/config.go` 与 `docker-compose.yml` 代码为准。**改品牌不代表配置字段已全部改名。**

| 配置项 | 用途 | 注意事项 |
| --- | --- | --- |
| `ARTEX_CONFIG` | 自定义 Go 配置文件路径 | 历史字段，保留兼容 |
| `ARTEX_PG_DSN` | 完整 PostgreSQL DSN，优先于配置文件中的 database 字段 | 密码不要写进日志 |
| `ARTEX_SKILL_DIR` | 自定义 Skills 根目录 | 路径必须受限并由可信方管理 |
| `ANTHROPIC_API_KEY`、`OPENAI_API_KEY` | 可选模型密钥 | 优先通过安全的密钥配置方式提供 |
| `ARTEX_LLM_PROVIDER`、`ARTEX_LLM_MODEL` | 历史模型配置接口 | 实际支持项须查代码和界面 |
| `ARTEX_LLM_BASE_URL`、`ARTEX_LLM_PROXY` | 模型服务地址与代理 | 审计 URL，防止误发敏感数据 |
| `KEXRA_IMAGE` | Compose 应用镜像名 | 默认 `kexra:local`；尚无官方镜像 |
| `POSTGRES_USER`、`POSTGRES_PASSWORD`、`POSTGRES_DB` | Compose PostgreSQL 参数 | 更换已有数据库凭据前必须做迁移 |
| `NEXT_PUBLIC_LOCALE` | Web 前端构建语言 | 默认 `zh`；设置 `ko` 为韩语 |
| `NEXT_PUBLIC_MOCK` | 无后端的 UI 演示模式 | 仅演示，不代表实时测试 |

## 示例配置

~~~json
{
  "database": {
    "host": "127.0.0.1",
    "port": 5432,
    "user": "kexra_dev",
    "password": "REPLACE_WITH_LOCAL_SECRET",
    "dbname": "kexra_dev",
    "sslmode": "disable"
  },
  "skill_dir": "./skills"
}
~~~

上面的数据库用户名和名称只是**新建本地环境的建议示例**。已有部署仍可能使用 `artex` 数据库/用户；不能仅凭品牌更名就批量改库名、删库或改密码。生产环境应按自己的凭据存储和 TLS 规则配置。

## 存储与密钥

- `data/`：运行数据与认证密钥等，包含不能上传到仓库的敏感文件。
- `config.json` 和 `.env`：可能包含数据库与模型密钥；不应加入 Git。
- `skills/`：可能包含可执行工具说明和操作权限，安装第三方技能前应审计来源。
- Web 浏览器兼容使用新的 `kexra_token` 与旧 `artex_token`；迁移完成前不要直接废弃后者。

## 修改配置时的检查

1. 先对照当前代码/部署确认字段仍被读取，不盲目“统一替换 ARTEX”。
2. 本地保存加密备份，验证新凭据与历史数据兼容。
3. 审查代理与目标网络范围，限制出站目标。
4. 更新文档和测试后再发布。

关联：[安装](INSTALLATION.md) · [部署](DEPLOYMENT.md) · [迁移](MIGRATION.md)。

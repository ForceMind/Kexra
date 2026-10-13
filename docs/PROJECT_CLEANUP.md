# Kexra 旧文件清理决策表

该表以已合并至 `main` 的 [PR #1](https://github.com/ForceMind/Kexra/pull/1) 为基线。目标是**减少旧宣传和重复文档**，不是删除有效源码、历史证据、兼容协议或法定声明。

## A. 可以在核对引用后移除或归档

| 文件 / 目录 | 处理建议 | 前置条件 |
| --- | --- | --- |
| `README.zh.md` | 已从 `main` 移除；旧历史仍在 Git 中 | 新 [README](../README.md) 已覆盖，历史内容在 Git 中可追溯；更新所有引用 |
| `README.ko.md` | 旧韩语说明仅用于历史；可移入 `docs/legacy/` 或在成熟后删除 | 确认韩语兼容维护策略与内部锚点引用 |
| `CHANGELOG.zh.md` | 继承的旧项目历史日志，建议只作存档 | 不改变既有版本史实及相关许可声明 |
| `CHANGELOG.en.md` | 现已改写为 Kexra 英文摘要，若继续维护双语则保留 | 仅在停止维护英文文档且旧链接已更新时考虑删除 |
| `MAINTAINING.en.md`、`CONTRIBUTING.en.md`、`SECURITY.en.md`、`CODE_OF_CONDUCT.en.md` | 如不维护英文双语可退化为简短导航文件，之后再删除 | 确认 README、Issue 模板和其他链接不依赖其锚点 |
| `screenshots/wx.png` | 已从 `main` 移除的旧上游推广二维码 | 确保文档与站点不再引用 |
| `screenshots/.gitkeep` | 已从 `main` 移除冗余占位文件 | 已确认 Git 跟踪该目录中其他文件 |
| `web/public/logo.png`、`web/src/app/icon.png`、`web/src/app/favicon.ico` | 已在 `main` 采用新 SVG 品牌并移除旧图标；历史素材保留在 Git 历史 | 先确认所有界面已改用 SVG，且不存在旧路径引用 |
| `scripts/rebrand_kexra.py`、`.github/workflows/kexra-migrate-once.yml` | 一次性迁移工具，已从品牌 PR 移除 | 不应恢复到正式主线 |

以上是**建议清理名单**，不等于这些文件已经被物理删除；删除前要验证实际文件依赖。可优先清理与旧品牌相关的广告入口，而保留真实软件来源记录。

## B. 绝对不能为了更名而直接删除

- **`LICENSE`、`web/LICENSE`、`THIRD_PARTY_NOTICES.md`**：继续使用相应代码时，必须保留适用的许可证与版权通知。
- **`docs/IMPORT_PROVENANCE.md`**：源版本与 Git Tree 固定校验信息，用于识别继承的代码来源。
- **`go.mod`、`go.sum`、`web/package-lock.json`**：依赖与完整性记录，不是旧品牌垃圾文件。
- **`cmd/kexra/`、`selfupdate/`、`server/`、`db/`**：实际代码；`main` 已将 CLI 从 `cmd/artex/` 迁移到 `cmd/kexra/`，不能删除这些有效源文件。
- **`detections/` 中带 `artex` 的 Sigma / Suricata 文件、事件、CSV、MISP 资料**：历史检测特征、原始指标、测试样例必须保持一致。随意改名会导致规则/测试失配或取证语义失真。
- **`skills/`、`screenshots/` 下的历史测试参考资料**：如果当前功能或回归依赖这些文件，必须先迁移引用再处理。
- **`data/`、`pgdata`、`jwt.key`、`.env`、`config.json`**（通常是本地或部署时生成）：其中可能有密钥、数据库和历史结果。绝不能以清理品牌为由删除或提交。

## C. 清理执行顺序

1. `git ls-files` 区分 Git 管理文件与运行时数据，先备份分支和数据库。
2. 选定单批次待移除项，全文搜索文件引用，核对 GitHub Actions 和部署/脚本。
3. 对历史内容先写迁移说明；优先将陈旧文档改成导航页，而不是失去审计路径。
4. 执行 `python3 -I scripts/check-doc-links.py` 和各代码测试，确保没有断链。
5. 在 PR 中列出被删文件、原因、风险和恢复方式，通过审核后合并。

**安全且有效的去上游化**是更换产品品牌、补齐自己维护的文档和发布流程，而不是删除开源协议或篡改历史指标。更多见 [迁移说明](MIGRATION.md) 与 [第三方声明](../THIRD_PARTY_NOTICES.md)。

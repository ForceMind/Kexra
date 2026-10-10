# Kexra 品牌迁移和兼容策略

当前产品品牌：**Kexra（科巡）**。新仓库目标：`ForceMind/Kexra`；真正的 GitHub 仓库重命名需管理员在 GitHub Settings 中执行。

## 迁移原则

1. 将产品首页、标题、可视化界面、登录展示、Logo 和构建产物改为 Kexra。
2. Go module 和 CLI 可以更换新命名，但需要通过 Go/Web/检测规则回归测试；旧客户端凭据和旧数据库不应被无条件重置。
3. 原许可文本、原作者版权、第三方依赖声明与真实历史导入记录必须保留。
4. 不能为了品牌更新而修改防御检测规则的历史数据、网络指标或证据原文。
5. 自更新必须限制到 Kexra 自身发布源；未完成发行前不得拉取或执行原上游二进制。
6. ARTEX_* 配置和 artex 数据库名称在迁移期保留为兼容入口；不将兼容常量误称作产品品牌。

## 仓库管理员后续操作

在 GitHub 仓库 Settings → General → Repository name 中把 `Artex` 改为 `Kexra`。随后更新本地 clone 的 origin：

```bash
git remote set-url origin https://github.com/ForceMind/Kexra.git
```

更新所有指向旧仓库的 CI、部署、Webhooks 和配置项；执行完整构建/测试并确认没有引用已不存在的 Docker 镜像。

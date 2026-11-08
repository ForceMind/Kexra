# 模型 HTTP 与工具装配边界

本分支 `codex/runtime-boundaries-20261108` 基于 main `70a468f87f9531f32ebdc63f0fdeee258d0f8e00`。本轮实现模型请求凭据保护和工具表读取故障处理，不包含其他草稿 PR 的代码，也不作为生产发布验收。

## 模型 HTTP 请求

HTTP 工厂从 provider.go 拆到纯标准库 provider_http.go；原 quota、录制和会话头包装仍由 quotaAwareHTTPClient 安装，SDK、API 格式、流式、重试和额度归一化逻辑保持不变。

重定向只允许同一 HTTP(S) 协议、主机和有效端口，最多 10 次。默认端口可省略；主机只折叠 ASCII 字母，非 ASCII 字节和 IPv6 zone 严格保留。不同来源、降级、userinfo 或 fragment 会在请求转发前报错。合法 HTTP 自建与内网模型端点继续支持，需要跳转到另一来源的服务应显式配置最终 BaseURL。

代理留空仍为直连，不继承 HTTP_PROXY/HTTPS_PROXY；http/https/socks5 与合法代理认证保留。代理解析/协议错误返回固定诊断，避免把代理用户名、密码或原始地址带入上层错误。此检查不替代完整代理、DNS、TLS、BaseURL 或日志安全审计。

## 工具装配

ListTools 读取失败时返回明确的非 nil 空列表，取消原先的代码默认放行。AugmentTools 在 findingWorkflowTools 之后再次保持已解析空列表为非 nil，并清除对应的延迟发现/解锁元数据；资源 cleanup 继续由原调用方管理。

这是必要的 SDK 契约：固定依赖 Norma v0.4.3 的 agentcore.NewSession 在 Tools==nil 时会重新装入 DefaultTools。原 findingWorkflowTools 的启用分支会通过 append(nil, empty...) 将空列表变回 nil；只在 server 返回空列表不足以保持这个边界。

未安装 ToolResolve 的原行为、有效非空工具及已有绑定/装饰流程保持兼容。此改动保护经 ToolResolve 装配的工具列表；SDK 独立配置的 WebFetch、搜索等附加选项保持原有设置，并非全会话网络/操作系统停机开关。无数据库迁移或新增权限配置。

## 真实验证

环境为 macOS arm64 / Go 1.27.0，未使用真实模型、代理、凭据或外部资产。HTTP 测试使用实际工厂和 net/http.Client 的重定向/重放逻辑，底层接收端为内存 RoundTripper，没有 socket/TLS/DNS 集成。

| 范围 | 实际结果 |
| --- | --- |
| 模型 HTTP 工厂单测 | 22 条通过记录，含子例，零失败/跳过 |
| 同一文件集 race / vet | 通过 |
| Windows amd64 工厂测试二进制 | 交叉编译通过，未原生执行 |
| 工具装配回归 | 真实 assembly.go 与真实 findingWorkflowTools/相关声明在裁剪工程运行：2 个顶层、4 个子例，合计 6 条通过记录，race / vet 通过 |
| 原始行为反例 | HTTP 工厂未加固前 12 fail / 10 pass（含父测试）；原 assembly.go 为 5 fail / 1 pass，实际复现 nil 回流和残留发现元数据 |
| 完整 SDK quota/session 包装与 server 工具表测试 | 已编写，未执行；需要完整项目依赖 |
| 完整 Go/Web、真实 PostgreSQL、网络、Docker/生产 | 未执行 |

装配验证的临时工程保留真实生产函数和官方 SDK 核心工具实现，裁剪未使用的任务管理/工具装饰。它验证空列表契约，不能代替完整 agent/server 集成；辅助裁剪代码没有提交到产品目录。服务端新增测试使用已关闭的 database/sql 句柄，设计上无需连接真实数据库，但本机缺完整模块，尚未执行。

可重复的标准库 HTTP 验证：

~~~bash
go test agent/provider_http.go agent/provider_redirect.go agent/provider_http_test.go -count=1
go test -race agent/provider_http.go agent/provider_redirect.go agent/provider_http_test.go -count=1
go vet agent/provider_http.go agent/provider_redirect.go agent/provider_http_test.go
~~~

完整依赖就绪后执行：

~~~bash
go test ./agent -run 'TestProvider|TestQuotaClientKeepsRedirectPolicy|TestResolved' -count=1
go test ./server -run '^TestToolsTableReadFailure' -count=1
go build ./...
go vet ./...
~~~

认证 PR #19、远程 MCP/通知 PR #20、策略加载 PR #21 和供应链 PR #15 各自保留原有门禁。本轮不合并/发布，不触发 Actions，也不修改运行中的数据库或服务。回退会恢复旧凭据/默认工具边界，不作为安全生产回退。


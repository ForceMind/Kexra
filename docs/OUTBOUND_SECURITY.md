# 远程 MCP 与通知出站边界

本文件记录开发分支 `codex/remote-egress-safety-20261107` 的实现与验证范围，基于主线 `70a468f87f9531f32ebdc63f0fdeee258d0f8e00`。它不是全项目网络沙箱或生产安全认证，尚未合并/发布。

## 远程 MCP

- 配置地址必须是绝对 HTTP(S) URL，不能包含用户信息或片段。HTTP 自建服务、显式授权的内网服务和现有 TLS 配置保持可用；Basic 等凭据应放入已经支持的自定义请求头。
- Streamable HTTP、legacy SSE GET/POST 和会话 DELETE 的重定向只允许同一来源：协议、主机和有效端口一致。默认 80/443 端口可省略；主机仅折叠 ASCII 字母，非 ASCII 字节和 IPv6 zone 保持原样，避免将不同地址误判为同源。重定向最多 10 次。
- legacy SSE 只从完整的 `event: endpoint` 帧读取后续 POST 地址。普通 message、日志和未命名 data 不作为地址；帧与行均受 64 KiB 线上字节限制，CRLF 计入上限。解析继续使用共享 reader，不丢弃已预读的后续 RPC 消息。
- 相对 endpoint 基于实际同源 GET 重定向后的地址解析。绝对地址、相对地址和 network-path 引用解析后都必须同源；跨主机、跨端口、协议变化、userinfo 或 fragment 会在发送下一请求前报错。需要不同来源路由的部署应明确配置最终受信地址。
- SDK 工具适配方法原样移到 `mcphttp/adapter.go`；工具命名、审批方式、输出 Capture 和公开调用接口保持不变。这个拆分让实际传输源码可独立编译验证，没有引入 SDK 桩或新依赖。

## 通知

- 保留原有跨 Host 拒绝、5 次重定向限制和地址拨号策略。任意一跳从 HTTPS 降级到非 HTTPS 都会被拒绝，避免把鉴权头和通知正文发到明文目标；HTTP 自建目标、HTTP → HTTPS 升级和原有安全同主机跳转保持兼容。
- `ARTEX_NOTIFY_ALLOW_LOCAL` 与私网投递策略未改变。本次未扩大或缩小授权目标地址范围。
- Go HTTP 客户端对非法 `Location` 的诊断会嵌入原始地址，包括含凭据的相对地址。这类错误改为固定的 Location 解析诊断；其他内层原因继续经过现有 URL 脱敏，避免凭据进入投递错误记录。
- 本次修复不等同于对全部第三方传输错误和日志完成审计。

## 实际验证

环境：macOS arm64、Go 1.27.0。所有凭据与正文均为测试 fixture；未调用真实模型、MCP 服务、通知渠道或外部资产。沙箱禁止监听环回端口，HTTP 回归使用真实 `net/http.Client` 的重定向/请求重放逻辑配合内存 RoundTripper；legacy SSE 正常往返通过 `io.Pipe` 模拟。

| 范围 | 结果 |
| --- | --- |
| 实际 MCP 传输源码 + 两份新测试 | 15 个顶层、88 个叶用例通过；含父测试 99 条 pass，零失败/跳过 |
| 同一 MCP 文件集 race、vet | 通过 |
| MCP Windows amd64 测试二进制 | 交叉编译通过，未原生执行 |
| 通知精选无网络用例 | 98 个顶层、151 个测试/子例通过；同一选择集 race 通过，整包 vet 通过 |
| SDK 适配/既有 socket SSE、真实 HTTP/TLS/DNS/代理、真实通知投递 | 未执行；不能用内存传输或交叉编译替代 |
| 完整 Go/Web/PostgreSQL/Docker/生产 | 未执行；仍按独立门禁验收 |

新增 MCP 反例先在未修复传输源码运行：原 74 叶矩阵重复两次均 70 失败/4 通过；加 CRLF 帧反例后为 71 失败/4 通过。独立复核还通过恢复旧主机比较器，实际观察到错误来源接收请求；新增校验回归成功阻断。通知降级与非法 Location 回显同样先复现再修复。没有把沙箱 bind 拒绝写成产品测试失败，也没有把未执行的网络测试写成通过。

可重复的 MCP 定向命令：

~~~bash
go test mcphttp/client.go mcphttp/origin.go mcphttp/client_security_test.go mcphttp/origin_unicode_test.go -count=1
go test -race mcphttp/client.go mcphttp/origin.go mcphttp/client_security_test.go mcphttp/origin_unicode_test.go -count=1
go vet mcphttp/client.go mcphttp/origin.go mcphttp/client_security_test.go mcphttp/origin_unicode_test.go
~~~

在具备依赖和环回端口权限的完整工程环境继续执行：

~~~bash
go test ./mcphttp ./notify -count=1
go test -race ./mcphttp ./notify -count=1
go build ./...
go vet ./...
~~~

仍需检查模型 HTTP 客户端、工具表读取异常、代理与出站授权、真实私有部署和生产升级；本次没有引入全局 IP 封禁或强制操作系统隔离。六套原 Actions 保持手动，没有触发 CI、推送镜像或创建 Release。源码回退会恢复旧边界；不将其作为安全生产回退。当前发布判断见 [工作状态](WORK_STATUS.md) 和 [发布清单](RELEASE_CHECKLIST.md)。

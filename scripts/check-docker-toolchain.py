#!/usr/bin/env python3
"""低成本检查 Docker 工具链的直接依赖锁定与默认网络隔离。

这是静态检查，不替代 Docker 镜像构建、真实浏览器启动、依赖 SBOM、
供应链审计或生产环境验收。运行时不会访问网络或执行主动安全扫描。
"""

from pathlib import Path
import re
import sys

root = Path(__file__).resolve().parents[1]
dockerfile = (root / "Dockerfile").read_text(encoding="utf-8")
compose = (root / "docker-compose.yml").read_text(encoding="utf-8")
errors: list[str] = []


def require(condition: bool, message: str) -> None:
    if not condition:
        errors.append(message)


require("FROM node:22.23.2-bookworm-slim AS node-runtime" in dockerfile,
        "Node 运行时未采用预定的版本化官方镜像")
require("COPY --from=node-runtime /usr/local/bin/node" in dockerfile,
        "未从官方 Node 镜像复制 Node 二进制")
require("COPY --from=node-runtime /usr/local/lib/node_modules/npm" in dockerfile,
        "未复制 Node 镜像自带的 npm")
require("deb.nodesource.com" not in dockerfile and "setup_20.x" not in dockerfile,
        "仍然依赖远程 NodeSource 安装脚本")
require(not re.search(r"npm\s+install[^\n]*@latest", dockerfile),
        "Playwright 工具依赖仍使用浮动 @latest 版本")
for package in ("@playwright/mcp@0.0.83", "@playwright/cli@0.1.22", "playwright@1.64.1"):
    require(package in dockerfile, f"未固定直接依赖：{package}")
require("playwright install --with-deps chromium" in dockerfile,
        "镜像没有预装 Chromium 和系统依赖")
for port in (8787, 8788):
    require(f"127.0.0.1:{port}:{port}" in compose,
            f"Compose 端口 {port} 默认未限制到本机回环")
require("KEXRA_JWT_KEY_DIR: /app/private" in compose,
        "Compose JWT 密钥目录未独立于工作空间")
require("jwtkeys:/app/private" in compose,
        "Compose 未持久化私有 JWT 密钥目录")

if errors:
    for message in errors:
        print("❌", message)
    sys.exit(1)

print("✅ Docker 直接依赖版本和 Compose 默认隔离静态检查通过。")
print("注意：apt/npm 传递依赖和基础镜像 digest 尚未完全锁定；必须执行真实镜像构建和安全审计。")

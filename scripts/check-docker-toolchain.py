#!/usr/bin/env python3
"""低成本检查 Docker 工具链的直接依赖锁定与默认网络隔离。

这是静态检查，不替代 Docker 镜像构建、真实浏览器启动、依赖 SBOM、
供应链审计或生产环境验收。运行时不会访问网络或执行主动安全扫描。
"""

from pathlib import Path
import json
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
tool_dir = root / "docker/browser-tools"
try:
    manifest = json.loads((tool_dir / "package.json").read_text(encoding="utf-8"))
    lock = json.loads((tool_dir / "package-lock.json").read_text(encoding="utf-8"))
    require(lock.get("lockfileVersion") == 3, "浏览器工具必须使用 npm v3 锁文件")
    packages = lock.get("packages", {})
    expected = {"@playwright/mcp": "0.0.83", "@playwright/cli": "0.1.22", "playwright": "1.64.1"}
    require(manifest.get("dependencies") == expected, "浏览器工具直接版本与已审查组合不同")
    require(packages.get("", {}).get("dependencies") == expected, "浏览器工具清单与锁文件不同步")
    for name, version in expected.items():
        require(packages.get(f"node_modules/{name}", {}).get("version") == version,
                f"浏览器工具锁文件未固定 {name}@{version}")
    for name, package in packages.items():
        if not name or package.get("link"):
            continue
        require(str(package.get("resolved", "")).startswith("https://registry.npmjs.org/"),
                f"工具依赖不来自 npm HTTPS 注册表：{name}")
        require(str(package.get("integrity", "")).startswith("sha512-"),
                f"工具依赖缺少 SHA512 完整性字段：{name}")
except (OSError, ValueError) as exc:
    errors.append(f"无法读取浏览器工具锁文件：{exc}")
require("npm ci --prefix /opt/kexra-browser-tools --omit=dev --ignore-scripts" in dockerfile,
        "浏览器工具未通过禁用生命周期脚本的 npm ci 安装")
require(not re.search(r"npm\s+install\s+-g", dockerfile),
        "镜像仍有不使用锁文件的全局 npm 安装")
require("COPY docker/browser-tools/npx-locked-browser.sh /usr/local/bin/kexra-npx" in dockerfile,
        "未保留内置 browser MCP 的锁定 npx 调用入口")
require("ln -sf /usr/local/bin/kexra-npx /usr/local/bin/npx" in dockerfile,
        "内置 browser MCP 的 npx 调用未连接到锁定入口")
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

print("✅ Docker 浏览器工具锁文件、完整性字段和 Compose 默认隔离静态检查通过。")
print("注意：apt 和基础镜像 digest 尚未完全锁定；静态检查不替代镜像构建与安全审计。")

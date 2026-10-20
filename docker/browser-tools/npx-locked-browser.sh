#!/bin/sh
set -eu
# 内置 browser MCP 的历史命令固定使用镜像内已锁定的工具。
# 其他包、版本和 npx 参数完整交给原始 npm CLI，不改写用户配置。
if [ "${1:-}" = "@playwright/mcp" ]; then
    shift
    # MCP 默认查找系统 Chrome；镜像只预装锁定的 Playwright Chromium。
    # 仅为未指定浏览器的历史调用提供默认值，保留用户参数/环境/配置。
    use_default=1
    if [ -n "${PLAYWRIGHT_MCP_BROWSER:-}${PLAYWRIGHT_MCP_EXECUTABLE_PATH:-}${PLAYWRIGHT_MCP_CONFIG:-}" ]; then
        use_default=0
    fi
    for arg in "$@"; do
        case "$arg" in
            --browser|--browser=*|--executable-path|--executable-path=*|--config|--config=*) use_default=0 ;;
        esac
    done
    if [ "$use_default" = 1 ]; then
        PLAYWRIGHT_MCP_EXECUTABLE_PATH=$(node -p 'require("/opt/kexra-browser-tools/node_modules/playwright").chromium.executablePath()')
        export PLAYWRIGHT_MCP_EXECUTABLE_PATH
        # 当前镜像以 root 运行；Chromium 的 root 启动需要显式关闭其沙箱。
        # 尊重已有环境配置；此默认值不作用于显式选择的浏览器。
        if [ "$(id -u)" = 0 ] && [ "${PLAYWRIGHT_MCP_NO_SANDBOX+x}" != x ]; then
            export PLAYWRIGHT_MCP_NO_SANDBOX=true
        fi
    fi
    exec /opt/kexra-browser-tools/node_modules/.bin/playwright-mcp "$@"
fi
exec node /usr/local/lib/node_modules/npm/bin/npx-cli.js "$@"

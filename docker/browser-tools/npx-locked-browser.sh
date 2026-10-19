#!/bin/sh
# 内置 browser MCP 的历史命令固定使用镜像内已锁定的工具。
# 其他包、版本和 npx 参数完整交给原始 npm CLI，不改写用户配置。
if [ "${1:-}" = "@playwright/mcp" ]; then
    shift
    exec /opt/kexra-browser-tools/node_modules/.bin/playwright-mcp "$@"
fi
exec node /usr/local/lib/node_modules/npm/bin/npx-cli.js "$@"

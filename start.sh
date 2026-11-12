#!/bin/sh
# Kexra 启动脚本（Linux、macOS 和 Docker ENTRYPOINT）
#
# 用法
#   ./start.sh                       前台运行（按 Ctrl+C 停止）
#   nohup ./start.sh >kexra.log 2>&1 &   后台运行
#   ./start.sh -addr :9000           将附加参数原样传给 kexra
#
# 启动 kexra，并根据进程退出码决定是否重启。
#
#   0      正常退出，结束循环
#   75     程序请求重启，立即重新运行（例如在界面执行更新或回滚）
#   其他   异常退出，延时重启（1→2→4 秒，最多 60 秒）
#
# 下载、SHA256 校验和版本换装由 kexra 启动时的 selfupdate 模块处理。
set -u

cd "$(dirname "$0")" || exit 1

BIN=./kexra
[ -x "$BIN" ] || { echo "[kexra] 找不到可执行文件：$BIN" >&2; exit 1; }

RESTART_CODE=75
MAX_DELAY=60

child=0
stopping=0

# Docker 将停止信号发给作为 PID 1 的脚本。转发信号，让 kexra 有机会正常退出。
forward() {
	stopping=1
	if [ "$child" -ne 0 ]; then
		kill -TERM "$child" 2>/dev/null || true
	fi
}
trap forward INT TERM

delay=1
while :; do
	"$BIN" "$@" &
	child=$!

	# 信号会中断 wait。再次等待子进程，获取其退出码。
	wait "$child"
	code=$?
	if [ "$code" -gt 128 ]; then
		wait "$child"
		code=$?
	fi
	child=0

	if [ "$stopping" -eq 1 ]; then
		echo "[kexra] 已停止"
		exit 0
	fi

	case "$code" in
		0)
			echo "[kexra] 正常退出"
			exit 0
			;;
		"$RESTART_CODE")
			# 更新或回滚已暂存，由下一次启动时的 selfupdate.Bootstrap 完成换装。
			echo "[kexra] 收到重启请求，正在切换版本…"
			delay=1
			;;
		*)
			echo "[kexra] 异常退出（code=$code），${delay} 秒后重启" >&2
			sleep "$delay"
			delay=$((delay * 2))
			[ "$delay" -gt "$MAX_DELAY" ] && delay=$MAX_DELAY
			;;
	esac
done

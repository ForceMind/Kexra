@echo off
rem 使用 UTF-8 代码页显示启动提示。
chcp 65001 >nul 2>&1
rem Kexra 启动脚本（Windows）
rem
rem 用法
rem   start.bat                  前台运行（按 Ctrl+C 停止）
rem   start.bat -addr :9000      将附加参数原样传给 kexra
rem
rem 启动 kexra.exe，并根据进程退出码决定是否重启。
rem
rem   0      正常退出，结束循环
rem   75     程序请求重启，立即重新运行（例如在界面执行更新或回滚）
rem   其他   异常退出，延时重启（1→2→4 秒，最多 60 秒）
rem
rem 下载、SHA256 校验和版本换装由 kexra 启动时的 selfupdate 模块处理。

setlocal enabledelayedexpansion
cd /d "%~dp0"

set "BIN=kexra.exe"
if not exist "%BIN%" (
	echo [kexra] 找不到可执行文件：%BIN% 1>&2
	exit /b 1
)

set "RESTART_CODE=75"
set "MAX_DELAY=60"
set /a delay=1

:loop
"%BIN%" %*
set "code=!ERRORLEVEL!"

if "!code!"=="0" (
	echo [kexra] 正常退出
	exit /b 0
)

if "!code!"=="%RESTART_CODE%" (
	rem 更新或回滚已暂存，由下一次启动完成换装。
	echo [kexra] 收到重启请求，正在切换版本…
	set /a delay=1
	goto loop
)

echo [kexra] 异常退出 ^(code=!code!^)，!delay! 秒后重启 1>&2
rem 用本机 ping 实现重定向控制台中的延时，N 秒需要 N+1 次。
set /a pings=!delay!+1
ping -n !pings! 127.0.0.1 >nul 2>&1
set /a delay=!delay!*2
if !delay! gtr %MAX_DELAY% set /a delay=%MAX_DELAY%
goto loop

#!/usr/bin/env python3
"""通过历史 npx 入口执行真实 MCP 浏览器调用；仅使用断网临时容器。"""

import argparse
import json
import os
import selectors
import subprocess
import tempfile
import time
import uuid


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--image", default="kexra:local")
    args = parser.parse_args()
    name = "kexra-mcp-smoke-" + uuid.uuid4().hex[:12]
    with tempfile.TemporaryFile() as stderr:
        proc = subprocess.Popen(
            ["docker", "run", "--rm", "-i", "--network", "none", "--name", name,
             "--entrypoint", "npx", args.image, "@playwright/mcp", "--headless"],
            stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=stderr,
        )
        selector = selectors.DefaultSelector()
        selector.register(proc.stdout, selectors.EVENT_READ)
        buffered = bytearray()
        request_id = 0

        def send(message: dict) -> None:
            proc.stdin.write(json.dumps({"jsonrpc": "2.0", **message}).encode() + b"\n")
            proc.stdin.flush()

        def call(method: str, params: dict) -> dict:
            nonlocal request_id
            request_id += 1
            send({"id": request_id, "method": method, "params": params})
            deadline = time.monotonic() + 45
            while True:
                if b"\n" not in buffered:
                    remaining = deadline - time.monotonic()
                    if remaining <= 0 or not selector.select(remaining):
                        raise RuntimeError(f"MCP 请求超时：{method}")
                    chunk = os.read(proc.stdout.fileno(), 65536)
                    if not chunk:
                        raise RuntimeError(f"MCP 提前退出：{method}")
                    buffered.extend(chunk)
                    if len(buffered) > 2 * 1024 * 1024:
                        raise RuntimeError("MCP 响应超过 2 MiB")
                    continue
                line, _, rest = buffered.partition(b"\n")
                buffered[:] = rest
                response = json.loads(line)
                if response.get("id") != request_id:
                    if time.monotonic() >= deadline:
                        raise RuntimeError(f"MCP 请求超时：{method}")
                    continue
                if "error" in response:
                    raise RuntimeError(f"MCP 协议错误：{response['error']}")
                result = response["result"]
                if result.get("isError"):
                    raise RuntimeError(f"MCP 工具失败：{method}: {result}")
                return result

        try:
            initialized = call("initialize", {
                "protocolVersion": "2024-11-05", "capabilities": {},
                "clientInfo": {"name": "kexra-isolated-smoke", "version": "1"},
            })
            print("MCP initialize:", initialized.get("serverInfo"), flush=True)
            send({"method": "notifications/initialized"})
            tools = call("tools/list", {})
            names = {tool["name"] for tool in tools["tools"]}
            required = {"browser_navigate", "browser_evaluate", "browser_snapshot", "browser_close"}
            if not required <= names:
                raise RuntimeError(f"MCP 缺少浏览器工具：{required - names}")
            call("tools/call", {"name": "browser_navigate", "arguments": {"url": "about:blank"}})
            result = call("tools/call", {"name": "browser_evaluate", "arguments": {
                "function": "() => { document.body.innerHTML = '<h1>Kexra 科巡隔离验证</h1><button onclick=\"this.textContent=\\'已点击\\'\">点击</button>'; document.querySelector('button').click(); return document.body.textContent; }",
            }})
            text = json.dumps(result, ensure_ascii=False)
            if "Kexra 科巡隔离验证" not in text or "已点击" not in text:
                raise RuntimeError(f"浏览器交互结果不符：{result}")
            snapshot = call("tools/call", {"name": "browser_snapshot", "arguments": {}})
            if "Kexra 科巡隔离验证" not in json.dumps(snapshot, ensure_ascii=False):
                raise RuntimeError("MCP 快照缺少中文页面内容")
            call("tools/call", {"name": "browser_close", "arguments": {}})
            print("MCP tools/list → navigate → evaluate/click → snapshot → close：通过（容器无网络）", flush=True)
        finally:
            selector.close()
            try:
                proc.stdin.close()
            except BrokenPipeError:
                pass
            subprocess.run(["docker", "rm", "-f", name], capture_output=True, timeout=15, check=False)
            try:
                proc.wait(timeout=10)
            except subprocess.TimeoutExpired:
                proc.kill()
                proc.wait(timeout=5)
            stderr.seek(0)
            diagnostic = stderr.read(16384).decode(errors="replace")
            if diagnostic:
                print("MCP stderr:", diagnostic, flush=True)


if __name__ == "__main__":
    main()

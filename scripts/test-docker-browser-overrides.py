#!/usr/bin/env python3
"""在无网络临时容器中确认浏览器默认值不覆盖用户参数与配置。"""

import argparse
import subprocess

PROBE = r'''
import json, os, pathlib, subprocess
entry = pathlib.Path('/opt/kexra-browser-tools/node_modules/.bin/playwright-mcp')
entry.unlink()
entry.write_text("#!/usr/local/bin/python3\nimport json,os,sys\nprint(json.dumps({'args':sys.argv[1:],'env':{k:v for k,v in os.environ.items() if k.startswith('PLAYWRIGHT_MCP_')}}))\n")
entry.chmod(0o755)
base = {k:v for k,v in os.environ.items() if not k.startswith('PLAYWRIGHT_MCP_')}
def probe(args, extra=None):
    result = subprocess.run(['npx','@playwright/mcp',*args],env={**base,**(extra or {})},capture_output=True,text=True,timeout=15,check=True)
    data = json.loads(result.stdout)
    assert data['args'] == args, data
    return data['env']
default = probe(['--headless'])
assert pathlib.Path(default['PLAYWRIGHT_MCP_EXECUTABLE_PATH']).is_file(), default
assert default['PLAYWRIGHT_MCP_NO_SANDBOX'] == 'true', default
for args, extra in [
    (['--headless','--browser','firefox'],{}),
    (['--browser=firefox','--headless'],{}),
    (['--executable-path','/custom browser/bin','--headless'],{}),
    (['--executable-path=/custom browser/bin'],{}),
    (['--config','/custom config.json'],{}),
    (['--config=/custom config.json'],{}),
    (['--headless'],{'PLAYWRIGHT_MCP_BROWSER':'firefox'}),
    (['--headless'],{'PLAYWRIGHT_MCP_EXECUTABLE_PATH':'/custom browser/bin'}),
    (['--headless'],{'PLAYWRIGHT_MCP_CONFIG':'/custom config.json'}),
]:
    assert probe(args, extra) == extra, (args, extra)
preserved = probe(['--headless'],{'PLAYWRIGHT_MCP_NO_SANDBOX':'false'})
assert preserved['PLAYWRIGHT_MCP_NO_SANDBOX'] == 'false', preserved
version = subprocess.run(['npx','--version'],env=base,capture_output=True,text=True,timeout=15,check=True).stdout.strip()
assert version and version[0].isdigit(), version
print('真实 npx wrapper：默认值、9 种配置覆盖、显式沙箱设置、常规 npx 均通过')
'''


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--image", default="kexra:local")
    args = parser.parse_args()
    # 只替换无持久卷临时容器内的 CLI 为探针，绝不修改宿主机工具。
    subprocess.run([
        "docker", "run", "--rm", "-i", "--network", "none", "--entrypoint", "python3",
        args.image, "-I", "-",
    ], input=PROBE, text=True, timeout=90, check=True)


if __name__ == "__main__":
    main()

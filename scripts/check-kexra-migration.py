#!/usr/bin/env python3
"""静态检查 Kexra 更名的一致性；无网络访问、不调用 GitHub Actions。

该检查不替代 Go 1.26 编译、数据库集成测试、Web 构建或运行时验收。
"""
from pathlib import Path
import re
import sys

ROOT = Path(__file__).resolve().parents[1]
errors = []

def check(cond: bool, problem: str) -> None:
    if not cond:
        errors.append(problem)

def read(path: str) -> str:
    p = ROOT / path
    if not p.is_file():
        errors.append(f"缺少文件：{path}")
        return ""
    return p.read_text(encoding="utf-8", errors="replace")

mod = read("go.mod")
check(bool(re.search(r"(?m)^module github\.com/ForceMind/Kexra\s*$", mod)),
      "Go module 未使用 github.com/ForceMind/Kexra")

check((ROOT / "cmd/kexra/main.go").is_file(), "缺少 cmd/kexra/main.go")
check((ROOT / "cmd/kexra/main_test.go").is_file(), "缺少 cmd/kexra/main_test.go")
check(not (ROOT / "cmd/artex").exists(), "历史 cmd/artex 仍存在；需完成目录迁移")

go_count = 0
for p in ROOT.rglob("*.go"):
    if ".git" in p.parts:
        continue
    go_count += 1
    src = p.read_text(encoding="utf-8", errors="replace")
    if "github.com/Autumn-27/artex/" in src:
        errors.append(f"发现旧模块内部引用：{p.relative_to(ROOT)}")

for path in ["build.sh", "install.sh", "update.sh", "dev.sh", "Dockerfile"]:
    check("./cmd/kexra" in read(path), f"{path} 未引用新的 Go 启动入口")

check('const Repo = "ForceMind/Kexra"' in read("selfupdate/github.go"),
      "内置更新源尚未指向 ForceMind/Kexra")
check('"kexra-' in read("selfupdate/github.go"),
      "更新资产包名没有使用 kexra- 前缀")
check('"kexra"' in read("selfupdate/stage.go"),
      "解压更新包未寻找 kexra 可执行文件")
check('name: Kexra Release' in read(".github/workflows/release.yml"),
      "Release 工作流未使用 Kexra 名称")
check('"name": "kexra-web"' in read("web/package.json"),
      "前端包名称未迁移")
check("Kexra" in read("web/src/config/app-config.ts"),
      "Web 产品名称未迁移")
check((ROOT / "web/public/logo.svg").is_file(), "缺少 Kexra SVG 图标")
check((ROOT / "web/src/app/icon.svg").is_file(), "缺少 Kexra 应用图标")

for path in ["LICENSE", "web/LICENSE", "THIRD_PARTY_NOTICES.md",
             "docs/IMPORT_PROVENANCE.md"]:
    check((ROOT / path).is_file(), f"第三方来源/许可证不可删除：{path}")

for path in [".github/workflows/ci.yml", ".github/workflows/web.yml",
             ".github/workflows/docs.yml", ".github/workflows/detections.yml",
             ".github/workflows/external-links.yml", ".github/workflows/release.yml"]:
    s = read(path)
    check(bool(re.search(r"(?m)^on:\s*\n\s+workflow_dispatch:", s)),
          f"{path} 非手动触发，可能意外消耗 GitHub Actions 额度")

print(f"已扫描 {go_count} 个 Go 文件。")
if errors:
    for item in errors:
        print("❌", item)
    print(f"更名静态检查未通过：{len(errors)} 项。")
    sys.exit(1)
print("✅ Kexra 命名及授权声明的静态一致性检查通过（不代表编译/运行验收通过）。")

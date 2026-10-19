#!/usr/bin/env python3
"""校验原样第三方样式和许可；禁止格式化或静默替换固定来源文件。"""

from hashlib import sha256
from pathlib import Path
import sys

root = Path(__file__).resolve().parents[1]
source = root / "web/src/styles/vendor/shadcn-tailwind.css"
license_path = root / "web/licenses/shadcn-MIT.txt"
expected = "bc7d83425702955b4cb67cb14ede9d603f9d912376d57a2d81d661094d2a782a"

try:
    if sha256(source.read_bytes()).hexdigest() != expected:
        raise ValueError("shadcn 样式与固定来源哈希不同，请审查来源并更新声明")
    license_text = license_path.read_text(encoding="utf-8")
    if "Copyright (c) 2023 shadcn" not in license_text or "Permission is hereby granted" not in license_text:
        raise ValueError("shadcn MIT 版权或许可正文缺失")
except (OSError, ValueError) as exc:
    print(f"第三方样式校验失败：{exc}", file=sys.stderr)
    sys.exit(1)

print("第三方 shadcn 样式 SHA256 与 MIT 声明校验通过。")

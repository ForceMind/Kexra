#!/usr/bin/env python3
"""检查现有中韩文案是否同一套 keys；不同语言的内容不要求一致。"""
import json
from pathlib import Path
ROOT = Path(__file__).resolve().parents[1] / "web" / "messages"

def flatten(data, prefix=""):
    keys = set()
    if not isinstance(data, dict):
        return {prefix}
    for name, value in data.items():
        path = f"{prefix}.{name}" if prefix else name
        keys |= flatten(value, path)
    return keys

zh = flatten(json.loads((ROOT / "zh.json").read_text(encoding="utf-8")))
ko = flatten(json.loads((ROOT / "ko.json").read_text(encoding="utf-8")))
if zh != ko:
    print("中韩翻译资源键集不一致")
    print("中文缺少：", sorted(ko - zh)[:30])
    print("韩文缺少：", sorted(zh - ko)[:30])
    raise SystemExit(1)
print(f"中韩翻译键集一致：{len(zh)} 个可翻译字段")

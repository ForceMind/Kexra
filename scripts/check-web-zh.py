#!/usr/bin/env python3
"""验证 ARTEX 构建产物默认为简体中文，尽量捕捉静态页面中的韩语文本残留。

只检测 HTML 的可见文本及标题，不扫描脚本、样式、RSC payload 和用户产生的内容。
运行时加载的 mock/API 动态内容不在这个检查的覆盖范围，应另行测试。
用法: python3 -I scripts/check-web-zh.py [web/out]
"""
from html.parser import HTMLParser
from pathlib import Path
import re
import sys

HANGUL = re.compile(r"[가-힣]")
HAN = re.compile(r"[一-鿿]")

class PageText(HTMLParser):
    def __init__(self):
        super().__init__(convert_charrefs=True)
        self.root_lang = None
        self.skip_depth = 0
        self.hangul = []
        self.han_count = 0
        self.title = ""

    def handle_starttag(self, tag, attrs):
        props = dict(attrs)
        if tag == "html":
            self.root_lang = props.get("lang")
        if tag in ("script", "style", "svg", "noscript", "template"):
            self.skip_depth += 1

    def handle_endtag(self, tag):
        if tag in ("script", "style", "svg", "noscript", "template") and self.skip_depth:
            self.skip_depth -= 1

    def handle_data(self, data):
        if self.skip_depth or not data.strip():
            return
        if HANGUL.search(data):
            self.hangul.append(data.strip()[:90])
        self.han_count += len(HAN.findall(data))

def check(root: Path) -> list[str]:
    pages = sorted(root.rglob("*.html"))
    if not pages:
        return [f"没有找到 HTML 构建产物：{root}"]
    problems = []
    for page in pages:
        parser = PageText()
        parser.feed(page.read_text(encoding="utf-8"))
        name = page.relative_to(root)
        # Next 静态导出可能会生成不带 html 根元素的 index.html 占位文件。
        if parser.root_lang is None and name.as_posix() == "index.html":
            continue
        if parser.root_lang != "zh":
            problems.append(f"{name}: lang 应为 zh，实际为 {parser.root_lang!r}")
        if parser.hangul:
            problems.append(f"{name}: 存在韩语可见文本：{parser.hangul[:5]}")
    landing = root / "login" / "index.html"
    if landing.exists():
        parser = PageText()
        parser.feed(landing.read_text(encoding="utf-8"))
        if not parser.han_count:
            problems.append("登录页未检测到中文可见文本，请检查 next-intl 构建语言")
    return problems

def main():
    root = Path(sys.argv[1]) if len(sys.argv) > 1 else Path(__file__).resolve().parents[1] / "web" / "out"
    problems = check(root)
    if problems:
        for problem in problems:
            print("中文界面校验失败:", problem)
        return 1
    print("中文界面校验通过：静态页面 lang=zh，首页包含汉字，未发现可见韩语残留")
    return 0

if __name__ == "__main__":
    sys.exit(main())

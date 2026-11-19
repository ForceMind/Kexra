#!/usr/bin/env python3
"""检查 PostgreSQL CI 的 Go 测试结果，避免数据库跳过被计为通过。"""

import json
from pathlib import Path
import re
import sys


DATABASE_SKIP = re.compile(
    r"^\s*[^\n]+\.go:\d+:\s*(?:"
    r"no (?:test )?database config(?:ured)?\b|postgres unavailable\b|"
    r"test postgres not configured\b|database unavailable\b|"
    r"requires isolated ARTEX_PG_DSN\b|ARTEX_PG_DSN is required\b)",
    re.IGNORECASE | re.MULTILINE,
)


def check_results(events):
    database_errors = set()
    skipped = []
    passed = 0
    for event in events:
        test = event.get("Test")
        if not test:
            continue
        key = (event["Package"], test)
        action = event["Action"]
        if action == "output" and DATABASE_SKIP.search(event["Output"]):
            database_errors.add(key)
        elif action == "pass":
            passed += 1
        elif action == "skip":
            skipped.append(key)

    print(f"Go 测试通过 {passed} 项，跳过 {len(skipped)} 项。")
    for key in skipped:
        package, test = key
        if key in database_errors:
            print(f"数据库测试未执行 {package}/{test}", file=sys.stderr)
        else:
            print(f"跳过 {package}/{test}")
    return int(passed == 0 or any(key in database_errors for key in skipped))


if __name__ == "__main__":
    with Path(sys.argv[1]).open(encoding="utf-8") as stream:
        sys.exit(check_results(map(json.loads, stream)))

#!/usr/bin/env python3
"""校验真实镜像审计覆盖范围并阻止高危/严重发现；不接受空报告为通过。"""

import argparse
from collections import Counter
import json
from pathlib import Path
import sys


def summarize(report: dict, sbom: dict) -> dict:
    if report.get("ArtifactType") != "container_image":
        raise ValueError("需要容器镜像报告，不能使用源码锁文件报告替代")
    if not report.get("Metadata", {}).get("ImageID"):
        raise ValueError("报告缺少实际镜像 ID")
    coverage = {}
    counts = Counter()
    blockers = []
    for result in report.get("Results", []):
        kind = result.get("Type")
        packages = result.get("Packages", [])
        if packages:
            coverage[kind] = coverage.get(kind, 0) + len(packages)
        for vulnerability in result.get("Vulnerabilities", []):
            severity = vulnerability.get("Severity", "UNKNOWN")
            counts[severity] += 1
            if severity in {"HIGH", "CRITICAL"}:
                blockers.append({"target": result.get("Target"), **{
                    key: vulnerability.get(key) for key in (
                        "VulnerabilityID", "PkgName", "InstalledVersion", "FixedVersion",
                        "Severity", "Status", "PrimaryURL",
                    )
                }})
    required = {"debian", "node-pkg", "python-pkg", "gobinary"}
    if not required <= coverage.keys():
        raise ValueError(f"镜像扫描缺少组件覆盖：{sorted(required - coverage.keys())}")
    if not str(sbom.get("spdxVersion", "")).startswith("SPDX-") or not sbom.get("packages"):
        raise ValueError("SPDX 镜像 SBOM 为空或格式无效")
    return {
        "image_id": report["Metadata"]["ImageID"],
        "coverage": coverage,
        "spdx_packages": len(sbom["packages"]),
        "counts": dict(sorted(counts.items())),
        "blockers": blockers,
    }


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("report", type=Path)
    parser.add_argument("sbom", type=Path)
    parser.add_argument("--summary", type=Path, required=True)
    args = parser.parse_args()
    try:
        summary = summarize(json.loads(args.report.read_text()), json.loads(args.sbom.read_text()))
        args.summary.write_text(json.dumps(summary, ensure_ascii=False, indent=2) + "\n")
    except (OSError, ValueError, TypeError, KeyError) as error:
        print(f"镜像审计无效：{error}", file=sys.stderr)
        return 2
    print("镜像审计：", json.dumps({k: v for k, v in summary.items() if k != "blockers"}))
    for row in summary["blockers"]:
        print("BLOCKER", json.dumps(row, ensure_ascii=False))
    if summary["blockers"]:
        print("镜像仍有高危/严重发现；完整报告和剩余风险必须保留。", file=sys.stderr)
        return 1
    print("已识别组件无高危/严重发现；独立浏览器及内嵌组件仍需专项验收。")
    return 0


if __name__ == "__main__":
    sys.exit(main())

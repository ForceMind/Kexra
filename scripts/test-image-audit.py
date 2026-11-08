#!/usr/bin/env python3
"""审计门禁回归：缺失覆盖和无修复版本的高危不能被当作零漏洞。"""

import copy
import importlib.util
import json
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

spec = importlib.util.spec_from_file_location("image_audit", Path(__file__).with_name("check-image-audit.py"))
audit = importlib.util.module_from_spec(spec)
spec.loader.exec_module(audit)


class ImageAuditTests(unittest.TestCase):
    def setUp(self):
        self.report = {
            "ArtifactType": "container_image", "Metadata": {"ImageID": "sha256:test-fixture"},
            "Results": [{"Target": kind, "Type": kind, "Packages": [{"Name": "fixture"}]} for kind in (
                "debian", "node-pkg", "python-pkg", "gobinary",
            )],
        }
        self.report["Results"][-1]["Target"] = "app/kexra"
        self.sbom = {"spdxVersion": "SPDX-2.3", "packages": [{"name": "fixture"}]}

    def test_empty_or_source_reports_are_rejected(self):
        for report in ({}, {**self.report, "ArtifactType": "repository"}, {**self.report, "Metadata": {}}):
            with self.subTest(report=report), self.assertRaises(ValueError):
                audit.summarize(report, self.sbom)

    def test_missing_component_coverage_is_rejected(self):
        for kind in ("debian", "node-pkg", "python-pkg", "gobinary"):
            report = copy.deepcopy(self.report)
            report["Results"] = [r for r in report["Results"] if r["Type"] != kind]
            with self.subTest(kind=kind), self.assertRaises(ValueError):
                audit.summarize(report, self.sbom)

    def test_empty_sbom_is_rejected(self):
        with self.assertRaises(ValueError):
            audit.summarize(self.report, {"spdxVersion": "SPDX-2.3", "packages": []})

    def test_unfixed_and_fixed_high_findings_both_block(self):
        findings = [
            {"VulnerabilityID": "fixture-unfixed", "PkgName": "fixture", "Severity": "HIGH", "Status": "affected"},
            {"VulnerabilityID": "fixture-fixed", "PkgName": "fixture", "Severity": "CRITICAL", "FixedVersion": "2"},
            {"VulnerabilityID": "fixture-medium", "PkgName": "fixture", "Severity": "MEDIUM"},
        ]
        self.report["Results"][0]["Vulnerabilities"] = findings
        summary = audit.summarize(self.report, self.sbom)
        self.assertEqual(len(summary["blockers"]), 2)
        self.assertEqual(summary["counts"], {"CRITICAL": 1, "HIGH": 1, "MEDIUM": 1})
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / "report.json").write_text(json.dumps(self.report))
            (root / "sbom.json").write_text(json.dumps(self.sbom))
            result = subprocess.run([
                sys.executable, "-I", str(Path(__file__).with_name("check-image-audit.py")),
                str(root / "report.json"), str(root / "sbom.json"), "--summary", str(root / "summary.json"),
            ], capture_output=True, text=True, check=False)
            self.assertEqual(result.returncode, 1, result.stderr)
            self.assertEqual(len(json.loads((root / "summary.json").read_text())["blockers"]), 2)

    def test_complete_report_with_no_blockers_passes(self):
        summary = audit.summarize(self.report, self.sbom)
        self.assertEqual(summary["blockers"], [])
        self.assertEqual(set(summary["coverage"]), {"debian", "node-pkg", "python-pkg", "gobinary"})

    def test_actual_application_target_accepts_container_path_spellings(self):
        for target in ("app/kexra", "/app/kexra"):
            report = copy.deepcopy(self.report)
            report["Results"][-1]["Target"] = target
            with self.subTest(target=target):
                self.assertEqual(audit.summarize(report, self.sbom)["blockers"], [])

    def test_unrelated_or_missing_go_target_cannot_replace_application(self):
        for target in (None, "", "/tmp/kexra", "/app/kexra.bak", "/app/other/kexra", "app/../app/kexra"):
            report = copy.deepcopy(self.report)
            if target is None:
                del report["Results"][-1]["Target"]
            else:
                report["Results"][-1]["Target"] = target
            with self.subTest(target=target), self.assertRaises(ValueError):
                audit.summarize(report, self.sbom)

    def test_application_must_have_go_packages(self):
        for changes in ({"Packages": []}, {"Type": "node-pkg"}):
            report = copy.deepcopy(self.report)
            report["Results"][-1].update(changes)
            report["Results"].append({
                "Target": "/usr/bin/other", "Type": "gobinary", "Packages": [{"Name": "fixture"}],
            })
            with self.subTest(changes=changes), self.assertRaises(ValueError):
                audit.summarize(report, self.sbom)

    def test_other_go_targets_keep_their_vulnerabilities(self):
        self.report["Results"].append({
            "Target": "/usr/bin/other", "Type": "gobinary", "Packages": [{"Name": "fixture"}],
            "Vulnerabilities": [{"VulnerabilityID": "fixture-other-go", "Severity": "HIGH"}],
        })
        summary = audit.summarize(self.report, self.sbom)
        self.assertEqual(summary["coverage"]["gobinary"], 2)
        self.assertEqual(summary["counts"], {"HIGH": 1})
        self.assertEqual(summary["blockers"][0]["target"], "/usr/bin/other")

    def test_unrelated_go_target_is_invalid_through_cli(self):
        self.report["Results"][-1]["Target"] = "/usr/bin/other"
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / "report.json").write_text(json.dumps(self.report))
            (root / "sbom.json").write_text(json.dumps(self.sbom))
            result = subprocess.run([
                sys.executable, "-I", str(Path(__file__).with_name("check-image-audit.py")),
                str(root / "report.json"), str(root / "sbom.json"), "--summary", str(root / "summary.json"),
            ], capture_output=True, text=True, check=False)
            self.assertEqual(result.returncode, 2, result.stdout + result.stderr)
            self.assertFalse((root / "summary.json").exists())


if __name__ == "__main__":
    unittest.main()

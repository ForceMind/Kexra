#!/usr/bin/env python3
"""覆盖 Go CI 的实际跳过语义和分包归属。"""

from contextlib import redirect_stderr, redirect_stdout
import importlib.util
import io
from pathlib import Path
import unittest


spec = importlib.util.spec_from_file_location(
    "postgres_test_results", Path(__file__).with_name("check-postgres-test-results.py")
)
results = importlib.util.module_from_spec(spec)
spec.loader.exec_module(results)


class PostgresTestResults(unittest.TestCase):
    def check(self, events):
        with redirect_stdout(io.StringIO()), redirect_stderr(io.StringIO()):
            return results.check_results(events)

    def test_db_skip_obscured_by_successful_unit_test_fails(self):
        reasons = (
            "no database config (...) — skipping",
            "no test database configured",
            "postgres unavailable (...) - skipping",
            "test postgres not configured",
            "database unavailable (...) — skipping management API test",
            "requires isolated ARTEX_PG_DSN",
            "ARTEX_PG_DSN is required for evidence integration tests",
        )
        for reason in reasons:
            with self.subTest(reason=reason):
                self.assertEqual(self.check([
                    {"Action": "pass", "Package": "db", "Test": "TestUnit"},
                    {"Action": "output", "Package": "db", "Test": "TestDatabase",
                     "Output": f"    db_test.go:70: {reason}\n"},
                    {"Action": "skip", "Package": "db", "Test": "TestDatabase"},
                    {"Action": "pass", "Package": "db"},
                ]), 1)

    def test_model_skip_and_database_error_text_in_passing_case(self):
        self.assertEqual(self.check([
            {"Action": "output", "Package": "agent", "Test": "TestDatabase",
             "Output": "    insert_assets_test.go:21: postgres unavailable (fixture)\n"},
            {"Action": "pass", "Package": "agent", "Test": "TestDatabase"},
            {"Action": "output", "Package": "server", "Test": "TestDatabase",
             "Output": "    intercept_live_test.go:22: set ARTEX_REVIEW_LIVE_CONFIG to an external private model config\n"},
            {"Action": "skip", "Package": "server", "Test": "TestDatabase"},
        ]), 0)

    def test_no_executed_test_fails(self):
        self.assertEqual(self.check([{"Action": "skip", "Package": "db"}]), 1)


if __name__ == "__main__":
    unittest.main()

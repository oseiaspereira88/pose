#!/usr/bin/env python3
"""Regression tests for the offline `pose stats --html` report."""
import json
import subprocess
import tempfile
import unittest
from pathlib import Path


SCRIPT = Path(__file__).resolve().parent.parent / "pose-stats.py"


class TestPoseStatsHTML(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.root = Path(self.tmp.name)
        self.history = self.root / ".pose/reports/history"
        self.specs = self.root / ".pose/specs/demo"
        self.history.mkdir(parents=True)
        self.specs.mkdir(parents=True)
        (self.history / "records.jsonl").write_text(
            json.dumps({"generated_at": "2026-07-17T00:00:00Z", "workflow": "feature<script>", "task_slug": "safe-task", "outcome": "pass"})
            + "\n" + json.dumps({"generated_at": "2026-07-17T01:00:00Z", "workflow": "feature", "task_slug": "safe-task", "outcome": "partial"})
            + "\nnot-json\n",
            encoding="utf-8",
        )
        (self.specs / "spec.md").write_text(
            "created_at: 2026-07-01\ncompleted_at: 2026-07-03\n- [open] follow-up\n",
            encoding="utf-8",
        )

    def tearDown(self):
        self.tmp.cleanup()

    def test_html_is_offline_escaped_and_contains_insights(self):
        output = self.root / "nested/report.html"
        result = subprocess.run(
            ["python3", str(SCRIPT), "--history-dir", str(self.history), "--html",
             "--out", str(output), "--specs-dir", str(self.specs.parent)],
            capture_output=True, text=True,
        )
        self.assertEqual(result.returncode, 0, result.stderr)
        content = output.read_text(encoding="utf-8")
        self.assertIn("Content-Security-Policy", content)
        self.assertIn("Open follow-ups", content)
        self.assertIn("Recurrence candidates", content)
        self.assertIn("safe-task", content)
        self.assertIn("2.0 days", content)
        self.assertIn("Invalid records skipped</b><br>1", content)
        self.assertIn("feature&lt;script&gt;", content)
        self.assertNotIn("feature<script>", content)

    def test_html_default_output_is_next_to_reports(self):
        result = subprocess.run(
            ["python3", str(SCRIPT), "--history-dir", str(self.history), "--html"],
            capture_output=True, text=True,
        )
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertTrue((self.history.parent / "pose-stats.html").is_file())


if __name__ == "__main__":
    unittest.main()

#!/usr/bin/env python3
"""Contract tests for optional EARS validation."""
import subprocess
import tempfile
import unittest
from pathlib import Path

SCRIPT = Path(__file__).resolve().parent.parent / "pose-lint-spec.py"


def spec(requirement: str) -> str:
    return "\n".join(["---", "status: in-progress", "---", "## 1. Intent", "x",
                      "## 2. Requirements", f"- R1: {requirement}", "## 3. Technical Plan", "x",
                      "## 4. Tasks", "x", "## 6. Validation", "x", "## 7. Final Report", "x"])


class TestEARS(unittest.TestCase):
    def run_lint(self, requirement, *flags):
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "spec.md"
            path.write_text(spec(requirement), encoding="utf-8")
            return subprocess.run(["python3", str(SCRIPT), "--spec", str(path), *flags], capture_output=True, text=True)

    def test_ears_is_opt_in(self):
        self.assertEqual(self.run_lint("Store a result.").returncode, 0)
        self.assertNotEqual(self.run_lint("Store a result.", "--ears").returncode, 0)

    def test_supported_ears_forms_pass(self):
        for requirement in ("The service shall store a result.", "When input arrives, the service shall store it.", "While offline, the client shall queue data.", "Where export is enabled, the service shall write a file.", "If validation fails, then the service shall return an error."):
            self.assertEqual(self.run_lint(requirement, "--ears").returncode, 0, requirement)


if __name__ == "__main__":
    unittest.main()

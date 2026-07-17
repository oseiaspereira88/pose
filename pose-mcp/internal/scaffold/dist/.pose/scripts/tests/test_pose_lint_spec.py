from __future__ import annotations

import pathlib
import subprocess
import tempfile
import textwrap
import unittest


ROOT = pathlib.Path(__file__).resolve().parents[3]
LINTER = ROOT / ".pose" / "scripts" / "pose-lint-spec.py"


def spec_text(*, created: str = "2026-07-14", completed: str = "2026-07-14", extra: str = "") -> str:
    base = textwrap.dedent(
        f"""\
        ---
        slug: fixture
        status: done
        created_at: {created}
        completed_at: {completed}
        depends_on:
        ---

        # Spec: fixture

        ## 1. Intent
        Conteúdo real.

        ## 2. Requirements
        - R1: Critério estável.

        ## 3. Technical Plan
        Plano real.

        ## 4. Tasks
        - [x] Feito.

        ## 5. Decisions
        Sem decisão estrutural.

        ## 6. Validation
        Gate verde.

        ## 7. Final Report
        Entrega validada.

        ### Follow-ups
        - [done] Nada pendente.
        """
    )
    return f"{base}\n{extra}\n"


class PoseLintSpecLifecycleTest(unittest.TestCase):
    def run_lint(self, content: str) -> subprocess.CompletedProcess[str]:
        with tempfile.TemporaryDirectory() as temp:
            path = pathlib.Path(temp) / "spec.md"
            path.write_text(content, encoding="utf-8")
            return subprocess.run(
                ["python3", str(LINTER), "--spec", str(path)],
                text=True,
                capture_output=True,
                check=False,
            )

    def test_accepts_valid_lifecycle_dates(self) -> None:
        result = self.run_lint(spec_text())
        self.assertEqual(result.returncode, 0, result.stderr)

    def test_rejects_completed_before_created(self) -> None:
        result = self.run_lint(spec_text(created="2026-07-14", completed="2026-07-13"))
        self.assertEqual(result.returncode, 1)
        self.assertIn("anterior a created_at", result.stderr)

    def test_rejects_non_iso_date(self) -> None:
        result = self.run_lint(spec_text(completed="14/07/2026"))
        self.assertEqual(result.returncode, 1)
        self.assertIn("deve usar ISO 8601", result.stderr)

    def test_accepts_legacy_rfc3339_and_quoted_iso(self) -> None:
        result = self.run_lint(
            spec_text(created='"2026-07-14"', completed="2026-07-14T18:30:00Z")
        )
        self.assertEqual(result.returncode, 0, result.stderr)

    def test_rejects_duplicate_numbered_heading(self) -> None:
        result = self.run_lint(spec_text(extra="## 8. Intent\nOutra intenção."))
        self.assertEqual(result.returncode, 1)
        self.assertIn("heading canônico duplicado: intent", result.stderr)

    def test_rejects_duplicate_followups_heading(self) -> None:
        result = self.run_lint(spec_text(extra="### Follow-ups\n- [done] Duplicado."))
        self.assertEqual(result.returncode, 1)
        self.assertIn("Follow-ups aparece 2 vezes", result.stderr)


if __name__ == "__main__":
    unittest.main()

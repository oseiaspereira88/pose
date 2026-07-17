#!/usr/bin/env python3
"""Testes do pose-upgrade.sh e do gate de schema no pose-check.sh
(spec pose-schema-versioning)."""
import os
import re
import shutil
import subprocess
import tempfile
import unittest
from pathlib import Path

SCRIPTS_DIR = Path(__file__).resolve().parent.parent
ENGINE_VERSION = int(
    re.search(r"^POSE_SCHEMA_VERSION=(\d+)$",
              (SCRIPTS_DIR / "pose-lib.sh").read_text(), re.M).group(1)
)


def make_instance(tmp: Path) -> Path:
    """Repo git mínimo com o motor de scripts copiado (instância sem versão)."""
    repo = tmp / "repo"
    repo.mkdir()
    subprocess.run(["git", "-C", str(repo), "init", "-q"], check=True)
    shutil.copytree(SCRIPTS_DIR, repo / ".pose" / "scripts")
    return repo


def run_upgrade(repo: Path, *args: str) -> subprocess.CompletedProcess:
    return subprocess.run(
        ["bash", str(repo / ".pose" / "scripts" / "pose-upgrade.sh"), *args],
        capture_output=True, text=True, cwd=repo,
    )


class TestPoseUpgrade(unittest.TestCase):
    def setUp(self):
        self._tmp = tempfile.TemporaryDirectory()
        self.repo = make_instance(Path(self._tmp.name))

    def tearDown(self):
        self._tmp.cleanup()

    def version_file(self) -> Path:
        return self.repo / ".pose" / "schema-version"

    def test_upgrade_from_unversioned_stamps_and_migrates(self):
        r = run_upgrade(self.repo)
        self.assertEqual(r.returncode, 0, r.stdout + r.stderr)
        self.assertEqual(self.version_file().read_text().strip(),
                         str(ENGINE_VERSION))
        # migração 001-baseline garantiu os diretórios
        for rel in (".pose/roadmaps", ".pose/changelogs/unreleased",
                    ".pose/reports/history"):
            self.assertTrue((self.repo / rel).is_dir(), rel)

    def test_upgrade_is_idempotent(self):
        run_upgrade(self.repo)
        r = run_upgrade(self.repo)
        self.assertEqual(r.returncode, 0)
        self.assertIn("Nada a fazer", r.stdout)

    def test_dry_run_applies_nothing(self):
        r = run_upgrade(self.repo, "--dry-run")
        self.assertEqual(r.returncode, 0, r.stdout + r.stderr)
        self.assertIn("DRY-RUN", r.stdout)
        self.assertFalse(self.version_file().exists())

    def test_downgrade_is_refused(self):
        self.version_file().write_text(str(ENGINE_VERSION + 1) + "\n")
        r = run_upgrade(self.repo)
        self.assertNotEqual(r.returncode, 0)
        self.assertIn("não há downgrade", r.stderr)

    def test_check_gate_warns_tolerant_fails_strict(self):
        # instância sem schema-version: o gate de check reclama
        check = self.repo / ".pose" / "scripts" / "pose-check.sh"
        # tolerant: aviso, exit 0 (as demais falhas estruturais do repo mínimo
        # não importam aqui — filtramos só a mensagem de schema)
        r = subprocess.run(["bash", str(check), "--tolerant"],
                           capture_output=True, text=True, cwd=self.repo)
        self.assertIn("schema", r.stdout)
        self.assertIn("pose upgrade", r.stdout)

    def test_check_gate_rejects_newer_instance(self):
        self.version_file().write_text(str(ENGINE_VERSION + 5) + "\n")
        check = self.repo / ".pose" / "scripts" / "pose-check.sh"
        r = subprocess.run(["bash", str(check), "--tolerant"],
                           capture_output=True, text=True, cwd=self.repo)
        self.assertIn("mais nova que o motor", r.stdout)
        self.assertNotEqual(r.returncode, 0)


if __name__ == "__main__":
    unittest.main()

#!/usr/bin/env python3
"""Fail on an unpaired producer step; exercise missing-asset negative controls."""
import importlib.util
from pathlib import Path
import re
import sys
import unittest

sys.dont_write_bytecode = True
ROOT = Path(__file__).resolve().parents[2]
spec = importlib.util.spec_from_file_location('assets', ROOT / 'tests/release/published-asset-set.py')
assets = importlib.util.module_from_spec(spec)
spec.loader.exec_module(assets)


def unpaired(workflow):
    job = workflow.split('\n  release:\n', 1)[1]
    steps = re.split(r'(?m)^      - ', job)[1:]
    steps = steps[:next(i for i, step in enumerate(steps) if 'uses: goreleaser/goreleaser-action@' in step)]
    missing = []
    for index, step in enumerate(steps):
        if re.search(r'(?m)^\s*run:', step) or step.startswith('run:'):
            if 'assert-clean-tree.sh' in step:
                continue
            if index + 1 >= len(steps) or 'run: bash tests/release/assert-clean-tree.sh' not in steps[index + 1]:
                missing.append(step.splitlines()[0])
    return missing


class Obligations(unittest.TestCase):
    def test_current_release_has_each_run_paired(self):
        self.assertEqual(unpaired((ROOT / '.github/workflows/release.yml').read_text()), [])

    def test_inserted_step_and_removed_assertion_fail(self):
        current = (ROOT / '.github/workflows/release.yml').read_text()
        injected = current.replace('      - uses: goreleaser/', '      - name: new producer step\n        run: echo unpaired\n      - uses: goreleaser/')
        self.assertTrue(unpaired(injected))
        removed = re.sub(r'^      - run: bash tests/release/assert-clean-tree.sh.*\n', '', current, count=1, flags=re.M)
        self.assertTrue(unpaired(removed))

    def test_every_asset_is_load_bearing(self):
        expected = assets.expected(ROOT, '6.2.0')
        self.assertEqual(len(expected), 35)
        full = {'assets': [{'name': name} for name in expected]}
        self.assertFalse(assets.missing(ROOT, 'v6.2.0', full))
        for name in expected:
            with self.subTest(name=name):
                absent = {'assets': [asset for asset in full['assets'] if asset['name'] != name]}
                self.assertEqual(assets.missing(ROOT, 'v6.2.0', absent), {name})


if __name__ == '__main__':
    unittest.main()

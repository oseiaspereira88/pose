#!/usr/bin/env python3
"""Exercise provider mutations and trust boundaries without repository writes."""
import copy
import importlib.util
import json
import sys
from pathlib import Path
import unittest
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[2]
sys.dont_write_bytecode = True
spec = importlib.util.spec_from_file_location('repair', ROOT / 'scripts/repair-dependabot-runtimes.py')
repair = importlib.util.module_from_spec(spec)
spec.loader.exec_module(repair)
OLD, NEW = 'a' * 40, 'b' * 40
REPO = 'example/pose'


class RepairTests(unittest.TestCase):
    def setUp(self):
        self.pr = {'number': 1, 'state': 'open', 'user': {'login': 'dependabot[bot]'}, 'changed_files': 1,
                   'head': {'sha': NEW, 'ref': 'dependabot/github_actions/actions-x', 'repo': {'full_name': REPO}},
                   'base': {'sha': OLD, 'ref': 'main', 'repo': {'full_name': REPO}}}
        self.event = {'repository': {'full_name': REPO}, 'workflow_run': {
            'name': 'CI', 'event': 'pull_request', 'head_sha': NEW, 'head_repository': {'full_name': REPO}}}
        self.calls = []

    def provider(self, path, method='GET', payload=None):
        self.calls.append((path, method, payload))
        if path.endswith('/pulls'):
            return [copy.deepcopy(self.pr)]
        if path.endswith('/pulls/1'):
            return copy.deepcopy(self.pr)
        if '/files?' in path:
            return [{'filename': '.github/workflows/ci.yml', 'status': 'modified'}]
        if '/git/commits/' in path:
            return {'tree': {'sha': OLD}}
        if method == 'POST':
            return {'sha': 'c' * 40}
        return None

    def run_repair(self, generated='{"runtimes":{"new":{}}}'):
        with patch.object(repair, 'api', side_effect=self.provider), \
             patch.object(repair, 'generate', return_value=generated), \
             patch.object(repair, 'content', return_value='{"runtimes":{}}'):
            return repair.repair(self.event, ROOT)

    def test_only_one_manifest_and_no_force_then_dispatch(self):
        self.assertIn('repaired', self.run_repair())
        writes = [c for c in self.calls if c[1] != 'GET']
        self.assertEqual([c[1] for c in writes], ['POST', 'POST', 'PATCH', 'POST'])
        self.assertEqual([p['path'] for p in writes[0][2]['tree']], [repair.MANIFEST])
        self.assertEqual(writes[1][2]['parents'], [NEW])
        self.assertFalse(writes[2][2]['force'])
        self.assertTrue(writes[3][0].endswith('/ci.yml/dispatches'))
        self.assertEqual(writes[3][2]['ref'], self.pr['head']['ref'])

    def test_noop_has_no_writes(self):
        self.assertIn('already current', self.run_repair('{"runtimes":{}}'))
        self.assertFalse(any(c[1] != 'GET' for c in self.calls))

    def test_identity_and_revision_boundaries(self):
        for change in ['user', 'fork', 'closed', 'stale', 'branch', 'base']:
            with self.subTest(change=change):
                pr = copy.deepcopy(self.pr)
                if change == 'user': pr['user']['login'] = 'other'
                if change == 'fork': pr['head']['repo']['full_name'] = 'other/pose'
                if change == 'closed': pr['state'] = 'closed'
                if change == 'stale': pr['head']['sha'] = OLD
                if change == 'branch': pr['head']['ref'] = 'main'
                if change == 'base': pr['base']['ref'] = 'release'
                self.assertFalse(repair.eligible(pr, REPO, NEW))

    def test_stale_during_generation_never_writes(self):
        def generate(*args):
            self.pr['head']['sha'] = OLD
            return '{"runtimes":{"new":{}}}'
        with patch.object(repair, 'api', side_effect=self.provider), \
             patch.object(repair, 'generate', side_effect=generate), \
             patch.object(repair, 'content', return_value='{"runtimes":{}}'):
            with self.assertRaisesRegex(ValueError, 'changed during'):
                repair.repair(self.event, ROOT)
        self.assertFalse(any(c[1] != 'GET' for c in self.calls))

    def test_non_fast_forward_failure_never_dispatches(self):
        provider = self.provider
        def reject(path, method='GET', payload=None):
            if method == 'PATCH':
                raise RuntimeError('non-fast-forward')
            return provider(path, method, payload)
        with patch.object(self, 'provider', side_effect=reject):
            with self.assertRaisesRegex(RuntimeError, 'non-fast-forward'):
                self.run_repair()
        self.assertFalse(any('/dispatches' in c[0] for c in self.calls))

    def test_pin_only_refuses_logic_action_identity_and_mutable_ref(self):
        before = 'steps:\n  - uses: actions/checkout@' + OLD + ' # v1.0\n  - run: echo safe\n'
        after = before.replace(OLD, NEW).replace('v1.0', 'v2.0')
        self.assertTrue(repair.pin_only(before, after))
        self.assertFalse(repair.pin_only(before, after.replace('echo safe', 'echo hostile')))
        self.assertFalse(repair.pin_only(before, after.replace('actions/checkout', 'other/checkout')))
        with self.assertRaises(ValueError): repair.pin_only(before, after.replace(NEW, 'main'))

    def test_mixed_or_truncated_inventory_refused(self):
        with patch.object(repair, 'api', return_value=[]):
            with self.assertRaisesRegex(ValueError, 'only modifications'):
                repair.generate(ROOT, REPO, OLD, NEW, [{'filename': 'scripts/evil.sh', 'status': 'modified'}])
        self.pr['changed_files'] = 2
        with self.assertRaisesRegex(ValueError, 'incomplete file inventory'): self.run_repair()
        self.assertFalse(any(c[1] != 'GET' for c in self.calls))


if __name__ == '__main__':
    unittest.main()

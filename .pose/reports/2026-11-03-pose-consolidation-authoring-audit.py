#!/usr/bin/env python3
"""Reproduce pinned documentation checks; do not qualify runtime delivery."""
import hashlib
import json
import os
from pathlib import Path
import re
import subprocess
import sys
import tempfile


def main():
    if len(sys.argv) != 3:
        raise SystemExit('usage: audit.py REPOSITORY OUTPUT_JSON')
    repo, output = (Path(arg).resolve() for arg in sys.argv[1:])
    if output == repo or repo in output.parents:
        raise SystemExit('choose an output outside the repository')
    manifest = repo / '.pose/results/pose-consolidation-authoring-audit.json'
    frozen = json.loads(manifest.read_text())['records']
    expected = {'pose-consolidation-program-bootstrap'} | {
        f'pose-consolidation-batch-{n}-authoring' for n in range(1, 16)}
    if len(frozen) != 16 or {r['slug'] for r in frozen} != expected:
        raise SystemExit('the manifest must name the sixteen unique authoring scopes')
    env = dict(os.environ)
    for key in ('GIT_DIR', 'GIT_WORK_TREE', 'GIT_INDEX_FILE', 'GIT_COMMON_DIR'):
        env.pop(key, None)
    env.update(GIT_CONFIG_GLOBAL=os.devnull, GIT_CONFIG_NOSYSTEM='1',
               GIT_NO_REPLACE_OBJECTS='1', GIT_GRAFT_FILE=os.devnull,
               PYTHONDONTWRITEBYTECODE='1')
    records = []
    for item in frozen:
        commit, report = item['snapshot'], item['report']
        if not re.fullmatch(r'[0-9a-f]{40}', commit):
            raise SystemExit('invalid snapshot pin')
        if not re.fullmatch(r'\.pose/reports/[0-9-]+pose-(?:batch-[0-9]+-authoring|plan-reconciliation)\.md', report):
            raise SystemExit('invalid report path')
        source = subprocess.check_output(
            ['git', '-C', str(repo), 'show', f'{commit}:{report}'], text=True, env=env, timeout=60)
        blocks = re.findall(r'^```python[ \t]*\n(.*?)^```[ \t]*$', source, re.S | re.M)
        if len(blocks) != 1:
            raise SystemExit('expected one complete Python fenced block')
        code = blocks[0]
        digest = hashlib.sha256(code.encode()).hexdigest()
        if digest != item['original_code_sha256'] or digest != item['executed_code_sha256']:
            raise SystemExit('the pinned program does not match its recorded digest')
        compile(code, f'{commit}:{report}', 'exec')
        with tempfile.TemporaryDirectory(prefix='pose-authoring-snapshot-', dir=output.parent) as scratch:
            subprocess.run(['git', 'clone', '--shared', '--no-checkout', str(repo), scratch],
                           check=True, capture_output=True, env=env, timeout=60)
            subprocess.run(['git', '-c', 'core.hooksPath=/dev/null', '-C', scratch,
                            'checkout', '--detach', commit], check=True, capture_output=True, env=env, timeout=60)
            try:
                run = subprocess.run([sys.executable, '-B', '-c', code], cwd=scratch,
                                     env=env, capture_output=True, text=True, timeout=60)
                record = dict(item, exit_code=run.returncode, stdout=run.stdout, stderr=run.stderr)
            except subprocess.TimeoutExpired:
                record = dict(item, exit_code=None, stdout='', stderr='timeout after 60 seconds')
        records.append(record)
        print(item['slug'], commit, 'exit=' + str(record['exit_code']), flush=True)
    output.write_text(json.dumps({'schema_version': 1, 'scope': 'historical-authoring-only',
                                 'records': records}, indent=2) + '\n')
    if any(r['exit_code'] != 0 for r in records):
        raise SystemExit(1)


if __name__ == '__main__':
    main()

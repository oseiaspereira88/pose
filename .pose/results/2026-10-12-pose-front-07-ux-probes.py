"""Observe POSE operational UX in disposable Git fixtures, without agents/network.
Run with --binary /absolute/path/pose --output /tmp/front07.json.
The source tree supplies the current spec template. Probes record observations,
including working controls; they do not require defects to remain after fixes.
"""
import argparse
import hashlib
import json
import os
import pathlib
import re
import subprocess
import tempfile

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument('--binary', required=True)
parser.add_argument('--output', required=True)
parser.add_argument('--source', default=str(pathlib.Path(__file__).resolve().parents[2]))
opts = parser.parse_args()
binary = pathlib.Path(opts.binary).resolve()
source = pathlib.Path(opts.source).resolve()
parent = pathlib.Path(tempfile.mkdtemp(prefix='pose-ux-probes-'))
env = {k: v for k, v in os.environ.items() if not k.startswith(('POSE_', 'HARNE8_'))}
results = {}


def write(root, path, value):
    p = root / path
    p.parent.mkdir(parents=True, exist_ok=True)
    p.write_text(json.dumps(value, indent=2) + '\n' if isinstance(value, dict) else value)


def git(root, *args):
    return subprocess.run(['git', '-C', str(root), '-c', 'user.name=Probe', '-c',
                           'user.email=probe@example.invalid', '-c', 'commit.gpgsign=false',
                           *args], check=True, capture_output=True, text=True).stdout.strip()


def spec(root, slug='alpha', status='draft', components=''):
    write(root, f'.pose/specs/{slug}.md', f'---\nslug: {slug}\nstatus: {status}\ncreated_at: 2026-10-01\ncompleted_at: 2026-10-02\ncomponents: {components}\n---\n\n# Spec: {slug}\n\n## 1. Intent\nFix one behavior.\n\n## 2. Requirements\n- R1: Preserve the behavior.\n\n## 6. Validation\n### Requirement trace\n- R1 [waived: synthetic fixture] explicit fixture exception\n')


def fixture(name, status='draft', required=False):
    root = parent / name
    root.mkdir()
    write(root, '.pose/project.json', {'schema_version': 1, 'project_id': 'proj.ux-probe'})
    write(root, '.pose/policy/review.json', {'schema_version': 2, 'enabled': required,
          'adopted_at': '2026-08-13', 'profiles': {'spec': 'spec-closeout@2'},
          'review_bundles': True, 'review_bundles_adopted_at': '2026-08-13',
          'atomic_start_version': 1})
    write(root, '.pose/review-profiles/spec-closeout.json', {
        'schema_version': 2, 'id': 'spec-closeout', 'version': 2, 'scope': 'spec',
        'criteria': [{'id': 'judgment', 'description': 'Needs a reviewing conclusion.'}]})
    spec(root, status=status)
    git(root, 'init', '-q')
    git(root, 'add', '-A')
    git(root, 'commit', '-qm', 'fixture')
    return root


def run(root, *args, locale=None):
    local_env = dict(env)
    if locale:
        local_env['POSE_LANG'] = locale
    r = subprocess.run([str(binary), *args], cwd=root, env=local_env,
                       capture_output=True, text=True, timeout=90)
    out = {'exit_code': r.returncode, 'stdout': r.stdout, 'stderr': r.stderr}
    try:
        out['json'] = json.loads(r.stdout)
        out['single_json'] = True
    except ValueError:
        out['single_json'] = False
    return out


root = fixture('read-only-preview')
before = (root / '.pose/specs/alpha.md').read_bytes()
results['P01'] = {'plan': run(root, 'close', 'spec:alpha', '--plan', '--json'),
                  'spec_unchanged': before == (root / '.pose/specs/alpha.md').read_bytes(),
                  'bundle_count': len(list((root / '.pose/review-bundles').glob('*.json')))}

root = fixture('done-is-not-verified', status='done', required=True)
results['P02'] = {'plan': run(root, 'close', 'spec:alpha', '--plan', '--json'),
                  'closeout': run(root, 'closeout-check', 'spec:alpha', '--json')}

results['P03'] = {'apply_json': run(root, 'close', 'spec:alpha', '--apply', '--json'),
                  'resume_json': run(root, 'close', 'spec:alpha', '--resume', '--json')}

root = fixture('context-plan-flags', status='in-progress')
results['P04'] = {'context': run(root, 'context', '--task', 'spec:alpha', '--json'),
                  'plan_with_context': run(root, 'close', 'spec:alpha', '--plan',
                                            '--expect-context', 'synthetic', '--json')}

root = fixture('facts-unbound-results', status='in-progress')
write(root, '.pose/indexes/delivery-integrity.json', {
    'schema_version': 1, 'provenance_digest': 'sha256:' + 'a' * 64,
    'nodes': [], 'edges': [], 'claims': [], 'change_sets': [], 'reverse': {}, 'findings': [],
    'validation_results': [{'id': 'unrelated', 'module': 'other', 'check': 'foreign-check',
        'evidence_class': 'unit', 'severity': 'required', 'outcome': 'pass',
        'provenance_digest': 'sha256:' + 'b' * 64}]})
results['P05'] = run(root, 'specs', 'facts', 'alpha', '--json')

root = fixture('invalid-filters')
spec(root, 'beta', status='in-progress')
results['P06'] = {'unknown_flag': run(root, 'specs', '--sttaus', 'done', '--json'),
                  'invalid_since': run(root, 'specs', '--since', 'nonsense', '--json'),
                  'invalid_recent': run(root, 'specs', '--recent', 'nonsense', '--json')}

root = fixture('conflicting-modes')
results['P07'] = {'start': run(root, 'start', 'spec:alpha', '--apply', '--status', '--json'),
                  'status_after': (root / '.pose/specs/alpha.md').read_text()}
root = fixture('conflicting-close-modes', status='done')
results['P07']['close_apply_then_plan'] = run(root, 'close', 'spec:alpha', '--apply', '--plan', '--json')
results['P07']['close_plan_then_apply'] = run(root, 'close', 'spec:alpha', '--plan', '--apply', '--json')

root = fixture('continuous-status-json')
results['P08'] = run(root, 'continuous-closeout', 'status', '--json')

root = fixture('assessment-reuse')
write(root, 'svc/go.mod', 'module example.invalid/svc\n\ngo 1.22\n')
write(root, 'svc/main.go', 'package svc\nfunc Value() int { return 1 }\n')
git(root, 'add', '-A')
git(root, 'commit', '-qm', 'component fixture')
first = run(root, 'assess', 'discover', '--component', 'svc', '--if-stale')
files = list((root / '.pose/state/components').glob('*.json'))
if not files:
    files = [p for p in (root / '.pose/state').rglob('*.json') if 'svc' in p.name]
before = {str(p.relative_to(root)): hashlib.sha256(p.read_bytes()).hexdigest() for p in files}
second = run(root, 'assess', 'discover', '--component', 'svc', '--if-stale')
after = {str(p.relative_to(root)): hashlib.sha256(p.read_bytes()).hexdigest() for p in files}
results['P09'] = {'first': first, 'second': second,
                  'component_state_found': bool(files), 'state_unchanged': before == after}

root = fixture('minimal-vs-full')
write(root, '.pose/templates/spec.md', (source / '.pose/templates/spec.md').read_text())
results['P10'] = {}
for mode in ('minimal', 'full'):
    r = run(root, 'new-spec', 'tiny-' + mode, '--surface', mode)
    matches = list((root / '.pose/specs').glob('*-tiny-' + mode + '.md'))
    body = matches[0].read_text() if matches else ''
    results['P10'][mode] = {'command': r, 'bytes': len(body.encode()),
                           'headings': re.findall(r'^## .+$', body, re.M)}

root = fixture('suggest-unavailable-rule')
write(root, '.pose/indexes/task-map.json', {'tasks': {'feature': {
    'workflow': '.pose/workflows/feature.md', 'skill': 'pose-feature', 'rules': ['security'],
    'rules_by_domain': {'backend-go': ['backend-go']}, 'requires_spec': True}}})
results['P11'] = run(root, 'suggest', 'feature', '--domain', 'backend-go', '--json')
results['P11']['backend_rule_exists'] = (root / '.pose/rules/backend-go.md').exists()

root = fixture('incomplete-attention')
write(root, '.pose/policy/review.json', '{not valid JSON')
results['P12'] = run(root, 'state', '--attention', '--json')

root = fixture('machine-context-and-start')
results['P13'] = {'context': run(root, 'context', '--task', 'spec:alpha', '--json'),
                  'start': run(root, 'start', 'spec:alpha', '--json')}

root = fixture('fresh-install-state')
results['P14'] = run(root, 'state', '--json')


root = fixture('mcp-closeout-parity', status='done', required=True)
mcp_env = dict(env)
mcp_env['POSE_PROJECT_ROOT'] = str(root)
mcp_env['PATH'] = str(binary.parent) + os.pathsep + mcp_env.get('PATH', '')
messages = [
    {'jsonrpc': '2.0', 'id': 1, 'method': 'initialize', 'params': {}},
    {'jsonrpc': '2.0', 'id': 2, 'method': 'tools/list', 'params': {}},
    {'jsonrpc': '2.0', 'id': 3, 'method': 'tools/call',
     'params': {'name': 'pose_closeout_state', 'arguments': {'scope': 'spec:alpha'}}}]
wire = subprocess.run([str(binary), 'serve-mcp', '--stdio'], cwd=root, env=mcp_env,
    input=''.join(json.dumps(m) + '\n' for m in messages), capture_output=True,
    text=True, timeout=60)
replies = [json.loads(line) for line in wire.stdout.splitlines() if line.strip()]
by_id = {r.get('id'): r for r in replies}
names = [x['name'] for x in by_id.get(2, {}).get('result', {}).get('tools', [])]
results['P15'] = {'process_exit': wire.returncode, 'reply_count': len(replies),
    'initialization': by_id.get(1), 'tool_count': len(names),
    'tools_containing_close': [x for x in names if 'close' in x],
    'has_native_close_plan': 'pose_closeout_plan' in names,
    'mcp_closeout': by_id.get(3),
    'cli_closeout': run(root, 'closeout-check', 'spec:alpha', '--json')}

pathlib.Path(opts.output).write_text(json.dumps({
    'binary_sha256': hashlib.sha256(binary.read_bytes()).hexdigest(),
    'version': subprocess.run([str(binary), 'version'], capture_output=True, text=True).stdout,
    'fixtures': str(parent), 'observations': results}, indent=2) + '\n')
print(json.dumps({'output': opts.output, 'probes': len(results), 'fixtures': str(parent)}))

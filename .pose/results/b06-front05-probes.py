"""Black-box probes of B06 (spec pose-source-coverage-integrity).

Runs the same cases against a POSE binary in synthetic Git repositories under the
system temporary directory: an unreadable spec beside a readable one (front 1 D01),
two unresolved follow-ups the readers count differently (front 5 P01) and a marker
after a very long line (front 5 P04). The obstacle is a dangling link in a disposable
fixture. Usage:

    python3 -I b06-front05-probes.py --binary /path/pose --output /tmp/out.json
"""
import argparse
import hashlib
import json
import os
import pathlib
import socket
import subprocess
import tempfile

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument('--binary', required=True)
parser.add_argument('--output', required=True)
a = parser.parse_args()
binary = pathlib.Path(a.binary).resolve()
parent = pathlib.Path(tempfile.mkdtemp(prefix='pose-b06-probes-'))
env = {k: v for k, v in os.environ.items() if not k.startswith(('POSE_', 'HARNE8_'))}
env['POSE_USAGE_SUPPRESS_CLI'] = '1'
env['POSE_FOLLOWUP_TODAY'] = '2026-10-12'
probes = {}


def write(root, path, text):
    p = root / path
    p.parent.mkdir(parents=True, exist_ok=True)
    p.write_text(text)


def git(root, *args):
    return subprocess.run(['git', '-C', str(root), '-c', 'user.name=Probe', '-c', 'user.email=probe@example.invalid',
                           '-c', 'commit.gpgsign=false', *args], capture_output=True, text=True, check=True).stdout.strip()


def fixture(name):
    root = parent / name
    root.mkdir()
    write(root, '.pose/project.json', '{"schema_version":1,"project_id":"proj.b06-probe"}')
    write(root, '.pose/policy/review.json', '{"schema_version":2,"enabled":false}')
    git(root, 'init', '-q')
    return root


def commit(root):
    git(root, 'add', '-A')
    git(root, 'commit', '-qm', 'fixture')


def run(root, *args):
    r = subprocess.run([str(binary), *args], cwd=root, env=env, capture_output=True, text=True, timeout=120)
    try:
        parsed = json.loads(r.stdout)
    except ValueError:
        parsed = None
    return {'args': list(args), 'exit_code': r.returncode, 'stdout': r.stdout.replace(str(parent), '<tmp>')[:1500],
            'stderr': r.stderr.replace(str(parent), '<tmp>')[-1500:], '_json': parsed}


def jsonout(result):
    return result.pop('_json')


SPEC = '---\nslug: {slug}\nstatus: done\ncreated_at: 2026-10-01\n---\n\n# Spec: {slug}\n\n## 7. Final Report\n\n### Follow-ups\n\n{items}\n'

# D01: one readable follow-up and one spec that is a dangling link.
r = fixture('unreadable-spec')
write(r, '.pose/specs/2026-10-11-visible.md', SPEC.format(slug='visible', items='- [open] Visible repair (owner:@core crit:low review:2099-01-01)'))
os.symlink('absent-target.md', r / '.pose/specs/2026-10-11-unreadable.md')
commit(r)
res = run(r, 'followups', '--all', '--json')
doc = res['_json'] or {}
res_copy = {k: v for k, v in res.items() if k != '_json'}
probes['D01_unreadable_spec'] = {'run': res_copy, 'open': doc.get('open'), 'says_coverage': 'coverage' in doc,
                                 'coverage_complete': doc.get('coverage_complete')}

# P01: an unmarked overdue bullet and an explicit [open]: how many does each reader count?
r = fixture('two-unresolved')
write(r, '.pose/specs/2026-10-01-late.md', SPEC.format(slug='late', items='- Missing mandatory correction (owner:@core crit:high review:2020-01-01)\n- [open] Confirmed regression (owner:@core crit:high review:2020-01-01)'))
commit(r)
backlog = jsonout(run(r, 'followups', '--open', '--json')) or {}
candidates = jsonout(run(r, 'followups', '--candidates', '--json')) or {}
attention = jsonout(run(r, 'state', '--attention', '--json')) or {}
projected = [o for o in (attention.get('obligations') or []) if (o.get('source') or {}).get('producer') == 'followups']
probes['P01_two_unresolved'] = {'backlog_open': backlog.get('open'), 'candidates': len(candidates.get('candidates') or []),
                                'attention_followups': len(projected),
                                'attention_reasons': sorted(o.get('reason_code') for o in projected)}

# D01 + Attention: the same unreadable spec seen through Attention.
r = fixture('attention-unreadable')
write(r, '.pose/specs/2026-10-11-visible.md', SPEC.format(slug='visible', items='- [open] Visible repair (owner:@core crit:low review:2099-01-01)'))
os.symlink('absent-target.md', r / '.pose/specs/2026-10-11-unreadable.md')
commit(r)
attention = jsonout(run(r, 'state', '--attention', '--json')) or {}
cov = [c for c in (attention.get('coverage') or []) if c.get('producer') == 'followups']
probes['D01_attention'] = {'followups_coverage': cov, 'complete': attention.get('complete')}

# P04: a TODO after a 70 000-character comment line.
r = fixture('long-line')
write(r, 'pkg/sample.go', 'package pkg\n// ' + 'x' * 70000 + '\n// TODO: after a long line\n')
commit(r)
res = run(r, 'assess', 'tech-debt', '--json')
doc = res['_json'] or {}
probes['P04_long_line'] = {'exit_code': res['exit_code'], 'total_markers': (doc.get('summary') or {}).get('total_markers'),
                           'says_scope': 'scanned_files' in doc, 'complete': doc.get('complete')}

# Control: everything readable.
r = fixture('control')
write(r, '.pose/specs/2026-10-11-visible.md', SPEC.format(slug='visible', items='- [open] Visible repair (owner:@core crit:low review:2099-01-01)'))
write(r, 'pkg/sample.go', 'package pkg\n// TODO: one\n')
commit(r)
f = jsonout(run(r, 'followups', '--open', '--json')) or {}
t = jsonout(run(r, 'assess', 'tech-debt', '--json')) or {}
probes['P05_control'] = {'open': f.get('open'), 'markers': (t.get('summary') or {}).get('total_markers')}

redact = {socket.gethostname(): '<host>'}


def scrub(value):
    if isinstance(value, dict):
        return {k: scrub(v) for k, v in value.items()}
    if isinstance(value, list):
        return [scrub(v) for v in value]
    if isinstance(value, str):
        for secret, placeholder in redact.items():
            if secret:
                value = value.replace(secret, placeholder)
    return value


pathlib.Path(a.output).write_text(json.dumps({'binary_sha256': hashlib.sha256(binary.read_bytes()).hexdigest(),
                                              'probes': scrub(probes)}, indent=2) + '\n')
print(a.output)

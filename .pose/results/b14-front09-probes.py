"""Black-box probes of B14 (spec pose-writer-coordination-and-atomic-publication).

Runs the same cases against a POSE binary in synthetic Git repositories under
the system temporary directory: shared staging, a state refresh whose history
fails and a history with a torn tail. Obstacles are injected only into these
disposable fixtures. Usage:

    python3 -I b14-front09-probes.py --binary /path/pose --output /tmp/out.json
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
parent = pathlib.Path(tempfile.mkdtemp(prefix='pose-b14-probes-'))
env = {k: v for k, v in os.environ.items() if not k.startswith(('POSE_', 'HARNE8_'))}
env['POSE_USAGE_SUPPRESS_CLI'] = '1'
probes = {}


def write(root, path, content):
    p = root / path
    p.parent.mkdir(parents=True, exist_ok=True)
    p.write_text(json.dumps(content) + '\n' if isinstance(content, dict) else content)
    return p


def git(root, *args):
    return subprocess.run(['git', '-C', str(root), '-c', 'user.name=Probe', '-c', 'user.email=probe@example.invalid',
                           '-c', 'commit.gpgsign=false', *args], capture_output=True, text=True, check=True).stdout.strip()


def fixture(name, status='in-progress'):
    root = parent / name
    root.mkdir()
    write(root, '.pose/project.json', {'schema_version': 1, 'project_id': 'proj.b14-probe'})
    write(root, '.pose/policy/review.json', {'schema_version': 1, 'enabled': False, 'atomic_start_version': 1})
    write(root, '.pose/specs/work.md', f'---\nslug: work\nstatus: {status}\ncreated_at: 2026-09-27\n---\n\n# Spec: work\n\n## 2. Requirements\n\n- R1: Keep compatibility.\n\n### Requirement trace\n- R1 [waived: synthetic probe]\n')
    git(root, 'init', '-q')
    git(root, 'add', '-A')
    git(root, 'commit', '-qm', 'fixture')
    return root


def run(root, *args):
    r = subprocess.run([str(binary), *args], cwd=root, env=env, capture_output=True, text=True, timeout=120)
    return {'args': list(args), 'exit_code': r.returncode, 'stdout': r.stdout.replace(str(parent), '<tmp>')[-4000:],
            'stderr': r.stderr.replace(str(parent), '<tmp>')[-4000:]}


def sha(path):
    try:
        return hashlib.sha256(pathlib.Path(path).read_bytes()).hexdigest()
    except OSError:
        return None


# P01: another writer's staging file at the fixed temporary path of an index.
r = fixture('staging')
(r / '.pose/indexes').mkdir(parents=True, exist_ok=True)
(r / '.pose/indexes/.services.json.tmp').mkdir()
probes['P01_index_with_foreign_staging'] = {'index': run(r, 'index'),
                                            'foreign_staging_kept': (r / '.pose/indexes/.services.json.tmp').is_dir(),
                                            'services_published': (r / '.pose/indexes/services.json').exists()}

# P02: a refresh whose history append fails after the state is published.
r = fixture('history-fails')
init = run(r, 'state', 'init')
history = r / '.pose/state/history.jsonl'
history.unlink()
history.mkdir()
spec = r / '.pose/specs/work.md'
spec.write_text(spec.read_text().replace('status: in-progress', 'status: done'))
before = sha(r / '.pose/state/project-state.md')
refresh = run(r, 'state', 'refresh')
log = [json.loads(line) for line in (r / '.pose/state/refresh-log.jsonl').read_text().splitlines() if line.strip()]
probes['P02_refresh_history_fails'] = {'init': init, 'refresh': refresh, 'state_changed': before != sha(r / '.pose/state/project-state.md'),
                                       'last_log_result': log[-1].get('result')}

# P03: a history whose last append was cut short.
r = fixture('history-torn')
run(r, 'state', 'init')
history = r / '.pose/state/history.jsonl'
torn = '{"generated_at":"2026-10-16T00:00:00Z","baseline_'
with history.open('a') as f:
    f.write(torn)
refresh = run(r, 'state', 'refresh')
lines = history.read_text().split('\n')
parsed = 0
for line in lines:
    try:
        json.loads(line)
        parsed += 1
    except ValueError:
        pass
probes['P03_history_torn_tail'] = {'refresh': refresh, 'torn_bytes_on_own_line': (torn in lines), 'parseable_entries': parsed,
                                   'line_count': len([line for line in lines if line])}


def read(path):
    try:
        return pathlib.Path(path).read_text().strip()
    except OSError:
        return ''


# P04 control: an index with nothing in the way. The coordination of close,
# start and transfer on the spec's lifecycle lock needs an approved review to
# reach the transition, so it is qualified by the Go regressions, not here.
r = fixture('index-free')
probes['P04_control_index'] = {'index': run(r, 'index'), 'services_published': (r / '.pose/indexes/services.json').exists()}

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

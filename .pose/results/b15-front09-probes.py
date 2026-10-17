"""Black-box probes of B15 (spec pose-index-generation-integrity), front 9 CONS-05.

Runs the same cases against a POSE binary in synthetic Git repositories under
the system temporary directory. The obstacle is injected only into these
disposable fixtures: the file `packages.json` is replaced by a directory, so
the fifth output of `pose index` cannot be published. Usage:

    python3 -I b15-front09-probes.py --binary /path/pose --output /tmp/out.json
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
parent = pathlib.Path(tempfile.mkdtemp(prefix='pose-b15-probes-'))
env = {k: v for k, v in os.environ.items() if not k.startswith(('POSE_', 'HARNE8_'))}
env['POSE_USAGE_SUPPRESS_CLI'] = '1'
probes = {}
FILES = ['repo-map.json', 'services.json', 'packages.json', 'spec-graph.json', 'roadmaps.json', 'delivery-integrity.json', 'releases.json']


def write(root, path, content):
    p = root / path
    p.parent.mkdir(parents=True, exist_ok=True)
    p.write_text(json.dumps(content) + '\n' if isinstance(content, dict) else content)
    return p


def git(root, *args):
    return subprocess.run(['git', '-C', str(root), '-c', 'user.name=Probe', '-c', 'user.email=probe@example.invalid',
                           '-c', 'commit.gpgsign=false', *args], capture_output=True, text=True, check=True).stdout.strip()


def fixture(name):
    root = parent / name
    root.mkdir()
    write(root, '.pose/project.json', {'schema_version': 1, 'project_id': 'proj.b15-probe'})
    write(root, '.pose/specs/work.md', '---\nslug: work\nstatus: in-progress\ncreated_at: 2026-09-27\n---\n\n# Spec: work\n')
    write(root, 'svc/go.mod', 'module example.com/svc\n')
    git(root, 'init', '-q')
    git(root, 'add', '-A')
    git(root, 'commit', '-qm', 'fixture')
    return root


def run(root, *args):
    r = subprocess.run([str(binary), *args], cwd=root, env=env, capture_output=True, text=True, timeout=120)
    return {'args': list(args), 'exit_code': r.returncode, 'stdout': r.stdout.replace(str(parent), '<tmp>')[-1500:],
            'stderr': r.stderr.replace(str(parent), '<tmp>')[-1500:]}


def sha(path):
    try:
        return hashlib.sha256(pathlib.Path(path).read_bytes()).hexdigest()
    except OSError:
        return None


def digests(root):
    return {name: sha(root / '.pose/indexes' / name) for name in FILES}


# P01: a first good index, a change that moves several outputs, then an output
# that cannot be written. Five times, because the order the outputs are written
# in was not fixed before this change.
runs = []
for attempt in range(5):
    r = fixture(f'blocked-{attempt}')
    first = run(r, 'index')
    before = digests(r)
    write(r, '.pose/specs/second.md', '---\nslug: second\nstatus: draft\ncreated_at: 2026-10-17\n---\n\n# Spec: second\n')
    write(r, 'svc2/go.mod', 'module example.com/svc2\n')
    descriptor_before = sha(r / '.pose/indexes/generation.json')
    packages = r / '.pose/indexes/packages.json'
    packages.unlink()
    packages.mkdir()
    blocked = run(r, 'index')
    after = digests(r)
    descriptor_unchanged = descriptor_before is not None and descriptor_before == sha(r / '.pose/indexes/generation.json')
    changed = sorted(n for n in FILES if n != 'packages.json' and before[n] != after[n])
    verify = run(r, 'index', '--verify')
    runs.append({'first_index_exit': first['exit_code'], 'blocked_exit': blocked['exit_code'], 'blocked_stderr': blocked['stderr'][-300:],
                 'outputs_changed_besides_the_blocked': changed, 'descriptor_unchanged': descriptor_unchanged,
                 'verify_exit': verify['exit_code'], 'verify_stderr': verify['stderr'][-300:]})
# Before this change the descriptor does not exist, so a set is only as whole as
# the files happen to be: a run is "mixed" when some outputs moved and others did not.
probes['P01_blocked_output'] = {'runs': runs,
                               'runs_with_some_but_not_all_outputs_updated': sum(1 for x in runs if 0 < len(x['outputs_changed_besides_the_blocked']) < 6),
                               'runs_with_the_previous_generation_still_in_force': sum(1 for x in runs if x['descriptor_unchanged'])}

# P02: the result is read back. A set the descriptor does not describe is not served.
r = fixture('hand-edited')
run(r, 'index')
write(r, '.pose/indexes/repo-map.json', '{"apps":[],"services":[],"packages":[]}\n')
probes['P02_hand_edited_index'] = {'verify': run(r, 'index', '--verify'), 'reindex': run(r, 'index'), 'verify_after': run(r, 'index', '--verify')}

# P03 control: an index with nothing in the way publishes a current set.
r = fixture('free')
probes['P03_control_index'] = {'index': run(r, 'index'), 'again': run(r, 'index'), 'verify': run(r, 'index', '--verify'),
                               'descriptor_present': (r / '.pose/indexes/generation.json').exists()}

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

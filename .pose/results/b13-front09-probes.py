"""Black-box probes of B13 (spec pose-journal-checkpoint-and-lock-recovery).

Runs the same cases against a POSE binary in synthetic Git repositories under
the system temporary directory: journal damage, checkpoint states and start
lock owners. Damage and fake owners are injected only into these disposable
fixtures. Linux only (the lock cases read /proc). Usage:

    python3 -I b13-front09-probes.py --binary /path/pose --output /tmp/out.json
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
parent = pathlib.Path(tempfile.mkdtemp(prefix='pose-b13-probes-'))
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


def fixture(name, status='draft'):
    root = parent / name
    root.mkdir()
    write(root, '.pose/project.json', {'schema_version': 1, 'project_id': 'proj.b13-probe'})
    write(root, '.pose/policy/review.json', {'schema_version': 1, 'enabled': False, 'atomic_start_version': 1})
    write(root, '.pose/policy/actions.json', {'schema_version': 1, 'roles': {'maintainer': ['human:maintainer']}})
    write(root, '.pose/specs/work.md', f'---\nslug: work\nstatus: {status}\ncreated_at: 2026-09-27\n---\n\n# Spec: work\n\n## 2. Requirements\n\n- R1: Keep compatibility.\n\n### Requirement trace\n- R1 [waived: synthetic probe]\n')
    git(root, 'init', '-q')
    git(root, 'add', '-A')
    git(root, 'commit', '-qm', 'fixture')
    return root


def run(root, *args):
    r = subprocess.run([str(binary), *args], cwd=root, env=env, capture_output=True, text=True, timeout=90)
    row = {'args': list(args), 'exit_code': r.returncode, 'stdout': r.stdout.replace(str(parent), '<tmp>'),
           'stderr': r.stderr.replace(str(parent), '<tmp>')}
    try:
        row['data'] = json.loads(r.stdout)
    except ValueError:
        pass
    return row


OPEN = ['action', 'open', '--origin', 'spec:work', '--kind', 'decision', '--question', 'Probe question?',
        '--requested-by', 'agent:probe', '--recipient-role', 'maintainer', '--option', 'keep=Keep compatibility',
        '--option', 'change=Change compatibility', '--target', 'requirement:R1', '--effect', 'closeout:block', '--apply', '--json']


def journal_case(name, damage):
    r = fixture('journal-' + name)
    opened = run(r, *OPEN)['data']['request']
    path = r / '.pose/actions' / (opened['id'] + '.jsonl')
    path.write_bytes(damage(path.read_bytes()))
    return {'show_json': run(r, 'action', 'show', opened['id'], '--json'), 'show': run(r, 'action', 'show', opened['id']),
            'list_json': run(r, 'action', 'list', '--json')}


# P01-P05: the journal is read whole; P06 is the control.
probes['P01_journal_extra_json_same_line'] = journal_case('extra', lambda b: b.rstrip(b'\n') + b' {"unconsumed":"tail"}\n')
probes['P02_journal_interrupted_tail'] = journal_case('tail', lambda b: b + b'{"schema_version":1,"seq":2,')
probes['P03_journal_duplicate_field'] = journal_case('duplicate', lambda b: b.replace(b'"seq":1,', b'"seq":1,"seq":1,', 1))
probes['P04_journal_unsupported_schema'] = journal_case('schema', lambda b: b.replace(b'"schema_version":1', b'"schema_version":2', 1))
probes['P05_journal_empty_line'] = journal_case('blank', lambda b: b + b'\n')
probes['P06_control_intact_journal'] = journal_case('intact', lambda b: b)


def checkpoint_case(name, content):
    r = fixture('checkpoint-' + name, status='in-progress')
    if content is not None:
        write(r, '.pose/closeout-plans/spec-work.json', content)
    return {'plan': run(r, 'close', 'spec:work', '--plan'), 'plan_json': run(r, 'close', 'spec:work', '--plan', '--json'),
            'resume_json': run(r, 'close', 'spec:work', '--resume', '--json')}


valid = {'schema_version': 1, 'scope': 'spec:work', 'started_at': '2026-10-12T12:00:00Z', 'plan_digest': 'probe', 'completed': ['context']}
# P07-P10: the checkpoint is never read as absent; P11-P12 are controls.
probes['P07_checkpoint_corrupt'] = checkpoint_case('corrupt', '{"schema_version":1,')
probes['P08_checkpoint_unsupported'] = checkpoint_case('unsupported', dict(valid, schema_version=2))
probes['P09_checkpoint_other_scope'] = checkpoint_case('scope', dict(valid, scope='spec:other'))
probes['P10_checkpoint_effect_not_reflected'] = checkpoint_case('lost', dict(valid, completed=['context', 'seal']))
probes['P11_control_valid_checkpoint'] = checkpoint_case('valid', valid)
probes['P12_control_absent_checkpoint'] = checkpoint_case('absent', None)


def read(path):
    try:
        return pathlib.Path(path).read_text().strip()
    except OSError:
        return ''


def stat_start(pid):
    text = read(f'/proc/{pid}/stat')
    return text[text.rindex(')') + 1:].split()[19]


def app_id(kind, raw):
    return hashlib.sha256(f'pose-start-lock/{kind}/{raw}'.encode()).hexdigest()[:32] if raw else ''


def owner(pid, start):
    return {'schema_version': 1, 'resource': 'spec:work', 'worktree': '<fixture>', 'operation': 'start.apply',
            'host': socket.gethostname(), 'machine_id': app_id('machine', read('/etc/machine-id')),
            'boot_id': app_id('boot', read('/proc/sys/kernel/random/boot_id')),
            'pid_namespace': os.readlink('/proc/self/ns/pid'), 'pid': pid, 'process_start': start, 'acquired_at': '2026-10-12T12:00:00Z'}


def lock_case(name, content, recover=False):
    r = fixture('lock-' + name)
    preview = run(r, 'start', 'spec:work', '--json')
    write(r, '.pose/starts/.work.lock', content if isinstance(content, str) else json.dumps(content) + '\n')
    row = {'apply': run(r, 'start', 'spec:work', '--apply', '--digest', preview['data']['digest'], '--json'),
           'status': run(r, 'start', 'spec:work', '--status', '--json')}
    lock = (row['status'].get('data') or {}).get('lock') or {}
    if recover and lock.get('owner_digest'):
        row['recover'] = run(r, 'start', 'spec:work', '--recover-lock', lock['owner_digest'], '--json')
        row['apply_after'] = run(r, 'start', 'spec:work', '--apply', '--digest', preview['data']['digest'], '--json')
    row['lock_file_exists'] = (r / '.pose/starts/.work.lock').exists()
    return row


sleeper = subprocess.Popen(['sleep', '300'])
try:
    gone = subprocess.Popen(['true'])
    gone.wait()
    # P13: legacy empty lock; P14: owner running; P15: owner gone; P16: PID reused; P17: another host.
    probes['P13_lock_without_owner'] = lock_case('legacy', '', recover=True)
    probes['P14_lock_owner_running'] = lock_case('active', owner(sleeper.pid, stat_start(sleeper.pid)), recover=True)
    probes['P15_lock_owner_gone'] = lock_case('dead', owner(gone.pid, '1'), recover=True)
    probes['P16_lock_pid_reused'] = lock_case('reused', owner(sleeper.pid, '1'), recover=True)
    probes['P17_lock_other_host'] = lock_case('host', dict(owner(gone.pid, '1'), host='elsewhere.invalid'), recover=True)
finally:
    sleeper.kill()

# The published output names no host, machine, boot or namespace.
redact = {socket.gethostname(): '<host>', app_id('machine', read('/etc/machine-id')): '<machine>',
          app_id('boot', read('/proc/sys/kernel/random/boot_id')): '<boot>', os.readlink('/proc/self/ns/pid'): '<pid-namespace>'}


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


probes = scrub(probes)
pathlib.Path(a.output).write_text(json.dumps({'binary_sha256': subprocess.run(['sha256sum', str(binary)], capture_output=True, text=True).stdout.split()[0],
                                              'probes': probes}, indent=2) + '\n')
print(a.output)

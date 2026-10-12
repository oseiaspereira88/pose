"""Black-box Linux consistency/recovery probes against an existing POSE binary.

Creates synthetic Git repositories under /tmp. No baseline writes, network,
external agents, engine rebuild or power-loss claim. File damage is injected
only into these disposable fixtures. Run --binary /path/pose --output /tmp/results.json.
"""
import argparse
import concurrent.futures
import fcntl
import hashlib
import json
import os
import pathlib
import shutil
import subprocess
import tempfile
import threading
import time

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument('--binary', required=True)
parser.add_argument('--output', required=True)
a = parser.parse_args()
binary = pathlib.Path(a.binary).resolve()
parent = pathlib.Path(tempfile.mkdtemp(prefix='pose-consistency-probes-'))
env = {k: v for k, v in os.environ.items() if not k.startswith(('POSE_', 'HARNE8_'))}
env['POSE_USAGE_SUPPRESS_CLI'] = '1'
observations = {}


def write(root, path, content):
    p = root / path
    p.parent.mkdir(parents=True, exist_ok=True)
    p.write_text(json.dumps(content) + '\n' if isinstance(content, dict) else content)
    return p


def git(root, *args):
    return subprocess.run(['git', '-C', str(root), '-c', 'user.name=Probe', '-c',
        'user.email=probe@example.invalid', '-c', 'commit.gpgsign=false', *args],
        capture_output=True, text=True, check=True).stdout.strip()


def fixture(name):
    root = parent / name
    root.mkdir()
    write(root, '.pose/project.json', {'schema_version': 1, 'project_id': 'proj.consistency-probe'})
    write(root, '.pose/policy/review.json', {'schema_version': 1, 'enabled': False, 'atomic_start_version': 1})
    write(root, '.pose/policy/actions.json', {'schema_version': 1, 'roles': {'maintainer': ['human:maintainer']}})
    write(root, '.pose/specs/work.md', '---\nslug: work\nstatus: draft\ncreated_at: 2026-09-27\n---\n\n# Spec: work\n\n## 2. Requirements\n\n- R1: Keep compatibility.\n\n### Requirement trace\n- R1 [waived: synthetic probe]\n')
    git(root, 'init', '-q')
    git(root, 'add', '-A')
    git(root, 'commit', '-qm', 'fixture')
    return root


def run(root, *args, extra=None):
    e = dict(env)
    e.update(extra or {})
    t = time.perf_counter()
    r = subprocess.run([str(binary), *args], cwd=root, env=e, capture_output=True, text=True, timeout=90)
    row = {'args': list(args), 'exit_code': r.returncode, 'seconds': time.perf_counter() - t,
           'stdout': r.stdout, 'stderr': r.stderr}
    try:
        row['data'] = json.loads(r.stdout)
    except ValueError:
        pass
    return row


def opening(question, supersedes=None):
    args = ['action', 'open', '--origin', 'spec:work', '--kind', 'decision',
            '--question', question, '--requested-by', 'agent:probe', '--recipient-role', 'maintainer',
            '--option', 'keep=Keep compatibility', '--option', 'change=Change compatibility',
            '--target', 'requirement:R1', '--effect', 'closeout:block', '--apply', '--json']
    if supersedes:
        args += ['--supersedes', supersedes]
    return args


# Two independent native processes race against the same expected journal revision.
r = fixture('action-writers')
opened = run(r, *opening('Keep compatibility?'))
assert opened['exit_code'] == 0, opened
request = opened['data']['request']
base = ['action', 'resolve', request['id'], '--actor', 'human:maintainer',
        '--request-digest', request['request_digest'], '--expected-revision', '1',
        '--answer', 'keep', '--apply', '--json']
barrier = threading.Barrier(2)

def writer(key):
    barrier.wait(timeout=10)
    return run(r, *base, '--idempotency-key', key)

with concurrent.futures.ThreadPoolExecutor(max_workers=2) as pool:
    futures = [pool.submit(writer, key) for key in ('writer-a', 'writer-b')]
    rows = [f.result() for f in futures]
assert sorted(x['exit_code'] for x in rows) == [0, 1], rows
winner = rows[[x['exit_code'] for x in rows].index(0)]
key = winner['args'][-1]
retry = run(r, *base, '--idempotency-key', key)
assert retry['exit_code'] == 0 and retry['data']['revision'] == 2, retry
conflict_args = base.copy()
conflict_args[conflict_args.index('--answer') + 1] = 'change'
conflict = run(r, *conflict_args, '--idempotency-key', key)
assert conflict['exit_code'] == 1 and 'action-idempotency-conflict' in conflict['stdout'] + conflict['stderr'], conflict
observations['action_revision_and_idempotency'] = {'writers': rows, 'same_write_retry': retry, 'different_write_same_key': conflict}

# A lock conflict between the two writes of supersession leaves a new journal.
# Retry must address the same generated ID; a second boundary causes a bounded retry of the fixture.
for attempt in range(3):
    r = fixture(f'supersession-{attempt}')
    prior = run(r, *opening('First question?'))['data']['request']
    old_path = r / '.pose/actions' / (prior['id'] + '.jsonl')
    args = opening('Replacement question?', prior['id'])
    with old_path.open('r+') as locked:
        fcntl.flock(locked, fcntl.LOCK_EX | fcntl.LOCK_NB)
        partial = run(r, *args)
        assert partial['exit_code'] == 1 and 'opened but' in partial['stdout'] + partial['stderr'], partial
        new_paths = [p for p in old_path.parent.glob('*.jsonl') if p != old_path]
        assert len(new_paths) == 1
        new_id = new_paths[0].stem
        fcntl.flock(locked, fcntl.LOCK_UN)
    retry = run(r, *args)
    if retry.get('data', {}).get('request', {}).get('id') != new_id:
        continue
    listing = run(r, 'action', 'list', '--json')
    old_view = run(r, 'action', 'show', prior['id'], '--json')
    assert retry['exit_code'] == 0 and old_view['data']['state'] == 'open'
    observations['supersession_partial_retry'] = {'partial': partial, 'same_id_retry': retry,
        'old_after_retry': old_view, 'listing': listing, 'same_new_id': True}
    break
else:
    raise AssertionError('Could not keep the generated ID across a CLI retry')

# Parser behavior for an incomplete tail versus ignored trailing bytes in one line.
r = fixture('journal-damage')
opened = run(r, *opening('Parser question?'))['data']['request']
p = r / '.pose/actions' / (opened['id'] + '.jsonl')
original = p.read_text()
p.write_text(original + '{"schema_version":1,"seq":2,')
truncated = run(r, 'action', 'show', opened['id'], '--json')
assert truncated['exit_code'] == 1, truncated
p.write_text(original.rstrip('\n') + ' {"unconsumed":"tail"}\n')
trailing = run(r, 'action', 'show', opened['id'], '--json')
assert trailing['exit_code'] == 0 and trailing['data']['revision'] == 1, trailing
observations['journal_parser'] = {'incomplete_next_line': truncated, 'extra_json_same_line': trailing}

# Start's per-spec lock refuses until its abandoned file is explicitly removed in the fixture.
r = fixture('start-lock')
preview = run(r, 'start', 'spec:work', '--json')
assert preview['exit_code'] == 0, preview
lock = write(r, '.pose/starts/.work.lock', '')
blocked = run(r, 'start', 'spec:work', '--apply', '--digest', preview['data']['digest'])
assert blocked['exit_code'] == 1 and 'start-in-progress' in blocked['stdout'] + blocked['stderr'], blocked
lock.unlink()  # Synthetic empty lock only; not an operator instruction for real repositories.
started = run(r, 'start', 'spec:work', '--apply', '--digest', preview['data']['digest'])
assert started['exit_code'] == 0, started
retry = run(r, 'start', 'spec:work', '--apply', '--digest', preview['data']['digest'])
assert retry['exit_code'] == 0, retry
observations['start_lock_and_retry'] = {'blocked': blocked, 'started': started, 'retry': retry}

# A damaged checkpoint is indistinguishable from absent to the text plan renderer.
r = fixture('checkpoint')
absent = run(r, 'close', 'spec:work', '--plan')
p = write(r, '.pose/closeout-plans/spec-work.json', {'schema_version': 1, 'scope': 'spec:work',
    'started_at': '2026-10-12T12:00:00Z', 'plan_digest': 'probe', 'completed': ['context']})
valid = run(r, 'close', 'spec:work', '--plan')
p.write_text('{"schema_version":1,')
damaged = run(r, 'close', 'spec:work', '--plan')
assert 'closeout_plan.interrupted=' in valid['stdout'], valid
assert damaged['exit_code'] == 0 and 'closeout_plan.interrupted=' not in damaged['stdout'], damaged
observations['checkpoint_read'] = {'absent': absent, 'synthetic_valid': valid, 'damaged': damaged,
                                  'absent_and_damaged_output_equal': absent['stdout'] == damaged['stdout']}

# The revision is valid, but git status fails. All other Git operations delegate to real Git.
r = fixture('git-status')
p = r / '.pose/specs/work.md'
p.write_text(p.read_text() + '\nTracked edit.\n')
normal = run(r, 'state', '--attention', '--json')
real_git = shutil.which('git')
shim_dir = parent / 'git-shim'
shim = write(shim_dir, 'git', '#!/usr/bin/env python3\nimport os,sys\nif "status" in sys.argv[1:]:\n sys.stderr.write("synthetic git status unavailable\\n");sys.exit(97)\nos.execv(' + repr(real_git) + ', [' + repr(real_git) + '] + sys.argv[1:])\n')
shim.chmod(0o755)
failed = run(r, 'state', '--attention', '--json', extra={'PATH': str(shim_dir) + os.pathsep + env.get('PATH', '')})
assert normal['data']['snapshot']['worktree_dirty'] is True, normal
assert failed['data']['snapshot'].get('worktree_dirty', False) is False, failed
observations['git_status_failure'] = {'normal_snapshot': normal['data']['snapshot'],
    'failed_status_snapshot': failed['data']['snapshot'], 'failed_status_exit_code': failed['exit_code'],
    'failed_status_complete': failed['data']['complete'], 'failed_status_coverage': failed['data']['coverage']}

# Construct an intermediate release prepare state, not a killed real process or power-loss experiment.
r = fixture('release-intermediate')
write(r, '.pose/release-policy.json', {'schema_version': 1, 'adopted_at': '2026-08-03', 'provider': 'github', 'repository': 'owner/repo'})
write(r, 'pose-mcp/internal/version/version.go', 'package version\n')
write(r, '.pose/specs/alpha/spec.md', '---\nslug: alpha\nstatus: done\n---\n')
p = write(r, '.pose/changelogs/unreleased/alpha.md', '---\nspec: alpha\ncategory: added\nbreaking: false\n---\n\nAdds alpha.\n')
preview = run(r, 'release', 'prepare', '--version', 'v7.4.1')
assert preview['exit_code'] == 0, preview
archive = r / '.pose/changelogs/v7.4.1'
archive.mkdir()
(r / '.pose/releases/v7.4.1').mkdir(parents=True)
p.rename(archive / p.name)
retry = run(r, 'release', 'prepare', '--version', 'v7.4.1', '--apply')
default_check = run(r, 'release', 'check', '--version', 'v7.4.1')
check = run(r, 'release', 'check', '--version', 'v7.4.1', '--strict')
assert retry['exit_code'] != 0 and check['exit_code'] != 0, (retry, check)
observations['release_intermediate_state'] = {'preview': preview, 'retry': retry, 'default_check': default_check, 'strict_check': check,
    'fragment_still_archived': (archive / p.name).exists(), 'manifest_exists': (r / '.pose/releases/v7.4.1/manifest.json').exists()}

# Fail one index publication deterministically. Map iteration decides how many other files preceded it.
r = fixture('index-partial')
names = ['repo-map.json', 'services.json', 'packages.json', 'spec-graph.json',
         'roadmaps.json', 'delivery-integrity.json', 'releases.json']
for name in names:
    write(r, '.pose/indexes/' + name, {'probe_generation': 'old'})
blocked = r / '.pose/indexes/.services.json.tmp'
blocked.mkdir()
failed = run(r, 'index', '--json')
assert failed['exit_code'] == 1, failed
changed = [n for n in names if json.loads((r / '.pose/indexes' / n).read_text()).get('probe_generation') != 'old']
unchanged = [n for n in names if n not in changed]
blocked.rmdir()
recovered = run(r, 'index', '--json')
assert recovered['exit_code'] == 0, recovered
observations['index_publication_failure'] = {'failed': failed, 'changed_before_failure': changed,
    'still_old_after_failure': unchanged, 'retry_after_removing_synthetic_obstacle': recovered}

# State publication precedes history append; force the latter to fail using a directory at its file path.
r = fixture('state-history-failure')
initialized = run(r, 'state', 'init')
assert initialized['exit_code'] == 0, initialized
state = r / '.pose/state/project-state.md'
before = hashlib.sha256(state.read_bytes()).hexdigest()
history = r / '.pose/state/history.jsonl'
if history.exists():
    history.unlink()
history.mkdir()
p = r / '.pose/specs/work.md'
p.write_text(p.read_text().replace('status: draft', 'status: in-progress'))
failed = run(r, 'state', 'refresh')
assert failed['exit_code'] == 1, failed
after = hashlib.sha256(state.read_bytes()).hexdigest()
assert before != after
observations['state_history_failure'] = {'init': initialized, 'refresh': failed,
    'state_hash_before': before, 'state_hash_after': after, 'state_changed_despite_error': True}

output = {'schema_version': 1, 'binary_sha256': hashlib.sha256(binary.read_bytes()).hexdigest(),
          'fixtures': str(parent), 'observations': observations,
          'limitations': ['Linux filesystem/process observations, not portability or power-loss proof',
             'No Go build, race detector, external agents or network',
             'Release/checkpoint damage constructs an intermediate state; no claim of actual crash injection',
             'Supersession retry assertion requires exactly the same generated ID']}
pathlib.Path(a.output).write_text(json.dumps(output, indent=2) + '\n')
print(json.dumps({'output': a.output, 'cases': len(observations)}))

"""Synthetic POSE 7.4.1 action/obligation probes. No vendor agents or network.

Run with --binary /absolute/path/to/pose --output /tmp/front04.json.
Every write affects new temporary Git fixtures. Fixtures remain for inspection.
This records observations, not tests requiring defects to remain in the engine.
"""
import argparse
import hashlib
import json
import os
import pathlib
import shutil
import subprocess
import tempfile

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument('--binary', required=True)
parser.add_argument('--output', required=True)
parser.add_argument('--scratch-root', default=tempfile.gettempdir())
opts = parser.parse_args()
binary = pathlib.Path(opts.binary).resolve()
parent = pathlib.Path(tempfile.mkdtemp(prefix='pose-actions-probes-', dir=opts.scratch_root))
env = {k: v for k, v in os.environ.items() if not k.startswith(('POSE_', 'HARNE8_'))}
git = shutil.which('git')
results = {}


def write(root, path, value):
    p = root / path
    p.parent.mkdir(parents=True, exist_ok=True)
    p.write_text(json.dumps(value) if isinstance(value, dict) else value)


def run(root, *args):
    return subprocess.run([str(binary), *args], cwd=root, env=env,
                          capture_output=True, text=True, timeout=30)


def data(root, *args):
    r = run(root, *args)
    if r.returncode:
        raise RuntimeError(r.stdout + r.stderr)
    return json.loads(r.stdout)


def fixture(name, verified=False):
    root = parent / name
    root.mkdir()
    write(root, '.pose/project.json', {'schema_version': 1, 'project_id': 'proj.action-probe'})
    write(root, '.pose/policy/review.json', {'schema_version': 2, 'enabled': False,
          'profiles': {'spec': 'spec-closeout@1'}, 'agency_readiness_version': 1,
          'atomic_start_version': 1})
    write(root, '.pose/policy/actions.json', {'schema_version': 1,
          'roles': {'maintainer': ['human:maintainer']},
          'identity_assurance': 'verified' if verified else 'declared'})
    for slug in ('alpha', 'beta'):
        write(root, f'.pose/specs/{slug}.md', f'---\nslug: {slug}\nstatus: draft\ncreated_at: 2026-08-02\n---\n\n# Spec: {slug}\n\n## 2. Requirements\n- R1: Preserve compatibility.\n')
    write(root, '.pose/roadmaps/portfolio.md', '---\nslug: portfolio\nstatus: draft\ncreated_at: 2026-08-02\n---\n\n# Roadmap: portfolio\n\n## Milestone: first\n- specs: alpha\n')
    subprocess.run([git, 'init', '-q', str(root)], check=True)
    subprocess.run([git, '-C', str(root), 'add', '-A'], check=True)
    subprocess.run([git, '-C', str(root), '-c', 'user.name=Probe', '-c',
                    'user.email=probe@example.invalid', '-c', 'commit.gpgsign=false',
                    'commit', '-qm', 'fixture'], check=True)
    return root


def opening(root, *extra, origin='spec:alpha', target='self', effect='start:block',
            kind='approval', question='Allow the scoped operation?'):
    return data(root, 'action', 'open', '--origin', origin, '--kind', kind,
                '--question', question, '--requested-by', 'agent:impl',
                '--recipient-role', 'maintainer', '--target', target,
                '--effect', effect, *extra, '--apply', '--json')


def resolution(root, view, verb='resolve', actor='human:maintainer', answer='approve',
               key='probe-key', apply=True):
    r = view['request']
    args = ['action', verb, r['id'], '--actor', actor, '--request-digest',
            r['request_digest'], '--expected-revision', str(view['revision']),
            '--idempotency-key', key, '--reason', 'synthetic fixture disposition']
    if verb == 'resolve':
        args += ['--answer', answer]
    if apply:
        args += ['--apply', '--json']
    return run(root, *args)


def start(root, slug):
    return data(root, 'start', 'spec:' + slug, '--json')


def attention(root, scope=''):
    args = ['state', '--attention', '--json']
    if scope:
        args += ['--scope', scope]
    return data(root, *args)


def present(root, actor=''):
    args = ['action', 'list', '--present', '--json']
    if actor:
        args += ['--actor', actor]
    return data(root, *args)


root = fixture('scope-control')
v = opening(root)
p = start(root, 'alpha')
results['same_origin_start_control'] = {'ready': p['ready'], 'codes': [x.get('code') for x in p['waiting_on']]}

root = fixture('cross-target')
v = opening(root, target='xref:proj.action-probe/spec:beta')
a, b = start(root, 'alpha'), start(root, 'beta')
results['cross_target_scope'] = {'origin_alpha_ready': a['ready'], 'target_beta_ready': b['ready'],
    'beta_attention_actions': attention(root, 'spec:beta')['counts'].get('actor-action', 0)}
apply = run(root, 'start', 'spec:beta', '--apply', '--digest', b['digest'], '--json')
results['cross_target_scope']['beta_start_apply_exit'] = apply.returncode
results['cross_target_scope']['beta_status_after_apply'] = 'in-progress' if 'status: in-progress' in (root / '.pose/specs/beta.md').read_text() else 'draft'

root = fixture('explicit-effect-scope')
v = opening(root, target='xref:proj.action-probe/spec:beta')
# Import the domain's typed effect.scope shape, unavailable in CLI open flags.
# This is a synthetic fixture construction, not tampering with a verified answer.
journal = root / '.pose/actions' / (v['request']['id'] + '.jsonl')
event = json.loads(journal.read_text())
request = event['request']
def material(r):
    keys = ['project_id', 'origin', 'recipient', 'kind', 'question', 'context',
            'options', 'targets', 'effects', 'subject']
    out = {k: r[k] for k in keys if k in r}
    out['effects'] = [{k: e[k] for k in ['phase', 'scope', 'mode'] if k in e} for e in r['effects']]
    return out
def material_digest(r):
    raw = json.dumps(material(r), separators=(',', ':'), ensure_ascii=False).encode()
    return 'sha256:' + hashlib.sha256(raw).hexdigest()
assert material_digest(request) == request['request_digest']
request['effects'][0]['scope'] = request['targets']
event['request_digest'] = request['request_digest'] = material_digest(request)
identity = ['action-request/v1', request['request_digest'], request['requested_at'],
            request['requested_by']['principal'], request.get('supersedes', '')]
request['id'] = 'act-' + hashlib.sha256(json.dumps(identity, separators=(',', ':')).encode()).hexdigest()[:16]
new_journal = journal.with_name(request['id'] + '.jsonl')
new_journal.write_text(json.dumps(event) + '\n')
journal.unlink()
loaded = data(root, 'action', 'show', request['id'], '--json')
a, b = start(root, 'alpha'), start(root, 'beta')
apply = run(root, 'start', 'spec:beta', '--apply', '--digest', b['digest'], '--json')
results['explicit_effect_scope'] = {'fixture_construction': 'imported typed opening with recomputed material digest and derived ID',
    'loaded_effect_scope': loaded['request']['effects'][0]['scope'],
    'origin_alpha_ready': a['ready'], 'target_beta_ready': b['ready'],
    'target_beta_start_apply_exit': apply.returncode}

root = fixture('roadmap-scope')
v = opening(root, origin='roadmap:portfolio')
a, b = start(root, 'alpha'), start(root, 'beta')
results['roadmap_scope'] = {'member_alpha_ready': a['ready'], 'nonmember_beta_ready': b['ready']}

root = fixture('terminal-origin')
v = opening(root)
p = root / '.pose/specs/alpha.md'
p.write_text(p.read_text().replace('status: draft', 'status: done'))
results['terminal_origin_visibility'] = {'project_actions': attention(root)['counts'].get('actor-action', 0),
    'explicit_scope_actions': attention(root, 'spec:alpha')['counts'].get('actor-action', 0),
    'present_requests': sum(len(x['requests']) for x in present(root)['groups'])}

root = fixture('verified-waiver', verified=True)
v = opening(root)
control = resolution(root, v)
r = resolution(root, v, verb='waive', key='waive-key')
after = data(root, 'action', 'show', v['request']['id'], '--json')
results['verified_lifecycle_waiver'] = {'unsigned_answer_exit': control.returncode,
    'unsigned_waiver_exit': r.returncode, 'state': after['state'],
    'satisfaction': after['satisfaction'], 'start_ready': start(root, 'alpha')['ready'],
    'last_event_has_proof': bool(after['events'][-1].get('claim') or after['events'][-1].get('ssh_signature'))}

root = fixture('supersession-authority', verified=True)
v = opening(root)
new = opening(root, '--supersedes', v['request']['id'], '--requested-by', 'agent:other', origin='spec:beta',
              effect='release:advisory', question='Different fixture question')
after = data(root, 'action', 'show', v['request']['id'], '--json')
results['supersession_changes_obligation'] = {'prior_state': after['state'],
    'new_origin': new['request']['origin'], 'prior_alpha_start_ready': start(root, 'alpha')['ready'],
    'superseding_actor': after['events'][-1]['actor']}

root = fixture('subject-reconfirmation')
v = opening(root, '--subject', 'requirement:R1')
assert resolution(root, v).returncode == 0
p = root / '.pose/specs/alpha.md'
p.write_text(p.read_text().replace('Preserve compatibility.', 'Change compatibility.'))
after = data(root, 'action', 'show', v['request']['id'], '--json')
again = resolution(root, after, key='fresh-key')
results['invalidated_presented_but_not_answerable'] = {'state': after['state'],
    'present_requests': sum(len(x['requests']) for x in present(root)['groups']),
    'fresh_resolution_exit': again.returncode,
    'not_open_diagnostic': 'action-request-not-open' in (again.stdout + again.stderr)}

root = fixture('missing-subject')
write(root, 'subject.txt', 'Fixture subject\n')
v = opening(root, '--subject', 'path:subject.txt')
assert resolution(root, v).returncode == 0
(root / 'subject.txt').unlink()
after = data(root, 'action', 'show', v['request']['id'], '--json')
results['missing_subject_routing'] = {'state': after['state'], 'satisfaction': after['satisfaction'],
    'subject_unknown': after.get('subject_unknown', False),
    'present_requests': sum(len(x['requests']) for x in present(root)['groups']),
    'blocking_start': len(attention(root)['attention']['blocking'].get('start', []))}

root = fixture('actor-routing')
v = opening(root)
results['role_holder_routing'] = {'principal_groups': len(present(root, 'human:maintainer')['groups']),
    'role_groups': len(present(root, 'maintainer')['groups'])}

root = fixture('preview-and-refusal')
v = opening(root)
pre = resolution(root, v, actor='agent:other', apply=False)
app = resolution(root, v, actor='agent:other')
after = data(root, 'action', 'show', v['request']['id'], '--json')
results['resolution_preview_and_refusal'] = {'preview_exit': pre.returncode, 'apply_exit': app.returncode,
    'revision_after_refusal': after['revision'], 'refused_events': sum(x['type'] == 'refused' for x in after['events'])}

root = fixture('decline-and-wait-metrics')
v = opening(root, effect='execution:block')
assert resolution(root, v, answer='decline').returncode == 0
# A synthetic historical interval, not a claim about observed real waiting.
# Times are outside RequestDigest; preserve all engine-produced material fields.
journal = root / '.pose/actions' / (v['request']['id'] + '.jsonl')
events = [json.loads(x) for x in journal.read_text().splitlines()]
events[0]['at'] = events[0]['request']['requested_at'] = '2026-10-01T00:00:00Z'
events[1]['at'] = '2026-10-02T00:00:00Z'
journal.write_text(''.join(json.dumps(x) + '\n' for x in events))
stats = data(root, 'stats', 'governance', '--waits', '--json')
results['decline_and_blocking_metrics'] = {'blocking_execution': len(attention(root)['attention']['blocking'].get('execution', [])),
    'present_requests': sum(len(x['requests']) for x in present(root)['groups']), 'stats': stats}

root = fixture('snapshot-changes')
v = opening(root)
before = attention(root)
assert resolution(root, v).returncode == 0
after = attention(root)
results['action_change_snapshot_identity'] = {'same_digest': before['snapshot']['digest'] == after['snapshot']['digest'],
    'same_source_revision': before['snapshot']['source_revision'] == after['snapshot']['source_revision'],
    'same_dirty': before['snapshot']['worktree_dirty'] == after['snapshot']['worktree_dirty'],
    'blocking_start_before': len(before['attention']['blocking'].get('start', [])),
    'blocking_start_after': len(after['attention']['blocking'].get('start', []))}

root = fixture('node-reference-validation')
v = opening(root, target='requirement:R404')
bad_subject = opening(root, '--subject', 'decision:R1', question='Different fixture subject')
results['node_reference_validation'] = {'missing_target_opened': v['state'] == 'open',
    'wrong_kind_subject_opened': bad_subject['state'] == 'open',
    'subject_node_kind': bad_subject['request']['subject']['node']['kind'],
    'subject_has_digest': bool(bad_subject['request']['subject']['digest'])}

root = fixture('declared-controls')
v = opening(root)
first = resolution(root, v)
retry = resolution(root, v)
after = data(root, 'action', 'show', v['request']['id'], '--json')
results['idempotency_control'] = {'first_exit': first.returncode, 'retry_exit': retry.returncode,
    'revision': after['revision'], 'satisfaction': after['satisfaction']}

pathlib.Path(opts.output).write_text(json.dumps(results, ensure_ascii=False, indent=2) + '\n')
print(json.dumps(results, ensure_ascii=False, indent=2))
print('Fixtures: ' + str(parent))

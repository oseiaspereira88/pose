"""Observe release sufficiency and evidence contracts in disposable Git fixtures.
Run with --binary /absolute/path/pose --output /tmp/front06.json. No network,
real publication, signing or tag push. Results are observations, not a contract
requiring defects to survive future corrections. Fixtures remain for inspection.
"""
import argparse
import hashlib
import json
import os
import pathlib
import subprocess
import tempfile

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument('--binary', required=True)
parser.add_argument('--output', required=True)
opts = parser.parse_args()
binary = pathlib.Path(opts.binary).resolve()
parent = pathlib.Path(tempfile.mkdtemp(prefix='pose-release-probes-'))
env = {k: v for k, v in os.environ.items() if not k.startswith(('POSE_', 'HARNE8_'))}
results = {}
version = 'v1.0.0'


def write(root, path, value):
    target = root / path
    target.parent.mkdir(parents=True, exist_ok=True)
    target.write_text(json.dumps(value, indent=2) + '\n' if isinstance(value, dict) else value)


def git(root, *args):
    r = subprocess.run(['git', '-C', str(root), '-c', 'user.name=Probe', '-c',
                        'user.email=probe@example.invalid', '-c', 'commit.gpgsign=false',
                        *args], capture_output=True, text=True, check=True)
    return r.stdout.strip()


def run(root, *args):
    r = subprocess.run([str(binary), *args], cwd=root, env=env,
                       capture_output=True, text=True, timeout=60)
    out = {'exit_code': r.returncode, 'stdout': r.stdout, 'stderr': r.stderr}
    try:
        out['json'] = json.loads(r.stdout)
    except ValueError:
        pass
    return out


def spec(root, slug='alpha', status='done'):
    write(root, f'.pose/specs/{slug}.md', f'---\nslug: {slug}\nstatus: {status}\ncreated_at: 2026-10-01\ncompleted_at: 2026-10-02\n---\n\n# Spec: {slug}\n')


def fragment(root, slug='alpha'):
    write(root, f'.pose/changelogs/unreleased/{slug}.md', f'---\nspec: {slug}\ncategory: fixed\nbreaking: false\n---\n\nRepairs {slug}.\n')


def fixture(name, status='done'):
    root = parent / name
    root.mkdir()
    write(root, '.pose/project.json', {'schema_version': 1, 'project_id': 'proj.release-probe'})
    write(root, '.pose/policy/review.json', {'schema_version': 2, 'enabled': False})
    write(root, '.pose/policy/release.json', {
        'schema_version': 1, 'adopted_at': '2026-08-03', 'provider': 'github',
        'repository': 'owner/release-probe', 'require_verification': True,
        'version_source': {'path': 'VERSION', 'kind': 'text'}})
    write(root, 'VERSION', '1.0.0\n')
    spec(root, status=status)
    fragment(root)
    git(root, 'init', '-q')
    git(root, 'add', '-A')
    git(root, 'commit', '-qm', 'fixture')
    return root


def prepare(root, target=version):
    return run(root, 'release', 'prepare', '--version', target, '--apply')


def check(root, target=version):
    return run(root, 'release', 'check', '--version', target, '--strict', '--json')


def commit_and_tag(root):
    git(root, 'add', '-A')
    git(root, 'commit', '-qm', 'prepared fixture')
    git(root, 'tag', '-a', version, '-m', 'synthetic tag')
    return git(root, 'rev-parse', 'HEAD')


def evidence(root, commit, name, **override):
    value = {'schema_version': 1, 'provider': 'github', 'repository': 'owner/release-probe',
             'version': version, 'tag': version, 'commit': commit,
             'published_at': '2026-10-12T00:00:00Z',
             'url': f'https://example.invalid/releases/{version}',
             'assets': {'pose.tar.gz': 'sha256:' + 'a' * 64}}
    value.update(override)
    write(root, name, value)
    return value


def record(root, state, path):
    return run(root, 'release', 'record', '--version', version,
               '--event', state, '--evidence', path)


def status(root):
    return run(root, 'release', 'status', '--version', version, '--json')


root = fixture('draft-and-failed-roadmap', status='draft')
write(root, '.pose/roadmaps/program.md', '---\nslug: program\nstatus: active\ncreated_at: 2026-10-01\n---\n\n## Milestone: core\n- specs: alpha\n\n## Cut criteria\n- C1: check:never-produced\n')
results['P01'] = {'roadmap': run(root, 'roadmap-check', 'program', '--json'),
                  'plan': run(root, 'release', 'plan', '--version', version, '--json'),
                  'prepare': prepare(root), 'check': check(root)}

root = fixture('heading-discovery')
write(root, '.pose/roadmaps/program.md', '---\nslug: program\nstatus: active\n---\n\n## Milestone: core\n- specs: alpha\n\n## Release cut criteria\n- C1: check:never-produced\n')
results['P02'] = run(root, 'roadmap-check', 'program', '--json')

root = fixture('manual-review-is-file')
write(root, 'docs/review.md', '')
write(root, '.pose/roadmaps/program.md', '---\nslug: program\nstatus: active\n---\n\n## Milestone: core\n- specs: alpha\n\n## Cut criteria\n- C1: manual-review:docs/review.md\n')
results['P03'] = run(root, 'roadmap-check', 'program', '--json')

root = fixture('manifest-projections')
prepare(root)
p = root / '.pose/releases/v1.0.0/manifest.json'
manifest = json.loads(p.read_text())
manifest.update({'schema_version': 999, 'specs': ['phantom'], 'breaking': True,
                 'categories': {'security': 999}})
write(root, '.pose/releases/v1.0.0/manifest.json', manifest)
results['P04'] = {'check': check(root), 'status': status(root)}

root = fixture('version-source-drift')
prepare(root)
write(root, 'VERSION', '9.0.0\n')
spec(root, 'beta')
fragment(root, 'beta')
results['P05'] = {'check': check(root), 'plan': run(root, 'release', 'plan', '--version', version, '--json')}

root = fixture('tag-tree-does-not-have-manifest')
base = git(root, 'rev-parse', 'HEAD')
prepare(root)
git(root, 'add', '-A')
git(root, 'commit', '-qm', 'prepared fixture')
git(root, 'tag', '-a', version, base, '-m', 'tag older fixture')
results['P06'] = {'check': check(root), 'tag_has_manifest': subprocess.run(
    ['git', '-C', str(root), 'cat-file', '-e', f'{version}:.pose/releases/{version}/manifest.json'],
    capture_output=True).returncode == 0}

root = fixture('mismatched-assets')
prepare(root)
commit = commit_and_tag(root)
evidence(root, commit, 'tag.json')
evidence(root, commit, 'publish.json')
record(root, 'tagged', 'tag.json')
record(root, 'published', 'publish.json')
events = [json.loads(x) for x in (root / '.pose/releases/v1.0.0/events.jsonl').read_text().splitlines()]
pub_digest = events[-1]['evidence_digest']
evidence(root, commit, 'verify.json', publication_digest=pub_digest,
         assets={'different.zip': 'sha256:' + 'b' * 64})
results['P07'] = {'record': record(root, 'verified', 'verify.json'),
                  'status': status(root), 'check': check(root)}

root = fixture('mismatched-provider-and-repository')
prepare(root)
commit = commit_and_tag(root)
evidence(root, commit, 'tag.json', provider='other-provider', repository='other/repository')
evidence(root, commit, 'publish.json', provider='other-provider', repository='other/repository')
results['P08'] = {'tag': record(root, 'tagged', 'tag.json'),
                  'published': record(root, 'published', 'publish.json'), 'check': check(root)}

root = fixture('read-time-integrity')
prepare(root)
commit = commit_and_tag(root)
evidence(root, commit, 'tag.json')
evidence(root, commit, 'publish.json')
record(root, 'tagged', 'tag.json')
record(root, 'published', 'publish.json')
p = root / '.pose/releases/v1.0.0/events.jsonl'
events = [json.loads(x) for x in p.read_text().splitlines()]
evidence(root, commit, 'verify.json', publication_digest=events[-1]['evidence_digest'])
record(root, 'verified', 'verify.json')
events = [json.loads(x) for x in p.read_text().splitlines()]
events[1]['evidence']['assets']['pose.tar.gz'] = 'sha256:' + 'c' * 64
p.write_text(''.join(json.dumps(x) + '\n' for x in events))
results['P09'] = {'status': status(root), 'check': check(root),
                  'open_next': run(root, 'release', 'open-next', '--version', 'v1.1.0')}

root = fixture('duplicate-across-release-versions')
prepare(root)
write(root, 'VERSION', '1.1.0\n')
fragment(root)
results['P10'] = {'plan': run(root, 'release', 'plan', '--version', 'v1.1.0', '--json'),
                  'prepare': prepare(root, 'v1.1.0'), 'check': check(root, 'v1.1.0')}

root = fixture('all-pending-is-selection')
spec(root, 'beta', status='draft')
fragment(root, 'beta')
results['P11'] = {'plan': run(root, 'release', 'plan', '--version', version, '--json'),
                  'prepare': prepare(root)}

root = fixture('negative-controls')
prepare(root)
p = root / '.pose/changelogs/v1.0.0/alpha.md'
p.write_text(p.read_text() + '\nChanged after freeze.\n')
results['P12'] = {'archive_tamper': check(root)}
root = fixture('wrong-publication-binding')
prepare(root)
commit = commit_and_tag(root)
evidence(root, commit, 'tag.json')
evidence(root, commit, 'publish.json')
record(root, 'tagged', 'tag.json')
record(root, 'published', 'publish.json')
evidence(root, commit, 'verify.json', publication_digest='sha256:' + 'd' * 64)
results['P12']['wrong_binding'] = record(root, 'verified', 'verify.json')


root = fixture('declared-surface-without-evidence')
write(root, '.pose/policy/artifacts.json', {'schema_version': 1, 'enabled': True, 'adopted_at': '2026-08-03', 'governed_roots': ['web']})
write(root, '.pose/policy/delivery.json', {'schema_version': 1, 'enabled': True, 'adopted_at': '2026-08-03', 'results_path': '.pose/results/current.json', 'roots': [], 'severities': {'surface-without-reachability': 'error', 'surface-without-provenance': 'error', 'roadmap-criterion': 'error'}})
write(root, '.pose/indexes/validation-matrix.json', {'defaults': {'mode': 'strict'}, 'deliveryProfiles': {'web-ui': {'kind': 'surface', 'requiredEvidenceClasses': ['reachability'], 'anyEvidenceClasses': ['integration', 'e2e']}}, 'stacks': {}, 'moduleOverrides': {}})
write(root, '.pose/specs/alpha.md', '---\nslug: alpha\nstatus: done\ncreated_at: 2026-10-01\ncompleted_at: 2026-10-02\ndelivers: surface:dashboard\n---\n\n# Spec: alpha\n\n## 2. Requirements\n- R1: dashboard\n\n## 3. Technical Plan\n\n### Artifacts\n- modified: web/view.go\n\n### Delivery targets\n- surface:dashboard module:web profile:web-ui entrypoint:cmd/app/main.go\n')
write(root, 'web/view.go', 'package web\n')
write(root, 'cmd/app/main.go', 'package main\n')
write(root, '.pose/roadmaps/program.md', '---\nslug: program\nstatus: active\ncreated_at: 2026-10-01\n---\n\n## Milestone: core\n- specs: alpha\n\n## Cut criteria\n- C1: surface:dashboard evidence:e2e\n')
git(root, 'add', '-A')
git(root, 'commit', '-qm', 'surface fixture declarations')
results['P13'] = {'roadmap': run(root, 'roadmap-check', 'program', '--json'), 'surface': run(root, 'surface-check', '--spec', 'alpha', '--strict', '--json')}

root = fixture('evidence-only-criterion')
write(root, '.pose/roadmaps/program.md', '---\nslug: program\nstatus: active\n---\n\n## Milestone: core\n- specs: alpha\n\n## Cut criteria\n- C1: evidence:e2e\n')
results['P14'] = run(root, 'roadmap-check', 'program', '--json')

pathlib.Path(opts.output).write_text(json.dumps({
    'binary_sha256': hashlib.sha256(binary.read_bytes()).hexdigest(),
    'version': subprocess.run([str(binary), 'version'], capture_output=True, text=True).stdout,
    'fixtures': str(parent), 'observations': results}, indent=2) + '\n')
print(json.dumps({'output': opts.output, 'probes': len(results), 'fixtures': str(parent)}))

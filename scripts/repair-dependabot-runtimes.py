#!/usr/bin/env python3
"""Repair one derived manifest, never execute PR code or force a branch."""
import base64
import json
import os
from pathlib import Path
import re
import subprocess
import tempfile

MANIFEST = '.github/action-runtimes.json'
SHA = re.compile(r'[0-9a-f]{40}')
WORKFLOW = re.compile(r'\.github/workflows/[\w-]+\.yml')
ACTION = re.compile(r'(?m)^\s*(?:-\s+)?uses:\s*([^\s#]+)')


def api(path, method='GET', payload=None):
    args = ['gh', 'api', path, '--method', method]
    if payload is not None:
        args += ['--input', '-']
    result = subprocess.run(args, input=json.dumps(payload) if payload is not None else None,
                            capture_output=True, text=True, check=True)
    return json.loads(result.stdout) if result.stdout.strip() else None


def content(repo, path, revision):
    if not SHA.fullmatch(revision):
        raise ValueError('immutable revision required')
    result = api(f'repos/{repo}/contents/{path}?ref={revision}')
    if result.get('encoding') != 'base64' or result.get('size', 0) > 1024 * 1024:
        raise ValueError('unsupported or oversized content')
    return base64.b64decode(result['content']).decode('utf-8')


def pin_only(before, after):
    """Ignore only action SHA/version-comment edits, not arbitrary 40-hex data."""
    def normalize(text):
        for match in ACTION.finditer(text):
            ref = match[1]
            if ref.startswith('./'):
                continue
            action, sep, sha = ref.partition('@')
            if not sep or not SHA.fullmatch(sha) or not re.fullmatch(r'[\w.-]+/[\w./-]+', action):
                raise ValueError('external action must use a full commit SHA')
        return re.sub(r'(?m)^(\s*(?:-\s+)?uses:\s*[^\s#]+@)[0-9a-f]{40}(\s*)(?:#\s*v[\w.+-]+\s*)?$',
                      r'\1PIN', text)
    return normalize(before) == normalize(after)


def eligible(pr, repo, tested):
    return (pr['state'] == 'open' and pr['user']['login'] == 'dependabot[bot]'
            and pr['head']['repo'] and pr['head']['repo']['full_name'] == repo
            and pr['base']['repo']['full_name'] == repo and pr['base']['ref'] == 'main'
            and pr['head']['ref'].startswith('dependabot/github_actions/')
            and pr['head']['sha'] == tested and SHA.fullmatch(tested))


def generate(root, repo, base, tested, files):
    workflows = api(f'repos/{repo}/contents/.github/workflows?ref={tested}')
    paths = {entry['path'] for entry in workflows if WORKFLOW.fullmatch(entry['path'])}
    changed = False
    for entry in files:
        path = entry['filename']
        if path == MANIFEST and entry['status'] == 'modified':
            continue  # A previous repair may be in this PR; regenerate from trusted policy.
        if entry['status'] != 'modified' or not WORKFLOW.fullmatch(path):
            raise ValueError('PR must contain only modifications to existing workflows')
        before, after = content(repo, path, base), content(repo, path, tested)
        if not pin_only(before, after):
            raise ValueError('PR changes workflow logic, action identity or non-pin content')
        changed |= before != after
    if not changed:
        return None
    refs = {}
    with tempfile.TemporaryDirectory() as temporary:
        work = Path(temporary)
        folder = work / '.github/workflows'
        folder.mkdir(parents=True)
        for path in sorted(paths):
            body = content(repo, path, tested)
            for match in ACTION.finditer(body):
                action, sep, pin = match[1].partition('@')
                if action.startswith('./'):
                    continue
                if not sep or not SHA.fullmatch(pin):
                    raise ValueError('all external actions must have immutable pins')
                if action in refs and refs[action] != pin:
                    raise ValueError('conflicting pins for an action')
                refs[action] = pin
            (work / path).write_text(body)
        # Policy comes from trusted base, not from PR-controlled JSON.
        (work / MANIFEST).write_text(content(repo, MANIFEST, base))
        subprocess.run(['git', 'init', '-q', temporary], check=True)
        subprocess.run(['bash', str(root / 'scripts/refresh-action-runtimes.sh')], cwd=work, check=True)
        return (work / MANIFEST).read_text()


def repair(event, root):
    repo = event['repository']['full_name']
    if not re.fullmatch(r'[\w.-]+/[\w.-]+', repo):
        raise ValueError('invalid repository')
    run = event['workflow_run']
    if (run['name'] != 'CI' or run['event'] != 'pull_request'
            or run['head_repository']['full_name'] != repo):
        return 'ineligible run'
    tested = run['head_sha']
    if not SHA.fullmatch(tested):
        raise ValueError('invalid tested revision')
    prs = api(f'repos/{repo}/commits/{tested}/pulls')
    candidates = [pr for pr in prs if eligible(pr, repo, tested)]
    if len(candidates) != 1:
        return 'no unique eligible PR'
    pr = api(f'repos/{repo}/pulls/{candidates[0]["number"]}')
    if not eligible(pr, repo, tested):
        return 'PR no longer eligible'
    # Refuse truncated file lists rather than silently ignoring a mixed change.
    if pr['changed_files'] > 100:
        raise ValueError('too many changed files')
    files = api(f'repos/{repo}/pulls/{pr["number"]}/files?per_page=100')
    if len(files) != pr['changed_files']:
        raise ValueError('incomplete file inventory')
    generated = generate(root, repo, pr['base']['sha'], tested, files)
    if generated is None or json.loads(generated) == json.loads(content(repo, MANIFEST, tested)):
        return 'runtime manifest already current'
    latest = api(f'repos/{repo}/pulls/{pr["number"]}')
    if not eligible(latest, repo, tested):
        raise ValueError('PR changed during repair')
    parent = api(f'repos/{repo}/git/commits/{tested}')
    tree = api(f'repos/{repo}/git/trees', 'POST', {
        'base_tree': parent['tree']['sha'],
        'tree': [{'path': MANIFEST, 'mode': '100644', 'type': 'blob', 'content': generated}],
    })
    commit = api(f'repos/{repo}/git/commits', 'POST', {
        'message': 'Refresh runtime evidence for Dependabot action pins\n\nPOSE-Spec: pose-action-runtime-currency-gate',
        'tree': tree['sha'], 'parents': [tested],
    })
    # If the head moved concurrently, this child of the old head cannot fast-forward it.
    branch = pr['head']['ref']
    api(f'repos/{repo}/git/refs/heads/{branch}', 'PATCH', {'sha': commit['sha'], 'force': False})
    api(f'repos/{repo}/actions/workflows/ci.yml/dispatches', 'POST', {'ref': branch})
    return 'runtime manifest repaired; CI dispatched'


if __name__ == '__main__':
    print(repair(json.loads(Path(os.environ['GITHUB_EVENT_PATH']).read_text()), Path(__file__).resolve().parents[1]))

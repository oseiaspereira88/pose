"""Bounded POSE CLI/MCP performance and freshness observations.
Run --binary /path/pose --root /isolated/baseline --output /tmp/front08.json.
Wall timings include subprocess or stdio round-trip overhead, not human work.
No network/vendor agents. CLI/Git read traces run separately from timing.
"""
import argparse
import hashlib
import json
import os
import pathlib
import selectors
import shutil
import statistics
import subprocess
import tempfile
import time

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument('--binary', required=True)
parser.add_argument('--root', required=True)
parser.add_argument('--output', required=True)
parser.add_argument('--repeats', type=int, default=3)
a = parser.parse_args()
if a.repeats < 1:
    parser.error('--repeats must be positive')
binary = pathlib.Path(a.binary).resolve()
root = pathlib.Path(a.root).resolve()
parent = pathlib.Path(tempfile.mkdtemp(prefix='pose-perf-probes-'))
env = {k: v for k, v in os.environ.items() if not k.startswith(('POSE_', 'HARNE8_'))}
env['POSE_USAGE_SUPPRESS_CLI'] = '1'
results = {}


def write(r, path, value):
    p = r / path
    p.parent.mkdir(parents=True, exist_ok=True)
    p.write_text(json.dumps(value) + '\n' if isinstance(value, dict) else value)


def git(r, *args):
    return subprocess.run(['git', '-C', str(r), '-c', 'user.name=Probe', '-c',
        'user.email=probe@example.invalid', '-c', 'commit.gpgsign=false', *args],
        capture_output=True, text=True, check=True).stdout.strip()


def fixture(name, n, followups=False):
    r = parent / name
    r.mkdir()
    write(r, '.pose/project.json', {'schema_version': 1, 'project_id': 'proj.perf-probe'})
    write(r, '.pose/policy/review.json', {'schema_version': 2, 'enabled': False})
    for i in range(n):
        slug = f'item-{i:04}'
        followup = f'\n## 7. Final Report\n### Follow-ups\n- [open] Repair behavior of item {i:04} after observation. (owner:@core crit:low review:2099-01-01)\n' if followups else ''
        write(r, f'.pose/specs/{slug}.md', f'---\nslug: {slug}\nstatus: draft\ncreated_at: 2026-10-01\n---\n\n# Spec: {slug}\n\n## 2. Requirements\n- R1: Retain behavior {i:04}.\n' + followup)
    git(r, 'init', '-q')
    git(r, 'add', '-A')
    git(r, 'commit', '-qm', 'fixture')
    return r


def cli(r, *args):
    start = time.perf_counter()
    p = subprocess.run([str(binary), *args], cwd=r, env=env,
                       capture_output=True, timeout=90)
    row = {'seconds': time.perf_counter() - start, 'exit_code': p.returncode,
           'stdout_bytes': len(p.stdout), 'stderr_bytes': len(p.stderr)}
    try:
        row['data'] = json.loads(p.stdout)
    except ValueError:
        row['text'] = p.stdout.decode(errors='replace')[-300:]
    return row


def summarize(rows):
    return {'samples': len(rows), 'seconds': [x['seconds'] for x in rows],
            'median_seconds': statistics.median(x['seconds'] for x in rows),
            'min_seconds': min(x['seconds'] for x in rows),
            'max_seconds': max(x['seconds'] for x in rows),
            'exit_codes': [x['exit_code'] for x in rows],
            'stdout_bytes': [x['stdout_bytes'] for x in rows]}


class MCP:
    def __init__(self, r, extra=None):
        e = dict(env)
        e['POSE_PROJECT_ROOT'] = str(r)
        e['PATH'] = str(binary.parent) + os.pathsep + e.get('PATH', '')
        e.update(extra or {})
        self.log = tempfile.TemporaryFile()
        self.p = subprocess.Popen([str(binary), 'serve-mcp', '--stdio'], cwd=r, env=e,
            stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=self.log)
        self.sequence = 0
        self.rpc('initialize', {})

    def rpc(self, method, params):
        self.sequence += 1
        request = {'jsonrpc': '2.0', 'id': self.sequence, 'method': method, 'params': params}
        start = time.perf_counter()
        self.p.stdin.write(json.dumps(request).encode() + b'\n')
        self.p.stdin.flush()
        with selectors.DefaultSelector() as select:
            select.register(self.p.stdout, selectors.EVENT_READ)
            if not select.select(100):
                raise TimeoutError(method)
        raw = self.p.stdout.readline()
        response = json.loads(raw)
        if response.get('id') != self.sequence:
            raise ValueError('unexpected response id')
        result = response.get('result', {})
        return {'seconds': time.perf_counter() - start, 'exit_code': 1 if 'error' in response or result.get('isError') else 0,
                'stdout_bytes': len(raw), 'data': result.get('structuredContent', result), 'rpc_error': response.get('error')}

    def call(self, tool, args):
        return self.rpc('tools/call', {'name': tool, 'arguments': args})

    def close(self):
        self.p.stdin.close()
        self.p.wait(timeout=15)
        self.log.close()


# Synthetic read/scaling costs, including response size, at fixed content.
scale = []
for n in (10, 100, 500):
    r = fixture(f'specs-{n}', n)
    server = MCP(r)
    rows = {'list_cli': [], 'recent_cli': [], 'list_mcp': [], 'page_mcp': [], 'get_mcp': []}
    for _ in range(a.repeats):
        rows['list_cli'].append(cli(r, 'specs', '--json'))
        rows['recent_cli'].append(cli(r, 'specs', '--recent', '5', '--json'))
        rows['list_mcp'].append(server.call('pose_list_specs', {}))
        rows['page_mcp'].append(server.call('pose_list_specs', {'limit': 5}))
        rows['get_mcp'].append(server.call('pose_get_spec', {'slug': 'item-0000'}))
    server.close()
    scale.append({'specs': n, 'measurements': {k: summarize(v) for k, v in rows.items()}})
results['synthetic_scale'] = scale

# Similar follow-ups stress lexical clustering separately from filesystem scale.
followups = []
for n in (10, 50, 100):
    r = fixture(f'followups-{n}', n, followups=True)
    rows = [cli(r, 'followups', '--open', '--json') for _ in range(a.repeats)]
    followups.append({'items': n, 'measurement': summarize(rows),
                      'near_duplicate_clusters': len(rows[-1]['data']['near_duplicate_candidates'])})
results['followup_scale'] = followups

# Read-only baseline round trips; the first MCP request and repeats share one server.
server = MCP(root)
real = []
for label, command, tool, args in [
    ('spec-list', ['specs', '--json'], 'pose_list_specs', {}),
    ('recent-specs', ['specs', '--recent', '5', '--json'], 'pose_list_specs', {'recent': 5}),
    ('attention', ['state', '--attention', '--json'], 'pose_obligations', {}),
    ('followups', ['followups', '--open', '--json'], 'pose_get_followups', {'all': False})]:
    cli_rows = [cli(root, *command) for _ in range(a.repeats)]
    mcp_rows = [server.call(tool, args) for _ in range(a.repeats)]
    row = {'operation': label, 'cli': summarize(cli_rows), 'resident_mcp': summarize(mcp_rows)}
    if label == 'attention':
        row['producer_seconds'] = cli_rows[-1]['data'].get('producer_seconds', {})
        row['payload_keys'] = list(cli_rows[-1]['data'])
    real.append(row)
server.close()
results['baseline_round_trips'] = real

# Freshness after same-size/mtime external updates and positional page mutation.
r = fixture('resident-freshness', 6)
server = MCP(r)
page = server.call('pose_list_specs', {'limit': 2})['data']
p = r / '.pose/specs/item-0000.md'
old = p.read_text()
stamp = p.stat()
server.call('pose_get_spec', {'slug': 'item-0000'})
p.write_text(old.replace('Retain behavior', 'Modify behavior'))
os.utime(p, ns=(stamp.st_atime_ns, stamp.st_mtime_ns))
changed = server.call('pose_get_spec', {'slug': 'item-0000'})['data']
write(r, '.pose/specs/aaa.md', '---\nslug: aaa\nstatus: draft\ncreated_at: 2026-10-01\n---\n\n# Spec: inserted\n')
next_page = server.call('pose_list_specs', {'limit': 2, 'cursor': page['next_cursor']})['data']
results['resident_freshness'] = {'body_updated': 'Modify behavior' in changed.get('body', ''),
    'same_file_size': p.stat().st_size == stamp.st_size, 'mtime_preserved': p.stat().st_mtime_ns == stamp.st_mtime_ns,
    'first_page': [x['slug'] for x in page['specs']], 'next_page_after_insert': [x['slug'] for x in next_page['specs']],
    'overlap': sorted(set(x['slug'] for x in page['specs']) & set(x['slug'] for x in next_page['specs']))}
server.close()

pathlib.Path(a.output).write_text(json.dumps({'partial': True, 'observations': results}, indent=2) + '\n')

# Separate read/process trace from normal timings, preserving trace overhead disclosure.
if shutil.which('strace'):
    r = parent / 'specs-100'
    traces = []
    for label, args in [('all', ['specs', '--json']), ('recent5', ['specs', '--recent', '5', '--json'])]:
        trace = parent / (label + '.strace')
        traced = subprocess.run(['strace', '-f', '-qq', '-e', 'trace=openat,execve', '-o', str(trace), str(binary), *args],
            cwd=r, env=env, stdout=subprocess.DEVNULL, stderr=subprocess.PIPE, timeout=90)
        if traced.returncode:
            traces.append({'operation': label, 'unavailable': True, 'exit_code': traced.returncode,
                           'reason': traced.stderr.decode(errors='replace').strip()})
            continue
        text = trace.read_text()
        opened = [line for line in text.splitlines() if 'openat(' in line and '/specs/' in line]
        traces.append({'operation': label, 'spec_openat_calls': len(opened),
                       'execve_calls': sum('execve(' in line for line in text.splitlines()),
                       'trace_sha256': hashlib.sha256(trace.read_bytes()).hexdigest()})
    results['read_trace'] = traces

# Timeout classification with a controlled subprocess, not a real gate failure.
r = fixture('controlled-timeout', 1)
write(r, 'slow.py', '#!/usr/bin/env python3\nimport time\ntime.sleep(40)\n')
(r / 'slow.py').chmod(0o755)
server = MCP(r, {'POSE_EXECUTABLE': str(r / 'slow.py')})
results['controlled_timeout'] = server.call('pose_check', {'strict': True})
server.close()

output = {'schema_version': 1, 'binary_sha256': hashlib.sha256(binary.read_bytes()).hexdigest(),
          'root_revision': git(root, 'rev-parse', 'HEAD'), 'repeats': a.repeats,
          'fixtures': str(parent), 'environment': {'host_cpus': os.cpu_count(),
            'affinity_cpus': len(os.sched_getaffinity(0)), 'cpu_max': pathlib.Path('/sys/fs/cgroup/cpu.max').read_text().strip()},
          'limitations': ['Timings include OS cache and startup/IPC; no drop-caches performed',
            'No p95 claim from three samples', 'Strace run is not used for latency comparisons',
            'Controlled timeout uses a synthetic native executable'], 'observations': results}
pathlib.Path(a.output).write_text(json.dumps(output, indent=2) + '\n')
print(json.dumps({'output': a.output, 'cases': len(results), 'fixtures': str(parent)}))

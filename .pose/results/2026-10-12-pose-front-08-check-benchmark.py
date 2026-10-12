"""Measure one Linux POSE check in an isolated baseline checkout.

Run separately from other benchmarks. The child maximum RSS is not aggregate
concurrent memory. Output includes the gate's full stdout/stderr for comparison.
"""
import argparse
import hashlib
import json
import os
import pathlib
import platform
import resource
import subprocess
import time

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument('--root', required=True)
parser.add_argument('--binary', required=True)
parser.add_argument('--output', required=True)
parser.add_argument('--workers', default='default')
a = parser.parse_args()
if a.workers != 'default' and (not a.workers.isdigit() or int(a.workers) < 1):
    parser.error('--workers must be default or a positive integer')
env = {k: v for k, v in os.environ.items() if not k.startswith(('POSE_', 'HARNE8_'))}
env['POSE_USAGE_SUPPRESS_CLI'] = '1'
if a.workers != 'default':
    env['POSE_CHECK_WORKERS'] = a.workers
before = resource.getrusage(resource.RUSAGE_CHILDREN)
start = time.perf_counter()
r = subprocess.run([str(pathlib.Path(a.binary).resolve()), 'check', '--strict'],
                   cwd=a.root, env=env, capture_output=True, text=True, timeout=1800)
elapsed = time.perf_counter() - start
after = resource.getrusage(resource.RUSAGE_CHILDREN)
out = {
    'command': ['pose', 'check', '--strict'], 'workers': a.workers,
    'wall_seconds': elapsed,
    'children_user_seconds': after.ru_utime - before.ru_utime,
    'children_system_seconds': after.ru_stime - before.ru_stime,
    'children_maxrss_kib': after.ru_maxrss, 'exit_code': r.returncode,
    'stdout': r.stdout, 'stderr': r.stderr,
    'output_sha256': hashlib.sha256((r.stdout + r.stderr).encode()).hexdigest(),
    'host_cpu_count': os.cpu_count(), 'cpu_affinity_count': len(os.sched_getaffinity(0)),
    'cpu_max': pathlib.Path('/sys/fs/cgroup/cpu.max').read_text().strip(),
    'platform': platform.platform(),
}
pathlib.Path(a.output).write_text(json.dumps(out, indent=2) + '\n')
print(json.dumps({k: v for k, v in out.items() if k not in ('stdout', 'stderr')}))

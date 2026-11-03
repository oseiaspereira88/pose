#!/usr/bin/env python3
"""Measure committed governance input in disposable worktrees.

Keep raw outputs and errors. wait4 CPU/RSS are observations, not heap/I/O
estimates. Setup is excluded; each sample starts in a fresh worktree.
"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import platform
import shutil
import signal
import statistics
import subprocess
import tempfile
import time


def sha(raw):
    return "sha256:" + hashlib.sha256(raw).hexdigest()


def measure(command, cwd, env, timeout, stdout, stderr):
    start = time.monotonic_ns()
    timed_out = False
    with stdout.open("wb") as out, stderr.open("wb") as err:
        process = subprocess.Popen(command, cwd=cwd, env=env, stdout=out, stderr=err, start_new_session=True)
        while True:
            pid, status, usage = os.wait4(process.pid, os.WNOHANG)
            if pid:
                break
            if (time.monotonic_ns() - start) / 1e9 >= timeout:
                timed_out = True
                try:
                    os.killpg(process.pid, signal.SIGKILL)
                except ProcessLookupError:
                    pass
                _, status, usage = os.wait4(process.pid, 0)
                break
            time.sleep(0.001)
        exit_code = process.returncode = os.waitstatus_to_exitcode(status)
    metrics = {"user_seconds": usage.ru_utime, "system_seconds": usage.ru_stime,
               "max_rss": usage.ru_maxrss, "max_rss_unit": "KiB" if platform.system() == "Linux" else "platform-native"}
    return exit_code, timed_out, (time.monotonic_ns() - start) / 1e6, metrics


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--runs", type=int, default=3)
    parser.add_argument("--binary", default="pose")
    parser.add_argument("--root", default=".")
    parser.add_argument("--timeout", type=float, default=600)
    parser.add_argument("--output-dir")
    parser.add_argument("--operation", action="append", choices=["state-attention", "check-strict", "index"])
    args = parser.parse_args()
    if not 1 <= args.runs <= 100 or not 0 < args.timeout <= 86400:
        parser.error("runs must be 1..100 and timeout must be >0 and <=86400 seconds")
    env = os.environ.copy()
    for key in list(env):
        if key.startswith(("POSE_", "HARNE8_", "GIT_")):
            env.pop(key, None)
    env.update(GIT_NO_REPLACE_OBJECTS="1", GIT_CONFIG_GLOBAL=os.devnull,
               GIT_CONFIG_NOSYSTEM="1", GIT_OPTIONAL_LOCKS="0", NO_COLOR="1", LC_ALL="C")
    git_command = ["git", "-c", "core.hooksPath=" + os.devnull, "-c", "core.fsmonitor=false"]

    def git(root, *argv):
        return subprocess.check_output([*git_command, "-C", str(root), *argv], env=env, timeout=30)

    root = Path(git(args.root, "rev-parse", "--show-toplevel").decode().strip()).resolve()
    binary = shutil.which(args.binary)
    if binary is None:
        parser.error("binary not found")
    output = Path(args.output_dir).resolve() if args.output_dir else Path(tempfile.mkdtemp(prefix="pose-governance-bench-"))
    if output == root or root in output.parents:
        parser.error("output directory must be outside the source checkout")
    output.mkdir(parents=True, exist_ok=True)
    if any(output.iterdir()):
        parser.error("output directory must be empty")
    binary_copy = output / "pose"
    shutil.copy2(binary, binary_copy)
    binary_copy.chmod(0o700)
    report = {"schema_version": 1, "setup": [], "samples": [], "summary": {}}
    (output / "report.json").write_text(json.dumps(report, indent=2) + "\n")

    def setup(argv, phase):
        stdout, stderr = output / (phase + ".stdout"), output / (phase + ".stderr")
        code, timed_out, elapsed, metrics = measure(argv, output, env, args.timeout, stdout, stderr)
        report["setup"].append({"phase": phase, "exit_code": code, "timeout": timed_out,
            "wall_ms": elapsed, "resource_usage": metrics, "stdout": str(stdout), "stderr": str(stderr)})
        (output / "report.json").write_text(json.dumps(report, indent=2) + "\n")
        if code != 0 or timed_out:
            print(json.dumps({"report": str(output / "report.json"), "setup_failed": phase}))
            raise SystemExit(1)
        return stdout.read_bytes()

    def capture_git(directory, phase, *argv):
        return setup([*git_command, "-C", str(directory), *argv], phase)

    revision = capture_git(root, "source-head", "rev-parse", "HEAD").decode().strip()
    refs = capture_git(root, "source-refs", "for-each-ref", "--format=%(refname) %(objectname)")
    version_exit, version_timeout, _, _ = measure([str(binary_copy), "version"], output, env,
        min(args.timeout, 30), output / "version.stdout", output / "version.stderr")
    version_stdout = (output / "version.stdout").read_bytes()
    report.update({"corpus_revision": revision, "refs_digest": sha(refs),
              "source_dirty_status_hex": capture_git(root, "source-status", "status", "--porcelain=v1", "-z").hex(),
              "binary_digest": sha(binary_copy.read_bytes()), "engine_version_exit": version_exit,
              "engine_version_timeout": version_timeout,
              "engine_version": version_stdout.decode(errors="replace").strip(), "platform": platform.platform(),
              "environment": {key: env.get(key) for key in ("LANG", "LC_ALL", "TZ", "GOMAXPROCS", "TMPDIR")},
              "timeout_seconds": args.timeout,
              "git_configuration": "global/system config disabled; setup hooks/fsmonitor disabled; fresh mirror local config; source local config not copied",
              "limits": ["committed corpus only; dirty files excluded without restoring them",
                         "heap/reads/parses/copies/Git bytes need separate instrumented producer profiles",
                         "setup excluded; fresh worktree per sample; CPU/RSS from Unix wait4 (RSS units platform dependent)",
                         "wait4 includes reaped descendants; background/unreaped descendants are not a complete process-tree profile"]})
    (output / "source-refs.txt").write_bytes(refs)
    (output / "report.json").write_text(json.dumps(report, indent=2) + "\n")

    mirror = output / "corpus.git"
    setup([*git_command, "clone", "--mirror", "--no-hardlinks", "--dissociate", str(root), str(mirror)], "clone")
    captured_refs = capture_git(mirror, "mirror-refs", "for-each-ref", "--format=%(refname) %(objectname)")
    current_head = capture_git(root, "source-head-recheck", "rev-parse", "HEAD").decode().strip()
    if captured_refs != refs or current_head != revision:
        report["capture_conflict"] = "corpus refs changed during capture; no timings are valid"
        (output / "report.json").write_text(json.dumps(report, indent=2) + "\n")
        print(json.dumps({"report": str(output / "report.json"), "capture_conflict": report["capture_conflict"]}))
        return 1
    report["matrix_digest"] = sha(capture_git(mirror, "matrix", "show", revision + ":.pose/indexes/validation-matrix.json"))
    operations = {"state-attention": ["state", "--attention"], "check-strict": ["check", "--strict"], "index": ["index"]}
    invalid = version_exit != 0
    for operation in args.operation or operations:
        values = []
        for repetition in range(args.runs):
            name = f"{operation}-{repetition + 1}"
            work = output / (name + "-work")
            setup([*git_command, "--git-dir", str(mirror), "worktree", "add", "--detach", str(work), revision], name + "-prepare")
            stdout, stderr = [output / (name + suffix) for suffix in (".stdout", ".stderr")]
            command = [str(binary_copy), *operations[operation]]
            exit_code, timed_out, elapsed, metrics = measure(command, work, env, args.timeout, stdout, stderr)
            valid = exit_code == 0 and not timed_out and version_exit == 0
            artifacts, capture_error = [], None
            if operation == "index":
                artifact_dir = output / (name + "-artifacts")
                artifact_dir.mkdir(mode=0o700)
                try:
                    for filename in ("repo-map.json", "services.json", "packages.json", "spec-graph.json", "roadmaps.json", "delivery-integrity.json", "releases.json", "generation.json"):
                        source = work / ".pose" / "indexes" / filename
                        if not source.is_file():
                            continue
                        target = artifact_dir / filename
                        shutil.copyfile(source, target)
                        artifacts.append({"name": filename, "path": str(target), "bytes": target.stat().st_size, "digest": sha(target.read_bytes())})
                except OSError as exc:
                    capture_error = str(exc)
                    valid = False
            report["samples"].append({"operation": operation, "repetition": repetition + 1,
                "command": command, "exit_code": exit_code, "timeout": timed_out, "valid": valid,
                "wall_ms": elapsed, "resource_usage": metrics, "stdout_bytes": stdout.stat().st_size,
                "stderr_bytes": stderr.stat().st_size, "stdout": str(stdout), "stderr": str(stderr),
                "artifacts": artifacts, "artifact_capture_error": capture_error})
            if valid:
                values.append(elapsed)
            else:
                invalid = True
            (output / "report.json").write_text(json.dumps(report, indent=2) + "\n")
            # This is our disposable worktree, never the source checkout.
            setup([*git_command, "--git-dir", str(mirror), "worktree", "remove", "--force", str(work)], name + "-cleanup")
        report["summary"][operation] = {"valid_samples": len(values), "invalid_samples": args.runs - len(values),
            "median_ms": statistics.median(values) if values else None,
            "min_ms": min(values) if values else None, "max_ms": max(values) if values else None,
            "stdev_ms": statistics.stdev(values) if len(values) > 1 else None}
    (output / "report.json").write_text(json.dumps(report, indent=2) + "\n")
    print(json.dumps({"report": str(output / "report.json"), "summary": report["summary"]}))
    return 1 if invalid else 0


if __name__ == "__main__":
    raise SystemExit(main())

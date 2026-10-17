#!/usr/bin/env bash
# The test scripts remove the temporary files they create (spec
# pose-test-scripts-remove-their-temp-files).
#
# Three things are checked:
#   1. every script that calls mktemp either has an EXIT trap that removes what
#      it created or initializes tests/lib/tempdir.sh before its first mktemp,
#      and the scan is shown rejecting the shapes that only look like cleanup;
#   2. the helper removes its directory when the script succeeds, fails an
#      assertion, errors under `set -e`, or is stopped by INT or TERM;
#   3. the quickstart script, which builds its own binary, run with a TMPDIR of
#      its own, leaves nothing in it.
set -euo pipefail

cd "$(dirname "$0")/../.."
ROOT="$PWD"
fail() { printf '\033[31mFAIL:\033[0m %s\n' "$1" >&2; exit 1; }
ok()   { printf '\033[32m  ok\033[0m %s\n' "$1"; }

SANDBOX="$(mktemp -d)"
trap 'chmod -R u+w "$SANDBOX" 2>/dev/null || true; rm -rf "$SANDBOX"' EXIT

# 1. Static: no script creates a temporary file it never removes. The scan
# reads only code (comment lines are dropped) and accepts either an
# initialization of the helper before the first mktemp, or an EXIT trap whose
# command removes a variable that the script assigned from mktemp.
scan() { python3 -I - "$@" <<'PY'
import re, sys
for path in sys.argv[1:]:
    lines = [l for l in open(path).read().split("\n") if not l.lstrip().startswith("#")]
    code = "\n".join(lines)
    mk = next((i for i, l in enumerate(lines) if "mktemp" in l), None)
    if mk is None:
        continue
    init = next((i for i, l in enumerate(lines) if re.fullmatch(r"\s*pose_test_tmp_init\s*", l)), None)
    if init is not None:
        if init > mk:
            print(f"{path}: calls mktemp (line {mk + 1}) before pose_test_tmp_init")
        continue
    made = set(re.findall(r"^\s*(?:local\s+)?(\w+)=.*mktemp", code, re.M))
    def removes(body):
        return any(re.search(r"\brm\s+-rf?\b[^;&|]*\$\{?" + v + r"\b", body) for v in made)
    ok = False
    for m in re.finditer(r"""^\s*trap\s+(?:'([^']*)'|"([^"]*)"|(\w+))\s+EXIT\b""", code, re.M):
        body = m.group(1) if m.group(1) is not None else m.group(2)
        if body is None:
            fn = re.search(r"\b" + m.group(3) + r"\s*\(\)\s*\{([^}]*)\}", code)
            body = fn.group(1) if fn else ""
        if removes(body):
            ok = True
    if not ok:
        print(f"{path}: creates temporary files with mktemp and never removes them: add pose_test_tmp_init (tests/lib/tempdir.sh) or an EXIT trap that removes them")
PY
}
mapfile -t scripts < <(git ls-files 'tests/*.sh' 'scripts/*.sh' 'examples/*.sh' 'install.sh' | sort | grep -v -e '^tests/lib/tempdir.sh$' -e '^tests/hygiene/temp-cleanup.sh$')
checked=0
for script in "${scripts[@]}"; do grep -q mktemp "$script" && checked=$((checked + 1)); done
[ "$checked" -gt 0 ] || fail "no script with mktemp was found: the scan itself is broken"
found="$(scan "${scripts[@]}")" || fail "the scan crashed"
[ -z "$found" ] || fail "$found"
ok "static: $checked scripts that call mktemp remove what they create"

# The scan must be able to fail: each of these leaks and is reported.
fixtures="$SANDBOX/fixtures"; mkdir -p "$fixtures"
mkfix() { printf '%s\n' "${@:2}" > "$fixtures/$1.sh"; }
mkfix empty-trap 'w="$(mktemp -d)"' "trap '' EXIT"
mkfix commented-trap 'w="$(mktemp -d)"' "# trap 'rm -rf \"\$w\"' EXIT"
mkfix wrong-signal 'w="$(mktemp -d)"' "trap 'rm -rf \"\$w\"' INT"
mkfix foreign-path 'w="$(mktemp -d)"' "trap 'rm -rf /var/cache/x' EXIT"
mkfix commented-init 'w="$(mktemp -d)"' '# pose_test_tmp_init'
mkfix init-after 'w="$(mktemp -d)"' 'pose_test_tmp_init'
mkfix empty-function 'w="$(mktemp -d)"' 'cleanup() { :; }' 'trap cleanup EXIT'
for f in "$fixtures"/*.sh; do
  [ -n "$(scan "$f")" ] || fail "the scan accepted $(basename "$f"), which leaks"
done
mkfix good-trap 'w="$(mktemp -d)"' "trap 'rm -rf \"\$w\"' EXIT"
mkfix good-function 'w="$(mktemp -d)"' 'cleanup() { rm -rf "$w"; }' 'trap cleanup EXIT'
mkfix good-init 'pose_test_tmp_init' 'w="$(mktemp -d)"'
for f in good-trap good-function good-init; do
  [ -z "$(scan "$fixtures/$f.sh")" ] || fail "the scan rejected $f: $(scan "$fixtures/$f.sh")"
done
ok "static: the scan rejects 7 leaking shapes and accepts 3 correct ones"

# 2. The helper cleans up on every way a script can end.
outer() { local d="$SANDBOX/$1"; mkdir -p "$d"; printf '%s' "$d"; }
run_helper() { # <name> <body>; the body runs in a shell that sourced the helper
  local dir; dir="$(outer "$1")"
  TMPDIR="$dir" bash -c ". '$ROOT/tests/lib/tempdir.sh'; pose_test_tmp_init; $2" >/dev/null 2>&1 || true
  [ -z "$(ls -A "$dir")" ] || fail "helper ($1): left behind: $(ls -A "$dir" | tr '\n' ' ')"
}
files='mktemp -d >/dev/null; mktemp >/dev/null; d="$(mktemp -d)"; echo x > "$d/f"; chmod 500 "$d"; mkdir "$d/a" "$d/b"; touch "$d/a/f"; chmod 400 "$d/a/f"; chmod 0000 "$d/a"; chmod 0400 "$d/b"'
run_helper success "set -e; $files"
run_helper assertion "set -e; $files; exit 1"
run_helper error-under-set-e "set -e; $files; false"
run_helper unset-variable "set -eu; $files; : \"\$NOT_SET\""
for sig in TERM INT; do
  dir="$(outer "signal-$sig")"
  # A background job of a non-interactive shell starts with INT ignored, and an
  # ignored signal cannot be trapped; python starts the probe with the default
  # dispositions and reports how it ended, so the signal is proven delivered.
  status="$(python3 -I - "$sig" "$dir" "$ROOT" "$files" <<'PY'
import os, signal, subprocess, sys, time
sig, d, root, files = sys.argv[1:5]
def reset():
    signal.signal(signal.SIGINT, signal.SIG_DFL)
    signal.signal(signal.SIGTERM, signal.SIG_DFL)
ready = d + ".ready"
code = f". '{root}/tests/lib/tempdir.sh'; pose_test_tmp_init; {files}; echo ready > '{ready}'; sleep 60 & wait"
p = subprocess.Popen(["bash", "-c", code], env=dict(os.environ, TMPDIR=d), preexec_fn=reset,
                     stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, start_new_session=True)
for _ in range(200):
    if os.path.exists(ready): break
    time.sleep(0.05)
else:
    p.kill(); print("never-started"); sys.exit(0)
os.kill(p.pid, getattr(signal, "SIG" + sig))
try:
    p.wait(timeout=20)
except subprocess.TimeoutExpired:
    print("did-not-stop"); sys.exit(0)
try: os.killpg(p.pid, signal.SIGKILL)
except ProcessLookupError: pass
print(p.returncode)
PY
)"
  want=$([ "$sig" = INT ] && echo 130 || echo 143)
  [ "$status" = "$want" ] || fail "helper ($sig): the probe ended with '$status', expected $want (the signal was not delivered and handled)"
  [ -z "$(ls -A "$dir")" ] || fail "helper ($sig): left behind: $(ls -A "$dir" | tr '\n' ' ')"
done
ok "helper: removes its directory on success, assertion, error, unset variable, TERM and INT"

# The hygiene check must be able to fail: a script that forgets to clean up is
# caught by the same logic.
leaky="$SANDBOX/leaky"; mkdir -p "$leaky"
TMPDIR="$leaky" bash -c 'mktemp -d >/dev/null' >/dev/null 2>&1
[ -n "$(ls -A "$leaky")" ] || fail "control: a script without the helper left nothing, so the checks above prove nothing"
ok "control: a script without the helper does leave its directory"

# 3. A real script, run with a TMPDIR of its own, leaves it empty. The
# quickstart builds its own binary, which is what used to be left behind.
dir="$(outer real)"
TMPDIR="$dir" bash tests/quickstart/first-governed-loop.sh >/dev/null 2>&1 ||
  fail "the quickstart script failed"
[ -z "$(ls -A "$dir")" ] || fail "the quickstart script left behind: $(ls -A "$dir" | tr '\n' ' ')"
ok "quickstart script: leaves its TMPDIR empty"

printf '\n\033[32mThe test scripts remove their temporary files.\033[0m\n'

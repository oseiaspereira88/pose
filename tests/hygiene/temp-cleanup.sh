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

# 1. Static: no script creates a temporary file it never removes. Shell cannot
# be proven to clean up by reading it, so the scan accepts only two plain forms
# and rejects everything else, including text that merely looks like cleanup:
#   - the helper: tests/lib/tempdir.sh sourced and `pose_test_tmp_init` called
#     as a top-level command before the first mktemp, with no later EXIT trap
#     that drops the helper's, and no redefinition of its functions;
#   - an own trap: every mktemp is assigned to a variable, exactly one
#     `trap '...' EXIT` (or a one-line function it names) exists, and it
#     removes each of those variables whole with `rm -rf "$var"`.
# A line that follows a top-level exit or return is unreachable and not counted.
scan() { python3 -I - "$@" <<'PY'
import re, sys

def reachable(lines):
    out = []
    for i, l in enumerate(lines):
        out.append((i, l))
        if re.match(r"(exit|return)\b", l):
            break
    return out

for path in sys.argv[1:]:
    raw = open(path).read().split("\n")
    lines = [("" if l.lstrip().startswith("#") else l) for l in raw]
    live = reachable(lines)
    text = "\n".join(l for _, l in live)
    mk = [(i, l) for i, l in live if "mktemp" in l]
    if not mk:
        continue
    first = mk[0][0]
    problems = []
    traps = [(i, l) for i, l in live if re.match(r"\s*trap\b", l)]
    exit_traps = [(i, l) for i, l in traps if re.search(r"\bEXIT\b", l)]
    sig_traps = [(i, l) for i, l in traps if re.search(r"\b(INT|TERM)\b", l)]
    uses_helper = any(re.fullmatch(r"pose_test_tmp_init", l) for _, l in live)
    if uses_helper:
        init = next(i for i, l in live if re.fullmatch(r"pose_test_tmp_init", l))
        src = [i for i, l in live if re.match(r"(source|\.)\s+.*tempdir\.sh\"?\s*$", l.strip())]
        if not src or min(src) > init:
            problems.append("pose_test_tmp_init is called without sourcing tempdir.sh before it")
        if init > first:
            problems.append(f"mktemp (line {first + 1}) comes before pose_test_tmp_init")
        if re.search(r"^\s*(function\s+)?pose_test_tmp_(init|remove)\b\s*(\(\))?\s*\{?", text, re.M) and re.search(r"^\s*(function\s+)?pose_test_tmp_(init|remove)\s*\(\)", text, re.M):
            problems.append("redefines a helper function")
        for i, l in exit_traps + sig_traps:
            if i > init and "pose_test_tmp_remove" not in l:
                problems.append(f"line {i + 1} replaces the helper's trap without calling pose_test_tmp_remove")
    else:
        assigned = {}
        for i, l in mk:
            m = re.match(r"\s*(?:local\s+)?(\w+)=\"?\$\(mktemp\b[^)]*\)\"?\s*$", l)
            if not m:
                problems.append(f"line {i + 1}: mktemp is not assigned whole to a variable")
            else:
                assigned[m.group(1)] = i
        if len(exit_traps) != 1:
            problems.append(f"expected exactly one EXIT trap, found {len(exit_traps)}")
        else:
            l = exit_traps[0][1]
            m = re.match(r"""\s*trap\s+(?:'([^']*)'|(\w+))\s+EXIT\s*$""", l)
            if not m:
                problems.append("the EXIT trap is not `trap '<command>' EXIT` or `trap <function> EXIT`")
            else:
                body = m.group(1)
                if body is None:
                    fn = re.search(r"^\s*" + m.group(2) + r"\s*\(\)\s*\{([^}]*)\}\s*$", text, re.M)
                    body = fn.group(1) if fn else ""
                stmts = [x.strip() for x in body.split(";") if x.strip()]
                removed = set()
                for st in stmts:
                    r = re.fullmatch(r"rm\s+-rf?\s+((?:\"\$\{?\w+\}?\"\s*)+)", st)
                    if r:
                        removed |= set(re.findall(r"\$\{?(\w+)\}?", r.group(1)))
                for v in assigned:
                    if v not in removed:
                        problems.append(f"the EXIT trap does not remove ${v}")
                if "#" in body or re.search(r"\becho\b|\bprintf\b", body):
                    problems.append("the EXIT trap body contains a comment or a print")
        for i, l in sig_traps:
            problems.append(f"line {i + 1}: an INT/TERM trap must exit so the EXIT trap runs; use the helper")
    for pr in problems:
        print(f"{path}: {pr}")
PY
}
mapfile -t scripts < <(git ls-files 'tests/*.sh' 'scripts/*.sh' 'examples/*.sh' 'install.sh' | sort | grep -v -e '^tests/lib/tempdir.sh$' -e '^tests/hygiene/temp-cleanup.sh$')
checked=0
for script in "${scripts[@]}"; do grep -q mktemp "$script" && checked=$((checked + 1)); done
[ "$checked" -gt 0 ] || fail "no script with mktemp was found: the scan itself is broken"
found="$(scan "${scripts[@]}")" || fail "the scan crashed"
[ -z "$found" ] || fail "$found"
ok "static: $checked scripts that call mktemp remove what they create"

# The scan must be able to fail: each of these leaks and must be reported.
fixtures="$SANDBOX/fixtures"; mkdir -p "$fixtures"
mkfix() { printf '%s\n' "${@:2}" > "$fixtures/$1.sh"; }
SRC='source tests/lib/tempdir.sh'
mkfix empty-trap 'w="$(mktemp -d)"' "trap '' EXIT"
mkfix commented-trap 'w="$(mktemp -d)"' "# trap 'rm -rf \"\$w\"' EXIT"
mkfix wrong-signal 'w="$(mktemp -d)"' "trap 'rm -rf \"\$w\"' INT"
mkfix foreign-path 'w="$(mktemp -d)"' "trap 'rm -rf /var/cache/x' EXIT"
mkfix commented-init 'w="$(mktemp -d)"' '# pose_test_tmp_init'
mkfix init-after 'w="$(mktemp -d)"' "$SRC" 'pose_test_tmp_init'
mkfix empty-function 'w="$(mktemp -d)"' 'cleanup() { :; }' 'trap cleanup EXIT'
mkfix one-of-two-roots 'a="$(mktemp -d)"' 'b="$(mktemp -d)"' "trap 'rm -rf \"\$a\"' EXIT"
mkfix extra-unassigned 'w="$(mktemp -d)"' 'mktemp -d >/dev/null' "trap 'rm -rf \"\$w\"' EXIT"
mkfix root-plus-file 'w="$(mktemp -d)/file"' "trap 'rm -rf \"\$w\"' EXIT"
mkfix child-only 'w="$(mktemp -d)"' "trap 'rm -rf \"\$w/child\"' EXIT"
mkfix echo-only 'w="$(mktemp -d)"' "trap 'echo rm -rf \"\$w\"' EXIT"
mkfix comment-in-body 'w="$(mktemp -d)"' "trap ': # rm -rf \"\$w\"' EXIT"
mkfix escaped-variable 'w="$(mktemp -d)"' "trap 'rm -rf \\\$w' EXIT"
mkfix overwritten-trap 'w="$(mktemp -d)"' "trap 'rm -rf \"\$w\"' EXIT" "trap ':' EXIT"
mkfix unreachable-trap 'w="$(mktemp -d)"' 'exit 0' "trap 'rm -rf \"\$w\"' EXIT"
mkfix noop-init 'w="$(mktemp -d)"' "$SRC" 'pose_test_tmp_init() { :; }' 'pose_test_tmp_init'
mkfix unreachable-init "$SRC" 'w="$(mktemp -d)"' 'exit 0' 'pose_test_tmp_init'
mkfix init-without-source 'pose_test_tmp_init' 'w="$(mktemp -d)"'
mkfix overwritten-helper-trap "$SRC" 'pose_test_tmp_init' "trap 'echo bye' EXIT" 'w="$(mktemp -d)"'
negatives=0
for f in "$fixtures"/*.sh; do
  negatives=$((negatives + 1))
  [ -n "$(scan "$f")" ] || fail "the scan accepted $(basename "$f"), which leaks"
done
mkfix good-trap 'w="$(mktemp -d)"' "trap 'rm -rf \"\$w\"' EXIT"
mkfix good-two-roots 'a="$(mktemp -d)"' 'b="$(mktemp -d)"' "trap 'rm -rf \"\$a\" \"\$b\"' EXIT"
mkfix good-function 'w="$(mktemp -d)"' 'cleanup() { rm -rf "$w"; }' 'trap cleanup EXIT'
mkfix good-init "$SRC" 'pose_test_tmp_init' 'w="$(mktemp -d)"'
mkfix good-init-own-trap "$SRC" 'pose_test_tmp_init' "trap 'pose_test_tmp_remove' EXIT" 'w="$(mktemp -d)"'
positives=0
for f in good-trap good-two-roots good-function good-init good-init-own-trap; do
  positives=$((positives + 1))
  [ -z "$(scan "$fixtures/$f.sh")" ] || fail "the scan rejected $f: $(scan "$fixtures/$f.sh")"
done
ok "static: the scan rejects $negatives leaking shapes and accepts $positives correct ones"

# 2. The helper cleans up on every way a script can end. Each case must reach
# the end of its setup (a marker file proves it), or a setup error would pass
# as a clean run.
outer() { local d="$SANDBOX/$1"; mkdir -p "$d"; printf '%s' "$d"; }
HELPER="$ROOT/tests/lib/tempdir.sh"
run_helper() { # <name> <body>; prints the entries left in TMPDIR after the body ran
  local dir mark; dir="$(outer "$1")"; mark="$SANDBOX/$1.done"
  rm -f "$mark"
  MARK="$mark" TMPDIR="$dir" bash -c ". '$HELPER'; pose_test_tmp_init; $2" >/dev/null 2>&1 || true
  [ -f "$mark" ] || { printf 'setup-not-reached'; return; }
  ls -A "$dir" | tr '\n' ' '
}
expect_clean() { local left; left="$(run_helper "$@")"; [ -z "$left" ] || fail "helper ($1): $left"; }
files='mktemp -d >/dev/null; mktemp >/dev/null; d="$(mktemp -d)"; mkdir "$d/a" "$d/b"; echo x > "$d/f"; touch "$d/a/f"; chmod 400 "$d/a/f"; chmod 0000 "$d/a"; chmod 0400 "$d/b"; chmod 500 "$d"; : > "$MARK"'
expect_clean success "set -e; $files"
expect_clean assertion "set -e; $files; exit 1"
expect_clean error-under-set-e "set -e; $files; false"
expect_clean unset-variable "set -eu; $files; : \"\$NOT_SET\""
# The permission cases must be able to fail: the helper as it was (chmod u+w
# only) leaves the restricted entries behind.
old="$SANDBOX/tempdir-old.sh"
sed 's/chmod -R u+rwx/chmod -R u+w/' "$HELPER" > "$old"
cmp -s "$old" "$HELPER" && fail "control: the defective helper is identical to the real one"
HELPER="$old"
left="$(run_helper permission-control "set -e; $files")"
HELPER="$ROOT/tests/lib/tempdir.sh"
case "$left" in ''|setup-not-reached) fail "control: the previous helper cleaned up (or setup failed: '$left'), so the permission cases prove nothing" ;; esac
chmod -R u+rwx "$SANDBOX/permission-control"
for sig in TERM INT; do
  dir="$(outer "signal-$sig")"
  # A background job of a non-interactive shell starts with INT ignored, and an
  # ignored signal cannot be trapped; python starts the probe with the default
  # dispositions and reports how it ended, so the signal is proven delivered.
  status="$(MARK="$SANDBOX/signal-$sig.done" python3 -I - "$sig" "$dir" "$ROOT" "$files" <<'PY'
import os, signal, subprocess, sys, time
sig, d, root, files = sys.argv[1:5]
def reset():
    signal.signal(signal.SIGINT, signal.SIG_DFL)
    signal.signal(signal.SIGTERM, signal.SIG_DFL)
ready = d + ".ready"
code = f"set -e; . '{root}/tests/lib/tempdir.sh'; pose_test_tmp_init; {files}; echo ready > '{ready}'; sleep 60 & wait"
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

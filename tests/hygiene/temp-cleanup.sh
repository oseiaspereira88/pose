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
# The restricted fixtures below (0400, 0000) must not outlive the gate, however
# it ends: restore access, then remove, on success, failure, INT and TERM.
trap 'chmod -R u+rwx "$SANDBOX" 2>/dev/null || true; rm -rf "$SANDBOX"' EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

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

HEREDOC = re.compile(r"<<-?\s*(['\"]?)(\w+)\1")
DOCKER = re.compile(r'if \[ -n "\$\w+" \]; then docker rm -f "\$\w+" >/dev/null; fi;?')

def live_lines(path):
    """Code lines with comments and heredoc bodies blanked, cut after a
    top-level exit/return; returns (lines, unsupported_reason)."""
    out, end = [], None
    for l in open(path).read().split("\n"):
        if end is not None:
            if l.strip() == end:
                end = None
            out.append("")
            continue
        if l.lstrip().startswith("#"):
            out.append("")
            continue
        out.append(l)
        m = HEREDOC.search(l)
        if m:
            end = m.group(2)
    return out

def depth_kinds(lines):
    """For every line, the list of block kinds enclosing it, or None when the
    nesting cannot be followed (unsupported)."""
    stack, kinds = [], []
    nid = 0
    for l in lines:
        kinds.append(list(stack))
        s = re.sub(r"'[^']*'|\"(?:[^\"\\]|\\.)*\"", '""', l.strip())
        if re.match(r"(if|for|while|until|case|select)\b", s):
            nid += 1; stack.append(("cond", nid))
        elif re.match(r"(function\s+\w+\s*(\(\))?|\w+\s*\(\))\s*\{\s*$", s):
            nid += 1; stack.append(("fn", nid))
        elif re.match(r"\{\s*$", s):
            nid += 1; stack.append(("group", nid))
        elif re.match(r"\(\s*$", s):
            nid += 1; stack.append(("sub", nid))
        for _ in re.findall(r"(?:^|[;\s])(?:fi|done|esac)\s*(?:$|[;&|)])", s):
            if not stack or stack[-1][0] != "cond":
                return None
            stack.pop()
        if re.match(r"\}", s) and stack and stack[-1][0] in ("fn", "group"):
            stack.pop()
        elif re.match(r"\)", s) and stack and stack[-1][0] == "sub":
            stack.pop()
    return kinds if not stack else None

def check(path):
    p = []
    lines = live_lines(path)
    # Statements after a top-level exit/return never run.
    for i, l in enumerate(lines):
        if re.match(r"(exit|return)\b", l):
            lines = lines[: i + 1]
            break
    text = "\n".join(lines)
    mk = [i for i, l in enumerate(lines) if re.search(r"\bmktemp\b", l)]
    if not mk:
        return p
    kinds = depth_kinds(lines)
    if kinds is None:
        return [f"{path}: block nesting cannot be followed (unsupported); simplify it or use the helper"]
    if re.search(r"\b(builtin|unset|alias|enable|exec)\b|\bcommand\s+trap\b", text):
        p.append("uses builtin/unset/alias/enable/exec/command trap, which can hide or drop cleanup")
    trap_lines = [i for i, l in enumerate(lines) if re.search(r"\btrap\b", l)]
    helper_init = [i for i, l in enumerate(lines) if re.fullmatch(r"\s*pose_test_tmp_init\s*", l)]
    if re.search(r"(?m)^\s*(function\s+)?pose_test_tmp_\w+\s*(\(\))?\s*\{", text) or re.search(r"function\s+pose_test_tmp_", text):
        p.append("defines a pose_test_tmp_* function")
    if re.search(r"(?m)^\s*(export\s+)?TMPDIR=", text) or re.search(r"\bexport\s+-n\b", text):
        p.append("assigns TMPDIR")
    def single_assignment(var):
        n = len(re.findall(r"(?<![\w$])" + var + r"\+?=", text))
        n += len(re.findall(r"\b(?:read|local|declare|typeset|printf\s+-v|mapfile)\b[^\n]*\b" + var + r"\b(?!=)", text))
        return n == 1 and not re.search(r"\(\(\s*" + var + r"\b", text)
    if helper_init:
        if len(helper_init) != 1:
            p.append("calls pose_test_tmp_init more than once")
        init = helper_init[0]
        src = [i for i, l in enumerate(lines) if re.match(r"(source|\.)\s+.*tempdir\.sh\"?\s*$", l.strip())]
        if len(src) != 1 or src[0] > init or kinds[src[0]]:
            p.append("tempdir.sh must be sourced once, at top level, before pose_test_tmp_init")
        if kinds[init]:
            p.append("pose_test_tmp_init must run at top level, not inside a block")
        if mk[0] < init:
            p.append(f"mktemp (line {mk[0] + 1}) comes before pose_test_tmp_init")
        for i in trap_lines:
            if i < init:
                continue
            m = re.fullmatch(r"""\s*trap\s+'([^']*)'\s+EXIT\s*""", lines[i])
            stmts = [x.strip() for x in m.group(1).split(";") if x.strip()] if m else []
            if not m or not stmts or stmts[-1] != "pose_test_tmp_remove" or any(re.match(r"(exit|return)\b", x) for x in stmts) or kinds[i] != kinds[init]:
                p.append(f"line {i + 1}: a trap after the helper must be a top-level `trap '<...>; pose_test_tmp_remove' EXIT`")
    else:
        assigned = []
        for i in mk:
            m = re.fullmatch(r"\s*(?:local\s+)?(\w+)=\"?\$\(mktemp\b[^()]*\)\"?\s*", lines[i])
            if not m:
                p.append(f"line {i + 1}: mktemp is not assigned whole to a variable")
            else:
                assigned.append((m.group(1), i))
        for v, i in assigned:
            if not single_assignment(v):
                p.append(f"${v} is assigned, read into or changed more than once, so the directory it held can be lost")
            if re.search(r"\bunset\b", lines[i]):
                p.append("unset")
        exit_traps = [i for i in trap_lines if re.search(r"\bEXIT\b", lines[i])]
        other_traps = [i for i in trap_lines if i not in exit_traps]
        if len(exit_traps) != 1 or len(trap_lines) != 1:
            p.append(f"expected exactly one trap, on EXIT; found {len(trap_lines)} trap lines")
        else:
            t = exit_traps[0]
            m = re.fullmatch(r"""\s*trap\s+(?:'([^']*)'|(\w+))\s+EXIT\s*""", lines[t])
            if not m:
                p.append("the EXIT trap is not `trap '<command>' EXIT` or `trap <function> EXIT`")
            else:
                for v, i in assigned:
                    if kinds[i] != kinds[t]:
                        p.append(f"the EXIT trap is not in the same block as the mktemp of ${v}")
                body = m.group(1)
                if body is None:
                    name = m.group(2)
                    defs = re.findall(r"(?m)^[ \t]*(?:function\s+)?" + name + r"\s*(?:\(\))?\s*\{([^\n}]*)\}\s*$", text)
                    anydef = re.findall(r"(?m)^[ \t]*(?:function\s+)?" + name + r"\s*(?:\(\))?\s*\{", text)
                    if len(defs) != 1 or len(anydef) != 1:
                        p.append(f"{name} must be defined exactly once, on one line")
                    body = defs[0] if defs else ""
                body = DOCKER.sub("", body)
                stmts = [x.strip() for x in body.split(";") if x.strip()]
                removed = set()
                for st in stmts:
                    r = re.fullmatch(r"rm\s+-rf?\s+((?:\"\$\{?\w+\}?\"\s*)+)", st)
                    if not r:
                        p.append(f"the EXIT trap runs `{st}`, which is not `rm -rf \"$var\"`")
                    else:
                        removed |= set(re.findall(r"\$\{?(\w+)\}?", r.group(1)))
                for v, i in assigned:
                    if v not in removed:
                        p.append(f"the EXIT trap does not remove ${v}")

    return [f"{path}: {x}" for x in p]

for path in sys.argv[1:]:
    for line in check(path):
        print(line)
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
mkfix var-reuse 'w="$(mktemp -d)"' 'w="$(mktemp -d)"' "trap 'rm -rf \"\$w\"' EXIT"
mkfix var-cleared 'w="$(mktemp -d)"' 'w=""' "trap 'rm -rf \"\$w\"' EXIT"
mkfix trap-in-if-false 'w="$(mktemp -d)"' 'if false; then' "  trap 'rm -rf \"\$w\"' EXIT" 'fi'
mkfix exit-in-trap 'w="$(mktemp -d)"' "trap 'exit 0; rm -rf \"\$w\"' EXIT"
mkfix return-in-function 'w="$(mktemp -d)"' 'cleanup() { return 0; rm -rf "$w"; }' 'trap cleanup EXIT'
mkfix redefined-function 'w="$(mktemp -d)"' 'cleanup() { rm -rf "$w"; }' 'cleanup() { :; }' 'trap cleanup EXIT'
mkfix builtin-trap 'w="$(mktemp -d)"' "trap 'rm -rf \"\$w\"' EXIT" "builtin trap : EXIT"
mkfix builtin-trap-over-helper "$SRC" 'pose_test_tmp_init' "builtin trap : EXIT" 'w="$(mktemp -d)"'
mkfix init-in-if-false "$SRC" 'if false; then' '  pose_test_tmp_init' 'fi' 'w="$(mktemp -d)"'
mkfix echo-remove-name "$SRC" 'pose_test_tmp_init' "trap 'echo pose_test_tmp_remove' EXIT" 'w="$(mktemp -d)"'
mkfix redefine-remove-no-parens "$SRC" 'pose_test_tmp_init' 'function pose_test_tmp_remove { :; }' 'w="$(mktemp -d)"'
mkfix heredoc-init 'cat <<EOF' "$SRC" 'pose_test_tmp_init' 'EOF' 'w="$(mktemp -d)"'
mkfix heredoc-trap 'w="$(mktemp -d)"' 'cat <<EOF' "trap 'rm -rf \"\$w\"' EXIT" 'EOF'
mkfix unset-tmpdir-after-init "$SRC" 'pose_test_tmp_init' 'unset TMPDIR' 'w="$(mktemp -d)"'
mkfix tmpdir-reassigned "$SRC" 'pose_test_tmp_init' 'TMPDIR=/var/tmp' 'w="$(mktemp -d)"'
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
# The same holds for a real script: repeating its mktemp line in a tracked copy
# makes the complete scan reject it.
sed '/^work="\$(mktemp -d)"$/p' tests/release/compat.sh > "$fixtures/compat-rebound.sh.txt"
cmp -s tests/release/compat.sh "$fixtures/compat-rebound.sh.txt" && fail "control: the rebound copy equals the original"
[ -n "$(scan "$fixtures/compat-rebound.sh.txt")" ] || fail "the scan accepted a tracked script with a rebound mktemp variable"
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

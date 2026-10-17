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

DOCKER = re.compile(r'if \[ -n "\$\w+" \]; then docker rm -f "\$\w+" >/dev/null; fi;?')

def heredoc_delimiter(line):
    """The delimiter of a heredoc started on this line, found outside quotes,
    comments and arithmetic (but inside `$( ... )`, which is code even within
    double quotes); None if the line starts none."""
    i, n = 0, len(line)
    q = None
    subs = []  # quote state to restore when each open `$(` closes
    depth = 0
    while i < n:
        c = line[i]
        if q:
            if c == "\\" and q == '"':
                i += 2
                continue
            if q == '"' and line.startswith("$(", i):
                subs.append(q)
                q = None
                depth = 1
                i += 2
                continue
            if c == q:
                q = None
        elif c == "\\":
            i += 2
            continue
        elif c in "'\"":
            q = c
        elif c == "(" and subs:
            depth += 1
        elif c == ")" and subs:
            depth -= 1
            if depth == 0:
                q = subs.pop()
        elif c == "#" and (i == 0 or line[i - 1] in " \t;&|("):
            return None
        elif line.startswith("<<<", i):
            i += 3
            continue
        elif line.startswith("<<", i):
            if line[:i].count("((") > line[:i].count("))"):
                i += 2
                continue
            m = re.match(r"<<(-?)\s*(?:'([^']+)'|\"([^\"]+)\"|([A-Za-z_][\w.-]*))", line[i:])
            if m:
                return (m.group(2) or m.group(3) or m.group(4), bool(m.group(1)))
        i += 1
    return None

def code_lines(path):
    """Lines with comments and heredoc bodies blanked."""
    out, end = [], None
    for l in open(path).read().split("\n"):
        if end is not None:
            word, strip = end
            out.append("")
            if (l.lstrip("\t") if strip else l) == word:
                end = None
            continue
        if l.lstrip().startswith("#"):
            out.append("")
            continue
        out.append(l)
        end = heredoc_delimiter(l)
    return out

def masked(l):
    return re.sub(r"'[^']*'|\"(?:[^\"\\]|\\.)*\"", '""', l.strip())

def block_stacks(lines):
    """For each line, the stack of enclosing blocks as (kind, id) pairs, or None
    when the nesting cannot be followed. `else`/`elif` start a new branch."""
    stack, stacks, nid = [], [], 0
    for l in lines:
        s = masked(l)
        stacks.append(list(stack))
        if re.match(r"(elif|else)\b", s):
            if not stack or stack[-1][0] != "if":
                return None
            nid += 1
            stack[-1] = ("if", nid)
        if re.match(r"if\b", s):
            nid += 1; stack.append(("if", nid))
        elif re.match(r"(for|while|until|select)\b", s):
            nid += 1; stack.append(("loop", nid))
        elif re.match(r"case\b", s):
            nid += 1; stack.append(("case", nid))
        elif re.match(r"(function\s+\w+\s*(\(\))?|\w+\s*\(\))\s*\{\s*$", s):
            nid += 1; stack.append(("fn", nid))
        elif re.match(r"\{\s*$", s):
            nid += 1; stack.append(("group", nid))
        elif re.match(r"\(\s*$", s):
            nid += 1; stack.append(("sub", nid))
        for _ in re.findall(r"(?:^|[;\s])(?:fi|done|esac)\s*(?:$|[;&|)])", s):
            if not stack or stack[-1][0] not in ("if", "loop", "case"):
                return None
            stack.pop()
        if re.match(r"\}", s) and stack and stack[-1][0] in ("fn", "group"):
            stack.pop()
        elif re.match(r"\)", s) and stack and stack[-1][0] == "sub":
            stack.pop()
    return stacks if not stack else None

def check(path):
    p = []
    lines = code_lines(path)
    if not any(re.search(r"\bmktemp\b", l) for l in lines):
        return p
    stacks = block_stacks(lines)
    if stacks is None:
        return ["block nesting cannot be followed (unsupported); simplify it or use the helper"]
    # Only a top-level exit/return ends the script: the lines after it never run.
    for i, l in enumerate(lines):
        if not stacks[i] and re.match(r"(exit|return)\b", masked(l)):
            lines, stacks = lines[: i + 1], stacks[: i + 1]
            break
    text = "\n".join(lines)
    mk = [i for i, l in enumerate(lines) if re.search(r"\bmktemp\b", l)]
    if not mk:
        return p
    if re.search(r"\b(builtin|unset|alias|enable|exec)\b|\bcommand\s+trap\b", text):
        p.append("uses builtin/unset/alias/enable/exec/command trap, which can hide or drop cleanup")
    if re.search(r"(?<![\w$])(TMPDIR|POSE_TEST_TMP)\+?=", text) or re.search(r"\b(export\s+-n|declare\s+\+x)\b", text):
        p.append("assigns TMPDIR or POSE_TEST_TMP")
    if re.search(r"(?m)^\s*(function\s+)?pose_test_tmp_\w+\s*(\(\))?\s*\{", text):
        p.append("defines a pose_test_tmp_* function")
    for i in mk:
        for args in re.findall(r"\bmktemp\b([^)\n]*)", lines[i]):
            if "/" in args or re.search(r"(^|\s)(-p|--tmpdir|--suffix)\b", args):
                p.append(f"line {i + 1}: mktemp with a path/tmpdir option escapes the owned directory")
    if re.search(r"(?m)^\s*(function\s+)?(rm|trap|chmod|find)\s*(\(\))?\s*\{", text):
        p.append("redefines rm, trap, chmod or find, which the cleanup relies on")
    def mutated(v):
        return bool(re.search(r"\bfor\s+" + v + r"\b|\b(?:select|getopts)\b[^\n]*\b" + v + r"\b|\bread\b[^\n]*\b" + v + r"\b|(?<![\w$])" + v + r"\[|\b(?:printf\s+-v|mapfile|readarray)\b[^\n]*\b" + v + r"\b|\(\(\s*" + v + r"\b", text))
    for v in ("TMPDIR", "POSE_TEST_TMP"):
        if mutated(v):
            p.append(f"changes {v} through for/read/select/array/printf -v")
    trap_lines = [i for i, l in enumerate(lines) if re.search(r"\btrap\b", l)]
    inits = [i for i, l in enumerate(lines) if re.fullmatch(r"\s*pose_test_tmp_init\s*", l)]
    if inits:
        init = inits[0]
        if len(inits) != 1:
            p.append("calls pose_test_tmp_init more than once")
        src = [i for i, l in enumerate(lines) if re.match(r"(source|\.)\s+.*tempdir\.sh\"?\s*$", l.strip())]
        if len(src) != 1 or src[0] > init or stacks[src[0]]:
            p.append("tempdir.sh must be sourced once, at top level, before pose_test_tmp_init")
        if stacks[init]:
            p.append("pose_test_tmp_init must run at top level, not inside a block")
        if mk[0] < init:
            p.append(f"mktemp (line {mk[0] + 1}) comes before pose_test_tmp_init")
        for i in trap_lines:
            if i > init and (not re.fullmatch(r"""\s*trap\s+'pose_test_tmp_remove'\s+EXIT\s*""", lines[i]) or stacks[i]):
                p.append(f"line {i + 1}: after the helper the only trap allowed is a top-level `trap 'pose_test_tmp_remove' EXIT`")
    else:
        def only_if(i):
            return all(k == "if" for k, _ in stacks[i])
        assigned = []
        for i in mk:
            m = re.fullmatch(r"\s*(?:local\s+)?(\w+)=\"?\$\(mktemp\b[^()]*\)\"?\s*", lines[i])
            if not m:
                p.append(f"line {i + 1}: mktemp is not assigned whole to a variable")
            elif not only_if(i):
                p.append(f"line {i + 1}: mktemp sits in a loop, function, group or case arm, where it can run more than once")
            else:
                assigned.append((m.group(1), i))
        for v, i in assigned:
            n = len(re.findall(r"(?<![\w$])" + v + r"\+?=", text))
            n += len(re.findall(r"\b(?:local|declare|typeset)\b[^\n]*\b" + v + r"\b(?!=)", text))
            if n != 1 or mutated(v):
                p.append(f"${v} is assigned, read into or changed more than once, so the directory it held can be lost")
        if len(trap_lines) != 1 or not re.search(r"\bEXIT\b", lines[trap_lines[0]]):
            p.append(f"expected exactly one trap, on EXIT; found {len(trap_lines)} trap lines")
        else:
            t = trap_lines[0]
            m = re.fullmatch(r"""\s*trap\s+(?:'([^']*)'|(\w+))\s+EXIT\s*""", lines[t])
            if not m:
                p.append("the EXIT trap is not `trap '<command>' EXIT` or `trap <function> EXIT`")
            else:
                if not only_if(t):
                    p.append("the EXIT trap sits in a loop, function, group or case arm")
                if assigned:
                    first_alloc = min(i for _, i in assigned)
                    names = {v for v, _ in assigned}
                    for j in range(first_alloc, t):
                        s = masked(lines[j])
                        if not s or j in {i for _, i in assigned}:
                            continue
                        if re.fullmatch(r"\w+=\"\$\{?\w+\}?(/[\w./-]*)?\"", s) or re.fullmatch(r"\w+=\"\"", s) or re.fullmatch(r"(function\s+)?\w+\s*(\(\))?\s*\{[^\n}]*\}", s):
                            continue
                        p.append(f"line {j + 1} runs between the allocation and the EXIT trap, so a failure there leaks the directory: put the trap right after the allocations")
                for v, i in assigned:
                    if stacks[i] != stacks[t]:
                        p.append(f"the EXIT trap is not in the same branch as the mktemp of ${v}")
                body = m.group(1)
                if body is None:
                    name = m.group(2)
                    defs = [i for i, l in enumerate(lines) if re.match(r"[ \t]*(?:function\s+)?" + name + r"\s*(?:\(\))?\s*\{", l)]
                    one = [re.fullmatch(r"[ \t]*(?:function\s+)?" + name + r"\s*(?:\(\))?\s*\{([^\n}]*)\}\s*", lines[i]) for i in defs]
                    split_def = re.search(r"(?m)^\s*" + name + r"\s*\(\)\s*$|^\s*function\s+" + name + r"\s*$", text)
                    if len(defs) != 1 or not one[0] or stacks[defs[0]] or split_def:
                        p.append(f"{name} must be defined once, on one line, at top level")
                        body = ""
                    else:
                        body = one[0].group(1)
                body = DOCKER.sub("", body)
                removed = set()
                for st in (x.strip() for x in body.split(";") if x.strip()):
                    r = re.fullmatch(r"rm\s+-rf?\s+((?:\"\$\{?\w+\}?\"\s*)+)", st)
                    if not r:
                        p.append(f"the EXIT trap runs `{st}`, which is not `rm -rf \"$var\"`")
                    else:
                        removed |= set(re.findall(r"\$\{?(\w+)\}?", r.group(1)))
                for v, _ in assigned:
                    if v not in removed:
                        p.append(f"the EXIT trap does not remove ${v}")
    return p

for path in sys.argv[1:]:
    for line in check(path):
        print(f"{path}: {line}")
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
mkfix exit-in-skipped-branch 'if false; then' '  exit 0' 'fi' 'w="$(mktemp -d)"'
mkfix alloc-then-trap-else 'if true; then' '  w="$(mktemp -d)"' 'else' "  trap 'rm -rf \"\$w\"' EXIT" 'fi'
mkfix alloc-in-loop 'for i in 1 2; do' '  w="$(mktemp -d)"' "  trap 'rm -rf \"\$w\"' EXIT" 'done'
mkfix alloc-in-function 'f() {' '  local w="$(mktemp -d)"' "  trap 'rm -rf \"\$w\"' EXIT" '}' 'f'
mkfix function-called-twice 'f() {' '  w="$(mktemp -d)"' '}' 'f' 'f' "trap 'rm -rf \"\$w\"' EXIT"
mkfix cleanup-defined-in-skipped-branch 'w="$(mktemp -d)"' 'if false; then' '  cleanup() { rm -rf "$w"; }' 'fi' 'trap cleanup EXIT'
mkfix alloc-and-trap-in-case-arms 'case "$1" in' '  a) w="$(mktemp -d)" ;;' "  b) trap 'rm -rf \"\$w\"' EXIT ;;" 'esac'
mkfix false-before-remove "$SRC" 'pose_test_tmp_init' "trap 'false; pose_test_tmp_remove' EXIT" 'w="$(mktemp -d)"'
mkfix cleared-helper-root "$SRC" 'pose_test_tmp_init' 'POSE_TEST_TMP=' 'w="$(mktemp -d)"'
mkfix absolute-template "$SRC" 'pose_test_tmp_init' 'w="$(mktemp -d /var/tmp/x.XXXXXX)"'
mkfix quoted-heredoc-marker 'echo "<<EOF"' 'w="$(mktemp -d)"'
mkfix comment-heredoc-marker 'echo hi # <<EOF' 'w="$(mktemp -d)"'
mkfix hyphen-heredoc-delimiter 'cat <<EOF-X' 'text' 'EOF-X' 'w="$(mktemp -d)"'
mkfix alloc-in-while 'while true; do' '  w="$(mktemp -d)"' '  break' 'done' "trap 'rm -rf \"\$w\"' EXIT"
mkfix alloc-in-group '{' '  w="$(mktemp -d)"' '}' "trap 'rm -rf \"\$w\"' EXIT"
mkfix alloc-in-subshell '(' '  w="$(mktemp -d)"' ')' "trap 'rm -rf \"\$w\"' EXIT"
mkfix alloc-in-condition 'if w="$(mktemp -d)"; then :; fi' "trap 'rm -rf \"\$w\"' EXIT"
mkfix alloc-one-line-if 'if true; then w="$(mktemp -d)"; fi' "trap 'rm -rf \"\$w\"' EXIT"
mkfix alloc-and-chain 'w="$(mktemp -d)" && :' "trap 'rm -rf \"\$w\"' EXIT"
mkfix init-or-true "$SRC" 'pose_test_tmp_init || true' 'w="$(mktemp -d)"'
mkfix extra-signal-trap 'w="$(mktemp -d)"' "trap 'rm -rf \"\$w\"' EXIT" "trap 'exit 1' INT"
mkfix split-redefinition 'w="$(mktemp -d)"' 'cleanup() { rm -rf "$w"; }' 'trap cleanup EXIT' 'cleanup()' '{ :; }'
mkfix false-before-trap 'set -euo pipefail' 'w="$(mktemp -d)"' 'false' "trap 'rm -rf \"\$w\"' EXIT"
mkfix assertion-before-trap 'w="$(mktemp -d)"' '[ 1 = 2 ] || exit 1' "trap 'rm -rf \"\$w\"' EXIT"
mkfix go-build-before-trap 'w="$(mktemp -d)"' 'go build ./...' "trap 'rm -rf \"\$w\"' EXIT"
mkfix shift-expansion 'bit=1' 'x=$((1 << bit))' 'w="$(mktemp -d)"'
mkfix shift-quoted 'bit=1' 'x="$((1 << bit))"' 'w="$(mktemp -d)"'
mkfix shift-command 'bit=1' '((x = 1 << bit))' 'w="$(mktemp -d)"'
mkfix for-rebinds 'w="$(mktemp -d)"' 'for w in a b; do :; done' "trap 'rm -rf \"\$w\"' EXIT"
mkfix array-element-rebinds 'w="$(mktemp -d)"' 'w[0]=' "trap 'rm -rf \"\$w\"' EXIT"
mkfix rm-function-noop 'rm() { return 0; }' 'w="$(mktemp -d)"' "trap 'rm -rf \"\$w\"' EXIT"
mkfix read-clears-root "$SRC" 'pose_test_tmp_init' 'read POSE_TEST_TMP < /dev/null' 'w="$(mktemp -d)"'
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
# The same holds for a real script: a tracked copy whose mktemp line is repeated,
# or run in a loop, is rejected.
sed '/^work="\$(mktemp -d)"$/p' tests/release/compat.sh > "$fixtures/compat-rebound.sh.txt"
sed 's/^work="\$(mktemp -d)"$/for _i in 1 2; do work="$(mktemp -d)"; done/' tests/release/compat.sh > "$fixtures/compat-looped.sh.txt"
sed '/^BUILD_WORK="\$(mktemp -d)"$/a false' examples/demo/record.sh > "$fixtures/record-fallible.sh.txt"
for f in compat-rebound compat-looped; do
  cmp -s tests/release/compat.sh "$fixtures/$f.sh.txt" && fail "control: $f equals the original"
  [ -n "$(scan "$fixtures/$f.sh.txt")" ] || fail "the scan accepted a tracked script changed to leak ($f)"
done
cmp -s examples/demo/record.sh "$fixtures/record-fallible.sh.txt" && fail "control: record-fallible equals the original"
[ -n "$(scan "$fixtures/record-fallible.sh.txt")" ] || fail "the scan accepted a tracked script with a fallible command before its trap"
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

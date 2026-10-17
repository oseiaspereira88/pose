#!/usr/bin/env bash
# The test scripts remove the temporary files they create (spec
# pose-test-scripts-remove-their-temp-files).
#
# Three things are checked:
#   1. every script that calls mktemp uses the helper in its plain form or is
#      pinned by hash after a reviewer read its own EXIT trap;
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

# 1. Static: no script is added or changed that creates a temporary file it
# never removes. Shell cannot be proven to clean up by reading it, so this is
# a regression guard with two explicit rules, not an analysis of arbitrary shell:
#   - a script that calls mktemp and is not pinned must use the helper in the
#     plain form: tests/lib/tempdir.sh sourced on its own line, then
#     `pose_test_tmp_init` on its own line, both before the first mktemp;
#   - the scripts that clean up with an EXIT trap of their own are pinned by the
#     SHA-256 of the whole file in tests/hygiene/temp-cleanup.pins. A change to
#     one fails here until a reviewer has re-read its allocation and trap and
#     runs `bash tests/hygiene/temp-cleanup.sh --repin`.
PINS="tests/hygiene/temp-cleanup.pins"
mapfile -t scripts < <(git ls-files 'tests/*.sh' 'scripts/*.sh' 'examples/*.sh' 'install.sh' | sort | grep -v -e '^tests/lib/tempdir.sh$' -e '^tests/hygiene/temp-cleanup.sh$')
mktemp_scripts=()
for script in "${scripts[@]}"; do grep -q mktemp "$script" && mktemp_scripts+=("$script"); done
[ "${#mktemp_scripts[@]}" -gt 0 ] || fail "no script with mktemp was found: the scan itself is broken"

# The helper form, read from the lines of a script: <path> -> 0 when plain.
uses_helper() {
  awk '
    /^[[:space:]]*#/ { next }
    /^(source|\.)[[:space:]]+.*tempdir\.sh"?[[:space:]]*$/ { if (!src) src = NR; next }
    /^pose_test_tmp_init[[:space:]]*$/ { if (!init && src) init = NR; next }
    /mktemp/ { if (!mk) mk = NR }
    END { exit !(src && init && (!mk || mk > init)) }
  ' "$1"
}
pin_of() { awk -v f="$2" '$2 == f { print $1 }' "$1"; }
digest() { sha256sum "$1" | cut -d' ' -f1; }

if [ "${1:-}" = "--repin" ]; then
  : > "$PINS"
  for script in "${mktemp_scripts[@]}"; do
    uses_helper "$script" || printf '%s  %s\n' "$(digest "$script")" "$script" >> "$PINS"
  done
  echo "re-pinned $(wc -l < "$PINS") scripts in $PINS"; exit 0
fi

problems=""
for script in "${mktemp_scripts[@]}"; do
  pin="$(pin_of "$PINS" "$script")"
  if [ -n "$pin" ]; then
    [ "$pin" = "$(digest "$script")" ] || problems="$problems$script changed since it was pinned: re-read its mktemp and EXIT trap, then run bash tests/hygiene/temp-cleanup.sh --repin"$'\n'
  elif ! uses_helper "$script"; then
    problems="$problems$script calls mktemp without the helper: source tests/lib/tempdir.sh and call pose_test_tmp_init before the first mktemp"$'\n'
  fi
done
while read -r _ pinned; do
  printf '%s\n' "${mktemp_scripts[@]}" | grep -qx "$pinned" || problems="$problems$pinned is pinned but no longer calls mktemp"$'\n'
done < "$PINS"
[ -z "$problems" ] || fail "$problems"
ok "static: ${#mktemp_scripts[@]} scripts that call mktemp use the helper or are pinned"

# The rules must be able to fail.
fixtures="$SANDBOX/fixtures"; mkdir -p "$fixtures"
mkfix() { printf '%s\n' "${@:2}" > "$fixtures/$1.sh"; }
SRC='source tests/lib/tempdir.sh'
mkfix good "$SRC" 'pose_test_tmp_init' 'w="$(mktemp -d)"'
uses_helper "$fixtures/good.sh" || fail "the helper form was rejected"
negatives=0
reject() { negatives=$((negatives + 1)); ! uses_helper "$fixtures/$1.sh" || fail "the helper form accepted $1, which does not initialize the helper first"; }
mkfix no-helper 'w="$(mktemp -d)"' "trap 'rm -rf \"\$w\"' EXIT"; reject no-helper
mkfix init-after "$SRC" 'w="$(mktemp -d)"' 'pose_test_tmp_init'; reject init-after
mkfix init-without-source 'pose_test_tmp_init' 'w="$(mktemp -d)"'; reject init-without-source
mkfix commented-init "$SRC" '# pose_test_tmp_init' 'w="$(mktemp -d)"'; reject commented-init
mkfix indented-init "$SRC" '  pose_test_tmp_init' 'w="$(mktemp -d)"'; reject indented-init
mkfix init-or-true "$SRC" 'pose_test_tmp_init || true' 'w="$(mktemp -d)"'; reject init-or-true
# A pinned script that changes no longer matches its pin, and a new script is not pinned.
first_pinned="$(awk 'NR == 1 { print $2 }' "$PINS")"
[ -n "$first_pinned" ] || fail "no script is pinned"
cp "$first_pinned" "$fixtures/pinned-copy.sh"; printf '# changed\n' >> "$fixtures/pinned-copy.sh"
[ "$(digest "$fixtures/pinned-copy.sh")" != "$(pin_of "$PINS" "$first_pinned")" ] || fail "a changed pinned script still matches its pin"
negatives=$((negatives + 1))
ok "static: the helper form rejects $negatives shapes and a changed pinned script no longer matches its pin"

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
# A relative caller TMPDIR keeps its meaning after the script changes directory.
relative_case() { # <helper>; prints what is left in the relative TMPDIR
  local base="$SANDBOX/relative"; mkdir -p "$base/tmp"
  (cd "$base" && CDPATH=. TMPDIR=tmp bash -c ". '$1'; pose_test_tmp_init; cd /; mktemp -d >/dev/null" >/dev/null 2>&1 || true)
  ls -A "$base/tmp" | tr '\n' ' '
  chmod -R u+rwx "$base"; rm -rf "$base"
}
left="$(relative_case "$HELPER")"
[ -z "$left" ] || fail "helper (relative TMPDIR): left behind: $left"
sed 's|POSE_TEST_TMP="$(CDPATH="" cd -- "$(mktemp -d)" \&\& pwd -P)"|POSE_TEST_TMP="$(mktemp -d)"|' "$HELPER" > "$SANDBOX/tempdir-relative-old.sh"
cmp -s "$SANDBOX/tempdir-relative-old.sh" "$HELPER" && fail "control: the relative-path regression is identical to the helper"
[ -n "$(relative_case "$SANDBOX/tempdir-relative-old.sh")" ] || fail "control: the helper without the absolute path cleaned a relative TMPDIR, so the case proves nothing"
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

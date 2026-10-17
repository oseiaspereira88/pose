#!/usr/bin/env bash
# The test scripts remove the temporary files they create (spec
# pose-test-scripts-remove-their-temp-files).
#
# Three things are checked:
#   1. every script that calls mktemp either installs its own EXIT trap or
#      sources tests/lib/tempdir.sh and initializes it before its first mktemp;
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

# 1. Static: no script creates a temporary file it never removes.
checked=0
while IFS= read -r script; do
  case "$script" in tests/lib/tempdir.sh|tests/hygiene/temp-cleanup.sh) continue ;; esac
  grep -q 'mktemp' "$script" || continue
  checked=$((checked + 1))
  if grep -q 'pose_test_tmp_init' "$script"; then
    init_line="$(grep -n 'pose_test_tmp_init$' "$script" | head -1 | cut -d: -f1)"
    first_mktemp="$(grep -n 'mktemp' "$script" | head -1 | cut -d: -f1)"
    [ -n "$init_line" ] && [ "$init_line" -lt "$first_mktemp" ] ||
      fail "$script calls mktemp (line $first_mktemp) before pose_test_tmp_init"
  elif ! grep -Eq "trap .*(rm -rf|cleanup).*EXIT|trap cleanup EXIT" "$script"; then
    fail "$script creates temporary files with mktemp and never removes them: add pose_test_tmp_init (tests/lib/tempdir.sh) or an EXIT trap"
  fi
done < <(git ls-files 'tests/*.sh' 'scripts/*.sh' 'examples/*.sh' 'install.sh' | sort)
[ "$checked" -gt 0 ] || fail "no script with mktemp was found: the scan itself is broken"
ok "static: $checked scripts that call mktemp remove what they create"

# 2. The helper cleans up on every way a script can end.
outer() { local d="$SANDBOX/$1"; mkdir -p "$d"; printf '%s' "$d"; }
run_helper() { # <name> <body>; the body runs in a shell that sourced the helper
  local dir; dir="$(outer "$1")"
  TMPDIR="$dir" bash -c ". '$ROOT/tests/lib/tempdir.sh'; pose_test_tmp_init; $2" >/dev/null 2>&1 || true
  [ -z "$(ls -A "$dir")" ] || fail "helper ($1): left behind: $(ls -A "$dir" | tr '\n' ' ')"
}
files='mktemp -d >/dev/null; mktemp >/dev/null; d="$(mktemp -d)"; echo x > "$d/f"; chmod 500 "$d"'
run_helper success "set -e; $files"
run_helper assertion "set -e; $files; exit 1"
run_helper error-under-set-e "set -e; $files; false"
run_helper unset-variable "set -eu; $files; : \"\$NOT_SET\""
for sig in TERM INT; do
  dir="$(outer "signal-$sig")"
  TMPDIR="$dir" bash -c ". '$ROOT/tests/lib/tempdir.sh'; pose_test_tmp_init; $files; echo ready > '$dir.ready'; sleep 60 & wait" >/dev/null 2>&1 &
  pid=$!
  for _ in $(seq 1 100); do [ -f "$dir.ready" ] && break; sleep 0.05; done
  [ -f "$dir.ready" ] || fail "helper ($sig): the probe never started"
  kill -"$sig" "$pid"
  wait "$pid" 2>/dev/null || true
  pkill -P "$pid" 2>/dev/null || true
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

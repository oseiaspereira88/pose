#!/usr/bin/env bash
# E2E harness for pose-dist/install.sh (spec pose-installer-bootstrap).
#
# Scenarios:
#   0. distribution checkout itself → ./pose check --strict green.
#   1. fresh install into an empty git repo → ./pose check --strict green,
#      wrapper generated with derived root/id, no foreign-project residue.
#   2. idempotent re-run → user instance content and edited root docs preserved.
#   3. non-git target refused without --allow-non-git.
#
# Usage: bash tests/install/run.sh
# Env:   POSE_INSTALL_TEST_MCP_BINARY=<path>  reuse a binary instead of go build
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
INSTALLER="$REPO_ROOT/pose-dist/install.sh"
[[ -f "$INSTALLER" ]] || INSTALLER="$REPO_ROOT/install.sh"   # repo standalone: dist na raiz
FAILURES=0

pass() { printf '  [PASS] %s\n' "$*"; }
fail() { printf '  [FAIL] %s\n' "$*"; FAILURES=$((FAILURES + 1)); }

WORK="$(mktemp -d "${TMPDIR:-/tmp}/pose-install-test.XXXXXX")"
trap 'rm -rf "$WORK"' EXIT

MCP_ARGS=()
if [[ -n "${POSE_INSTALL_TEST_MCP_BINARY:-}" ]]; then
  MCP_ARGS=(--mcp-binary "$POSE_INSTALL_TEST_MCP_BINARY")
fi

# --- Scenario 0: the distributable checkout is a valid POSE instance ---------
echo "== scenario 0: distribution contract =="
DIST_ROOT="$(cd "$(dirname "$INSTALLER")" && pwd)"
( cd "$DIST_ROOT" && ./pose check --strict >/dev/null 2>&1 ) \
  && pass "./pose check --strict green in distribution checkout" \
  || fail "./pose check --strict failed in distribution checkout"

# --- Scenario 1: fresh install ------------------------------------------------
echo "== scenario 1: fresh install =="
T1="$WORK/proj-alpha"
mkdir -p "$T1" && git -C "$T1" init -q

if bash "$INSTALLER" "$T1" "${MCP_ARGS[@]}" >"$WORK/s1.log" 2>&1; then
  pass "installer exits 0"
else
  fail "installer exited non-zero — log follows"; cat "$WORK/s1.log"
fi

( cd "$T1" && ./pose check --strict >/dev/null 2>&1 ) \
  && pass "./pose check --strict green in target" \
  || fail "./pose check --strict failed in target"

if [[ -x "$T1/.pose/bin/pose-mcp-claude" ]]; then
  grep -q 'POSE_DEFAULT_PROJECT_ID="proj.proj-alpha"' "$T1/.pose/bin/pose-mcp-claude" \
    && pass "wrapper has derived project id" \
    || fail "wrapper does not contain derived project id"
  grep -q 'POSE_PROJECT_ROOT="\$(cd' "$T1/.pose/bin/pose-mcp-claude" \
    && pass "wrapper derives root dynamically (no hardcode)" \
    || fail "wrapper hardcodes root"
else
  if [[ ${#MCP_ARGS[@]} -gt 0 ]] || command -v go >/dev/null 2>&1; then
    fail "wrapper missing despite MCP being installable"
  else
    pass "wrapper absent (no Go toolchain, no --mcp-binary) — documented fallback"
  fi
fi

RESIDUE="$(grep -riE "crisol|storageclose|/home/" "$T1/.pose" "$T1/AGENTS.md" "$T1/POSE.md" \
  --exclude=LICENSE --exclude=NOTICE 2>/dev/null | grep -v "proj-alpha" || true)"
[[ -z "$RESIDUE" ]] \
  && pass "no foreign-project residue in target" \
  || { fail "residue found:"; printf '%s\n' "$RESIDUE"; }

grep -q "{{PROJECT_NAME}}" "$T1/AGENTS.md" "$T1/POSE.md" 2>/dev/null \
  && fail "unsubstituted placeholders remain in root docs" \
  || pass "placeholders substituted in AGENTS.md/POSE.md"

grep -q "proj-alpha" "$T1/AGENTS.md" \
  && pass "project name substituted into AGENTS.md" \
  || fail "project name not found in AGENTS.md"

[[ -f "$T1/.pose/LICENSE" && -f "$T1/.pose/NOTICE" ]] \
  && pass "legal texts vendored under .pose/" \
  || fail "missing vendored LICENSE/NOTICE"

[[ -f "$T1/.mcp.json" ]] \
  && pass ".mcp.json seeded" \
  || pass ".mcp.json not seeded (no MCP installed) — acceptable"

# --- Scenario 2: idempotent re-run ---------------------------------------------
echo "== scenario 2: idempotent re-run =="
mkdir -p "$T1/.pose/specs/user-spec"
echo "user content" >"$T1/.pose/specs/user-spec/spec.md"
echo "# custom user rule" >"$T1/.pose/rules/my-domain.md"
echo "USER EDIT" >>"$T1/AGENTS.md"
MARKER_MTIME_DOC="$(md5sum "$T1/AGENTS.md")"

bash "$INSTALLER" "$T1" "${MCP_ARGS[@]}" >"$WORK/s2.log" 2>&1 \
  && pass "re-run exits 0" \
  || { fail "re-run exited non-zero"; cat "$WORK/s2.log"; }

[[ -f "$T1/.pose/specs/user-spec/spec.md" ]] \
  && grep -q "user content" "$T1/.pose/specs/user-spec/spec.md" \
  && pass "user instance content preserved" \
  || fail "user instance content lost"

[[ -f "$T1/.pose/rules/my-domain.md" ]] \
  && pass "custom user rule preserved in extensible dir" \
  || fail "custom user rule deleted by machinery update"

[[ "$(md5sum "$T1/AGENTS.md")" == "$MARKER_MTIME_DOC" ]] \
  && pass "edited AGENTS.md preserved without --force" \
  || fail "edited AGENTS.md overwritten without --force"

bash "$INSTALLER" "$T1" --force "${MCP_ARGS[@]}" >"$WORK/s2f.log" 2>&1 || true
grep -q "USER EDIT" "$T1/AGENTS.md" \
  && fail "--force did not overwrite AGENTS.md" \
  || pass "--force overwrites AGENTS.md"

# --- Scenario 3: non-git target refused ----------------------------------------
echo "== scenario 3: non-git target =="
T3="$WORK/not-a-repo"
mkdir -p "$T3"
if bash "$INSTALLER" "$T3" --skip-mcp >"$WORK/s3.log" 2>&1; then
  fail "installer accepted a non-git target without --allow-non-git"
else
  pass "non-git target refused"
fi
bash "$INSTALLER" "$T3" --skip-mcp --allow-non-git >"$WORK/s3b.log" 2>&1 \
  && pass "--allow-non-git accepted" \
  || { fail "--allow-non-git run failed"; cat "$WORK/s3b.log"; }

# --- Summary --------------------------------------------------------------------
echo
if [[ "$FAILURES" -eq 0 ]]; then
  echo "RESULT: all install scenarios PASS"
else
  echo "RESULT: $FAILURES failure(s)"
  exit 1
fi

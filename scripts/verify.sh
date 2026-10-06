#!/usr/bin/env bash
# Reproduce the CI gates locally, from a fresh clone, with no maintainer
# access (spec pose-community-contribution-surfaces, R5).
#
# POSE gates its own repository strictly. A contributor who cannot reproduce
# those gates before opening a pull request finds out what failed from a CI log
# on someone else's schedule — which is the most common reason an outside
# contribution is abandoned. This script is the single documented command that
# closes that gap.
#
# It mirrors .github/workflows/ci.yml. When a gate is added there, add it here
# in the same change: a contributor script that silently lags CI is worse than
# none, because it produces false confidence.
#
# Usage:
#   bash scripts/verify.sh          # everything
#   bash scripts/verify.sh --fast   # skip the container and installer E2E steps
#
# Requires: Go (see pose-mcp/go.mod), git, jq, python3 and shellcheck.
# Full mode additionally requires Docker and authenticated gh read access.

set -euo pipefail

cd "$(dirname "$0")/.."

FAST=0
for arg in "$@"; do
  case "$arg" in
    --fast) FAST=1 ;;
    *) echo "usage: bash scripts/verify.sh [--fast]" >&2; exit 2 ;;
  esac
done

VERIFY_WORK="$(mktemp -d)"
trap 'rm -rf "$VERIFY_WORK"' EXIT
BIN="$VERIFY_WORK/pose"
FAILED=()

step() {
  local name="$1"; shift
  printf '\n\033[1m==> %s\033[0m\n' "$name"
  if "$@"; then
    printf '\033[32m    OK\033[0m\n'
  else
    printf '\033[31m    FAILED\033[0m\n'
    FAILED+=("$name")
  fi
}

printf '\033[1m==> Building the development binary\033[0m\n'
go -C pose-mcp build -o "$BIN" ./cmd/pose
"$BIN" version

step "Go tests"                 env POSE_RELEASE_HISTORY_AVAILABLE=true go -C pose-mcp test ./... -count=1
step "Shellcheck" shellcheck --severity=warning install.sh scripts/*.sh tests/*.sh tests/*/*.sh examples/demo/*.sh
step "Structural gate"          "$BIN" check --strict
step "Agent Skills conformance" "$BIN" skills-check --strict
step "Spec structure gate"       "$BIN" lint-spec --all
step "History gate"             "$BIN" history-check
step "Public claims gate"       "$BIN" public-claims --strict

step "Artifact-identity negative gate" bash tests/release/verify-negative.sh
step "Migration guides" bash tests/import/migration-guides.sh
step "First governed loop" bash tests/quickstart/first-governed-loop.sh
step "Install and upgrade journeys" bash tests/journeys/install-and-upgrade.sh
step "Demo fixture verification" bash examples/demo/record.sh --verify

if [ "$FAST" -eq 0 ]; then
  step "Installer E2E" bash tests/install/run.sh
  step "Container build" bash tests/release/container-build.sh
  step "Action runtime verification (online)" bash tests/release/action-runtime-verify.sh
else
  printf '\n\033[33m--fast: skipped installer E2E, container build and online action-runtime verification.\033[0m\n'
  printf '\033[33mCI still runs them; run without --fast before opening a pull request.\033[0m\n'
fi

printf '\n'
if [ ${#FAILED[@]} -eq 0 ]; then
  printf '\033[32mAll gates passed.\033[0m\n'
  exit 0
fi

printf '\033[31m%d gate(s) failed:\033[0m\n' "${#FAILED[@]}"
for f in "${FAILED[@]}"; do printf '  - %s\n' "$f"; done
printf '\nEvery gate runs to completion rather than stopping at the first\n'
printf 'failure, so one run tells you everything that needs fixing.\n'
exit 1

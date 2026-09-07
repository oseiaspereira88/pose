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
# Requires: Go (see pose-mcp/go.mod) and git. --fast additionally avoids
# needing Docker.

set -euo pipefail

cd "$(dirname "$0")/.."

FAST=0
for arg in "$@"; do
  case "$arg" in
    --fast) FAST=1 ;;
    *) echo "usage: bash scripts/verify.sh [--fast]" >&2; exit 2 ;;
  esac
done

BIN="$(mktemp -d)/pose"
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

step "Go tests"                 go -C pose-mcp test ./... -count=1
step "Structural gate"          "$BIN" check --strict
step "Agent Skills conformance" "$BIN" skills-check --strict
step "History gate"             "$BIN" history-check
step "Public claims gate"       "$BIN" public-claims --strict

if [ "$FAST" -eq 0 ]; then
  step "Installer E2E"          bash tests/install/run.sh
  step "Artifact-identity negative gate" bash tests/release/verify-negative.sh
else
  printf '\n\033[33m--fast: skipped the installer E2E and negative artifact gate.\033[0m\n'
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

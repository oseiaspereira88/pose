#!/usr/bin/env bash
# pre-commit provider E2E. Shell is only the test harness.
set -euo pipefail

command -v pre-commit >/dev/null 2>&1 || { echo "pre-commit 4.4+ required" >&2; exit 2; }
repo_root="$(git rev-parse --show-toplevel)"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT
mkdir -p "$work/bin" "$work/provider" "$work/project"
(cd "$repo_root/pose-mcp" && GOCACHE="${GOCACHE:-$work/go-cache}" go build -o "$work/bin/pose" ./cmd/pose)
export PATH="$work/bin:$PATH"

git -C "$work/provider" init -q
cp "$repo_root/.pre-commit-hooks.yaml" "$work/provider/.pre-commit-hooks.yaml"
git -C "$work/provider" add .
git -C "$work/provider" -c user.name=POSE -c user.email=pose@example.invalid commit -qm hooks
rev="$(git -C "$work/provider" rev-parse HEAD)"

git -C "$work/project" init -q
git -C "$work/project" config user.name POSE
git -C "$work/project" config user.email pose@example.invalid
pose install "$work/project" --skip-mcp >/dev/null
printf 'repos:\n  - repo: %s\n    rev: %s\n    hooks: [{id: pose-check}, {id: pose-lint-spec}, {id: pose-history-check}]\n' "$work/provider" "$rev" > "$work/project/.pre-commit-config.yaml"
git -C "$work/project" add .
git -C "$work/project" commit -qm init
(cd "$work/project" && pre-commit run --all-files)
echo "native pre-commit hooks: PASS"

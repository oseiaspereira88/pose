#!/usr/bin/env bash
# Native-only installer E2E. Shell is the test harness, never the POSE runtime.
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel)"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT
binary="$work/pose"
(cd "$repo_root/pose-mcp" && GOCACHE="${GOCACHE:-$work/go-cache}" go build -o "$binary" ./cmd/pose)

target="$work/project"
mkdir -p "$target"
git -C "$target" init -q
"$binary" install "$target" --skip-mcp >/dev/null
(cd "$target" && "$binary" check --strict >/dev/null)

test -f "$target/.pose/schema-version"
test -f "$target/AGENTS.md"
test ! -e "$target/.pose/scripts"
test ! -e "$target/pose"

mkdir -p "$target/.pose/specs/user-spec"
printf 'user content\n' > "$target/.pose/specs/user-spec/spec.md"
"$binary" install "$target" --skip-mcp >/dev/null
grep -q 'user content' "$target/.pose/specs/user-spec/spec.md"

non_git="$work/non-git"
mkdir -p "$non_git"
if "$binary" install "$non_git" --skip-mcp >/dev/null 2>&1; then
  echo "installer accepted non-git target" >&2
  exit 1
fi
"$binary" install "$non_git" --skip-mcp --allow-non-git >/dev/null

# Release bootstrap: install.sh must prefer a native binary beside itself and
# work without a source tree or Go on PATH.
bundle="$work/release-bundle"
bundle_target="$work/release-project"
mkdir -p "$bundle" "$bundle_target"
cp "$binary" "$bundle/pose"
cp "$repo_root/install.sh" "$bundle/install.sh"
git -C "$bundle_target" init -q
PATH="$(dirname "$(command -v git)")" bash "$bundle/install.sh" "$bundle_target" --skip-mcp >/dev/null
(cd "$bundle_target" && PATH="$(dirname "$(command -v git)")" "$bundle/pose" check --strict >/dev/null)
echo "native installer scenarios: PASS"

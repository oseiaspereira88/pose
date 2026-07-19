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

# Verified-download contract (spec pose-public-install-contract): archive named
# per the goreleaser template, checksum verified before the binary reaches
# PATH (R2), then doctor --json + check --strict on a clean host (R3).
sha_cmd="sha256sum"
command -v "$sha_cmd" >/dev/null 2>&1 || sha_cmd="shasum -a 256"
version_base="$("$binary" version | awk 'NR==1{sub(/-dev$/, "", $2); print $2}')"
asset="pose_${version_base}_$(go env GOOS)_$(go env GOARCH).tar.gz"
asset_dir="$work/assets"
mkdir -p "$asset_dir"
tar -C "$(dirname "$binary")" -czf "$asset_dir/$asset" pose
(cd "$asset_dir" && $sha_cmd "$asset" > checksums.txt && $sha_cmd --check checksums.txt >/dev/null)
extract="$work/extract"
mkdir -p "$extract"
tar -C "$extract" -xzf "$asset_dir/$asset" pose
verified_target="$work/verified-project"
mkdir -p "$verified_target"
git -C "$verified_target" init -q
clean_path="$extract:$(dirname "$(command -v git)")"
PATH="$clean_path" pose install "$verified_target" --skip-mcp >/dev/null
(cd "$verified_target" && PATH="$clean_path" pose doctor --json > "$work/doctor.json")
grep -q '"binary"' "$work/doctor.json"
(cd "$verified_target" && PATH="$clean_path" pose check --strict >/dev/null)
echo "native installer scenarios: PASS"

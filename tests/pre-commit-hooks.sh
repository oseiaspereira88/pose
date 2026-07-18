#!/usr/bin/env bash
set -euo pipefail

if ! command -v pre-commit >/dev/null 2>&1; then
  echo "pre-commit-hooks: pre-commit 4.4+ is required" >&2
  exit 2
fi

repo_root="$(git rev-parse --show-toplevel)"
fixture_root="$(mktemp -d)"
provider_root="$(mktemp -d)"
cleanup() { rm -rf "$fixture_root" "$provider_root"; }
trap cleanup EXIT

git -C "$provider_root" init -q
mkdir -p "$provider_root/pre-commit"
cp "$repo_root/.pre-commit-hooks.yaml" "$provider_root/.pre-commit-hooks.yaml"
cp "$repo_root/pre-commit/run-pose-hook" "$provider_root/pre-commit/run-pose-hook"
git -C "$provider_root" add .
git -C "$provider_root" -c user.name="POSE Test" -c user.email="pose-test@example.invalid" commit -qm "test: publish hooks"
provider_rev="$(git -C "$provider_root" rev-parse HEAD)"

git -C "$fixture_root" init -q
git -C "$fixture_root" config user.name "POSE Test"
git -C "$fixture_root" config user.email "pose-test@example.invalid"
bash "$repo_root/install.sh" "$fixture_root" --skip-mcp >/dev/null
cat >"$fixture_root/.pre-commit-config.yaml" <<EOF
repos:
  - repo: $provider_root
    rev: $provider_rev
    hooks:
      - id: pose-check
      - id: pose-lint-spec
      - id: pose-history-check
EOF
git -C "$fixture_root" add .
git -C "$fixture_root" commit -qm "test: initialize POSE fixture"

(cd "$fixture_root" && pre-commit run --all-files)

mkdir -p "$fixture_root/.pose/specs/broken-import"
printf '%s\n' '---' 'slug: broken-import' 'status: done' '---' >"$fixture_root/.pose/specs/broken-import/spec.md"
if (cd "$fixture_root" && pre-commit run pose-lint-spec --all-files); then
  echo "pre-commit-hooks: pose-lint-spec accepted an invalid spec" >&2
  exit 1
fi

echo "pre-commit-hooks: manifest and runtime checks passed"

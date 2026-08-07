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
# work without a source tree or Go on PATH. A poisoned `curl` shadows the real
# one, so any attempt to reach the provider fails the scenario instead of
# silently installing the previously published release (spec
# pose-installer-local-binary-precedence).
bundle="$work/release-bundle"
bundle_target="$work/release-project"
mkdir -p "$bundle" "$bundle_target"
cp "$binary" "$bundle/pose"
cp "$repo_root/install.sh" "$bundle/install.sh"
git -C "$bundle_target" init -q
offline_path="$work/offline-bin"
mkdir -p "$offline_path"
printf '#!/usr/bin/env bash\necho "installer reached the network" >&2\nexit 1\n' > "$offline_path/curl"
chmod +x "$offline_path/curl"
offline_env="$offline_path:$(dirname "$(command -v git)")"
PATH="$offline_env" bash "$bundle/install.sh" "$bundle_target" --skip-mcp >/dev/null
# `check --strict` here is the regression that matters: it includes the
# manual-parity gate, which is what a provider-downloaded (older) engine failed.
(cd "$bundle_target" && PATH="$offline_env" "$bundle/pose" check --strict >/dev/null)

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
# Provider-download branch (spec pose-release-cycle-debt-closure, R3). The only
# branch every public `curl | bash` user takes was covered by nothing: the
# bundle scenarios above all skip it. A curl stub serving from a local directory
# is the local origin — it exercises the script's real logic (release lookup,
# asset naming, extraction, install) without pointing the public installer at a
# configurable origin it would then carry forever.
origin="$work/origin"
mkdir -p "$origin"
printf '{"tag_name": "v9.9.9"}\n' > "$origin/latest.json"
tar -C "$(dirname "$binary")" -czf "$origin/pose_9.9.9_$(go env GOOS)_$(go env GOARCH).tar.gz" pose

stub_dir="$work/stub"
mkdir -p "$stub_dir"
cat > "$stub_dir/curl" <<'STUB'
#!/usr/bin/env bash
# Serves the release API and asset downloads from $POSE_TEST_ORIGIN.
out=""; url=""
while [[ $# -gt 0 ]]; do
  case "$1" in
    -o) out="$2"; shift 2 ;;
    -*) shift ;;
    *) url="$1"; shift ;;
  esac
done
if [[ "$url" == *"/releases/latest" ]]; then
  cat "$POSE_TEST_ORIGIN/latest.json"; exit 0
fi
asset="${url##*/}"
[[ -f "$POSE_TEST_ORIGIN/$asset" ]] || exit 22
if [[ -n "$out" ]]; then cp "$POSE_TEST_ORIGIN/$asset" "$out"; else cat "$POSE_TEST_ORIGIN/$asset"; fi
STUB
chmod +x "$stub_dir/curl"

download_target="$work/download-project"
mkdir -p "$download_target"
git -C "$download_target" init -q
download_home="$work/download-home"
mkdir -p "$download_home"
# install.sh alone in its directory, so the bundle branch cannot be taken.
solo="$work/solo"
mkdir -p "$solo"
cp "$repo_root/install.sh" "$solo/install.sh"
POSE_TEST_ORIGIN="$origin" HOME="$download_home" PATH="$stub_dir:$PATH" \
  bash "$solo/install.sh" "$download_target" --skip-mcp >/dev/null
test -x "$download_home/.local/bin/pose"
test -f "$download_target/.pose/schema-version"

# A malformed asset must fail the install rather than leave a broken binary on
# PATH: truncating the archive is the closest thing to a corrupted download.
bad_origin="$work/bad-origin"
mkdir -p "$bad_origin"
cp "$origin/latest.json" "$bad_origin/latest.json"
head -c 64 "$origin/pose_9.9.9_$(go env GOOS)_$(go env GOARCH).tar.gz" \
  > "$bad_origin/pose_9.9.9_$(go env GOOS)_$(go env GOARCH).tar.gz"
bad_target="$work/bad-project"
mkdir -p "$bad_target"
git -C "$bad_target" init -q
bad_home="$work/bad-home"
mkdir -p "$bad_home"
if POSE_TEST_ORIGIN="$bad_origin" HOME="$bad_home" PATH="$stub_dir:$PATH" \
     bash "$solo/install.sh" "$bad_target" --skip-mcp >/dev/null 2>&1; then
  echo "installer accepted a truncated asset" >&2
  exit 1
fi

echo "native installer scenarios: PASS"

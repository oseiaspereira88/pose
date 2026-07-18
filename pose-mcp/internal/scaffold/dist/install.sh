#!/usr/bin/env bash
# Bootstrap for release bundles and source checkouts. It only locates or builds
# the native Go CLI; it is not part of the POSE runtime command engine.
set -euo pipefail

if [[ $# -lt 1 ]]; then
  echo "Usage: bash install.sh <target-dir> [pose install options]" >&2
  exit 2
fi

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if [[ -x "$script_dir/pose" ]]; then
  exec "$script_dir/pose" install "$@"
fi
if command -v pose >/dev/null 2>&1; then
  exec pose install "$@"
fi
if [[ -f "$script_dir/pose-mcp/go.mod" ]] && command -v go >/dev/null 2>&1; then
  cd "$script_dir/pose-mcp"
  exec go run ./cmd/pose install "$@"
fi

echo "install.sh: place the released pose binary beside this script or on PATH; source checkouts may use Go." >&2
exit 1

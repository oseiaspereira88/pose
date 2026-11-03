#!/usr/bin/env bash
# Benchmark committed input in disposable worktrees. Never restore operator files.
# Usage: scripts/bench-governance.sh [runs] [pose-binary] [additional Python flags]
set -euo pipefail

runs="${1:-3}"
pose_bin="${2:-pose}"
if (( $# >= 2 )); then shift 2; else shift "$#"; fi
exec python3 "$(dirname "${BASH_SOURCE[0]}")/bench_governance.py" --runs "$runs" --binary "$pose_bin" "$@"

#!/usr/bin/env bash
# Container build gate (spec pose-container-build-gate).
#
# pose-mcp/Dockerfile and mcp-enforce/Dockerfile are delivery artifacts: the
# harne8 monorepo composes both in production from this repository. Nothing here
# built either one, so a broken Dockerfile surfaced when someone brought up the
# composition stack, in a different repository.
#
# The two build contexts differ and are not interchangeable:
#   pose-mcp     → repository root, because it consumes the sibling mcp-enforce
#                  module through `replace => ../mcp-enforce`
#   mcp-enforce  → its own directory, because it copies go.mod from the context
#                  root
# Swapping them produces `"/go.mod": not found`, which reads like a broken
# Dockerfile rather than a wrong invocation. These are the same contexts
# docker-compose.prod.yaml uses.
#
# Building is not enough: an image that builds and cannot start is the failure
# that reaches a compose stack as a crash-loop, so each entrypoint is run and
# required to announce itself.
#
# Usage: bash tests/release/container-build.sh
# Requires: docker.
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel)"
cd "$repo_root"

fail=0
containers=()
images=()
cleanup() {
  for c in "${containers[@]:-}"; do [ -n "$c" ] && docker rm -f "$c" >/dev/null 2>&1 || true; done
  for i in "${images[@]:-}"; do [ -n "$i" ] && docker rmi -f "$i" >/dev/null 2>&1 || true; done
}
trap cleanup EXIT

# build_and_smoke <label> <dockerfile> <context> <expected-log-substring>
build_and_smoke() {
  local label="$1" dockerfile="$2" context="$3" expect="$4"
  local tag="pose-buildgate-$label"

  echo "container-build: building $label (context: $context)"
  if ! docker build -q -f "$dockerfile" -t "$tag" "$context" >/dev/null 2>&1; then
    echo "container-build: FAIL: $label does not build from $context" >&2
    # Re-run without -q so the reason reaches the log.
    docker build -f "$dockerfile" -t "$tag" "$context" 2>&1 | tail -20 >&2 || true
    fail=1
    return
  fi
  images+=("$tag")

  local cid
  if ! cid="$(docker run -d "$tag" 2>&1)"; then
    echo "container-build: FAIL: $label built but could not be started: $cid" >&2
    fail=1
    return
  fi
  containers+=("$cid")

  # Poll rather than sleep a fixed amount: a healthy start is sub-second, and a
  # crash-loop should be reported quickly rather than after a worst-case wait.
  local waited=0 logs=""
  while [ "$waited" -lt 20 ]; do
    logs="$(docker logs "$cid" 2>&1 || true)"
    if printf '%s' "$logs" | grep -qF "$expect"; then
      break
    fi
    if [ "$(docker inspect -f '{{.State.Status}}' "$cid" 2>/dev/null)" = "exited" ]; then
      echo "container-build: FAIL: $label exited instead of starting" >&2
      printf '%s\n' "$logs" | tail -20 >&2
      fail=1
      return
    fi
    sleep 1
    waited=$((waited + 1))
  done

  if ! printf '%s' "$logs" | grep -qF "$expect"; then
    echo "container-build: FAIL: $label did not announce itself within ${waited}s (expected \"$expect\")" >&2
    printf '%s\n' "$logs" | tail -20 >&2
    fail=1
    return
  fi

  echo "container-build: OK: $label builds and starts"
}

build_and_smoke "pose-mcp" "pose-mcp/Dockerfile" "." "pose-mcp listening addr="
build_and_smoke "mcp-enforce" "mcp-enforce/Dockerfile" "mcp-enforce" "mcp-enforce-sidecar listening addr="

if [ "$fail" -eq 0 ]; then
  echo "container-build: both images build and start"
else
  echo "container-build: a delivery image is broken — the composition stack would fail on it" >&2
fi
exit "$fail"

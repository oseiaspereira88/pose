#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
cd "$ROOT/pose-mcp"
export LC_ALL=C

mode="${1:-all}"
if [ "$#" -gt 1 ]; then
  mode=invalid
fi
case "$mode" in
  all|other|pose-0|pose-1|inventory) ;;
  *) printf '%s\n' 'usage: test-engine-go.sh [all|other|pose-0|pose-1|inventory]' >&2; exit 2 ;;
esac

run_other_packages() {
  local heavy listed package
  local -a other=()
  heavy="$(go list ./internal/pose)"
  listed="$(go list ./...)"
  while IFS= read -r package; do
    if [ "$package" != "$heavy" ]; then
      other+=("$package")
    fi
  done <<< "$listed"
  if [ "${#other[@]}" -eq 0 ] || [ "${#other[@]}" -eq "$(printf '%s\n' "$listed" | wc -l)" ]; then
    printf '%s\n' 'test-engine-go: the Go package inventory did not contain the expected package split' >&2
    return 1
  fi
  go test -timeout 10m -count=1 "${other[@]}"
}

list_names() {
  go test -list "$1" ./internal/pose | sed -n -E '/^(Test|Fuzz|Example)[^[:space:]]*$/p' | sort
}

prepare_pose_shards() {
  local all selected0 selected1 combined test_name
  local i=0
  local -a first=() second=()
  all="$(list_names '^(Test|Fuzz|Example)')"
  while IFS= read -r test_name; do
    if [ "$((i % 2))" -eq 0 ]; then
      first+=("$test_name")
    else
      second+=("$test_name")
    fi
    i=$((i + 1))
  done <<< "$all"
  if [ -z "$all" ] || [ "${#second[@]}" -eq 0 ]; then
    printf '%s\n' 'test-engine-go: expected two nonempty Go test shards' >&2
    return 1
  fi
  pose_pattern_0="^($(IFS='|'; printf '%s' "${first[*]}"))$"
  pose_pattern_1="^($(IFS='|'; printf '%s' "${second[*]}"))$"
  selected0="$(list_names "$pose_pattern_0")"
  selected1="$(list_names "$pose_pattern_1")"
  combined="$(printf '%s\n%s\n' "$selected0" "$selected1" | sort)"
  if [ "$combined" != "$all" ]; then
    printf '%s\n' 'test-engine-go: the Go-selected shards overlap or omit a test' >&2
    return 1
  fi
  printf 'internal/pose inventory: %d + %d = %d tests, fuzz targets and examples\n'     "${#first[@]}" "${#second[@]}" "$i"
}

if [ "$mode" = other ]; then
  run_other_packages
  exit 0
fi

prepare_pose_shards
case "$mode" in
  all)
    run_other_packages
    go test -timeout 10m -count=1 -run "$pose_pattern_0" ./internal/pose
    go test -timeout 10m -count=1 -run "$pose_pattern_1" ./internal/pose
    ;;
  pose-0) go test -timeout 10m -count=1 -run "$pose_pattern_0" ./internal/pose ;;
  pose-1) go test -timeout 10m -count=1 -run "$pose_pattern_1" ./internal/pose ;;
  inventory) ;;
esac

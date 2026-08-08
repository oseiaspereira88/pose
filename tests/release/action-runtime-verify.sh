#!/usr/bin/env bash
# Online half of the action-runtime contract (spec pose-action-runtime-currency-gate).
#
# The offline check reads `.github/action-runtimes.json`, which is a second
# source of truth whose only failure mode is drifting from the real action.yml.
# This resolves each pinned action at its pinned ref and requires the recorded
# runtime to be what the action actually declares — without it, the offline
# check is bookkeeping that agrees with itself.
#
# Usage: bash tests/release/action-runtime-verify.sh
# Requires: gh (authenticated read), jq, python3 for base64 decoding.
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel)"
manifest="$repo_root/.github/action-runtimes.json"

if [ ! -f "$manifest" ]; then
  echo "action-runtime-verify: $manifest is missing" >&2
  exit 1
fi

fail=0
checked=0

while IFS=$'\t' read -r action ref recorded; do
  [ -n "$action" ] || continue
  checked=$((checked + 1))

  base="$(printf '%s' "$action" | cut -d/ -f1,2)"
  sub="$(printf '%s' "$action" | cut -d/ -f3-)"
  if [ -n "$sub" ]; then path="$sub/action.yml"; else path="action.yml"; fi

  body=""
  for candidate in "$path" "${path%.yml}.yaml"; do
    if body="$(gh api "repos/$base/contents/$candidate?ref=$ref" --jq '.content' 2>/dev/null)"; then
      [ -n "$body" ] && break
    fi
    body=""
  done

  if [ -z "$body" ]; then
    echo "action-runtime-verify: FAIL: cannot read action.yml for $action at $ref" >&2
    fail=1
    continue
  fi

  actual="$(printf '%s' "$body" | python3 -c '
import sys, base64, re
raw = base64.b64decode(sys.stdin.read()).decode("utf-8", "replace")
m = re.search(r"^\s*using:\s*[\x27\"]?([\w.-]+)", raw, re.M)
print(m.group(1) if m else "unknown")')"

  if [ "$actual" != "$recorded" ]; then
    echo "action-runtime-verify: FAIL: $action declares '$actual' at its pinned ref, recorded as '$recorded'" >&2
    fail=1
  else
    echo "action-runtime-verify: OK: $action -> $actual"
  fi
done < <(jq -r '.runtimes | to_entries[] | [.key, .value.ref, .value.using] | @tsv' "$manifest")

if [ "$checked" -eq 0 ]; then
  echo "action-runtime-verify: no actions were checked — the manifest is empty or unreadable" >&2
  exit 1
fi

if [ "$fail" -eq 0 ]; then
  echo "action-runtime-verify: $checked recorded runtime(s) match the actions themselves"
else
  echo "action-runtime-verify: the runtime record disagrees with reality" >&2
fi
exit "$fail"

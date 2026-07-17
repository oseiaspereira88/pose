#!/usr/bin/env bash
set -euo pipefail

# shellcheck source=pose-lib.sh
source "$(dirname "${BASH_SOURCE[0]}")/pose-lib.sh"

REPO_ROOT="$(pose_repo_root)"
ROOT="$REPO_ROOT/.pose/knowledge"
ARCHIVE="$ROOT/archive"
MODE="${1:-list-expired}"
APPLY=false
NOW_EPOCH="$(date -u +%s)"

if [[ "${2:-}" == "--apply" ]]; then
  APPLY=true
fi
if [[ "${2:-}" == "--dry-run" || "${3:-}" == "--dry-run" ]]; then
  APPLY=false
fi

mkdir -p "$ARCHIVE"

extract_expires() {
  local file="$1"
  awk -F': ' '/^expires_at:/ {print $2; exit}' "$file" | tr -d '"'
}

is_expired() {
  local expires="$1"
  [[ -z "$expires" ]] && return 1
  local exp_epoch
  exp_epoch="$(date -u -d "$expires" +%s 2>/dev/null || true)"
  [[ -z "$exp_epoch" ]] && return 1
  [[ "$exp_epoch" -lt "$NOW_EPOCH" ]]
}

list_expired_files() {
  find "$ROOT" -maxdepth 1 -type f -name '*.md' | while read -r f; do
    local expires
    expires="$(extract_expires "$f")"
    if is_expired "$expires"; then
      printf '%s|%s\n' "$f" "$expires"
    fi
  done
}

case "$MODE" in
  list-expired)
    list_expired_files || true
    ;;
  archive-expired)
    list_expired_files | while IFS='|' read -r file expires; do
      target="$ARCHIVE/$(basename "$file")"
      if $APPLY; then
        mv "$file" "$target"
        echo "ARCHIVED|$file|$target|$expires"
      else
        echo "DRY-RUN ARCHIVE|$file|$target|$expires"
      fi
    done
    ;;
  purge-archived)
    find "$ARCHIVE" -maxdepth 1 -type f -name '*.md' | while read -r f; do
      expires="$(extract_expires "$f")"
      exp_epoch="$(date -u -d "$expires" +%s 2>/dev/null || true)"
      [[ -z "$exp_epoch" ]] && continue
      cutoff=$(( exp_epoch + 180*24*3600 ))
      if [[ "$cutoff" -lt "$NOW_EPOCH" ]]; then
        if $APPLY; then
          rm -f "$f"
          echo "PURGED|$f|$expires"
        else
          echo "DRY-RUN PURGE|$f|$expires"
        fi
      fi
    done
    ;;
  *)
    echo "Uso: $0 [list-expired|archive-expired|purge-archived] [--dry-run|--apply]" >&2
    exit 1
    ;;
esac

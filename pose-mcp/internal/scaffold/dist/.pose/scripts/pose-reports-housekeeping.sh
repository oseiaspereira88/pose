#!/usr/bin/env bash
set -euo pipefail

# shellcheck source=pose-lib.sh
source "$(dirname "${BASH_SOURCE[0]}")/pose-lib.sh"

usage() {
  cat <<'USAGE'
Uso: pose-reports-housekeeping.sh <comando> [opções]

Comandos:
  list-stale [--older-than N]   Lista relatórios .md mais antigos que N dias
                                (default: 120).
  archive-stale [--older-than N] [--dry-run|--apply]
                                Move relatórios stale para .pose/reports/archive/.
                                Default: --dry-run.
  purge-archived [--older-than N] [--dry-run|--apply]
                                Remove arquivados mais antigos que N dias
                                (default: 365). Default: --dry-run.

Observações:
- Mantém histórico em .pose/reports/history/ intocado (necessário para
  comparações temporais por task/spec via ./pose report).
- README.md e arquivos sem padrão de data (YYYY-MM-DD-*) são ignorados.
USAGE
}

CMD="${1:-}"
if [[ -z "$CMD" || "$CMD" == "-h" || "$CMD" == "--help" ]]; then
  usage
  [[ -z "$CMD" ]] && exit 2 || exit 0
fi
shift

OLDER_THAN=""
APPLY=false

while [[ $# -gt 0 ]]; do
  case "$1" in
    --older-than)
      [[ $# -ge 2 && "${2:-}" =~ ^[0-9]+$ ]] || { echo "Erro: --older-than exige inteiro > 0" >&2; exit 2; }
      OLDER_THAN="$2"; shift 2 ;;
    --apply) APPLY=true; shift ;;
    --dry-run) APPLY=false; shift ;;
    *) echo "Erro: argumento inválido: $1" >&2; usage; exit 2 ;;
  esac
done

REPO_ROOT="$(pose_repo_root)"
REPORTS_DIR="$REPO_ROOT/.pose/reports"
ARCHIVE_DIR="$REPORTS_DIR/archive"
mkdir -p "$ARCHIVE_DIR"

NOW_EPOCH="$(date -u +%s)"

# Extrai a data ISO embutida no nome do arquivo: YYYY-MM-DD-...md
extract_date_from_name() {
  local name="$1"
  if [[ "$name" =~ ^([0-9]{4}-[0-9]{2}-[0-9]{2})- ]]; then
    printf '%s' "${BASH_REMATCH[1]}"
  fi
}

date_to_epoch() {
  date -u -d "$1" +%s 2>/dev/null || true
}

list_stale_files() {
  local cutoff_days="$1"
  local cutoff_epoch=$(( NOW_EPOCH - cutoff_days * 24 * 3600 ))
  local f date_str epoch
  while IFS= read -r f; do
    [[ -f "$f" ]] || continue
    date_str="$(extract_date_from_name "$(basename "$f")")"
    [[ -n "$date_str" ]] || continue
    epoch="$(date_to_epoch "$date_str")"
    [[ -n "$epoch" ]] || continue
    if (( epoch < cutoff_epoch )); then
      printf '%s|%s\n' "$f" "$date_str"
    fi
  done < <(find "$REPORTS_DIR" -maxdepth 1 -type f -name '*.md' ! -name 'README.md')
}

list_archived_files() {
  local cutoff_days="$1"
  local cutoff_epoch=$(( NOW_EPOCH - cutoff_days * 24 * 3600 ))
  local f date_str epoch
  while IFS= read -r f; do
    [[ -f "$f" ]] || continue
    date_str="$(extract_date_from_name "$(basename "$f")")"
    [[ -n "$date_str" ]] || continue
    epoch="$(date_to_epoch "$date_str")"
    [[ -n "$epoch" ]] || continue
    if (( epoch < cutoff_epoch )); then
      printf '%s|%s\n' "$f" "$date_str"
    fi
  done < <(find "$ARCHIVE_DIR" -maxdepth 1 -type f -name '*.md')
}

case "$CMD" in
  list-stale)
    list_stale_files "${OLDER_THAN:-120}" || true
    ;;
  archive-stale)
    list_stale_files "${OLDER_THAN:-120}" | while IFS='|' read -r file date_str; do
      target="$ARCHIVE_DIR/$(basename "$file")"
      if $APPLY; then
        mv "$file" "$target"
        echo "ARCHIVED|$file|$target|$date_str"
      else
        echo "DRY-RUN ARCHIVE|$file|$target|$date_str"
      fi
    done
    ;;
  purge-archived)
    list_archived_files "${OLDER_THAN:-365}" | while IFS='|' read -r file date_str; do
      if $APPLY; then
        rm -f "$file"
        echo "PURGED|$file|$date_str"
      else
        echo "DRY-RUN PURGE|$file|$date_str"
      fi
    done
    ;;
  *)
    echo "Erro: comando desconhecido: $CMD" >&2
    usage
    exit 2
    ;;
esac

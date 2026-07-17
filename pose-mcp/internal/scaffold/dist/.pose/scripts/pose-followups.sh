#!/usr/bin/env bash
set -euo pipefail

# shellcheck source=pose-lib.sh
source "$(dirname "${BASH_SOURCE[0]}")/pose-lib.sh"

usage() {
  cat <<'USAGE'
Uso: pose-followups.sh [--open|--all] [--json] [--similarity N]

Agrega os follow-ups de Final Report > Follow-ups de todas as specs em
.pose/specs/ e mostra o backlog vivo + candidatos a near-duplicate (follow-ups
de specs diferentes com similaridade léxica acima do limiar). Ferramenta de
descoberta determinística — sempre exit 0. O gate de obrigação (toda spec done
com follow-ups triados) vive em ./pose lint-spec; o julgamento semântico e a
confirmação de reaproveitamento vivem na skill pose-spec-closeout.

Opções:
  --open           Lista só follow-ups [open] ou sem disposição (default).
  --all            Lista todos os follow-ups com spec + disposição.
  --json           Saída machine-readable.
  --similarity N   Limiar 0..100 de similaridade léxica (default 60).
  -h, --help

Disposições: [open] [spawned: <slug>] [covered: <slug>] [duplicate: <slug>]
             [done] [wont-do: <motivo>]
USAGE
}

EXTRA=()
while [[ $# -gt 0 ]]; do
  case "$1" in
    --open) EXTRA+=(--open); shift ;;
    --all) EXTRA+=(--all); shift ;;
    --json) EXTRA+=(--json); shift ;;
    --similarity)
      [[ $# -ge 2 && "${2:-}" =~ ^[0-9]+$ ]] || { echo "Erro: --similarity exige inteiro 0..100." >&2; exit 2; }
      EXTRA+=(--similarity "$2"); shift 2 ;;
    -h|--help) usage; exit 0 ;;
    *) echo "Erro: opção desconhecida: $1" >&2; usage; exit 2 ;;
  esac
done

REPO_ROOT="$(pose_repo_root)"
SPECS_DIR="$REPO_ROOT/.pose/specs"
AGGREGATOR="$REPO_ROOT/.pose/scripts/pose-followups.py"

if [[ ! -x "$AGGREGATOR" ]]; then
  echo "Erro: agregador ausente ou sem permissão de execução: $AGGREGATOR" >&2
  exit 2
fi

exec python3 "$AGGREGATOR" --specs-dir "$SPECS_DIR" "${EXTRA[@]}"

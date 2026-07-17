#!/usr/bin/env bash
set -euo pipefail

# shellcheck source=pose-lib.sh
source "$(dirname "${BASH_SOURCE[0]}")/pose-lib.sh"

usage() {
  cat <<'USAGE'
Uso: pose-recurrence-check.sh [--strict|--tolerant] [--window-days N] [--threshold T] [--include-pass]

Analisa .pose/reports/history/*.jsonl em busca de task_slugs com ≥ T ocorrências
na janela de N dias. Em strict, falha (exit 1) quando flagged > 0; em tolerant,
apenas reporta (exit 0).

Por padrão ignora outcome=pass — recorrência problemática é falha repetida.
Use --include-pass para auditar também tarefas estáveis com alta frequência.

Próximo passo após um flag: escalar via .pose/workflows/recurrence-escalation.md.
USAGE
}

MODE="strict"
WINDOW_DAYS=14
THRESHOLD=3
INCLUDE_PASS=false

while [[ $# -gt 0 ]]; do
  case "$1" in
    --strict) MODE="strict"; shift ;;
    --tolerant) MODE="tolerant"; shift ;;
    --window-days)
      [[ $# -ge 2 && "${2:-}" =~ ^[0-9]+$ ]] || { echo "Erro: --window-days exige inteiro > 0" >&2; exit 2; }
      WINDOW_DAYS="$2"; shift 2 ;;
    --threshold)
      [[ $# -ge 2 && "${2:-}" =~ ^[0-9]+$ ]] || { echo "Erro: --threshold exige inteiro > 0" >&2; exit 2; }
      THRESHOLD="$2"; shift 2 ;;
    --include-pass) INCLUDE_PASS=true; shift ;;
    -h|--help) usage; exit 0 ;;
    *) echo "Erro: argumento inválido: $1" >&2; usage; exit 2 ;;
  esac
done

REPO_ROOT="$(pose_repo_root)"
HISTORY_DIR="$REPO_ROOT/.pose/reports/history"
DETECTOR="$REPO_ROOT/.pose/scripts/pose-recurrence-detect.py"

if [[ ! -d "$HISTORY_DIR" ]]; then
  echo "Erro: history ausente: $HISTORY_DIR" >&2
  exit 2
fi
if [[ ! -x "$DETECTOR" ]]; then
  echo "Erro: detector ausente ou sem permissão de execução: $DETECTOR" >&2
  exit 2
fi

extra_args=()
$INCLUDE_PASS && extra_args+=(--include-pass)

set +e
python3 "$DETECTOR" \
  --history-dir "$HISTORY_DIR" \
  --window-days "$WINDOW_DAYS" \
  --threshold "$THRESHOLD" \
  "${extra_args[@]}"
detect_exit=$?
set -e

if (( detect_exit == 0 )); then
  echo "Resultado: SUCESSO (nenhuma chave acima do threshold)"
  exit 0
fi
if (( detect_exit == 1 )); then
  echo "Resultado: FALHA (chaves recorrentes detectadas — consulte .pose/workflows/recurrence-escalation.md)"
  if [[ "$MODE" == "strict" ]]; then
    exit 1
  fi
  echo "Modo tolerant: registrar follow-up de escalação."
  echo "Resultado: FALHA_TOLERADA"
  exit 0
fi

echo "Erro: detector retornou exit $detect_exit" >&2
exit "$detect_exit"

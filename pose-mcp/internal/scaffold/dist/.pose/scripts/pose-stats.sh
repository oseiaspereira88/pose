#!/usr/bin/env bash
set -euo pipefail

# shellcheck source=pose-lib.sh
source "$(dirname "${BASH_SOURCE[0]}")/pose-lib.sh"

usage() {
  cat <<'USAGE'
Uso: pose-stats.sh [outcomes|workflows|tasks|contexts] [--since-days N] [--json]

Agrega outcomes do .pose/reports/history/*.jsonl. Útil para promover checks
optional → required por taxa de sucesso real, identificar workflows instáveis,
e comparar contextos (ci vs manual vs auto-validate).

Subcomandos:
  outcomes [--by workflow|task|context]   agrupamento explícito (default: workflow)
  workflows                                atalho de --by workflow
  tasks                                    atalho de --by task
  contexts                                 atalho de --by context

Opções:
  --since-days N     Considera apenas registros dos últimos N dias (default: 0 = todos)
  --json             Saída em JSON
  -h, --help         Mostra esta ajuda

Exemplos:
  ./pose stats workflows
  ./pose stats tasks --since-days 30
  ./pose stats outcomes --by context --json
USAGE
}

SUB=""
GROUP_BY="workflow"
SINCE_DAYS=0
EMIT_JSON=false

while [[ $# -gt 0 ]]; do
  case "$1" in
    -h|--help) usage; exit 0 ;;
    --json) EMIT_JSON=true; shift ;;
    --since-days)
      [[ $# -ge 2 && "${2:-}" =~ ^[0-9]+$ ]] || { echo "Erro: --since-days exige inteiro >= 0" >&2; exit 2; }
      SINCE_DAYS="$2"; shift 2 ;;
    --by)
      [[ $# -ge 2 && -n "${2:-}" && "${2:-}" != --* ]] || { echo "Erro: --by exige um valor" >&2; exit 2; }
      case "$2" in
        workflow|task|context) GROUP_BY="$2" ;;
        *) echo "Erro: --by inválido: $2 (use workflow|task|context)" >&2; exit 2 ;;
      esac
      shift 2 ;;
    outcomes|workflows|tasks|contexts)
      if [[ -n "$SUB" ]]; then
        echo "Erro: subcomando duplicado: $1" >&2; exit 2
      fi
      SUB="$1"
      shift ;;
    --*) echo "Erro: opção desconhecida: $1" >&2; usage; exit 2 ;;
    *) echo "Erro: argumento inválido: $1" >&2; usage; exit 2 ;;
  esac
done

case "$SUB" in
  ""|outcomes) ;; # GROUP_BY já default
  workflows) GROUP_BY="workflow" ;;
  tasks) GROUP_BY="task" ;;
  contexts) GROUP_BY="context" ;;
esac

REPO_ROOT="$(pose_repo_root)"
HISTORY_DIR="$REPO_ROOT/.pose/reports/history"
STATS_SCRIPT="$REPO_ROOT/.pose/scripts/pose-stats.py"

if [[ ! -d "$HISTORY_DIR" ]]; then
  echo "Erro: history ausente: $HISTORY_DIR" >&2
  exit 2
fi
if [[ ! -x "$STATS_SCRIPT" ]]; then
  echo "Erro: helper ausente ou sem permissão de execução: $STATS_SCRIPT" >&2
  exit 2
fi

extra=()
$EMIT_JSON && extra+=(--json)

python3 "$STATS_SCRIPT" \
  --history-dir "$HISTORY_DIR" \
  --by "$GROUP_BY" \
  --since-days "$SINCE_DAYS" \
  "${extra[@]}"

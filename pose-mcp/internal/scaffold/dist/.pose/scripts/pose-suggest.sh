#!/usr/bin/env bash
set -euo pipefail

# shellcheck source=pose-lib.sh
source "$(dirname "${BASH_SOURCE[0]}")/pose-lib.sh"

usage() {
  cat <<'USAGE'
Uso: pose-suggest.sh [<tipo-de-tarefa>] [--domain <d>] [--path <p>] [--json]

Sugere a trilha canônica (workflow + skill + rules + spec/ADR + knowledge)
para um tipo de tarefa, lendo .pose/indexes/task-map.json.

Sem <tipo>, lista todos os tipos disponíveis.

Opções:
  --domain <d>   Aplica rules adicionais do domínio (ex.: frontend, backend-go, k8s).
  --path <p>     Caminho dentro do repo; infere o domínio via heurísticas
                 (k8s/, charts/) e via .pose/indexes/repo-map.json (language →
                 frontend/backend-go). --domain explícito tem precedência.
  --json         Emite saída em JSON (machine-readable).
  -h, --help     Mostra esta ajuda.

Exemplos:
  ./pose suggest                                       # lista todos os tipos
  ./pose suggest feature                               # trilha para feature
  ./pose suggest feature --domain frontend             # explícito
  ./pose suggest feature --path local/storage-ui/src   # inferido
  ./pose suggest bugfix --path global/account-service  # → backend-go via repo-map
  ./pose suggest review --json
USAGE
}

TASK_TYPE=""
DOMAIN=""
PATH_HINT=""
EMIT_JSON=false

while [[ $# -gt 0 ]]; do
  case "$1" in
    -h|--help) usage; exit 0 ;;
    --json) EMIT_JSON=true; shift ;;
    --domain)
      [[ $# -ge 2 && -n "${2:-}" && "${2:-}" != --* ]] || { echo "Erro: --domain exige um valor." >&2; exit 2; }
      DOMAIN="$2"; shift 2 ;;
    --path)
      [[ $# -ge 2 && -n "${2:-}" && "${2:-}" != --* ]] || { echo "Erro: --path exige um valor." >&2; exit 2; }
      PATH_HINT="$2"; shift 2 ;;
    --*) echo "Erro: opção desconhecida: $1" >&2; usage; exit 2 ;;
    *)
      if [[ -z "$TASK_TYPE" ]]; then
        TASK_TYPE="$1"
      else
        echo "Erro: argumento extra: $1" >&2
        exit 2
      fi
      shift ;;
  esac
done

REPO_ROOT="$(pose_repo_root)"
TASKMAP="$REPO_ROOT/.pose/indexes/task-map.json"
REPOMAP="$REPO_ROOT/.pose/indexes/repo-map.json"
SUGGESTER="$REPO_ROOT/.pose/scripts/pose-suggest.py"

if [[ ! -f "$TASKMAP" ]]; then
  echo "Erro: task-map ausente: $TASKMAP" >&2
  exit 2
fi
if [[ ! -x "$SUGGESTER" ]]; then
  echo "Erro: helper ausente ou sem permissão de execução: $SUGGESTER" >&2
  exit 2
fi

extra=()
$EMIT_JSON && extra+=(--json)
[[ -n "$DOMAIN" ]] && extra+=(--domain "$DOMAIN")
[[ -n "$PATH_HINT" ]] && extra+=(--path "$PATH_HINT")
[[ -n "$TASK_TYPE" ]] && extra+=(--task-type "$TASK_TYPE")

python3 "$SUGGESTER" \
  --task-map "$TASKMAP" \
  --repo-map "$REPOMAP" \
  --repo-root "$REPO_ROOT" \
  "${extra[@]}"

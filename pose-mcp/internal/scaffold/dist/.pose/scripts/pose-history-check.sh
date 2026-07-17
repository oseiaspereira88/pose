#!/usr/bin/env bash
set -euo pipefail

# shellcheck source=pose-lib.sh
source "$(dirname "${BASH_SOURCE[0]}")/pose-lib.sh"

usage() {
  cat <<'USAGE'
Uso: pose-history-check.sh [--strict|--tolerant]

Verifica que todos os .jsonl em .pose/reports/history/ estão sob versionamento
git e sem mudanças não-staged. Sem isso, recurrence-check e stats podem
divergir entre máquinas (history é fonte de verdade para promoções de check).

Modos:
  --strict     Falha (exit 1) se há JSONL untracked ou modificado não-staged.
  --tolerant   Sempre exit 0; reporta como aviso (default).

Saída para consumo por shell:
  history.untracked=<N>
  history.modified_unstaged=<N>
  history.staged_or_clean=<N>
USAGE
}

MODE="tolerant"
while [[ $# -gt 0 ]]; do
  case "$1" in
    --strict) MODE="strict"; shift ;;
    --tolerant) MODE="tolerant"; shift ;;
    -h|--help) usage; exit 0 ;;
    *) echo "Erro: argumento inválido: $1" >&2; usage; exit 2 ;;
  esac
done

REPO_ROOT="$(pose_repo_root)"
HISTORY_DIR="$REPO_ROOT/.pose/reports/history"
if [[ ! -d "$HISTORY_DIR" ]]; then
  echo "Erro: history dir ausente: $HISTORY_DIR" >&2
  exit 2
fi
if ! git -C "$REPO_ROOT" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  echo "Erro: não é um repositório git: $REPO_ROOT" >&2
  exit 2
fi

untracked=0
modified=0
clean=0

while IFS= read -r -d '' file; do
  rel="${file#"$REPO_ROOT"/}"
  status="$(git -C "$REPO_ROOT" status --porcelain=v1 -- "$rel" 2>/dev/null || true)"
  if [[ -z "$status" ]]; then
    clean=$((clean + 1))
    continue
  fi
  # porcelain v1 format: "XY <path>". X=index, Y=worktree.
  # untracked: "?? path"; modified unstaged: " M path"; staged: "M  path".
  case "$status" in
    "??"*)
      echo "[AVISO] JSONL untracked: $rel" >&2
      untracked=$((untracked + 1))
      ;;
    " M "*|" D "*|" T "*)
      echo "[AVISO] JSONL modificado e não-staged: $rel" >&2
      modified=$((modified + 1))
      ;;
    *)
      # Index has changes (staged) — OK do ponto de vista do gate.
      clean=$((clean + 1))
      ;;
  esac
done < <(find "$HISTORY_DIR" -maxdepth 1 -type f -name '*.jsonl' -print0)

echo "history.untracked=$untracked"
echo "history.modified_unstaged=$modified"
echo "history.staged_or_clean=$clean"

problems=$((untracked + modified))
if (( problems > 0 )); then
  echo "Resultado: FALHA ($problems JSONL fora do versionamento)"
  if [[ "$MODE" == "strict" ]]; then
    echo "Para corrigir: git add .pose/reports/history/" >&2
    exit 1
  fi
  echo "Modo tolerant: registrar e versionar antes do próximo merge."
  echo "Resultado: FALHA_TOLERADA"
  exit 0
fi

echo "Resultado: SUCESSO"

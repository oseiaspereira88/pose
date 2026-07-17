#!/usr/bin/env bash
set -euo pipefail

# shellcheck source=pose-lib.sh
source "$(dirname "${BASH_SOURCE[0]}")/pose-lib.sh"

usage() {
  cat <<'USAGE'
Uso: pose-lint-spec.sh <slug>|--all [--strict|--tolerant] [--required-only]

Lint de specs em .pose/specs/<slug>/spec.md. Detecta seções vazias ou
esqueléticas (apenas placeholders/comments HTML) e aplica o gate de ciclo
de vida: specs com frontmatter `status: done` exigem `completed_at` e
disposição válida em cada follow-up. Specs legadas (sem status) não disparam
o gate.

Argumentos:
  <slug>             Nome do diretório dentro de .pose/specs/.
  --all              Lint em todas as specs encontradas.

Opções:
  --strict           Exit 1 se qualquer spec tem seção obrigatória vazia (default).
  --tolerant         Sempre exit 0; reporta como aviso.
  --required-only    Ignora seções opcionais (Decisions).

Seções obrigatórias: Intent, Requirements, Technical Plan, Tasks, Validation, Final Report.
Seções opcionais: Decisions.
USAGE
}

MODE="strict"
REQUIRED_ONLY=false
READY_CHECK=false
TARGET=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --strict) MODE="strict"; shift ;;
    --tolerant) MODE="tolerant"; shift ;;
    --required-only) REQUIRED_ONLY=true; shift ;;
    --ready-check) READY_CHECK=true; shift ;;
    --all) TARGET="--all"; shift ;;
    -h|--help) usage; exit 0 ;;
    --*) echo "Erro: opção desconhecida: $1" >&2; usage; exit 2 ;;
    *)
      if [[ -z "$TARGET" ]]; then
        TARGET="$1"
      else
        echo "Erro: argumento extra: $1" >&2
        exit 2
      fi
      shift ;;
  esac
done

if [[ -z "$TARGET" ]]; then
  echo "Erro: informe <slug> ou --all" >&2
  usage
  exit 2
fi

REPO_ROOT="$(pose_repo_root)"
SPECS_DIR="$REPO_ROOT/.pose/specs"
LINTER="$REPO_ROOT/.pose/scripts/pose-lint-spec.py"

if [[ ! -x "$LINTER" ]]; then
  echo "Erro: linter ausente ou sem permissão de execução: $LINTER" >&2
  exit 2
fi

extra_args=()
$REQUIRED_ONLY && extra_args+=(--required-only)
$READY_CHECK && extra_args+=(--ready-check)

lint_one() {
  local spec_path="$1"
  set +e
  python3 "$LINTER" --spec "$spec_path" "${extra_args[@]}"
  local rc=$?
  set -e
  return "$rc"
}

total_failed=0
total_linted=0

if [[ "$TARGET" == "--all" ]]; then
  while IFS= read -r -d '' spec_md; do
    total_linted=$((total_linted + 1))
    echo "---"
    if ! lint_one "$spec_md"; then
      total_failed=$((total_failed + 1))
    fi
  done < <(find "$SPECS_DIR" -mindepth 2 -maxdepth 2 -type f -name 'spec.md' -print0)

  # Specs antigas (sem spec.md consolidado) também são listadas como aviso.
  while IFS= read -r -d '' spec_dir; do
    if [[ ! -f "$spec_dir/spec.md" ]]; then
      echo "---"
      echo "[AVISO] $(basename "$spec_dir"): sem spec.md consolidado (formato pré-template-único)" >&2
    fi
  done < <(find "$SPECS_DIR" -mindepth 1 -maxdepth 1 -type d -print0)
else
  spec_md="$SPECS_DIR/$TARGET/spec.md"
  if [[ ! -f "$spec_md" ]]; then
    # Fallback: aceita slug duplo "specs/pose-knowledge-governance.md"
    legacy="$SPECS_DIR/$TARGET.md"
    if [[ -f "$legacy" ]]; then
      spec_md="$legacy"
    else
      echo "Erro: spec não encontrada: $spec_md" >&2
      exit 2
    fi
  fi
  total_linted=1
  lint_one "$spec_md" || total_failed=1
fi

echo
echo "lint.specs.checked=$total_linted"
echo "lint.specs.failed=$total_failed"

if (( total_failed > 0 )); then
  echo "Resultado: FALHA ($total_failed spec(s) com seção obrigatória vazia/esquelética ou gate de ciclo de vida violado)"
  if [[ "$MODE" == "strict" ]]; then
    exit 1
  fi
  echo "Modo tolerant: registrar follow-up para completar specs."
  echo "Resultado: FALHA_TOLERADA"
  exit 0
fi

echo "Resultado: SUCESSO"

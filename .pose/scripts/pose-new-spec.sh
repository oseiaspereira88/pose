#!/usr/bin/env bash
set -euo pipefail

# shellcheck source=pose-lib.sh
source "$(dirname "${BASH_SOURCE[0]}")/pose-lib.sh"

FEATURE="${1:-}"
if [ -z "$FEATURE" ]; then
  echo "Uso: ./pose new-spec <feature-slug>" >&2
  exit 2
fi

ROOT_DIR="$(pose_repo_root)"
SPEC_DIR="$ROOT_DIR/.pose/specs/$FEATURE"
TPL_FILE="$ROOT_DIR/.pose/templates/spec.md"

if [[ ! -f "$TPL_FILE" ]]; then
  echo "Erro: template ausente: $TPL_FILE" >&2
  exit 2
fi

if [ -d "$SPEC_DIR" ]; then
  echo "Erro: spec já existe: $SPEC_DIR" >&2
  exit 1
fi

CREATED_AT="$(date -u +%F)"

mkdir -p "$SPEC_DIR"
sed \
  -e "s/<feature-slug>/$FEATURE/g" \
  -e "s/<created_at>/$CREATED_AT/g" \
  "$TPL_FILE" > "$SPEC_DIR/spec.md"

echo "Spec criada: $SPEC_DIR/spec.md (status: draft, created_at: $CREATED_AT)"
echo "Ao concluir, rode o fechamento (skill pose-spec-closeout):"
echo "  - status: done + completed_at preenchido;"
echo "  - disposição em cada follow-up ('./pose followups --open' para o backlog);"
echo "  - './pose lint-spec $FEATURE --strict' como gate de saída."

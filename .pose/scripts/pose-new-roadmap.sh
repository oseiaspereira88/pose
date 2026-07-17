#!/usr/bin/env bash
set -euo pipefail

# shellcheck source=pose-lib.sh
source "$(dirname "${BASH_SOURCE[0]}")/pose-lib.sh"

ROADMAP="${1:-}"
if [ -z "$ROADMAP" ]; then
  echo "Uso: ./pose new-roadmap <roadmap-slug>" >&2
  exit 2
fi

ROOT_DIR="$(pose_repo_root)"
ROADMAP_DIR="$ROOT_DIR/.pose/roadmaps"
TPL_FILE="$ROOT_DIR/.pose/templates/roadmap.md"
TARGET="$ROADMAP_DIR/$ROADMAP.md"

if [[ ! -f "$TPL_FILE" ]]; then
  echo "Erro: template ausente: $TPL_FILE" >&2
  exit 2
fi

if [ -f "$TARGET" ]; then
  echo "Erro: roadmap já existe: $TARGET" >&2
  exit 1
fi

CREATED_AT="$(date -u +%F)"

mkdir -p "$ROADMAP_DIR"
sed \
  -e "s/<roadmap-slug>/$ROADMAP/g" \
  -e "s/<created_at>/$CREATED_AT/g" \
  "$TPL_FILE" > "$TARGET"

echo "Roadmap criado: $TARGET (status: draft)"
echo "Edite os milestones (## Milestone: <id>) e valide com './pose check --strict'."

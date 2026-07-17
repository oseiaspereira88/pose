#!/usr/bin/env bash
set -euo pipefail

# shellcheck source=pose-lib.sh
source "$(dirname "${BASH_SOURCE[0]}")/pose-lib.sh"

TITLE="${*:-}"
if [ -z "$TITLE" ]; then
  echo "Uso: ./pose new-adr <título>" >&2
  exit 2
fi

ROOT_DIR="$(pose_repo_root)"
DATE="$(date +%Y-%m-%d)"
SLUG="$(pose_slugify "$TITLE")"
ADR_DIR="$ROOT_DIR/.pose/adr"

FILE="$ADR_DIR/${DATE}-${SLUG}.md"
mkdir -p "$ADR_DIR"

if [[ -e "$FILE" ]]; then
  echo "Erro: ADR já existe: $FILE" >&2
  exit 1
fi

cat > "$FILE" <<EOT
# ADR: $TITLE

## Status
Proposed

## Context

## Decision

## Consequences
EOT
echo "ADR criada: $FILE"

#!/usr/bin/env bash
# POSE post-merge hook (gerenciado).
# Origem: .pose/hooks/post-merge.sh — NÃO editar a cópia em .git/hooks/.
# Reinstale com: ./pose hooks install
set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
cd "$ROOT"

# Regenera índices silenciosamente após merge (caso novos módulos tenham
# entrado no repo via package.json/go.mod/etc.).
if ./pose index >/dev/null 2>&1; then
  echo "[pose] índices regenerados após merge."
else
  echo "[pose] aviso: ./pose index falhou após merge; rode manualmente." >&2
fi

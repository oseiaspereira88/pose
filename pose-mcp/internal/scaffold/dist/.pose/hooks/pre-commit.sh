#!/usr/bin/env bash
# POSE pre-commit hook (gerenciado).
# Origem: .pose/hooks/pre-commit.sh — NÃO editar a cópia em .git/hooks/.
# Reinstale com: ./pose hooks install
set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
cd "$ROOT"

# Roda check tolerant (rápido, ~50ms) — avisa sem bloquear commit em pequenos
# desvios; falha apenas em quebras estruturais.
if ! ./pose check --tolerant; then
  echo "" >&2
  echo "[pose] pre-commit: estrutura POSE com erros. Resolva antes do commit," >&2
  echo "       ou use 'git commit --no-verify' se for um ajuste emergencial." >&2
  exit 1
fi

#!/usr/bin/env bash
# Migration 001-baseline — materializa o baseline do contrato v1:
# diretórios de instância introduzidos até 2026-07 (roadmaps, changelogs,
# reports/history). Idempotente; nunca toca conteúdo de usuário.
set -euo pipefail
ROOT_DIR="${1:?uso: 001-baseline.sh <repo-root>}"
mkdir -p \
  "$ROOT_DIR/.pose/roadmaps" \
  "$ROOT_DIR/.pose/changelogs/unreleased" \
  "$ROOT_DIR/.pose/reports/history"
echo "[migration 001] baseline garantido (roadmaps/, changelogs/unreleased/, reports/history/)"

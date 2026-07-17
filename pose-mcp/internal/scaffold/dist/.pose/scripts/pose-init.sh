#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
POSE_DIR="$ROOT_DIR/.pose"

# --wizard delega ao onboarding assistido (pose-init-wizard). Os demais args
# (ex.: --yes) seguem para o wizard.
if [[ "${1:-}" == "--wizard" ]]; then
  shift
  exec bash "$(dirname "${BASH_SOURCE[0]}")/pose-init-wizard.sh" "$@"
fi

# Diretórios mínimos do contrato POSE. Idempotente: cria apenas o que falta.
REQUIRED_DIRS=(
  "$POSE_DIR/workflows"
  "$POSE_DIR/templates"
  "$POSE_DIR/rules"
  "$POSE_DIR/scripts"
  "$POSE_DIR/specs"
  "$POSE_DIR/adr"
  "$POSE_DIR/indexes"
  "$POSE_DIR/reports"
  "$POSE_DIR/reports/history"
  "$POSE_DIR/knowledge"
  "$POSE_DIR/roadmaps"
  "$POSE_DIR/changelogs/unreleased"
  "$ROOT_DIR/.agents/skills"
)

created=0
for dir in "${REQUIRED_DIRS[@]}"; do
  if [[ ! -d "$dir" ]]; then
    mkdir -p "$dir"
    echo "[OK] criado: ${dir#$ROOT_DIR/}"
    created=$((created + 1))
  fi
done

if [[ "$created" -eq 0 ]]; then
  echo "[INFO] estrutura POSE já presente. Execute: ./pose check"
else
  echo "[INFO] $created diretório(s) criado(s). Execute: ./pose check"
fi

#!/usr/bin/env bash
# pose-upgrade.sh — migra o contrato .pose/ da versão da instância até a do
# motor (POSE_SCHEMA_VERSION em pose-lib.sh). Spec: pose-schema-versioning.
#
# Uso: ./pose upgrade [--dry-run]
#
# - Instância sem .pose/schema-version é tratada como versão 0 (pré-schema).
# - Migrações vivem em .pose/scripts/migrations/NNN-<slug>.sh e são aplicadas
#   em ordem, cada uma idempotente, recebendo o repo root como $1.
# - Downgrade (instância > motor) é sempre erro: motor antigo não opera
#   contrato novo.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=/dev/null
source "$SCRIPT_DIR/pose-lib.sh"

ROOT_DIR="$(pose_repo_root)"
POSE_DIR="$ROOT_DIR/.pose"
VERSION_FILE="$POSE_DIR/schema-version"
MIGRATIONS_DIR="$SCRIPT_DIR/migrations"

DRY_RUN=0
case "${1:-}" in
  --dry-run) DRY_RUN=1 ;;
  "") ;;
  *) echo "[ERRO] opção desconhecida: $1 (uso: pose upgrade [--dry-run])" >&2; exit 2 ;;
esac

if ! git -C "$ROOT_DIR" rev-parse --git-dir >/dev/null 2>&1; then
  echo "[ERRO] pose upgrade exige um repositório git (mesma guarda do installer)." >&2
  exit 1
fi
[[ -d "$POSE_DIR" ]] || { echo "[ERRO] sem .pose/ em $ROOT_DIR — nada a migrar." >&2; exit 1; }

current=0
if [[ -f "$VERSION_FILE" ]]; then
  current="$(tr -d '[:space:]' <"$VERSION_FILE")"
  [[ "$current" =~ ^[0-9]+$ ]] || { echo "[ERRO] schema-version inválido: '$current'" >&2; exit 1; }
fi
target="$POSE_SCHEMA_VERSION"

if (( current > target )); then
  echo "[ERRO] instância no schema v$current, motor suporta até v$target — atualize o motor POSE (não há downgrade)." >&2
  exit 1
fi

if (( current == target )); then
  echo "[INFO] instância já no schema v$current. Nada a fazer."
  exit 0
fi

echo "[INFO] upgrade do schema: v$current → v$target"
applied=0
for (( v = current + 1; v <= target; v++ )); do
  printf -v pattern '%03d-' "$v"
  migration="$(find "$MIGRATIONS_DIR" -maxdepth 1 -name "${pattern}*.sh" 2>/dev/null | sort | head -1)"
  if [[ -z "$migration" ]]; then
    echo "[ERRO] migração v$v ausente em $MIGRATIONS_DIR" >&2
    exit 1
  fi
  if (( DRY_RUN )); then
    echo "[DRY-RUN] aplicaria: $(basename "$migration")"
    continue
  fi
  echo "[INFO] aplicando: $(basename "$migration")"
  bash "$migration" "$ROOT_DIR"
  printf '%s\n' "$v" >"$VERSION_FILE"
  applied=$((applied + 1))
done

if (( DRY_RUN )); then
  echo "Resultado: DRY-RUN — plano listado, nada aplicado."
else
  echo "Resultado: SUCESSO — schema em v$(tr -d '[:space:]' <"$VERSION_FILE") ($applied migração(ões) aplicada(s))."
fi

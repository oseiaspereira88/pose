#!/usr/bin/env bash
set -euo pipefail

# shellcheck source=pose-lib.sh
source "$(dirname "${BASH_SOURCE[0]}")/pose-lib.sh"

usage() {
  cat <<'USAGE'
Uso: pose-knowledge-check.sh [--strict|--tolerant] [--max-overdue <n>]

Exemplos:
  ./pose knowledge-check --strict
  ./pose knowledge-check --tolerant --max-overdue 2
USAGE
}

MODE="strict"
MAX_OVERDUE=0
MAX_OVERDUE_EXPLICIT=false

while [[ $# -gt 0 ]]; do
  case "$1" in
    --strict) MODE="strict"; shift ;;
    --tolerant) MODE="tolerant"; shift ;;
    --max-overdue)
      [[ $# -ge 2 && "${2:-}" =~ ^[0-9]+$ ]] || { echo "Erro: --max-overdue exige inteiro >= 0" >&2; usage; exit 2; }
      MAX_OVERDUE="$2"
      MAX_OVERDUE_EXPLICIT=true
      shift 2
      ;;
    -h|--help) usage; exit 0 ;;
    *) echo "Erro: argumento inválido: $1" >&2; usage; exit 2 ;;
  esac
done

if [[ "$MAX_OVERDUE_EXPLICIT" == false ]]; then
  if [[ "$MODE" == "tolerant" ]]; then
    MAX_OVERDUE=2
  else
    MAX_OVERDUE=0
  fi
fi

repo_root="$(pose_repo_root)"
knowledge_dir="$repo_root/.pose/knowledge"
housekeeping="$repo_root/.pose/scripts/pose-knowledge-housekeeping.sh"
validator="$repo_root/.pose/scripts/pose-knowledge-validate.py"

if [[ ! -x "$housekeeping" ]]; then
  echo "Erro: script de housekeeping ausente ou sem permissão de execução: $housekeeping" >&2
  exit 2
fi
if [[ ! -x "$validator" ]]; then
  echo "Erro: script de validação ausente ou sem permissão de execução: $validator" >&2
  exit 2
fi

schema_errors=0
schema_warnings=0
schema_checked=0
schema_failed=false

if schema_output="$(python3 "$validator" --knowledge-dir "$knowledge_dir" 2>&1 1>/tmp/pose-knowledge-validate.$$)"; then
  :
else
  schema_failed=true
fi
schema_stdout="$(cat /tmp/pose-knowledge-validate.$$)"
rm -f /tmp/pose-knowledge-validate.$$

# Reemite stderr (linhas [ERRO]/[AVISO]) e stdout (métricas) para o operador.
if [[ -n "$schema_output" ]]; then
  printf '%s\n' "$schema_output" >&2
fi
printf '%s\n' "$schema_stdout"

while IFS='=' read -r key value; do
  case "$key" in
    knowledge.schema.errors) schema_errors="$value" ;;
    knowledge.schema.warnings) schema_warnings="$value" ;;
    knowledge.schema.checked) schema_checked="$value" ;;
  esac
done <<< "$schema_stdout"

overdue_count="$("$housekeeping" list-expired | sed '/^[[:space:]]*$/d' | wc -l | tr -d ' ')"
echo "knowledge.overdue_count=$overdue_count"
echo "knowledge.max_overdue=$MAX_OVERDUE"

# Gate por schema: strict bloqueia em qualquer erro; tolerant permite passar
# com aviso visível para acionamento de housekeeping.
if [[ "$schema_failed" == true || "$schema_errors" -gt 0 ]]; then
  echo "Resultado: FALHA (schema de knowledge inválido em $schema_errors arquivo(s))"
  if [[ "$MODE" == "strict" ]]; then
    exit 1
  fi
  echo "Modo tolerant: corrigir frontmatter antes do próximo ciclo."
  echo "Resultado: FALHA_TOLERADA"
  exit 0
fi

if (( overdue_count > MAX_OVERDUE )); then
  echo "Resultado: FALHA (backlog vencido acima do limite)"
  if [[ "$MODE" == "strict" ]]; then
    exit 1
  fi
  echo "Modo tolerant: registrar follow-up e executar housekeeping."
  echo "Resultado: FALHA_TOLERADA"
  exit 0
fi

echo "Resultado: SUCESSO"
